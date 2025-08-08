# Trellis Scripts

This directory contains utility scripts for development and operations.

## Directory Structure

```
scripts/
├── clickhouse/         # ClickHouse related scripts
│   ├── init.sql       # Initial database creation
│   └── schema.sql     # Complete schema definition
├── load-test.sh       # Load testing with vegeta
└── setup.sh           # Setup helper script
```

## Scripts

### ClickHouse Scripts

#### `clickhouse/init.sql`
Basic initialization that runs when ClickHouse container starts. Creates the database.

#### `clickhouse/schema.sql`
Complete schema definition including:
- Main `events` table for traffic data
- `campaigns` table for campaign definitions
- `discovered_patterns` table for ML discoveries
- `postbacks` table for conversion tracking
- Materialized views for aggregations
- Indexes and partitioning setup

Apply with:
```bash
make db-schema
```

### Load Testing

#### `load-test.sh`
Comprehensive load testing script using vegeta:
- Tests at 100, 1000, and 5000 RPS
- Generates random traffic patterns
- Measures latency percentiles
- Verifies data in ClickHouse

Run with:
```bash
make load-test
# or directly:
./scripts/load-test.sh
```

## Adding New Scripts

When adding new scripts:
1. Make them executable: `chmod +x script.sh`
2. Add error handling: `set -e` at the top
3. Include help text and usage examples
4. Document in this README
5. Add corresponding Make target if appropriate

## Environment Variables

Scripts respect these environment variables:
- `API_URL` - API endpoint (default: http://localhost:8080)
- `CLICKHOUSE_HOST` - ClickHouse host (default: localhost)
- `REDIS_URL` - Redis connection URL (default: redis://localhost:6379)
