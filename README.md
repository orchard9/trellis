# Trellis

> **Universal traffic ingestion gateway with time-travel analytics**

Trellis captures every bit of incoming traffic data and enables retroactive analysis through powerful data warehousing. Define campaigns after traffic arrives. Discover patterns you didn't know existed. Never lose attribution data again.

## Core Features

### **Universal Capture**

-   Ingests ANY traffic source without prior configuration
-   Stores complete request context (headers, params, body)
-   No data model requirements - completely schema-agnostic

### **Time-Travel Analytics**

-   Apply new attribution models to historical data
-   Create campaigns retroactively
-   Discover patterns post-hoc

### **Data Warehouse Native**

-   ClickHouse for blazing-fast analytics
-   PostgreSQL for campaign management and metadata
-   SQL-accessible raw data

### **Scale Ready**

-   200K to 20M+ requests/day
-   Sub-100ms redirect latency
-   Global distribution ready

## Quick Start

### Prerequisites

-   Warden service running for organization management
-   Valid Warden API key for your organization

```bash
# Clone the repository
git clone https://github.com/orchard9/trellis.git
cd trellis

# Configure environment
cp .env.example .env
# Edit .env with your Warden connection details and API key

# Start the stack
docker-compose up -d

# Verify health (requires authentication)
curl -H "Authorization: Bearer wdn_your_api_key" http://localhost:8080/health
```

### Organization Setup

1. **Create Organization in Warden**:

```bash
grpcurl -plaintext -d '{
  "name": "Your Company",
  "slug": "your-company",
  "description": "Your organization description"
}' localhost:21382 warden.v1.OrganizationService/CreateOrganization
```

2. **Create Service Account for Trellis**:

```bash
grpcurl -plaintext -H "Authorization: Bearer your_jwt" -d '{
  "organization_id": "your-org-id",
  "name": "Trellis Ingestion Service",
  "description": "Service account for traffic ingestion"
}' localhost:21382 warden.v1.OrganizationService/CreateServiceAccount
```

3. **Generate API Key**:

```bash
grpcurl -plaintext -H "Authorization: Bearer your_jwt" -d '{
  "account_id": "service-account-id",
  "name": "Trellis API Key",
  "scopes": ["traffic:ingest", "campaigns:manage"]
}' localhost:21382 warden.v1.AuthService/CreateApiKey
```

4. **Update Configuration**:

```bash
# Add to .env file
WARDEN_ADDRESS=your-warden-host:21382
WARDEN_SERVICE_API_KEY=wdn_your_generated_api_key
```

## Basic Usage

### 1. Send Traffic (Organization-Aware!)

```bash
# All traffic ingestion requires organization authentication
curl -H "Authorization: Bearer wdn_your_api_key" \
  "https://trellis.yourdomain.com/in?source=facebook&campaign=summer2024&click_id=abc123&custom_param=anything"

# Trellis captures EVERYTHING within your organization scope and redirects
# → 302 Redirect to your configured destination
# → Data automatically isolated by organization_id
```

### 2. Explore Your Data (Organization-Scoped)

```sql
-- Query your organization's traffic in ClickHouse
SELECT
    JSONExtractString(raw_params, 'source') as source,
    COUNT(*) as clicks,
    COUNT(DISTINCT ip) as unique_ips
FROM events
WHERE organization_id = 'your-org-id'
  AND event_date >= today() - 30
GROUP BY source
ORDER BY clicks DESC
```

### 3. Create Campaigns (Organization-Scoped & Retroactive!)

```bash
# All campaign management requires authentication
curl -X POST -H "Authorization: Bearer wdn_your_api_key" \
  -H "Content-Type: application/json" \
  http://localhost:8080/api/v1/campaigns -d '{
  "organization_id": "your-org-id",
  "campaign_id": "summer_social_2024",
  "name": "Summer Social Campaign",
  "status": "active",
  "rules": [
    {
      "field": "source",
      "operator": "in",
      "values": ["facebook", "instagram"],
      "priority": 10
    }
  ],
  "destination_url": "https://landing.example.com",
  "append_params": true
}'

# Trellis retroactively attributes historical traffic within your organization
```

## Traffic Ingestion

All traffic ingestion requires organization authentication via Warden API keys:

```bash
# GET redirect (with authentication)
curl -H "Authorization: Bearer wdn_your_api_key" \
  "http://localhost:8080/in?source=google&click_id=abc123"

# POST pass-through (with authentication)
curl -X POST -H "Authorization: Bearer wdn_your_api_key" \
  -H "Content-Type: application/json" \
  http://localhost:8080/in -d '{"custom": "data"}'

# Pixel tracking (with authentication)
curl -H "Authorization: Bearer wdn_your_api_key" \
  "http://localhost:8080/pixel.gif?event=view&click_id=abc123"

# Postback endpoint (with authentication)
curl -H "Authorization: Bearer wdn_your_api_key" \
  "http://localhost:8080/postback?click_id=abc123&status=converted&value=25.00"
```

**Organization Isolation**: All data is automatically scoped to your organization. You can only access and manage traffic within your organization's boundaries.

## Data Access

### Real-Time API (Organization-Aware)

```bash
# Get organization-scoped raw logs
curl -H "Authorization: Bearer wdn_your_api_key" \
  "http://localhost:8080/api/v1/analytics/raw-logs?from=2024-01-01&to=2024-01-31&filter=source:google"

# Organization-scoped campaign stats
curl -H "Authorization: Bearer wdn_your_api_key" \
  "http://localhost:8080/api/v1/campaigns/your-campaign/stats?group_by=hour&metrics=clicks,uniques"

# Custom SQL query (automatically scoped to your organization)
curl -X POST -H "Authorization: Bearer wdn_your_api_key" \
  -H "Content-Type: application/json" \
  http://localhost:8080/api/v1/analytics/query -d '{
  "sql": "SELECT * FROM events WHERE raw_params LIKE '%utm_source=google%'"
}'
```

**Note**: All queries are automatically filtered to your organization. You cannot access other organizations' data.

### Data Warehouse Sync (Organization-Partitioned)

```yaml
# PostgreSQL for campaign and organization data
Dataset: trellis_raw
Tables:
    - events_YYYYMMDD (partitioned by date and organization_id)
    - aggregates_hourly (organization-scoped aggregations)
    - campaign_attribution (per-organization campaign data)
# Each organization's data is isolated:
# - Organization-scoped tables with row-level isolation
# - Organization-specific access controls
# - Isolated data export processes
```

## Campaign Intelligence

### Pattern Discovery

```json
// Let Trellis discover patterns
// POST /api/v1/discovery/patterns
{
  "timeframe": "last_30_days",
  "min_volume": 1000
}

// Returns:
{
  "discovered_sources": [
    {
      "pattern": "utm_source=secret_partner",
      "volume": 45000,
      "unique_ips": 12000,
      "suggested_campaign": "unknown_partner_001"
    }
  ]
}
```

### Retroactive Attribution

```json
// Changed your attribution model? Apply it to old data
// POST /api/v1/discovery/reattribute
{
	"from": "2024-01-01",
	"to": "2024-12-31",
	"new_model": {
		"window": "30_days",
		"method": "last_click"
	}
}
```

## Architecture Overview

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│   Traffic   │────▶│   Trellis    │────▶│ ClickHouse  │
└─────────────┘     │   Gateway    │     └─────────────┘
                    └──────────────┘              │
                           │                      ▼
                           │              ┌─────────────┐
                           └─────────────▶│ PostgreSQL  │
                                         └─────────────┘
```

## Configuration

### Environment Variables

```bash
# Core
TRELLIS_PORT=8080
TRELLIS_ENV=production

# ClickHouse
CLICKHOUSE_HOST=localhost
CLICKHOUSE_PORT=8123
CLICKHOUSE_DATABASE=trellis

# Google Cloud
GCP_PROJECT_ID=your-project
BIGQUERY_DATASET=trellis_warehouse
GCS_BUCKET=trellis-cold-storage

# Redis (for dedup & routing cache)
REDIS_URL=redis://localhost:6379

# Monitoring
SENTRY_DSN=https://xxx@sentry.io/yyy
```

## Fraud Detection

Trellis flags suspicious traffic without blocking:

```json
// Fraud flags in data
{
	"fraud_flags": [
		"duplicate_click", // Same fingerprint within 5 seconds
		"datacenter_ip", // Non-residential IP
		"high_click_velocity", // >10 clicks/second from same source
		"bot_ua_pattern" // Known bot user-agent
	],
	"fraud_score": 0.75 // ML-based probability
}
```

## API Reference

See [API Documentation](./docs/api.md) for complete reference.

## Performance

-   **Ingestion**: 100K+ requests/second per node (Go implementation)
-   **Redirect Latency**: <50ms p99 (Go's superior concurrency)
-   **Query Performance**: 1B rows in <2 seconds (ClickHouse)
-   **Storage**: ~1KB per event (compressed)
-   **Memory Usage**: 50% less than Node.js equivalent
-   **Container Size**: ~10MB (distroless Go binary)

## Deployment

### Docker Compose (Development)

```bash
docker-compose up
```

### Kubernetes (Production)

```bash
kubectl apply -f k8s/
```

### Google Cloud Run (Serverless)

```bash
gcloud run deploy trellis --source .
```

## License

Proprietary

---

Built by the Orchard9 team. Never lose traffic data again.
