# CLAUDE.md

Written by world-class traffic attribution and data warehouse architecture specialists with expertise in real-time ingestion systems and retroactive analytics platforms

project: trellis
repo: github.com/orchard9/trellis
description: Universal traffic ingestion gateway with time-travel analytics - capture everything now, understand it later through powerful retroactive attribution

## Where is this deployed
Dev: https://dev.trellis.orchard9.com/
Production: https://trellis.orchard9.com/

## Core Philosophy

**"Store first, understand later"** - Every byte of traffic data is valuable. Campaigns are created AFTER traffic arrives. Pattern discovery happens retroactively. Never lose attribution data again.

## How to develop locally

1. Ensure prerequisites: Go 1.24+, Docker, Make, ClickHouse, Redis, Warden service
2. Clone repository and navigate to project directory
3. Run `cp .env.example .env` and configure environment variables (including Warden connection)
4. Set up Warden organization and API key:
   - Create organization in Warden: `grpcurl -plaintext -d '{"name": "Dev Org", "slug": "dev-org"}' localhost:21382 warden.v1.OrganizationService/CreateOrganization`
   - Create service account: Follow Warden documentation for API key generation
   - Update `WARDEN_SERVICE_API_KEY` in `.env`
5. Run `docker-compose up -d` to start ClickHouse, Redis, and supporting services
6. Run `make dev` to start the development server
7. Access API at http://localhost:8080 with proper authentication headers
8. Access ClickHouse UI at http://localhost:8123
9. Run `docker-compose down` to shutdown services

**Development Server Notes:**
- Port: 8080 (configurable via TRELLIS_PORT)
- Hot reload enabled with Air
- Use `Ctrl+C` to stop the server

**IMPORTANT: Code Quality Requirements**
Before submitting any code changes, ensure:
1. **Build passes**: `go build ./...` must succeed without errors
2. **Linting passes**: `make lint` must pass with zero issues (golangci-lint)
3. **Tests pass**: `make test` must succeed with 80%+ coverage
4. **Format check**: Code must be properly formatted with `gofmt`
5. **Performance benchmarks**: `make bench` must not regress

## How to run tests

1. Unit tests: `make test` or `make test-unit`
2. Integration tests: `make test-integration` (requires services running)
3. All tests: `make test-all`
4. Test coverage: `make test-coverage`
5. Coverage check: `make coverage-check`
   - Phase 1: 60% threshold (core ingestion)
   - Phase 2: 70% threshold (analytics layer)
   - Phase 3: 80% threshold (pattern discovery)
   - Phase 4: 90% threshold (production ready)
6. Benchmarks: `make bench`
7. Load testing: `make load-test`
8. Full CI pipeline: `make ci`

## How to deploy

1. Development environment: `make deploy-dev` (automated pipeline)
2. Production environment: `make deploy-prod` (with confirmation)
3. Check deployment health: `make deploy-health ENV=dev|prod`
4. View deployment logs: `kubectl logs -f deployment/trellis-api`
5. Rollback if needed: `make deploy-rollback ENV=dev|prod`
6. Scale deployment: `kubectl scale deployment/trellis-api --replicas=10`
7. Deploy workers separately: `make deploy-workers`

## How to monitor and operate

1. Health check: GET http://localhost:8080/health
2. Readiness check: GET http://localhost:8080/ready
3. Metrics: GET http://localhost:8080/metrics (Prometheus format)
4. Ingestion stats: GET http://localhost:8080/api/v1/stats/ingestion
5. View logs: `make logs`
6. ClickHouse console: `clickhouse-client --host localhost`
7. Redis CLI: `redis-cli -h localhost -p 6379`
8. Real-time monitoring: Open Grafana dashboard at http://localhost:3000
9. Alert manager: http://localhost:9093

## How to manage databases

### ClickHouse Operations
1. Apply migrations: `make db-migrate-clickhouse`
2. Create tables: `make db-schema-clickhouse`
3. Optimize tables: `make db-optimize`
4. Backup data: `make db-backup-clickhouse`
5. Query console: `clickhouse-client --host localhost --query "SELECT * FROM events LIMIT 10"`
6. Check table sizes: `make db-stats`
7. Manual compaction: `make db-compact`

### PostgreSQL Operations
1. Apply migrations: `make db-migrate-postgres`
2. Create schema: `make db-schema-postgres`
3. Backup database: `make db-backup-postgres`
4. Query via CLI: `psql -d trellis -c "SELECT * FROM campaigns LIMIT 10"`

## Forge Project Management

- `forge-cli audit` - manages code quality reviews for ingestion pipeline optimization
- `forge-cli batch` - bulk operations on tasks for efficient sprint management
- `forge-cli create-task [title] --definition --notes --criteria --dependencies` - creates tasks with traffic pattern details
- `forge-cli delete-task [task-id] --force --reason` - removes obsolete tasks
- `forge-cli get-next-task --persona --auto-start` - finds highest priority ingestion or analytics feature
- `forge-cli get-task [task-id] --detailed` - retrieves task with ClickHouse query examples
- `forge-cli list-tasks --status --format` - displays sprint backlog
- `forge-cli move-task [task-id] [status]` - transitions task through workflow
- `forge-cli release start/finalize [version]` - manages platform releases
- `forge-cli requirements` - tracks traffic attribution feature specifications
- `forge-cli research` - manages pattern discovery algorithm research
- `forge-cli semantic store/search --category` - manages knowledge base
- `forge-cli sync --fix-duplicates` - synchronizes task state
- `forge-cli update-task [task-id] --notes --confidence` - updates implementation plans

## Development workflows

1. Ingestion pipeline guide - see docs/ingestion-architecture.md
2. ClickHouse queries - see docs/clickhouse-patterns.md
3. Campaign creation flow - see internal/campaign/engine.go
4. Pattern discovery - see internal/discovery/patterns.go
5. Fraud detection - see internal/ingestion/fraud.go
6. SSE streaming - see internal/handlers/stream.go
7. Testing traffic flow - see test/ingestion_test.go
8. Load simulations:
   - Basic traffic: `go run cmd/simulator/main.go --scenario basic --rps 1000`
   - Burst traffic: `go run cmd/simulator/main.go --scenario burst --rps 10000`
   - Pattern testing: `go run cmd/simulator/main.go --scenario patterns`

## How to do the next task

1. Use `forge-cli get-next-task` to find highest priority work
2. Review task focusing on ingestion performance implications
3. If confidence below 80%, research ClickHouse optimization or Go concurrency patterns
4. Move to in-progress: `forge-cli move-task [task-id] in-progress`
5. Update with ClickHouse queries: `forge-cli update-task [task-id] --notes`
6. Write performance benchmarks first targeting sub-100ms latency
7. Implement following clean architecture principles
8. Run `make ci` ensuring all checks pass
9. Test with load simulator at 10K RPS
10. Verify data persistence with ClickHouse queries
11. Document any new traffic patterns discovered
12. Start dev server and manually test ingestion endpoints
13. Complete task: `forge-cli move-task [task-id] completed`
14. Commit with descriptive message about performance improvements

## How to create a task

1. Review docs/architecture.md to understand system design
2. Check api.md for endpoint patterns and query examples
3. Review internal/README.md for component interactions
4. Examine recent tasks: `forge-cli list-tasks --status completed`
5. Develop implementation plan including:
   - ClickHouse schema changes needed
   - Go concurrency patterns to use
   - Redis caching strategy
   - PostgreSQL schema modifications
6. Include specific file paths and line numbers
7. Add example ClickHouse queries in notes
8. Assign confidence based on ingestion complexity
9. Create task: `forge-cli create-task --title --definition --notes --confidence`

## How to ingest traffic

All traffic ingestion now requires organization-aware authentication via Warden API keys:

1. Basic redirect with auth: `curl -H "Authorization: Bearer wdn_your_api_key" "http://localhost:8080/in?click_id=test123&source=manual"`
2. With campaign: `curl -H "Authorization: Bearer wdn_your_api_key" "http://localhost:8080/in/camp_123?click_id=test456"`
3. POST data: `curl -X POST -H "Authorization: Bearer wdn_your_api_key" -H "Content-Type: application/json" http://localhost:8080/in -d '{"data":"test"}'`
4. Pixel tracking: `curl -H "Authorization: Bearer wdn_your_api_key" "http://localhost:8080/pixel.gif?event=view"`
5. Bulk ingestion: Use `scripts/bulk-ingest.sh` with proper API key configuration
6. Load test: Configure authentication in targets.txt for `vegeta attack -rate=1000 -duration=30s -targets=targets.txt`

## How to create campaigns retroactively

Organization-scoped campaign creation:

1. Discover patterns first: `make discover-patterns DAYS=30 ORG=your_org_id`
2. Review unattributed traffic in ClickHouse: `SELECT * FROM events WHERE organization_id = 'your_org_id' AND campaign_id IS NULL`
3. Create campaign JSON definition with organization context
4. Apply retroactively: `curl -X POST -H "Authorization: Bearer wdn_your_api_key" localhost:8080/api/v1/campaigns -d @campaign.json`
5. Verify attribution: `make verify-attribution CAMPAIGN=your_org_id/camp_123`
6. Check performance impact: `make query-performance ORG=your_org_id`

## How to debug ingestion issues

### Common Issues and Solutions

**Issue: High redirect latency (>100ms)**
- Check: Redis connection pool exhaustion
- Debug: `redis-cli INFO clients`
- Check: Async processing blocking
- Solution: Increase pool size or use fire-and-forget

**Issue: Duplicate detection failing**
- Check: Redis TTL settings
- Debug: `redis-cli TTL click:*`
- Solution: Adjust dedup window in config

**Issue: ClickHouse insert delays**
- Check: Batch size and flush interval
- Debug: `clickhouse-client --query "SHOW PROCESSLIST"`
- Solution: Tune batch processor settings

**Issue: Memory growth in workers**
- Check: Goroutine leaks
- Debug: `curl localhost:8080/debug/pprof/goroutine`
- Solution: Ensure proper context cancellation

### Debug Commands
```bash
# Test ingestion pipeline
scripts/test-ingestion-chain

# Monitor Redis operations
redis-cli MONITOR

# Watch ClickHouse inserts by organization
watch 'clickhouse-client --query "SELECT organization_id, count(*) FROM events WHERE event_time > now() - INTERVAL 1 MINUTE GROUP BY organization_id"'

# Profile CPU usage
go tool pprof http://localhost:8080/debug/pprof/profile

# Check memory usage
go tool pprof http://localhost:8080/debug/pprof/heap
```

## Error handling standards

1. **FAIL FAST PHILOSOPHY**
   - NEVER drop traffic data silently
   - NEVER use lossy queues without alerting
   - If we can't store it, we reject it
   - Better to know we're losing data than lose it silently

2. **Ingestion errors**
   - Log and continue for enrichment failures
   - Return 503 if storage is unavailable
   - Use circuit breakers for downstream services
   - Implement exponential backoff for retries

3. **Error propagation pattern**
   ```go
   // Service layer - propagate with context
   if err != nil {
       return fmt.Errorf("clickhouse insert: %w", err)
   }

   // Handler layer - map to HTTP status
   switch {
   case errors.Is(err, context.DeadlineExceeded):
       http.Error(w, "Request timeout", http.StatusGatewayTimeout)
   case errors.Is(err, ErrDuplicate):
       http.Error(w, "Duplicate click", http.StatusConflict)
   default:
       http.Error(w, "Internal error", http.StatusInternalServerError)
   }
   ```

## Logging standards

DEBUG = Detailed request parsing, batch processing metrics, query plans
INFO = Traffic accepted, campaigns created, patterns discovered
WARN = High latency, approaching limits, failed enrichments
ERROR = Failed ingestion, ClickHouse errors, Redis connection issues
CRITICAL = Data loss risk, service degradation, queue overflow

NO EMOJIS = Logs must be parseable by log aggregation systems

## Key operational commands

- `make dev` - start development server with hot reload
- `make ci` - run full CI pipeline with benchmarks
- `make test` - run test suite
- `make bench` - run performance benchmarks
- `make load-test` - run load testing suite
- `docker-compose up` - start all services
- `make db-schema-clickhouse` - apply ClickHouse schema
- `make build` - build production binary
- `make logs` - tail application logs
- `make status` - check system health
- `make discover-patterns` - run pattern discovery
- `make compact-memories` - trigger memory compaction

## Performance targets

- **Ingestion rate**: 100K+ requests/second per node
- **Redirect latency**: <50ms p99
- **Dedup check**: <10ms
- **Async processing**: Fire-and-forget (non-blocking)
- **Batch insert**: 1000 events per batch
- **Query performance**: <2s for 1B rows
- **Memory usage**: <1GB per 100K RPS
- **CPU usage**: <50% at 50K RPS

## Testing traffic patterns

Test various traffic patterns to ensure robustness:
```bash
# High-frequency partner
go run cmd/simulator/main.go --scenario partner --rps 5000

# Bot traffic simulation
go run cmd/simulator/main.go --scenario bot --pattern suspicious

# Geographic distribution
go run cmd/simulator/main.go --scenario geo --regions us,eu,asia

# Click flooding
go run cmd/simulator/main.go --scenario flood --ips 10 --rps 10000

# Parameter fuzzing
go run cmd/simulator/main.go --scenario fuzz --iterations 10000
```

## Helpful Tips

*Lessons learned from building high-throughput systems:*

1. **Always use fire-and-forget for non-critical paths**
   - Mistake: Blocking on async operations in request path
   - Correct: Use goroutines for background processing
   - Why: Reduces p99 latency by 50%

2. **Batch everything going to ClickHouse**
   - Mistake: Individual inserts causing high CPU
   - Correct: Batch with time and size limits
   - Why: 100x throughput improvement

3. **Use context timeouts aggressively**
   - Mistake: Hanging goroutines from slow operations
   - Correct: Context with timeout on all I/O
   - Why: Prevents resource exhaustion

4. **Profile before optimizing**
   - Mistake: Guessing at performance bottlenecks
   - Correct: Use pprof to identify actual issues
   - Why: 80% of time is in 20% of code

5. **Design for partial failures**
   - Mistake: All-or-nothing processing
   - Correct: Degrade gracefully, log failures
   - Why: Some data is better than no data

## Quick diagnostic checklist

When things go wrong, check in this order:
1. Health endpoint responding?
2. Redis connected and responsive?
3. ClickHouse accepting inserts?
4. Background workers processing?
5. Worker goroutines within limits?
6. Memory usage stable?
7. Disk space available?
8. Network latency normal?
9. Error rate within SLA?
10. Upstream services healthy?
