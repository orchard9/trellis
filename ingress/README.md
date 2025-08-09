# Trellis Ingress Service

The Trellis Ingress Service is responsible for capturing all incoming traffic data with organization-aware authentication and routing. It serves as the entry point for all traffic attribution data.

## Purpose

- **Universal Traffic Capture**: Accepts traffic from any source without pre-configuration
- **Organization Isolation**: Ensures complete data separation between organizations
- **Real-time Processing**: Handles high-volume traffic with sub-100ms redirect latency
- **Fraud Detection**: Identifies and flags suspicious traffic patterns
- **Campaign Routing**: Routes traffic to appropriate destinations based on campaign rules

## Architecture

The ingress service is built with Go for maximum performance and uses:

- **Chi Router**: Lightweight HTTP routing with middleware support
- **Warden Integration**: Organization-aware authentication and authorization
- **ClickHouse**: Primary data storage for events and campaigns
- **Redis**: Deduplication caching and session management
- **Async Workers**: Background goroutines for non-blocking event storage

## Key Features

### Organization-Aware Authentication
All endpoints require valid Warden API keys. Traffic is automatically scoped to the authenticated organization.

### High-Performance Ingestion
- Sub-100ms redirect latency
- 100K+ requests/second per node
- Fire-and-forget async processing
- Efficient deduplication

### Flexible Campaign Routing
- Dynamic campaign creation and modification
- Rule-based traffic routing
- Retroactive campaign attribution
- Parameter preservation and forwarding

### Privacy-Conscious User Tracking
- Generates consistent user IDs without invasive tracking
- Uses Warden JWT for authenticated users
- Non-invasive fingerprinting for anonymous users
- Snowflake ID generation based on organization context

## API Endpoints

### Traffic Ingestion
- `GET|POST /in` - Main traffic ingestion endpoint
- `GET|POST /in/{campaign_id}` - Campaign-specific traffic ingestion
- `GET /pixel.gif` - Pixel tracking for impressions
- `GET|POST /postback` - Conversion tracking endpoint

### Management
- `GET /health` - Service health check
- `GET /ready` - Readiness probe
- `GET /api/v1/health` - Authenticated organization health check

## Configuration

See `.env.example` for all configuration options. Key settings:

- `WARDEN_ADDRESS`: Warden service endpoint for authentication
- `CLICKHOUSE_HOST`: ClickHouse database for event storage
- `REDIS_URL`: Redis instance for caching and deduplication
- `WORKER_POOL_SIZE`: Number of async workers for event storage (default: 100)

## Development

```bash
# Install dependencies
go mod download

# Run with development settings
go run cmd/api/main.go

# Build for production
go build -o bin/ingress cmd/api/main.go
```

## Deployment

The service is designed to run in Kubernetes with horizontal scaling:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: trellis-ingress
spec:
  replicas: 3
  selector:
    matchLabels:
      app: trellis-ingress
  template:
    spec:
      containers:
      - name: ingress
        image: trellis/ingress:latest
        ports:
        - containerPort: 8080
        env:
        - name: WARDEN_ADDRESS
          value: "warden.trellis.svc.cluster.local:21382"
```

## Performance Characteristics

- **Latency**: <50ms p99 redirect time
- **Throughput**: 100K+ requests/second per node
- **Memory Usage**: <1GB per 100K RPS
- **CPU Usage**: <50% at 50K RPS

## Monitoring

The service exposes metrics at `/metrics` in Prometheus format:

- `trellis_ingress_requests_total` - Total requests processed
- `trellis_ingress_redirect_latency` - Redirect latency histogram
- `trellis_ingress_fraud_detections` - Fraud detections by type
- `trellis_ingress_organization_requests` - Requests by organization

## Related Services

- **Warehouse Service**: Consumes ingested data for analytics
- **Campaigns Service**: Manages campaign definitions and rules
- **Warden**: Provides organization authentication and authorization