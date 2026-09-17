import { devError, devLog } from '../../utils/devLogger';

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
 * Persistence Destination Architecture:
 * - v1 (Active): Client-side localStorage partition under key `uisce_workspace_layout_v1`.
 *   Provides zero-latency offline recovery and per-device desk layout persistence.
 * - v2 (Server Roadmap): Optional synchronization endpoint `/api/user/preferences/workspace-layout`
 *   for roaming multi-screen profiles backed by PostgreSQL.
 */
export class LayoutManager {
  private detachedWindows: Map<string, DetachedWindowInfo> = new Map();

  constructor() {
    this.hydrateDetachedWindows();
  }

  private hydrateDetachedWindows(): void {
    const state = this.loadLayout();
    if (state?.detachedWindows && Array.isArray(state.detachedWindows)) {
      state.detachedWindows.forEach((w) => {
        if (w.windowId) {
          this.detachedWindows.set(w.windowId, w);
        }
      });
    }
  }

  public saveLayout(dockviewLayout?: unknown): void {
    try {
      const state: DeskWorkspaceState = {
        version: 1,
        dockviewLayout,
        detachedWindows: Array.from(this.detachedWindows.values()),
        updatedAt: Date.now(),
      };
      localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
      devLog('[LayoutManager] Desk layout saved successfully');
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
      devError('[LayoutManager] Failed to load desk layout:', err);
      return null;
    }
  }

  public clearLayout(): void {
    localStorage.removeItem(STORAGE_KEY);
    this.detachedWindows.clear();
  }

  public registerDetachedWindow(win: DetachedWindowInfo): void {
    this.detachedWindows.set(win.windowId, win);
    // Persist updated list
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
