# CLAUDE.md - Trellis Campaigns Service  

Written by world-class frontend architects with expertise in React performance optimization, organization-aware UX design, and real-time analytics dashboards.

project: trellis-campaigns
repo: github.com/orchard9/trellis/campaigns  
description: Organization-aware React frontend for traffic attribution campaign management with real-time analytics

## Where is this deployed
Dev: https://campaigns-dev.trellis.orchard9.com/
Production: https://campaigns.trellis.orchard9.com/

## Core Philosophy

**"Discover Patterns, Create Campaigns, Optimize Retroactively"** - Modern campaign management through intelligent pattern discovery. Organizations completely isolated. Real-time everything. Every insight immediately actionable.

## How to develop locally

1. Ensure prerequisites: Node.js 18+, npm/yarn, Vite, API services running (warehouse + ingress)
2. Clone repository and navigate to campaigns directory: `cd campaigns/`
3. Run `cp .env.example .env` and configure environment variables
4. Set up API endpoints in `.env`:
   - `VITE_WAREHOUSE_API_URL=http://localhost:8090`
   - `VITE_INGRESS_API_URL=http://localhost:8080`
   - `VITE_WARDEN_API_URL=http://localhost:21382`
5. Install dependencies: `npm install`
6. Start development server: `npm run dev`
7. Open browser to http://localhost:3000
8. Test with organization authentication via Warden API key

**Development Server Notes:**
- Port: 3000 (configurable via `VITE_PORT`)
- Hot reload: Instant updates with Vite HMR
- API proxying: All `/api/*` requests proxied to backend services
- Organization context: Automatically managed via authentication

**IMPORTANT: Code Quality Requirements**
Before submitting any code changes, ensure:
1. **Build passes**: `npm run build` must succeed without errors or warnings
2. **Type checking**: `npm run type-check` must pass with zero TypeScript errors  
3. **Linting passes**: `npm run lint` must pass with zero ESLint issues
4. **Tests pass**: `npm run test` must succeed with organization isolation tests
5. **Performance**: All dashboard loads <3s, navigation <500ms

## How to run tests

1. Unit tests: `npm run test`
2. Component tests: `npm run test:components` 
3. Integration tests: `npm run test:integration` (requires API services)
4. E2E tests: `npm run test:e2e` (requires full environment)
5. Performance tests: `npm run test:performance`
6. Type checking: `npm run type-check`
7. All checks: `npm run ci`

## How to manage campaigns

All campaign management requires organization authentication:

### Campaign Creation
```bash
# Access campaign creation wizard
http://localhost:3000/campaigns/new

# Required fields:
- Campaign name
- Traffic source patterns (source, medium, etc.)
- Destination URL
- Attribution rules
- Budget and targeting settings
```

### Retroactive Campaign Creation
```bash
# Pattern discovery interface  
http://localhost:3000/patterns

# Workflow:
1. AI discovers patterns in unattributed traffic
2. Review pattern performance and potential
3. Create campaign to capture historical + future matching traffic
4. Verify attribution accuracy
5. Monitor real-time performance
```

### Campaign Analytics
```bash
# Campaign performance dashboard
http://localhost:3000/campaigns/{campaign_id}/analytics

# Available views:
- Real-time metrics (last hour/day)
- Historical performance trends  
- Attribution journey analysis
- Fraud detection alerts
- A/B testing results
- ROI optimization suggestions
```

## How to debug frontend issues

### Common Issues and Solutions

**Issue: Organization context not loading**
- Check: Warden API authentication token
- Debug: Browser dev tools Network tab for auth failures
- Solution: Verify API key format and Warden service connectivity

**Issue: Slow dashboard loading (>3s)**
- Check: API response times and data payload sizes
- Debug: React DevTools Profiler for render bottlenecks
- Solution: Implement pagination, memoization, or data virtualization

**Issue: Dashboard data not updating**
- Check: API polling interval and response times
- Debug: Network tab for failed API requests
- Solution: Check API endpoint availability and authentication

**Issue: Charts not rendering or showing incorrect data**
- Check: Data format and API response structure  
- Debug: Console errors for Chart.js or data processing issues
- Solution: Verify data transformation logic and chart configuration

### Debug Commands
```bash
# Start with debug logging enabled
VITE_DEBUG=true npm run dev

# Type check without running
npm run type-check

# Build and analyze bundle size
npm run build && npm run analyze

# Test specific component
npm run test -- CampaignDashboard

# E2E test specific workflow  
npm run test:e2e -- campaign-creation

# Performance profiling
npm run dev --profile
```

## Performance Standards

### Frontend Performance Requirements
- **Initial page load**: <3s First Contentful Paint
- **Navigation between pages**: <500ms transition time
- **Dashboard data loading**: <2s from click to complete render
- **Chart rendering**: <1s for complex visualizations with 10K+ data points
- **Form interactions**: <100ms response time for all inputs

### User Experience Standards
- **Mobile responsiveness**: Full feature parity on mobile devices
- **Accessibility**: WCAG 2.1 AA compliance for all components
- **Offline capability**: Basic navigation and cached data access
- **Progressive loading**: Skeleton screens and incremental data loading
- **Error handling**: Graceful degradation with retry mechanisms

### Organization Isolation Standards
- **Data separation**: Zero possibility of seeing other organizations' data
- **URL structure**: All routes organization-scoped (/org/{org_id}/...)
- **Cache isolation**: Organization-specific cache keys and storage
- **API calls**: All requests include organization context automatically

## Frontend Architecture Patterns

### Organization Context Pattern
```tsx
// Every component that uses data must be organization-aware
export const CampaignDashboard: React.FC = () => {
  const { organization } = useAuth();
  
  // All queries automatically scoped to organization
  const { data: campaigns } = useCampaigns(organization.id);
  
  if (!organization) {
    return <Navigate to="/login" replace />;
  }
  
  return (
    <DashboardLayout>
      <CampaignGrid campaigns={campaigns} />
    </DashboardLayout>
  );
};
```

### API Integration Pattern  
```tsx
// API calls with automatic organization context and error handling
export const useCampaigns = (orgId: string) => {
  return useQuery({
    queryKey: ['campaigns', orgId],
    queryFn: () => api.campaigns.list(orgId),
    staleTime: 5 * 60 * 1000,
    retry: 3,
    onError: (error) => {
      if (error.status === 403) {
        // Redirect to login if organization access denied
        window.location.href = '/login';
      }
    }
  });
};
```

### Real-time Updates Pattern
```tsx
// Polling for campaign metrics updates
export const useMetricsPolling = (orgId: string, campaignId?: string) => {
  const [metrics, setMetrics] = useState<Metrics | null>(null);
  
  useEffect(() => {
    const fetchMetrics = async () => {
      const endpoint = campaignId 
        ? `/api/v1/campaigns/${campaignId}/metrics`
        : `/api/v1/analytics/current`;
      
      const response = await api.get(endpoint);
      setMetrics(response.data);
    };
    
    // Initial fetch
    fetchMetrics();
    
    // Poll every 30 seconds
    const interval = setInterval(fetchMetrics, 30000);
    
    return () => clearInterval(interval);
  }, [orgId, campaignId]);
  
  return metrics;
};
```

### Performance Optimization Pattern
```tsx
// Memoization and virtualization for large datasets
export const CampaignList: React.FC<{ campaigns: Campaign[] }> = ({ campaigns }) => {
  // Memoize expensive computations
  const sortedCampaigns = useMemo(() => 
    campaigns.sort((a, b) => b.performance.revenue - a.performance.revenue),
    [campaigns]
  );
  
  // Virtual scrolling for large lists
  const listItems = useMemo(() => 
    sortedCampaigns.map(campaign => ({
      id: campaign.id,
      component: <CampaignCard key={campaign.id} campaign={campaign} />
    })),
    [sortedCampaigns]
  );
  
  return (
    <VirtualizedList
      items={listItems}
      itemHeight={120}
      containerHeight={600}
    />
  );
};
```

## Testing Patterns

### Organization Isolation Testing
```tsx
describe('Campaign Dashboard Organization Isolation', () => {
  it('should only show campaigns for authenticated organization', async () => {
    // Setup two organizations with different campaigns
    const org1 = await setupTestOrg('org1', { campaigns: 5 });
    const org2 = await setupTestOrg('org2', { campaigns: 3 });
    
    // Login as org1
    await loginAs(org1.user);
    render(<CampaignDashboard />);
    
    // Should see org1 campaigns only
    await waitFor(() => {
      expect(screen.getAllByTestId('campaign-card')).toHaveLength(5);
    });
    
    // Verify no org2 campaign data is present
    expect(screen.queryByText(org2.campaigns[0].name)).not.toBeInTheDocument();
  });
});
```

### Performance Testing  
```tsx
describe('Dashboard Performance', () => {
  it('should load dashboard within performance budget', async () => {
    const perfObserver = new PerformanceObserver();
    const largeCampaignSet = generateCampaigns(1000);
    
    const startTime = performance.now();
    
    render(<CampaignDashboard campaigns={largeCampaignSet} />);
    
    await waitFor(() => {
      expect(screen.getByTestId('campaign-grid')).toBeInTheDocument();
    });
    
    const loadTime = performance.now() - startTime;
    expect(loadTime).toBeLessThan(2000); // 2 second budget
  });
});
```

### Real-time Updates Testing
```tsx
describe('Real-time Campaign Updates', () => {
  it('should update metrics on polling interval', async () => {
    const mockApi = jest.spyOn(api, 'get');
    mockApi.mockResolvedValueOnce({ 
      data: { clicks: 0, conversions: 0 } 
    });
    
    render(<CampaignMetrics campaignId="test-campaign" />);
    
    // Initial state
    expect(screen.getByTestId('clicks-count')).toHaveTextContent('0');
    
    // Mock updated data for next poll
    mockApi.mockResolvedValueOnce({ 
      data: { clicks: 1247, conversions: 38 } 
    });
    
    // Fast-forward polling interval
    act(() => {
      jest.advanceTimersByTime(30000);
    });
    
    // Should update UI after polling
    await waitFor(() => {
      expect(screen.getByTestId('clicks-count')).toHaveTextContent('1,247');
      expect(screen.getByTestId('conversions-count')).toHaveTextContent('38');
    });
  });
});
```

## Error Handling Standards

1. **NEVER SHOW OTHER ORGANIZATIONS' DATA**
   - If organization context is missing, redirect to login
   - If API returns unauthorized, clear session and redirect
   - Never render data without verifying organization ownership

2. **Graceful Degradation**
   - If API polling fails, show last known data with timestamp
   - If charts can't render, show tabular data
   - If images fail to load, show meaningful placeholders

3. **User-Friendly Errors**
   - Network errors: "Connection issue - retrying..."
   - Authorization errors: "Please log in to continue"
   - Data errors: "Unable to load campaign data - refresh page"

## Logging Standards

DEBUG = Component renders, state changes, API polling events
INFO = Successful API calls, navigation events, user interactions
WARN = Slow performance, API timeouts, polling failures
ERROR = Failed API calls, rendering errors, authentication failures  
CRITICAL = Organization data leakage attempts, security violations

All client-side logs should be sanitized and never include sensitive data.

## Key Operational Commands

- `npm run dev` - start development server with hot reload
- `npm run build` - create production build
- `npm run preview` - preview production build locally  
- `npm run test` - run test suite
- `npm run type-check` - TypeScript compilation check
- `npm run lint` - ESLint code quality check
- `npm run analyze` - bundle size analysis

## Component Architecture

### Page Components (Routes)
```tsx
src/pages/
├── Dashboard.tsx         # Main campaign overview
├── CampaignList.tsx     # All campaigns table/grid
├── CampaignDetail.tsx   # Individual campaign analytics
├── CreateCampaign.tsx   # Campaign creation wizard
├── PatternDiscovery.tsx # AI pattern discovery interface
├── Analytics.tsx        # Advanced analytics and reports
└── Settings.tsx         # Organization settings
```

### UI Component Library
```tsx  
src/components/ui/
├── Button.tsx           # Standard button variants
├── Input.tsx           # Form inputs with validation
├── Card.tsx            # Content containers
├── Modal.tsx           # Overlay dialogs
├── Table.tsx           # Data tables with sorting/filtering
├── Chart.tsx           # Chart.js wrapper components
└── VirtualizedList.tsx # Performance-optimized lists
```

### Business Logic Components
```tsx
src/components/campaigns/
├── CampaignCard.tsx     # Campaign summary cards
├── CampaignForm.tsx     # Campaign creation/editing
├── CampaignMetrics.tsx  # Real-time performance metrics
├── AttributionChart.tsx # Customer journey visualization
└── PatternSuggestion.tsx # AI-discovered patterns

src/components/analytics/
├── TrafficChart.tsx     # Traffic overview visualizations
├── ConversionFunnel.tsx # Funnel analysis
├── CohortAnalysis.tsx   # User retention analysis
└── AttributionModel.tsx # Multi-touch attribution
```

## State Management

### Global State (Zustand)
```tsx
interface AppState {
  // Authentication
  user: User | null;
  organization: Organization | null;
  
  // UI State
  sidebarOpen: boolean;
  currentView: 'dashboard' | 'campaigns' | 'analytics' | 'patterns';
  
  // Data Cache
  campaigns: Campaign[];
  realtimeMetrics: RealtimeMetrics | null;
  
  // Actions
  setOrganization: (org: Organization) => void;
  updateCampaign: (id: string, updates: Partial<Campaign>) => void;
  toggleSidebar: () => void;
}
```

### Server State (React Query)
- Campaign data with automatic refetching
- Analytics queries with intelligent caching
- Metrics updates via periodic API polling
- Pattern discovery with background updates

This architecture ensures Trellis Campaigns delivers a modern, performant, and organization-aware frontend experience for managing traffic attribution campaigns with real-time analytics and AI-powered insights.