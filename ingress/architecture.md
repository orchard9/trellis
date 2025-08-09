# Trellis Ingress Architecture

## System Overview

Trellis Ingress is the traffic routing and data capture service that redirects traffic based on campaign rules while storing every piece of traffic data for later analysis.

```
┌─────────────────────────────────────────────────────────────────┐
│                       INGRESS ARCHITECTURE                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────┐      ┌──────────┐      ┌──────────┐             │
│  │  HTTP    │─────▶│  Ingress │─────▶│ Campaign │─────┐       │
│  │ Request  │      │    API   │      │  Router  │     │       │
│  └──────────┘      └──────────┘      └──────────┘     │       │
│                           │                            │       │
│                           ▼                            ▼       │
│                    ┌──────────┐                 ┌──────────┐  │
│                    │  Warden  │                 │ Redirect │  │
│                    │   Auth   │                 │ Response │  │
│                    └──────────┘                 └──────────┘  │
│                           │                                    │
│                           ▼                                    │
│                    ┌──────────────────────┐                   │
│                    │   Data Storage       │                   │
│                    ├──────────┬───────────┤                   │
│                    │ClickHouse│PostgreSQL │                   │
│                    └──────────┴───────────┘                   │
└─────────────────────────────────────────────────────────────────┘
```

## Core Functionality

The ingress service has two primary responsibilities:
1. **Route traffic** based on campaign rules (<50ms redirect latency)
2. **Capture all traffic data** for retroactive analysis

### 1. Traffic Routing & Data Capture

```go
// GET /in - Main traffic ingestion endpoint with redirect
// GET /in/{campaign_id} - Campaign-specific ingestion
// POST /postback - Conversion tracking endpoint
type TrafficHandler struct {
    router         *CampaignRouter
    asyncProcessor *AsyncProcessor
    userTracker    *UserTracker
}

func (h *TrafficHandler) HandleTraffic(w http.ResponseWriter, r *http.Request) {
    start := time.Now()
    
    // 1. Extract organization from API key
    orgCtx, err := h.extractOrganization(r)
    if err != nil {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    
    // 2. Generate or retrieve user tracking ID
    userID := h.userTracker.GetOrCreateUserID(r, orgCtx)
    
    // 3. Capture all request data
    event := h.captureEvent(r, orgCtx, userID)
    
    // 4. Determine redirect destination based on campaign rules
    destination, campaign := h.router.GetDestination(orgCtx, event)
    event.CampaignID = campaign.ID
    
    // 5. Store event asynchronously (non-blocking)
    h.asyncProcessor.ProcessEvent(event)
    
    // 6. Redirect user (<50ms target)
    h.recordMetrics(time.Since(start), orgCtx.ID, campaign.ID)
    http.Redirect(w, r, destination, http.StatusFound)
}
```

### 2. Campaign Routing Engine

Campaigns have flexible routing rules that can be based on:
- Direct 1:1 campaign to URL mapping
- User recognition (returning vs new)
- GeoIP location
- Device type
- Time-based conditions
- Custom rule expressions

```go
type Campaign struct {
    ID             string
    OrganizationID string
    Name           string
    Status         string // active, paused, archived
    Destination    string // Default destination URL
    Rules          []Rule // Ordered list of routing rules
}

type Rule struct {
    Type       string      // "direct", "user", "geo", "device", "time", "custom"
    Conditions []Condition // AND conditions
    Action     Action      // What to do when conditions match
    Priority   int         // Rule evaluation order
}

type Condition struct {
    Field    string      // e.g., "country", "device_type", "hour_of_day"
    Operator string      // "equals", "contains", "regex", "in", "between"
    Value    interface{} // Value(s) to compare against
}

type Action struct {
    Type        string // "redirect", "split", "tag"
    Destination string // Where to send traffic
    Weight      int    // For split testing
    Tags        []string // Additional tracking tags
}

func (r *CampaignRouter) GetDestination(org *Organization, event *Event) (string, *Campaign) {
    // 1. Check if campaign specified in URL
    if campaignID := extractCampaignID(event.URL); campaignID != "" {
        if campaign := r.getCampaign(org.ID, campaignID); campaign != nil {
            return r.evaluateCampaign(campaign, event), campaign
        }
    }
    
    // 2. Find matching campaign based on rules
    campaigns := r.getActiveCampaigns(org.ID)
    for _, campaign := range campaigns {
        if destination := r.evaluateCampaign(campaign, event); destination != "" {
            return destination, campaign
        }
    }
    
    // 3. Use organization's default campaign
    if defaultCampaign := r.getCampaign(org.ID, "default"); defaultCampaign != nil {
        return defaultCampaign.Destination, defaultCampaign
    }
    
    // 4. Fallback (configurable per organization)
    return org.DefaultDestination, nil
}
```

### 3. User Tracking

Generate consistent user IDs without invasive tracking:

```go
type UserTracker struct {
    redis *redis.Client
}

func (ut *UserTracker) GetOrCreateUserID(r *http.Request, org *Organization) string {
    // 1. Check for Warden JWT (authenticated user)
    if userID := extractWardenUserID(r); userID != "" {
        return fmt.Sprintf("wdn_%s", userID)
    }
    
    // 2. Check for existing tracking cookie
    if cookie, err := r.Cookie("trellis_uid"); err == nil {
        return cookie.Value
    }
    
    // 3. Generate snowflake ID based on non-invasive fingerprint
    fingerprint := generateFingerprint(r)
    userID := generateSnowflakeID(org.ID, fingerprint)
    
    // Note: Cookie setting happens after redirect for performance
    return userID
}

func generateFingerprint(r *http.Request) string {
    // Non-invasive fingerprinting using:
    // - User-Agent family (not full string)
    // - Accept-Language (normalized)
    // - IP subnet (not full IP)
    // This provides consistency without being invasive
    h := sha256.New()
    h.Write([]byte(normalizeUserAgent(r.UserAgent())))
    h.Write([]byte(normalizeLanguage(r.Header.Get("Accept-Language"))))
    h.Write([]byte(getIPSubnet(r.RemoteAddr)))
    return hex.EncodeToString(h.Sum(nil))[:16]
}
```

### 4. Data Storage

Events are stored in ClickHouse for analytics and PostgreSQL for relational data:

**ClickHouse Schema (Analytics)**
```sql
CREATE TABLE events (
    -- Core identifiers
    organization_id String,
    event_id UUID DEFAULT generateUUIDv4(),
    user_id String,
    session_id String,
    
    -- Event data
    timestamp DateTime64(3),
    event_type String,
    url String,
    
    -- Campaign attribution
    campaign_id Nullable(String),
    click_id String,
    
    -- Request details (for retroactive analysis)
    method String,
    headers String, -- JSON
    query_params String, -- JSON
    body String, -- JSON
    
    -- Enriched data
    ip String,
    user_agent String,
    referer String,
    country FixedString(2),
    region String,
    city String,
    device_type String,
    
    -- Tracking
    is_conversion UInt8 DEFAULT 0,
    conversion_value Decimal64(2) DEFAULT 0,
    
    -- Metadata
    ingested_at DateTime64(3) DEFAULT now64(3)
)
ENGINE = MergeTree()
PARTITION BY (toYYYYMM(timestamp), organization_id)
ORDER BY (organization_id, timestamp, event_id)
SAMPLE BY xxHash32(user_id)
TTL timestamp + INTERVAL 7 YEAR;
```

**PostgreSQL Schema (Campaigns & Rules)**
```sql
-- Campaigns table
CREATE TABLE campaigns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL,
    campaign_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    destination_url TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(organization_id, campaign_id)
);

-- Campaign rules table
CREATE TABLE campaign_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id UUID REFERENCES campaigns(id) ON DELETE CASCADE,
    rule_type VARCHAR(50) NOT NULL,
    conditions JSONB NOT NULL,
    action JSONB NOT NULL,
    priority INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Conversion tracking
CREATE TABLE conversions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL,
    click_id VARCHAR(255) NOT NULL,
    event_id UUID NOT NULL,
    conversion_value DECIMAL(10,2),
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(organization_id, click_id)
);
```

### 5. Async Processing Architecture

Events are processed asynchronously to maintain sub-50ms redirect latency:

```go
type AsyncProcessor struct {
    workers    int
    eventQueue chan *Event
    dlq        *DeadLetterQueue
    storage    *StorageEngine
    wg         sync.WaitGroup
}

func NewAsyncProcessor(workers int) *AsyncProcessor {
    ap := &AsyncProcessor{
        workers:    workers,
        eventQueue: make(chan *Event, workers * 100),
        dlq:        NewDeadLetterQueue(),
        storage:    NewStorageEngine(),
    }
    
    // Start worker pool
    for i := 0; i < workers; i++ {
        ap.wg.Add(1)
        go ap.worker(i)
    }
    
    return ap
}

func (ap *AsyncProcessor) worker(id int) {
    defer ap.wg.Done()
    
    for event := range ap.eventQueue {
        // Try to store event
        if err := ap.storage.StoreEvent(event); err != nil {
            // On failure, add to dead letter queue
            ap.dlq.Enqueue(event)
            log.WithError(err).WithField("worker", id).Error("Failed to store event")
        }
    }
}

func (ap *AsyncProcessor) ProcessEvent(event *Event) {
    select {
    case ap.eventQueue <- event:
        // Event queued successfully
    default:
        // Queue full, use dead letter queue
        ap.dlq.Enqueue(event)
        log.Warn("Event queue full, added to DLQ")
    }
}
```

### 6. Dead Letter Queue

Handle storage failures without losing data:

```go
type DeadLetterQueue struct {
    redis      *redis.Client
    maxRetries int
    batchSize  int
}


// Background worker processes dead letter queue
func (dlq *DeadLetterQueue) ProcessQueue() {
    for {
        events := dlq.dequeueBatch(dlq.batchSize)
        for _, event := range events {
            retries := dlq.getRetryCount(event.ID)
            if retries >= dlq.maxRetries {
                // Move to permanent failure storage
                dlq.moveToFailureLog(event)
                continue
            }
            
            // Retry storage
            if err := storage.StoreEvent(event); err != nil {
                dlq.incrementRetry(event.ID)
                dlq.requeueWithBackoff(event, retries)
            }
        }
        
        time.Sleep(10 * time.Second)
    }
}
```

### 6. API Endpoints

```
GET /in
- Main traffic ingestion with redirect
- Query params captured and stored
- Response: 302 redirect based on campaign rules

GET /in/{campaign_id}
- Campaign-specific ingestion
- Forces specific campaign attribution
- Response: 302 redirect to campaign destination

POST /postback
- Conversion tracking endpoint
- Updates conversion status for click_id
- Response: 200 OK

GET /pixel
- Tracking pixel endpoint
- Returns 1x1 transparent GIF
- Captures impression data

POST /campaigns
- Create/update campaign (requires org auth)
- Define routing rules and destinations
- Response: Campaign configuration

GET /health
- Health check endpoint
- No authentication required
- Response: Service status
```

## Campaign Routing Examples

### Example 1: Direct Campaign Mapping
```json
{
  "campaign_id": "summer_2024",
  "name": "Summer Sale 2024",
  "destination": "https://shop.example.com/summer-sale",
  "rules": [
    {
      "type": "direct",
      "priority": 100,
      "action": {
        "type": "redirect",
        "destination": "https://shop.example.com/summer-sale",
        "append_params": true
      }
    }
  ]
}
```
**URL**: `https://track.example.com/in/summer_2024?click_id=abc123&source=facebook`
**Result**: Redirects to `https://shop.example.com/summer-sale?click_id=abc123&source=facebook`

### Example 2: Geographic Routing
```json
{
  "campaign_id": "global_launch",
  "name": "Global Product Launch",
  "rules": [
    {
      "type": "geo",
      "priority": 90,
      "conditions": [
        {"field": "country", "operator": "in", "value": ["US", "CA"]}
      ],
      "action": {
        "type": "redirect",
        "destination": "https://na.example.com/launch"
      }
    },
    {
      "type": "geo",
      "priority": 80,
      "conditions": [
        {"field": "country", "operator": "in", "value": ["GB", "DE", "FR"]}
      ],
      "action": {
        "type": "redirect",
        "destination": "https://eu.example.com/launch"
      }
    },
    {
      "type": "direct",
      "priority": 10,
      "action": {
        "type": "redirect",
        "destination": "https://global.example.com/launch"
      }
    }
  ]
}
```

### Example 3: User Recognition Routing
```json
{
  "campaign_id": "retention_campaign",
  "name": "Customer Retention",
  "rules": [
    {
      "type": "user",
      "priority": 100,
      "conditions": [
        {"field": "user_type", "operator": "equals", "value": "returning"}
      ],
      "action": {
        "type": "redirect",
        "destination": "https://shop.example.com/welcome-back",
        "tags": ["returning_user", "vip"]
      }
    },
    {
      "type": "user",
      "priority": 90,
      "conditions": [
        {"field": "user_type", "operator": "equals", "value": "new"}
      ],
      "action": {
        "type": "redirect",
        "destination": "https://shop.example.com/first-time-offer"
      }
    }
  ]
}
```

### Example 4: Device-Based Routing
```json
{
  "campaign_id": "mobile_app_download",
  "name": "Mobile App Campaign",
  "rules": [
    {
      "type": "device",
      "priority": 100,
      "conditions": [
        {"field": "device_type", "operator": "equals", "value": "mobile"},
        {"field": "os", "operator": "equals", "value": "ios"}
      ],
      "action": {
        "type": "redirect",
        "destination": "https://apps.apple.com/app/example"
      }
    },
    {
      "type": "device",
      "priority": 90,
      "conditions": [
        {"field": "device_type", "operator": "equals", "value": "mobile"},
        {"field": "os", "operator": "equals", "value": "android"}
      ],
      "action": {
        "type": "redirect",
        "destination": "https://play.google.com/store/apps/details?id=com.example"
      }
    },
    {
      "type": "device",
      "priority": 80,
      "conditions": [
        {"field": "device_type", "operator": "equals", "value": "desktop"}
      ],
      "action": {
        "type": "redirect",
        "destination": "https://example.com/download"
      }
    }
  ]
}
```

### Example 5: Time-Based Routing
```json
{
  "campaign_id": "flash_sale",
  "name": "Weekend Flash Sale",
  "rules": [
    {
      "type": "time",
      "priority": 100,
      "conditions": [
        {"field": "day_of_week", "operator": "in", "value": ["saturday", "sunday"]},
        {"field": "hour", "operator": "between", "value": [10, 20]}
      ],
      "action": {
        "type": "redirect",
        "destination": "https://shop.example.com/flash-sale-active"
      }
    },
    {
      "type": "direct",
      "priority": 10,
      "action": {
        "type": "redirect",
        "destination": "https://shop.example.com/flash-sale-coming-soon"
      }
    }
  ]
}
```

### Example 6: A/B Split Testing
```json
{
  "campaign_id": "landing_page_test",
  "name": "Landing Page A/B Test",
  "rules": [
    {
      "type": "custom",
      "priority": 100,
      "action": {
        "type": "split",
        "destinations": [
          {
            "url": "https://example.com/landing-a",
            "weight": 50,
            "tag": "variant_a"
          },
          {
            "url": "https://example.com/landing-b",
            "weight": 30,
            "tag": "variant_b"
          },
          {
            "url": "https://example.com/landing-c",
            "weight": 20,
            "tag": "variant_c"
          }
        ]
      }
    }
  ]
}
```

### Fallback Behavior
When no campaign matches, the system follows this hierarchy:
1. Check for campaign ID in URL path (`/in/{campaign_id}`)
2. Evaluate all active campaign rules in priority order
3. Use organization's default campaign if configured
4. Fall back to organization's default destination URL
5. Return 404 if no destination is configured

## Retroactive Campaign Management

Campaigns can be created, updated, or deleted retroactively:

```go
func (s *CampaignService) ApplyRetroactiveCampaign(ctx context.Context, campaign Campaign, timeRange TimeRange) error {
    // 1. Validate campaign belongs to organization
    if err := s.validateOrganization(ctx, campaign); err != nil {
        return err
    }
    
    // 2. Update historical data in ClickHouse
    query := `
        ALTER TABLE events 
        UPDATE campaign_id = ? 
        WHERE organization_id = ?
          AND timestamp BETWEEN ? AND ?
          AND campaign_id IS NULL
          AND (conditions matching campaign rules)
    `
    
    // 3. Execute retroactive attribution
    return s.clickhouse.Exec(ctx, query, campaign.ID, campaign.OrganizationID, timeRange.Start, timeRange.End)
}
```

## Performance Targets

- **Redirect latency**: <50ms p99 (critical path)
- **Data capture**: Async, non-blocking
- **Rule evaluation**: <5ms for complex rules
- **Throughput**: 100K+ requests/second per node
- **Storage reliability**: 99.99% with dead letter queue

## Security & Privacy

- **No PII collection**: Only collect necessary attribution data
- **User privacy**: Non-invasive fingerprinting, no detailed tracking
- **Data isolation**: Complete organization separation
- **GDPR compliant**: No personal data that requires deletion

The ingress service balances high-performance traffic routing with comprehensive data capture, enabling both real-time campaign optimization and retroactive attribution analysis.