# Trellis Performance Standards

This document consolidates all performance targets, SLAs, and monitoring requirements for the Trellis platform.

## Service Level Objectives (SLOs)

### Ingress Service Performance
| Metric | Target | Description | Measurement |
|--------|--------|-------------|-------------|
| **Redirect Latency** | <50ms p99 | Time from request to redirect response | HTTP response time |
| **Throughput** | 100K+ RPS | Requests per second per node | Load testing verified |
| **Storage Reliability** | 99.99% | Events successfully stored or queued | DLQ + primary storage |
| **Dedup Check** | <10ms p95 | Redis lookup time for duplicate detection | Redis latency |
| **Authentication** | <10ms p95 | Warden API key validation time | gRPC call latency |
| **Campaign Resolution** | <5ms p95 | Time to determine campaign routing | Rule evaluation time |
| **Memory Usage** | <1GB per 100K RPS | Memory consumption at scale | Process monitoring |
| **CPU Usage** | <50% at 50K RPS | CPU utilization under load | System monitoring |
| **Worker Queue** | <1000 pending | Async processing backlog | Queue depth monitoring |

### Warehouse Service Performance  
| Metric | Target | Description | Measurement |
|--------|--------|-------------|-------------|
| **Simple Queries** | <500ms p95 | Single metric, basic filters | Dashboard loads |
| **Complex Queries** | <2s p95 | Multi-dimensional analysis | Ad-hoc analytics |
| **Real-time Queries** | <100ms p95 | Last hour metrics | Live dashboards |
| **Custom Queries** | <10s p95 | User-defined complex analysis | Query builder |
| **Cache Hit Ratio** | >80% | Percentage of queries served from cache | Redis/L1 cache stats |
| **Concurrent Queries** | 1000+ per node | Simultaneous query processing | Connection pooling |
| **Memory Efficiency** | <2GB per 1K queries | Memory usage per concurrent query | Resource monitoring |
| **Data Freshness** | <1 minute | Lag from ingestion to queryable | Data pipeline latency |

### Campaigns Service Performance
| Metric | Target | Description | Measurement |
|--------|--------|-------------|-------------|
| **Initial Load** | <3s | First Contentful Paint | Web Vitals |
| **Navigation** | <500ms | Route transitions | Client-side timing |
| **API Calls** | <2s | Dashboard data loading | Network timing |
| **Chart Rendering** | <1s | Complex visualizations | Render performance |
| **Bundle Size** | <500KB gzipped | JavaScript bundle size | Build analysis |
| **Polling Interval** | 30 seconds | Real-time data updates | Configurable refresh |

## Data Storage Performance

### ClickHouse Analytics Database
| Metric | Target | Description |
|--------|--------|-------------|
| **Query Latency** | <2s for 1B rows | Large dataset aggregations |
| **Insert Throughput** | 100K+ events/sec | Batch insertion rate |
| **Storage Compression** | 10:1 ratio | Columnar compression efficiency |
| **Concurrent Reads** | 1000+ queries | Simultaneous analytics queries |
| **Data Retention** | 7 years | Configurable per organization |

### PostgreSQL Relational Database  
| Metric | Target | Description |
|--------|--------|-------------|
| **Transaction Latency** | <10ms p95 | Campaign CRUD operations |
| **Connection Pool** | 50 connections | Max concurrent connections |
| **Replication Lag** | <1 second | Read replica synchronization |
| **Backup Time** | <30 minutes | Full database backup |

### Redis Cache Performance
| Metric | Target | Description |
|--------|--------|-------------|
| **Get Operations** | <1ms p95 | Cache lookup latency |
| **Set Operations** | <2ms p95 | Cache write latency |
| **Memory Usage** | <80% capacity | Redis memory utilization |
| **Hit Ratio** | >80% | Cache effectiveness |
| **Eviction Rate** | <5% | Key eviction frequency |

## Monitoring & Alerting Thresholds

### Critical Alerts (Page on-call)
- Redirect latency > 100ms p95 for 5+ minutes
- Ingress error rate > 1% for 3+ minutes  
- Storage failure rate > 0.1% for 3+ minutes
- Warehouse query failures > 5% for 5+ minutes
- Authentication failures > 2% for 3+ minutes
- Service completely unavailable

### Warning Alerts (Slack notification)
- Redirect latency > 75ms p95 for 10+ minutes
- Cache hit ratio < 70% for 15+ minutes
- Query latency > 5s p95 for 10+ minutes
- Memory usage > 80% for 10+ minutes  
- CPU usage > 75% for 15+ minutes
- Queue depth > 5000 events for 5+ minutes

### Performance Degradation Response
1. **Latency spike**: Check cache performance, scale horizontally
2. **High error rate**: Investigate logs, check service dependencies  
3. **Memory issues**: Restart services, check for memory leaks
4. **Query slowdown**: Check ClickHouse performance, optimize queries
5. **Queue backlog**: Scale async workers, check storage availability

## Load Testing Standards

### Ingress Service Load Tests
```bash
# Baseline performance test
vegeta attack -rate=50000 -duration=60s \
  -targets=ingress-targets.txt \
  | vegeta report

# Peak traffic simulation  
vegeta attack -rate=100000 -duration=30s \
  -targets=ingress-peak.txt \
  | vegeta report

# Sustained load test
vegeta attack -rate=25000 -duration=300s \
  -targets=ingress-sustained.txt \
  | vegeta report
```

**Target File Example:**
```
GET https://ingress.trellis.com/in?click_id=test_${ID}&source=load_test
Authorization: Bearer wdn_sk_load_test_key
```

### Warehouse Service Load Tests  
```bash
# Analytics dashboard load
vegeta attack -rate=100 -duration=60s \
  -targets=warehouse-dashboard.txt \
  | vegeta report

# Complex query load
vegeta attack -rate=10 -duration=300s \
  -targets=warehouse-complex.txt \
  | vegeta report
```

### Campaign Frontend Load Tests
```bash
# Frontend performance testing
lighthouse --chrome-flags="--headless" \
  --output=json \
  --output-path=./reports/lighthouse.json \
  https://campaigns.trellis.com

# Bundle analysis
npm run analyze
```

## Performance Optimization Checklist

### Ingress Service Optimizations
- [ ] Connection pooling configured (25 ClickHouse, 50 Redis)
- [ ] Async processing with worker pools (100+ workers)
- [ ] Dead letter queue for failure handling
- [ ] Request deduplication with Redis
- [ ] Campaign rule caching (15-minute TTL)
- [ ] Gzip compression enabled
- [ ] HTTP/2 support enabled
- [ ] Keep-alive connections enabled

### Warehouse Service Optimizations  
- [ ] Multi-level caching (L1 in-memory, L2 Redis, L3 pre-computed)
- [ ] Query result pagination (max 10K rows)
- [ ] Connection pooling for all databases
- [ ] Materialized views for common queries
- [ ] Query plan caching
- [ ] Columnar compression in ClickHouse
- [ ] Read replicas for analytics queries
- [ ] Async query processing for heavy operations

### Campaigns Frontend Optimizations
- [ ] Code splitting by route
- [ ] Bundle size optimization (<500KB)
- [ ] Image optimization and lazy loading  
- [ ] Virtual scrolling for large lists
- [ ] React memoization for expensive components
- [ ] API response caching with React Query
- [ ] Service worker for offline functionality
- [ ] CDN for static assets

### Database Optimizations
- [ ] ClickHouse partitioning by organization + time
- [ ] Proper indexing on query columns
- [ ] Regular OPTIMIZE TABLE operations
- [ ] Connection pooling and prepared statements
- [ ] Query timeout enforcement (30s)
- [ ] Automatic failover to read replicas
- [ ] Regular vacuum/analyze operations (PostgreSQL)

## Capacity Planning

### Traffic Growth Projections
| Timeline | Expected RPS | Nodes Required | Scaling Strategy |
|----------|--------------|----------------|------------------|
| Current | 10K RPS | 1 ingress node | Single instance |
| 6 months | 50K RPS | 2 ingress nodes | Horizontal scaling |
| 1 year | 100K RPS | 3-4 nodes | Load balancer + auto-scaling |
| 2 years | 500K RPS | 10+ nodes | Multi-region deployment |

### Storage Growth Projections  
| Timeline | Events/Day | Storage/Day | Total Storage |
|----------|------------|-------------|---------------|
| Current | 100M | 50GB | 500GB |
| 6 months | 500M | 250GB | 5TB |
| 1 year | 1B | 500GB | 50TB |
| 2 years | 5B | 2.5TB | 500TB |

### Cost Optimization Targets
- Storage cost per million events: <$0.50
- Compute cost per million requests: <$2.00
- Cache hit ratio improvement: 5% reduction in database load
- Auto-scaling efficiency: 30% cost reduction vs fixed capacity

## Performance Review Process

### Weekly Performance Review
1. Review all SLO metrics and alert frequency
2. Analyze slow query logs and optimize problematic queries
3. Check capacity utilization and scaling needs
4. Review error logs and fix performance-impacting bugs
5. Update performance baselines based on traffic growth

### Monthly Capacity Planning
1. Project traffic growth for next 3 months
2. Plan infrastructure scaling requirements
3. Review and update performance targets
4. Conduct load testing at projected capacity
5. Update disaster recovery and failover procedures

### Quarterly Performance Optimization
1. Comprehensive performance audit of all services
2. Database query optimization and schema updates
3. Frontend performance audit and optimization
4. Update monitoring and alerting thresholds
5. Review and update SLOs based on business requirements

This performance framework ensures Trellis maintains high availability and responsiveness while scaling efficiently with traffic growth.