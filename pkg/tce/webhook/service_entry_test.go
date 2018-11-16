package webhook

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/aspenmesh/tce/pkg/trafficclaim"
	"github.com/golang/mock/gomock"
	networking "istio.io/api/networking/v1alpha3"
)

var _ = Describe("validate ServiceEntry", func() {
	var (
		mockDb *mockDb
		mockV  *trafficclaim.MockVerification
	)

	BeforeEach(func() {
		mockDb, mockV = newMockDb()
	})

	AfterEach(func() {
		mockDb.Finish()
	})

	validate := func(se *networking.ServiceEntry) error {
		return validateServiceEntry("tce-test", mockDb.server, se)
	}

	se := &networking.ServiceEntry{
		Hosts: []string{"foo.com", "bar.com"},
		Ports: []*networking.Port{
			&networking.Port{
				Number:   80,
				Protocol: "HTTP",
				Name:     "my-favorite-port",
			},
			&networking.Port{
				Number:   443,
				Protocol: "HTTPS",
				Name:     "my-true-favorite-port",
			},
		},
	}
	It("passes if all entries pass", func() {
		gomock.InOrder(
			mockV.EXPECT().IsPortAllowed("foo.com", uint32(80)).Return(true),
			mockV.EXPECT().IsPortAllowed("foo.com", uint32(443)).Return(true),
			mockV.EXPECT().IsPortAllowed("bar.com", uint32(80)).Return(true),
			mockV.EXPECT().IsPortAllowed("bar.com", uint32(443)).Return(true),
		)
		Expect(validate(se)).Should(Succeed())
	})

	It("fails if any host fails", func() {
		gomock.InOrder(
			mockV.EXPECT().IsPortAllowed("foo.com", uint32(80)).Return(true),
			mockV.EXPECT().IsPortAllowed("foo.com", uint32(443)).Return(true),
			mockV.EXPECT().IsPortAllowed("bar.com", uint32(80)).Return(true),
			mockV.EXPECT().IsPortAllowed("bar.com", uint32(443)).Return(false),
		)
		Expect(validate(se)).ShouldNot(Succeed())
	})
})
