import React, { useMemo } from 'react';
import ReactECharts from 'echarts-for-react';
import { Box } from '@mui/material';
import { QueryExecutionResponse, ChartViewMode } from '../types/explorerTypes';
import { useExplorerTheme } from '../hooks/useExplorerTheme';

interface DynamicChartProps {
  results: QueryExecutionResponse;
  viewMode: ChartViewMode;
  onPointClick?: (params: { dimensionKey: string; dimensionValue: string }) => void;
}

export const DynamicChart: React.FC<DynamicChartProps> = ({ results, viewMode, onPointClick }) => {
  const theme = useExplorerTheme();

  const option = useMemo(() => {
    const dimensions = results.columns.filter((c) => c.category === 'dimension' || c.category === 'time');
    const measures = results.columns.filter((c) => c.category === 'measure' || c.type === 'number');
    const primaryDimKey = dimensions[0]?.key || results.columns[0]?.key;

    if (viewMode === 'pie') {
      const primaryMeasureKey = measures[0]?.key || results.columns[1]?.key;
      const pieData = results.rows.map((row) => ({
        name: String(row[primaryDimKey] || 'Unknown'),
        value: Number(row[primaryMeasureKey] || 0),
      }));

      return {
        backgroundColor: 'transparent',
        tooltip: {
          trigger: 'item',
          formatter: '{b}: {c} ({d}%)',
        },
        legend: {
          top: '5%',
          type: 'scroll',
          textStyle: { color: theme.textMuted },
        },
        series: [
          {
            name: measures[0]?.label || 'Value',
            type: 'pie',
            radius: ['40%', '70%'],
            itemStyle: {
              borderRadius: 5,
              borderColor: theme.background,
              borderWidth: 2,
            },
            data: pieData,
          },
        ],
      };
    }

    const xAxisData = results.rows.map((row) => String(row[primaryDimKey] || ''));

    const series = measures.map((measure) => ({
      name: measure.label,
      type: viewMode === 'area' ? 'line' : viewMode === 'kpi' || viewMode === 'table' ? 'bar' : viewMode,
      areaStyle: viewMode === 'area' ? { opacity: 0.25 } : undefined,
      smooth: viewMode === 'line' || viewMode === 'area',
      symbolSize: viewMode === 'scatter' ? 10 : 6,
      data: results.rows.map((row) => Number(row[measure.key] || 0)),
    }));

    return {
      backgroundColor: 'transparent',
      tooltip: {
        trigger: 'axis',
        axisPointer: { type: 'cross' },
      },
      legend: {
        top: 0,
        textStyle: { color: theme.textMuted },
      },
      grid: {
        left: '3%',
        right: '4%',
        bottom: '5%',
        containLabel: true,
      },
      xAxis: {
        type: 'category',
        data: xAxisData,
        axisLabel: { color: theme.textMuted, rotate: xAxisData.length > 8 ? 30 : 0 },
        axisLine: { lineStyle: { color: theme.border } },
      },
      yAxis: {
        type: 'value',
        splitLine: {
          lineStyle: {
            color: theme.borderSubtle,
            type: 'dashed',
          },
        },
        axisLabel: { color: theme.textMuted },
      },
      series,
    };
  }, [results, viewMode, theme]);

  const onEvents = useMemo(() => {
    return {
      click: (params: any) => {
        if (onPointClick && params?.name) {
          const dimensions = results.columns.filter((c) => c.category === 'dimension' || c.category === 'time');
          const primaryDimKey = dimensions[0]?.key || results.columns[0]?.key;
          onPointClick({ dimensionKey: primaryDimKey, dimensionValue: params.name });
        }
      },
    };
  }, [onPointClick, results]);

  return (
    <Box sx={{ width: '100%', height: '100%', minHeight: 400, p: 1 }}>
      <ReactECharts
        option={option}
        theme={theme.isDark ? 'dark' : 'light'}
        style={{ height: '100%', width: '100%' }}
        opts={{ renderer: 'svg' }}
        notMerge={true}
        onEvents={onEvents}
      />
    </Box>
  );
};

export default DynamicChart;
