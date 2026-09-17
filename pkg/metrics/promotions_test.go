package metrics

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

func TestCollectors_recordPromotion(t *testing.T) {
	testCases := []struct {
		name   string
		record func(*collectors)
		assert func(*testing.T, *collectors)
	}{
		{
			name: "increments the counter for the given labels",
			record: func(c *collectors) {
				c.recordPromotion("my-project", "test", "Succeeded", "admin")
			},
			assert: func(t *testing.T, c *collectors) {
				require.Equal(
					t, float64(1),
					testutil.ToFloat64(
						c.promotionsTotal.WithLabelValues("my-project", "test", "Succeeded", "admin"),
					),
				)
			},
		},
		{
			name: "distinguishes labels",
			record: func(c *collectors) {
				c.recordPromotion("my-project", "test", "Succeeded", "admin")
				c.recordPromotion("my-project", "test", "Failed", "admin")
			},
			assert: func(t *testing.T, c *collectors) {
				require.Equal(
					t, float64(1),
					testutil.ToFloat64(
						c.promotionsTotal.WithLabelValues("my-project", "test", "Succeeded", "admin"),
					),
				)
				require.Equal(
					t, float64(1),
					testutil.ToFloat64(
						c.promotionsTotal.WithLabelValues("my-project", "test", "Failed", "admin"),
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

func TestCollectors_recordPromotionLeadTime(t *testing.T) {
	c := newCollectors(prometheus.NewRegistry())
	c.recordPromotionLeadTime("my-project", "test", 90*time.Second)

	h, ok := c.promotionLeadTime.WithLabelValues("my-project", "test").(prometheus.Metric)
	require.True(t, ok)

	m := &dto.Metric{}
	require.NoError(t, h.Write(m))
	require.Equal(t, uint64(1), m.GetHistogram().GetSampleCount())
	require.InDelta(t, 90.0, m.GetHistogram().GetSampleSum(), 0.001)
}
