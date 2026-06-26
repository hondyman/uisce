import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';

const API_PREFIX = '/api/v1';

export interface ExplorerItem {
    id: string;
    folderId?: string;
    itemType: 'query' | 'workbook' | 'folder';
    itemId?: string;
    name: string;
    position: number;
    [key: string]: unknown;
}

export interface ExplorerFolder {
    id: string;
    name: string;
    parentId?: string | null;
    items: ExplorerItem[];
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
    // Assuming /api/v1/folders returns a list of folders with nested items
    return request<ExplorerFolder[]>(`${API_PREFIX}/folders`);
};

export const useFolders = () =>
    useQuery({
        queryKey: ['explorer', 'folders'],
        queryFn: fetchFolders,
    });

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
