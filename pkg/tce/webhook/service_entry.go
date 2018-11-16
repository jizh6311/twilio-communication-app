package webhook

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	networking "istio.io/api/networking/v1alpha3"
)

func validateServiceEntry(ns string, s *webhookServer, spec proto.Message) error {
	se, ok := spec.(*networking.ServiceEntry)
	if !ok {
		return fmt.Errorf("Failed to use spec as ServiceEntry")
	}

	v, err := s.claimDb.NewVerification(ns)
	if err != nil {
		return err
	}

	for _, h := range se.GetHosts() {
		for _, p := range se.GetPorts() {
			// An empty port specification is not "all ports", it is invalid
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

	// FIXME: We do not enforce addresses.  They are not used
	// for HTTP services anyway.

	return nil
}
