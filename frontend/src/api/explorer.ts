import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';

const API_PREFIX = '/api/v1';

export interface ExplorerItem {
    id: string;
    folderId?: string;
    itemType: 'query' | 'workbook' | 'folder' | 'report';
    itemId?: string;
    name: string;
    position: number;
    [key: string]: unknown;
}

export interface ExplorerFolder {
    id: string;
    name: string;
    description?: string;
    parentId?: string | null;
    isCore?: boolean;
    ownerUserId?: string;
    items: ExplorerItem[];
    createdAt?: string;
    updatedAt?: string;
    [key: string]: unknown;
}

// Helper to make requests (basic version, usually shared)
async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
    const response = await fetch(path, options);
    if (!response.ok) {
        throw new Error(`API Error: ${response.statusText}`);
    }
    return response.json() as Promise<T>;
}

// --- Folders ---

export const fetchFolders = async (): Promise<ExplorerFolder[]> => {
    return request<ExplorerFolder[]>(`${API_PREFIX}/folders`);
};

export const useFolders = () =>
    useQuery({
        queryKey: ['explorer', 'folders'],
        queryFn: fetchFolders,
    });

export const useCreateFolder = () => {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async ({ name, description, parentId }: { name: string; description?: string; parentId?: string }) => {
            return request<ExplorerFolder>(`${API_PREFIX}/folders`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ name, description, parentId }),
            });
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['explorer', 'folders'] });
        },
    });
};

export const useUpdateFolder = () => {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async ({ id, name, description }: { id: string; name: string; description?: string }) => {
            return request<ExplorerFolder>(`${API_PREFIX}/folders/${id}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ name, description }),
            });
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['explorer', 'folders'] });
        },
    });
};

export const useDeleteFolder = () => {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async (id: string) => {
            return request<void>(`${API_PREFIX}/folders/${id}`, {
                method: 'DELETE',
            });
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['explorer', 'folders'] });
        },
    });
};

// --- Items ---

export const useAddItemToFolder = () => {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async ({ folderId, itemType, itemId }: { folderId: string; itemType: string; itemId: string }) => {
            return request(`${API_PREFIX}/folders/${folderId}/items`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ itemType, itemId }),
            });
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['explorer', 'folders'] });
        },
    });
};

