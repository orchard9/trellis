package ingestion

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"log/slog"

	"cloud.google.com/go/pubsub"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/orchard9/trellis/internal/auth"
	"github.com/redis/go-redis/v9"
)

// Handler manages traffic ingestion with organization awareness
type Handler struct {
	pubsub   *pubsub.Topic
	redis    *redis.Client
	routing  *RoutingEngine
	metrics  *Metrics
}

// Event represents a traffic event with organization context
type Event struct {
	EventID        string            `json:"event_id"`
	Timestamp      int64             `json:"timestamp"`
	OrganizationID string            `json:"organization_id"`
	ClickID        string            `json:"click_id"`
	CampaignID     string            `json:"campaign_id,omitempty"`
	RawRequest     RawRequest        `json:"raw_request"`
	Enriched       EnrichedData      `json:"enriched,omitempty"`
	FraudFlags     []string          `json:"fraud_flags,omitempty"`
	FraudScore     float32           `json:"fraud_score,omitempty"`
}

// RawRequest contains the complete HTTP request information
type RawRequest struct {
	Method  string              `json:"method"`
	URL     string              `json:"url"`
	Path    string              `json:"path"`
	Headers map[string]string   `json:"headers"`
	Body    json.RawMessage     `json:"body,omitempty"`
	IP      string              `json:"ip"`
	Params  map[string][]string `json:"params"`
}

// EnrichedData contains processed information
type EnrichedData struct {
	Country         string  `json:"country,omitempty"`
	City            string  `json:"city,omitempty"`
	DeviceType      string  `json:"device_type,omitempty"`
	OS              string  `json:"os,omitempty"`
	Browser         string  `json:"browser,omitempty"`
	IsBot           bool    `json:"is_bot,omitempty"`
	Source          string  `json:"source,omitempty"`
	Medium          string  `json:"medium,omitempty"`
	Referrer        string  `json:"referrer,omitempty"`
	ReferrerDomain  string  `json:"referrer_domain,omitempty"`
}

// NewHandler creates a new ingestion handler
func NewHandler(pubsubTopic *pubsub.Topic, redisClient *redis.Client, routing *RoutingEngine, metrics *Metrics) *Handler {
	return &Handler{
		pubsub:  pubsubTopic,
		redis:   redisClient,
		routing: routing,
		metrics: metrics,
	}
}

// HandleTraffic processes incoming traffic with organization awareness
func (h *Handler) HandleTraffic(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := r.Context()

	// Extract organization context from authentication
	orgCtx, ok := auth.GetOrganizationContext(ctx)
	if !ok {
		http.Error(w, "Organization context not found", http.StatusInternalServerError)
		return
	}

	// Create event with organization context
	event := &Event{
		EventID:        uuid.New().String(),
		Timestamp:      time.Now().UnixNano(),
		OrganizationID: orgCtx.OrganizationID,
		ClickID:        h.extractClickID(r),
		RawRequest: RawRequest{
			Method:  r.Method,
			URL:     r.URL.String(),
			Path:    r.URL.Path,
			Headers: h.flattenHeaders(r.Header),
			Body:    h.readBody(r),
			IP:      h.getRealIP(r),
			Params:  r.URL.Query(),
		},
	}

	// Extract campaign ID from route (organization-scoped)
	if campaignID := chi.URLParam(r, "campaign_id"); campaignID != "" {
		event.CampaignID = fmt.Sprintf("%s/%s", orgCtx.OrganizationID, campaignID)
	}

	// Async publish to Pub/Sub
	go func(e *Event) {
		publishCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		data, err := json.Marshal(e)
		if err != nil {
			slog.Error("failed to marshal event", "error", err, "organization_id", e.OrganizationID)
			return
		}

		result := h.pubsub.Publish(publishCtx, &pubsub.Message{
			Data: data,
			Attributes: map[string]string{
				"event_id":        e.EventID,
				"click_id":        e.ClickID,
				"campaign_id":     e.CampaignID,
				"organization_id": e.OrganizationID,
			},
		})

		if _, err := result.Get(publishCtx); err != nil {
			slog.Error("failed to publish event", 
				"error", err, 
				"event_id", e.EventID, 
				"organization_id", e.OrganizationID)
		}
	}(event)

	// Organization-scoped deduplication check
	if h.isDuplicate(ctx, event.OrganizationID, event.ClickID) {
		event.FraudFlags = append(event.FraudFlags, "duplicate_click")
	}

	// Get destination from organization-aware routing
	destination := h.routing.GetDestination(event.OrganizationID, event.CampaignID, event.RawRequest.Params)

	// Record metrics with organization context
	h.metrics.RecordRedirect(time.Since(start), event.OrganizationID, event.CampaignID)

	// Perform redirect
	http.Redirect(w, r, destination, http.StatusFound)
}

// HandlePixel processes pixel tracking requests
func (h *Handler) HandlePixel(w http.ResponseWriter, r *http.Request) {
	// Extract organization context
	orgCtx, ok := auth.GetOrganizationContext(r.Context())
	if !ok {
		// For pixel tracking, we might want to be more lenient
		// Return 1x1 transparent gif even on auth failure
		h.servePixel(w)
		return
	}

	// Create pixel event
	event := &Event{
		EventID:        uuid.New().String(),
		Timestamp:      time.Now().UnixNano(),
		OrganizationID: orgCtx.OrganizationID,
		ClickID:        h.extractClickID(r),
		RawRequest: RawRequest{
			Method:  "GET",
			URL:     r.URL.String(),
			Path:    r.URL.Path,
			Headers: h.flattenHeaders(r.Header),
			IP:      h.getRealIP(r),
			Params:  r.URL.Query(),
		},
	}

	// Async publish
	go h.publishEvent(event)

	// Serve 1x1 transparent gif
	h.servePixel(w)
}

// HandlePostback processes postback/conversion tracking
func (h *Handler) HandlePostback(w http.ResponseWriter, r *http.Request) {
	orgCtx, ok := auth.GetOrganizationContext(r.Context())
	if !ok {
		http.Error(w, "Organization context not found", http.StatusUnauthorized)
		return
	}

	// Extract postback parameters
	clickID := r.URL.Query().Get("click_id")
	if clickID == "" {
		http.Error(w, "Missing click_id parameter", http.StatusBadRequest)
		return
	}

	// Create postback event
	event := &Event{
		EventID:        uuid.New().String(),
		Timestamp:      time.Now().UnixNano(),
		OrganizationID: orgCtx.OrganizationID,
		ClickID:        clickID,
		RawRequest: RawRequest{
			Method:  r.Method,
			URL:     r.URL.String(),
			Path:    r.URL.Path,
			Headers: h.flattenHeaders(r.Header),
			Body:    h.readBody(r),
			IP:      h.getRealIP(r),
			Params:  r.URL.Query(),
		},
	}

	// Async publish
	go h.publishEvent(event)

	// Return success
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// isDuplicate checks for duplicate clicks within organization scope
func (h *Handler) isDuplicate(ctx context.Context, organizationID, clickID string) bool {
	if clickID == "" {
		return false
	}

	// Organization-scoped Redis key
	key := fmt.Sprintf("click:%s:%s", organizationID, clickID)

	// Use short context for Redis operation
	redisCtx, cancel := context.WithTimeout(ctx, 10*time.Millisecond)
	defer cancel()

	// SetNX returns true if key was set (not duplicate)
	ok, err := h.redis.SetNX(redisCtx, key, 1, 5*time.Second).Result()
	if err != nil {
		// Log error but don't block redirect
		slog.Warn("redis dedup check failed", "error", err, "organization_id", organizationID)
		return false
	}

	return !ok // Return true if key already existed (duplicate)
}

// publishEvent publishes event to Pub/Sub
func (h *Handler) publishEvent(event *Event) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	data, err := json.Marshal(event)
	if err != nil {
		slog.Error("failed to marshal event", "error", err, "organization_id", event.OrganizationID)
		return
	}

	result := h.pubsub.Publish(ctx, &pubsub.Message{
		Data: data,
		Attributes: map[string]string{
			"event_id":        event.EventID,
			"click_id":        event.ClickID,
			"organization_id": event.OrganizationID,
		},
	})

	if _, err := result.Get(ctx); err != nil {
		slog.Error("failed to publish event", 
			"error", err, 
			"event_id", event.EventID,
			"organization_id", event.OrganizationID)
	}
}

// extractClickID extracts click ID from various parameter names
func (h *Handler) extractClickID(r *http.Request) string {
	// Try common parameter names
	params := []string{"click_id", "clickid", "cid", "transaction_id", "tid"}

	for _, param := range params {
		if id := r.URL.Query().Get(param); id != "" {
			return id
		}
	}

	// Generate one if not found
	return uuid.New().String()
}

// getRealIP extracts the real IP address from headers
func (h *Handler) getRealIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Return first IP in the chain
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return xff
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	if idx := strings.LastIndex(r.RemoteAddr, ":"); idx != -1 {
		return r.RemoteAddr[:idx]
	}

	return r.RemoteAddr
}

// flattenHeaders converts http.Header to map[string]string
func (h *Handler) flattenHeaders(headers http.Header) map[string]string {
	flat := make(map[string]string)
	for key, values := range headers {
		if len(values) > 0 {
			flat[strings.ToLower(key)] = values[0]
		}
	}
	return flat
}

// readBody reads and returns request body
func (h *Handler) readBody(r *http.Request) json.RawMessage {
	if r.Body == nil {
		return nil
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Warn("failed to read request body", "error", err)
		return nil
	}

	// Reset body for potential future reads
	r.Body = io.NopCloser(strings.NewReader(string(body)))

	return json.RawMessage(body)
}

// servePixel serves a 1x1 transparent gif
func (h *Handler) servePixel(w http.ResponseWriter) {
	// 1x1 transparent gif
	pixel := []byte{
		0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00,
		0x01, 0x00, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x21, 0xF9, 0x04, 0x01, 0x00,
		0x00, 0x00, 0x00, 0x2C, 0x00, 0x00, 0x00, 0x00,
		0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x02, 0x04,
		0x01, 0x00, 0x3B,
	}

	w.Header().Set("Content-Type", "image/gif")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.WriteHeader(http.StatusOK)
	w.Write(pixel)
}