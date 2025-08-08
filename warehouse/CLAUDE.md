# CLAUDE.md - Trellis Warehouse Service

Written by world-class data warehouse architects with expertise in high-performance analytics APIs, organization-aware query engines, and retroactive pattern discovery systems.

project: trellis-warehouse
repo: github.com/orchard9/trellis/warehouse  
description: Organization-aware analytics API with retroactive campaign attribution and sub-second query performance

## Where is this deployed
Dev: https://warehouse-dev.trellis.orchard9.com/
Production: https://warehouse.trellis.orchard9.com/

## Core Philosophy

**"Understand Everything, Retroactively"** - Complete traffic analytics with unlimited query flexibility. Organizations are completely isolated. Every query sub-2 seconds, every insight immediately actionable.

## How to develop locally

1. Ensure prerequisites: Go 1.21+, Docker, Make, ClickHouse, Redis, BigQuery emulator, Warden service
2. Clone repository and navigate to warehouse directory: `cd warehouse/`
3. Run `cp .env.example .env` and configure environment variables
4. Set up Warden organization and API key:
   - Create organization in Warden
   - Create service account with analytics permissions
   - Update `WARDEN_SERVICE_API_KEY` in `.env`
5. Run `docker-compose up -d` from root to start ClickHouse, Redis
6. Run `go run cmd/api/main.go` to start the warehouse API server
7. Test with authentication: `curl -H "Authorization: Bearer wdn_your_key" http://localhost:8090/health`
8. Query analytics: `curl -H "Authorization: Bearer wdn_your_key" "http://localhost:8090/api/v1/analytics/traffic?start_date=2024-01-01&end_date=2024-01-31"`

**Development Server Notes:**
- Port: 8090 (configurable via WAREHOUSE_PORT)  
- Hot reload: Use `air` for development
- All endpoints require Warden authentication except /health and /ready
- Query timeout: 30 seconds for complex analytics

**IMPORTANT: Code Quality Requirements**
Before submitting any code changes, ensure:
1. **Build passes**: `go build ./...` must succeed without errors
2. **Tests pass**: `go test ./...` must succeed with organization isolation tests
3. **Performance**: All queries must complete within timeout limits
4. **Query correctness**: All analytics queries properly organization-scoped
5. **Cache efficiency**: Cache hit ratio >80% for common queries

## How to run tests

1. Unit tests: `go test ./...`
2. Integration tests: `go test -tags integration ./...` (requires ClickHouse)
3. Performance tests: `go test -tags benchmark ./...`
4. Organization isolation tests: `go test -tags security ./...`
5. Query accuracy tests: `go test -tags analytics ./...`

## How to query analytics

All analytics queries require organization authentication:

### Basic Traffic Analytics
```bash
# Daily traffic overview
curl -H "Authorization: Bearer wdn_your_key" \
  "http://localhost:8090/api/v1/analytics/traffic?start_date=2024-01-01&end_date=2024-01-31&group_by=day"

# Campaign performance
curl -H "Authorization: Bearer wdn_your_key" \
  "http://localhost:8090/api/v1/analytics/campaigns?campaign_id=summer_2024&metrics=clicks,conversions,revenue"

# Real-time dashboard data
curl -H "Authorization: Bearer wdn_your_key" \
  "http://localhost:8090/api/v1/analytics/realtime?metrics=clicks,conversions&last=1h"
```

### Advanced Analytics
```bash
# Attribution analysis for specific click
curl -H "Authorization: Bearer wdn_your_key" \
  "http://localhost:8090/api/v1/analytics/attribution?click_id=abc123&include_journey=true"

# Cohort analysis  
curl -H "Authorization: Bearer wdn_your_key" \
  "http://localhost:8090/api/v1/analytics/cohorts?cohort_type=weekly&metric=retention&periods=12"

# Funnel analysis
curl -H "Authorization: Bearer wdn_your_key" \
  "http://localhost:8090/api/v1/analytics/funnel?steps=click,view,signup,purchase&start_date=2024-01-01"
```

### Custom Query Builder
```bash
# Complex custom analytics
curl -X POST -H "Authorization: Bearer wdn_your_key" \
  -H "Content-Type: application/json" \
  http://localhost:8090/api/v1/analytics/query -d '{
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
}'
```

## How to discover patterns

Organization-scoped pattern discovery for retroactive campaigns:

```bash
# Discover traffic patterns in unattributed data
curl -X POST -H "Authorization: Bearer wdn_your_key" \
  -H "Content-Type: application/json" \
  http://localhost:8090/api/v1/patterns/discover -d '{
  "time_range": {"start": "2024-01-01", "end": "2024-01-31"},
  "min_traffic": 100,
  "min_confidence": 0.8
}'

# Analyze specific pattern
curl -H "Authorization: Bearer wdn_your_key" \
  "http://localhost:8090/api/v1/patterns/analyze?pattern_id=pattern_123&include_forecast=true"

# Generate campaign suggestions
curl -H "Authorization: Bearer wdn_your_key" \
  "http://localhost:8090/api/v1/patterns/suggestions?pattern_id=pattern_123"
```

## How to debug query performance

### Query Performance Issues

**Issue: Query timeout (>30s)**
- Check: Query complexity and data volume
- Debug: `EXPLAIN` query plan in ClickHouse console
- Solution: Add query hints, optimize filters, use pre-aggregated tables

**Issue: High memory usage during queries**
- Check: Large result sets without pagination
- Debug: Monitor ClickHouse memory usage
- Solution: Add LIMIT clauses, stream large results

**Issue: Poor cache hit ratio (<50%)**
- Check: Cache key generation and TTL settings
- Debug: Redis cache statistics
- Solution: Optimize cache keys, adjust TTL based on query type

**Issue: Organization data leakage**
- Check: All queries include organization_id filter
- Debug: Review query logs for missing organization filters
- Solution: Add organization_id constraints to all queries

### Debug Commands
```bash
# Test analytics query with debug info
curl -H "Authorization: Bearer wdn_your_key" -v \
  "http://localhost:8090/api/v1/analytics/traffic?debug=true&start_date=2024-01-01&end_date=2024-01-31"

# Check query performance metrics
curl -H "Authorization: Bearer wdn_your_key" \
  "http://localhost:8090/api/v1/debug/query-stats?organization_id=your_org_id"

# Monitor cache performance
watch 'redis-cli -p 6379 INFO stats | grep keyspace'

# Profile query execution
go tool pprof http://localhost:8090/debug/pprof/profile

# Check ClickHouse query performance
clickhouse-client --query "SELECT query, query_duration_ms, read_rows, read_bytes FROM system.query_log WHERE type = 'QueryFinish' ORDER BY event_time DESC LIMIT 10"
```

## Performance Standards

### Query Performance Requirements
- **Simple queries**: <500ms p95 (traffic overview, single metric)
- **Complex queries**: <2s p95 (multi-dimensional analysis, large date ranges)
- **Real-time queries**: <100ms p95 (last hour metrics)
- **Custom queries**: <10s p95 (unlimited complexity with user timeout)

### Throughput Targets
- **Concurrent queries**: 1000+ simultaneous per server
- **Cache hit ratio**: >80% for common dashboard queries
- **Memory efficiency**: <2GB per 1000 concurrent queries
- **Query accuracy**: 100% organization isolation, zero data leakage

### Organization Isolation Standards
- **Data separation**: 100% isolation between organizations in all queries
- **Query performance**: No cross-organization performance impact
- **Cache isolation**: Organization-scoped Redis keys prevent data leakage
- **Audit trails**: All queries logged with organization context

## Query Architecture Patterns

### Organization-Scoped Query Pattern
```go
// ALL analytics queries MUST include organization filter
func (qe *QueryEngine) buildBaseQuery(orgID string) string {
    return fmt.Sprintf(`
        SELECT * FROM analytics_events 
        WHERE organization_id = '%s'
        AND event_time BETWEEN ? AND ?
    `, orgID)
}

// Never allow queries without organization context
func (h *Handler) ExecuteQuery(w http.ResponseWriter, r *http.Request) {
    orgCtx, ok := auth.GetOrganizationContext(r.Context())
    if !ok {
        http.Error(w, "Organization context required", http.StatusUnauthorized)
        return
    }
    
    // All queries scoped to organization
    query.OrganizationID = orgCtx.OrganizationID
    // ...
}
```

### Multi-Level Caching Pattern
```go
// Cache keys always include organization for isolation
func (cm *CacheManager) buildCacheKey(orgID, queryType string, params map[string]interface{}) string {
    hash := sha256.Sum256([]byte(fmt.Sprintf("%v", params)))
    return fmt.Sprintf("analytics:%s:%s:%x", orgID, queryType, hash[:8])
}

// Different TTLs based on data freshness requirements
func (cm *CacheManager) getTTL(queryType string) time.Duration {
    switch queryType {
    case "realtime":
        return 1 * time.Minute
    case "hourly":
        return 15 * time.Minute
    case "daily":
        return 4 * time.Hour
    case "historical":
        return 24 * time.Hour
    default:
        return 5 * time.Minute
    }
}
```

### Query Optimization Pattern
```sql
-- Use organization-aware partitioning for performance
SELECT 
    toStartOfDay(event_time) as date,
    source,
    count(*) as clicks,
    countIf(conversion_value > 0) as conversions,
    sum(conversion_value) as revenue
FROM analytics_events
WHERE organization_id = 'org_123'  -- CRITICAL: Always filter by org first
    AND event_time BETWEEN '2024-01-01' AND '2024-01-31'
    AND source IN ('google', 'facebook')  -- Additional filters after org filter
GROUP BY date, source
ORDER BY date DESC, revenue DESC
```

## Testing Patterns

### Organization Isolation Testing
```go
func TestAnalyticsOrganizationIsolation(t *testing.T) {
    // Create events for different organizations
    org1Events := createEventsForOrg(t, "org1", 100)
    org2Events := createEventsForOrg(t, "org2", 100)
    
    // Query as org1
    org1Result := queryAnalytics(t, "org1", AnalyticsQuery{
        TimeRange: last30Days(),
        Metrics: []string{"clicks", "conversions"},
    })
    
    // Verify org1 sees only org1 data
    assert.Equal(t, 100, org1Result.TotalClicks)
    assert.NotContains(t, org1Result.RawData, org2Events)
    
    // Query as org2
    org2Result := queryAnalytics(t, "org2", AnalyticsQuery{
        TimeRange: last30Days(),
        Metrics: []string{"clicks", "conversions"},
    })
    
    // Verify org2 sees only org2 data
    assert.Equal(t, 100, org2Result.TotalClicks) 
    assert.NotContains(t, org2Result.RawData, org1Events)
}
```

### Query Performance Testing
```go
func BenchmarkComplexAnalyticsQuery(b *testing.B) {
    setupTestData(b, 1000000) // 1M events
    
    query := AnalyticsQuery{
        Dimensions: []string{"source", "campaign_id", "country"},
        Metrics:    []string{"clicks", "conversions", "revenue"},
        TimeRange:  last90Days(),
        Filters: []Filter{
            {Field: "source", Operator: "in", Values: []string{"google", "facebook"}},
        },
        GroupBy: "day",
        OrderBy: []Sort{{Field: "revenue", Direction: "desc"}},
        Limit:   1000,
    }
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        start := time.Now()
        result := executeQuery(query)
        latency := time.Since(start)
        
        if latency > 2*time.Second {
            b.Errorf("Query too slow: %v", latency)
        }
        
        if len(result.Data) == 0 {
            b.Error("Query returned no results")
        }
    }
}
```

## Error Handling Standards

1. **NEVER LEAK ORGANIZATION DATA**
   - If organization context missing, reject immediately
   - Log all cross-organization access attempts as security violations
   - Never fall back to default organization or unfiltered queries

2. **Query Timeout Handling**
   - 30-second hard timeout on all queries
   - Progressive timeout warnings at 10s and 20s
   - Graceful degradation for complex queries

3. **Data Accuracy**
   - All analytics results must be mathematically accurate
   - Handle edge cases like division by zero gracefully
   - Validate all calculated metrics before returning

## Logging Standards

DEBUG = Query plans, cache operations, organization context extraction
INFO = Query execution, cache hits/misses, successful analytics requests  
WARN = Slow queries, low cache hit ratios, approaching memory limits
ERROR = Query failures, organization isolation violations, timeout errors
CRITICAL = Data integrity issues, security violations, service unavailable

All logs MUST include organization_id for proper isolation and debugging.

## Key Operational Commands

- `go run cmd/api/main.go` - start warehouse API server
- `go test ./...` - run all tests including organization isolation
- `go test -bench=. ./...` - run analytics performance benchmarks
- `golangci-lint run` - code quality check
- `curl -H "Authorization: Bearer wdn_key" localhost:8090/health` - health check
- `go tool pprof http://localhost:8090/debug/pprof/profile` - performance profiling

## Analytics Patterns

### Time-Series Analytics Pattern
```go
// Always use appropriate time granularity
func (qe *QueryEngine) buildTimeSeriesQuery(orgID string, granularity string) string {
    var timeFunc string
    switch granularity {
    case "hour":
        timeFunc = "toStartOfHour(event_time)"
    case "day":
        timeFunc = "toStartOfDay(event_time)"
    case "week":  
        timeFunc = "toStartOfWeek(event_time)"
    case "month":
        timeFunc = "toStartOfMonth(event_time)"
    default:
        timeFunc = "toStartOfDay(event_time)"
    }
    
    return fmt.Sprintf(`
        SELECT 
            %s as time_bucket,
            count(*) as clicks,
            countIf(conversion_value > 0) as conversions,
            sum(conversion_value) as revenue
        FROM analytics_events
        WHERE organization_id = '%s'
            AND event_time BETWEEN ? AND ?
        GROUP BY time_bucket
        ORDER BY time_bucket
    `, timeFunc, orgID)
}
```

### Multi-Touch Attribution Pattern
```sql
-- Attribution modeling with organization isolation
WITH customer_journeys AS (
    SELECT 
        organization_id,
        session_id,
        click_id,
        campaign_id,
        source,
        event_time,
        conversion_value,
        ROW_NUMBER() OVER (
            PARTITION BY organization_id, session_id 
            ORDER BY event_time
        ) as touch_sequence,
        COUNT(*) OVER (
            PARTITION BY organization_id, session_id
        ) as total_touches
    FROM analytics_events 
    WHERE organization_id = 'your_org_id'
        AND session_id IN (
            SELECT session_id 
            FROM analytics_events 
            WHERE organization_id = 'your_org_id'
                AND conversion_value > 0
        )
)
SELECT 
    source,
    touch_sequence,
    count(*) as touches,
    sum(conversion_value / total_touches) as attributed_revenue
FROM customer_journeys
GROUP BY source, touch_sequence
ORDER BY attributed_revenue DESC
```

## Security Checklist

Before deploying any warehouse code:

1. All queries include organization_id filter as first WHERE condition
2. All cache keys prefixed with organization ID  
3. All API responses exclude other organizations' data
4. All database connections use organization-scoped connection strings where possible
5. All logs include organization_id for audit trails
6. No direct SQL query execution without organization context
7. All aggregation functions properly handle organization boundaries
8. Rate limiting applied per organization
9. Query complexity limits enforced per organization tier
10. All analytics results validated for mathematical accuracy

## Helpful Tips

*Lessons learned from building high-performance analytics systems:*

1. **Always partition by organization first**
   - Mistake: Time-based partitioning only
   - Correct: (organization_id, time) compound partitioning
   - Why: Enables data isolation and query performance

2. **Pre-aggregate common metrics**
   - Mistake: Calculating everything from raw events
   - Correct: Hourly/daily rollup tables for common queries
   - Why: 100x performance improvement for dashboards

3. **Cache at multiple levels**
   - Mistake: Single Redis cache for everything
   - Correct: L1 (in-memory), L2 (Redis), L3 (pre-computed)
   - Why: Sub-second response times for complex queries

4. **Stream large result sets**
   - Mistake: Loading entire result sets into memory
   - Correct: Streaming responses with pagination
   - Why: Prevents OOM errors and improves user experience

5. **Organization-scope all operations**
   - Mistake: Adding organization filtering as an afterthought
   - Correct: Organization context built into every query builder
   - Why: Eliminates accidental data leakage