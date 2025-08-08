-- Trellis ClickHouse Schema
-- Phase 1: Foundation Schema for basic ingestion

-- Create database if not exists
CREATE DATABASE IF NOT EXISTS trellis;

USE trellis;

-- Main events table for storing all traffic
CREATE TABLE IF NOT EXISTS events
(
    -- Core identifiers
    event_id UUID DEFAULT generateUUIDv4(),
    event_date Date DEFAULT toDate(event_time),
    event_time DateTime64(3) DEFAULT now64(3),
    
    -- Organization isolation
    organization_id String,
    
    -- Click tracking
    click_id String,
    campaign_id Nullable(String),
    
    -- Request information
    method String,
    url String,
    path String,
    raw_params String,  -- JSON string for flexibility
    headers Map(String, String),
    body Nullable(String),
    
    -- Network information
    ip IPv4,
    
    -- Enriched data (Phase 2)
    country Nullable(FixedString(2)),
    city Nullable(String),
    coordinates Nullable(Tuple(Float64, Float64)),
    
    -- Device information (Phase 2)
    device_type Nullable(String),
    os Nullable(String),
    os_version Nullable(String),
    browser Nullable(String),
    browser_version Nullable(String),
    is_bot Nullable(UInt8),
    
    -- Fraud signals (Phase 2)
    fraud_flags Array(String),
    fraud_score Nullable(Float32),
    
    -- Attribution
    source Nullable(String),
    medium Nullable(String),
    referrer Nullable(String),
    referrer_domain Nullable(String),
    
    -- Processing metadata
    ingested_at DateTime64(3) DEFAULT now64(3),
    processed_at Nullable(DateTime64(3)),
    enriched_at Nullable(DateTime64(3)),
    
    -- Deduplication
    is_duplicate UInt8 DEFAULT 0,
    
    -- Indexes for common queries
    INDEX idx_click_id click_id TYPE bloom_filter(0.01) GRANULARITY 1,
    INDEX idx_campaign_id campaign_id TYPE bloom_filter(0.01) GRANULARITY 1,
    INDEX idx_source source TYPE bloom_filter(0.01) GRANULARITY 1,
    INDEX idx_country country TYPE bloom_filter(0.01) GRANULARITY 1,
    INDEX idx_organization_id organization_id TYPE bloom_filter(0.01) GRANULARITY 1
)
ENGINE = MergeTree()
PARTITION BY (toYYYYMM(event_date), organization_id)
ORDER BY (organization_id, event_date, event_time, event_id)
SAMPLE BY xxHash32(click_id)
TTL event_date + INTERVAL 90 DAY
SETTINGS index_granularity = 8192;

-- Campaign definitions table
CREATE TABLE IF NOT EXISTS campaigns
(
    -- Organization isolation
    organization_id String,
    campaign_id String,
    name String,
    status String,  -- active, paused, archived
    
    -- Rules stored as JSON
    rules String,  -- JSON array of matching rules
    
    -- Destination
    destination_url String,
    append_params UInt8 DEFAULT 1,
    
    -- Metadata
    created_at DateTime64(3) DEFAULT now64(3),
    updated_at DateTime64(3) DEFAULT now64(3),
    created_by Nullable(String),
    
    -- Statistics (denormalized for performance)
    total_clicks UInt64 DEFAULT 0,
    unique_clicks UInt64 DEFAULT 0,
    last_click_at Nullable(DateTime64(3))
)
ENGINE = ReplacingMergeTree(updated_at)
ORDER BY (organization_id, campaign_id)
SETTINGS index_granularity = 8192;

-- Discovered patterns table (Phase 3)
CREATE TABLE IF NOT EXISTS discovered_patterns
(
    -- Organization isolation
    organization_id String,
    pattern_id String,
    pattern_type String,  -- parameter, temporal, geographic, behavioral
    pattern_data String,  -- JSON with pattern details
    
    -- Metrics
    volume UInt64,
    unique_sources UInt64,
    confidence Float32,
    
    -- Status
    status String,  -- new, reviewed, applied, rejected
    campaign_id Nullable(String),  -- if converted to campaign
    
    -- Metadata
    discovered_at DateTime64(3) DEFAULT now64(3),
    first_seen DateTime64(3),
    last_seen DateTime64(3),
    
    INDEX idx_status status TYPE bloom_filter(0.01) GRANULARITY 1
)
ENGINE = MergeTree()
ORDER BY (organization_id, discovered_at, pattern_id)
SETTINGS index_granularity = 8192;

-- Postbacks table for conversion tracking (Phase 3)
CREATE TABLE IF NOT EXISTS postbacks
(
    postback_id UUID DEFAULT generateUUIDv4(),
    received_at DateTime64(3) DEFAULT now64(3),
    
    -- Organization isolation
    organization_id String,
    
    -- Linking
    click_id String,
    transaction_id Nullable(String),
    
    -- Postback data
    status String,
    value Nullable(Float64),
    currency Nullable(String),
    custom_data Nullable(String),  -- JSON
    
    -- Processing
    processed UInt8 DEFAULT 0,
    retry_count UInt8 DEFAULT 0,
    last_retry_at Nullable(DateTime64(3)),
    
    INDEX idx_click_id click_id TYPE bloom_filter(0.01) GRANULARITY 1
)
ENGINE = MergeTree()
ORDER BY (organization_id, received_at, click_id)
SETTINGS index_granularity = 8192;

-- Materialized view for hourly statistics (Phase 2)
CREATE MATERIALIZED VIEW IF NOT EXISTS events_hourly
ENGINE = SummingMergeTree()
PARTITION BY (toYYYYMM(hour), organization_id)
ORDER BY (organization_id, campaign_id, hour)
AS
SELECT
    organization_id,
    toStartOfHour(event_time) AS hour,
    campaign_id,
    source,
    country,
    device_type,
    
    -- Metrics
    count() AS clicks,
    uniq(click_id) AS unique_clicks,
    uniq(ip) AS unique_ips,
    countIf(is_bot = 1) AS bot_clicks,
    countIf(is_duplicate = 1) AS duplicate_clicks,
    avgIf(fraud_score, fraud_score IS NOT NULL) AS avg_fraud_score
FROM events
WHERE campaign_id IS NOT NULL
GROUP BY organization_id, hour, campaign_id, source, country, device_type;

-- System metrics table for monitoring
CREATE TABLE IF NOT EXISTS system_metrics
(
    timestamp DateTime64(3) DEFAULT now64(3),
    metric_type String,
    metric_name String,
    metric_value Float64,
    labels Map(String, String)
)
ENGINE = MergeTree()
PARTITION BY toDate(timestamp)
ORDER BY (timestamp, metric_type, metric_name)
TTL timestamp + INTERVAL 7 DAY
SETTINGS index_granularity = 8192;

-- Create user for application
-- Note: Run this manually with admin privileges
-- CREATE USER IF NOT EXISTS 'trellis' IDENTIFIED BY 'trellis_dev';
-- GRANT ALL ON trellis.* TO 'trellis';

-- Sample data for testing (remove in production)
INSERT INTO campaigns (organization_id, campaign_id, name, status, rules, destination_url) VALUES
    ('demo_org', 'demo_campaign', 'Demo Campaign', 'active', '[{"source": ["test", "demo"]}]', 'https://example.com/landing'),
    ('demo_org', 'default', 'Default Campaign', 'active', '[]', 'https://example.com/');

-- Optimization settings
OPTIMIZE TABLE events FINAL;
OPTIMIZE TABLE campaigns FINAL;

-- Show tables
SHOW TABLES;
