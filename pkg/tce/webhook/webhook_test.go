package webhook

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/aspenmesh/tce/pkg/trafficclaim"
	"github.com/golang/mock/gomock"
	networking "istio.io/api/networking/v1alpha3"
)

func TestProxy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "TrafficClaim")
}

func newHostVs(hosts ...string) *networking.VirtualService {
	return &networking.VirtualService{
		Hosts: hosts,
		Http: []*networking.HTTPRoute{
			&networking.HTTPRoute{
				Route: []*networking.HTTPRouteDestination{
					&networking.HTTPRouteDestination{
						Destination: &networking.Destination{
							Host: "sleep.example.svc.cluster.local",
						},
					},
				},
			},
		},
	}
}

// One (dummy) route per match, not one route with multiple matches
func newMultiRouteVs(host string, matches ...*networking.HTTPMatchRequest) *networking.VirtualService {
	routes := []*networking.HTTPRoute{}
	for i, m := range matches {
		routes = append(routes, &networking.HTTPRoute{
			Match: []*networking.HTTPMatchRequest{m},
			Route: []*networking.HTTPRouteDestination{
				&networking.HTTPRouteDestination{
					Destination: &networking.Destination{
						Host: fmt.Sprintf("route-%d.example.svc.cluster.local", i),
					},
				},
			},
		})
	}
	return &networking.VirtualService{
		Hosts: []string{host},
		Http:  routes,
	}
}

func newPortVs(host string, ports ...uint32) *networking.VirtualService {
	matches := []*networking.HTTPMatchRequest{}
	for _, p := range ports {
		matches = append(matches, &networking.HTTPMatchRequest{Port: p})
	}
	return newMultiRouteVs(host, matches...)
}

func newPortPathVs(host string, port uint32, paths ...string) *networking.VirtualService {
	matches := []*networking.HTTPMatchRequest{}
	for _, p := range paths {
		matches = append(matches, &networking.HTTPMatchRequest{
			Port: port,
			Uri: &networking.StringMatch{
				MatchType: &networking.StringMatch_Exact{
					Exact: p,
				},
			},
		})
	}
	return newMultiRouteVs(host, matches...)
}

// We don't use mocking for this because its only job in life is to return
// a mock verification - that's the thing we care about.
type mmockDb struct {
	v *trafficclaim.MockVerification
}

func (m *mmockDb) NewVerification(ns string) (trafficclaim.Verification, error) {
	Expect(ns).To(Equal("tce-test"))
	return m.v, nil
}

var _ = Describe("validate VirtualService", func() {
	var (
		mockCtrl *gomock.Controller
		mockV    *trafficclaim.MockVerification
		mockDb   *mmockDb
		server   *webhookServer
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockV = trafficclaim.NewMockVerification(mockCtrl)
		mockDb = &mmockDb{v: mockV}
		server = &webhookServer{claimDb: mockDb}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	validate := func(vs *networking.VirtualService) error {
		return validateVirtualService("tce-test", server, vs)
	}

	expectHostAllowed := func(host string) *gomock.Call {
		return mockV.EXPECT().IsConfigAllowed(&trafficclaim.Config{
			Host: host,
		})
	}

	expectPortAllowed := func(host string, port uint32) *gomock.Call {
		return mockV.EXPECT().IsConfigAllowed(&trafficclaim.Config{
			Host: host,
			Port: port,
		})
	}

	hostVs := newHostVs("foo.example.com", "bar.example.com")
	It("passes if all hosts pass", func() {
		gomock.InOrder(
			expectHostAllowed("foo.example.com").Return(true),
			expectHostAllowed("bar.example.com").Return(true),
		)
		Expect(validate(hostVs)).Should(Succeed())
	})

	It("fails if any host fails", func() {
		gomock.InOrder(
			expectHostAllowed("foo.example.com").Return(true),
			expectHostAllowed("bar.example.com").Return(false),
		)
		Expect(validate(hostVs)).ShouldNot(Succeed())
	})

	portVs := newPortVs("foo.example.com", 80, 443)
	It("passes if all ports pass", func() {
		gomock.InOrder(
			expectPortAllowed("foo.example.com", 80).Return(true),
			expectPortAllowed("foo.example.com", 443).Return(true),
		)
		Expect(validate(portVs)).Should(Succeed())
	})

	It("fails if any ports fail", func() {
		gomock.InOrder(
			expectPortAllowed("foo.example.com", 80).Return(true),
			expectPortAllowed("foo.example.com", 443).Return(false),
		)
		Expect(validate(portVs)).ShouldNot(Succeed())
	})

	portPathVs := newPortPathVs("foo.example.com", 80, "/path1", "/path2")
	It("passes if all ports pass", func() {
		gomock.InOrder(
			mockV.EXPECT().IsConfigAllowed(&trafficclaim.Config{
				Host:      "foo.example.com",
				Port:      80,
				ExactPath: "/path1",
			}).Return(true),
			mockV.EXPECT().IsConfigAllowed(&trafficclaim.Config{
				Host:      "foo.example.com",
				Port:      80,
				ExactPath: "/path2",
			}).Return(true),
		)
		Expect(validate(portPathVs)).Should(Succeed())
	})

	It("fails if any ports fail", func() {
		gomock.InOrder(
			mockV.EXPECT().IsConfigAllowed(&trafficclaim.Config{
				Host:      "foo.example.com",
				Port:      80,
				ExactPath: "/path1",
			}).Return(true),
			mockV.EXPECT().IsConfigAllowed(&trafficclaim.Config{
				Host:      "foo.example.com",
				Port:      80,
				ExactPath: "/path2",
			}).Return(false),
		)
		Expect(validate(portPathVs)).ShouldNot(Succeed())
	})

	mixedPathMatches := []*networking.HTTPMatchRequest{
		&networking.HTTPMatchRequest{
			Port: 80,
			Uri: &networking.StringMatch{
				MatchType: &networking.StringMatch_Exact{Exact: "/exact1"},
			},
		},
		&networking.HTTPMatchRequest{
			Port: 80,
			Uri: &networking.StringMatch{
				MatchType: &networking.StringMatch_Prefix{Prefix: "/prefix/path"},
			},
		},
		&networking.HTTPMatchRequest{
			Port: 443,
			Uri: &networking.StringMatch{
				MatchType: &networking.StringMatch_Exact{Exact: "/prefix/exact2"},
			},
		},
	}
	mixedPathVs := newMultiRouteVs("foo.com", mixedPathMatches...)
	It("checks combinations of exact and prefix paths", func() {
		gomock.InOrder(
			mockV.EXPECT().IsConfigAllowed(&trafficclaim.Config{
				Host:      "foo.com",
				Port:      80,
				ExactPath: "/exact1",
			}).Return(true),
			mockV.EXPECT().IsConfigAllowed(&trafficclaim.Config{
				Host:       "foo.com",
				Port:       80,
				PrefixPath: "/prefix/path",
			}).Return(true),
			mockV.EXPECT().IsConfigAllowed(&trafficclaim.Config{
				Host:      "foo.com",
				Port:      443,
				ExactPath: "/prefix/exact2",
			}).Return(true),
		)
		Expect(validate(mixedPathVs)).Should(Succeed())
	})

	authorityMatches := []*networking.HTTPMatchRequest{
		&networking.HTTPMatchRequest{
			Port: 80,
			Authority: &networking.StringMatch{
				MatchType: &networking.StringMatch_Exact{Exact: "foo.example.com"},
			},
		},
		&networking.HTTPMatchRequest{
			Port: 80,
			Authority: &networking.StringMatch{
				MatchType: &networking.StringMatch_Prefix{Prefix: "foo"},
			},
		},
		&networking.HTTPMatchRequest{
			Port: 80,
			Authority: &networking.StringMatch{
				MatchType: &networking.StringMatch_Regex{Regex: ".*.example.com"},
			},
		},
	}
	authorityVs := newMultiRouteVs("", authorityMatches...)
	authorityVs.Hosts = []string{"foo.com", "foo.example.com", "bar.com"}
	It("handles authority matches", func() {
		gomock.InOrder(
			// Exact match authority narrows to only that host.
			mockV.EXPECT().IsConfigAllowed(&trafficclaim.Config{
				Host: "foo.example.com",
				Port: 80,
			}).Return(true),

			// Prefix match is the same as "all hosts"
			mockV.EXPECT().IsConfigAllowed(&trafficclaim.Config{
				Host: "foo.com",
				Port: 80,
			}).Return(true),
			mockV.EXPECT().IsConfigAllowed(&trafficclaim.Config{
				Host: "foo.example.com",
				Port: 80,
			}).Return(true),
			mockV.EXPECT().IsConfigAllowed(&trafficclaim.Config{
				Host: "bar.com",
				Port: 80,
			}).Return(true),

			// Regex match is the same as "all hosts"
			mockV.EXPECT().IsConfigAllowed(&trafficclaim.Config{
				Host: "foo.com",
				Port: 80,
			}).Return(true),
			mockV.EXPECT().IsConfigAllowed(&trafficclaim.Config{
				Host: "foo.example.com",
				Port: 80,
			}).Return(true),
			mockV.EXPECT().IsConfigAllowed(&trafficclaim.Config{
				Host: "bar.com",
				Port: 80,
			}).Return(true),
		)
		Expect(validate(authorityVs)).Should(Succeed())
	})
})
