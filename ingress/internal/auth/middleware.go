package auth

import (
	"context"
	"net/http"
	"strings"

	"log/slog"

	wardenv1 "github.com/orchard9/warden/api/gen/go/warden/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// OrganizationContext holds organization information for the request
type OrganizationContext struct {
	OrganizationID   string
	OrganizationSlug string
	AccountID        string
	Role             string
	Permissions      []string
}

// ContextKey is used for storing organization context in request context
type ContextKey string

const (
	OrganizationContextKey ContextKey = "organization_context"
)

// WardenClient wraps the Warden gRPC client
type WardenClient struct {
	authClient   wardenv1.AuthServiceClient
	orgClient    wardenv1.OrganizationServiceClient
	conn         *grpc.ClientConn
}

// NewWardenClient creates a new Warden client
func NewWardenClient(wardenAddr string) (*WardenClient, error) {
	conn, err := grpc.Dial(wardenAddr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	return &WardenClient{
		authClient: wardenv1.NewAuthServiceClient(conn),
		orgClient:  wardenv1.NewOrganizationServiceClient(conn),
		conn:       conn,
	}, nil
}

// Close closes the Warden client connection
func (w *WardenClient) Close() error {
	return w.conn.Close()
}

// AuthenticationMiddleware validates API keys and extracts organization context
func (w *WardenClient) AuthenticationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(wr http.ResponseWriter, r *http.Request) {
		// Extract API key from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(wr, "Missing Authorization header", http.StatusUnauthorized)
			return
		}

		// Validate Bearer token format
		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(wr, "Invalid Authorization header format", http.StatusUnauthorized)
			return
		}

		apiKey := strings.TrimPrefix(authHeader, "Bearer ")
		if !strings.HasPrefix(apiKey, "wdn_") {
			http.Error(wr, "Invalid API key format", http.StatusUnauthorized)
			return
		}

		// Validate API key with Warden
		ctx := r.Context()
		orgCtx, err := w.validateAPIKey(ctx, apiKey)
		if err != nil {
			slog.Error("API key validation failed", "error", err)
			http.Error(wr, "Invalid API key", http.StatusUnauthorized)
			return
		}

		// Add organization context to request
		ctx = context.WithValue(ctx, OrganizationContextKey, orgCtx)
		next.ServeHTTP(wr, r.WithContext(ctx))
	})
}

// validateAPIKey validates the API key with Warden and returns organization context
func (w *WardenClient) validateAPIKey(ctx context.Context, apiKey string) (*OrganizationContext, error) {
	// Create gRPC context with API key
	grpcCtx := metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+apiKey)

	// Validate API key and get account information
	validateResp, err := w.authClient.ValidateApiKey(grpcCtx, &wardenv1.ValidateApiKeyRequest{
		ApiKey: apiKey,
	})
	if err != nil {
		return nil, err
	}

	// Get organization information for the account
	orgResp, err := w.orgClient.GetAccountOrganizations(grpcCtx, &wardenv1.GetAccountOrganizationsRequest{
		AccountId: validateResp.AccountId,
	})
	if err != nil {
		return nil, err
	}

	// For now, use the first organization (in production, this might be determined by subdomain or API key scope)
	if len(orgResp.Organizations) == 0 {
		return nil, err
	}

	org := orgResp.Organizations[0]
	membership := org.Membership

	return &OrganizationContext{
		OrganizationID:   org.Organization.Id,
		OrganizationSlug: org.Organization.Slug,
		AccountID:        validateResp.AccountId,
		Role:             membership.Role,
		Permissions:      membership.Permissions,
	}, nil
}

// GetOrganizationContext extracts organization context from request context
func GetOrganizationContext(ctx context.Context) (*OrganizationContext, bool) {
	orgCtx, ok := ctx.Value(OrganizationContextKey).(*OrganizationContext)
	return orgCtx, ok
}

// RequirePermission creates middleware that checks for specific permissions
func (w *WardenClient) RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(wr http.ResponseWriter, r *http.Request) {
			orgCtx, ok := GetOrganizationContext(r.Context())
			if !ok {
				http.Error(wr, "Organization context not found", http.StatusInternalServerError)
				return
			}

			// Check if user has required permission
			hasPermission := false
			for _, perm := range orgCtx.Permissions {
				if perm == permission {
					hasPermission = true
					break
				}
			}

			if !hasPermission {
				http.Error(wr, "Insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(wr, r)
		})
	}
}

// RequireRole creates middleware that checks for specific roles
func (w *WardenClient) RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(wr http.ResponseWriter, r *http.Request) {
			orgCtx, ok := GetOrganizationContext(r.Context())
			if !ok {
				http.Error(wr, "Organization context not found", http.StatusInternalServerError)
				return
			}

			// Check if user has required role
			hasRole := false
			for _, role := range roles {
				if orgCtx.Role == role {
					hasRole = true
					break
				}
			}

			if !hasRole {
				http.Error(wr, "Insufficient role", http.StatusForbidden)
				return
			}

			next.ServeHTTP(wr, r)
		})
	}
}