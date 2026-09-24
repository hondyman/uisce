import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import { LayoutManager, DeskWorkspaceState } from '../../services/docking/LayoutManager';
import * as layoutApi from '../../services/docking/layoutProfileApi';

describe('LayoutManager & Roaming Profile Sync', () => {
  beforeEach(() => {
    localStorage.clear();
    vi.restoreAllMocks();
  });

  afterEach(() => {
    localStorage.clear();
    vi.restoreAllMocks();
  });

  it('enforces single-writer rule: hub window saves to server, non-hub popouts do not', async () => {
    const saveServerSpy = vi.spyOn(layoutApi, 'saveServerLayoutProfile').mockResolvedValue(true);
    const manager = new LayoutManager();

    // 1. As Writer (Hub window)
    manager.setWriterRole(true);
    await manager.saveLayout({ grid: 'data' }, 'default');

    expect(saveServerSpy).toHaveBeenCalledTimes(1);
    expect(saveServerSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        dockviewLayout: { grid: 'data' },
      }),
      'default'
    );

    // 2. As Reader (Detached popout window)
    saveServerSpy.mockClear();
    manager.setWriterRole(false);
    await manager.saveLayout({ grid: 'data2' }, 'default');

    // Must NOT write to server!
    expect(saveServerSpy).not.toHaveBeenCalled();

    // But MUST save to local cache
    const cached = manager.loadLayout();
    expect(cached?.dockviewLayout).toEqual({ grid: 'data2' });
  });

  it('applies server-authoritative merge: server profile updates local cache when available', async () => {
    const serverState: DeskWorkspaceState = {
      version: 1,
      dockviewLayout: { serverGrid: true },
      detachedWindows: [
        {
          windowId: 'win_rebalancer',
          route: '/view/rebalancer',
          title: 'Rebalancer',
          targetScreenIndex: 1,
          width: 1200,
          height: 800,
        },
      ],
      updatedAt: Date.now() + 10000,
    };

    vi.spyOn(layoutApi, 'fetchServerLayoutProfile').mockResolvedValue(serverState);

    const manager = new LayoutManager();
    const synced = await manager.syncFromServer('default');

    expect(synced).toEqual(serverState);
    expect(manager.getDetachedWindows()).toHaveLength(1);
    expect(manager.getDetachedWindows()[0].windowId).toBe('win_rebalancer');

    // LocalStorage must reflect server state
    const cached = manager.loadLayout();
    expect(cached?.dockviewLayout).toEqual({ serverGrid: true });
  });

  it('preserves local cache if server is offline or returns 404', async () => {
    // Seed local cache
    const localState: DeskWorkspaceState = {
      version: 1,
      dockviewLayout: { localOfflineGrid: true },
      detachedWindows: [],
      updatedAt: 100,
    };
    localStorage.setItem('uisce_workspace_layout_v1', JSON.stringify(localState));

    // Simulate 404 / network offline
    vi.spyOn(layoutApi, 'fetchServerLayoutProfile').mockResolvedValue(null);

    const manager = new LayoutManager();
    const synced = await manager.syncFromServer('default');

    expect(synced?.dockviewLayout).toEqual({ localOfflineGrid: true });
    expect(manager.loadLayout()?.dockviewLayout).toEqual({ localOfflineGrid: true });
  });
});
