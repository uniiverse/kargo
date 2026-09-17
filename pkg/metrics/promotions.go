package metrics

import "time"

// RecordPromotion increments the count of Promotions that reached a terminal
// phase for the given project (namespace), stage, phase, and initiator.
func RecordPromotion(project, stage, phase, initiator string) {
	defaultCollectors.recordPromotion(project, stage, phase, initiator)
}

func (c *collectors) recordPromotion(project, stage, phase, initiator string) {
	c.promotionsTotal.WithLabelValues(project, stage, phase, initiator).Inc()
}

// RecordPromotionLeadTime observes the lead time -- the duration between
// Freight discovery and a successful Promotion of that Freight into a
// Stage -- for the given project (namespace) and stage.
func RecordPromotionLeadTime(project, stage string, leadTime time.Duration) {
	defaultCollectors.recordPromotionLeadTime(project, stage, leadTime)
}

func (c *collectors) recordPromotionLeadTime(project, stage string, leadTime time.Duration) {
	c.promotionLeadTime.WithLabelValues(project, stage).Observe(leadTime.Seconds())
}
