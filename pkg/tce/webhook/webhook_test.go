package webhook

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/aspenmesh/tce/pkg/trafficclaim"
	"github.com/golang/mock/gomock"
)

func TestProxy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "TrafficClaim")
}

// We don't use mocking for this because its only job in life is to return
// a mock verification - that's the thing we care about.
type mockDb struct {
	ctrl   *gomock.Controller
	server *webhookServer
	v      *trafficclaim.MockVerification
}

func newMockDb() (*mockDb, *trafficclaim.MockVerification) {
	ctrl := gomock.NewController(GinkgoT())
	v := trafficclaim.NewMockVerification(ctrl)
	db := &mockDb{ctrl: ctrl, v: v}
	db.server = &webhookServer{claimDb: db}
	return db, v
}

func (m *mockDb) Finish() {
	m.ctrl.Finish()
}

func (m *mockDb) NewVerification(ns string) (trafficclaim.Verification, error) {
	Expect(ns).To(Equal("tce-test"))
	return m.v, nil
}
