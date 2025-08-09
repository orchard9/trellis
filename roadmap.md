# Trellis Development Roadmap

## Vision: "Store first, understand later" - Every click captured, every campaign discoverable

This roadmap breaks down Trellis development into testable releases that progressively build toward the MVP: **organization-native traffic attribution with retroactive campaign creation**.

---

## Step 0: Foundation
**Goal**: Basic infrastructure and authentication

### Ingress Phase
- [ ] Basic HTTP server with Chi router
- [ ] Warden authentication integration
- [ ] Health check endpoints
- [ ] Basic request logging

### Warehouse Phase  
- [ ] ClickHouse connection and basic schema
- [ ] PostgreSQL setup for relational data
- [ ] Redis caching layer
- [ ] Basic data models

### UX Phase
- [ ] React app setup with TypeScript
- [ ] Authentication flow with Warden JWT
- [ ] Basic layout and navigation
- [ ] Health monitoring dashboard

**Success Criteria**: Services start, authenticate, and connect to databases

---

## Release 1: Traffic Capture & Redirect
**Goal**: Capture every click and redirect users quickly

### Ingress Phase
- [ ] Basic traffic ingestion endpoint (`/in`)
- [ ] Simple redirect functionality (<100ms)
- [ ] Request data capture (headers, params, IP)
- [ ] Async event storage
- [ ] Dead letter queue for failures

### Warehouse Phase
- [ ] Event storage in ClickHouse
- [ ] Basic event queries by organization
- [ ] Simple analytics API endpoints
- [ ] Data retention policies

### UX Phase
- [ ] Real-time traffic monitoring
- [ ] Basic event viewer
- [ ] Organization context display
- [ ] Simple metrics dashboard

**Success Criteria**: Click a link → get redirected → see the event in dashboard

---

## Release 2: Analytics Foundation
**Goal**: Understand traffic patterns and performance

### Ingress Phase
- [ ] Campaign ID extraction from URLs
- [ ] Enhanced request enrichment (GeoIP, device detection)
- [ ] User tracking with snowflake IDs
- [ ] Conversion tracking endpoint (`/postback`)

### Warehouse Phase
- [ ] Traffic analytics queries (clicks, sources, countries)
- [ ] Time-series aggregations
- [ ] Campaign performance metrics
- [ ] Multi-level caching strategy

### UX Phase
- [ ] Analytics dashboard with charts
- [ ] Traffic source breakdown
- [ ] Geographic distribution maps
- [ ] Real-time metrics with polling

**Success Criteria**: See where traffic comes from and how it performs

---

## Release 3: Campaign Management
**Goal**: Create and manage traffic routing campaigns

### Ingress Phase
- [ ] Campaign routing engine with rules
- [ ] Multiple destination support
- [ ] Rule priority system
- [ ] Campaign-specific ingestion endpoints

### Warehouse Phase
- [ ] Campaign storage in PostgreSQL
- [ ] Campaign performance analytics
- [ ] Rule effectiveness tracking
- [ ] Organization-scoped campaign queries

### UX Phase
- [ ] Campaign creation wizard
- [ ] Rule builder interface
- [ ] Campaign list and management
- [ ] Performance comparison views

**Success Criteria**: Create a campaign, route traffic, measure performance

---

## Release 4: Retroactive Attribution (Core MVP)
**Goal**: Create campaigns for traffic that already happened

### Ingress Phase
- [ ] Enhanced data capture for retroactive analysis
- [ ] Parameter preservation and forwarding
- [ ] Campaign rule matching for historical data
- [ ] Bulk attribution updates

### Warehouse Phase
- [ ] Time-travel query capabilities
- [ ] Retroactive campaign application
- [ ] Historical traffic pattern analysis
- [ ] Attribution accuracy validation

### UX Phase
- [ ] Retroactive campaign creation wizard
- [ ] Historical traffic browser
- [ ] Before/after attribution comparison
- [ ] Pattern discovery interface

**Success Criteria**: Find unattributed traffic, create campaign, apply retroactively

---

## Release 5: Pattern Discovery
**Goal**: AI discovers winning patterns automatically

### Ingress Phase
- [ ] Advanced fraud detection
- [ ] Traffic quality scoring  
- [ ] Pattern-based alerts
- [ ] Suspicious traffic handling

### Warehouse Phase
- [ ] Machine learning pattern detection
- [ ] Traffic clustering algorithms
- [ ] Anomaly detection system
- [ ] Pattern confidence scoring

### UX Phase
- [ ] AI-suggested campaigns
- [ ] Pattern visualization
- [ ] Fraud alert dashboard
- [ ] Quality score displays

**Success Criteria**: AI discovers profitable traffic patterns and suggests campaigns

---

## Release 6: Advanced Analytics
**Goal**: Deep insights and multi-touch attribution

### Ingress Phase
- [ ] A/B testing infrastructure
- [ ] Advanced user journey tracking
- [ ] Cross-device attribution support
- [ ] Enhanced conversion tracking

### Warehouse Phase
- [ ] Multi-touch attribution models
- [ ] Customer journey analysis
- [ ] Cohort and funnel analytics
- [ ] Advanced segmentation

### UX Phase
- [ ] Attribution modeling interface
- [ ] Customer journey visualization
- [ ] Advanced analytics reports
- [ ] Custom dashboard builder

**Success Criteria**: Understand complete customer journeys and optimize attribution

---

## Release 7: Production Polish
**Goal**: Scale, reliability, and advanced features

### Ingress Phase
- [ ] Performance optimization (<50ms p99)
- [ ] Horizontal scaling support
- [ ] Advanced caching strategies
- [ ] Production monitoring

### Warehouse Phase
- [ ] Query optimization for billions of rows
- [ ] Real-time data streaming
- [ ] Advanced caching and pre-aggregation
- [ ] Data warehouse scale-out

### UX Phase
- [ ] Advanced user experience polish
- [ ] Mobile-responsive design
- [ ] Collaboration features
- [ ] Advanced customization options

**Success Criteria**: Handle 100K+ RPS with sub-second analytics queries

---

## Testing Strategy Per Release

### Integration Tests
- [ ] End-to-end traffic flow testing
- [ ] API contract testing between services
- [ ] Authentication and authorization testing
- [ ] Data consistency validation

### Performance Tests
- [ ] Load testing at target RPS
- [ ] Latency measurement and optimization
- [ ] Database performance under load
- [ ] Memory and CPU usage profiling

### User Acceptance Tests
- [ ] Core user workflows validation
- [ ] Dashboard functionality testing
- [ ] Campaign creation and management
- [ ] Analytics accuracy verification

---

## Success Metrics by Release

| Release | Traffic Handling | Analytics Speed | User Experience |
|---------|------------------|-----------------|------------------|
| 1 | 1K RPS | N/A | Basic monitoring |
| 2 | 5K RPS | <5s queries | Simple analytics |
| 3 | 10K RPS | <3s queries | Campaign management |
| 4 | 25K RPS | <2s queries | Retroactive creation |
| 5 | 50K RPS | <2s queries | AI suggestions |
| 6 | 75K RPS | <1s queries | Advanced analytics |
| 7 | 100K+ RPS | <1s queries | Production ready |

This roadmap ensures each release delivers testable value while building systematically toward the complete Trellis vision.