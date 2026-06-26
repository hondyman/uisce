/**
 * React Query Hooks for Semantic Reporting
 * 
 * Provides React Query hooks for the semantic reporting API.
 * Automatically handles tenant scope from context.
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useMemo } from 'react';
import SemanticReportingClient, {
  ReportDefinition,
  ReportExtension,
  ReportSchedule,
  CreateDefinitionRequest,
  CreateExtensionRequest,
  RenderReportRequest,
} from '../api/semanticReporting';
import { useTenant } from '../contexts/TenantContext';

// Query keys
const QUERY_KEYS = {
  definitions: 'report-definitions',
  definition: 'report-definition',
  extensions: 'report-extensions',
  extension: 'report-extension',
  instances: 'report-instances',
  instance: 'report-instance',
  schedules: 'report-schedules',
  schedule: 'report-schedule',
};

// Hook to get the reporting client
function useReportingClient(): SemanticReportingClient | null {
  const { tenant, datasource } = useTenant();

  return useMemo(() => {
    if (!tenant?.id || !datasource?.id) {
      return null;
    }
    const baseUrl = import.meta.env.VITE_API_URL || 'http://localhost:8080/api';
    return new SemanticReportingClient(baseUrl, tenant.id, datasource.id);
  }, [tenant?.id, datasource?.id]);
}

// ============================================================================
// REPORT DEFINITIONS HOOKS
// ============================================================================

export function useReportDefinitions(filters?: {
  category?: string;
  status?: string;
  is_core?: boolean;
}) {
  const client = useReportingClient();

  return useQuery({
    queryKey: [QUERY_KEYS.definitions, filters],
    queryFn: () => client!.listDefinitions(filters),
    enabled: !!client,
    staleTime: 5 * 60 * 1000, // 5 minutes
  });
}

export function useReportDefinition(id: string | undefined) {
  const client = useReportingClient();

  return useQuery({
    queryKey: [QUERY_KEYS.definition, id],
    queryFn: () => client!.getDefinition(id!),
    enabled: !!client && !!id,
  });
}

export function useCreateReportDefinition() {
  const client = useReportingClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (request: CreateDefinitionRequest) => client!.createDefinition(request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [QUERY_KEYS.definitions] });
    },
  });
}

export function useUpdateReportDefinition() {
  const client = useReportingClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, updates }: { id: string; updates: Partial<ReportDefinition> }) =>
      client!.updateDefinition(id, updates),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: [QUERY_KEYS.definitions] });
      queryClient.invalidateQueries({ queryKey: [QUERY_KEYS.definition, id] });
    },
  });
}

export function useDeleteReportDefinition() {
  const client = useReportingClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => client!.deleteDefinition(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [QUERY_KEYS.definitions] });
    },
  });
}

export function usePublishReportDefinition() {
  const client = useReportingClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => client!.publishDefinition(id),
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: [QUERY_KEYS.definitions] });
      queryClient.invalidateQueries({ queryKey: [QUERY_KEYS.definition, id] });
    },
  });
}

// ============================================================================
// REPORT EXTENSIONS HOOKS
// ============================================================================

export function useReportExtensions(baseReportId?: string) {
  const client = useReportingClient();

  return useQuery({
    queryKey: [QUERY_KEYS.extensions, baseReportId],
    queryFn: () => client!.listExtensions(baseReportId),
    enabled: !!client,
    staleTime: 5 * 60 * 1000,
  });
}

export function useReportExtension(id: string | undefined) {
  const client = useReportingClient();

  return useQuery({
    queryKey: [QUERY_KEYS.extension, id],
    queryFn: () => client!.getExtension(id!),
    enabled: !!client && !!id,
  });
}

export function useCreateReportExtension() {
  const client = useReportingClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (request: CreateExtensionRequest) => client!.createExtension(request),
    onSuccess: (_, request) => {
      queryClient.invalidateQueries({ queryKey: [QUERY_KEYS.extensions] });
      queryClient.invalidateQueries({ queryKey: [QUERY_KEYS.extensions, request.base_report_id] });
    },
  });
}

export function useUpdateReportExtension() {
  const client = useReportingClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, updates }: { id: string; updates: Partial<ReportExtension> }) =>
      client!.updateExtension(id, updates),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: [QUERY_KEYS.extensions] });
      queryClient.invalidateQueries({ queryKey: [QUERY_KEYS.extension, id] });
    },
  });
}

export function useDeleteReportExtension() {
  const client = useReportingClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => client!.deleteExtension(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [QUERY_KEYS.extensions] });
    },
  });
}

// ============================================================================
// REPORT RENDERING HOOKS
// ============================================================================

export function useRenderReport() {
  const client = useReportingClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (request: RenderReportRequest) => client!.renderReport(request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [QUERY_KEYS.instances] });
    },
  });
}

export function useRenderReportAsync() {
  const client = useReportingClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (request: RenderReportRequest) => client!.renderReportAsync(request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [QUERY_KEYS.instances] });
    },
  });
}

// ============================================================================
// REPORT INSTANCES HOOKS
// ============================================================================

export function useReportInstances(limit?: number) {
  const client = useReportingClient();

  return useQuery({
    queryKey: [QUERY_KEYS.instances, limit],
    queryFn: () => client!.listInstances(limit),
    enabled: !!client,
    staleTime: 30 * 1000, // 30 seconds
  });
}

export function useReportInstance(id: string | undefined) {
  const client = useReportingClient();

  return useQuery({
    queryKey: [QUERY_KEYS.instance, id],
    queryFn: () => client!.getInstance(id!),
    enabled: !!client && !!id,
    refetchInterval: (query) => {
      // Poll while generating
      const data = query.state.data;
      if (data?.status === 'pending' || data?.status === 'generating') {
        return 2000; // 2 seconds
      }
      return false;
    },
  });
}

export function useDownloadReport() {
  const client = useReportingClient();

  return useMutation({
    mutationFn: async (instanceId: string) => {
      const blob = await client!.downloadInstance(instanceId);
      return { blob, instanceId };
    },
    onSuccess: ({ blob, instanceId }) => {
      // Create download link
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `report-${instanceId}.${blob.type.includes('pdf') ? 'pdf' : 'html'}`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      window.URL.revokeObjectURL(url);
    },
  });
}

// ============================================================================
// REPORT SCHEDULES HOOKS
// ============================================================================

export function useReportSchedules() {
  const client = useReportingClient();

  return useQuery({
    queryKey: [QUERY_KEYS.schedules],
    queryFn: () => client!.listSchedules(),
    enabled: !!client,
    staleTime: 5 * 60 * 1000,
  });
}

export function useReportSchedule(id: string | undefined) {
  const client = useReportingClient();

  return useQuery({
    queryKey: [QUERY_KEYS.schedule, id],
    queryFn: () => client!.getSchedule(id!),
    enabled: !!client && !!id,
  });
}

export function useCreateReportSchedule() {
  const client = useReportingClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (schedule: Omit<ReportSchedule, 'id' | 'tenant_id' | 'tenant_tenant_instance_id'>) =>
      client!.createSchedule(schedule),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [QUERY_KEYS.schedules] });
    },
  });
}

export function useUpdateReportSchedule() {
  const client = useReportingClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, updates }: { id: string; updates: Partial<ReportSchedule> }) =>
      client!.updateSchedule(id, updates),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: [QUERY_KEYS.schedules] });
      queryClient.invalidateQueries({ queryKey: [QUERY_KEYS.schedule, id] });
    },
  });
}

export function useDeleteReportSchedule() {
  const client = useReportingClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => client!.deleteSchedule(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [QUERY_KEYS.schedules] });
    },
  });
}
