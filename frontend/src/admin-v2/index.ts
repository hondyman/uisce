// Admin UI v2 - Production-grade control plane
export { AdminRoutes } from "./AdminRoutes";
export { AdminLayout } from "./layout/AdminLayout";

// Pages
export { GlobalOpsDashboard } from "./pages/GlobalOpsDashboard";
export { TenantsPage } from "./pages/TenantsPage";
export { APIKeysPage } from "./pages/APIKeysPage";

// Components
export { Card } from "./components/Card";
export { Table } from "./components/Table";
export { Modal } from "./components/Modal";
export { LineChart, BarChart } from "./components/Charts";
export { Spinner, ErrorBanner, SuccessBanner, Skeleton } from "./components/Feedback";
export { CreateTenantModal } from "./components/CreateTenantModal";
export { CreateAPIKeyModal } from "./components/CreateAPIKeyModal";
export { HealthBadge, HealthComponents } from "./components/HealthBadge";
export { HeatmapChart } from "./components/HeatmapChart";
export { AlertList } from "./components/AlertList";
export { ErrorFingerprints } from "./components/ErrorFingerprints";

// Hooks
export {
  useTenants,
  useTenant,
  useCreateTenant,
  useUpdateTenant,
  useSuspendTenant,
  useUnsuspendTenant,
  useDeleteTenant,
} from "./hooks/useTenants";
export {
  useAPIKeys,
  useAPIKey,
  useAPIKeyUsage,
  useCreateAPIKey,
  useRevokeAPIKey,
  useRotateAPIKey,
} from "./hooks/useAPIKeys";
export {
  useTenantDailyUsage,
  useTenantEndpointUsage,
  useTenantRecentRequests,
  useGlobalUsage,
  useGlobalErrors,
  useGlobalLatency,
  useTopTenants,
  useTopEndpoints,
  useRecentErrors,
} from "./hooks/useUsage";
export {
  useAlerts,
  useAlert,
  useAlertEvents,
  useCreateAlert,
  useUpdateAlert,
  useDeleteAlert,
  useEvaluateAlerts,
  useTenantHealth,
  useEndpointHealthList,
  useEndpointHealth,
  useLatencyHeatmap,
  useRegionHeatmap,
  useTenantHeatmap,
  useEndpointHeatmap,
  useErrorFingerprints,
  useErrorFingerprintHistory,
} from "./hooks/useOps";

// Types
export type {
  Tenant,
  CreateTenantRequest,
  UpdateTenantRequest,
  APIKey,
  CreateAPIKeyRequest,
  APIKeyUsage,
  DailyUsage,
  LatencyPoint,
  ErrorPoint,
  UsagePoint,
  EndpointUsage,
  TopTenant,
  RecentError,
  ListResponse,
  SingleResponse,
  APIError,
  Alert,
  AlertEvent,
  TenantHealth,
  EndpointHealth,
  HeatmapSeriesPoint,
  HeatmapSeries,
  Heatmap,
  ErrorFingerprint,
  ErrorEvent,
  HealthStatus,
} from "./types";
export { getHealthStatus } from "./types";

// API
export { api, queryClient } from "./api";
