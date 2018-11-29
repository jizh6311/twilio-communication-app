package webhook

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/aspenmesh/tce/pkg/trafficclaim"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/glog"
	pilotCrd "istio.io/istio/pilot/pkg/config/kube/crd"
	pilotModel "istio.io/istio/pilot/pkg/model"
	admitv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

var (
	runtimeScheme   = runtime.NewScheme()
	codecs          = serializer.NewCodecFactory(runtimeScheme)
	deserializer    = codecs.UniversalDeserializer()
	pilotDescriptor = pilotModel.IstioConfigTypes
)

type WebhookServer interface {
	Serve(w http.ResponseWriter, r *http.Request)
}

type webhookServer struct {
	mux     *http.ServeMux
	claimDb trafficclaim.Db
}

func NewWebhookServer(m *http.ServeMux, c trafficclaim.Db) (WebhookServer, error) {
	whs := &webhookServer{
		mux:     m,
		claimDb: c,
	}
	m.HandleFunc("/validate", whs.Serve)
	return whs, nil
}

func createResponse(review *admitv1beta1.AdmissionReview, resp *admitv1beta1.AdmissionResponse) *admitv1beta1.AdmissionReview {
	respReview := admitv1beta1.AdmissionReview{}
	if resp != nil {
		respReview.Response = resp
		if review.Request != nil {
			respReview.Response.UID = review.Request.UID
		}
	}
	return &respReview
}

func failValidation(review *admitv1beta1.AdmissionReview, err error, statusReason metav1.StatusReason) *admitv1beta1.AdmissionReview {
	resp := &admitv1beta1.AdmissionResponse{
		Allowed: false,
		Result: &metav1.Status{
			Message: err.Error(),
		},
	}
	return createResponse(review, resp)
}

func failValidationUnauth(review *admitv1beta1.AdmissionReview, err error) *admitv1beta1.AdmissionReview {
	RejectedMetric.WithLabelValues(
		review.Request.Namespace,
		review.Request.Kind.Group,
		review.Request.Kind.Kind,
		"unauthorized",
	).Inc()
	return failValidation(review, err, metav1.StatusReasonUnauthorized)
}

func failValidationInvalid(review *admitv1beta1.AdmissionReview, err error) *admitv1beta1.AdmissionReview {
	RejectedMetric.WithLabelValues(
		review.Request.Namespace,
		review.Request.Kind.Group,
		review.Request.Kind.Kind,
		"invalid",
	).Inc()
	return failValidation(review, err, metav1.StatusReasonInvalid)
}

func reportValidationPass(review *admitv1beta1.AdmissionReview) *admitv1beta1.AdmissionReview {
	AllowedMetric.WithLabelValues(
		review.Request.Namespace,
		review.Request.Kind.Group,
		review.Request.Kind.Kind,
	).Inc()
	return createResponse(review, &admitv1beta1.AdmissionResponse{Allowed: true})
}

type dispatch struct {
	Kind              metav1.GroupVersionKind
	JsonUnmarshalType pilotCrd.IstioObject
	PilotProtoSchema  pilotModel.ProtoSchema
	Validator         func(ns string, s *webhookServer, spec proto.Message) error
}

var (
	VirtualService = dispatch{
		Kind: metav1.GroupVersionKind{
			Group:   "networking.istio.io",
			Version: "v1alpha3",
			Kind:    "VirtualService",
		},

		JsonUnmarshalType: &pilotCrd.VirtualService{},
		PilotProtoSchema:  pilotModel.VirtualService,
		Validator:         validateVirtualService,
	}

	Gateway = dispatch{
		Kind: metav1.GroupVersionKind{
			Group:   "networking.istio.io",
			Version: "v1alpha3",
			Kind:    "Gateway",
		},

		JsonUnmarshalType: &pilotCrd.Gateway{},
		PilotProtoSchema:  pilotModel.Gateway,
		Validator:         validateGateway,
	}

	ServiceEntry = dispatch{
		Kind: metav1.GroupVersionKind{
			Group:   "networking.istio.io",
			Version: "v1alpha3",
			Kind:    "ServiceEntry",
		},

		JsonUnmarshalType: &pilotCrd.ServiceEntry{},
		PilotProtoSchema:  pilotModel.ServiceEntry,
		Validator:         validateServiceEntry,
	}

	DestinationRule = dispatch{
		Kind: metav1.GroupVersionKind{
			Group:   "networking.istio.io",
			Version: "v1alpha3",
			Kind:    "DestinationRule",
		},

		JsonUnmarshalType: &pilotCrd.DestinationRule{},
		PilotProtoSchema:  pilotModel.DestinationRule,
		Validator:         validateDestinationRule,
	}

	Dispatchers = []*dispatch{
		&VirtualService,
		&Gateway,
		&ServiceEntry,
		&DestinationRule,
	}
)

func (s *webhookServer) validate(review *admitv1beta1.AdmissionReview) *admitv1beta1.AdmissionReview {
	req := review.Request
	switch req.Operation {
	case admitv1beta1.Create, admitv1beta1.Update:
	default:
		return failValidationInvalid(
			review,
			fmt.Errorf("Unsupported webhook operation %v", req.Operation),
		)
	}

	var d *dispatch
	for _, dd := range Dispatchers {
		if req.Kind == dd.Kind {
			d = dd
			break
		}
	}
	if d == nil {
		return failValidationInvalid(
			review,
			fmt.Errorf("Unexpected kind %v", req.Kind),
		)
	}

	vs := d.JsonUnmarshalType.DeepCopyObject().(pilotCrd.IstioObject)
	if err := json.Unmarshal(req.Object.Raw, &vs); err != nil {
		return failValidationInvalid(
			review,
			fmt.Errorf("Could not unmarshal raw object: %v", err),
		)
	}

	obj, err := pilotCrd.ConvertObject(d.PilotProtoSchema, vs, "cluster.local")
	if err != nil {
		return failValidationInvalid(
			review,
			fmt.Errorf("Could not convert object as Istio config: %v", err),
		)
	}

	err = d.Validator(obj.Namespace, s, obj.Spec)
	if err != nil {
		return failValidationUnauth(review, err)
	}
	return reportValidationPass(review)
}

func (s *webhookServer) Serve(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil {
		var err error
		body, err = ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "error reading request", http.StatusBadRequest)
			HTTPErrorMetric.WithLabelValues("read_error").Inc()
			glog.Warning("Error reading request from client")
			return
		}
	}
	if len(body) == 0 {
		http.Error(w, "no body", http.StatusBadRequest)
		HTTPErrorMetric.WithLabelValues("no_body").Inc()
		glog.Warning("Rejecting client request with no body")
		return
	}

	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "unexpected Content-Type", http.StatusUnsupportedMediaType)
		HTTPErrorMetric.WithLabelValues("unexpected_content_type").Inc()
		glog.Warning("Rejecting client request with unexpected content-type")
		return
	}

	var reviewResp *admitv1beta1.AdmissionReview
	review := admitv1beta1.AdmissionReview{}
	if _, _, err := deserializer.Decode(body, nil, &review); err != nil {
		glog.Warningf("Rejecting client request, couldn't decode body: %v", err)
		reviewResp = failValidationInvalid(&review, fmt.Errorf("Couldn't decode body"))
	} else {
		glog.V(4).Infof("Client request resp: %v", review.Response)
		reviewResp = s.validate(&review)
	}

	respBody, err := json.Marshal(reviewResp)
	if err != nil {
		glog.Errorf("Can't marshal response: %v", err)
		HTTPErrorMetric.WithLabelValues("marshal_response").Inc()
		http.Error(w, fmt.Sprintf("Can't marshal response"), http.StatusInternalServerError)
	}
	if _, err := w.Write(respBody); err != nil {
		glog.Warningf("Can't write response: %v", err)
		HTTPErrorMetric.WithLabelValues("write_response").Inc()
		http.Error(w, fmt.Sprintf("Can't write response"), http.StatusInternalServerError)
	}
}
