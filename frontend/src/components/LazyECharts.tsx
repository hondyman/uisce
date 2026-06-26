import React, { useEffect, useState } from 'react';

type EChartsProps = {
  option: any;
  style?: React.CSSProperties;
  className?: string;
  notMerge?: boolean;
  lazyUpdate?: boolean;
  [key: string]: any;
};

/**
 * LazyECharts
 * Dynamically imports `echarts` + `echarts-for-react` on first mount so the large
 * charting library doesn't land in the initial bundle.
 *
 * The wrapper attempts to import the most granular modules where available
 * (e.g. `echarts/core`) but falls back to the default package. It renders a
 * lightweight placeholder while loading.
 */
export default function LazyECharts(props: EChartsProps) {
  const { option, style, className, ...rest } = props;
  const [Chart, setChart] = useState<React.ComponentType<any> | null>(null);
  const [loadError, setLoadError] = useState<Error | null>(null);

  useEffect(() => {
    let mounted = true;

    (async () => {
      try {
        // Prefer global (CDN-loaded) ECharts first to avoid bundling.
        const win = (window as any);
        if (win && win.echarts && (win.EChartsReact || win['EChartsReact'])) {
          // ECharts & echarts-for-react UMD already loaded via <script> tags
          const globalWrapper = win.EChartsReact || win['EChartsReact'];
          if (mounted) setChart(() => globalWrapper);
          return;
        }

        // Fallback: attempt dynamic import of the React wrapper and core packages
        const reactWrapper = await import('echarts-for-react').then(mod => mod.default || mod);

        try {
          await import('echarts/core');
        } catch (_) {
          try {
            await import('echarts');
          } catch (e) {
            // allow react wrapper to function if it bundles echarts transitively
          }
        }

        if (mounted) setChart(() => reactWrapper);
  } catch (e: unknown) {
  const err = e instanceof Error ? e : new Error(String(e));
  // dev logger
  const { devError } = require('../utils/devLogger');
  devError('LazyECharts failed to load:', err);
    if (mounted) setLoadError(err);
      }
    })();

    return () => {
      mounted = false;
    };
  }, []);

  if (loadError) return <div className={className}>Chart failed to load</div>;
  if (!Chart) return <div className={className}>Loading chart…</div>;

  // Render the lazily-loaded wrapper from echarts-for-react.
  return <Chart option={option} style={style} className={className} {...rest} />;
}
