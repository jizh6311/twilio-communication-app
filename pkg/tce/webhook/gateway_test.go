package webhook

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/aspenmesh/tce/pkg/trafficclaim"
	"github.com/golang/mock/gomock"
	networking "istio.io/api/networking/v1alpha3"
)

var _ = Describe("validate Gateway", func() {
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

	validate := func(gw *networking.Gateway) error {
		return validateGateway("tce-test", server, gw)
	}

	gw := &networking.Gateway{
		Servers: []*networking.Server{
			&networking.Server{
				Port: &networking.Port{
					Number:   80,
					Protocol: "HTTP",
					Name:     "my-favorite-port",
				},
				Hosts: []string{"foo.com", "bar.com"},
			},
			&networking.Server{
				Port: &networking.Port{
					Number:   443,
					Protocol: "HTTPS",
					Name:     "my-true-favorite-port",
				},
				Hosts: []string{"foo.com", "baz.com"},
			},
		},
	}

	It("passes if all hosts pass", func() {
		gomock.InOrder(
			mockV.EXPECT().IsPortAllowed("foo.com", uint32(80)).Return(true),
			mockV.EXPECT().IsPortAllowed("bar.com", uint32(80)).Return(true),
			mockV.EXPECT().IsPortAllowed("foo.com", uint32(443)).Return(true),
			mockV.EXPECT().IsPortAllowed("baz.com", uint32(443)).Return(true),
		)
		Expect(validate(gw)).Should(Succeed())
	})

	It("fails if any host fails", func() {
		gomock.InOrder(
			mockV.EXPECT().IsPortAllowed("foo.com", uint32(80)).Return(true),
			mockV.EXPECT().IsPortAllowed("bar.com", uint32(80)).Return(true),
			mockV.EXPECT().IsPortAllowed("foo.com", uint32(443)).Return(true),
			mockV.EXPECT().IsPortAllowed("baz.com", uint32(443)).Return(false),
		)
		Expect(validate(gw)).ShouldNot(Succeed())
	})
})
