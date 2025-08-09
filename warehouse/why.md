# Why Trellis Warehouse?

## The Analytics Problem

Traditional analytics systems fail when you need to understand your traffic data. Here's why:

### 1. **Rigid Schema Requirements**
Most analytics platforms force you to define your schema upfront:
- Fixed column structures that can't evolve
- Pre-defined metrics that miss important insights
- Limited ability to ask new questions about historical data
- Schema changes require data migrations and downtime

### 2. **Query Limitations**
Traditional systems can't handle complex questions:
- "Show me all traffic that converted within 30 days, grouped by original source and final touchpoint"
- "Which parameter combinations predict the highest lifetime value?"
- "How do fraud patterns change across different geographic regions and time periods?"
- "What would our attribution look like if we used a different model?"

### 3. **Organization Blindness**
Multi-tenant analytics systems often have:
- Shared databases with poor isolation
- Complex access control that slows down queries
- Risk of data leakage between organizations
- Inability to scale per-organization workloads independently

## The Warehouse Solution

**Query Everything, Understand Deeply**

Trellis Warehouse is built on the principle that **your data should answer any question you can think of**, not just the questions you thought to ask when you set it up.

### Core Principles

1. **Schema-Free Analytics**
   - Query any parameter combination ever captured
   - Retroactively apply new analytics models
   - No migrations needed for new analysis types
   - Complete flexibility in how you slice and dice data

2. **Organization-First Architecture**
   - Complete data isolation between organizations
   - Per-organization query performance optimization
   - Secure multi-tenancy without performance penalties
   - Independent scaling based on organization needs

3. **Performance Without Compromise**
   - Sub-second queries on billion-row datasets
   - Intelligent caching that learns your patterns
   - Multi-engine architecture (ClickHouse + PostgreSQL)
   - Automatic query optimization

### The Warehouse Advantage

**Traditional Analytics:**
```
Question → Fixed Schema → Limited Query → Pre-computed Answer
          ↑ (constrains what you can ask)
```

**Trellis Warehouse:**
```
Question → Flexible Engine → Raw Data → Real-time Answer
                           ↑ (can ask anything)
```

## Real-World Impact

### Problem: Cross-Campaign Attribution
**Traditional System**: "We can only show last-click attribution. Multi-touch requires a complete rebuild."
**Trellis Warehouse**: "Here are 12 different attribution models applied to your historical data. Which one gives you better insights?"

### Problem: Retroactive Analysis
**Traditional System**: "We can't analyze that pattern - we weren't tracking those fields when the data was collected."
**Trellis Warehouse**: "We captured everything. Let me show you that pattern across the last 2 years of data."

### Problem: Organization Performance
**Traditional System**: "Queries are slow because we have to filter millions of rows for your organization."
**Trellis Warehouse**: "Your data is isolated and optimized. Billion-row queries return in under 2 seconds."

### Problem: Complex Questions
**Traditional System**: "That requires a data scientist and 2 weeks of custom development."
**Trellis Warehouse**: "Here's the SQL query. Results in 1.3 seconds."

## Technical Excellence

### Multi-Engine Architecture
- **ClickHouse**: Real-time analytics and event storage
- **PostgreSQL**: Campaign metadata and relational data
- **Automatic Routing**: System chooses the optimal engine for each query

### Organization-Aware Performance
```go
// Every query automatically optimized for organization
SELECT * FROM events 
WHERE organization_id = 'your-org-id'  -- Automatic partition pruning
  AND event_time > now() - 7 days      -- Hot data in ClickHouse
```

### Intelligent Caching
```yaml
cache_strategy:
  per_organization: true
  query_fingerprinting: enabled
  automatic_invalidation: true
  hit_rate_target: 90%
```

### Export Capabilities
- **Multiple Formats**: JSON, CSV, Parquet
- **Streaming Exports**: Handle datasets of any size
- **Cloud Integration**: Direct export to GCS/S3
- **Scheduled Exports**: Automated data delivery

## Why Now?

1. **Columnar Databases Mature**: ClickHouse makes billion-row queries practical
2. **Cloud Storage Cheap**: Keeping everything costs less than losing insights
3. **Analytics Democratized**: Business users need direct access to data
4. **Real-Time Decisions**: Modern business moves too fast for batch analytics

## Query Examples That Were Impossible Before

### Cross-Campaign Journey Analysis
```sql
WITH user_journeys AS (
  SELECT 
    ip,
    arrayJoin(groupArray(source)) AS touchpoint_sequence,
    min(event_time) as first_touch,
    max(event_time) as last_touch,
    countIf(fraud_score < 0.5) as quality_touchpoints
  FROM events 
  WHERE organization_id = 'your-org-id'
    AND event_time >= now() - 30 days
  GROUP BY ip
)
SELECT 
  touchpoint_sequence,
  count(*) as journey_frequency,
  avg(quality_touchpoints) as avg_quality
FROM user_journeys
WHERE quality_touchpoints > 2
GROUP BY touchpoint_sequence
ORDER BY journey_frequency DESC
```

### Fraud Pattern Evolution
```sql
SELECT 
  toStartOfWeek(event_time) as week,
  country,
  avg(fraud_score) as avg_fraud_score,
  countIf(fraud_score > 0.8) as high_fraud_count,
  count(*) as total_clicks
FROM events
WHERE organization_id = 'your-org-id'
  AND event_time >= now() - 90 days
GROUP BY week, country
HAVING total_clicks > 1000
ORDER BY week DESC, avg_fraud_score DESC
```

### Retroactive Attribution Modeling
```sql
-- Apply different attribution models to the same data
SELECT 
  'last_click' as model,
  source,
  count(*) as attributed_conversions
FROM events e1
WHERE organization_id = 'your-org-id'
  AND EXISTS (
    SELECT 1 FROM conversions c 
    WHERE c.click_id = e1.click_id
  )
UNION ALL
SELECT 
  'first_click' as model,
  first_value(source) OVER (
    PARTITION BY ip 
    ORDER BY event_time 
    ROWS UNBOUNDED PRECEDING
  ) as source,
  count(*) as attributed_conversions
FROM events e2
WHERE organization_id = 'your-org-id'
  AND EXISTS (
    SELECT 1 FROM conversions c 
    WHERE c.click_id = e2.click_id
  )
```

## The Bottom Line

Your traffic data contains insights you haven't discovered yet. Trellis Warehouse ensures you can ask any question of your data, get answers in seconds, and trust that your organization's data is completely secure.

Stop asking "What reports do we have?" 
Start asking "What do we want to understand?"

Every query reveals new insights. Trellis Warehouse makes sure you never run out of questions to ask.