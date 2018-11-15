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

func newFake(claims ...*crd.TrafficClaim) Interface {
	tc := faketcclient.NewSimpleClientset()
	for _, c := range claims {
		tc.NetworkingV1alpha3().TrafficClaims(c.Namespace).Create(c)
	}
	kube := fakekubeclient.NewSimpleClientset()
	return &kubeclient{
		tc:   tc,
		kube: kube,
	}
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
		d := NewDb(newFake())
		It("rejects all external hosts", func() {
			Expect(d.IsHostAllowed("tce-test", "www.bofa.com")).To(BeFalse())
			Expect(d.IsHostAllowed("tce-test", "example.com")).To(BeFalse())
		})
		It("allows namespace-default services", func() {
			Expect(d.IsHostAllowed("tce-test", "foo.tce-test.svc.cluster.local")).To(BeTrue())
		})
		It("rejects services from other namespaces", func() {
			Expect(d.IsHostAllowed("tce-test", "foo.other.svc.cluster.local")).To(BeFalse())
		})
	})

	Describe("Host validation", func() {
		tcWwwBofaCom := newHostsClaim("tce-test", "a", "www.bofa.com")
		It("accepts a basic valid host", func() {
			d := NewDb(newFake(tcWwwBofaCom))
			Expect(d.IsHostAllowed("tce-test", "www.bofa.com")).To(BeTrue())
		})
		It("rejects overbroad hosts", func() {
			d := NewDb(newFake(tcWwwBofaCom))
			Expect(d.IsHostAllowed("tce-test", "bofa.com")).To(BeFalse())
			Expect(d.IsHostAllowed("tce-test", ".bofa.com")).To(BeFalse())
			Expect(d.IsHostAllowed("tce-test", "*.bofa.com")).To(BeFalse())
			Expect(d.IsHostAllowed("tce-test", "*.com")).To(BeFalse())
		})
		It("doesnt match claims in other namespaces", func() {
			d := NewDb(newFake(tcWwwBofaCom))
			Expect(d.IsHostAllowed("not-tce-test", "bofa.com")).To(BeFalse())
		})

		tcBofaCom := newHostsClaim("tce-test", "b", "*.bofa.com")
		It("accepts basic glob matched hosts", func() {
			d := NewDb(newFake(tcBofaCom))
			Expect(d.IsHostAllowed("tce-test", "*.bofa.com")).To(BeTrue())
			Expect(d.IsHostAllowed("tce-test", "www.bofa.com")).To(BeTrue())

			// It looks like DNS globbing doesn't allow this but I think Istio globbing does
			Expect(d.IsHostAllowed("tce-test", "test.www.bofa.com")).To(BeTrue())
		})
		It("rejects overbroad glob matched hosts", func() {
			d := NewDb(newFake(tcBofaCom))
			Expect(d.IsHostAllowed("tce-test", "bofa.com")).To(BeFalse())
			Expect(d.IsHostAllowed("tce-test", ".bofa.com")).To(BeFalse())
			Expect(d.IsHostAllowed("tce-test", "*.com")).To(BeFalse())
		})

		tcExampleCom := newHostsClaim("tce-test", "c", "*.example.com")
		It("accepts hosts that match either TrafficClaim resource", func() {
			d := NewDb(newFake(tcBofaCom, tcExampleCom))
			Expect(d.IsHostAllowed("tce-test", "www.bofa.com")).To(BeTrue())
			Expect(d.IsHostAllowed("tce-test", "www.example.com")).To(BeTrue())
		})

		tcBoth := newHostsClaim("tce-test", "d", "*.example.com", "*.bofa.com")
		It("accepts hosts that match either host in the same claim", func() {
			d := NewDb(newFake(tcBoth))
			Expect(d.IsHostAllowed("tce-test", "www.bofa.com")).To(BeTrue())
			Expect(d.IsHostAllowed("tce-test", "www.example.com")).To(BeTrue())
		})

		tcBothTwoClaims := newHostsSeparateClaim(
			"tce-test",
			"e",
			"*.example.com",
			"*.bofa.com",
		)
		It("accepts hosts that match either claim in the same resource", func() {
			d := NewDb(newFake(tcBothTwoClaims))
			Expect(d.IsHostAllowed("tce-test", "www.bofa.com")).To(BeTrue())
			Expect(d.IsHostAllowed("tce-test", "www.example.com")).To(BeTrue())
		})

		// TrafficClaim allows this, though Istio may implement object-specific behavior
		tcOtherNamespaceClaim := newHostsClaim("tce-test", "f", "*.other.svc.cluster.local")
		It("accepts hosts for other namespaces if explicitly granted", func() {
			d := NewDb(newFake(tcOtherNamespaceClaim))
			Expect(d.IsHostAllowed("tce-test", "foo.other.svc.cluster.local")).To(BeTrue())
		})

		tc443 := newPortsClaim("tce-test", "g", "www.bofa.com", 443)
		It("rejects claims for hosts when only port claimed", func() {
			d := NewDb(newFake(tc443))
			Expect(d.IsHostAllowed("tce-test", "www.bofa.com")).To(BeFalse())
		})
	})

	Describe("Port validation", func() {
		tc443 := newPortsClaim("tce-test", "a", "www.bofa.com", 443)
		It("accepts claims for specific ports", func() {
			d := NewDb(newFake(tc443))
			Expect(d.IsPortAllowed("tce-test", "www.bofa.com", 443)).To(BeTrue())
		})
		It("rejects claims for incorrect ports", func() {
			d := NewDb(newFake(tc443))
			Expect(d.IsPortAllowed("tce-test", "www.bofa.com", 80)).To(BeFalse())
		})

		tcMultiport := newPortsClaim("tce-test", "b", "www.bofa.com", 443, 8443, 9443)
		It("accepts claims for any of multiple ports", func() {
			d := NewDb(newFake(tcMultiport))
			Expect(d.IsPortAllowed("tce-test", "www.bofa.com", 443)).To(BeTrue())
			Expect(d.IsPortAllowed("tce-test", "www.bofa.com", 8443)).To(BeTrue())
			Expect(d.IsPortAllowed("tce-test", "www.bofa.com", 9443)).To(BeTrue())
		})

		tcDifferentPortsOnEachHost := newClaim("tce-test", "c", crd.Claim{
			Hosts: []string{"a.example.com"},
			Ports: []uint32{443, 8443},
		}, crd.Claim{
			Hosts: []string{"b.example.com"},
			Ports: []uint32{80, 8080},
		})
		It("doesn't allow ports on one host just because they're on another", func() {
			d := NewDb(newFake(tcDifferentPortsOnEachHost))
			Expect(d.IsPortAllowed("tce-test", "a.example.com", 443)).To(BeTrue())
			Expect(d.IsPortAllowed("tce-test", "a.example.com", 8443)).To(BeTrue())
			Expect(d.IsPortAllowed("tce-test", "b.example.com", 80)).To(BeTrue())
			Expect(d.IsPortAllowed("tce-test", "b.example.com", 8080)).To(BeTrue())
			Expect(d.IsPortAllowed("tce-test", "a.example.com", 80)).To(BeFalse())
			Expect(d.IsPortAllowed("tce-test", "a.example.com", 8080)).To(BeFalse())
			Expect(d.IsPortAllowed("tce-test", "b.example.com", 443)).To(BeFalse())
			Expect(d.IsPortAllowed("tce-test", "b.example.com", 8443)).To(BeFalse())
		})

		// Someday maybe an admission controller should prevent this from being
		// accepted at all.
		tcPortNoHost := newClaim("tce-test", "d", crd.Claim{Ports: []uint32{80}})
		It("doesn't treat no-host as any-host", func() {
			d := NewDb(newFake(tcPortNoHost))
			Expect(d.IsPortAllowed("tce-test", "a.example.com", 80)).To(BeFalse())
		})
	})

	Describe("Path validation", func() {
		tcProducts := newPathsClaim("tce-test", "a", "example.com", 80, "/products")
		It("accepts claims for specific paths", func() {
			d := NewDb(newFake(tcProducts))
			Expect(d.IsPortPathAllowed("tce-test", "example.com", 80, "/products")).To(BeTrue())
		})
		It("rejects claims for incorrect paths", func() {
			d := NewDb(newFake(tcProducts))
			Expect(d.IsPortPathAllowed("tce-test", "example.com", 80, "/not-products")).To(BeFalse())
			Expect(d.IsPortPathAllowed("tce-test", "example.com", 80, "/prod")).To(BeFalse())
			Expect(d.IsPortPathAllowed("tce-test", "example.com", 81, "/products")).To(BeFalse())
			Expect(d.IsPortPathAllowed("tce-test", "foobar.com", 80, "/products")).To(BeFalse())
			Expect(d.IsPortPathAllowed("tce-test", "foobar.com", 80, "/")).To(BeFalse())
		})
		It("rejects claims for exact paths that would pass if prefix", func() {
			d := NewDb(newFake(tcProducts))
			Expect(d.IsPortPathAllowed("tce-test", "example.com", 80, "/products/foo")).To(BeFalse())
			Expect(d.IsPortPathAllowed("tce-test", "example.com", 80, "/products/")).To(BeFalse())
		})

		tcTwoPaths := newPathsClaim("tce-test", "b", "example.com", 80, "/products", "/orders")
		It("accepts multiple paths in a claim", func() {
			d := NewDb(newFake(tcTwoPaths))
			Expect(d.IsPortPathAllowed("tce-test", "example.com", 80, "/products")).To(BeTrue())
			Expect(d.IsPortPathAllowed("tce-test", "example.com", 80, "/orders")).To(BeTrue())
			Expect(d.IsPortPathAllowed("tce-test", "example.com", 80, "/")).To(BeFalse())
		})

		tcPrefix := newPathPrefixesClaim("tce-test", "d", "example.com", 80, "/products")
		It("accepts prefix claims", func() {
			d := NewDb(newFake(tcPrefix))
			Expect(d.IsPortPathAllowed("tce-test", "example.com", 80, "/products")).To(BeTrue())
			Expect(d.IsPortPathAllowed("tce-test", "example.com", 80, "/products/")).To(BeTrue())
			Expect(d.IsPortPathAllowed("tce-test", "example.com", 80, "/products/baz")).To(BeTrue())
			Expect(d.IsPortPathAllowed("tce-test", "example.com", 80, "/products/baz/bar")).To(BeTrue())
			Expect(d.IsPortPathAllowed("tce-test", "example.com", 80, "/products/products")).To(BeTrue())
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
			d := NewDb(newFake(tcBoth))

			// Configs for exact paths
			Expect(d.IsPortPathAllowed("tce-test", "example.com", 80, "/exact")).To(BeTrue())
			Expect(d.IsPortPathAllowed("tce-test", "example.com", 80, "/prefix1")).To(BeTrue())
			Expect(d.IsPortPathAllowed("tce-test", "example.com", 80, "/prefix1/inside")).To(BeTrue())
			Expect(d.IsPortPathAllowed("tce-test", "example.com", 80, "/prefix2")).To(BeTrue())
			Expect(d.IsPortPathAllowed("tce-test", "example.com", 80, "/prefix2/inside")).To(BeTrue())
			Expect(d.IsPortPathAllowed("tce-test", "example.com", 80, "/prefix2/shouldalsobegranted")).To(BeTrue())
			Expect(d.IsPortPathAllowed("tce-test", "example.com", 80, "/prefix2/shouldalsobegranted/morebyprefix")).To(BeTrue())
			Expect(d.IsPortPathAllowed("tce-test", "example.com", 80, "/exact/nope")).To(BeFalse())

			// Configs for prefixes
			c := &Config{Namespace: "tce-test", Host: "example.com", Port: 80}
			c.PrefixPath = "/prefix2/inside"
			Expect(d.IsConfigAllowed(c)).To(BeTrue())
			c.PrefixPath = "/prefix2"
			Expect(d.IsConfigAllowed(c)).To(BeTrue())
			c.PrefixPath = "/exact"
			Expect(d.IsConfigAllowed(c)).To(BeFalse())
			c.PrefixPath = "/prefix2/shouldalsobegranted"
			Expect(d.IsConfigAllowed(c)).To(BeTrue())
			c.PrefixPath = "/prefix2/shouldalsobegranted/morebyprefix"
			Expect(d.IsConfigAllowed(c)).To(BeTrue())
			c.PrefixPath = "/"
			Expect(d.IsConfigAllowed(c)).To(BeFalse())
		})
	})
})
