package webhook

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/aspenmesh/tce/pkg/trafficclaim"
	"github.com/golang/mock/gomock"
	admitv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func isAllowed(review *admitv1beta1.AdmissionReview) bool {
	resp := review.Response
	if resp == nil {
		return false
	}
	return resp.Allowed
}

var _ = Describe("unmarshaller", func() {
	var (
		mockCtrl *gomock.Controller
		mockDb   *trafficclaim.MockDb
		server   *webhookServer
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockDb = trafficclaim.NewMockDb(mockCtrl)
		server = &webhookServer{claimDb: mockDb}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	It("handles golden VirtualService", func() {
		gomock.InOrder(
			mockDb.EXPECT().IsConfigAllowed(&trafficclaim.Config{
				Namespace: "tce-test",
				Host:      "foo.com",
				Port:      80,
				ExactPath: "/admin/login",
			}).Return(true),
			mockDb.EXPECT().IsConfigAllowed(&trafficclaim.Config{
				Namespace:  "tce-test",
				Host:       "foo.com",
				Port:       80,
				PrefixPath: "/products/goodproducts",
			}).Return(true),
			mockDb.EXPECT().IsConfigAllowed(&trafficclaim.Config{
				Namespace: "tce-test",
				Host:      "foo.com",
				Port:      80,
				ExactPath: "/products/matchesprefix",
			}).Return(true),
		)

		review := &admitv1beta1.AdmissionReview{
			TypeMeta: metav1.TypeMeta{},
			Request: &admitv1beta1.AdmissionRequest{
				Operation: admitv1beta1.Create,
				Namespace: "tce-test",
				Name:      "foo",
				Kind: metav1.GroupVersionKind{
					Group:   "networking.istio.io",
					Version: "v1alpha3",
					Kind:    "VirtualService",
				},
				Object: runtime.RawExtension{
					Raw: []byte(goldenVirtualService),
				},
			},
			Response: nil,
		}
		Expect(isAllowed(server.validate(review))).To(BeTrue())
	})
})

var (
	goldenVirtualService = `{"apiVersion":"networking.istio.io/v1alpha3","kind":"VirtualService","metadata":{"name":"foopath","namespace":"tce-test"},"spec":{"hosts":["foo.com"],"http":[{"match":[{"port":80,"uri":{"exact":"/admin/login"}}],"route":[{"destination":{"host":"sleep1"}}]},{"match":[{"port":80,"uri":{"prefix":"/products/goodproducts"}}],"route":[{"destination":{"host":"sleep2"}}]},{"match":[{"port":80,"uri":{"exact":"/products/matchesprefix"}}],"route":[{"destination":{"host":"sleep3"}}]}]}}
`
)
