package webhook

import (
	"fmt"
	"math/rand"

	"github.com/gogo/protobuf/proto"
)

func validateVirtualService(ns string, s *webhookServer, spec proto.Message) error {
	// FIXME: This is a silly random reject-half for now.
	if rand.Int()%2 == 0 {
		return fmt.Errorf("Not allowed to configure virtualservice because TrafficClaim")
	}
	return nil
}
