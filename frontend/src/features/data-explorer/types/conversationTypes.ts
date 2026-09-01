/**
 * Data Explorer — Conversation + Folder + Share model.
 *
 * Conversations carry their own message history plus an attached
 * ExplorerQueryState so a user can flip back to a prior exploration
 * and re-run it. Folders group conversations like a workspace.
 * Sharing is keyed by user-id; share-list is just an array of strings
 * the tenant authorises.
 */

import type { ExplorerQueryState } from './dataExplorerTypes';

export type ConversationMessageRole = 'user' | 'assistant' | 'system';

export interface ConversationMessage {
  id: string;
  role: ConversationMessageRole;
  content: string;
  timestamp: string;
  /** Optional snapshot of the QueryState derived from this message. */
  querySnapshot?: ExplorerQueryState;
  /** Backend-generated SQL that came back from NL→SQL for this turn. */
  generatedSql?: string;
  /** Confidence score (0..1) from the backend's parsed intent. */
  confidence?: number;
}

export interface Conversation {
  id: string;
  title: string;
  sourceId?: string;
  bindingId?: string;
  folderId?: string | null;
  isPinned: boolean;
  isShared: boolean;
  sharedWith: string[];
  messages: ConversationMessage[];
  queryState?: ExplorerQueryState;
  createdAt: string;
  updatedAt: string;
  ownerUserId?: string;
  ownerEmail?: string;
}

export interface ConversationFolder {
  id: string;
  name: string;
  parentId?: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface ConversationStoreState {
  folders: ConversationFolder[];
  conversations: Conversation[];
}

/**
 * Conversations scoped to "active" — recent activity list (default landing view).
 */
export type ConversationScope =
  | { kind: 'recent' }
  | { kind: 'pinned' }
  | { kind: 'shared' }
  | { kind: 'folder'; folderId: string }
  | { kind: 'trash' };
