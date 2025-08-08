module github.com/orchard9/trellis/ingress

go 1.21

require (
	// HTTP Framework
	github.com/go-chi/chi/v5 v5.0.12
	github.com/go-chi/cors v1.2.1
	github.com/go-chi/httprate v0.8.0
	
	// Database Drivers
	github.com/ClickHouse/clickhouse-go/v2 v2.23.0
	github.com/redis/go-redis/v9 v9.5.1
	
	// Google Cloud
	cloud.google.com/go/pubsub v1.37.0
	
	// Authentication & Authorization
	github.com/orchard9/warden/api/gen/go v0.1.0
	google.golang.org/grpc v1.60.0
	google.golang.org/protobuf v1.31.0
	
	// Caching for routing engine
	github.com/dgraph-io/ristretto v0.1.1
	
	// Utilities
	github.com/google/uuid v1.6.0
	github.com/joho/godotenv v1.5.1
	
	// Logging
	golang.org/x/exp v0.0.0-20240222234643-814bf88cf225 // for slog
)