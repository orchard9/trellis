# Trellis Practical Examples

This document provides end-to-end examples of common Trellis workflows, from traffic ingestion to analytics and campaign management.

## Example 1: Complete Traffic Flow

### Step 1: User Clicks Marketing Link
A user clicks on a Facebook ad with this URL:
```
https://track.acmecorp.com/in/summer_sale?click_id=fb_123abc&source=facebook&medium=cpc&utm_campaign=summer2024
```

### Step 2: Ingress Service Processing
```go
// 1. Request arrives at ingress service
// 2. Extract API key from Warden auth
orgCtx := {
    OrganizationID: "org_acmecorp",
    Permissions: ["traffic:write", "campaigns:read"]
}

// 3. Generate user tracking ID
userID := "usr_1234567890abcdef" // Snowflake ID based on fingerprint

// 4. Capture event data
event := Event{
    OrganizationID: "org_acmecorp",
    EventID: "evt_9876543210fedcba",
    UserID: "usr_1234567890abcdef",
    ClickID: "fb_123abc",
    Timestamp: "2024-06-15T10:30:45.123Z",
    URL: "/in/summer_sale",
    QueryParams: {
        "click_id": "fb_123abc",
        "source": "facebook",
        "medium": "cpc",
        "utm_campaign": "summer2024"
    },
    Headers: {
        "User-Agent": "Mozilla/5.0...",
        "Accept-Language": "en-US,en;q=0.9",
        "Referer": "https://www.facebook.com/"
    },
    IP: "198.51.100.42",
    // Enriched data
    Country: "US",
    Region: "CA",
    City: "San Francisco",
    DeviceType: "mobile",
    OS: "iOS",
    Browser: "Safari"
}

// 5. Determine redirect based on campaign rules
campaign := "summer_sale"
destination := "https://shop.acmecorp.com/summer-sale?click_id=fb_123abc&source=facebook"

// 6. Process asynchronously (non-blocking)
asyncProcessor.ProcessEvent(event) // Returns immediately

// 7. Redirect user in <50ms
http.Redirect(w, r, destination, http.StatusFound)
```

### Step 3: Async Storage
```go
// Background worker processes event
func (w *Worker) processEvent(event *Event) {
    // Store in ClickHouse
    err := clickhouse.Insert(`
        INSERT INTO events (
            organization_id, event_id, user_id, click_id,
            timestamp, campaign_id, source, medium,
            country, device_type, ip, headers, query_params
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `, event.OrganizationID, event.EventID, event.UserID, event.ClickID,
       event.Timestamp, "summer_sale", "facebook", "cpc",
       event.Country, event.DeviceType, event.IP, 
       event.Headers, event.QueryParams)
    
    if err != nil {
        // Add to dead letter queue
        dlq.Enqueue(event)
    }
}
```

### Step 4: User Converts
User makes a purchase and the merchant calls the postback endpoint:
```bash
curl -X POST \
  -H "Authorization: Bearer wdn_sk_live_acmecorp_key" \
  -H "Content-Type: application/json" \
  https://track.acmecorp.com/postback \
  -d '{
    "click_id": "fb_123abc",
    "event": "purchase",
    "value": 99.99,
    "currency": "USD",
    "order_id": "ORD-456789"
  }'
```

### Step 5: Conversion Storage
```sql
-- Update ClickHouse with conversion
UPDATE events
SET 
    conversion_event = 'purchase',
    conversion_value = 99.99,
    conversion_time = now()
WHERE 
    organization_id = 'org_acmecorp'
    AND click_id = 'fb_123abc';

-- Also store in PostgreSQL for quick lookups
INSERT INTO conversions (
    organization_id, click_id, event_id, 
    conversion_value, metadata, created_at
) VALUES (
    'org_acmecorp', 'fb_123abc', 'evt_conv_123',
    99.99, '{"order_id": "ORD-456789"}', NOW()
);
```

## Example 2: Retroactive Campaign Creation

### Scenario
Acme Corp discovers they've been receiving unattributed traffic from TikTok for the past 3 months. They want to create a campaign retroactively to understand this traffic.

### Step 1: Discover Pattern
```bash
# Use warehouse API to discover patterns
curl -X POST \
  -H "Authorization: Bearer wdn_sk_live_acmecorp_key" \
  -H "Content-Type: application/json" \
  https://warehouse.trellis.com/api/v1/patterns/discover \
  -d '{
    "time_range": {
      "start": "2024-03-01",
      "end": "2024-06-01"
    },
    "min_traffic": 100,
    "min_confidence": 0.8
  }'
```

**Response:**
```json
{
  "patterns": [
    {
      "pattern_id": "pat_tiktok_organic",
      "confidence": 0.92,
      "sample_size": 15420,
      "characteristics": {
        "source": ["tiktok.com", "vm.tiktok.com"],
        "device_types": ["mobile"],
        "countries": ["US", "CA", "UK"],
        "time_patterns": {
          "peak_hours": [18, 19, 20, 21],
          "peak_days": ["thursday", "friday", "saturday"]
        }
      },
      "suggested_campaign": {
        "name": "TikTok Organic Discovery",
        "estimated_value": 125000,
        "estimated_conversions": 450,
        "confidence": 0.92
      }
    }
  ]
}
```

### Step 2: Create Retroactive Campaign
```bash
curl -X POST \
  -H "Authorization: Bearer wdn_sk_live_acmecorp_key" \
  -H "Content-Type: application/json" \
  https://ingress.trellis.com/api/v1/campaigns \
  -d '{
    "campaign_id": "tiktok_organic_q2_2024",
    "name": "TikTok Organic Q2 2024",
    "status": "active",
    "retroactive": true,
    "time_range": {
      "start": "2024-03-01",
      "end": "2024-06-01"
    },
    "rules": [
      {
        "type": "custom",
        "priority": 100,
        "conditions": [
          {
            "field": "referer",
            "operator": "contains",
            "value": "tiktok.com"
          }
        ],
        "action": {
          "type": "redirect",
          "destination": "https://shop.acmecorp.com/tiktok-landing",
          "tags": ["tiktok", "organic", "social"]
        }
      }
    ]
  }'
```

### Step 3: Apply Attribution
```sql
-- ClickHouse automatically updates historical data
ALTER TABLE events
UPDATE 
    campaign_id = 'tiktok_organic_q2_2024',
    tags = ['tiktok', 'organic', 'social']
WHERE 
    organization_id = 'org_acmecorp'
    AND timestamp BETWEEN '2024-03-01' AND '2024-06-01'
    AND campaign_id IS NULL
    AND (
        referer LIKE '%tiktok.com%' 
        OR referer LIKE '%vm.tiktok.com%'
    );

-- Result: 15,420 events attributed
```

### Step 4: Analyze Performance
```bash
curl -H "Authorization: Bearer wdn_sk_live_acmecorp_key" \
  "https://warehouse.trellis.com/api/v1/analytics/campaigns?campaign_id=tiktok_organic_q2_2024&metrics=clicks,conversions,revenue"
```

**Response:**
```json
{
  "campaign_id": "tiktok_organic_q2_2024",
  "period": "2024-03-01 to 2024-06-01",
  "metrics": {
    "clicks": 15420,
    "unique_users": 12350,
    "conversions": 468,
    "conversion_rate": 3.04,
    "revenue": 127840.50,
    "average_order_value": 273.16,
    "roas": 0,  // Organic traffic, no ad spend
    "top_converting_hours": [19, 20, 21],
    "top_converting_days": ["friday", "saturday"],
    "device_breakdown": {
      "mobile": 14200,
      "tablet": 1100,
      "desktop": 120
    }
  }
}
```

## Example 3: Multi-Touch Attribution Journey

### Scenario
Track a customer's complete journey across multiple touchpoints.

### Step 1: Customer Journey Data
```sql
-- Query in ClickHouse for customer journey
SELECT 
    event_time,
    source,
    medium,
    campaign_id,
    event_type,
    conversion_value
FROM events
WHERE 
    organization_id = 'org_acmecorp'
    AND user_id = 'usr_customer123'
ORDER BY event_time;
```

**Results:**
```
2024-06-01 10:15:00 | facebook  | cpc     | summer_awareness  | click | 0
2024-06-03 14:22:00 | google    | organic | null             | click | 0  
2024-06-05 19:45:00 | email     | newsletter | june_promo    | click | 0
2024-06-05 19:52:00 | direct    | null    | null             | click | 0
2024-06-05 19:55:00 | direct    | null    | null             | purchase | 149.99
```

### Step 2: Attribution Analysis
```bash
curl -H "Authorization: Bearer wdn_sk_live_acmecorp_key" \
  "https://warehouse.trellis.com/api/v1/analytics/attribution?user_id=usr_customer123&model=time_decay"
```

**Response:**
```json
{
  "user_id": "usr_customer123",
  "total_value": 149.99,
  "attribution_model": "time_decay",
  "touchpoints": [
    {
      "timestamp": "2024-06-01T10:15:00Z",
      "source": "facebook",
      "campaign": "summer_awareness",
      "days_before_conversion": 4.4,
      "attributed_value": 22.50,
      "attribution_percent": 15
    },
    {
      "timestamp": "2024-06-03T14:22:00Z",
      "source": "google",
      "campaign": null,
      "days_before_conversion": 2.2,
      "attributed_value": 37.50,
      "attribution_percent": 25
    },
    {
      "timestamp": "2024-06-05T19:45:00Z",
      "source": "email",
      "campaign": "june_promo",
      "days_before_conversion": 0.07,
      "attributed_value": 52.50,
      "attribution_percent": 35
    },
    {
      "timestamp": "2024-06-05T19:52:00Z",
      "source": "direct",
      "campaign": null,
      "days_before_conversion": 0.05,
      "attributed_value": 37.49,
      "attribution_percent": 25
    }
  ],
  "insights": [
    "Email was the final major touchpoint before conversion",
    "Customer journey spanned 4.4 days with 4 touchpoints",
    "Facebook initiated awareness, email drove conversion"
  ]
}
```

## Example 4: Real-Time Dashboard Updates

### Frontend Polling Implementation
```typescript
// React component using periodic polling
const CampaignDashboard: React.FC = () => {
  const { organization } = useAuth();
  const [metrics, setMetrics] = useState<RealtimeMetrics | null>(null);
  
  useEffect(() => {
    // Initial fetch
    fetchMetrics();
    
    // Poll every 30 seconds
    const interval = setInterval(fetchMetrics, 30000);
    
    return () => clearInterval(interval);
  }, [organization.id]);
  
  const fetchMetrics = async () => {
    const response = await fetch(
      `/api/v1/analytics/realtime?org_id=${organization.id}`,
      {
        headers: {
          'Authorization': `Bearer ${getAuthToken()}`
        }
      }
    );
    
    const data = await response.json();
    setMetrics(data);
  };
  
  return (
    <Dashboard>
      <MetricCard 
        title="Live Clicks (Last Hour)"
        value={metrics?.clicks_last_hour || 0}
        change={metrics?.clicks_change || 0}
      />
      <MetricCard 
        title="Conversions Today"
        value={metrics?.conversions_today || 0}
        revenue={metrics?.revenue_today || 0}
      />
      <TrafficChart data={metrics?.traffic_timeline || []} />
    </Dashboard>
  );
};
```

### Backend Real-Time Query
```go
func (h *Handler) GetRealtimeMetrics(w http.ResponseWriter, r *http.Request) {
    orgID := r.Context().Value("org_id").(string)
    
    // Query with aggressive caching (1 minute TTL)
    query := `
        SELECT 
            toStartOfMinute(event_time) as minute,
            count(*) as clicks,
            countIf(conversion_value > 0) as conversions,
            sum(conversion_value) as revenue
        FROM events
        WHERE 
            organization_id = ?
            AND event_time >= now() - INTERVAL 1 HOUR
        GROUP BY minute
        ORDER BY minute DESC
        LIMIT 60
    `
    
    results := h.queryWithCache(query, orgID, 60*time.Second)
    
    // Calculate aggregates
    response := RealtimeMetrics{
        ClicksLastHour: sumClicks(results),
        ConversionsToday: h.getConversionsToday(orgID),
        RevenueToday: h.getRevenueToday(orgID),
        TrafficTimeline: results,
        LastUpdated: time.Now(),
    }
    
    json.NewEncoder(w).Encode(response)
}
```

## Example 5: Fraud Detection Alert

### Suspicious Pattern Detection
```sql
-- Real-time fraud detection query
SELECT 
    ip,
    country,
    COUNT(*) as click_count,
    COUNT(DISTINCT user_agent) as ua_count,
    COUNT(DISTINCT click_id) as unique_clicks,
    AVG(time_between_clicks) as avg_click_interval
FROM (
    SELECT 
        ip,
        country,
        user_agent,
        click_id,
        event_time,
        event_time - LAG(event_time) OVER (PARTITION BY ip ORDER BY event_time) as time_between_clicks
    FROM events
    WHERE 
        organization_id = 'org_acmecorp'
        AND event_time >= now() - INTERVAL 1 HOUR
)
GROUP BY ip, country
HAVING 
    click_count > 100 
    OR (click_count > 20 AND ua_count = 1)
    OR avg_click_interval < 1
ORDER BY click_count DESC;
```

### Automated Response
```go
func (f *FraudDetector) HandleSuspiciousTraffic(ip string, pattern FraudPattern) {
    // 1. Log the detection
    log.WithFields(log.Fields{
        "ip": ip,
        "pattern": pattern.Type,
        "confidence": pattern.Confidence,
        "org_id": pattern.OrganizationID,
    }).Warn("Suspicious traffic detected")
    
    // 2. Update fraud score in real-time
    f.redis.Set(
        fmt.Sprintf("fraud_score:%s:%s", pattern.OrganizationID, ip),
        pattern.Confidence,
        24*time.Hour,
    )
    
    // 3. Send alert if high confidence
    if pattern.Confidence > 0.8 {
        f.alertManager.Send(Alert{
            Type: "fraud_detection",
            Severity: "high",
            OrganizationID: pattern.OrganizationID,
            Message: fmt.Sprintf("High fraud probability from IP %s", ip),
            Data: pattern,
        })
    }
    
    // 4. Auto-block if critical
    if pattern.Confidence > 0.95 {
        f.blockIP(pattern.OrganizationID, ip, 24*time.Hour)
    }
}
```

These examples demonstrate the complete Trellis workflow from traffic ingestion through analytics, showcasing the platform's real-time capabilities and retroactive attribution features.