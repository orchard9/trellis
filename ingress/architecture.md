# Trellis Ingress Architecture

## System Overview

Trellis Ingress is the data capture service that stores every piece of traffic data for later analysis and campaign creation.

```
┌─────────────────────────────────────────────────────────────────┐
│                       INGRESS ARCHITECTURE                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────┐         ┌──────────┐         ┌──────────┐       │
│  │  HTTP    │────────▶│  Ingress │────────▶│ClickHouse│       │
│  │ Request  │         │    API   │         │ BigQuery │       │
│  └──────────┘         └──────────┘         └──────────┘       │
│                             │                                   │
│                             ▼                                   │
│                      ┌──────────┐                             │
│                      │  Warden  │                             │
│                      │   Auth   │                             │
│                      └──────────┘                             │
└─────────────────────────────────────────────────────────────────┘
```

## Core Functionality

The ingress service has one job: capture traffic data and store it.

### 1. Data Capture Endpoint

```go
// POST /events
// GET /track
// POST /pixel
type TrafficEvent struct {
    // Organization context from Warden
    OrganizationID string
    
    // Everything from the request
    Timestamp      time.Time
    Method         string
    URL            string
    Headers        map[string]string
    QueryParams    map[string]string
    Body           json.RawMessage
    IP             string
    UserAgent      string
    Referer        string
    
    // Any additional context
    EventType      string
    ClickID        string
    SessionID      string
}
```

### 2. Organization Authentication

Every request must include a Warden API key to identify the organization:

```go
func (h *Handler) CaptureEvent(w http.ResponseWriter, r *http.Request) {
    // Extract organization from Warden API key
    orgCtx, err := h.warden.ValidateAPIKey(r.Header.Get("Authorization"))
    if err != nil {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    
    // Capture all request data
    event := extractEventData(r)
    event.OrganizationID = orgCtx.OrganizationID
    
    // Store in data warehouse
    err = h.storage.StoreEvent(event)
    if err != nil {
        log.Error("Failed to store event", err)
        http.Error(w, "Storage error", http.StatusInternalServerError)
        return
    }
    
    w.WriteHeader(http.StatusOK)
}
```

### 3. Data Storage

Events are stored in both ClickHouse (for fast queries) and BigQuery (for complex analytics):

**ClickHouse Schema**
```sql
CREATE TABLE events (
    organization_id String,
    event_id UUID DEFAULT generateUUIDv4(),
    timestamp DateTime64(3),
    
    -- Request data
    method String,
    url String,
    headers String, -- JSON
    query_params String, -- JSON
    body String, -- JSON
    ip String,
    user_agent String,
    referer String,
    
    -- Event metadata
    event_type String,
    click_id String,
    session_id String,
    
    -- Ingestion metadata
    ingested_at DateTime64(3) DEFAULT now64(3)
)
ENGINE = MergeTree()
PARTITION BY (toYYYYMM(timestamp), organization_id)
ORDER BY (organization_id, timestamp, event_id)
```

**BigQuery Schema**
```sql
CREATE TABLE trellis.events (
    organization_id STRING NOT NULL,
    event_id STRING NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    
    -- Request data stored as JSON for flexibility
    request_data JSON,
    
    -- Extracted fields for common queries
    event_type STRING,
    click_id STRING,
    session_id STRING,
    source STRING,
    medium STRING,
    campaign STRING,
    
    -- Metadata
    ingested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP()
)
PARTITION BY DATE(timestamp)
CLUSTER BY organization_id, event_type;
```

### 4. API Endpoints

```
POST /events
- Capture structured event data
- Request body: JSON event data
- Response: 200 OK or error

GET /track 
- Capture data via query parameters
- Example: /track?event=pageview&page=/home&source=google
- Response: 200 OK or 1x1 pixel

POST /pixel
- Capture data from tracking pixels
- Supports both GET and POST
- Returns transparent 1x1 GIF

POST /batch
- Bulk event ingestion
- Request body: Array of events
- Response: 200 OK with success/failure counts
```

## Data Flow

1. **Receive Request**: HTTP request arrives with tracking data
2. **Authenticate**: Validate Warden API key, extract organization context
3. **Extract Data**: Capture all request information (headers, params, body)
4. **Enrich**: Add metadata like timestamps and IDs
5. **Store**: Write to both ClickHouse and BigQuery
6. **Respond**: Return success response or pixel

## Design Principles

### 1. Store Everything
- Never drop data because we don't understand it yet
- Keep raw request data for future analysis
- Storage is cheap, lost data is expensive

### 2. Schema Flexibility  
- Store structured data in JSON columns
- Allow new fields without schema changes
- Enable retroactive parsing of historical data

### 3. Organization Isolation
- Every event tagged with organization_id
- Data partitioned by organization for performance
- No possibility of cross-organization data access

### 4. Simple and Reliable
- Minimal processing in the ingestion path
- No complex validation or transformation
- Focus on capture completeness over real-time processing

## Performance Considerations

- **Batch writes**: Buffer events and write in batches to ClickHouse
- **Async processing**: Return response immediately, process in background
- **Compression**: Use ClickHouse compression for efficient storage
- **Partitioning**: Partition by time and organization for query performance

## Integration with Other Services

- **Warehouse Service**: Queries the stored events for analytics
- **Campaigns Service**: Uses stored data to create retroactive campaigns
- **Pattern Discovery**: Analyzes historical data to find opportunities

The ingress service is intentionally simple - its only job is to reliably capture and store traffic data so the other services can analyze it later.