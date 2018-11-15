package trafficclaim

import (
	tcclient "github.com/aspenmesh/tce/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func CreateInterface(kubeconfig string) (Interface, error) {
	var restConfig *rest.Config
	var err error
	if len(kubeconfig) == 0 {
		restConfig, err = rest.InClusterConfig()
	} else {
		restConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	if err != nil {
		return nil, err
	}

	kube, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	tc, err := tcclient.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	return &kubeclient{
		kube: kube,
		tc:   tc,
	}, nil
}

type Interface interface {
	Kube() kubernetes.Interface
	Tc() tcclient.Interface
}

type kubeclient struct {
	kube kubernetes.Interface
	tc   tcclient.Interface
}

func (kc *kubeclient) Kube() kubernetes.Interface {
	return kc.kube
}

func (kc *kubeclient) Tc() tcclient.Interface {
	return kc.tc
}
