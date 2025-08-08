package ingestion

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/dgraph-io/ristretto"
)

// RoutingEngine manages organization-aware campaign routing
type RoutingEngine struct {
	clickhouse clickhouse.Conn
	cache      *ristretto.Cache
	mu         sync.RWMutex
	campaigns  map[string]*Campaign // org_id/campaign_id -> Campaign
}

// Campaign represents a traffic routing campaign
type Campaign struct {
	OrganizationID  string    `json:"organization_id"`
	CampaignID      string    `json:"campaign_id"`
	Name            string    `json:"name"`
	Status          string    `json:"status"`
	Rules           []Rule    `json:"rules"`
	DestinationURL  string    `json:"destination_url"`
	AppendParams    bool      `json:"append_params"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Rule defines campaign matching criteria
type Rule struct {
	Field     string      `json:"field"`      // source, medium, country, etc.
	Operator  string      `json:"operator"`   // equals, contains, in, regex
	Values    []string    `json:"values"`
	Priority  int         `json:"priority"`   // higher priority rules match first
}

// MatchResult contains routing decision information
type MatchResult struct {
	Campaign    *Campaign
	Matched     bool
	Rule        *Rule
	Destination string
}

// NewRoutingEngine creates a new routing engine
func NewRoutingEngine(ch clickhouse.Conn) (*RoutingEngine, error) {
	// Create cache for routing rules
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1000000,   // 10x expected entries
		MaxCost:     104857600, // 100MB
		BufferItems: 64,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create cache: %w", err)
	}

	re := &RoutingEngine{
		clickhouse: ch,
		cache:      cache,
		campaigns:  make(map[string]*Campaign),
	}

	// Load initial campaigns
	if err := re.loadCampaigns(context.Background()); err != nil {
		slog.Warn("failed to load initial campaigns", "error", err)
	}

	// Start background campaign refresh
	go re.refreshCampaigns()

	return re, nil
}

// GetDestination determines the destination URL for a request
func (re *RoutingEngine) GetDestination(organizationID, campaignID string, params map[string][]string) string {
	// If campaign is explicitly specified, use it
	if campaignID != "" {
		key := fmt.Sprintf("%s/%s", organizationID, campaignID)
		if campaign := re.getCampaign(key); campaign != nil && campaign.Status == "active" {
			return re.buildDestinationURL(campaign, params)
		}
	}

	// Otherwise, find best matching campaign
	campaign := re.findBestMatch(organizationID, params)
	if campaign != nil {
		return re.buildDestinationURL(campaign, params)
	}

	// Default fallback - try to find default campaign for organization
	defaultKey := fmt.Sprintf("%s/default", organizationID)
	if defaultCampaign := re.getCampaign(defaultKey); defaultCampaign != nil {
		return re.buildDestinationURL(defaultCampaign, params)
	}

	// Ultimate fallback
	return "https://example.com/"
}

// findBestMatch finds the best matching campaign for the given parameters
func (re *RoutingEngine) findBestMatch(organizationID string, params map[string][]string) *Campaign {
	re.mu.RLock()
	defer re.mu.RUnlock()

	var bestMatch *Campaign
	var bestScore int

	// Convert params to flat map for easier matching
	flatParams := make(map[string]string)
	for key, values := range params {
		if len(values) > 0 {
			flatParams[key] = values[0]
		}
	}

	// Check each campaign for this organization
	for key, campaign := range re.campaigns {
		// Only consider campaigns for this organization
		if !strings.HasPrefix(key, organizationID+"/") {
			continue
		}

		if campaign.Status != "active" {
			continue
		}

		score := re.calculateMatchScore(campaign, flatParams)
		if score > bestScore {
			bestMatch = campaign
			bestScore = score
		}
	}

	return bestMatch
}

// calculateMatchScore calculates how well a campaign matches the parameters
func (re *RoutingEngine) calculateMatchScore(campaign *Campaign, params map[string]string) int {
	score := 0

	for _, rule := range campaign.Rules {
		if re.ruleMatches(rule, params) {
			score += rule.Priority
		}
	}

	return score
}

// ruleMatches checks if a rule matches the given parameters
func (re *RoutingEngine) ruleMatches(rule Rule, params map[string]string) bool {
	paramValue, exists := params[rule.Field]
	if !exists {
		return false
	}

	switch rule.Operator {
	case "equals":
		for _, value := range rule.Values {
			if paramValue == value {
				return true
			}
		}
	case "contains":
		for _, value := range rule.Values {
			if strings.Contains(strings.ToLower(paramValue), strings.ToLower(value)) {
				return true
			}
		}
	case "in":
		for _, value := range rule.Values {
			if paramValue == value {
				return true
			}
		}
	case "prefix":
		for _, value := range rule.Values {
			if strings.HasPrefix(paramValue, value) {
				return true
			}
		}
	}

	return false
}

// buildDestinationURL creates the final destination URL with optional parameter appending
func (re *RoutingEngine) buildDestinationURL(campaign *Campaign, params map[string][]string) string {
	baseURL := campaign.DestinationURL

	if !campaign.AppendParams || len(params) == 0 {
		return baseURL
	}

	// Parse base URL
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		slog.Warn("invalid destination URL", "url", baseURL, "error", err)
		return baseURL
	}

	// Add parameters
	query := parsedURL.Query()
	for key, values := range params {
		for _, value := range values {
			query.Add(key, value)
		}
	}

	parsedURL.RawQuery = query.Encode()
	return parsedURL.String()
}

// getCampaign retrieves a campaign from cache or database
func (re *RoutingEngine) getCampaign(key string) *Campaign {
	re.mu.RLock()
	campaign, exists := re.campaigns[key]
	re.mu.RUnlock()

	if exists {
		return campaign
	}

	return nil
}

// loadCampaigns loads campaigns from ClickHouse
func (re *RoutingEngine) loadCampaigns(ctx context.Context) error {
	query := `
		SELECT 
			organization_id,
			campaign_id,
			name,
			status,
			rules,
			destination_url,
			append_params,
			created_at,
			updated_at
		FROM campaigns 
		WHERE status = 'active'
		ORDER BY organization_id, campaign_id
	`

	rows, err := re.clickhouse.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query campaigns: %w", err)
	}
	defer rows.Close()

	campaigns := make(map[string]*Campaign)

	for rows.Next() {
		var campaign Campaign
		var rulesJSON string

		err := rows.Scan(
			&campaign.OrganizationID,
			&campaign.CampaignID,
			&campaign.Name,
			&campaign.Status,
			&rulesJSON,
			&campaign.DestinationURL,
			&campaign.AppendParams,
			&campaign.CreatedAt,
			&campaign.UpdatedAt,
		)
		if err != nil {
			slog.Warn("failed to scan campaign row", "error", err)
			continue
		}

		// Parse rules JSON
		if err := json.Unmarshal([]byte(rulesJSON), &campaign.Rules); err != nil {
			slog.Warn("failed to parse campaign rules", 
				"campaign_id", campaign.CampaignID, 
				"error", err)
			continue
		}

		key := fmt.Sprintf("%s/%s", campaign.OrganizationID, campaign.CampaignID)
		campaigns[key] = &campaign
	}

	// Update campaigns atomically
	re.mu.Lock()
	re.campaigns = campaigns
	re.mu.Unlock()

	slog.Info("loaded campaigns", "count", len(campaigns))
	return nil
}

// refreshCampaigns periodically refreshes campaign data
func (re *RoutingEngine) refreshCampaigns() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			if err := re.loadCampaigns(ctx); err != nil {
				slog.Error("failed to refresh campaigns", "error", err)
			}
			cancel()
		}
	}
}

// CreateCampaign creates a new campaign in the database
func (re *RoutingEngine) CreateCampaign(ctx context.Context, campaign *Campaign) error {
	rulesJSON, err := json.Marshal(campaign.Rules)
	if err != nil {
		return fmt.Errorf("failed to marshal rules: %w", err)
	}

	query := `
		INSERT INTO campaigns (
			organization_id, campaign_id, name, status, rules, 
			destination_url, append_params, created_by
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	err = re.clickhouse.Exec(ctx, query,
		campaign.OrganizationID,
		campaign.CampaignID,
		campaign.Name,
		campaign.Status,
		string(rulesJSON),
		campaign.DestinationURL,
		campaign.AppendParams,
		"api", // created_by - could be extracted from auth context
	)

	if err != nil {
		return fmt.Errorf("failed to create campaign: %w", err)
	}

	// Update local cache
	key := fmt.Sprintf("%s/%s", campaign.OrganizationID, campaign.CampaignID)
	re.mu.Lock()
	re.campaigns[key] = campaign
	re.mu.Unlock()

	return nil
}

// UpdateCampaign updates an existing campaign
func (re *RoutingEngine) UpdateCampaign(ctx context.Context, campaign *Campaign) error {
	rulesJSON, err := json.Marshal(campaign.Rules)
	if err != nil {
		return fmt.Errorf("failed to marshal rules: %w", err)
	}

	query := `
		ALTER TABLE campaigns UPDATE 
			name = ?, 
			status = ?, 
			rules = ?, 
			destination_url = ?, 
			append_params = ?, 
			updated_at = now64(3)
		WHERE organization_id = ? AND campaign_id = ?
	`

	err = re.clickhouse.Exec(ctx, query,
		campaign.Name,
		campaign.Status,
		string(rulesJSON),
		campaign.DestinationURL,
		campaign.AppendParams,
		campaign.OrganizationID,
		campaign.CampaignID,
	)

	if err != nil {
		return fmt.Errorf("failed to update campaign: %w", err)
	}

	// Update local cache
	key := fmt.Sprintf("%s/%s", campaign.OrganizationID, campaign.CampaignID)
	re.mu.Lock()
	re.campaigns[key] = campaign
	re.mu.Unlock()

	return nil
}

// DeleteCampaign marks a campaign as deleted
func (re *RoutingEngine) DeleteCampaign(ctx context.Context, organizationID, campaignID string) error {
	query := `
		ALTER TABLE campaigns UPDATE 
			status = 'deleted',
			updated_at = now64(3)
		WHERE organization_id = ? AND campaign_id = ?
	`

	err := re.clickhouse.Exec(ctx, query, organizationID, campaignID)
	if err != nil {
		return fmt.Errorf("failed to delete campaign: %w", err)
	}

	// Remove from local cache
	key := fmt.Sprintf("%s/%s", organizationID, campaignID)
	re.mu.Lock()
	delete(re.campaigns, key)
	re.mu.Unlock()

	return nil
}

// GetOrganizationCampaigns returns all campaigns for an organization
func (re *RoutingEngine) GetOrganizationCampaigns(organizationID string) []*Campaign {
	re.mu.RLock()
	defer re.mu.RUnlock()

	var campaigns []*Campaign
	prefix := organizationID + "/"

	for key, campaign := range re.campaigns {
		if strings.HasPrefix(key, prefix) {
			campaigns = append(campaigns, campaign)
		}
	}

	return campaigns
}