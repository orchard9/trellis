# Trellis Campaigns Architecture

## System Overview

Trellis Campaigns is designed as a modern, organization-aware React frontend that provides an intuitive interface for managing traffic attribution campaigns, analyzing performance, and discovering patterns in historical data.

```
┌─────────────────────────────────────────────────────────────────┐
│                     CAMPAIGNS ARCHITECTURE                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│    ┌──────────┐         ┌──────────┐         ┌──────────┐      │
│    │  React   │────────▶│   API    │────────▶│ Backend  │      │
│    │   App    │         │ Gateway  │         │ Services │      │
│    │  (Vite)  │         │          │         │          │      │
│    └──────────┘         └──────────┘         └──────────┘      │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│                    FRONTEND ARCHITECTURE                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────┐ │
│  │   State Mgmt    │    │   UI Components │    │  Data Layer │ │
│  │  (Zustand)      │    │   (Tailwind)    │    │ (React Query│ │
│  │                 │    │                 │    │   + Axios)  │ │
│  └─────────────────┘    └─────────────────┘    └─────────────┘ │
└─────────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│                     BACKEND INTEGRATION                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────┐         ┌─────────────────┐                │
│  │   Warehouse     │         │    Ingress      │                │
│  │     API         │         │     API         │                │
│  │  (Analytics)    │         │  (Campaign Mgmt)│                │
│  └─────────────────┘         └─────────────────┘                │
└─────────────────────────────────────────────────────────────────┘
```

## Component Architecture

### 1. Frontend Framework Stack

**React 18 + TypeScript + Vite**
```tsx
// Modern React with hooks and TypeScript for type safety
interface CampaignData {
  organization_id: string;
  campaign_id: string;
  name: string;
  status: 'active' | 'paused' | 'archived';
  rules: CampaignRule[];
  performance: CampaignMetrics;
}

const CampaignDashboard: React.FC = () => {
  const { organization } = useAuth();
  const { data: campaigns, isLoading } = useCampaigns(organization.id);
  
  if (isLoading) return <LoadingSpinner />;
  
  return (
    <DashboardLayout>
      <CampaignGrid campaigns={campaigns} />
      <PerformanceCharts campaigns={campaigns} />
    </DashboardLayout>
  );
};
```

**State Management with Zustand**
```tsx
// Lightweight, organization-aware state management
interface AppState {
  organization: Organization | null;
  campaigns: Campaign[];
  analytics: AnalyticsData | null;
  
  // Actions
  setOrganization: (org: Organization) => void;
  updateCampaign: (id: string, updates: Partial<Campaign>) => void;
  refreshAnalytics: (timeRange: TimeRange) => Promise<void>;
}

export const useAppStore = create<AppState>((set, get) => ({
  organization: null,
  campaigns: [],
  analytics: null,
  
  setOrganization: (org) => set({ organization: org }),
  
  updateCampaign: (id, updates) => set((state) => ({
    campaigns: state.campaigns.map(c => 
      c.id === id ? { ...c, ...updates } : c
    )
  })),
  
  refreshAnalytics: async (timeRange) => {
    const org = get().organization;
    if (!org) return;
    
    const analytics = await analyticsAPI.getMetrics(org.id, timeRange);
    set({ analytics });
  },
}));
```

### 2. UI Component Library

**Tailwind CSS + Headless UI**
```tsx
// Consistent, accessible UI components
interface ButtonProps {
  variant: 'primary' | 'secondary' | 'outline' | 'ghost';
  size: 'sm' | 'md' | 'lg';
  loading?: boolean;
  children: React.ReactNode;
  onClick?: () => void;
}

const Button: React.FC<ButtonProps> = ({ 
  variant, 
  size, 
  loading, 
  children, 
  onClick 
}) => {
  const baseClasses = 'btn focus:outline-none focus:ring-2 focus:ring-offset-2';
  const variantClasses = {
    primary: 'btn-primary',
    secondary: 'btn-secondary', 
    outline: 'btn-outline',
    ghost: 'btn-ghost'
  };
  
  return (
    <button
      className={cn(baseClasses, variantClasses[variant], {
        'opacity-50 cursor-not-allowed': loading
      })}
      disabled={loading}
      onClick={onClick}
    >
      {loading && <Spinner className="mr-2 h-4 w-4" />}
      {children}
    </button>
  );
};
```

**Component Architecture**
```tsx
// Modular component structure
src/
├── components/
│   ├── ui/              # Reusable UI primitives
│   │   ├── Button.tsx
│   │   ├── Input.tsx
│   │   ├── Card.tsx
│   │   └── Modal.tsx
│   ├── layout/          # Layout components
│   │   ├── Layout.tsx
│   │   ├── Sidebar.tsx
│   │   └── Header.tsx
│   ├── campaigns/       # Campaign-specific components
│   │   ├── CampaignList.tsx
│   │   ├── CampaignForm.tsx
│   │   └── CampaignMetrics.tsx
│   └── analytics/       # Analytics components
│       ├── TrafficChart.tsx
│       ├── ConversionFunnel.tsx
│       └── AttributionModel.tsx
```

### 3. Data Layer Architecture

**React Query for Server State**
```tsx
// Efficient API data management with caching
export const useCampaigns = (orgId: string) => {
  return useQuery({
    queryKey: ['campaigns', orgId],
    queryFn: () => api.campaigns.list(orgId),
    staleTime: 5 * 60 * 1000, // 5 minutes
    cacheTime: 30 * 60 * 1000, // 30 minutes
    retry: 3,
    retryDelay: attemptIndex => Math.min(1000 * 2 ** attemptIndex, 30000),
  });
};

export const useAnalytics = (orgId: string, timeRange: TimeRange) => {
  return useQuery({
    queryKey: ['analytics', orgId, timeRange],
    queryFn: () => api.analytics.getMetrics(orgId, timeRange),
    staleTime: 2 * 60 * 1000, // 2 minutes for fresh data
    enabled: !!orgId && !!timeRange,
  });
};

// Mutations for data updates
export const useCreateCampaign = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (campaign: CreateCampaignRequest) => 
      api.campaigns.create(campaign),
    onSuccess: (newCampaign) => {
      // Invalidate and refetch campaigns list
      queryClient.invalidateQueries(['campaigns', newCampaign.organization_id]);
    },
  });
};
```

**API Client with Organization Context**
```tsx
// Centralized API client with automatic organization scoping
class APIClient {
  private baseURL: string;
  private organizationId: string | null = null;
  
  constructor(baseURL: string) {
    this.baseURL = baseURL;
  }
  
  setOrganization(orgId: string) {
    this.organizationId = orgId;
  }
  
  private async request<T>(
    endpoint: string, 
    options: RequestInit = {}
  ): Promise<T> {
    const url = `${this.baseURL}${endpoint}`;
    const token = auth.getToken();
    
    const response = await fetch(url, {
      ...options,
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json',
        ...options.headers,
      },
    });
    
    if (!response.ok) {
      throw new APIError(response.status, await response.text());
    }
    
    return response.json();
  }
  
  // Campaign Management
  campaigns = {
    list: () => this.request<Campaign[]>('/api/v1/campaigns'),
    
    create: (campaign: CreateCampaignRequest) => 
      this.request<Campaign>('/api/v1/campaigns', {
        method: 'POST',
        body: JSON.stringify(campaign),
      }),
      
    update: (id: string, updates: UpdateCampaignRequest) =>
      this.request<Campaign>(`/api/v1/campaigns/${id}`, {
        method: 'PUT', 
        body: JSON.stringify(updates),
      }),
  };
  
  // Analytics
  analytics = {
    getMetrics: (timeRange: TimeRange) =>
      this.request<AnalyticsData>(`/api/v1/analytics/traffic?${timeRange.toQuery()}`),
      
    getCampaignPerformance: (campaignId: string, timeRange: TimeRange) =>
      this.request<CampaignMetrics>(`/api/v1/analytics/campaigns/${campaignId}?${timeRange.toQuery()}`),
  };
}
```

### 4. Routing and Navigation

**React Router v6 with Organization Context**
```tsx
// Organization-aware routing
const AppRouter: React.FC = () => {
  const { organization, isLoading } = useAuth();
  
  if (isLoading) {
    return <LoadingScreen />;
  }
  
  if (!organization) {
    return <Navigate to="/login" replace />;
  }
  
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Layout />}>
          <Route index element={<Navigate to="/dashboard" replace />} />
          <Route path="/dashboard" element={<Dashboard />} />
          <Route path="/campaigns" element={<CampaignList />} />
          <Route path="/campaigns/:id" element={<CampaignDetail />} />
          <Route path="/campaigns/new" element={<CreateCampaign />} />
          <Route path="/analytics" element={<Analytics />} />
          <Route path="/patterns" element={<PatternDiscovery />} />
          <Route path="/settings" element={<Settings />} />
        </Route>
        <Route path="/login" element={<LoginPage />} />
        <Route path="*" element={<NotFoundPage />} />
      </Routes>
    </BrowserRouter>
  );
};
```

### 5. Data Polling

**Periodic Updates for Dashboard Data**
```tsx
// Periodic analytics updates
export const useAnalyticsPolling = (orgId: string, interval = 30000) => {
  const [metrics, setMetrics] = useState<RealtimeMetrics | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  
  const fetchMetrics = useCallback(async () => {
    setIsLoading(true);
    try {
      const response = await api.analytics.getCurrentMetrics(orgId);
      setMetrics(response.data);
    } catch (error) {
      console.error('Failed to fetch metrics:', error);
    } finally {
      setIsLoading(false);
    }
  }, [orgId]);
  
  useEffect(() => {
    // Initial fetch
    fetchMetrics();
    
    // Set up polling
    const pollInterval = setInterval(fetchMetrics, interval);
    
    return () => clearInterval(pollInterval);
  }, [fetchMetrics, interval]);
  
  return { metrics, isLoading, refresh: fetchMetrics };
};
```

### 6. Performance Optimizations

**Code Splitting and Lazy Loading**
```tsx
// Route-based code splitting
const Dashboard = lazy(() => import('../pages/Dashboard'));
const Analytics = lazy(() => import('../pages/Analytics')); 
const CampaignDetail = lazy(() => import('../pages/CampaignDetail'));

const App: React.FC = () => (
  <Suspense fallback={<LoadingSpinner />}>
    <AppRouter />
  </Suspense>
);
```

**Virtual Scrolling for Large Lists**
```tsx
// Efficient rendering of large campaign lists
import { FixedSizeList as List } from 'react-window';

const CampaignList: React.FC<{ campaigns: Campaign[] }> = ({ campaigns }) => {
  const CampaignRow = ({ index, style }) => (
    <div style={style}>
      <CampaignCard campaign={campaigns[index]} />
    </div>
  );
  
  return (
    <List
      height={600}
      itemCount={campaigns.length}
      itemSize={120}
      className="campaign-list"
    >
      {CampaignRow}
    </List>
  );
};
```

**Memoization for Expensive Computations**
```tsx
// Optimize chart rendering
const TrafficChart: React.FC<{ data: ChartData }> = ({ data }) => {
  const chartConfig = useMemo(() => ({
    responsive: true,
    plugins: {
      legend: { position: 'top' },
      title: { display: true, text: 'Traffic Overview' },
    },
  }), []);
  
  const processedData = useMemo(() => 
    processChartData(data), [data]
  );
  
  return <Line data={processedData} options={chartConfig} />;
};
```

## Security Architecture

### 1. Authentication Flow
```tsx
// Warden-based authentication with JWT tokens
export const useAuth = () => {
  const [user, setUser] = useState<User | null>(null);
  const [organization, setOrganization] = useState<Organization | null>(null);
  
  const login = async (credentials: LoginCredentials) => {
    // Authenticate with Warden to get JWT token
    const response = await wardenAuth.login(credentials);
    
    // Store Warden JWT and organization context
    localStorage.setItem('warden_token', response.jwt);
    setUser(response.user);
    setOrganization(response.organization);
    
    // Configure API client with Warden token
    api.setAuthToken(response.jwt);
    api.setOrganization(response.organization.id);
  };
  
  const switchOrganization = async (orgId: string) => {
    // Warden handles multi-org users
    const newContext = await wardenAuth.switchOrganization(orgId);
    setOrganization(newContext.organization);
    api.setOrganization(newContext.organization.id);
  };
  
  const logout = () => {
    localStorage.removeItem('warden_token');
    setUser(null);
    setOrganization(null);
    api.clearAuth();
  };
  
  return { user, organization, login, logout, switchOrganization };
};
```

### 2. Route Protection
```tsx
// Protect routes based on organization membership
const ProtectedRoute: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { organization, isLoading } = useAuth();
  
  if (isLoading) return <LoadingSpinner />;
  if (!organization) return <Navigate to="/login" />;
  
  return <>{children}</>;
};
```

## Performance Targets

### Frontend Performance
- **Initial load**: <3s (First Contentful Paint)
- **Navigation**: <500ms (Route transitions)
- **API calls**: <2s (Dashboard data loading)
- **Chart rendering**: <1s (Complex analytics visualizations)

### User Experience
- **Responsive design**: Mobile-first, works on all screen sizes
- **Accessibility**: WCAG 2.1 AA compliance
- **Offline support**: Service worker for basic functionality
- **Progressive loading**: Skeleton screens and incremental data loading

This architecture ensures Trellis Campaigns provides a modern, performant, and organization-aware frontend for managing traffic attribution campaigns and analytics.