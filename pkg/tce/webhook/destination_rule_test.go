package webhook

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/aspenmesh/tce/pkg/trafficclaim"
	"github.com/golang/mock/gomock"
	networking "istio.io/api/networking/v1alpha3"
)

var _ = Describe("validate DestinationRule", func() {
	var (
		mockV  *trafficclaim.MockVerification
		mockDb *mockDb
	)

	BeforeEach(func() {
		mockDb, mockV = newMockDb()
	})

	AfterEach(func() {
		mockDb.Finish()
	})

	validate := func(dr *networking.DestinationRule) error {
		return validateDestinationRule("tce-test", mockDb.server, dr)
	}

	hostDr := &networking.DestinationRule{
		Host: "foo.com",
		TrafficPolicy: &networking.TrafficPolicy{
			PortLevelSettings: []*networking.TrafficPolicy_PortTrafficPolicy{
				&networking.TrafficPolicy_PortTrafficPolicy{
					Port: &networking.PortSelector{
						Port: &networking.PortSelector_Number{Number: 80},
					},
					LoadBalancer: &networking.LoadBalancerSettings{
						LbPolicy: &networking.LoadBalancerSettings_Simple{},
					},
				},
			},
			OutlierDetection: &networking.OutlierDetection{
				ConsecutiveErrors: 47,
			},
		},
	}
	It("passes if host-wide and host passes", func() {
		gomock.InOrder(
			mockV.EXPECT().IsHostAllowed("foo.com").Return(true),
			// If host is allowed, all ports are allowed.
		)
		Expect(validate(hostDr)).Should(Succeed())
	})
	It("fails if host-wide and host fails", func() {
		gomock.InOrder(
			mockV.EXPECT().IsHostAllowed("foo.com").Return(false),
			// If host is allowed, all ports are allowed.
		)
		Expect(validate(hostDr)).ShouldNot(Succeed())
	})

	portDr := &networking.DestinationRule{
		Host: "foo.com",
		TrafficPolicy: &networking.TrafficPolicy{
			PortLevelSettings: []*networking.TrafficPolicy_PortTrafficPolicy{
				&networking.TrafficPolicy_PortTrafficPolicy{
					Port: &networking.PortSelector{
						Port: &networking.PortSelector_Number{Number: 80},
					},
					LoadBalancer: &networking.LoadBalancerSettings{
						LbPolicy: &networking.LoadBalancerSettings_Simple{},
					},
				},
			},
		},
	}
	It("passes if port-specific and port passes", func() {
		gomock.InOrder(
			mockV.EXPECT().IsPortAllowed("foo.com", uint32(80)).Return(true),
		)
		Expect(validate(portDr)).Should(Succeed())
	})
	It("fails if port-specific and port fails", func() {
		gomock.InOrder(
			mockV.EXPECT().IsPortAllowed("foo.com", uint32(80)).Return(false),
		)
		Expect(validate(portDr)).ShouldNot(Succeed())
	})
})
