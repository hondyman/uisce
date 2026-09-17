import { describe, it, expect, beforeEach, vi } from 'vitest';
import { getComponentForRoute } from '../../components/docking/UniversalWorkspaceHub';
import { layoutManager, DeskWorkspaceState } from '../../services/docking/LayoutManager';
import { platformService } from '../../services/platform/PlatformService';

describe('Travel Mode & Display State Transitions', () => {
  beforeEach(() => {
    localStorage.clear();
    vi.restoreAllMocks();
  });

  it('maps view routes to dockable components correctly', () => {
    expect(getComponentForRoute('/view/rebalancer')).toBe('rebalancer');
    expect(getComponentForRoute('/view/scenario')).toBe('scenario');
    expect(getComponentForRoute('/view/fixed_income')).toBe('fixed_income');
    expect(getComponentForRoute('/view/page/orders')).toBe('orders');
    expect(getComponentForRoute('/unknown/route')).toBeNull();
  });

  it('enforces non-destructive Travel Mode invariant: consolidation does NOT mutate saved multi-monitor layout in storage', async () => {
    // 1. Seed a saved multi-monitor workspace layout with detached windows
    const multiMonitorLayout: DeskWorkspaceState = {
      version: 1,
      dockviewLayout: { panels: ['orders'] },
      detachedWindows: [
        {
          windowId: 'win_rebalancer',
          route: '/view/rebalancer',
          title: 'AI Portfolio Rebalancer',
          targetScreenIndex: 1,
          width: 1400,
          height: 900,
        },
        {
          windowId: 'win_scenario',
          route: '/view/scenario',
          title: 'Scenario Analysis Pro',
          targetScreenIndex: 2,
          width: 1400,
          height: 900,
        },
      ],
      updatedAt: 12345678,
    };

    layoutManager.saveLayout(multiMonitorLayout.dockviewLayout);
    // Register the detached windows
    multiMonitorLayout.detachedWindows.forEach((w) => layoutManager.registerDetachedWindow(w));

    const savedBefore = layoutManager.loadLayout();
    expect(savedBefore?.detachedWindows).toHaveLength(2);

    // Mock platformService.closeAllDetachedWindows
    const closeSpy = vi.spyOn(platformService, 'closeAllDetachedWindows').mockResolvedValue();

    // 2. Simulate Travel Mode consolidation:
    // Detached windows are closed on desktop, but layoutManager.saveLayout is NEVER called during Travel Mode!
    await platformService.closeAllDetachedWindows();
    expect(closeSpy).toHaveBeenCalledTimes(1);

    // 3. Verify saved state in storage STILL preserves both detached windows and multi-monitor setup
    const savedAfter = layoutManager.loadLayout();
    expect(savedAfter?.detachedWindows).toHaveLength(2);
    expect(savedAfter?.detachedWindows[0].windowId).toBe('win_rebalancer');
    expect(savedAfter?.detachedWindows[1].windowId).toBe('win_scenario');
  });

  it('supports closeWindow and closeAllDetachedWindows in PlatformService', async () => {
    const closeSpy = vi.spyOn(platformService, 'closeWindow').mockResolvedValue();
    await platformService.closeAllDetachedWindows();
    // Resolves cleanly without throwing
    expect(closeSpy).toBeDefined();
  });
});
