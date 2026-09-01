/**
 * useExplorerConversations — list, save, share, delete, pin, move folders.
 *
 * Wraps the api service with React Query so the rail re-renders
 * automatically when a conversation is added/updated/deleted.
 */

import { useCallback, useMemo } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  appendMessage,
  createConversation,
  createFolder,
  deleteConversation,
  deleteFolder,
  getConversation,
  listConversations,
  listFolders,
  moveConversationToFolder,
  renameFolder,
  togglePin,
  toggleShare,
  updateConversation,
} from '../services/explorerConversationsApi';
import type {
  Conversation,
  ConversationFolder,
  ConversationMessage,
  ConversationScope,
} from '../types/conversationTypes';

const CONV_KEY = ['data-explorer', 'conversations'] as const;
const FOLDER_KEY = ['data-explorer', 'folders'] as const;

export function useExplorerConversations() {
  const queryClient = useQueryClient();

  const conversationsQuery = useQuery({
    queryKey: CONV_KEY,
    queryFn: () => listConversations(),
  });

  const foldersQuery = useQuery({
    queryKey: FOLDER_KEY,
    queryFn: () => listFolders(),
  });

  const invalidate = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: CONV_KEY });
    queryClient.invalidateQueries({ queryKey: FOLDER_KEY });
  }, [queryClient]);

  const setQueriesData = useCallback(
    (conversations: Conversation[]) => {
      queryClient.setQueryData<Conversation[]>(CONV_KEY, conversations);
    },
    [queryClient]
  );

  const setFoldersData = useCallback(
    (folders: ConversationFolder[]) => {
      queryClient.setQueryData<ConversationFolder[]>(FOLDER_KEY, folders);
    },
    [queryClient]
  );

  const currentConversations = conversationsQuery.data ?? [];
  const currentFolders = foldersQuery.data ?? [];

  const createMutation = useMutation({
    mutationFn: (initial?: Partial<Conversation>) => createConversation(initial),
    onSuccess: (record) => {
      setQueriesData([record, ...currentConversations]);
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, patch }: { id: string; patch: Partial<Conversation> }) =>
      updateConversation(id, patch),
    onSuccess: (record) => {
      if (!record) return;
      setQueriesData(
        currentConversations.map((c) => (c.id === record.id ? record : c))
      );
    },
  });

  const removeMutation = useMutation({
    mutationFn: (id: string) => deleteConversation(id),
    onSuccess: (_void, id) => {
      setQueriesData(currentConversations.filter((c) => c.id !== id));
    },
  });

  const appendMutation = useMutation({
    mutationFn: ({
      conversationId,
      message,
    }: {
      conversationId: string;
      message: Omit<ConversationMessage, 'id' | 'timestamp'>;
    }) => appendMessage(conversationId, message),
    onSuccess: (record) => {
      if (!record) return;
      setQueriesData(
        currentConversations.map((c) => (c.id === record.id ? record : c))
      );
    },
  });

  const moveMutation = useMutation({
    mutationFn: ({ id, folderId }: { id: string; folderId: string | null }) =>
      moveConversationToFolder(id, folderId),
    onSuccess: (record) => {
      if (!record) return;
      setQueriesData(
        currentConversations.map((c) => (c.id === record.id ? record : c))
      );
    },
  });

  const createFolderMutation = useMutation({
    mutationFn: ({ name, parentId }: { name: string; parentId?: string | null }) =>
      createFolder(name, parentId),
    onSuccess: (folder) => {
      setFoldersData([...currentFolders, folder]);
    },
  });

  const renameFolderMutation = useMutation({
    mutationFn: ({ id, name }: { id: string; name: string }) => renameFolder(id, name),
    onSuccess: (record) => {
      if (!record) return;
      setFoldersData(currentFolders.map((f) => (f.id === record.id ? record : f)));
    },
  });

  const deleteFolderMutation = useMutation({
    mutationFn: (id: string) => deleteFolder(id),
    onSuccess: () => invalidate(),
  });

  const pinMutation = useMutation({
    mutationFn: (id: string) => togglePin(id),
    onSuccess: (record) => {
      if (!record) return;
      setQueriesData(
        currentConversations.map((c) => (c.id === record.id ? record : c))
      );
    },
  });

  const shareMutation = useMutation({
    mutationFn: ({ id, shareWith }: { id: string; shareWith?: string[] }) =>
      toggleShare(id, shareWith),
    onSuccess: (record) => {
      if (!record) return;
      setQueriesData(
        currentConversations.map((c) => (c.id === record.id ? record : c))
      );
    },
  });

  const conversationsByScope = useMemo(() => {
    const grouped: Record<ConversationScope['kind'], Conversation[]> = {
      recent: [],
      pinned: [],
      shared: [],
      folder: [],
      trash: [],
    };
    currentConversations.forEach((c) => {
      grouped.recent.push(c);
      if (c.isPinned) grouped.pinned.push(c);
      if (c.isShared) grouped.shared.push(c);
    });
    grouped.recent.sort((a, b) => b.updatedAt.localeCompare(a.updatedAt));
    grouped.pinned.sort((a, b) => a.title.localeCompare(b.title));
    grouped.shared.sort((a, b) => b.updatedAt.localeCompare(a.updatedAt));
    return grouped;
  }, [currentConversations]);

  const conversationsInFolder = useCallback(
    (folderId: string | null) => {
      const list = currentConversations.filter((c) =>
        folderId === null ? !c.folderId : c.folderId === folderId
      );
      return list.sort((a, b) => b.updatedAt.localeCompare(a.updatedAt));
    },
    [currentConversations]
  );

  return {
    conversations: currentConversations,
    folders: currentFolders,
    conversationsByScope,
    conversationsInFolder,
    isLoading: conversationsQuery.isLoading || foldersQuery.isLoading,
    isError: conversationsQuery.isError || foldersQuery.isError,
    refresh: invalidate,
    getConversation: (id: string) =>
      getConversation(id).then((r) => r),
    createConversation: createMutation.mutateAsync,
    updateConversation: (id: string, patch: Partial<Conversation>) =>
      updateMutation.mutateAsync({ id, patch }),
    deleteConversation: removeMutation.mutateAsync,
    appendMessage: (conversationId: string, message: Omit<ConversationMessage, 'id' | 'timestamp'>) =>
      appendMutation.mutateAsync({ conversationId, message }),
    moveToFolder: (id: string, folderId: string | null) =>
      moveMutation.mutateAsync({ id, folderId }),
    pin: (id: string) => pinMutation.mutateAsync(id),
    share: (id: string, shareWith: string[] = []) =>
      shareMutation.mutateAsync({ id, shareWith }),
    createFolder: (name: string, parentId?: string | null) =>
      createFolderMutation.mutateAsync({ name, parentId }),
    renameFolder: (id: string, name: string) =>
      renameFolderMutation.mutateAsync({ id, name }),
    deleteFolder: (id: string) => deleteFolderMutation.mutateAsync(id),
  };
}
