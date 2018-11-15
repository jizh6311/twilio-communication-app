package trafficclaim

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	crd "github.com/aspenmesh/tce/pkg/api/networking/v1alpha3"
	faketcclient "github.com/aspenmesh/tce/pkg/client/clientset/versioned/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakekubeclient "k8s.io/client-go/kubernetes/fake"
)

func TestProxy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "TrafficClaim")
}

func newFakeDb(claims ...*crd.TrafficClaim) Db {
	tc := faketcclient.NewSimpleClientset()
	for _, c := range claims {
		tc.NetworkingV1alpha3().TrafficClaims(c.Namespace).Create(c)
	}
	kube := fakekubeclient.NewSimpleClientset()
	kubeclient := &kubeclient{
		tc:   tc,
		kube: kube,
	}
	return NewDb(kubeclient)
}

func newFake(claims ...*crd.TrafficClaim) Verification {
	verification, _ := newFakeDb(claims...).NewVerification("tce-test")
	return verification
}

var _ = Describe("isHostValid", func() {
	It("accepts non-glob host", func() {
		Expect(isHostValid("www.example.com")).To(BeTrue())
		Expect(isHostValid("com")).To(BeTrue())
		Expect(isHostValid("example.com")).To(BeTrue())
	})
	It("rejects dot-glob host", func() {
		// FIXME(andrew): I think this isn't allowed in Istio?
		Expect(isHostValid(".example.com")).To(BeFalse())
		Expect(isHostValid(".com")).To(BeFalse())
	})
	It("allows star-glob host", func() {
		Expect(isHostValid("*.www.example.com")).To(BeTrue())
		Expect(isHostValid("*.example.com")).To(BeTrue())
		Expect(isHostValid("*.com")).To(BeTrue())
		Expect(isHostValid("*")).To(BeTrue())
	})
	It("rejects star-prefix but non-glob host", func() {
		Expect(isHostValid("*-foo.example.com")).To(BeFalse())
		Expect(isHostValid("*foo.com")).To(BeFalse())
		Expect(isHostValid("**foo.com")).To(BeFalse())
	})
	It("rejects multiple star-globs", func() {
		Expect(isHostValid("*.*.com")).To(BeFalse())
		Expect(isHostValid("**.example.com")).To(BeFalse())
	})
})

func newClaim(ns string, name string, claims ...crd.Claim) *crd.TrafficClaim {
	return &crd.TrafficClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Claims: claims,
	}
}

func newHostsClaim(ns string, name string, hosts ...string) *crd.TrafficClaim {
	return &crd.TrafficClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Claims: []crd.Claim{
			crd.Claim{
				Hosts: hosts,
			},
		},
	}
}

func newHostsSeparateClaim(ns string, name string, hosts ...string) *crd.TrafficClaim {
	var claims []crd.Claim
	for _, h := range hosts {
		claims = append(claims, crd.Claim{Hosts: []string{h}})
	}
	return &crd.TrafficClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Claims: claims,
	}
}

func newPortsClaim(ns string, name string, host string, ports ...uint32) *crd.TrafficClaim {
	return &crd.TrafficClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Claims: []crd.Claim{
			crd.Claim{
				Hosts: []string{host},
				Ports: ports,
			},
		},
	}
}

func newPathsClaim(ns string, name string, host string, port uint32, paths ...string) *crd.TrafficClaim {
	return &crd.TrafficClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Claims: []crd.Claim{
			crd.Claim{
				Hosts: []string{host},
				Ports: []uint32{port},
				Http: &crd.Http{
					Paths: &crd.Paths{
						Exact:  paths,
						Prefix: []string{},
					},
				},
			},
		},
	}
}

func newPathPrefixesClaim(ns string, name string, host string, port uint32, paths ...string) *crd.TrafficClaim {
	return &crd.TrafficClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Claims: []crd.Claim{
			crd.Claim{
				Hosts: []string{host},
				Ports: []uint32{port},
				Http: &crd.Http{
					Paths: &crd.Paths{
						Exact:  []string{},
						Prefix: paths,
					},
				},
			},
		},
	}
}

var _ = Describe("TrafficClaim Hosts", func() {
	Describe("Basic validation", func() {
		v := newFake()
		It("rejects all external hosts", func() {
			Expect(v.IsHostAllowed("www.bofa.com")).To(BeFalse())
			Expect(v.IsHostAllowed("example.com")).To(BeFalse())
		})
		It("allows namespace-default services", func() {
			Expect(v.IsHostAllowed("foo.tce-test.svc.cluster.local")).To(BeTrue())
		})
		It("rejects services from other namespaces", func() {
			Expect(v.IsHostAllowed("foo.other.svc.cluster.local")).To(BeFalse())
		})
	})

	Describe("Host validation", func() {
		tcWwwBofaCom := newHostsClaim("tce-test", "a", "www.bofa.com")
		It("accepts a basic valid host", func() {
			v := newFake(tcWwwBofaCom)
			Expect(v.IsHostAllowed("www.bofa.com")).To(BeTrue())
		})
		It("rejects overbroad hosts", func() {
			v := newFake(tcWwwBofaCom)
			Expect(v.IsHostAllowed("bofa.com")).To(BeFalse())
			Expect(v.IsHostAllowed(".bofa.com")).To(BeFalse())
			Expect(v.IsHostAllowed("*.bofa.com")).To(BeFalse())
			Expect(v.IsHostAllowed("*.com")).To(BeFalse())
		})
		It("doesnt match claims in other namespaces", func() {
			d := newFakeDb(tcWwwBofaCom)
			vOther, err := d.NewVerification("not-tce-test")
			Expect(err).NotTo(HaveOccurred())
			Expect(vOther.IsHostAllowed("www.bofa.com")).To(BeFalse())
			vThis, err := d.NewVerification("tce-test")
			Expect(err).NotTo(HaveOccurred())
			Expect(vThis.IsHostAllowed("www.bofa.com")).To(BeTrue())
		})

		tcBofaCom := newHostsClaim("tce-test", "b", "*.bofa.com")
		It("accepts basic glob matched hosts", func() {
			v := newFake(tcBofaCom)
			Expect(v.IsHostAllowed("*.bofa.com")).To(BeTrue())
			Expect(v.IsHostAllowed("www.bofa.com")).To(BeTrue())

			// It looks like DNS globbing doesn't allow this but I think Istio globbing does
			Expect(v.IsHostAllowed("test.www.bofa.com")).To(BeTrue())
		})
		It("rejects overbroad glob matched hosts", func() {
			v := newFake(tcBofaCom)
			Expect(v.IsHostAllowed("bofa.com")).To(BeFalse())
			Expect(v.IsHostAllowed(".bofa.com")).To(BeFalse())
			Expect(v.IsHostAllowed("*.com")).To(BeFalse())
		})

		tcExampleCom := newHostsClaim("tce-test", "c", "*.example.com")
		It("accepts hosts that match either TrafficClaim resource", func() {
			v := newFake(tcBofaCom, tcExampleCom)
			Expect(v.IsHostAllowed("www.bofa.com")).To(BeTrue())
			Expect(v.IsHostAllowed("www.example.com")).To(BeTrue())
		})

		tcBoth := newHostsClaim("tce-test", "d", "*.example.com", "*.bofa.com")
		It("accepts hosts that match either host in the same claim", func() {
			v := newFake(tcBoth)
			Expect(v.IsHostAllowed("www.bofa.com")).To(BeTrue())
			Expect(v.IsHostAllowed("www.example.com")).To(BeTrue())
		})

		tcBothTwoClaims := newHostsSeparateClaim(
			"tce-test",
			"e",
			"*.example.com",
			"*.bofa.com",
		)
		It("accepts hosts that match either claim in the same resource", func() {
			v := newFake(tcBothTwoClaims)
			Expect(v.IsHostAllowed("www.bofa.com")).To(BeTrue())
			Expect(v.IsHostAllowed("www.example.com")).To(BeTrue())
		})

		// TrafficClaim allows this, though Istio may implement object-specific behavior
		tcOtherNamespaceClaim := newHostsClaim("tce-test", "f", "*.other.svc.cluster.local")
		It("accepts hosts for other namespaces if explicitly granted", func() {
			v := newFake(tcOtherNamespaceClaim)
			Expect(v.IsHostAllowed("foo.other.svc.cluster.local")).To(BeTrue())
		})

		tc443 := newPortsClaim("tce-test", "g", "www.bofa.com", 443)
		It("rejects claims for hosts when only port claimed", func() {
			v := newFake(tc443)
			Expect(v.IsHostAllowed("www.bofa.com")).To(BeFalse())
		})
	})

	Describe("Port validation", func() {
		tc443 := newPortsClaim("tce-test", "a", "www.bofa.com", 443)
		It("accepts claims for specific ports", func() {
			v := newFake(tc443)
			Expect(v.IsPortAllowed("www.bofa.com", 443)).To(BeTrue())
		})
		It("rejects claims for incorrect ports", func() {
			v := newFake(tc443)
			Expect(v.IsPortAllowed("www.bofa.com", 80)).To(BeFalse())
		})

		tcMultiport := newPortsClaim("tce-test", "b", "www.bofa.com", 443, 8443, 9443)
		It("accepts claims for any of multiple ports", func() {
			v := newFake(tcMultiport)
			Expect(v.IsPortAllowed("www.bofa.com", 443)).To(BeTrue())
			Expect(v.IsPortAllowed("www.bofa.com", 8443)).To(BeTrue())
			Expect(v.IsPortAllowed("www.bofa.com", 9443)).To(BeTrue())
		})

		tcDifferentPortsOnEachHost := newClaim("tce-test", "c", crd.Claim{
			Hosts: []string{"a.example.com"},
			Ports: []uint32{443, 8443},
		}, crd.Claim{
			Hosts: []string{"b.example.com"},
			Ports: []uint32{80, 8080},
		})
		It("doesn't allow ports on one host just because they're on another", func() {
			v := newFake(tcDifferentPortsOnEachHost)
			Expect(v.IsPortAllowed("a.example.com", 443)).To(BeTrue())
			Expect(v.IsPortAllowed("a.example.com", 8443)).To(BeTrue())
			Expect(v.IsPortAllowed("b.example.com", 80)).To(BeTrue())
			Expect(v.IsPortAllowed("b.example.com", 8080)).To(BeTrue())
			Expect(v.IsPortAllowed("a.example.com", 80)).To(BeFalse())
			Expect(v.IsPortAllowed("a.example.com", 8080)).To(BeFalse())
			Expect(v.IsPortAllowed("b.example.com", 443)).To(BeFalse())
			Expect(v.IsPortAllowed("b.example.com", 8443)).To(BeFalse())
		})

		// Someday maybe an admission controller should prevent this from being
		// accepted at all.
		tcPortNoHost := newClaim("tce-test", "d", crd.Claim{Ports: []uint32{80}})
		It("doesn't treat no-host as any-host", func() {
			v := newFake(tcPortNoHost)
			Expect(v.IsPortAllowed("a.example.com", 80)).To(BeFalse())
		})
	})

	Describe("Path validation", func() {
		tcProducts := newPathsClaim("tce-test", "a", "example.com", 80, "/products")
		It("accepts claims for specific paths", func() {
			v := newFake(tcProducts)
			Expect(v.IsPortPathAllowed("example.com", 80, "/products")).To(BeTrue())
		})
		It("rejects claims for incorrect paths", func() {
			v := newFake(tcProducts)
			Expect(v.IsPortPathAllowed("example.com", 80, "/not-products")).To(BeFalse())
			Expect(v.IsPortPathAllowed("example.com", 80, "/prod")).To(BeFalse())
			Expect(v.IsPortPathAllowed("example.com", 81, "/products")).To(BeFalse())
			Expect(v.IsPortPathAllowed("foobar.com", 80, "/products")).To(BeFalse())
			Expect(v.IsPortPathAllowed("foobar.com", 80, "/")).To(BeFalse())
		})
		It("rejects claims for exact paths that would pass if prefix", func() {
			v := newFake(tcProducts)
			Expect(v.IsPortPathAllowed("example.com", 80, "/products/foo")).To(BeFalse())
			Expect(v.IsPortPathAllowed("example.com", 80, "/products/")).To(BeFalse())
		})

		tcTwoPaths := newPathsClaim("tce-test", "b", "example.com", 80, "/products", "/orders")
		It("accepts multiple paths in a claim", func() {
			v := newFake(tcTwoPaths)
			Expect(v.IsPortPathAllowed("example.com", 80, "/products")).To(BeTrue())
			Expect(v.IsPortPathAllowed("example.com", 80, "/orders")).To(BeTrue())
			Expect(v.IsPortPathAllowed("example.com", 80, "/")).To(BeFalse())
		})

		tcPrefix := newPathPrefixesClaim("tce-test", "d", "example.com", 80, "/products")
		It("accepts prefix claims", func() {
			v := newFake(tcPrefix)
			Expect(v.IsPortPathAllowed("example.com", 80, "/products")).To(BeTrue())
			Expect(v.IsPortPathAllowed("example.com", 80, "/products/")).To(BeTrue())
			Expect(v.IsPortPathAllowed("example.com", 80, "/products/baz")).To(BeTrue())
			Expect(v.IsPortPathAllowed("example.com", 80, "/products/baz/bar")).To(BeTrue())
			Expect(v.IsPortPathAllowed("example.com", 80, "/products/products")).To(BeTrue())
		})

		tcBoth := newClaim("tce-test", "e", crd.Claim{
			Hosts: []string{"example.com"},
			Ports: []uint32{80},
			Http: &crd.Http{
				Paths: &crd.Paths{
					Exact: []string{
						"/exact",
						"/prefix2/shouldalsobegranted",
					},
					Prefix: []string{
						"/prefix1",
						"/prefix2",
					},
				},
			},
		})
		It("accepts both exact and prefix specs in a claim", func() {
			v := newFake(tcBoth)

			// Configs for exact paths
			Expect(v.IsPortPathAllowed("example.com", 80, "/exact")).To(BeTrue())
			Expect(v.IsPortPathAllowed("example.com", 80, "/prefix1")).To(BeTrue())
			Expect(v.IsPortPathAllowed("example.com", 80, "/prefix1/inside")).To(BeTrue())
			Expect(v.IsPortPathAllowed("example.com", 80, "/prefix2")).To(BeTrue())
			Expect(v.IsPortPathAllowed("example.com", 80, "/prefix2/inside")).To(BeTrue())
			Expect(v.IsPortPathAllowed("example.com", 80, "/prefix2/shouldalsobegranted")).To(BeTrue())
			Expect(v.IsPortPathAllowed("example.com", 80, "/prefix2/shouldalsobegranted/morebyprefix")).To(BeTrue())
			Expect(v.IsPortPathAllowed("example.com", 80, "/exact/nope")).To(BeFalse())

			// Configs for prefixes
			c := &Config{Host: "example.com", Port: 80}
			c.PrefixPath = "/prefix2/inside"
			Expect(v.IsConfigAllowed(c)).To(BeTrue())
			c.PrefixPath = "/prefix2"
			Expect(v.IsConfigAllowed(c)).To(BeTrue())
			c.PrefixPath = "/exact"
			Expect(v.IsConfigAllowed(c)).To(BeFalse())
			c.PrefixPath = "/prefix2/shouldalsobegranted"
			Expect(v.IsConfigAllowed(c)).To(BeTrue())
			c.PrefixPath = "/prefix2/shouldalsobegranted/morebyprefix"
			Expect(v.IsConfigAllowed(c)).To(BeTrue())
			c.PrefixPath = "/"
			Expect(v.IsConfigAllowed(c)).To(BeFalse())
		})
	})
})
