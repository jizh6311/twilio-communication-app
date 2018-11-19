package webhook

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	AllowedMetric = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "traffic_claim_enforcer_allowed",
			Help: "Number of resources allowed",
		},
		[]string{"namespace", "group", "kind"},
	)
	RejectedMetric = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "traffic_claim_enforcer_rejected",
			Help: "Number of resources rejected",
		},
		[]string{"namespace", "group", "kind", "reason"},
	)
	HTTPErrorMetric = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "traffic_claim_enforcer_http_error",
			Help: "Number of HTTP requests failed due to HTTP or parse errors",
		},
		[]string{"reason"},
	)
)

func init() {
	prometheus.MustRegister(AllowedMetric, RejectedMetric)
}
