import React, { useEffect, useState, useCallback } from 'react';
import { Card, CardHeader, CardContent, Typography, Box, CircularProgress } from '@mui/material';
import ReactECharts from 'echarts-for-react';
import { PageWidgetDef } from '../PageDesignerTypes';
import { usePageEventBus } from '../PageEventBusContext';

interface DynamicAnalyticsChartWidgetProps {
  widget: PageWidgetDef;
}

export const DynamicAnalyticsChartWidget: React.FC<DynamicAnalyticsChartWidgetProps> = ({ widget }) => {
  const { parameters, setParameter } = usePageEventBus();
  const [loading, setLoading] = useState(false);
  const [chartData, setChartData] = useState<{ categories: string[]; values: number[]; rawRecords: any[] }>({
    categories: ['Equity', 'Fixed Income', 'Alternatives', 'Cash & Collateral', 'Derivatives'],
    values: [420, 310, 180, 95, 60],
    rawRecords: [],
  });

  // Extract active subscribed filters
  const activeFilters = (widget.subscribedParams || []).reduce((acc, paramKey) => {
    if (parameters[paramKey] !== undefined) acc[paramKey] = parameters[paramKey];
    return acc;
  }, {} as Record<string, any>);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/v1/query/preview', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          boKey: widget.boKey,
          queryId: widget.queryId,
          filters: Object.entries(activeFilters).map(([k, v]) => ({
            fieldKey: k,
            operator: '=',
            values: [String(v)],
          })),
          dimensions: ['account_category'],
          measures: ['total_aum'],
        }),
      });
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      const rows = data.rows || [];
      if (rows.length > 0) {
        setChartData({
          categories: rows.map((r: any) => r.account_category || r.category || r.name || 'Unknown'),
          values: rows.map((r: any) => Number(r.total_aum || r.amount || r.value) || 0),
          rawRecords: rows,
        });
      }
    } catch (err) {
      console.warn('Visualization preview fallback to sample data:', err);
    } finally {
      setLoading(false);
    }
  }, [widget.boKey, widget.queryId, JSON.stringify(activeFilters)]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const onChartClick = (params: any) => {
    if (!widget.events?.onChartSelect) return;
    const clickedIdx = params.dataIndex;
    const rawRow = chartData.rawRecords[clickedIdx] || {};

    widget.events.onChartSelect.forEach((action) => {
      if (action.actionType === 'SET_PARAMETER') {
        const val = rawRow[action.sourcePropertyKey] ?? params.name;
        setParameter(action.targetChannel, val);
      }
    });
  };

  const option = {
    backgroundColor: 'transparent',
    tooltip: { trigger: 'axis', backgroundColor: '#071526', borderColor: '#1E293B', textStyle: { color: '#F8FAFC' } },
    xAxis: {
      type: 'category',
      data: chartData.categories,
      axisLabel: { color: '#94A3B8', fontSize: 10 },
      axisLine: { lineStyle: { color: '#1E293B' } },
    },
    yAxis: {
      type: 'value',
      axisLabel: { color: '#94A3B8', fontSize: 10 },
      splitLine: { lineStyle: { color: '#0B1E36' } },
    },
    series: [
      {
        data: chartData.values,
        type: 'bar',
        itemStyle: { color: '#00D4FF', borderRadius: [4, 4, 0, 0] },
      },
    ],
  };

  return (
    <Card sx={{ bgcolor: '#071526', border: '1px solid #1E293B', color: '#F8FAFC' }}>
      <CardHeader
        title={<Typography variant="subtitle2" fontWeight={700} color="#38BDF8">{widget.title}</Typography>}
        sx={{ borderBottom: '1px solid #1E293B', py: 1 }}
      />
      <CardContent sx={{ p: 1 }}>
        {loading ? (
          <Box sx={{ height: 260, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            <CircularProgress size={24} sx={{ color: '#00D4FF' }} />
          </Box>
        ) : (
          <ReactECharts
            option={option}
            style={{ height: 260, width: '100%' }}
            onEvents={{ click: onChartClick }}
          />
        )}
      </CardContent>
    </Card>
  );
};
