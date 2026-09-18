import React, { useState, useEffect, useRef, useMemo } from 'react';
import { fdc3Agent, USER_CHANNELS } from '../../services/fdc3';
import { platformService } from '../../services/platform/PlatformService';
import { StandardIntent } from '../../services/fdc3/intentTypes';
import { searchInstruments, InstrumentSearchResultItem } from '../../services/api/instrumentsApi';

export interface CommandItem {
  id: string;
  category: 'Intent' | 'Channel' | 'Layout' | 'Instrument';
  title: string;
  subtitle?: string;
  badge?: string;
  badgeColor?: string;
  onSelect: () => void;
}

export interface WorkstationCommandBarProps {
  isOpen: boolean;
  onClose: () => void;
  onSelectIntent?: (intent: StandardIntent, contextTicker?: string) => void;
  onApplyLayout?: (action: 'save' | 'restore' | 'travel' | 'reset') => void;
}

export const MOCK_INSTRUMENTS: Array<{ symbol: string; name: string; type: string; isin?: string }> = [
  { symbol: 'AAPL', name: 'Apple Inc.', type: 'Equity', isin: 'US0378331005' },
  { symbol: 'MSFT', name: 'Microsoft Corporation', type: 'Equity', isin: 'US5949181045' },
  { symbol: 'NVDA', name: 'NVIDIA Corporation', type: 'Equity', isin: 'US67066G1040' },
  { symbol: 'GOOGL', name: 'Alphabet Inc.', type: 'Equity', isin: 'US02079K3059' },
  { symbol: 'TSLA', name: 'Tesla Inc.', type: 'Equity', isin: 'US88160R1014' },
  { symbol: 'US10Y', name: 'US 10-Year Treasury Yield', type: 'Fixed Income' },
  { symbol: 'US02Y', name: 'US 2-Year Treasury Yield', type: 'Fixed Income' },
  { symbol: 'BND', name: 'Vanguard Total Bond Market ETF', type: 'Fixed Income', isin: 'US9219378356' },
];

export const WorkstationCommandBar: React.FC<WorkstationCommandBarProps> = ({
  isOpen,
  onClose,
  onSelectIntent,
  onApplyLayout,
}) => {
  const [query, setQuery] = useState('');
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [liveInstruments, setLiveInstruments] = useState<InstrumentSearchResultItem[] | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const abortControllerRef = useRef<AbortController | null>(null);
  const isWails = platformService.isWails();

  useEffect(() => {
    if (isOpen) {
      setQuery('');
      setSelectedIndex(0);
      setLiveInstruments(null);
      setIsLoading(false);
      setTimeout(() => inputRef.current?.focus(), 50);
    } else {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
        abortControllerRef.current = null;
      }
    }
  }, [isOpen]);

  // Live instrument search with 200ms debounce and AbortController cancellation
  useEffect(() => {
    const trimmed = query.trim();
    if (!trimmed) {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
        abortControllerRef.current = null;
      }
      setLiveInstruments(null);
      setIsLoading(false);
      return;
    }

    // Abort any prior in-flight request immediately upon query change
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
      abortControllerRef.current = null;
    }

    setIsLoading(true);
    const timer = setTimeout(async () => {
      const controller = new AbortController();
      abortControllerRef.current = controller;

      try {
        const results = await searchInstruments(trimmed, {
          limit: 10,
          signal: controller.signal,
        });
        setLiveInstruments(results);
      } catch (err: unknown) {
        if (err instanceof DOMException && err.name === 'AbortError') {
          return; // Stale request was aborted
        }
        // Fallback gracefully on network error or offline
        setLiveInstruments(null);
      } finally {
        if (abortControllerRef.current === controller) {
          setIsLoading(false);
          abortControllerRef.current = null;
        }
      }
    }, 200);

    return () => {
      clearTimeout(timer);
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
        abortControllerRef.current = null;
      }
    };
  }, [query]);

  // Static Local System Commands (Intents, Channels, Layout)
  const systemCommands: CommandItem[] = useMemo(() => {
    const cmds: CommandItem[] = [];

    // 1. Intents
    cmds.push(
      {
        id: 'intent-rebalancer',
        category: 'Intent',
        title: 'View Analysis: AI Portfolio Rebalancer',
        subtitle: 'Raise ViewAnalysis intent targeted to AIPortfolioRebalancer view',
        badge: 'Intent',
        badgeColor: '#38bdf8',
        onSelect: () => {
          if (onSelectIntent) {
            onSelectIntent('ViewAnalysis', 'AAPL');
          } else {
            fdc3Agent.raiseIntent('ViewAnalysis', { type: 'fdc3.instrument', id: { ticker: 'AAPL' } });
          }
        },
      },
      {
        id: 'intent-scenario',
        category: 'Intent',
        title: 'View Analysis: Scenario Analysis Pro',
        subtitle: 'Raise ViewAnalysis intent targeted to ScenarioAnalysisPro view',
        badge: 'Intent',
        badgeColor: '#38bdf8',
        onSelect: () => {
          if (onSelectIntent) {
            onSelectIntent('ViewAnalysis', 'NVDA');
          } else {
            fdc3Agent.raiseIntent('ViewAnalysis', { type: 'fdc3.instrument', id: { ticker: 'NVDA' } });
          }
        },
      },
      {
        id: 'intent-fixed-income',
        category: 'Intent',
        title: 'View Instrument: Fixed Income Analytics',
        subtitle: 'Raise ViewInstrument intent targeted to Fixed Income dashboard',
        badge: 'Intent',
        badgeColor: '#38bdf8',
        onSelect: () => {
          if (onSelectIntent) {
            onSelectIntent('ViewInstrument', 'US10Y');
          } else {
            fdc3Agent.raiseIntent('ViewInstrument', { type: 'fdc3.instrument', id: { ticker: 'US10Y' } });
          }
        },
      },
      {
        id: 'intent-orders',
        category: 'Intent',
        title: 'View Orders: OMS Order Blotter',
        subtitle: 'Raise ViewOrders intent filtered by focused symbol',
        badge: 'Intent',
        badgeColor: '#38bdf8',
        onSelect: () => {
          if (onSelectIntent) {
            onSelectIntent('ViewOrders', 'AAPL');
          } else {
            fdc3Agent.raiseIntent('ViewOrders', { type: 'fdc3.instrument', id: { ticker: 'AAPL' } });
          }
        },
      }
    );

    // 2. FDC3 Color Channels
    for (const ch of USER_CHANNELS) {
      cmds.push({
        id: `channel-${ch.id}`,
        category: 'Channel',
        title: `Join Channel: ${ch.name}`,
        subtitle: `Switch current window to FDC3 ${ch.id} channel context`,
        badge: ch.id.toUpperCase(),
        badgeColor: ch.color,
        onSelect: () => {
          fdc3Agent.joinUserChannel(ch.id);
        },
      });
    }

    // 3. Layout Presets & Travel Mode
    cmds.push(
      {
        id: 'layout-save',
        category: 'Layout',
        title: 'Save Current Workspace Layout',
        subtitle: 'Persist current Dockview layout and multi-monitor window bounds',
        badge: 'Layout',
        badgeColor: '#10b981',
        onSelect: () => onApplyLayout?.('save'),
      },
      {
        id: 'layout-restore',
        category: 'Layout',
        title: 'Restore Multi-Monitor Desk',
        subtitle: 'Re-spawn and re-clamp detached windows across all physical monitors',
        badge: 'Layout',
        badgeColor: '#10b981',
        onSelect: () => onApplyLayout?.('restore'),
      },
      {
        id: 'layout-travel',
        category: 'Layout',
        title: 'Toggle Travel Mode',
        subtitle: 'Consolidate detached windows into primary tabs (or restore)',
        badge: 'Mode',
        badgeColor: '#f59e0b',
        onSelect: () => onApplyLayout?.('travel'),
      },
      {
        id: 'layout-reset',
        category: 'Layout',
        title: 'Reset Workspace to Default',
        subtitle: 'Clear customized layout and return to default multi-panel grid',
        badge: 'Reset',
        badgeColor: '#ef4444',
        onSelect: () => onApplyLayout?.('reset'),
      }
    );

    return cmds;
  }, [onSelectIntent, onApplyLayout]);

  // Combine system commands with instrument items (live or fallback)
  const allCommands: CommandItem[] = useMemo(() => {
    const cmds: CommandItem[] = [...systemCommands];

    if (liveInstruments !== null) {
      // Use live API search results
      for (const inst of liveInstruments) {
        cmds.push({
          id: `inst-${inst.symbol}`,
          category: 'Instrument',
          title: `${inst.symbol} — ${inst.name}`,
          subtitle: `Broadcast ${inst.symbol} onto active FDC3 channel (${inst.asset_class}${inst.isin ? ` · ${inst.isin}` : ''})`,
          badge: inst.symbol,
          badgeColor: '#6366f1',
          onSelect: () => {
            const idObj: { ticker: string; ISIN?: string } = { ticker: inst.symbol };
            if (inst.isin) {
              idObj.ISIN = inst.isin;
            }
            fdc3Agent.broadcast({
              type: 'fdc3.instrument',
              id: idObj,
              name: inst.name,
            });
          },
        });
      }
    } else {
      // Static fallback instruments
      for (const inst of MOCK_INSTRUMENTS) {
        cmds.push({
          id: `inst-${inst.symbol}`,
          category: 'Instrument',
          title: `${inst.symbol} — ${inst.name}`,
          subtitle: `Broadcast ${inst.symbol} onto active FDC3 channel (${inst.type})`,
          badge: inst.symbol,
          badgeColor: '#6366f1',
          onSelect: () => {
            const idObj: { ticker: string; ISIN?: string } = { ticker: inst.symbol };
            if (inst.isin) {
              idObj.ISIN = inst.isin;
            }
            fdc3Agent.broadcast({
              type: 'fdc3.instrument',
              id: idObj,
              name: inst.name,
            });
          },
        });
      }
    }

    return cmds;
  }, [systemCommands, liveInstruments]);

  // Filter commands by query (system commands filter client-side; live instruments are already filtered)
  const filtered = useMemo(() => {
    if (!query.trim()) return allCommands;
    const q = query.toLowerCase();
    return allCommands.filter((cmd) => {
      // If live instruments are active and this is an instrument, keep it since the server matched it
      if (liveInstruments !== null && cmd.category === 'Instrument') {
        return true;
      }
      return (
        cmd.title.toLowerCase().includes(q) ||
        cmd.category.toLowerCase().includes(q) ||
        cmd.subtitle?.toLowerCase().includes(q)
      );
    });
  }, [allCommands, query, liveInstruments]);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Escape') {
      e.preventDefault();
      onClose();
    } else if (e.key === 'ArrowDown') {
      e.preventDefault();
      setSelectedIndex((prev) => (filtered.length ? (prev + 1) % filtered.length : 0));
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      setSelectedIndex((prev) => (filtered.length ? (prev - 1 + filtered.length) % filtered.length : 0));
    } else if (e.key === 'Enter') {
      e.preventDefault();
      if (filtered[selectedIndex]) {
        filtered[selectedIndex].onSelect();
        onClose();
      }
    }
  };

  if (!isOpen) return null;

  return (
    <div
      data-testid="command-bar-overlay"
      onClick={onClose}
      style={{
        position: 'fixed',
        top: 0,
        left: 0,
        right: 0,
        bottom: 0,
        background: 'rgba(2, 6, 23, 0.75)',
        backdropFilter: 'blur(4px)',
        zIndex: 99999,
        display: 'flex',
        alignItems: 'flex-start',
        justifyContent: 'center',
        paddingTop: '60px', // clears top caption bar in popouts
      }}
    >
      <div
        data-testid="command-bar-modal"
        onClick={(e) => e.stopPropagation()}
        style={{
          width: '580px',
          maxWidth: '92vw',
          maxHeight: '480px',
          background: '#0b1329',
          border: '1px solid #1e293b',
          borderRadius: '8px',
          boxShadow: '0 25px 50px -12px rgba(0, 0, 0, 0.7)',
          display: 'flex',
          flexDirection: 'column',
          overflow: 'hidden',
          fontFamily: 'Inter, system-ui, sans-serif',
        }}
      >
        {/* Input Bar */}
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            padding: '12px 16px',
            borderBottom: '1px solid #1e293b',
            background: '#070d1e',
          }}
        >
          <span style={{ color: '#64748b', marginRight: '12px', fontSize: '15px' }}>
            {isLoading ? '⏳' : '🔍'}
          </span>
          <input
            ref={inputRef}
            data-testid="command-bar-input"
            type="text"
            value={query}
            onChange={(e) => {
              setQuery(e.target.value);
              setSelectedIndex(0);
            }}
            onKeyDown={handleKeyDown}
            placeholder="Type a command, intent, channel, or ticker..."
            style={{
              flexGrow: 1,
              background: 'transparent',
              border: 'none',
              outline: 'none',
              color: '#f8fafc',
              fontSize: '14px',
            }}
          />
          <span
            style={{
              fontSize: '11px',
              padding: '2px 6px',
              borderRadius: '4px',
              background: '#1e293b',
              color: '#94a3b8',
            }}
          >
            ESC to close
          </span>
        </div>

        {/* Results List */}
        <div style={{ overflowY: 'auto', flexGrow: 1, padding: '6px 0' }}>
          {filtered.length === 0 ? (
            <div style={{ padding: '24px', textAlign: 'center', color: '#64748b', fontSize: '13px' }}>
              No matching commands found.
            </div>
          ) : (
            filtered.map((cmd, idx) => {
              const isSelected = idx === selectedIndex;
              return (
                <div
                  key={cmd.id}
                  data-testid={`command-item-${cmd.id}`}
                  onClick={() => {
                    cmd.onSelect();
                    onClose();
                  }}
                  onMouseEnter={() => setSelectedIndex(idx)}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    padding: '8px 16px',
                    cursor: 'pointer',
                    background: isSelected ? '#1e293b' : 'transparent',
                    borderLeft: isSelected ? '3px solid #38bdf8' : '3px solid transparent',
                  }}
                >
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '2px' }}>
                    <span style={{ fontSize: '13px', fontWeight: 500, color: isSelected ? '#ffffff' : '#e2e8f0' }}>
                      {cmd.title}
                    </span>
                    {cmd.subtitle && (
                      <span style={{ fontSize: '11px', color: '#64748b' }}>
                        {cmd.subtitle}
                      </span>
                    )}
                  </div>
                  {cmd.badge && (
                    <span
                      style={{
                        fontSize: '10px',
                        fontWeight: 700,
                        padding: '2px 6px',
                        borderRadius: '4px',
                        background: `${cmd.badgeColor || '#38bdf8'}22`,
                        color: cmd.badgeColor || '#38bdf8',
                        border: `1px solid ${cmd.badgeColor || '#38bdf8'}44`,
                      }}
                    >
                      {cmd.badge}
                    </span>
                  )}
                </div>
              );
            })
          )}
        </div>

        {/* Footer with Shortcut Help (Respects Asymmetry) */}
        <div
          style={{
            padding: '8px 16px',
            background: '#070d1e',
            borderTop: '1px solid #1e293b',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            fontSize: '11px',
            color: '#64748b',
          }}
        >
          <span>Navigate: <strong>↑ ↓</strong> | Select: <strong>Enter</strong></span>
          <span>
            {isWails ? (
              <span>Desktop Windows: <strong>Cmd+1..9</strong></span>
            ) : (
              <span>Browser Tabs: <strong>Alt+1..9</strong></span>
            )}
          </span>
        </div>
      </div>
    </div>
  );
};
