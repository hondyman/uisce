import React, { useEffect, useRef, useState, useCallback } from 'react';
import { LatestQuoteTable } from '../../services/streaming/LatestQuoteTable';
import { TickStreamConsumer } from '../../services/streaming/TickStreamConsumer';
import { SyntheticTickGenerator, DEFAULT_TICK_UNIVERSE } from '../../services/streaming/SyntheticTickGenerator';
import { QuoteRow, TickSource } from '../../services/streaming/types';
import { useFdc3 } from '../../services/fdc3/useFdc3';

export interface HighDensityCanvasGridProps {
  customSource?: TickSource;
  universe?: string[];
  rowHeight?: number;
  onSelectSymbol?: (symbol: string) => void;
}

const COLUMN_DEFS = [
  { key: 'symbol', label: 'SYMBOL', width: 90, align: 'left' as const },
  { key: 'price', label: 'LAST', width: 85, align: 'right' as const },
  { key: 'change', label: 'CHG', width: 75, align: 'right' as const },
  { key: 'changePct', label: 'CHG %', width: 75, align: 'right' as const },
  { key: 'bid', label: 'BID', width: 85, align: 'right' as const },
  { key: 'ask', label: 'ASK', width: 85, align: 'right' as const },
  { key: 'volume', label: 'VOLUME', width: 95, align: 'right' as const },
  { key: 'high', label: 'HIGH', width: 80, align: 'right' as const },
  { key: 'low', label: 'LOW', width: 80, align: 'right' as const },
];

export const HighDensityCanvasGrid: React.FC<HighDensityCanvasGridProps> = ({
  customSource,
  universe = DEFAULT_TICK_UNIVERSE,
  rowHeight = 24,
  onSelectSymbol,
}) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);

  // React Render Counter Ref: Proves zero React re-renders from incoming ticks
  const renderCountRef = useRef<number>(0);
  renderCountRef.current++;

  const [isDegraded, setIsDegraded] = useState<boolean>(false);
  const [stats, setStats] = useState({ msgsSec: 0, coalesced: 0, p95: 0 });
  const [selectedSymbol, setSelectedSymbol] = useState<string | null>(null);
  const selectedSymbolRef = useRef<string | null>(null);
  selectedSymbolRef.current = selectedSymbol;

  // FDC3 context broadcast on user row selection (loop-guard compliant)
  const { broadcast } = useFdc3();

  // Engine refs outside React component state
  const quoteTableRef = useRef<LatestQuoteTable>(new LatestQuoteTable(universe));
  const consumerRef = useRef<TickStreamConsumer | null>(null);
  const animFrameIdRef = useRef<number | null>(null);
  const scrollTopRef = useRef<number>(0);

  const handleRowClick = useCallback((symbol: string) => {
    setSelectedSymbol(symbol);
    selectedSymbolRef.current = symbol;
    onSelectSymbol?.(symbol);
    // Broadcast user selection over FDC3 context bus
    broadcast({
      type: 'fdc3.instrument',
      id: { ticker: symbol },
      name: `${symbol} Equity`,
    });
  }, [broadcast, onSelectSymbol]);

  useEffect(() => {
    const table = quoteTableRef.current;
    const source = customSource || new SyntheticTickGenerator({
      symbols: universe,
      targetRatePerSec: 5000,
    });

    const consumer = new TickStreamConsumer({
      source,
      quoteTable: table,
    });

    consumer.setDegradedCallback((degraded) => {
      setIsDegraded(degraded);
    });

    consumer.start();
    consumerRef.current = consumer;

    // Canvas animation paint loop
    let lastFrameTime = performance.now();

    const renderLoop = (now: number) => {
      const frameDelta = now - lastFrameTime;
      lastFrameTime = now;

      // Report frame duration to consumer for hysteresis backpressure management
      consumer.reportFrameTime(frameDelta);

      const canvas = canvasRef.current;
      const ctx = canvas?.getContext('2d');
      if (canvas && ctx) {
        paintCanvas(canvas, ctx, table, rowHeight, scrollTopRef.current, selectedSymbolRef.current);
      }

      animFrameIdRef.current = requestAnimationFrame(renderLoop);
    };

    animFrameIdRef.current = requestAnimationFrame(renderLoop);

    // Periodically update UI stats badge (once every 1s, distinct from high-freq ticks)
    const statsTimer = setInterval(() => {
      const m = consumer.getMetrics();
      setStats({
        msgsSec: m.ticksPerSecond,
        coalesced: m.coalescedTicks,
        p95: m.p95LatencyMs,
      });
    }, 1000);

    return () => {
      if (animFrameIdRef.current) cancelAnimationFrame(animFrameIdRef.current);
      clearInterval(statsTimer);
      consumer.stop();
    };
  }, [customSource, universe, rowHeight]);

  // Handle Canvas Resizing
  useEffect(() => {
    const updateCanvasSize = () => {
      const canvas = canvasRef.current;
      const container = containerRef.current;
      if (!canvas || !container) return;

      const dpr = window.devicePixelRatio || 1;
      const rect = container.getBoundingClientRect();
      canvas.width = rect.width * dpr;
      canvas.height = (rect.height - 36) * dpr; // minus header height
      canvas.style.width = `${rect.width}px`;
      canvas.style.height = `${rect.height - 36}px`;

      const ctx = canvas.getContext('2d');
      if (ctx) {
        // Reset transform before scaling to prevent scale accumulation on repeated resize
        ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
      }
    };

    updateCanvasSize();
    window.addEventListener('resize', updateCanvasSize);
    return () => window.removeEventListener('resize', updateCanvasSize);
  }, []);

  const handleCanvasClick = (e: React.MouseEvent<HTMLCanvasElement>) => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const rect = canvas.getBoundingClientRect();
    const y = e.clientY - rect.top + scrollTopRef.current;
    const rowIndex = Math.floor(y / rowHeight);
    const rows = quoteTableRef.current.getAllRows();
    if (rowIndex >= 0 && rowIndex < rows.length) {
      handleRowClick(rows[rowIndex].symbol);
    }
  };

  const handleScroll = (e: React.UIEvent<HTMLDivElement>) => {
    scrollTopRef.current = e.currentTarget.scrollTop;
  };

  return (
    <div
      ref={containerRef}
      style={{
        width: '100%',
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
        background: '#040d1a',
        color: '#e2e8f0',
        fontFamily: 'JetBrains Mono, ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
        overflow: 'hidden',
        position: 'relative',
        userSelect: 'none',
      }}
    >
      {/* Top Status & Metrics Bar */}
      <div
        style={{
          height: '36px',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '0 12px',
          background: '#0a1628',
          borderBottom: '1px solid #1e293b',
          fontSize: '11px',
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
          <span style={{ fontWeight: 700, color: '#38bdf8', letterSpacing: '0.05em' }}>
            HIGH-DENSITY STREAMING BLOTTER
          </span>
          <span style={{ color: '#64748b' }}>
            {universe.length} Instruments | ~5,000 msg/sec
          </span>
          {isDegraded && (
            <span
              data-testid="degraded-badge"
              style={{
                background: '#dc2626',
                color: '#ffffff',
                padding: '2px 8px',
                borderRadius: '4px',
                fontWeight: 700,
                fontSize: '10px',
                animation: 'pulse 1.5s infinite',
              }}
            >
              ⚡ DEGRADED: COALESCING TICKS
            </span>
          )}
        </div>

        <div style={{ display: 'flex', alignItems: 'center', gap: '14px', color: '#94a3b8' }}>
          <span>Throughput: <strong style={{ color: '#34d399' }}>{stats.msgsSec.toLocaleString()}</strong> msg/s</span>
          <span>Coalesced: <strong style={{ color: '#fbbf24' }}>{stats.coalesced.toLocaleString()}</strong></span>
          <span>p95 Latency: <strong style={{ color: '#cbd5e1' }}>{stats.p95}ms</strong></span>
          <span>React Renders: <strong data-testid="render-counter" style={{ color: '#60a5fa' }}>{renderCountRef.current}</strong></span>
        </div>
      </div>

      {/* Grid Headers */}
      <div
        style={{
          height: '24px',
          display: 'flex',
          background: '#091322',
          borderBottom: '1px solid #1e293b',
          fontSize: '10px',
          fontWeight: 600,
          color: '#64748b',
          alignItems: 'center',
        }}
      >
        {COLUMN_DEFS.map((col) => (
          <div
            key={col.key}
            style={{
              width: col.width,
              textAlign: col.align,
              padding: '0 8px',
              boxSizing: 'border-box',
            }}
          >
            {col.label}
          </div>
        ))}
      </div>

      {/* Virtual Scroll Container containing HTML5 Canvas */}
      <div
        onScroll={handleScroll}
        style={{
          flexGrow: 1,
          overflowY: 'auto',
          position: 'relative',
        }}
      >
        {/* Phantom scroll height spacer */}
        <div style={{ height: universe.length * rowHeight, width: '100%', position: 'absolute', top: 0, left: 0 }} />

        <canvas
          ref={canvasRef}
          onClick={handleCanvasClick}
          style={{
            position: 'sticky',
            top: 0,
            left: 0,
            display: 'block',
            cursor: 'pointer',
          }}
        />
      </div>
    </div>
  );
};

export function paintCanvas(
  canvas: HTMLCanvasElement,
  ctx: CanvasRenderingContext2D,
  table: LatestQuoteTable,
  rowHeight: number,
  scrollTop: number,
  selectedSymbol: string | null
) {
  const width = canvas.width / (window.devicePixelRatio || 1);
  const height = canvas.height / (window.devicePixelRatio || 1);

  ctx.clearRect(0, 0, width, height);

  const rows = table.getAllRows();
  const startRow = Math.max(0, Math.floor(scrollTop / rowHeight));
  const visibleCount = Math.ceil(height / rowHeight) + 1;
  const endRow = Math.min(rows.length, startRow + visibleCount);

  for (let i = startRow; i < endRow; i++) {
    const row: QuoteRow = rows[i];
    const y = i * rowHeight - scrollTop;

    // Row Background
    if (row.symbol === selectedSymbol) {
      ctx.fillStyle = '#1e3a5f';
    } else if (i % 2 === 0) {
      ctx.fillStyle = '#040d1a';
    } else {
      ctx.fillStyle = '#071224';
    }
    ctx.fillRect(0, y, width, rowHeight);

    // Bottom border
    ctx.fillStyle = '#0f1d33';
    ctx.fillRect(0, y + rowHeight - 1, width, 1);

    // Paint cells
    let x = 0;
    ctx.font = '11px JetBrains Mono, monospace';
    ctx.textBaseline = 'middle';

    const textY = y + rowHeight / 2;

    for (const col of COLUMN_DEFS) {
      let val = '';
      let color = '#cbd5e1';

      switch (col.key) {
        case 'symbol':
          val = row.symbol;
          color = '#38bdf8';
          break;
        case 'price':
          val = row.price.toFixed(2);
          color = row.tickDirection === 'up' ? '#34d399' : row.tickDirection === 'down' ? '#f87171' : '#f1f5f9';
          break;
        case 'change':
          val = (row.change >= 0 ? '+' : '') + row.change.toFixed(2);
          color = row.change >= 0 ? '#34d399' : '#f87171';
          break;
        case 'changePct':
          val = (row.changePct >= 0 ? '+' : '') + row.changePct.toFixed(2) + '%';
          color = row.changePct >= 0 ? '#34d399' : '#f87171';
          break;
        case 'bid':
          val = row.bid.toFixed(2);
          color = '#94a3b8';
          break;
        case 'ask':
          val = row.ask.toFixed(2);
          color = '#94a3b8';
          break;
        case 'volume':
          val = row.volume.toLocaleString();
          color = '#64748b';
          break;
        case 'high':
          val = row.high.toFixed(2);
          color = '#64748b';
          break;
        case 'low':
          val = row.low.toFixed(2);
          color = '#64748b';
          break;
      }

      ctx.fillStyle = color;
      if (col.align === 'right') {
        ctx.textAlign = 'right';
        ctx.fillText(val, x + col.width - 8, textY);
      } else {
        ctx.textAlign = 'left';
        ctx.fillText(val, x + 8, textY);
      }

      x += col.width;
    }
  }
}
