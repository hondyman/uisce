/**
 * Conversation API — localStorage-backed CRUD for Phase 1.
 *
 * Schema mirrors what we'd persist server-side so we can later swap
 * to a backend (POST/GET/DELETE /api/data-explorer/conversations,
 * /api/data-explorer/folders, /api/data-explorer/share) without
 * changing call sites.
 */

import { apiFetch } from '../../../lib/apiClient';
import { devError } from '../../../utils/devLogger';
import type {
  Conversation,
  ConversationFolder,
  ConversationMessage,
  ConversationStoreState,
} from '../types/conversationTypes';

const STORE_KEY = 'data_explorer.conversations.v1';
const REMOTE_BASE = '/api/data-explorer';

function newId(prefix: string) {
  return `${prefix}-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 7)}`;
}

function emptyStore(): ConversationStoreState {
  return { folders: [], conversations: [] };
}

function readStore(): ConversationStoreState {
  if (typeof window === 'undefined') return emptyStore();
  try {
    const raw = window.localStorage.getItem(STORE_KEY);
    if (!raw) return emptyStore();
    const parsed = JSON.parse(raw);
    if (!parsed || typeof parsed !== 'object') return emptyStore();
    return {
      folders: Array.isArray(parsed.folders) ? parsed.folders : [],
      conversations: Array.isArray(parsed.conversations) ? parsed.conversations : [],
    };
  } catch {
    return emptyStore();
  }
}

function writeStore(state: ConversationStoreState): void {
  if (typeof window === 'undefined') return;
  try {
    window.localStorage.setItem(STORE_KEY, JSON.stringify(state));
  } catch (err) {
    devError('Failed to persist conversation store', err);
  }
}

async function tryRemote<T>(path: string, init?: RequestInit): Promise<T | null> {
  try {
    const response = await apiFetch(path, {
      headers: { 'Content-Type': 'application/json' },
      ...init,
    });
    if (!response.ok) return null;
    const contentType = response.headers.get('content-type') || '';
    if (contentType.includes('application/json')) {
      return (await response.json()) as T;
    }
    return null;
  } catch {
    return null;
  }
}

async function load(): Promise<ConversationStoreState> {
  const remote = await tryRemote<ConversationStoreState>(`${REMOTE_BASE}/state`);
  if (remote && Array.isArray(remote.conversations) && Array.isArray(remote.folders)) {
    writeStore(remote);
    return remote;
  }
  return readStore();
}

// ─── Folders ────────────────────────────────────────────────────────────────

export async function listFolders(): Promise<ConversationFolder[]> {
  const state = await load();
  return [...state.folders].sort((a, b) => a.name.localeCompare(b.name));
}

export async function createFolder(name: string, parentId?: string | null): Promise<ConversationFolder> {
  const state = readStore();
  const now = new Date().toISOString();
  const folder: ConversationFolder = {
    id: newId('fld'),
    name: name.trim() || 'New folder',
    parentId: parentId ?? null,
    createdAt: now,
    updatedAt: now,
  };
  const next = { ...state, folders: [...state.folders, folder] };
  writeStore(next);
  void tryRemote(`${REMOTE_BASE}/folders`, {
    method: 'POST',
    body: JSON.stringify(folder),
  });
  return folder;
}

export async function renameFolder(id: string, name: string): Promise<ConversationFolder | null> {
  const state = readStore();
  const idx = state.folders.findIndex((f) => f.id === id);
  if (idx === -1) return null;
  const updated: ConversationFolder = {
    ...state.folders[idx],
    name: name.trim() || state.folders[idx].name,
    updatedAt: new Date().toISOString(),
  };
  const folders = [...state.folders];
  folders[idx] = updated;
  writeStore({ ...state, folders });
  void tryRemote(`${REMOTE_BASE}/folders/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    body: JSON.stringify({ name: updated.name }),
  });
  return updated;
}

export async function deleteFolder(id: string): Promise<void> {
  const state = readStore();
  const folders = state.folders.filter((f) => f.id !== id);
  // Detach conversations from this folder rather than deleting them
  const conversations = state.conversations.map((c) =>
    c.folderId === id ? { ...c, folderId: null, updatedAt: new Date().toISOString() } : c
  );
  writeStore({ folders, conversations });
  void tryRemote(`${REMOTE_BASE}/folders/${encodeURIComponent(id)}`, { method: 'DELETE' });
}

// ─── Conversations ──────────────────────────────────────────────────────────

export async function listConversations(filter?: {
  pinned?: boolean;
  shared?: boolean;
  folderId?: string | null;
  search?: string;
}): Promise<Conversation[]> {
  const state = await load();
  let list = state.conversations;
  if (filter?.pinned !== undefined) list = list.filter((c) => c.isPinned === filter.pinned);
  if (filter?.shared !== undefined) list = list.filter((c) => c.isShared === filter.shared);
  if (filter?.folderId !== undefined) {
    if (filter.folderId === null) list = list.filter((c) => !c.folderId);
    else list = list.filter((c) => c.folderId === filter.folderId);
  }
  if (filter?.search) {
    const q = filter.search.toLowerCase();
    list = list.filter(
      (c) =>
        c.title.toLowerCase().includes(q) ||
        c.messages.some((m) => m.content.toLowerCase().includes(q))
    );
  }
  return [...list].sort((a, b) => b.updatedAt.localeCompare(a.updatedAt));
}

export async function getConversation(id: string): Promise<Conversation | null> {
  const state = readStore();
  return state.conversations.find((c) => c.id === id) ?? null;
}

export async function createConversation(
  initial?: Partial<Conversation>
): Promise<Conversation> {
  const state = readStore();
  const now = new Date().toISOString();
  const conversation: Conversation = {
    id: newId('cnv'),
    title: initial?.title ?? 'New conversation',
    sourceId: initial?.sourceId,
    bindingId: initial?.bindingId,
    folderId: initial?.folderId ?? null,
    isPinned: initial?.isPinned ?? false,
    isShared: initial?.isShared ?? false,
    sharedWith: initial?.sharedWith ?? [],
    messages: initial?.messages ?? [],
    queryState: initial?.queryState,
    createdAt: now,
    updatedAt: now,
    ownerUserId: initial?.ownerUserId,
    ownerEmail: initial?.ownerEmail,
  };
  const next = { ...state, conversations: [conversation, ...state.conversations] };
  writeStore(next);
  void tryRemote(`${REMOTE_BASE}/conversations`, {
    method: 'POST',
    body: JSON.stringify(conversation),
  });
  return conversation;
}

export async function updateConversation(
  id: string,
  patch: Partial<Conversation>
): Promise<Conversation | null> {
  const state = readStore();
  const idx = state.conversations.findIndex((c) => c.id === id);
  if (idx === -1) return null;
  const updated: Conversation = {
    ...state.conversations[idx],
    ...patch,
    id: state.conversations[idx].id,
    createdAt: state.conversations[idx].createdAt,
    updatedAt: new Date().toISOString(),
  };
  const conversations = [...state.conversations];
  conversations[idx] = updated;
  writeStore({ ...state, conversations });
  void tryRemote(`${REMOTE_BASE}/conversations/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    body: JSON.stringify(updated),
  });
  return updated;
}

export async function appendMessage(
  conversationId: string,
  message: Omit<ConversationMessage, 'id' | 'timestamp'>
): Promise<Conversation | null> {
  const state = readStore();
  const idx = state.conversations.findIndex((c) => c.id === conversationId);
  if (idx === -1) return null;
  const fullMessage: ConversationMessage = {
    id: newId('msg'),
    timestamp: new Date().toISOString(),
    ...message,
  };
  const updated: Conversation = {
    ...state.conversations[idx],
    messages: [...state.conversations[idx].messages, fullMessage],
    updatedAt: new Date().toISOString(),
    // Auto-name from first user message if title is empty
    title:
      state.conversations[idx].title ||
      (fullMessage.role === 'user' ? truncate(fullMessage.content, 60) : state.conversations[idx].title),
  };
  const conversations = [...state.conversations];
  conversations[idx] = updated;
  writeStore({ ...state, conversations });
  void tryRemote(`${REMOTE_BASE}/conversations/${encodeURIComponent(conversationId)}/messages`, {
    method: 'POST',
    body: JSON.stringify(fullMessage),
  });
  return updated;
}

export async function deleteConversation(id: string): Promise<void> {
  const state = readStore();
  const conversations = state.conversations.filter((c) => c.id !== id);
  writeStore({ ...state, conversations });
  void tryRemote(`${REMOTE_BASE}/conversations/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  });
}

export async function togglePin(id: string): Promise<Conversation | null> {
  const conv = await getConversation(id);
  if (!conv) return null;
  return updateConversation(id, { isPinned: !conv.isPinned });
}

export async function toggleShare(
  id: string,
  shareWith: string[] = []
): Promise<Conversation | null> {
  const conv = await getConversation(id);
  if (!conv) return null;
  const nextShared = !conv.isShared;
  const uniqueShared = Array.from(
    new Set(shareWith.map((s) => s.trim()).filter((s) => s.length > 0))
  );
  return updateConversation(id, {
    isShared: nextShared,
    sharedWith: nextShared ? uniqueShared : [],
  });
}

export async function moveConversationToFolder(
  id: string,
  folderId: string | null
): Promise<Conversation | null> {
  return updateConversation(id, { folderId });
}

function truncate(value: string, max: number): string {
  const trimmed = value.trim().replace(/\s+/g, ' ');
  return trimmed.length > max ? `${trimmed.slice(0, max - 1)}…` : trimmed;
}
