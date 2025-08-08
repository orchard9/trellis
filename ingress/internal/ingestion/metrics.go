package ingestion

import (
	"time"
	"log/slog"
)

// Metrics interface for recording ingestion metrics
type Metrics interface {
	RecordRedirect(duration time.Duration, organizationID, campaignID string)
	RecordEvent(organizationID string)
	RecordDuplicate(organizationID string)
	RecordFraud(organizationID, fraudType string)
}

// SimpleMetrics provides basic logging-based metrics
type SimpleMetrics struct{}

// NewSimpleMetrics creates a new simple metrics instance
func NewSimpleMetrics() *SimpleMetrics {
	return &SimpleMetrics{}
}

// RecordRedirect logs redirect timing and campaign information
func (m *SimpleMetrics) RecordRedirect(duration time.Duration, organizationID, campaignID string) {
	slog.Info("redirect completed",
		"duration_ms", duration.Milliseconds(),
		"organization_id", organizationID,
		"campaign_id", campaignID,
	)
}

// RecordEvent logs event processing
func (m *SimpleMetrics) RecordEvent(organizationID string) {
	slog.Debug("event processed",
		"organization_id", organizationID,
	)
}

// RecordDuplicate logs duplicate detection
func (m *SimpleMetrics) RecordDuplicate(organizationID string) {
	slog.Info("duplicate detected",
		"organization_id", organizationID,
	)
}

// RecordFraud logs fraud detection
func (m *SimpleMetrics) RecordFraud(organizationID, fraudType string) {
	slog.Warn("fraud detected",
		"organization_id", organizationID,
		"fraud_type", fraudType,
	)
}