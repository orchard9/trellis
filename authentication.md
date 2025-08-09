# Trellis Authentication Architecture

## Overview

Trellis uses Warden for organization-aware authentication across all services. This document explains the authentication flows, token types, and security model.

## Authentication Types

### 1. Service Authentication (API Keys)
Used by backend services and automated systems to authenticate with Trellis APIs.

```
Format: Authorization: Bearer wdn_sk_live_a1b2c3d4e5f6
Prefix: wdn_sk_ (service key)
Usage: Backend-to-backend communication
Scope: Organization-specific operations
```

### 2. User Authentication (JWT Tokens)
Used by human users accessing the Campaigns frontend.

```
Format: Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
Type: JWT (JSON Web Token)
Usage: Frontend user sessions
Contains: User ID, Organization ID, permissions
```

## Authentication Flows

### Service-to-Service Authentication Flow
```
┌─────────────────────────────────────────────────────────────────────────┐
│                   SERVICE AUTHENTICATION FLOW                           │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────┐         ┌─────────────┐         ┌─────────────┐      │
│  │   Client    │────1───▶│   Ingress   │────2───▶│   Warden    │      │
│  │  (w/ API    │         │   Service   │         │   Service   │      │
│  │    Key)     │◀───4────│             │◀───3────│             │      │
│  └─────────────┘         └─────────────┘         └─────────────┘      │
│                                                                         │
│  1. Request with API key: Authorization: Bearer wdn_sk_live_xxx        │
│  2. Validate API key via gRPC                                          │
│  3. Return organization context + permissions                          │
│  4. Process request with organization scope                            │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

**Example: Traffic Ingestion**
```bash
# 1. Client sends traffic with API key
curl -H "Authorization: Bearer wdn_sk_live_a1b2c3d4e5f6" \
     "https://ingress.trellis.com/in?click_id=test123&source=google"

# 2. Ingress extracts API key and validates with Warden
orgCtx, err := wardenClient.ValidateAPIKey(ctx, &warden.ValidateAPIKeyRequest{
    ApiKey: "wdn_sk_live_a1b2c3d4e5f6",
})

# 3. Warden returns organization context
{
    "organization_id": "org_1234567890",
    "organization_name": "Acme Corp",
    "permissions": ["traffic:write", "campaigns:read"],
    "rate_limits": {
        "requests_per_second": 1000,
        "requests_per_day": 10000000
    }
}

# 4. Ingress processes request with organization scope
event.OrganizationID = orgCtx.OrganizationId
```

### User Authentication Flow
```
┌─────────────────────────────────────────────────────────────────────────┐
│                      USER AUTHENTICATION FLOW                           │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────┐         ┌─────────────┐         ┌─────────────┐      │
│  │   Browser   │────1───▶│  Campaigns  │────2───▶│   Warden    │      │
│  │             │         │   Frontend  │         │   Service   │      │
│  │             │◀───4────│             │◀───3────│             │      │
│  └─────────────┘         └─────────────┘         └─────────────┘      │
│         │                                                     │         │
│         │                      5. Store JWT                  │         │
│         └─────────────────────────────────────────────────────┘         │
│                                                                         │
│  1. Login with email/password                                          │
│  2. Authenticate via Warden gRPC                                       │
│  3. Return JWT with user + org context                                 │
│  4. Redirect to dashboard                                              │
│  5. Store JWT in localStorage                                          │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

**JWT Token Structure**
```json
{
  "header": {
    "alg": "HS256",
    "typ": "JWT"
  },
  "payload": {
    "sub": "user_1234567890",          // User ID
    "org": "org_0987654321",           // Organization ID
    "name": "John Doe",
    "email": "john@example.com",
    "permissions": [
      "campaigns:read",
      "campaigns:write",
      "analytics:read"
    ],
    "iat": 1609459200,                 // Issued at
    "exp": 1609545600                  // Expires at (24 hours)
  },
  "signature": "..."
}
```

### Organization Context Propagation
```
┌─────────────────────────────────────────────────────────────────────────┐
│                  ORGANIZATION CONTEXT PROPAGATION                      │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────┐         ┌─────────────┐         ┌─────────────┐      │
│  │  Campaigns  │────1───▶│  Warehouse  │────2───▶│ ClickHouse  │      │
│  │  Frontend   │         │    API      │         │  Database   │      │
│  │ (JWT Token) │◀───4────│             │◀───3────│             │      │
│  └─────────────┘         └─────────────┘         └─────────────┘      │
│                                                                         │
│  1. API request with JWT: Authorization: Bearer eyJ...                 │
│  2. Query with org filter: WHERE organization_id = 'org_0987654321'    │
│  3. Return org-scoped data only                                        │
│  4. Response contains only authorized organization's data              │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

## Implementation Examples

### 1. Extracting Organization Context (Go)
```go
// Middleware for API key authentication
func (a *AuthMiddleware) ValidateAPIKey(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Extract API key from header
        apiKey := extractBearerToken(r.Header.Get("Authorization"))
        if !strings.HasPrefix(apiKey, "wdn_sk_") {
            http.Error(w, "Invalid API key format", http.StatusUnauthorized)
            return
        }
        
        // Validate with Warden
        resp, err := a.wardenClient.ValidateAPIKey(r.Context(), &warden.ValidateAPIKeyRequest{
            ApiKey: apiKey,
        })
        if err != nil {
            http.Error(w, "Authentication failed", http.StatusUnauthorized)
            return
        }
        
        // Add organization context to request
        ctx := context.WithValue(r.Context(), "org_context", &OrgContext{
            OrganizationID: resp.OrganizationId,
            Permissions:    resp.Permissions,
            RateLimits:     resp.RateLimits,
        })
        
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// Middleware for JWT authentication
func (a *AuthMiddleware) ValidateJWT(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Extract JWT from header
        tokenString := extractBearerToken(r.Header.Get("Authorization"))
        
        // Parse and validate JWT
        token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
            return a.jwtSecret, nil
        })
        
        if err != nil || !token.Valid {
            http.Error(w, "Invalid token", http.StatusUnauthorized)
            return
        }
        
        // Extract claims
        claims := token.Claims.(jwt.MapClaims)
        
        // Add user context to request
        ctx := context.WithValue(r.Context(), "user_context", &UserContext{
            UserID:         claims["sub"].(string),
            OrganizationID: claims["org"].(string),
            Permissions:    claims["permissions"].([]string),
        })
        
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### 2. Frontend Authentication (React/TypeScript)
```tsx
// Authentication hook
export const useAuth = () => {
    const [user, setUser] = useState<User | null>(null);
    const [organization, setOrganization] = useState<Organization | null>(null);
    const [loading, setLoading] = useState(true);
    
    // Login function
    const login = async (email: string, password: string) => {
        try {
            // Authenticate with Warden
            const response = await fetch('/api/auth/login', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ email, password }),
            });
            
            if (!response.ok) throw new Error('Login failed');
            
            const { token, user, organization } = await response.json();
            
            // Store JWT
            localStorage.setItem('warden_token', token);
            
            // Update state
            setUser(user);
            setOrganization(organization);
            
            // Configure API client
            apiClient.setAuthToken(token);
            apiClient.setOrganization(organization.id);
            
            return { success: true };
        } catch (error) {
            return { success: false, error: error.message };
        }
    };
    
    // Check authentication on mount
    useEffect(() => {
        const token = localStorage.getItem('warden_token');
        if (token) {
            // Validate token and restore session
            validateToken(token).then(({ user, organization }) => {
                setUser(user);
                setOrganization(organization);
                apiClient.setAuthToken(token);
                apiClient.setOrganization(organization.id);
            }).finally(() => {
                setLoading(false);
            });
        } else {
            setLoading(false);
        }
    }, []);
    
    return { user, organization, login, logout, loading };
};
```

### 3. Organization Switching
```tsx
// For users with access to multiple organizations
export const useOrganizationSwitcher = () => {
    const { user } = useAuth();
    const [organizations, setOrganizations] = useState<Organization[]>([]);
    const [currentOrg, setCurrentOrg] = useState<Organization | null>(null);
    
    // Fetch available organizations
    useEffect(() => {
        if (user) {
            fetchUserOrganizations().then(setOrganizations);
        }
    }, [user]);
    
    // Switch organization
    const switchOrganization = async (orgId: string) => {
        try {
            // Request new JWT for different organization
            const response = await fetch('/api/auth/switch-org', {
                method: 'POST',
                headers: {
                    'Authorization': `Bearer ${localStorage.getItem('warden_token')}`,
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ organization_id: orgId }),
            });
            
            if (!response.ok) throw new Error('Switch failed');
            
            const { token, organization } = await response.json();
            
            // Update token and context
            localStorage.setItem('warden_token', token);
            setCurrentOrg(organization);
            
            // Refresh all data with new context
            window.location.reload();
        } catch (error) {
            console.error('Failed to switch organization:', error);
        }
    };
    
    return { organizations, currentOrg, switchOrganization };
};
```

## Security Best Practices

### 1. API Key Security
- **Never expose API keys in client-side code**
- **Rotate keys regularly** (recommended: every 90 days)
- **Use environment-specific keys** (dev, staging, prod)
- **Implement key scoping** (read-only vs read-write)

### 2. JWT Security
- **Short expiration times** (24 hours max)
- **Refresh token rotation** for extended sessions
- **Secure storage** (httpOnly cookies preferred over localStorage)
- **Token revocation** support for logout/security events

### 3. Organization Isolation
- **Always validate organization context** before data access
- **Never allow cross-organization queries**
- **Log all authentication attempts** with organization context
- **Implement rate limiting** per organization

### 4. Network Security
- **Always use HTTPS** for all API communication
- **Implement CORS** properly for frontend origins
- **Use secure gRPC** connections to Warden
- **Monitor for suspicious authentication patterns**

## Troubleshooting

### Common Issues

1. **"Invalid API key format"**
   - Ensure key starts with `wdn_sk_`
   - Check for extra spaces or newlines
   - Verify key hasn't been revoked

2. **"Organization context required"**
   - API key might be invalid or expired
   - Warden service might be unreachable
   - Check network connectivity to Warden

3. **"JWT expired"**
   - Implement token refresh logic
   - Prompt user to re-authenticate
   - Clear localStorage and redirect to login

4. **"Permission denied"**
   - User/service lacks required permissions
   - Check permission array in token/context
   - Contact organization admin for access

This authentication architecture ensures complete organization isolation while providing flexible authentication options for both services and users.