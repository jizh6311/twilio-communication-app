package webhook

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/aspenmesh/tce/pkg/trafficclaim"
)

func TestProxy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "TrafficClaim")
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
