package metrics

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

func TestCollectors_recordVerification(t *testing.T) {
	testCases := []struct {
		name   string
		record func(*collectors)
		assert func(*testing.T, *collectors)
	}{
		{
			name: "increments the counter for the given labels",
			record: func(c *collectors) {
				c.recordVerification("my-project", "test", "Successful")
			},
			assert: func(t *testing.T, c *collectors) {
				require.Equal(
					t, float64(1),
					testutil.ToFloat64(
						c.verificationsTotal.WithLabelValues("my-project", "test", "Successful"),
					),
				)
			},
		},
		{
			name: "distinguishes phases",
			record: func(c *collectors) {
				c.recordVerification("my-project", "test", "Failed")
				c.recordVerification("my-project", "test", "Successful")
				c.recordVerification("my-project", "test", "Successful")
			},
			assert: func(t *testing.T, c *collectors) {
				require.Equal(
					t, float64(1),
					testutil.ToFloat64(
						c.verificationsTotal.WithLabelValues("my-project", "test", "Failed"),
					),
				)
				require.Equal(
					t, float64(2),
					testutil.ToFloat64(
						c.verificationsTotal.WithLabelValues("my-project", "test", "Successful"),
					),
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			c := newCollectors(prometheus.NewRegistry())
			testCase.record(c)
			testCase.assert(t, c)
		})
	}
}

func TestCollectors_recordRecoveryDuration(t *testing.T) {
	c := newCollectors(prometheus.NewRegistry())
	c.recordRecoveryDuration("my-project", "test", 5*time.Minute)

	h, ok := c.recoveryDuration.WithLabelValues("my-project", "test").(prometheus.Metric)
	require.True(t, ok)

	m := &dto.Metric{}
	require.NoError(t, h.Write(m))
	require.Equal(t, uint64(1), m.GetHistogram().GetSampleCount())
	require.InDelta(t, 300.0, m.GetHistogram().GetSampleSum(), 0.001)
}
