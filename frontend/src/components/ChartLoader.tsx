import { Suspense, lazy, useState, useEffect } from 'react';
import type { FC, ReactNode } from 'react';

// Re-export the local LazyECharts wrapper (it dynamically imports echarts at runtime).
export { default as LazyECharts } from './LazyECharts';

// Lazy wrapper for Recharts (kept lazy to avoid adding large libs to main bundle)
export const LazyRecharts = lazy(async () => {
  const mod = await import('recharts');
  // prefer the module's default export (if present) otherwise fall back to the module itself
  // the shape is cast to satisfy the lazy contract comfortably without broad `as any` at the call site
  const maybe = (mod as unknown as { default?: unknown }).default ?? mod;
  return { default: maybe } as unknown as { default: any };
});

export const RechartsFallback: FC<{ children?: ReactNode }> = ({ children }) => (
  <Suspense fallback={<div className="chart-loading">Loading chart...</div>}>
    {children}
  </Suspense>
);

export default function ChartLoader() {
  return null; // module only exports lazy components
}

// Concrete lazy-renderer for a simple BarChart to avoid accessing named exports on a lazy component
export const RechartsBarChart: FC<{ data: any[]; xKey: string; yKey: string }> = ({ data, xKey, yKey }) => {
  const [R, setR] = useState<any | null>(null);

  useEffect(() => {
    let mounted = true;
    (async () => {
      const mod = await import('recharts');
      if (mounted) setR(mod);
    })();
    return () => { mounted = false; };
  }, []);

  if (!R) return <div className="chart-loading">Loading chart...</div>;

  const { ResponsiveContainer, BarChart, CartesianGrid, XAxis, YAxis, Tooltip, Legend, Bar } = R;

  return (
    <ResponsiveContainer width="100%" height="100%">
      <BarChart data={data}>
        <CartesianGrid strokeDasharray="3 3" />
        <XAxis dataKey={xKey} />
        <YAxis />
        <Tooltip />
        <Legend />
        <Bar dataKey={yKey} fill="#8884d8" />
      </BarChart>
    </ResponsiveContainer>
  );
};
