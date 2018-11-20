package webhook

import (
	"fmt"
	"strings"

	"github.com/aspenmesh/tce/pkg/trafficclaim"
	"github.com/gogo/protobuf/proto"
	networking "istio.io/api/networking/v1alpha3"
)

func noClaimError(ns string, c *trafficclaim.Config) error {
	parms := ""
	if c.Host != "" {
		parms += fmt.Sprintf("host %s ", c.Host)
	}
	if c.Port != 0 {
		parms += fmt.Sprintf("port %d ", c.Port)
	}
	if c.ExactPath != "" {
		parms += fmt.Sprintf("path %s ", c.ExactPath)
	}
	if c.PrefixPath != "" {
		parms += fmt.Sprintf("path prefix %s ", c.PrefixPath)
	}
	return fmt.Errorf("No TrafficClaim in namespace %s grants %s",
		ns,
		strings.TrimSpace(parms),
	)
}

func validateVirtualService(ns string, s *webhookServer, spec proto.Message) error {
	vs, ok := spec.(*networking.VirtualService)
	if !ok {
		return fmt.Errorf("Failed to use spec as VirtualService")
	}
	v, err := s.claimDb.NewVerification(ns)
	if err != nil {
		return err
	}

	hosts := vs.GetHosts()

	for _, route := range vs.GetHttp() {

		// Matches apply to only a subset of traffic for a host.
		matches := route.GetMatch()

		// If a route has no match conditions, it applies to all traffic, and
		// a claim allowing config for the whole host is required.
		if len(matches) == 0 {
			for _, h := range hosts {
				config := trafficclaim.Config{Host: h}
				if !v.IsConfigAllowed(&config) {
					return noClaimError(ns, &config)
				}
			}

			// If we've checked for the entire host, no more narrow validation will
			// fail, so return early.
			return nil
		}

		for _, match := range matches {
			config := trafficclaim.Config{}
			if uri := match.GetUri(); uri != nil {
				switch u := uri.MatchType.(type) {
				case *networking.StringMatch_Exact:
					config.ExactPath = u.Exact
				case *networking.StringMatch_Prefix:
					config.PrefixPath = u.Prefix
				case *networking.StringMatch_Regex:
					// Too hard to figure out what the regex will match - we treat it as
					// "no path" (configuring the entire host+port)
				default:
					return fmt.Errorf("Unexpected Uri MatchType")
				}
			}

			if port := match.GetPort(); port != 0 {
				config.Port = port
			}

			// If an authority is specified and parseable then we'll consider that
			// as a more specific host match.
			var authHost string
			if authMatch := match.GetAuthority(); authMatch != nil {
				switch auth := authMatch.MatchType.(type) {
				case *networking.StringMatch_Exact:
					authHost = auth.Exact
				case *networking.StringMatch_Prefix:
					// Weird because for authorities you care about the suffix
					// e.g. if prefix: "foo" this would allow "foo.com",
					// "foo.example.com", "fooexample.com".
					// So, treat this as "all hosts"
				case *networking.StringMatch_Regex:
					// Too hard to figure out what the regex will match - we treat it as
					// "all hosts".
				default:
					return fmt.Errorf("Unexpected authority MatchType")
				}
			}

			if authHost != "" {
				config.Host = authHost
				if !v.IsConfigAllowed(&config) {
					return noClaimError(ns, &config)
				}
			} else {
				// No authority specified in the match condition so this applies to
				// all hosts.
				for _, h := range hosts {
					config.Host = h
					if !v.IsConfigAllowed(&config) {
						return noClaimError(ns, &config)
					}
				}
			}
		}
	}
	return nil
}
