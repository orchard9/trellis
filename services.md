# Trellis Services Configuration

## Service Overview

Trellis consists of three core services that work together to provide complete traffic attribution and campaign management:

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          TRELLIS PLATFORM                               │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────┐      ┌─────────────┐      ┌─────────────┐           │
│  │   Ingress   │      │  Warehouse  │      │  Campaigns  │           │
│  │   Service   │      │   Service   │      │   Service   │           │
│  │   (Go)      │      │   (Go)      │      │   (React)   │           │
│  │             │      │             │      │             │           │
│  │  Port 8080  │      │  Port 8090  │      │  Port 3000  │           │
│  └─────────────┘      └─────────────┘      └─────────────┘           │
│         │                     │                     │                   │
│         └─────────────────────┴─────────────────────┘                   │
│                               │                                         │
│                    ┌─────────────────────┐                             │
│                    │   Shared Services   │                             │
│                    ├─────────────────────┤                             │
│                    │ • ClickHouse (9000) │                             │
│                    │ • PostgreSQL (5432) │                             │
│                    │ • Redis (6379)      │                             │
│                    │ • Warden (21382)    │                             │
│                    └─────────────────────┘                             │
└─────────────────────────────────────────────────────────────────────────┘
```

## Service Ports & Endpoints

### Ingress Service (Port 8080)
**Purpose**: Traffic capture and routing with sub-50ms redirect latency

| Endpoint | Method | Auth Required | Description |
|----------|--------|---------------|-------------|
| `/health` | GET | No | Health check |
| `/ready` | GET | No | Readiness probe |
| `/in` | GET/POST | Yes | Main traffic ingestion with redirect |
| `/in/{campaign_id}` | GET/POST | Yes | Campaign-specific ingestion |
| `/pixel.gif` | GET | Yes | Tracking pixel endpoint |
| `/postback` | POST | Yes | Conversion tracking |
| `/api/v1/campaigns` | POST | Yes | Create/update campaigns |
| `/api/v1/health` | GET | Yes | Authenticated health check |

### Warehouse Service (Port 8090)
**Purpose**: Analytics API for retroactive campaign analysis

| Endpoint | Method | Auth Required | Description |
|----------|--------|---------------|-------------|
| `/health` | GET | No | Health check |
| `/ready` | GET | No | Readiness probe |
| `/api/v1/analytics/traffic` | GET | Yes | Traffic analytics |
| `/api/v1/analytics/campaigns` | GET | Yes | Campaign performance |
| `/api/v1/analytics/attribution` | GET | Yes | Attribution analysis |
| `/api/v1/analytics/funnel` | GET | Yes | Funnel analysis |
| `/api/v1/analytics/cohorts` | GET | Yes | Cohort analysis |
| `/api/v1/analytics/realtime` | GET | Yes | Real-time metrics |
| `/api/v1/analytics/query` | POST | Yes | Custom analytics queries |
| `/api/v1/patterns/discover` | POST | Yes | Pattern discovery |
| `/api/v1/patterns/analyze` | GET | Yes | Pattern analysis |

### Campaigns Service (Port 3000)
**Purpose**: React frontend for campaign management

| Route | Description |
|-------|-------------|
| `/` | Redirects to dashboard |
| `/login` | Authentication page |
| `/dashboard` | Main campaign overview |
| `/campaigns` | Campaign list |
| `/campaigns/new` | Create new campaign |
| `/campaigns/:id` | Campaign details |
| `/analytics` | Analytics dashboard |
| `/patterns` | Pattern discovery |
| `/settings` | Organization settings |

## Data Flow

### 1. Traffic Ingestion Flow
```
User Click → Ingress Service → Campaign Router → Redirect User
                    ↓
              Async Worker Pool
                    ↓
              ClickHouse Storage
                    ↓
              Dead Letter Queue (on failure)
```

### 2. Analytics Query Flow
```
Frontend Request → Warehouse API → Query Builder → Cache Check
                                        ↓              ↓
                                   ClickHouse ← Redis Cache
                                        ↓
                                   Response → Frontend
```

### 3. Campaign Management Flow
```
Create Campaign → Ingress API → PostgreSQL → Campaign Router Cache
                                     ↓
                              Retroactive Attribution
                                     ↓
                              ClickHouse Update
```

## Inter-Service Communication

### Authentication Flow
All services use Warden for organization-aware authentication:

1. **Service Authentication** (API Keys)
   - Format: `Authorization: Bearer wdn_service_key`
   - Used by: Backend services
   - Validates: Organization context, permissions

2. **User Authentication** (JWT Tokens)
   - Format: `Authorization: Bearer eyJ...` (JWT)
   - Used by: Frontend users
   - Contains: User ID, organization ID, permissions

### Service Dependencies

```yaml
ingress:
  depends_on:
    - warden      # Authentication
    - clickhouse  # Event storage
    - redis       # Caching & dedup
    - postgres    # Campaign data

warehouse:
  depends_on:
    - warden      # Authentication
    - clickhouse  # Analytics queries
    - postgres    # Campaign metadata
    - redis       # Query caching

campaigns:
  depends_on:
    - ingress     # Campaign management
    - warehouse   # Analytics data
    - warden      # User auth
```

## Environment Configuration

### Common Environment Variables
```bash
# Authentication
WARDEN_ADDRESS=localhost:21382
WARDEN_SERVICE_API_KEY=wdn_service_key_here

# Databases
CLICKHOUSE_HOST=localhost
CLICKHOUSE_PORT=9000
CLICKHOUSE_DATABASE=trellis
CLICKHOUSE_USER=default
CLICKHOUSE_PASSWORD=

POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DATABASE=trellis
POSTGRES_USER=trellis
POSTGRES_PASSWORD=trellis_pass

REDIS_URL=redis://localhost:6379/0

# Organization
DEFAULT_ORGANIZATION_ID=org_default
```

### Service-Specific Configuration

**Ingress Service**
```bash
TRELLIS_PORT=8080
WORKER_POOL_SIZE=100
BATCH_SIZE=1000
BATCH_FLUSH_INTERVAL=5s
DEDUP_WINDOW=24h
REDIRECT_TIMEOUT=50ms
```

**Warehouse Service**
```bash
WAREHOUSE_PORT=8090
QUERY_TIMEOUT=30s
CACHE_TTL_REALTIME=1m
CACHE_TTL_HOURLY=15m
CACHE_TTL_DAILY=4h
MAX_QUERY_ROWS=1000000
CONCURRENT_QUERIES=1000
```

**Campaigns Service**
```bash
VITE_PORT=3000
VITE_WAREHOUSE_API_URL=http://localhost:8090
VITE_INGRESS_API_URL=http://localhost:8080
VITE_WARDEN_API_URL=http://localhost:21382
VITE_POLLING_INTERVAL=30000
```

## Performance Targets

### Ingress Service
- **Redirect latency**: <50ms p99
- **Throughput**: 100K+ RPS per node
- **Storage reliability**: 99.99% with DLQ

### Warehouse Service
- **Query latency**: <2s for complex queries
- **Cache hit ratio**: >80%
- **Concurrent queries**: 1000+ per node

### Campaigns Service
- **Page load**: <3s initial load
- **Navigation**: <500ms transitions
- **API response**: <2s for analytics

## Monitoring & Health Checks

### Health Check Endpoints
Each service exposes health endpoints for monitoring:

```bash
# Basic health (no auth required)
curl http://localhost:8080/health
curl http://localhost:8090/health

# Detailed health (auth required)
curl -H "Authorization: Bearer wdn_key" http://localhost:8080/api/v1/health
curl -H "Authorization: Bearer wdn_key" http://localhost:8090/api/v1/health
```

### Key Metrics to Monitor
- Request latency (p50, p95, p99)
- Error rates by endpoint
- Database connection pool usage
- Cache hit ratios
- Queue depths (async workers, DLQ)
- Memory and CPU usage

## Deployment Considerations

### Scaling Strategy
- **Ingress**: Scale horizontally based on RPS
- **Warehouse**: Scale based on query load
- **Campaigns**: CDN for static assets, scale API gateway

### High Availability
- Multiple instances of each service
- Redis Sentinel for cache HA
- ClickHouse cluster with replicas
- PostgreSQL with read replicas
- Geographic load balancing

This configuration ensures all services work together seamlessly while maintaining complete organization isolation and high performance.