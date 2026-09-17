package metrics

import (
	"testing"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

// findSample locates, within the named metric family gathered from
// ctrlmetrics.Registry, the sample whose labels exactly match wantLabels.
func findSample(t *testing.T, family string, wantLabels map[string]string) *dto.Metric {
	t.Helper()
	families, err := ctrlmetrics.Registry.Gather()
	require.NoError(t, err)
	for _, mf := range families {
		if mf.GetName() != family {
			continue
		}
		for _, m := range mf.GetMetric() {
			gotLabels := make(map[string]string, len(m.GetLabel()))
			for _, l := range m.GetLabel() {
				gotLabels[l.GetName()] = l.GetValue()
			}
			match := len(gotLabels) == len(wantLabels)
			if match {
				for k, v := range wantLabels {
					if gotLabels[k] != v {
						match = false
						break
					}
				}
			}
			if match {
				return m
			}
		}
	}
	return nil
}

// These tests exercise the exported Record* functions -- as opposed to the
// unexported *collectors methods covered by promotions_test.go and
// verifications_test.go -- to confirm they are wired to the correct
// underlying collector and to controller-runtime's default metrics registry,
// the same registry served on the controller's /metrics endpoint.
func TestRecordPromotion_registersOnDefaultRegistry(t *testing.T) {
	RecordPromotion("metrics-test-project", "metrics-test-stage", "Succeeded", "metrics-test-initiator")

	m := findSample(t, "kargo_promotions_total", map[string]string{
		"project":   "metrics-test-project",
		"stage":     "metrics-test-stage",
		"phase":     "Succeeded",
		"initiator": "metrics-test-initiator",
	})
	require.NotNil(t, m, "expected a kargo_promotions_total sample with the recorded labels")
	require.Equal(t, float64(1), m.GetCounter().GetValue())
}

func TestRecordPromotionLeadTime_registersOnDefaultRegistry(t *testing.T) {
	RecordPromotionLeadTime("metrics-test-project", "metrics-test-lead-time-stage", 42*time.Second)

	m := findSample(t, "kargo_promotion_lead_time_seconds", map[string]string{
		"project": "metrics-test-project",
		"stage":   "metrics-test-lead-time-stage",
	})
	require.NotNil(t, m, "expected a kargo_promotion_lead_time_seconds sample with the recorded labels")
	require.Equal(t, uint64(1), m.GetHistogram().GetSampleCount())
	require.InDelta(t, 42.0, m.GetHistogram().GetSampleSum(), 0.001)
}

func TestRecordVerification_registersOnDefaultRegistry(t *testing.T) {
	RecordVerification("metrics-test-project", "metrics-test-verify-stage", "Failed")

	m := findSample(t, "kargo_stage_verifications_total", map[string]string{
		"project": "metrics-test-project",
		"stage":   "metrics-test-verify-stage",
		"phase":   "Failed",
	})
	require.NotNil(t, m, "expected a kargo_stage_verifications_total sample with the recorded labels")
	require.Equal(t, float64(1), m.GetCounter().GetValue())
}

func TestRecordRecoveryDuration_registersOnDefaultRegistry(t *testing.T) {
	RecordRecoveryDuration("metrics-test-project", "metrics-test-recovery-stage", 90*time.Second)

	m := findSample(t, "kargo_stage_recovery_duration_seconds", map[string]string{
		"project": "metrics-test-project",
		"stage":   "metrics-test-recovery-stage",
	})
	require.NotNil(t, m, "expected a kargo_stage_recovery_duration_seconds sample with the recorded labels")
	require.Equal(t, uint64(1), m.GetHistogram().GetSampleCount())
	require.InDelta(t, 90.0, m.GetHistogram().GetSampleSum(), 0.001)
}
