// Package metrics defines Prometheus metrics that describe the delivery
// performance of Kargo's promotion and verification processes, so that
// DORA-style measures (deployment frequency, lead time for changes, change
// failure rate, and mean time to recovery) can be computed from them via
// PromQL.
//
// Metrics are registered against controller-runtime's default metrics
// registry, so they are served on the same /metrics endpoint as
// controller-runtime's own reconciler metrics -- no separate registration
// step or endpoint is required.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

// durationBuckets spans one minute to roughly 11 days (60s * 2^14), doubling
// each time. Lead time and recovery duration are both measured on that
// scale, so the library's default buckets (5ms-10s) would be useless here.
var durationBuckets = prometheus.ExponentialBuckets(60, 2, 15)

type collectors struct {
	promotionsTotal    *prometheus.CounterVec
	promotionLeadTime  *prometheus.HistogramVec
	verificationsTotal *prometheus.CounterVec
	recoveryDuration   *prometheus.HistogramVec
}

func newCollectors(reg prometheus.Registerer) *collectors {
	f := promauto.With(reg)
	return &collectors{
		promotionsTotal: f.NewCounterVec(prometheus.CounterOpts{
			Name: "kargo_promotions_total",
			Help: "Total number of Promotions that reached a terminal phase, " +
				"by project, stage, phase, and initiator.",
		}, []string{"project", "stage", "phase", "initiator"}),
		promotionLeadTime: f.NewHistogramVec(prometheus.HistogramOpts{
			Name: "kargo_promotion_lead_time_seconds",
			Help: "Time from Freight discovery to a successful Promotion of " +
				"that Freight into a Stage, in seconds.",
			Buckets: durationBuckets,
		}, []string{"project", "stage"}),
		verificationsTotal: f.NewCounterVec(prometheus.CounterOpts{
			Name: "kargo_stage_verifications_total",
			Help: "Total number of Stage Freight verifications that reached " +
				"a terminal phase, by project, stage, and phase.",
		}, []string{"project", "stage", "phase"}),
		recoveryDuration: f.NewHistogramVec(prometheus.HistogramOpts{
			Name: "kargo_stage_recovery_duration_seconds",
			Help: "Time from a Failed or Error verification outcome to the " +
				"next Successful verification outcome for the same deployed " +
				"Freight, in seconds. Does not cover recovery via a new " +
				"Promotion of different Freight.",
			Buckets: durationBuckets,
		}, []string{"project", "stage"}),
	}
}

// defaultCollectors is registered against controller-runtime's default
// metrics registry as soon as this package is imported.
var defaultCollectors = newCollectors(ctrlmetrics.Registry)
