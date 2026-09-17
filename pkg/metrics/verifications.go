package metrics

import "time"

// RecordVerification increments the count of Stage Freight verifications
// that reached a terminal phase for the given project (namespace), stage,
// and phase.
func RecordVerification(project, stage, phase string) {
	defaultCollectors.recordVerification(project, stage, phase)
}

func (c *collectors) recordVerification(project, stage, phase string) {
	c.verificationsTotal.WithLabelValues(project, stage, phase).Inc()
}

// RecordRecoveryDuration observes the duration between a Failed or Error
// verification outcome and the next Successful verification outcome for the
// given project (namespace) and stage.
func RecordRecoveryDuration(project, stage string, d time.Duration) {
	defaultCollectors.recordRecoveryDuration(project, stage, d)
}

func (c *collectors) recordRecoveryDuration(project, stage string, d time.Duration) {
	c.recoveryDuration.WithLabelValues(project, stage).Observe(d.Seconds())
}
