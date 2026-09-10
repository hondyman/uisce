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
  item_count?: number;
  report_count?: number; // Normalized from item_count
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
  // Standard identifier sent in JSON payload as template_id.
  // Note: Backend also accepts report_id as a backwards-compatible alias.
  templateId: string;
}

export interface RemoveReportFromFolderInput {
  folderId: string;
  templateId: string;
}

export interface FolderTreeNode {
  folder: ReportFolder;
  children: FolderTreeNode[];
  depth: number;
}

export function buildFolderTree(folders: ReportFolder[]): FolderTreeNode[] {
  const map = new Map<string, FolderTreeNode>();
  const roots: FolderTreeNode[] = [];

  const normalized = folders.map(f => ({
    ...f,
    report_count: f.item_count ?? f.report_count ?? 0,
  }));

  for (const folder of normalized) {
    map.set(folder.id, {
      folder,
      children: [],
      depth: 0,
    });
  }

  for (const folder of normalized) {
    const node = map.get(folder.id)!;
    if (folder.parent_id && map.has(folder.parent_id)) {
      const parent = map.get(folder.parent_id)!;
      parent.children.push(node);
    } else {
      roots.push(node);
    }
  }

  function updateDepths(node: FolderTreeNode, currentDepth: number) {
    node.depth = currentDepth;
    for (const child of node.children) {
      updateDepths(child, currentDepth + 1);
    }
  }
  for (const root of roots) {
    updateDepths(root, 0);
  }

  return roots;
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
