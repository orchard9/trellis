#!/bin/bash
# Trellis Load Testing Script
# Tests ingestion capability at various rates

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Configuration
API_URL="${API_URL:-http://localhost:8080}"
DURATION="${DURATION:-30s}"

echo "🚀 Trellis Load Testing"
echo "======================"
echo "Target: $API_URL"
echo "Duration: $DURATION"
echo ""

# Check if vegeta is installed
if ! command -v vegeta &> /dev/null; then
    echo -e "${RED}❌ vegeta is not installed${NC}"
    echo "Install with: go install github.com/tsenart/vegeta/v12@latest"
    exit 1
fi

# Check if API is responding
if ! curl -s -f "$API_URL/health" > /dev/null 2>&1; then
    echo -e "${RED}❌ API is not responding at $API_URL/health${NC}"
    echo "Make sure the API is running: make dev"
    exit 1
fi

echo -e "${GREEN}✅ API is healthy${NC}"
echo ""

# Function to generate random parameters
generate_params() {
    local sources=("google" "facebook" "twitter" "reddit" "direct")
    local campaigns=("summer2024" "black_friday" "launch" "retarget" "brand")
    
    local source=${sources[$RANDOM % ${#sources[@]}]}
    local campaign=${campaigns[$RANDOM % ${#campaigns[@]}]}
    local click_id=$(uuidgen | tr '[:upper:]' '[:lower:]')
    
    echo "source=$source&campaign=$campaign&click_id=$click_id&ts=$(date +%s)"
}

# Create targets file
echo "Generating test targets..."
> targets.txt
for i in {1..1000}; do
    params=$(generate_params)
    echo "GET $API_URL/in?$params" >> targets.txt
done

echo -e "${GREEN}✅ Generated 1000 test URLs${NC}"
echo ""

# Test 1: Baseline (100 RPS)
echo "Test 1: Baseline Load (100 RPS)"
echo "--------------------------------"
vegeta attack -rate=100/s -duration=10s -targets=targets.txt | \
    vegeta report -type=text

echo ""

# Test 2: Target Load (1K RPS)
echo "Test 2: Target Load (1,000 RPS)"
echo "--------------------------------"
vegeta attack -rate=1000/s -duration=$DURATION -targets=targets.txt | \
    tee results_1k.bin | \
    vegeta report -type=text

echo ""

# Test 3: Stress Test (5K RPS)
echo "Test 3: Stress Test (5,000 RPS)"
echo "--------------------------------"
echo -e "${YELLOW}⚠️  This may stress your system${NC}"
read -p "Continue? (y/n) " -n 1 -r
echo ""
if [[ $REPLY =~ ^[Yy]$ ]]; then
    vegeta attack -rate=5000/s -duration=10s -targets=targets.txt | \
        tee results_5k.bin | \
        vegeta report -type=text
fi

echo ""

# Generate detailed report
if [ -f results_1k.bin ]; then
    echo "Detailed Report for 1K RPS Test"
    echo "--------------------------------"
    vegeta report -type=json results_1k.bin | jq '{
        requests: .requests,
        rate: .rate,
        duration: .duration,
        latencies: {
            mean: .latencies.mean,
            p50: .latencies."50th",
            p95: .latencies."95th",
            p99: .latencies."99th",
            max: .latencies.max
        },
        success_ratio: .success,
        errors: .errors
    }'
fi

echo ""

# Check data in ClickHouse
echo "Verifying data in ClickHouse..."
docker exec trellis-clickhouse clickhouse-client \
    --user trellis \
    --password trellis_dev \
    --query "SELECT count(*) as events_ingested FROM trellis.events WHERE event_time > now() - INTERVAL 5 MINUTE FORMAT TabSeparated" 2>/dev/null | \
    while read count; do
        echo -e "${GREEN}✅ Events ingested: $count${NC}"
    done

echo ""

# Clean up
rm -f targets.txt results_*.bin

echo "Load testing complete!"
echo ""
echo "Next steps:"
echo "  make query-events    # View ingested events"
echo "  make db-shell        # Query ClickHouse directly"
echo "  make status          # Check service health"
