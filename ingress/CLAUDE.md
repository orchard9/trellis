# CLAUDE.md - Trellis Ingress Service

Written by world-class traffic ingestion specialists with expertise in high-performance Go services, organization-aware authentication, and real-time data processing.

project: trellis-ingress
repo: github.com/orchard9/trellis/ingress  
description: Organization-aware traffic ingestion service with sub-100ms redirect latency

## Where is this deployed
Dev: https://ingress-dev.trellis.orchard9.com/
Production: https://ingress.trellis.orchard9.com/

## Core Philosophy

**"Capture Everything, Redirect Immediately"** - Every byte of traffic data is valuable. Organizations are completely isolated. Never lose data, never slow down users.

## How to develop locally

1. Ensure prerequisites: Go 1.21+, Docker, Make, ClickHouse, Redis, Warden service
2. Clone repository and navigate to ingress directory: `cd ingress/`
3. Run `cp .env.example .env` and configure environment variables
4. Set up Warden organization and API key:
   - Create organization in Warden
   - Create service account and generate API key
   - Update `WARDEN_SERVICE_API_KEY` in `.env`
5. Run `docker-compose up -d` from root to start ClickHouse, Redis
6. Run `go run cmd/api/main.go` to start the ingress server
7. Test with authentication: `curl -H "Authorization: Bearer wdn_your_key" http://localhost:8080/health`
8. Send test traffic: `curl -H "Authorization: Bearer wdn_your_key" "http://localhost:8080/in?click_id=test123"`

**Development Server Notes:**
- Port: 8080 (configurable via TRELLIS_PORT)
- Hot reload: Use `air` for development
- All endpoints require Warden authentication except /health and /ready

**IMPORTANT: Code Quality Requirements**
Before submitting any code changes, ensure:
1. **Build passes**: `go build ./...` must succeed without errors
2. **Tests pass**: `go test ./...` must succeed with organization isolation tests
3. **Linting passes**: `golangci-lint run` must pass with zero issues
4. **Performance**: Redirect latency must stay <100ms under load
5. **Security**: All organization data properly isolated

## How to run tests

1. Unit tests: `go test ./...`
2. Integration tests: `go test -tags integration ./...` (requires services)
3. Load tests: `go test -tags load ./...`
4. Organization isolation tests: `go test -tags security ./...`
5. Performance tests: `go test -bench=. ./...`

## How to ingest traffic

All traffic ingestion requires organization authentication:

1. Basic redirect: `curl -H "Authorization: Bearer wdn_your_key" "http://localhost:8080/in?click_id=test123&source=google"`
2. Campaign-specific: `curl -H "Authorization: Bearer wdn_your_key" "http://localhost:8080/in/camp_123?click_id=test456"`  
3. POST data: `curl -X POST -H "Authorization: Bearer wdn_your_key" -H "Content-Type: application/json" http://localhost:8080/in -d '{"custom":"data"}'`
4. Pixel tracking: `curl -H "Authorization: Bearer wdn_your_key" "http://localhost:8080/pixel.gif?event=view&click_id=test789"`
5. Postback: `curl -H "Authorization: Bearer wdn_your_key" "http://localhost:8080/postback?click_id=test123&status=converted&value=25.00"`

## How to manage campaigns

Organization-scoped campaign management:

1. Create campaign: 
```bash
curl -X POST -H "Authorization: Bearer wdn_your_key" \
  -H "Content-Type: application/json" \
  http://localhost:8080/api/v1/campaigns -d '{
  "campaign_id": "summer_2024",
  "name": "Summer Campaign",
  "status": "active",
  "rules": [
    {"field": "source", "operator": "equals", "values": ["facebook"], "priority": 10}
  ],
  "destination_url": "https://landing.example.com",
  "append_params": true
}'
```

2. Update campaign routing rules
3. Monitor campaign performance
4. Archive completed campaigns

## How to debug ingestion issues

### Common Issues and Solutions

**Issue: Authentication failures**
- Check: API key format (must start with `wdn_`)
- Debug: Verify Warden service is running and accessible
- Solution: Regenerate API key or check Warden connection

**Issue: High redirect latency (>100ms)**
- Check: Campaign rule complexity
- Debug: `curl localhost:8080/debug/pprof/profile`
- Solution: Optimize campaign rules or increase cache TTL

**Issue: Organization data mixing**
- Check: All queries include organization_id filter
- Debug: Review database queries in logs
- Solution: Add organization_id constraints to all queries

**Issue: Traffic loss during spikes**
- Check: Async worker queue depth and processing errors
- Debug: Check worker pool capacity and storage connectivity
- Solution: Scale async workers or implement circuit breaker

### Debug Commands
```bash
# Test ingestion with organization context
curl -H "Authorization: Bearer wdn_your_key" -v http://localhost:8080/in?test=1

# Monitor organization-scoped traffic
watch 'clickhouse-client --query "SELECT organization_id, count(*) FROM events WHERE event_time > now() - INTERVAL 1 MINUTE GROUP BY organization_id"'

# Profile ingestion performance  
go tool pprof http://localhost:8080/debug/pprof/profile

# Check organization authentication
grpcurl -plaintext -H "authorization: Bearer wdn_your_key" localhost:21382 warden.v1.AuthService.ValidateApiKey
```

## Performance Standards

### Latency Requirements
- **Redirect latency**: <50ms p99, <100ms p99.9
- **Authentication**: <10ms p95
- **Campaign resolution**: <5ms p95
- **Deduplication check**: <2ms p95

### Throughput Targets
- **Per node**: 100K requests/second sustained
- **Organization isolation**: Zero performance penalty
- **Memory usage**: <1GB per 100K RPS
- **CPU usage**: <60% at peak load

### Organization Isolation Standards
- **Data separation**: 100% isolation between organizations
- **Query performance**: No cross-organization data leakage
- **Cache isolation**: Organization-scoped Redis keys
- **Metrics isolation**: All metrics tagged with organization_id

## Error Handling Standards

1. **NEVER DROP TRAFFIC DATA**
   - If async processing fails, use dead letter queue
   - If ClickHouse is down, queue events for later processing
   - If Redis is unavailable, allow traffic through without dedup
   - Better to have duplicate data than lost data

2. **Organization Isolation Errors**
   - If organization context missing, reject request immediately
   - Log all cross-organization access attempts as security events
   - Never fallback to default organization

3. **Performance Degradation**
   - If latency exceeds 100ms, enable circuit breaker
   - If memory usage exceeds 80%, enable backpressure
   - If error rate exceeds 1%, alert immediately

## Logging Standards

DEBUG = Request details, campaign rule evaluation, organization context extraction
INFO = Traffic accepted, successful redirects, campaign matches
WARN = High latency, approaching limits, authentication retries
ERROR = Failed redirects, organization isolation violations, dependency failures
CRITICAL = Data loss risk, security violations, service unavailable

All logs MUST include organization_id for proper isolation and debugging.

## Key Operational Commands

- `go run cmd/api/main.go` - start ingress server
- `go test ./...` - run all tests including organization isolation
- `go test -bench=. ./...` - run performance benchmarks
- `golangci-lint run` - code quality check
- `curl -H "Authorization: Bearer wdn_key" localhost:8080/health` - health check with auth
- `go tool pprof http://localhost:8080/debug/pprof/profile` - performance profiling

## Architecture Patterns

### Organization Context Pattern
```go
// Every handler must extract organization context
func (h *Handler) HandleTraffic(w http.ResponseWriter, r *http.Request) {
    orgCtx, ok := auth.GetOrganizationContext(r.Context())
    if !ok {
        http.Error(w, "Organization context required", http.StatusUnauthorized)
        return
    }
    
    // All operations scoped to organization
    event.OrganizationID = orgCtx.OrganizationID
    // ...
}
```

### Campaign Routing Pattern
```go
// Campaign lookups always organization-scoped
func (r *RoutingEngine) GetDestination(orgID string, event *Event) string {
    key := fmt.Sprintf("%s/%s", orgID, event.CampaignID)
    if campaign := r.getCampaign(key); campaign != nil {
        return r.buildURL(campaign, event.Params)
    }
    return r.getDefaultDestination(orgID)
}
```

### Database Query Pattern
```sql
-- ALL queries MUST include organization_id
SELECT * FROM events 
WHERE organization_id = ? 
  AND event_time > now() - INTERVAL 1 HOUR
ORDER BY event_time DESC
```

## Testing Patterns

### Organization Isolation Testing
```go
func TestOrganizationIsolation(t *testing.T) {
    org1Events := createEventsForOrg(t, "org1")
    org2Events := createEventsForOrg(t, "org2")
    
    // Verify org1 can only see org1 data
    org1Data := queryAsOrg(t, "org1") 
    assert.Contains(t, org1Data, org1Events)
    assert.NotContains(t, org1Data, org2Events)
    
    // Verify org2 can only see org2 data
    org2Data := queryAsOrg(t, "org2")
    assert.Contains(t, org2Data, org2Events)
    assert.NotContains(t, org2Data, org1Events)
}
```

### Performance Testing
```go
func BenchmarkTrafficIngestion(b *testing.B) {
    handler := setupTestHandler(b)
    
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            req := createTestRequest()
            w := httptest.NewRecorder()
            
            start := time.Now()
            handler.HandleTraffic(w, req)
            latency := time.Since(start)
            
            if latency > 100*time.Millisecond {
                b.Errorf("Latency too high: %v", latency)
            }
        }
    })
}
```

## Security Checklist

Before deploying any ingress code:

1. All endpoints require organization authentication
2. All database queries include organization_id filter  
3. All cache keys include organization prefix
4. All logs include organization_id
5. All metrics include organization label
6. No hardcoded organization references
7. No shared state between organizations
8. Authentication errors logged as security events
9. Rate limiting applied per organization
10. All PII handling compliant with data retention policies

## Helpful Tips

*Lessons learned from building high-performance ingress systems:*

1. **Always authenticate first**
   - Mistake: Processing data before authentication
   - Correct: Extract organization context immediately
   - Why: Prevents accidental data leakage

2. **Use organization-scoped keys everywhere**
   - Mistake: Generic cache keys
   - Correct: `org:{org_id}:campaign:{id}` format
   - Why: Eliminates cross-organization data access

3. **Pool everything**
   - Mistake: Creating new objects per request
   - Correct: sync.Pool for events, buffers, connections
   - Why: Reduces GC pressure at high RPS

4. **Fire-and-forget for non-critical paths**
   - Mistake: Blocking on storage operations in request path
   - Correct: Async processing with worker goroutines
   - Why: Keeps redirect latency low

5. **Circuit breakers for dependencies**
   - Mistake: Cascading failures when Redis is down
   - Correct: Degrade gracefully, allow traffic through
   - Why: Uptime is more important than perfect deduplication