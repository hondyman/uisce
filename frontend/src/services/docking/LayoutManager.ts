import { devError, devLog } from '../../utils/devLogger';
import {
  fetchServerLayoutProfile,
  saveServerLayoutProfile,
} from './layoutProfileApi';

export interface DetachedWindowInfo {
  windowId: string;
  route: string;
  title: string;
  targetScreenIndex: number;
  width: number;
  height: number;
}

export interface DeskWorkspaceState {
  version: number;
  dockviewLayout?: unknown;
  detachedWindows: DetachedWindowInfo[];
  updatedAt: number;
}

const STORAGE_KEY = 'uisce_workspace_layout_v1';

/**
 * Manages persistence for Dockview in-tab layout and detached multi-monitor windows.
 * 
 * Persistence Architecture:
 * - Local Cache (Offline First): Instant retrieval and zero-latency local recovery
 *   via localStorage under `uisce_workspace_layout_v1`.
 * - PostgreSQL Server Profiles (Roaming Multi-Tenant): Synchronizes with
 *   `/api/user/preferences/workspace-layout` for roaming across devices.
 * - Single-Writer Rule: Only the main workspace hub window (/workspace) writes to
 *   PostgreSQL. Detached popout windows are strictly read-only consumers.
 * - Server-Authoritative Merge Rule: Server wins on conflict, localStorage serves as offline cache.
 */
export class LayoutManager {
  private detachedWindows: Map<string, DetachedWindowInfo> = new Map();
  private isWriter: boolean = true;

  constructor() {
    this.hydrateDetachedWindows();
    this.evaluateWriterRole();
  }

  private evaluateWriterRole(): void {
    if (typeof window !== 'undefined' && window.location) {
      const path = window.location.pathname;
      // Main hub is writer; view popouts (/view/rebalancer, etc.) are readers
      this.isWriter = path === '/' || path.startsWith('/workspace');
    }
  }

  public setWriterRole(isWriter: boolean): void {
    this.isWriter = isWriter;
  }

  public getIsWriter(): boolean {
    return this.isWriter;
  }

  private hydrateDetachedWindows(): void {
    const state = this.loadLayout();
    this.detachedWindows.clear();
    if (state?.detachedWindows && Array.isArray(state.detachedWindows)) {
      state.detachedWindows.forEach((w) => {
        if (w.windowId) {
          this.detachedWindows.set(w.windowId, w);
        }
      });
    }
  }

  public async saveLayout(dockviewLayout?: unknown, profileName: string = 'default'): Promise<void> {
    try {
      const state: DeskWorkspaceState = {
        version: 1,
        dockviewLayout,
        detachedWindows: Array.from(this.detachedWindows.values()),
        updatedAt: Date.now(),
      };
      this.saveToLocalStorage(state);
      devLog('[LayoutManager] Desk layout saved to local cache');

      // Single-Writer Rule: only the hub window writes to PostgreSQL
      if (this.isWriter) {
        await saveServerLayoutProfile(state, profileName);
      }
    } catch (err) {
      devError('[LayoutManager] Failed to save desk layout:', err);
    }
  }

  public loadLayout(): DeskWorkspaceState | null {
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      if (!raw) return null;
      return JSON.parse(raw) as DeskWorkspaceState;
    } catch (err) {
      devError('[LayoutManager] Failed to load desk layout from localStorage:', err);
      return null;
    }
  }

  /**
   * Synchronizes layout profile from PostgreSQL.
   * Server-Authoritative Merge:
   * If server has a saved profile, server state updates local storage and memory.
   * If server profile is unavailable or offline, local cache is preserved.
   */
  public async syncFromServer(profileName: string = 'default'): Promise<DeskWorkspaceState | null> {
    const serverState = await fetchServerLayoutProfile(profileName);
    if (serverState) {
      const localState = this.loadLayout();
      if (!localState || (serverState.updatedAt && serverState.updatedAt >= (localState.updatedAt || 0))) {
        devLog(`[LayoutManager] Server profile "${profileName}" is authoritative. Updating local workspace.`);
        this.saveToLocalStorage(serverState);
        this.hydrateDetachedWindows();
        return serverState;
      }
    }
    return this.loadLayout();
  }

  private saveToLocalStorage(state: DeskWorkspaceState): void {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
    } catch (err) {
      devError('[LayoutManager] Failed to cache state in localStorage:', err);
    }
  }

  public clearLayout(): void {
    localStorage.removeItem(STORAGE_KEY);
    this.detachedWindows.clear();
  }

  public registerDetachedWindow(win: DetachedWindowInfo): void {
    this.detachedWindows.set(win.windowId, win);
    const current = this.loadLayout();
    this.saveLayout(current?.dockviewLayout);
  }

  public unregisterDetachedWindow(windowId: string): void {
    if (this.detachedWindows.delete(windowId)) {
      const current = this.loadLayout();
      this.saveLayout(current?.dockviewLayout);
    }
  }

  public clearDetachedWindows(): void {
    this.detachedWindows.clear();
    const current = this.loadLayout();
    this.saveLayout(current?.dockviewLayout);
  }

  public getDetachedWindows(): DetachedWindowInfo[] {
    return Array.from(this.detachedWindows.values());
  }
}

export const layoutManager = new LayoutManager();

