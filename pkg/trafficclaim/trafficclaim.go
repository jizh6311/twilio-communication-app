package trafficclaim

import (
	"fmt"
	"strings"

	crd "github.com/aspenmesh/tce/pkg/api/networking/v1alpha3"
	"github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Config describes a requested config that may be allowed by a TrafficClaim
// At most one of ExactPath or PrefixPath can be set.  If both are "", then
// no paths are specified - the config is listening to all paths.
type Config struct {
	// Namespace is required, it is where the config is being created
	Namespace string

	// Host is required, the hostname or glob that is being configured
	Host string

	// Port is optional, the port listening on (0: "all ports for this Host")
	Port uint32

	// ExactPath is optional, the exact path being listened.
	// At most one of ExactPath or PrefixPath can be specified.
	ExactPath string

	// PrefixPath is optional, the prefix of paths being listened
	// At most one of ExactPath or PrefixPath can be specified.
	PrefixPath string
}

//go:generate mockgen -destination=./mocks.go -package trafficclaim github.com/aspenmesh/tce/pkg/trafficclaim Db

// Db can check whether configs are allowed by the database of trafficclaims
type Db interface {
	// IsConfigAllowed returns true if the TrafficClaims allow config
	IsConfigAllowed(config *Config) bool

	// IsHostAllowed is shorthand for IsConfigAllowed for host configs
	IsHostAllowed(ns, host string) bool

	// IsPortAllowed is shorthand for IsConfigAllowed for host+port configs
	IsPortAllowed(ns, host string, port uint32) bool

	// IsPortPathAllowed is shorthand for IsConfigAllowed for host+port+exact path configs
	IsPortPathAllowed(ns, host string, port uint32, path string) bool
}

type db struct {
	kube Interface
}

func NewDb(kube Interface) Db {
	return &db{
		kube: kube,
	}
}

// hostGlobsCovered returns true if everything in sub is a subset of sup
func hostGlobsCovered(sup string, sub string) bool {
	// FIXME(andrew): Does Istio allow glob matching anywhere?
	// Is there a utility function for this?
	if len(sup) == 0 {
		return len(sub) == 0
	}
	if len(sub) == 0 {
		return false
	}
	if sup[0] == '*' && sub[0] == '*' {
		// Two globbed host specs - only works if sub's suffix covered by sup's
		subSuffix := sub[1:]
		supSuffix := sup[1:]
		return strings.HasSuffix(subSuffix, supSuffix)
	}
	if sup[0] == '*' {
		// sup is a glob - only works if sub is covered by sup.
		supSuffix := sup[1:]
		if supSuffix == sub {
			// sup was "*.foo.com" and sub is ".foo.com", not allowed
			return false
		}
		return strings.HasSuffix(sub, supSuffix)
	}
	if sub[0] == '*' {
		// Asking for a * from only a specific host sup
		return false
	}
	return sup == sub
}

func isHostValid(host string) bool {
	if len(host) == 0 {
		return false
	}
	deglobbed := strings.SplitN(host, "*", 3)
	if len(deglobbed) >= 3 {
		// There is more than one "*":  *foo.*bar.com
		return false
	}
	if len(deglobbed) == 0 {
		// Empty string
		return false
	}
	if len(deglobbed) == 1 {
		return host[0] != '.'
	}

	if len(deglobbed[0]) > 0 {
		// There is something before the first "*": foo*.bar.com
		return false
	}
	if len(deglobbed[1]) > 0 && deglobbed[1][0] != '.' {
		// The first thing after a * isn't a .: *foo.bar.com
		return false
	}
	return true
}

func (d *db) claimsForNamespace(ns string) (*crd.TrafficClaimList, error) {
	// FIXME: timeout, limit, continue
	listOpts := metav1.ListOptions{}
	return d.kube.Tc().NetworkingV1alpha3().TrafficClaims(ns).List(listOpts)
}

func evalConfigAgainstClaims(config *Config, claims []crd.Claim) bool {
	if config.Host == "" {
		glog.Errorf("Config host required")
		return false
	}
	if !isHostValid(config.Host) {
		glog.Errorf("Config host %s invalid", config.Host)
	}
	if config.ExactPath != "" && config.PrefixPath != "" {
		glog.Errorf("Can't check for exact path and prefix path in one check")
		return false
	}

	for _, c := range claims {
		var foundHost bool
		hosts := c.GetHosts()
		for _, h := range hosts {
			if hostGlobsCovered(h, config.Host) {
				foundHost = true
				break
			}
		}
		if !foundHost {
			continue
		}

		ports := c.GetPorts()
		if len(ports) == 0 {
			// This claim does not specify a port, so it allows ALL ports
		} else if config.Port == 0 {
			// This config is asking for all ports, but the claim only grants some
			continue
		} else {
			var foundPort bool
			for _, p := range ports {
				if p == config.Port {
					foundPort = true
					break
				}
			}
			if !foundPort {
				continue
			}
		}

		var paths *crd.Paths
		if http := c.GetHttp(); http != nil {
			paths = http.GetPaths()
		}
		if paths == nil || (len(paths.Exact) == 0 && len(paths.Prefix) == 0) {
			// This claim does not specify a path or prefix, so it allows ALL paths
		} else if config.ExactPath == "" && config.PrefixPath == "" {
			// This config is asking for all paths, but the claim only grants some
			continue
		} else if config.ExactPath != "" {
			// exact path can be granted by either exact path claim or prefix claim
			var foundPath bool
			for _, ep := range paths.Exact {
				if config.ExactPath == ep {
					foundPath = true
					break
				}
			}
			for _, pp := range paths.Prefix {
				if strings.HasPrefix(config.ExactPath, pp) {
					foundPath = true
					break
				}
			}
			if !foundPath {
				continue
			}
		} else {
			// prefix path can be granted by prefix path
			var foundPath bool
			for _, pp := range paths.Prefix {
				if strings.HasPrefix(config.PrefixPath, pp) {
					foundPath = true
					break
				}
			}
			if !foundPath {
				continue
			}
		}

		// Above validations were successful - this claim grants this config
		return true
	}
	// None of the claims granted a superset of what the config wants
	return false
}

func isHostAllowedByNamespaceDefaultRule(ns string, host string) bool {
	nsClaim := crd.Claim{
		Hosts: []string{fmt.Sprintf("*.%s.svc.cluster.local", ns)},
	}
	return evalConfigAgainstClaims(
		&Config{Namespace: ns, Host: host},
		[]crd.Claim{nsClaim},
	)
}

func (d *db) IsConfigAllowed(config *Config) bool {
	if isHostAllowedByNamespaceDefaultRule(config.Namespace, config.Host) {
		return true
	}

	// FIXME(andrew): Get the claims at most once for each validation.
	tcs, err := d.claimsForNamespace(config.Namespace)
	if err != nil {
		glog.Errorf("Failed to get trafficclaims from kubernetes: %v", err)
		return false
	}
	for _, tc := range tcs.Items {
		if evalConfigAgainstClaims(config, tc.Claims) {
			return true
		}
	}
	return false
}

func (d *db) IsHostAllowed(ns, host string) bool {
	return d.IsConfigAllowed(&Config{
		Namespace: ns,
		Host:      host,
	})
}

func (d *db) IsPortAllowed(ns, host string, port uint32) bool {
	return d.IsConfigAllowed(&Config{
		Namespace: ns,
		Host:      host,
		Port:      port,
	})
}

func (d *db) IsPortPathAllowed(ns, host string, port uint32, path string) bool {
	return d.IsConfigAllowed(&Config{
		Namespace: ns,
		Host:      host,
		Port:      port,
		ExactPath: path,
	})
}
