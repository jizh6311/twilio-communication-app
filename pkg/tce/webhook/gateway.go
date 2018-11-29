package webhook

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	networking "istio.io/api/networking/v1alpha3"
)

func validateGateway(ns string, s *webhookServer, spec proto.Message) error {
	gw, ok := spec.(*networking.Gateway)
	if !ok {
		return fmt.Errorf("Failed to use spec as Gateway")
	}

	v, err := s.claimDb.NewVerification(ns)
	if err != nil {
		return err
	}

	for _, s := range gw.GetServers() {
		p := s.GetPort()
		if p == nil {
			return fmt.Errorf("Port required")
		}
		for _, h := range s.GetHosts() {
			// An empty host specification is not "all hosts", it is invalid
			if !v.IsPortAllowed(h, p.Number) {
				return fmt.Errorf(
					"No TrafficClaim in namespace %s grants host %s port %d",
					ns,
					h,
					p.Number,
				)
			}
		}
	}
	return nil
}
