package webhook

import (
	"fmt"

	"github.com/aspenmesh/tce/pkg/trafficclaim"
	"github.com/gogo/protobuf/proto"
	networking "istio.io/api/networking/v1alpha3"
)

func evalTrafficPolicy(ns string, host string, v trafficclaim.Verification, tp *networking.TrafficPolicy) error {
	if tp == nil {
		return nil
	}

	// Examine host-wide behavior
	if tp.GetLoadBalancer() != nil ||
		tp.GetConnectionPool() != nil ||
		tp.GetOutlierDetection() != nil ||
		tp.GetTls() != nil {
		if !v.IsHostAllowed(host) {
			return fmt.Errorf(
				"No TrafficClaim in namespace %s grants host %s",
				ns,
				host,
			)
		}

		// If this is allowed for the entire host, any port is also allowed
		return nil
	}

	// Examine port-specific behavior
	for _, pls := range tp.GetPortLevelSettings() {
		port := pls.GetPort().GetNumber()
		if !v.IsPortAllowed(host, port) {
			return fmt.Errorf(
				"No TrafficClaim in namespace %s grants host %s port %d",
				ns,
				host,
				port,
			)
		}
	}
	return nil
}

func validateDestinationRule(ns string, s *webhookServer, spec proto.Message) error {
	dr, ok := spec.(*networking.DestinationRule)
	if !ok {
		return fmt.Errorf("Failed to use spec as DestinationRule")
	}

	v, err := s.claimDb.NewVerification(ns)
	if err != nil {
		return err
	}

	host := dr.GetHost()

	if err := evalTrafficPolicy(ns, host, v, dr.GetTrafficPolicy()); err != nil {
		return err
	}

	for _, subset := range dr.GetSubsets() {
		if err := evalTrafficPolicy(ns, host, v, subset.GetTrafficPolicy()); err != nil {
			return err
		}
	}

	return nil
}
