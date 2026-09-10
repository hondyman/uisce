import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import apiClient from '../utils/apiClient';

const API_PREFIX = '/api/v1/reports/folders';

export interface ReportFolder {
  id: string;
  name: string;
  parent_id?: string | null;
  tenant_id: string;
  user_id: string;
  created_at: string;
  updated_at: string;
  report_count?: number;
}

export interface CreateReportFolderInput {
  name: string;
  parent_id?: string | null;
}

export interface MoveReportFolderInput {
  folderId: string;
  parent_id?: string | null;
}

export interface RenameReportFolderInput {
  folderId: string;
  name: string;
}

export interface AddReportToFolderInput {
  folderId: string;
  templateId: string;
}

export interface RemoveReportFromFolderInput {
  folderId: string;
  templateId: string;
}

export const fetchReportFolders = async (): Promise<ReportFolder[]> => {
  const data = await apiClient<ReportFolder[]>(API_PREFIX);
  return Array.isArray(data) ? data : [];
};

export const useReportFolders = () =>
  useQuery({
    queryKey: ['reporting', 'folders'],
    queryFn: fetchReportFolders,
    staleTime: 30_000,
  });

export const useCreateReportFolder = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateReportFolderInput) =>
      apiClient<ReportFolder>(API_PREFIX, {
        method: 'POST',
        body: JSON.stringify(input),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['reporting', 'folders'] });
    },
  });
};

export const useRenameReportFolder = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ folderId, name }: RenameReportFolderInput) =>
      apiClient<ReportFolder>(`${API_PREFIX}/${folderId}`, {
        method: 'PUT',
        body: JSON.stringify({ name }),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['reporting', 'folders'] });
    },
  });
};

export const useMoveReportFolder = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ folderId, parent_id }: MoveReportFolderInput) =>
      apiClient<{ id: string; parent_id: string | null }>(`${API_PREFIX}/${folderId}/move`, {
        method: 'POST',
        body: JSON.stringify({ parent_id }),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['reporting', 'folders'] });
    },
  });
};

export const useDeleteReportFolder = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (folderId: string) =>
      apiClient<void>(`${API_PREFIX}/${folderId}`, {
        method: 'DELETE',
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['reporting', 'folders'] });
    },
  });
};

export const fetchFolderReportIDs = async (folderId: string): Promise<string[]> => {
  const data = await apiClient<string[]>(`${API_PREFIX}/${folderId}/items`);
  return Array.isArray(data) ? data : [];
};

export const useFolderReportIDs = (folderId: string | null | undefined) =>
  useQuery({
    queryKey: ['reporting', 'folders', folderId, 'items'],
    queryFn: () => fetchFolderReportIDs(folderId!),
    enabled: !!folderId,
    staleTime: 30_000,
  });

export const useAddReportToFolder = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ folderId, templateId }: AddReportToFolderInput) =>
      apiClient<{ folder_id: string; template_id: string }>(`${API_PREFIX}/${folderId}/items`, {
        method: 'POST',
        body: JSON.stringify({ template_id: templateId }),
      }),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ['reporting', 'folders'] });
      queryClient.invalidateQueries({ queryKey: ['reporting', 'folders', variables.folderId, 'items'] });
    },
  });
};

export const useRemoveReportFromFolder = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ folderId, templateId }: RemoveReportFromFolderInput) =>
      apiClient<void>(`${API_PREFIX}/${folderId}/items/${templateId}`, {
        method: 'DELETE',
      }),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ['reporting', 'folders'] });
      queryClient.invalidateQueries({ queryKey: ['reporting', 'folders', variables.folderId, 'items'] });
    },
  });
};
