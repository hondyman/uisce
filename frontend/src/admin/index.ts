// Admin UI Index - Export all components and configuration

// Layout
export { AdminLayout } from "./layout/AdminLayout";

// Pages
export { DashboardPage } from "./pages/DashboardPage";
export { TenantsPage } from "./pages/TenantsPage";
export { APIKeysPage } from "./pages/APIKeysPage";
export { UsageAnalyticsPage } from "./pages/UsageAnalyticsPage";

// Hooks
export {
  useTenants,
  useTenant,
  useCreateTenant,
  useUpdateTenant,
  useSuspendTenant,
  useAPIKeys,
  useAPIKeyUsage,
  useTenantDailyUsage,
  useTenantEndpointUsage,
} from "./hooks/useAdmin";

// Types
export type {
  Tenant,
  TenantCreateRequest,
  TenantUpdateRequest,
  APIKey,
  APIKeyUsage,
  DailyUsageStats,
  EndpointUsageStats,
  TenantUsageSummary,
  ListTenantsResponse,
  ListAPIKeysResponse,
} from "./types";

// Routes
export { adminRoutes } from "./routes";
