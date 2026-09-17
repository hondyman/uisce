import { describe, it, expect } from 'vitest';
import { runSpikeBenchmark } from '../../services/streaming/benchmark_spike';

describe('High-Density Streaming Spike Benchmark', () => {
  it('sustains high-throughput streaming with real Canvas rendering, 0 React renders, and controlled latency', async () => {
    // Real render counter ref analog (isolated from rAF canvas loop)
    const renderCounterRef = { current: 0 };
    const result = await runSpikeBenchmark(2000, renderCounterRef);

    console.log('\n--- SPRINT 3 HIGH-DENSITY STREAMING SPIKE BENCHMARK TABLE ---');
    console.table([
      {
        'Metric': 'Total Ticks Ingested',
        'Value': result.totalTicks,
      },
      {
        'Metric': 'Sustained Rate (msgs/sec)',
        'Value': result.sustainedRate,
      },
      {
        'Metric': 'Table Mutations (Applied)',
        'Value': result.tableUpdates,
      },
      {
        'Metric': 'Coalesced Ticks (Backpressure)',
        'Value': result.coalescedTicks,
      },
      {
        'Metric': 'p50 Ingestion Latency (ms)',
        'Value': result.p50LatencyMs,
      },
      {
        'Metric': 'p95 Ingestion Latency (ms)',
        'Value': result.p95LatencyMs,
      },
      {
        'Metric': 'p99 Ingestion Latency (ms)',
        'Value': result.p99LatencyMs,
      },
      {
        'Metric': 'p50 Frame Paint Duration (ms)',
        'Value': `${result.p50FrameTimeMs} ms`,
      },
      {
        'Metric': 'p95 Frame Paint Duration (ms)',
        'Value': `${result.p95FrameTimeMs} ms`,
      },
      {
        'Metric': 'p99 Frame Paint Duration (ms)',
        'Value': `${result.p99FrameTimeMs} ms`,
      },
      {
        'Metric': 'Frames Rendered to Canvas',
        'Value': result.framesRendered,
      },
      {
        'Metric': 'Avg JSON.parse Cost (µs)',
        'Value': `${result.jsonParseCostAvgUs} µs`,
      },
      {
        'Metric': 'React Component Re-Renders',
        'Value': result.reactRenderCount,
      },
      {
        'Metric': 'Degraded Mode Hysteresis Triggered',
        'Value': result.degradedTriggered,
      }
    ]);

    expect(result.totalTicks).toBeGreaterThan(4000);
    expect(result.sustainedRate).toBeGreaterThan(2000);
    expect(result.framesRendered).toBeGreaterThan(30);
    expect(result.p95FrameTimeMs).toBeLessThan(16.67); // Proven under 16.67ms 60fps budget!
    expect(result.reactRenderCount).toBe(0); // Strict acceptance criterion: 0 React re-renders!
    expect(result.degradedTriggered).toBe(true);
    expect(result.coalescedTicks).toBeGreaterThan(0);
    expect(result.tableUpdates).toBeLessThan(result.totalTicks); // Measurable reduction in table mutations!
  });
});
