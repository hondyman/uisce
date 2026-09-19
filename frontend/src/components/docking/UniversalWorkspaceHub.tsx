import React, { useCallback, useEffect, useRef, useState } from 'react';
import { DockviewReact, DockviewReadyEvent, IDockviewPanelProps, DockviewApi } from 'dockview-react';
import 'dockview-react/dist/styles/dockview.css';
import '../../services/docking/theme.css';
import { platformService, ScreenLayoutTarget } from '../../services/platform/PlatformService';
import { layoutManager } from '../../services/docking/LayoutManager';
import { devLog, devWarn } from '../../utils/devLogger';
import { useFdc3 } from '../../services/fdc3/useFdc3';
import { UserChannelId, Fdc3Context } from '../../services/fdc3/types';
import { fdc3Agent } from '../../services/fdc3/Fdc3DesktopAgent';
import { IntentResolverModal } from './IntentResolverModal';
import { IntentTarget, StandardIntent } from '../../services/fdc3/intentTypes';
import { PanelErrorBoundary } from './PanelErrorBoundary';
import { WorkstationCommandBar } from './WorkstationCommandBar';

const StandalonePageRenderer = React.lazy<React.ComponentType<{ slug?: string; recordId?: string }>>(() =>
  import('../../pages/PageBrowser').then((m) => ({ default: m.StandalonePageRenderer }))
);
const AIPortfolioRebalancer = React.lazy(() => import('../AIPortfolioRebalancer'));
const ScenarioAnalysisPro = React.lazy(() => import('../ScenarioAnalysisPro'));
const FixedIncomeDashboard = React.lazy(() => import('../FixedIncomeDashboard'));

// Module-level guard to prevent React StrictMode double-restoration on desktop boot
let hasAutoRestoredOnBoot = false;

const OrderBlotterPanel: React.FC = () => {
  const { activeChannel, broadcast, raiseIntent } = useFdc3();
  const demoTickers = ['AAPL', 'MSFT', 'NVDA', 'GOOGL', 'TSLA'];

  return (
    <div style={{ height: '100%', width: '100%', display: 'flex', flexDirection: 'column', background: '#050d1a' }}>
      {/* Test / Demo FDC3 quick-sync bar */}
      <div style={{
        background: '#091428',
        borderBottom: '1px solid #1e293b',
        padding: '6px 12px',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        flexWrap: 'wrap',
        gap: '8px',
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px', flexWrap: 'wrap' }}>
          <span style={{ fontSize: '11px', color: '#94a3b8', fontWeight: 600 }}>
            Demo / Test FDC3 Sync:
          </span>
          {demoTickers.map((ticker) => (
            <button
              key={ticker}
              onClick={() => {
                broadcast({
                  type: 'fdc3.instrument',
                  id: { ticker },
                  name: `${ticker} Equity`,
                });
              }}
              style={{
                padding: '2px 8px',
                fontSize: '11px',
                fontWeight: 600,
                borderRadius: '4px',
                border: '1px solid #38bdf840',
                background: '#0284c720',
                color: '#38bdf8',
                cursor: 'pointer',
              }}
              title={`Broadcast ${ticker} to all linked windows`}
            >
              {ticker}
            </button>
          ))}

          {/* Raise Intent Trigger */}
          <button
            onClick={async () => {
              try {
                await raiseIntent('ViewAnalysis', {
                  type: 'fdc3.instrument',
                  id: { ticker: 'NVDA' },
                  name: 'NVIDIA Corp',
                });
              } catch (err: any) {
                console.warn('[OrderBlotter] Intent resolution notice:', err?.message || err);
              }
            }}
            style={{
              padding: '2px 8px',
              fontSize: '11px',
              fontWeight: 600,
              borderRadius: '4px',
              border: '1px solid #a855f760',
              background: '#7e22ce20',
              color: '#c084fc',
              cursor: 'pointer',
              marginLeft: '4px',
            }}
            title="Raise 'ViewAnalysis' intent across workstation mesh (prompts resolver if multiple targets exist)"
          >
            ⚡ Raise ViewAnalysis
          </button>
        </div>
        <div style={{ fontSize: '11px', color: '#64748b' }}>
          Active Channel: <strong style={{ color: '#38bdf8' }}>{activeChannel}</strong>
        </div>
      </div>

      <div style={{ flex: 1, overflow: 'auto' }}>
        <React.Suspense fallback={<div style={{ padding: 20, color: '#94a3b8' }}>Loading Order Blotter...</div>}>
          <StandalonePageRenderer slug="orders" />
        </React.Suspense>
      </div>
    </div>
  );
};

// Panel components registered with Dockview wrapped in PanelErrorBoundary
const dockComponents = {
  orders: (_props: IDockviewPanelProps) => (
    <PanelErrorBoundary panelTitle="OMS Order Blotter">
      <OrderBlotterPanel />
    </PanelErrorBoundary>
  ),
  rebalancer: (_props: IDockviewPanelProps) => (
    <PanelErrorBoundary panelTitle="AI Portfolio Rebalancer">
      <div style={{ height: '100%', width: '100%', overflow: 'auto', background: '#050d1a' }}>
        <React.Suspense fallback={<div style={{ padding: 20, color: '#94a3b8' }}>Loading Rebalancer...</div>}>
          <AIPortfolioRebalancer />
        </React.Suspense>
      </div>
    </PanelErrorBoundary>
  ),
  scenario: (_props: IDockviewPanelProps) => (
    <PanelErrorBoundary panelTitle="Scenario Analysis Pro">
      <div style={{ height: '100%', width: '100%', overflow: 'auto', background: '#050d1a' }}>
        <React.Suspense fallback={<div style={{ padding: 20, color: '#94a3b8' }}>Loading Scenario Analysis...</div>}>
          <ScenarioAnalysisPro />
        </React.Suspense>
      </div>
    </PanelErrorBoundary>
  ),
  fixed_income: (_props: IDockviewPanelProps) => (
    <PanelErrorBoundary panelTitle="Fixed Income Dashboard">
      <div style={{ height: '100%', width: '100%', overflow: 'auto', background: '#050d1a' }}>
        <React.Suspense fallback={<div style={{ padding: 20, color: '#94a3b8' }}>Loading Fixed Income...</div>}>
          <FixedIncomeDashboard />
        </React.Suspense>
      </div>
    </PanelErrorBoundary>
  ),
};

export interface WorkstationAlert {
  type: 'browser-restore' | 'travel-mode' | 'reconnect';
  title: string;
  message: string;
  actionLabel: string;
  dismissLabel: string;
  onAction: () => void;
  onDismiss: () => void;
}

export function getComponentForRoute(route: string): 'orders' | 'rebalancer' | 'scenario' | 'fixed_income' | null {
  if (route.includes('rebalancer')) return 'rebalancer';
  if (route.includes('scenario')) return 'scenario';
  if (route.includes('fixed_income') || route.includes('fixed-income')) return 'fixed_income';
  if (route.includes('orders')) return 'orders';
  return null;
}

export const UniversalWorkspaceHub: React.FC = () => {
  const [dockApi, setDockApi] = useState<DockviewApi | null>(null);
  const [screens, setScreens] = useState<ScreenLayoutTarget[]>([]);
  const [isDesktop, setIsDesktop] = useState<boolean>(false);
  const [statusMessage, setStatusMessage] = useState<string>('');
  const statusTimerRef = useRef<NodeJS.Timeout | null>(null);

  // Single cohesive alert surface (replaces separate stacked banners)
  const [activeAlert, setActiveAlert] = useState<WorkstationAlert | null>(null);

  // Travel Mode state (ephemeral view-time consolidation)
  const [isTravelMode, setIsTravelMode] = useState<boolean>(false);
  const [consolidatedPanelIds, setConsolidatedPanelIds] = useState<string[]>([]);
  const prevScreenCountRef = useRef<number>(1);
  const isTravelModeRef = useRef<boolean>(false);
  isTravelModeRef.current = isTravelMode;

  // Cross-window intent resolver prompt modal state
  const [resolverModal, setResolverModal] = useState<{
    isOpen: boolean;
    intent: StandardIntent | null;
    context: Fdc3Context | null;
    targets: IntentTarget[];
    resolve?: (target: IntentTarget) => void;
    reject?: (err: Error) => void;
  }>({
    isOpen: false,
    intent: null,
    context: null,
    targets: [],
  });

  const { activeChannel, setChannel } = useFdc3();
  const availableChannels: UserChannelId[] = ['blue', 'green', 'red', 'orange', 'purple'];

  // Internal Command Bar state
  const [isCommandBarOpen, setIsCommandBarOpen] = useState<boolean>(false);

  // Local-First Workstation Heartbeat Telemetry (1-second cheap poll)
  const [heartbeat, setHeartbeat] = useState<{
    windowCount: number;
    openWindowIds: string[];
    screenCount: number;
    vaultTokenCount: number;
    timestamp: number;
  }>({
    windowCount: 1,
    openWindowIds: ['win_main'],
    screenCount: 1,
    vaultTokenCount: 0,
    timestamp: Date.now(),
  });

  useEffect(() => {
    let mounted = true;
    const pollHeartbeat = async () => {
      try {
        const hb = await platformService.getHeartbeat();
        if (mounted) setHeartbeat(hb);
      } catch {}
    };
    pollHeartbeat();
    const interval = setInterval(pollHeartbeat, 1000);
    return () => {
      mounted = false;
      clearInterval(interval);
    };
  }, []);

  const showStatus = (msg: string) => {
    setStatusMessage(msg);
    if (statusTimerRef.current) clearTimeout(statusTimerRef.current);
    statusTimerRef.current = setTimeout(() => setStatusMessage(''), 4000);
  };

  // Keyboard shortcut listener: Cmd/Ctrl+K for Command Bar, Alt+1..9 for Browser Tab Switching
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Cmd+K (Mac) or Ctrl+K (Windows/Linux)
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        setIsCommandBarOpen((prev) => !prev);
        return;
      }

      // Browser mode: Alt+1..9 switches active Dockview panel tabs
      if (e.altKey && !e.ctrlKey && !e.metaKey && e.code.startsWith('Digit')) {
        const digit = parseInt(e.code.replace('Digit', ''), 10);
        if (digit >= 1 && digit <= 9 && dockApi) {
          const panels = dockApi.panels;
          const targetIndex = digit - 1;
          if (targetIndex < panels.length) {
            e.preventDefault();
            panels[targetIndex].api.setActive();
            showStatus(`Switched to tab ${digit}: ${panels[targetIndex].title}`);
          }
        }
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [dockApi]);

  useEffect(() => {
    // Wire FDC3 agent intent resolver prompt to UI modal
    fdc3Agent.setIntentResolverPrompt((intent, targets, context) => {
      return new Promise<IntentTarget>((resolve, reject) => {
        setResolverModal({
          isOpen: true,
          intent,
          context: context || null,
          targets,
          resolve,
          reject,
        });
      });
    });
  }, []);

  useEffect(() => {
    document.title = 'Uisce Multi-Monitor Workstation';
    const desktop = platformService.isWails();
    setIsDesktop(desktop);

    // Synchronize roaming workspace profile from PostgreSQL
    layoutManager.syncFromServer().then((synced) => {
      if (synced?.dockviewLayout && dockApi) {
        try {
          dockApi.fromJSON(synced.dockviewLayout as any);
        } catch (e) {
          devLog('[UniversalWorkspaceHub] Failed to apply server layout:', e);
        }
      }
    }).catch(console.error);

    platformService.getScreens().then((availScreens) => {
      setScreens(availScreens);
      prevScreenCountRef.current = availScreens.length;

      const detached = layoutManager.getDetachedWindows();
      if (detached.length > 0 && !hasAutoRestoredOnBoot) {
        hasAutoRestoredOnBoot = true;

        if (desktop) {
          // Desktop mode: auto-spawn windows to their displays with clamping
          devLog(`[UniversalWorkspaceHub] Auto-restoring ${detached.length} detached windows on desktop boot...`);
          detached.forEach((win) => {
            const maxScreen = Math.max(0, availScreens.length - 1);
            const clampedIndex = Math.min(win.targetScreenIndex, maxScreen);
            platformService.displayView({
              viewId: win.windowId.replace(/^win_/, ''),
              route: win.route,
              title: win.title,
              targetScreenIndex: clampedIndex,
              width: win.width,
              height: win.height,
            }).catch(console.error);
          });
          showStatus(`Auto-restored ${detached.length} multi-monitor window(s)`);
        } else {
          // Browser mode: prompt via unified alert strip
          setActiveAlert({
            type: 'browser-restore',
            title: 'Restore Session',
            message: `Found ${detached.length} detached desk window(s) from previous session.`,
            actionLabel: 'Restore Windows',
            dismissLabel: 'Dismiss',
            onAction: () => {
              handleRestoreDesk();
              setActiveAlert(null);
            },
            onDismiss: () => {
              layoutManager.clearDetachedWindows();
              setActiveAlert(null);
            },
          });
        }
      }
    }).catch(console.error);
  }, []);

  // Initialize default layout or restore saved layout
  const onReady = useCallback((event: DockviewReadyEvent) => {
    const api = event.api;
    setDockApi(api);

    const savedState = layoutManager.loadLayout();
    if (savedState?.dockviewLayout) {
      try {
        api.fromJSON(savedState.dockviewLayout as any);
        devLog('[UniversalWorkspaceHub] Restored saved dockview layout');
        return;
      } catch (err) {
        devLog('[UniversalWorkspaceHub] Could not restore layout, initializing default:', err);
      }
    }

    // Default multi-panel layout
    const panelOrders = api.addPanel({
      id: 'panel_orders',
      component: 'orders',
      title: 'OMS Order Blotter',
    });

    api.addPanel({
      id: 'panel_rebalancer',
      component: 'rebalancer',
      title: 'AI Portfolio Rebalancer',
      position: { referencePanel: panelOrders, direction: 'right' },
    });

    api.addPanel({
      id: 'panel_scenario',
      component: 'scenario',
      title: 'Scenario Analysis Pro',
      position: { referencePanel: panelOrders, direction: 'below' },
    });
  }, []);

  // Save current workspace layout
  const handleSaveLayout = async () => {
    if (!dockApi) return;
    await platformService.reconcileDetachedWindows();
    const json = dockApi.toJSON();
    await layoutManager.saveLayout(json);
    showStatus('Workspace layout saved');
  };

  // Restore entire desk (Dockview layout + detached multi-monitor windows)
  const handleRestoreDesk = async () => {
    const saved = layoutManager.loadLayout();
    if (!saved) {
      showStatus('No saved desk layout found');
      return;
    }

    if (saved.dockviewLayout && dockApi) {
      try {
        dockApi.fromJSON(saved.dockviewLayout as any);
      } catch (e) {
        console.error('Dockview restore error:', e);
      }
    }

    const detached = saved.detachedWindows || [];
    if (detached.length > 0) {
      const availScreens = screens.length > 0 ? screens : await platformService.getScreens();
      const maxScreen = Math.max(0, availScreens.length - 1);
      for (const win of detached) {
        const clampedIndex = Math.min(win.targetScreenIndex, maxScreen);
        await platformService.displayView({
          viewId: win.windowId.replace(/^win_/, ''),
          route: win.route,
          title: win.title,
          targetScreenIndex: clampedIndex,
          width: win.width,
          height: win.height,
        });
      }
      setActiveAlert(null);
      showStatus(`Restored workspace & ${detached.length} detached window(s)`);
    } else {
      showStatus('Workspace dock layout restored');
    }
  };

  // Reset to default layout
  const handleResetLayout = () => {
    layoutManager.clearLayout();
    setActiveAlert(null);
    setIsTravelMode(false);
    setConsolidatedPanelIds([]);
    if (dockApi) {
      dockApi.clear();
      const p1 = dockApi.addPanel({
        id: 'panel_orders',
        component: 'orders',
        title: 'OMS Order Blotter',
      });
      dockApi.addPanel({
        id: 'panel_rebalancer',
        component: 'rebalancer',
        title: 'AI Portfolio Rebalancer',
        position: { referencePanel: p1, direction: 'right' },
      });
      dockApi.addPanel({
        id: 'panel_scenario',
        component: 'scenario',
        title: 'Scenario Analysis Pro',
        position: { referencePanel: p1, direction: 'below' },
      });
    }
    showStatus('Layout reset to default');
  };

  // Travel Mode: Consolidate detached multi-monitor windows into docked tabs (Non-destructive view-time mode)
  const handleConsolidateToTabs = async () => {
    if (!dockApi) return;
    const detached = layoutManager.getDetachedWindows();

    const newPanelIds: string[] = [];
    for (const win of detached) {
      const comp = getComponentForRoute(win.route);
      if (comp) {
        const panelId = `panel_consolidated_${win.windowId}`;
        try {
          dockApi.addPanel({
            id: panelId,
            component: comp,
            title: `${win.title} (Tab)`,
          });
          newPanelIds.push(panelId);
        } catch (err) {
          devWarn('[UniversalWorkspaceHub] Failed to add consolidated panel:', err);
        }
      }
    }

    // Close detached windows on external screens so they do not linger off-screen
    if (detached.length > 0) {
      await platformService.closeAllDetachedWindows();
    }

    setConsolidatedPanelIds(newPanelIds);
    setIsTravelMode(true);
    setActiveAlert(null);
    showStatus(
      detached.length > 0
        ? `Travel Mode active: Consolidated ${newPanelIds.length} window(s) into tabs`
        : 'Travel Mode active (compact single display)'
    );
  };

  // Reconnect / Exit Travel Mode: Restore detached windows to physical monitors
  const handleRestoreFromTravelMode = async () => {
    // 1. Remove temporary consolidated panels from Dockview
    if (dockApi) {
      for (const panelId of consolidatedPanelIds) {
        try {
          const p = dockApi.getPanel(panelId);
          if (p) {
            dockApi.removePanel(p);
          }
        } catch (e) {
          devWarn('[UniversalWorkspaceHub] Error removing consolidated panel:', e);
        }
      }
    }
    setConsolidatedPanelIds([]);
    setIsTravelMode(false);
    setActiveAlert(null);

    // 2. Restore detached windows to external monitors
    await handleRestoreDesk();
    showStatus('Restored multi-monitor desk windows');
  };

  // Display change polling effect for dynamic monitor disconnect / reconnect
  useEffect(() => {
    const monitorPollInterval = setInterval(async () => {
      try {
        const currentScreens = await platformService.getScreens();
        const currentCount = currentScreens.length;
        const prevCount = prevScreenCountRef.current;

        if (currentCount !== prevCount) {
          prevScreenCountRef.current = currentCount;
          setScreens(currentScreens);

          const detached = layoutManager.getDetachedWindows();

          // Monitor disconnected: dropped from multi-monitor to 1 screen
          if (prevCount > 1 && currentCount === 1 && detached.length > 0 && !isTravelModeRef.current) {
            setActiveAlert({
              type: 'travel-mode',
              title: 'Travel Mode',
              message: `External display disconnected. Consolidate ${detached.length} detached window(s) into tabs?`,
              actionLabel: 'Consolidate to Tabs',
              dismissLabel: 'Keep Hidden',
              onAction: () => handleConsolidateToTabs(),
              onDismiss: () => setActiveAlert(null),
            });
          }
          // Monitor reconnected: increased from 1 to > 1 screens
          else if (prevCount === 1 && currentCount > 1 && (isTravelModeRef.current || detached.length > 0)) {
            setActiveAlert({
              type: 'reconnect',
              title: 'Displays Detected',
              message: `External display reconnected (${currentCount} screens). Restore multi-monitor desk?`,
              actionLabel: 'Restore Multi-Monitor Desk',
              dismissLabel: 'Dismiss',
              onAction: () => handleRestoreFromTravelMode(),
              onDismiss: () => setActiveAlert(null),
            });
          }
        }
      } catch (err) {
        devWarn('[UniversalWorkspaceHub] Display poll error:', err);
      }
    }, 4000);

    return () => clearInterval(monitorPollInterval);
  }, [dockApi, consolidatedPanelIds]);

  // Add individual panel to workspace
  const handleAddPanel = (component: 'orders' | 'rebalancer' | 'scenario' | 'fixed_income', title: string) => {
    if (!dockApi) return;
    const id = `panel_${component}_${Date.now()}`;
    dockApi.addPanel({
      id,
      component,
      title,
    });
  };

  // Pop out panel to external monitor/window
  const handlePopout = async (viewId: string, route: string, title: string, targetScreenIndex: number = 0) => {
    try {
      await platformService.displayView({
        viewId,
        route,
        title,
        targetScreenIndex,
        width: 1400,
        height: 900,
      });

      layoutManager.registerDetachedWindow({
        windowId: `win_${viewId}`,
        route,
        title,
        targetScreenIndex,
        width: 1400,
        height: 900,
      });

      showStatus(`Opened "${title}" on Display ${targetScreenIndex + 1}`);
    } catch (err) {
      console.error('Popout error:', err);
      showStatus(`Failed to open view: ${err}`);
    }
  };

  // Distribute key financial views across physical displays
  const handleDistributeMultiMonitor = async () => {
    const screenList = screens.length > 0 ? screens : await platformService.getScreens();
    const count = screenList.length;

    if (count <= 1) {
      showStatus('Single display detected. Opening popouts as auxiliary windows.');
    } else {
      showStatus(`Distributing trading workspace across ${count} physical displays...`);
    }

    // Screen 1: OMS Orders
    await handlePopout('orders_screen', '/view/page/orders', 'OMS Order Blotter', 0);

    // Screen 2: Rebalancer
    const screenForRebalancer = count > 1 ? 1 : 0;
    await handlePopout('rebalancer_screen', '/view/rebalancer', 'AI Portfolio Rebalancer', screenForRebalancer);

    // Screen 3: Scenario Analysis
    if (count > 2) {
      await handlePopout('scenario_screen', '/view/scenario', 'Scenario Analysis Pro', 2);
    }
  };

  const btnStyle: React.CSSProperties = {
    padding: '4px 8px',
    background: '#1e293b',
    color: '#94a3b8',
    border: '1px solid #334155',
    borderRadius: '4px',
    cursor: 'pointer',
    fontSize: '11px',
    fontWeight: 500,
  };

  const secondaryBtnStyle: React.CSSProperties = {
    ...btnStyle,
    background: '#0f172a',
    color: '#cbd5e1',
  };

  return (
    <div style={{ height: '100vh', display: 'flex', flexDirection: 'column', background: '#050d1a' }}>
      {/* Unified Multi-Monitor & Travel Mode Alert Strip */}
      {activeAlert && (
        <div style={{
          background: activeAlert.type === 'travel-mode' ? '#7c2d12' : activeAlert.type === 'reconnect' ? '#065f46' : '#0369a1',
          color: '#ffffff',
          padding: '8px 16px',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          fontSize: '12px',
          fontWeight: 500,
          borderBottom: '1px solid rgba(255, 255, 255, 0.1)',
        }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <span style={{ fontSize: '14px' }}>
              {activeAlert.type === 'travel-mode' ? '✈️' : activeAlert.type === 'reconnect' ? '🖥️' : '🪟'}
            </span>
            <span><strong>{activeAlert.title}:</strong> {activeAlert.message}</span>
          </div>
          <div style={{ display: 'flex', gap: '8px' }}>
            <button
              onClick={activeAlert.onAction}
              style={{
                padding: '4px 12px',
                background: '#ffffff',
                color: '#0f172a',
                border: 'none',
                borderRadius: '4px',
                cursor: 'pointer',
                fontWeight: 600,
                fontSize: '11px',
              }}
            >
              {activeAlert.actionLabel}
            </button>
            <button
              onClick={activeAlert.onDismiss}
              style={{
                padding: '4px 8px',
                background: 'transparent',
                color: '#e2e8f0',
                border: '1px solid rgba(255, 255, 255, 0.4)',
                borderRadius: '4px',
                cursor: 'pointer',
                fontSize: '11px',
              }}
            >
              {activeAlert.dismissLabel}
            </button>
          </div>
        </div>
      )}

      {/* Workspace Hub Header Toolbar */}
      <header
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '8px 16px',
          background: '#091428',
          borderBottom: '1px solid #1e293b',
          flexShrink: 0,
          gap: '16px',
          flexWrap: 'wrap',
        }}
      >
        {/* Title & Platform Tag */}
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
          <span style={{ fontSize: '13px', fontWeight: 700, letterSpacing: '0.05em', color: '#e2e8f0' }}>
            UNIVERSAL WORKSPACE
          </span>
          <span
            style={{
              padding: '2px 8px',
              borderRadius: '4px',
              fontSize: '11px',
              fontWeight: 600,
              background: isDesktop ? '#1e3a8a' : '#334155',
              color: isDesktop ? '#93c5fd' : '#cbd5e1',
            }}
          >
            {isDesktop ? 'Wails v3 Desktop' : 'Web Browser'}
          </span>
          <span
            data-testid="heartbeat-display-badge"
            style={{
              padding: '2px 8px',
              borderRadius: '4px',
              fontSize: '11px',
              background: '#0f172a',
              border: '1px solid #1e293b',
              color: '#94a3b8',
            }}
            title={`Active Windows: ${heartbeat.windowCount} (${heartbeat.openWindowIds.join(', ')}) | Active Vault Leases: ${heartbeat.vaultTokenCount}`}
          >
            🖥️ {heartbeat.screenCount} Display{heartbeat.screenCount === 1 ? '' : 's'} | 🪟 {heartbeat.windowCount} Win{heartbeat.windowCount === 1 ? '' : 's'} | 🔒 {heartbeat.vaultTokenCount} Lease{heartbeat.vaultTokenCount === 1 ? '' : 's'}
          </span>

          {/* FDC3 User Channel Selector */}
          <div style={{ display: 'flex', alignItems: 'center', gap: '4px', marginLeft: '8px' }}>
            <span style={{ fontSize: '11px', color: '#64748b' }}>Channel:</span>
            <select
              value={activeChannel}
              onChange={(e) => setChannel(e.target.value as UserChannelId)}
              style={{
                background: '#0f172a',
                color: '#38bdf8',
                border: '1px solid #1e293b',
                borderRadius: '4px',
                padding: '2px 6px',
                fontSize: '11px',
                fontWeight: 600,
                cursor: 'pointer',
              }}
              title="Select active FDC3 context channel"
            >
              {availableChannels.map((c) => (
                <option key={c} value={c}>{c.toUpperCase()}</option>
              ))}
            </select>
          </div>

          {statusMessage && (
            <span style={{ fontSize: '12px', color: '#22c55e', marginLeft: '8px', animation: 'fadeIn 0.2s' }}>
              ✓ {statusMessage}
            </span>
          )}
        </div>

        {/* Action Controls */}
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          <button
            onClick={handleDistributeMultiMonitor}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: '6px',
              padding: '6px 12px',
              background: '#2563eb',
              color: '#ffffff',
              border: 'none',
              borderRadius: '4px',
              cursor: 'pointer',
              fontSize: '12px',
              fontWeight: 600,
            }}
            title="Distribute views across detected physical displays"
          >
            <span>⛶</span> Distribute to Multi-Monitor
          </button>

          <button
            data-testid="command-bar-trigger"
            onClick={() => setIsCommandBarOpen(true)}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: '6px',
              padding: '6px 12px',
              background: '#1e293b',
              color: '#38bdf8',
              border: '1px solid #334155',
              borderRadius: '4px',
              cursor: 'pointer',
              fontSize: '12px',
              fontWeight: 600,
            }}
            title="Open Internal Command Bar (Cmd+K / Ctrl+K)"
          >
            <span>⌘K</span> Commands
          </button>

          {/* Add Views */}
          <div style={{ display: 'flex', gap: '4px', borderLeft: '1px solid #1e293b', paddingLeft: '8px' }}>
            <button
              onClick={() => handleAddPanel('orders', 'OMS Order Blotter')}
              style={btnStyle}
              title="Add Order Blotter"
            >
              + Orders
            </button>
            <button
              onClick={() => handleAddPanel('rebalancer', 'AI Portfolio Rebalancer')}
              style={btnStyle}
              title="Add Rebalancer"
            >
              + Rebalancer
            </button>
            <button
              onClick={() => handleAddPanel('scenario', 'Scenario Analysis Pro')}
              style={btnStyle}
              title="Add Scenario Analysis"
            >
              + Scenario
            </button>
            <button
              onClick={() => handleAddPanel('fixed_income', 'Fixed Income Analytics')}
              style={btnStyle}
              title="Add Fixed Income"
            >
              + Fixed Income
            </button>
          </div>

          {/* Layout Controls */}
          <div style={{ display: 'flex', gap: '4px', borderLeft: '1px solid #1e293b', paddingLeft: '8px' }}>
            <button onClick={handleSaveLayout} style={secondaryBtnStyle} title="Save current workspace layout">
              Save
            </button>
            <button onClick={handleRestoreDesk} style={secondaryBtnStyle} title="Restore workspace layout & detached windows">
              Restore Desk
            </button>
            <button
              onClick={isTravelMode ? handleRestoreFromTravelMode : handleConsolidateToTabs}
              style={{
                ...secondaryBtnStyle,
                color: isTravelMode ? '#34d399' : '#fbbf24',
                borderColor: isTravelMode ? '#059669' : '#d97706',
              }}
              title={isTravelMode ? 'Restore multi-monitor desk windows' : 'Consolidate detached windows to tabs (Travel Mode)'}
            >
              {isTravelMode ? '✈️ Exit Travel' : '✈️ Travel Mode'}
            </button>
            <button onClick={handleResetLayout} style={secondaryBtnStyle} title="Reset layout to default">
              Reset
            </button>
          </div>
        </div>
      </header>

      {/* Dockview Container */}
      <div style={{ flexGrow: 1, position: 'relative', width: '100%', height: '100%', minHeight: 0 }}>
        <DockviewReact
          className="dockview-theme-uisce"
          components={dockComponents}
          onReady={onReady}
        />
      </div>

      {/* FDC3 Intent Resolver Modal */}
      <IntentResolverModal
        isOpen={resolverModal.isOpen}
        intent={resolverModal.intent}
        context={resolverModal.context}
        targets={resolverModal.targets}
        onSelect={(target) => {
          if (resolverModal.resolve) {
            resolverModal.resolve(target);
          }
          setResolverModal({ isOpen: false, intent: null, context: null, targets: [] });
        }}
        onCancel={() => {
          if (resolverModal.reject) {
            resolverModal.reject(new Error('Intent resolution cancelled by user'));
          }
          setResolverModal({ isOpen: false, intent: null, context: null, targets: [] });
        }}
      />

      {/* Internal Command Bar (Cmd+K / Ctrl+K) */}
      <WorkstationCommandBar
        isOpen={isCommandBarOpen}
        onClose={() => setIsCommandBarOpen(false)}
        onSelectIntent={(intent, ticker) => {
          fdc3Agent.raiseIntent(intent, {
            type: 'fdc3.instrument',
            id: { ticker: ticker || 'AAPL' },
            name: `${ticker || 'AAPL'} Equity`,
          });
        }}
        onApplyLayout={(action) => {
          switch (action) {
            case 'save':
              handleSaveLayout();
              break;
            case 'restore':
              handleRestoreDesk();
              break;
            case 'travel':
              if (isTravelMode) {
                handleRestoreFromTravelMode();
              } else {
                handleConsolidateToTabs();
              }
              break;
            case 'reset':
              handleResetLayout();
              break;
          }
        }}
      />
    </div>
  );
};

export default UniversalWorkspaceHub;
