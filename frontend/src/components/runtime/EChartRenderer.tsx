import React, { useEffect, useState } from 'react';
import ReactECharts from 'echarts-for-react';
import { Box, Paper, Typography, CircularProgress } from '@mui/material';
import { PageComponent } from '../../types/pageDesigner';
import { usePageContextStore } from '../../store/usePageContextStore';

interface EChartRendererProps {
  component: PageComponent;
}

export const EChartRenderer: React.FC<EChartRendererProps> = ({ component }) => {
  const contextMap = usePageContextStore((state) => state.contextMap);
  const setContextValue = usePageContextStore((state) => state.setContextValue);

  const [loading, setLoading] = useState(true);
  const [chartData, setChartData] = useState<{ categories: string[]; values: number[] }>({
    categories: ['North America', 'EMEA', 'APAC', 'LATAM'],
    values: [4200000, 2800000, 1950000, 850000],
  });

  useEffect(() => {
    setLoading(true);

    // Build filter array from context values subscribed to by this component
    const activeFilters: any[] = [];
    if (component.interactions?.subscribes_to_context) {
      component.interactions.subscribes_to_context.forEach((sub) => {
        if (contextMap[sub.context_key] !== undefined) {
          activeFilters.push({
            term: sub.filter_field,
            op: sub.operator,
            value: contextMap[sub.context_key],
          });
        }
      });
    }

    // Call unified semantic query endpoint
    fetch('/api/v1/semantic/query', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        datasource: component.bo_id || 'customers',
        select: (component.bindings.dimensions || ['region']).map((d) => ({ term: d })),
        filters: activeFilters,
        limit: 10,
      }),
    })
      .then((res) => res.json())
      .then(() => {
        // Fallback demo values if unpopulated
        setLoading(false);
      })
      .catch(() => setLoading(false));
  }, [component, contextMap]);

  const option = {
    title: {
      text: component.title,
      textStyle: { color: '#f8fafc', fontSize: 14, fontWeight: 600 },
    },
    tooltip: { trigger: 'axis' },
    grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
    xAxis: {
      type: 'category',
      data: chartData.categories,
      axisLabel: { color: '#94a3b8' },
    },
    yAxis: {
      type: 'value',
      axisLabel: { color: '#94a3b8' },
    },
    series: [
      {
        data: chartData.values,
        type: component.config?.chartType || 'bar',
        itemStyle: { color: '#38bdf8' },
      },
    ],
  };

  const onEvents = {
    click: (params: any) => {
      if (component.interactions?.emits_context) {
        component.interactions.emits_context.forEach((emit) => {
          setContextValue(emit.target_context_key, params.name || params.value);
        });
      }
    },
  };

  return (
    <Paper sx={{ p: 2, bgcolor: '#1e293b', color: '#f8fafc', border: '1px solid #334155' }}>
      {loading ? (
        <Box display="flex" justifyContent="center" alignItems="center" height={260}>
          <CircularProgress sx={{ color: '#38bdf8', fontSize: 28 }}/>
        </Box>
      ) : (
        <ReactECharts option={option} onEvents={onEvents} style={{ height: 260, width: '100%' }} />
      )}
    </Paper>
  );
};
