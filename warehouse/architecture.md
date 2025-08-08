# Trellis Warehouse Architecture

## System Overview

Trellis Warehouse is designed as a high-performance, organization-aware analytics API that provides real-time and historical insights into traffic attribution data with powerful retroactive analysis capabilities.

```
┌─────────────────────────────────────────────────────────────────┐
│                      WAREHOUSE ARCHITECTURE                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐  │
│  │   API    │───▶│   Query  │───▶│   Cache  │───▶│   Data   │  │
│  │Gateway   │    │ Builder  │    │  Layer   │    │Transform │  │
│  │          │    │          │    │          │    │          │  │
│  └──────────┘    └──────────┘    └──────────┘    └──────────┘  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│                    AUTHENTICATION LAYER                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────┐         ┌─────────────────┐                  │
│  │   Warden     │◀────────│  Organization   │                  │
│  │   gRPC API   │         │   Context       │                  │
│  │              │         │  Extraction     │                  │
│  └──────────────┘         └─────────────────┘                  │
└─────────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│                      DATA SOURCES                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────┐ │
│  │   ClickHouse    │    │    BigQuery     │    │    Redis    │ │
│  │   (Primary)     │    │  (Analytics)    │    │   (Cache)   │ │
│  │                 │    │                 │    │             │ │
│  └─────────────────┘    └─────────────────┘    └─────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

## Component Architecture

### 1. API Gateway Layer

**Authentication & Authorization**
```go
type WarehouseHandler struct {
    auth        *auth.Middleware
    queryEngine *QueryEngine
    cache       *CacheManager
    metrics     *MetricsCollector
}

func (h *WarehouseHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // 1. Extract organization context
    orgCtx, err := h.auth.ExtractOrganization(r)
    if err != nil {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    
    // 2. Parse and validate query parameters
    query, err := h.parseQuery(r, orgCtx)
    if err != nil {
        http.Error(w, "Bad Request", http.StatusBadRequest)
        return
    }
    
    // 3. Execute organization-scoped query
    result, err := h.queryEngine.Execute(query)
    if err != nil {
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }
    
    // 4. Return results
    h.writeJSON(w, result)
}
```

### 2. Query Engine

**Organization-Aware Query Builder**
```go
type QueryEngine struct {
    clickhouse clickhouse.Conn
    bigquery   *bigquery.Client
    redis      *redis.Client
    
    // Query optimization
    queryCache  map[string]*PreparedQuery
    planCache   *ristretto.Cache
}

type Query struct {
    OrganizationID string
    TimeRange      TimeRange
    Metrics        []string
    Dimensions     []string
    Filters        []Filter
    Sorting        []Sort
    Limit          int
    Offset         int
}

func (qe *QueryEngine) Execute(q *Query) (*QueryResult, error) {
    // 1. Build organization-scoped query
    sql := qe.buildSQL(q)
    
    // 2. Check cache first
    cacheKey := qe.buildCacheKey(q)
    if cached := qe.redis.Get(cacheKey); cached != nil {
        return cached.(*QueryResult), nil
    }
    
    // 3. Execute query with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    result, err := qe.executeWithRetry(ctx, sql, q.OrganizationID)
    if err != nil {
        return nil, err
    }
    
    // 4. Cache result
    qe.redis.Set(cacheKey, result, 5*time.Minute)
    
    return result, nil
}
```

### 3. Data Sources Integration

**ClickHouse Primary Storage**
```sql
-- Organization-partitioned schema for fast analytics
CREATE TABLE analytics_events (
    -- Core dimensions
    organization_id String,
    event_date Date,
    event_time DateTime64(3),
    event_id String,
    
    -- Traffic attribution
    click_id String,
    campaign_id Nullable(String),
    source Nullable(String),
    medium Nullable(String),
    term Nullable(String),
    content Nullable(String),
    
    -- Engagement metrics
    session_id Nullable(String),
    page_views UInt32 DEFAULT 0,
    session_duration UInt32 DEFAULT 0,
    bounce_rate Float32 DEFAULT 0,
    
    -- Conversion metrics
    conversion_event Nullable(String),
    conversion_value Decimal64(4) DEFAULT 0,
    conversion_time Nullable(DateTime64(3)),
    
    -- Device & location
    device_type Nullable(String),
    browser Nullable(String),
    os Nullable(String),
    country FixedString(2) DEFAULT '',
    region Nullable(String),
    city Nullable(String),
    
    -- Quality metrics
    fraud_score Float32 DEFAULT 0,
    quality_score Float32 DEFAULT 100,
    
    -- Enrichment status
    enriched UInt8 DEFAULT 0,
    enriched_at Nullable(DateTime64(3))
)
ENGINE = MergeTree()
PARTITION BY (toYYYYMM(event_date), organization_id)
ORDER BY (organization_id, event_date, campaign_id, event_time)
SAMPLE BY xxHash32(click_id)
```

**BigQuery Analytics Views**
```sql
-- Materialized views for complex analytics
CREATE MATERIALIZED VIEW campaign_performance AS
SELECT 
    organization_id,
    campaign_id,
    source,
    medium,
    DATE(event_time) as date,
    
    -- Traffic metrics
    COUNT(*) as clicks,
    COUNT(DISTINCT session_id) as sessions,
    COUNT(DISTINCT click_id) as unique_clicks,
    
    -- Engagement metrics  
    AVG(session_duration) as avg_session_duration,
    AVG(page_views) as avg_page_views,
    AVG(bounce_rate) as bounce_rate,
    
    -- Conversion metrics
    COUNTIF(conversion_event IS NOT NULL) as conversions,
    SUM(conversion_value) as revenue,
    AVG(conversion_value) as avg_order_value,
    
    -- Quality metrics
    AVG(fraud_score) as avg_fraud_score,
    AVG(quality_score) as avg_quality_score
    
FROM analytics_events
WHERE enriched = 1
GROUP BY 1,2,3,4,5
```

### 4. Caching Strategy

**Multi-Level Cache Architecture**
```go
type CacheManager struct {
    l1Cache *ristretto.Cache  // In-memory, 100MB
    l2Cache *redis.Client     // Redis, 1GB
    l3Cache *bigquery.Client  // Pre-computed views
}

type CacheConfig struct {
    // Different TTLs for different data types
    RealtimeData    time.Duration // 1 minute
    HourlyMetrics   time.Duration // 15 minutes
    DailyMetrics    time.Duration // 4 hours
    WeeklyMetrics   time.Duration // 24 hours
    MonthlyMetrics  time.Duration // 7 days
}

func (cm *CacheManager) Get(query *Query) (*QueryResult, bool) {
    // 1. Check L1 (in-memory) first
    if result := cm.l1Cache.Get(query.CacheKey()); result != nil {
        return result.(*QueryResult), true
    }
    
    // 2. Check L2 (Redis) 
    if result := cm.l2Cache.Get(query.CacheKey()); result != nil {
        // Populate L1 for future requests
        cm.l1Cache.Set(query.CacheKey(), result, 1)
        return result.(*QueryResult), true
    }
    
    // 3. Check L3 (pre-computed views)
    if cm.canUsePrecomputed(query) {
        result := cm.getFromBigQuery(query)
        if result != nil {
            // Populate both L1 and L2
            cm.l1Cache.Set(query.CacheKey(), result, 1)
            cm.l2Cache.Set(query.CacheKey(), result, cm.getTTL(query))
            return result, true
        }
    }
    
    return nil, false
}
```

### 5. API Endpoints Design

**Analytics Endpoints**
```go
// Traffic Overview
GET /api/v1/analytics/traffic?start_date=2024-01-01&end_date=2024-01-31&group_by=day

// Campaign Performance  
GET /api/v1/analytics/campaigns?campaign_id=summer_2024&metrics=clicks,conversions,revenue

// Attribution Analysis
GET /api/v1/analytics/attribution?click_id=abc123&include_journey=true

// Funnel Analysis
GET /api/v1/analytics/funnel?steps=click,view,signup,purchase&start_date=2024-01-01

// Cohort Analysis
GET /api/v1/analytics/cohorts?cohort_type=weekly&metric=retention&periods=12

// Real-time Dashboard
GET /api/v1/analytics/realtime?metrics=clicks,conversions&last=1h

// Custom Reports
POST /api/v1/analytics/query
{
  "dimensions": ["source", "campaign_id", "country"],
  "metrics": ["clicks", "conversions", "revenue"],
  "filters": [
    {"field": "source", "operator": "in", "values": ["google", "facebook"]},
    {"field": "conversion_value", "operator": "gt", "value": 10}
  ],
  "time_range": {"start": "2024-01-01", "end": "2024-01-31"},
  "group_by": "day",
  "order_by": [{"field": "revenue", "direction": "desc"}],
  "limit": 1000
}
```

### 6. Data Transformation Pipeline

**Real-time Enrichment**
```go
type EnrichmentPipeline struct {
    geoIP        *geoip.Database
    deviceParser *uaparser.Parser
    fraudEngine  *fraud.Detector
    
    workers      int
    queue        chan *Event
    done         chan bool
}

func (ep *EnrichmentPipeline) ProcessEvent(event *Event) *EnrichedEvent {
    enriched := &EnrichedEvent{
        Event: *event,
        EnrichedAt: time.Now(),
    }
    
    // 1. Geographic enrichment
    if location := ep.geoIP.Lookup(event.IP); location != nil {
        enriched.Country = location.Country
        enriched.Region = location.Region
        enriched.City = location.City
        enriched.Timezone = location.Timezone
    }
    
    // 2. Device enrichment
    if ua := ep.deviceParser.Parse(event.UserAgent); ua != nil {
        enriched.DeviceType = ua.DeviceType
        enriched.Browser = ua.Browser
        enriched.OS = ua.OS
        enriched.IsMobile = ua.IsMobile
    }
    
    // 3. Fraud detection
    fraudScore := ep.fraudEngine.Score(event)
    enriched.FraudScore = fraudScore
    enriched.FraudFlags = ep.fraudEngine.GetFlags(event, fraudScore)
    
    // 4. Quality scoring
    enriched.QualityScore = ep.calculateQualityScore(enriched)
    
    return enriched
}
```

### 7. Pattern Discovery Engine

**Retroactive Campaign Creation**
```go
type PatternDiscovery struct {
    clickhouse clickhouse.Conn
    mlEngine   *ml.Engine
    
    // Pattern detection algorithms
    clusteringAlgo *clustering.KMeans
    anomalyDetector *anomaly.IsolationForest
}

type DiscoveredPattern struct {
    OrganizationID    string
    PatternID         string
    Confidence        float64
    SampleSize        int
    
    // Pattern characteristics
    TimeRange         TimeRange
    TrafficSources    []string
    GeoRegions        []string
    DeviceTypes       []string
    
    // Suggested campaign
    SuggestedName     string
    EstimatedValue    float64
    PotentialReach    int
}

func (pd *PatternDiscovery) DiscoverPatterns(orgID string, timeRange TimeRange) ([]*DiscoveredPattern, error) {
    // 1. Extract unattributed traffic
    query := `
        SELECT 
            source, medium, country, device_type,
            toStartOfHour(event_time) as hour,
            count(*) as traffic,
            avg(conversion_value) as avg_value
        FROM analytics_events 
        WHERE organization_id = ? 
            AND campaign_id IS NULL
            AND event_time BETWEEN ? AND ?
        GROUP BY source, medium, country, device_type, hour
        HAVING traffic > 10
        ORDER BY traffic DESC
    `
    
    // 2. Apply clustering to find patterns
    trafficData := pd.executeQuery(query, orgID, timeRange.Start, timeRange.End)
    clusters := pd.clusteringAlgo.Fit(trafficData)
    
    // 3. Analyze each cluster for campaign potential
    var patterns []*DiscoveredPattern
    for _, cluster := range clusters {
        if pattern := pd.analyzeCluster(cluster, orgID); pattern != nil {
            patterns = append(patterns, pattern)
        }
    }
    
    // 4. Rank by potential value
    sort.Slice(patterns, func(i, j int) bool {
        return patterns[i].EstimatedValue > patterns[j].EstimatedValue
    })
    
    return patterns, nil
}
```

## Performance Optimizations

### Query Performance
- **Pre-aggregated tables** for common metrics (hourly/daily rollups)
- **Materialized views** for complex calculations
- **Columnar compression** in ClickHouse for 10:1 storage reduction
- **Query plan caching** to avoid re-parsing common queries

### Memory Management
- **Connection pooling**: 25 ClickHouse, 50 Redis connections
- **Result streaming**: Large result sets streamed to avoid OOM
- **LRU caching**: Automatic eviction of old cache entries
- **Object pooling**: Reuse query objects and result buffers

### Network Optimization  
- **Compression**: Gzip response compression
- **CDN integration**: Cache static responses at edge
- **HTTP/2**: Multiplexed connections for dashboard queries
- **WebSocket**: Real-time metric updates for dashboards

## Scalability Design

### Horizontal Scaling
- **Stateless API servers**: Scale based on CPU/memory
- **Read replicas**: Separate ClickHouse nodes for analytics
- **Cache distribution**: Redis Cluster for high availability  
- **Load balancing**: Geographic distribution of API servers

### Data Partitioning
- **Time-based partitioning**: Monthly partitions in ClickHouse
- **Organization partitioning**: Complete data isolation
- **Hot/Cold storage**: Recent data in SSD, historical in HDD
- **Auto-archival**: Move old data to cheap storage after 2 years

### Performance Targets
- **Query latency**: <2s for 1B row aggregations
- **Dashboard load**: <5s for complex dashboards
- **Real-time updates**: <10s latency from ingestion
- **Concurrent users**: 1000+ per API server
- **Data freshness**: <1 minute for real-time metrics

This architecture enables Trellis Warehouse to provide powerful analytics capabilities while maintaining organization isolation and sub-second query performance on massive datasets.