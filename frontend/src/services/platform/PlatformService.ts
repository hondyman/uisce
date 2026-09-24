import { devWarn } from '../../utils/devLogger';
import { layoutManager } from '../docking/LayoutManager';

export type PlatformType = 'wails-desktop' | 'web-browser';

export interface ScreenLayoutTarget {
  id: string;
  name: string;
  isPrimary: boolean;
  index: number;
  scale?: number;
  x?: number;
  y?: number;
  width?: number;
  height?: number;
}

export interface SpawnOptions {
  viewId: string;
  route: string;
  title: string;
  targetScreenIndex?: number;
  x?: number;
  y?: number;
  width?: number;
  height?: number;
}

export interface IWorkspaceAdapter {
  getPlatformType(): PlatformType;
  isWails(): boolean;
  getScreens(): Promise<ScreenLayoutTarget[]>;
  displayView(options: SpawnOptions): Promise<void>;
}

export class UniversalPlatformService implements IWorkspaceAdapter {
  private openedPopups = new Map<string, Window>();

  constructor() {
    if (typeof window !== 'undefined') {
      window.addEventListener('desktop:window-closed', (e: any) => {
        const winId = e?.detail?.windowId;
        if (winId) {
          layoutManager.unregisterDetachedWindow(winId);
        }
      });
    }
  }

  /**
   * Lazily re-evaluates whether the Wails v3 desktop bindings are present.
   * Never cached as a module-level constant because Wails injects bindings asynchronously.
   */
  public isWails(): boolean {
    if (typeof window === 'undefined') return false;
    const win = window as unknown as {
      go?: { main?: { DeskWindowManager?: unknown } };
      wails?: unknown;
    };
    return typeof win.go?.main?.DeskWindowManager !== 'undefined';
  }

  public getPlatformType(): PlatformType {
    return this.isWails() ? 'wails-desktop' : 'web-browser';
  }

  /**
   * Enumerates connected physical or browser screens.
   */
  public async getScreens(): Promise<ScreenLayoutTarget[]> {
    if (this.isWails()) {
      try {
        const deskManager = (window as unknown as {
          go?: { main?: { DeskWindowManager?: { GetMonitors?: () => Promise<any[]> } } };
        })?.go?.main?.DeskWindowManager;

        if (typeof deskManager?.GetMonitors === 'function') {
          const screens = await deskManager.GetMonitors();
          if (Array.isArray(screens) && screens.length > 0) {
            return screens.map((s, idx) => ({
              id: s.id || `screen-${idx}`,
              name: s.name || `Display ${idx + 1}`,
              isPrimary: !!s.isPrimary,
              index: typeof s.index === 'number' ? s.index : idx,
              scale: s.scaleFactor || s.scale || 1,
              x: s.x,
              y: s.y,
              width: s.width,
              height: s.height,
            }));
          }
        }
      } catch (err) {
        devWarn('[PlatformService] Failed to query Wails monitors:', err);
      }
    }

    // Standard Browser Multi-Screen API fallback
    if (typeof window !== 'undefined' && 'getScreenDetails' in window) {
      try {
        const details = await (window as any).getScreenDetails();
        if (details?.screens?.length > 0) {
          return details.screens.map((s: any, idx: number) => ({
            id: `browser-screen-${idx}`,
            name: s.label || `Display ${idx + 1}`,
            isPrimary: !!s.isPrimary,
            index: idx,
            scale: s.devicePixelRatio || 1,
            width: s.width,
            height: s.height,
          }));
        }
      } catch {
        // User denied or browser doesn't permit screen enumeration
      }
    }

    // Default single-display fallback
    return [
      {
        id: 'primary',
        name: 'Primary Display',
        isPrimary: true,
        index: 0,
        width: typeof window !== 'undefined' ? window.innerWidth : 1920,
        height: typeof window !== 'undefined' ? window.innerHeight : 1080,
      },
    ];
  }

  /**
   * Universal view launcher: spawns an OS window in Desktop mode, or a popup in Browser mode.
   */
  public async displayView(options: SpawnOptions): Promise<void> {
    const { viewId, route, title, targetScreenIndex = 0, width = 1280, height = 800 } = options;

    if (this.isWails()) {
      const deskManager = (window as unknown as {
        go?: {
          main?: {
            DeskWindowManager?: {
              SpawnWindow?: (opts: unknown) => Promise<string>;
            };
          };
        };
      })?.go?.main?.DeskWindowManager;

      if (typeof deskManager?.SpawnWindow === 'function') {
        const jwt =
          sessionStorage.getItem('AUTH_TOKEN') ||
          localStorage.getItem('AUTH_TOKEN') ||
          localStorage.getItem('token') ||
          '';

        await deskManager.SpawnWindow({
          windowId: `win_${viewId}`,
          route,
          title,
          targetScreenIndex,
          x: options.x ?? 40,
          y: options.y ?? 40,
          width,
          height,
          sessionJwt: jwt,
        });
        return;
      }
    }

    // Browser Fallback: Open popup window
    if (typeof window !== 'undefined') {
      const left = options.x ?? 100;
      const top = options.y ?? 100;
      const features = `width=${width},height=${height},left=${left},top=${top},menubar=no,toolbar=no,location=no,status=no`;
      const winId = `win_${viewId}`;
      const popup = window.open(route, winId, features);
      if (popup) {
        this.openedPopups.set(winId, popup);
        const pollInterval = setInterval(() => {
          try {
            if (!popup || popup.closed) {
              clearInterval(pollInterval);
              this.openedPopups.delete(winId);
              layoutManager.unregisterDetachedWindow(winId);
            }
          } catch {
            clearInterval(pollInterval);
          }
        }, 1000);
      }
    }
  }

  /**
   * Reconciles registered detached windows with currently active OS/browser windows
   * and removes any orphaned or closed windows.
   */
  public async reconcileDetachedWindows(): Promise<void> {
    const detached = layoutManager.getDetachedWindows();
    if (detached.length === 0) return;

    if (this.isWails()) {
      try {
        const deskManager = (window as unknown as {
          go?: {
            main?: {
              DeskWindowManager?: {
                GetOpenWindowIDs?: () => Promise<string[]>;
              };
            };
          };
        })?.go?.main?.DeskWindowManager;

        if (typeof deskManager?.GetOpenWindowIDs === 'function') {
          const openIds = await deskManager.GetOpenWindowIDs();
          if (Array.isArray(openIds)) {
            const openSet = new Set(openIds);
            for (const win of detached) {
              if (!openSet.has(win.windowId)) {
                layoutManager.unregisterDetachedWindow(win.windowId);
              }
            }
          }
        }
      } catch (err) {
        devWarn('[PlatformService] Failed to reconcile Wails open windows:', err);
      }
    } else {
      // Browser popup reconciliation
      for (const win of detached) {
        const popup = this.openedPopups.get(win.windowId);
        if (popup && popup.closed) {
          this.openedPopups.delete(win.windowId);
          layoutManager.unregisterDetachedWindow(win.windowId);
        }
      }
    }
  }

  /**
   * Closes a specific detached window (Wails OS window or browser popup).
   */
  public async closeWindow(windowId: string): Promise<void> {
    if (this.isWails()) {
      try {
        const deskManager = (window as unknown as {
          go?: {
            main?: {
              DeskWindowManager?: {
                CloseWindow?: (id: string) => Promise<void>;
              };
            };
          };
        })?.go?.main?.DeskWindowManager;
        if (typeof deskManager?.CloseWindow === 'function') {
          await deskManager.CloseWindow(windowId);
          return;
        }
      } catch (err) {
        devWarn('[PlatformService] Failed to close Wails window:', err);
      }
    }

    // Browser popup
    const popup = this.openedPopups.get(windowId);
    if (popup && !popup.closed) {
      popup.close();
    }
    this.openedPopups.delete(windowId);
  }

  /**
   * Closes all active detached windows across the workstation.
   */
  public async closeAllDetachedWindows(): Promise<void> {
    const detached = layoutManager.getDetachedWindows();
    for (const win of detached) {
      await this.closeWindow(win.windowId);
    }
  }

  /**
   * Retrieves operational health telemetry from desktop host or browser mock.
   * Strictly secret-free: zero JWTs, tokens, or credentials exposed.
   */
  public async getHeartbeat(): Promise<{
    windowCount: number;
    openWindowIds: string[];
    screenCount: number;
    vaultTokenCount: number;
    timestamp: number;
  }> {
    if (this.isWails()) {
      try {
        const deskManager = (window as unknown as {
          go?: {
            main?: {
              DeskWindowManager?: {
                GetHeartbeat?: () => Promise<{
                  windowCount: number;
                  openWindowIds: string[];
                  screenCount: number;
                  vaultTokenCount: number;
                  timestamp: number;
                }>;
              };
            };
          };
        })?.go?.main?.DeskWindowManager;
        if (typeof deskManager?.GetHeartbeat === 'function') {
          return await deskManager.GetHeartbeat();
        }
      } catch (err) {
        devWarn('[PlatformService] Failed to retrieve Wails heartbeat:', err);
      }
    }

    const screens = await this.getScreens();
    const detached = layoutManager.getDetachedWindows();
    return {
      windowCount: 1 + detached.length,
      openWindowIds: ['win_main', ...detached.map((d) => d.windowId)],
      screenCount: screens.length,
      vaultTokenCount: 0,
      timestamp: Date.now(),
    };
  }
}

export const platformService = new UniversalPlatformService();

