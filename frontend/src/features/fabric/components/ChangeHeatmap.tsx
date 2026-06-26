import React, { useMemo } from 'react';
import { Chart as ChartJS, Tooltip, Legend, CategoryScale, LinearScale } from 'chart.js';
import { MatrixController, MatrixElement } from 'chartjs-chart-matrix';
import { Chart } from 'react-chartjs-2';
import { Alert } from '@mui/material';
import { format } from 'date-fns';

ChartJS.register(MatrixController, MatrixElement, Tooltip, Legend, CategoryScale, LinearScale);

interface HeatmapProps {
  timeline: any[];
  bucketSize: 'week' | 'month';
  onCellClick?: (bucket: string, changeType: string) => void;
}

const ChangeHeatmap: React.FC<HeatmapProps> = ({ timeline, bucketSize, onCellClick }) => {
  const { chartData, options } = useMemo(() => {
    const aggregatedData: Record<string, { count: number; topCodes: Record<string, number> }> = {};

    timeline.forEach((d) => {
      if (d.decision_a === d.decision_b) return;

      const timestamp = new Date(d.timestamp);
      const timeBucket =
        bucketSize === 'week'
          ? `W${format(timestamp, 'ww')} '${format(timestamp, 'yy')}`
          : format(timestamp, 'yyyy-MM');

      const changeType = `${d.decision_a} → ${d.decision_b}`;
      const key = `${timeBucket}|${changeType}`;

      if (!aggregatedData[key]) {
        aggregatedData[key] = { count: 0, topCodes: {} };
      }
      aggregatedData[key].count++;

      d.violations_added?.forEach((v: any) => {
        aggregatedData[key].topCodes[v.rule_id] = (aggregatedData[key].topCodes[v.rule_id] || 0) + 1;
      });
    });

    const changeTypes = [...new Set(Object.keys(aggregatedData).map((k) => k.split('|')[1]))].sort();
    const buckets = [...new Set(Object.keys(aggregatedData).map((k) => k.split('|')[0]))].sort();

    const dataset = Object.entries(aggregatedData).map(([key, value]) => {
      const [bucket, type] = key.split('|');
      const topCodes = Object.entries(value.topCodes)
        .sort((a, b) => b[1] - a[1])
        .slice(0, 3)
        .map((entry) => entry[0]);

      return {
        x: bucket,
        y: type,
        v: value.count,
        codes: topCodes,
      };
    });

    const chartData = {
      datasets: [
        {
          label: 'Decision Changes',
          data: dataset,
          backgroundColor: (ctx: any) => {
            if (!ctx.raw) return 'rgba(241, 245, 249, 0.8)'; // slate-100
            const value = ctx.raw.v;
            if (value > 10) return 'rgba(220, 38, 38, 0.8)'; // red-600
            if (value > 5) return 'rgba(249, 115, 22, 0.8)'; // orange-500
            if (value > 0) return 'rgba(253, 224, 71, 0.8)'; // yellow-400
            return 'rgba(241, 245, 249, 0.8)';
          },
          borderColor: '#cbd5e1', // slate-300
          borderWidth: 1,
          width: ({ chart }: any) => (chart.chartArea || {}).width / buckets.length - 1,
          height: ({ chart }: any) => (chart.chartArea || {}).height / changeTypes.length - 1,
        },
      ],
    };

    const options = {
      responsive: true,
      maintainAspectRatio: false,
      scales: {
        x: {
          type: 'category' as const,
          labels: buckets,
          position: 'top' as const,
          ticks: {
            autoSkip: true,
            maxRotation: 90,
            minRotation: 45,
          },
        },
        y: {
          type: 'category' as const,
          labels: changeTypes,
          offset: true,
        },
      },
      plugins: {
        legend: {
          display: false,
        },
        tooltip: {
          callbacks: {
            title: (ctx: any) => `${ctx[0].raw.y} in ${ctx[0].raw.x}`,
            label: (ctx: any) => {
              const d = ctx.raw;
              const topCodes = d.codes.length > 0 ? `Top new violations: ${d.codes.join(', ')}` : 'No new violations';
              return [`${d.v} changes`, topCodes];
            },
          },
        },
      },
      onClick: (_evt: any, elements: any[]) => {
        if (elements.length > 0 && onCellClick) {
          const { x: bucket, y: changeType } = elements[0].element.$context.raw;
          onCellClick(bucket, changeType);
        }
      },
    };

    return { chartData, options };
  }, [timeline, bucketSize, onCellClick]);

  if (!timeline || timeline.length === 0) {
    return <Alert severity="info">No decision changes to display in the heatmap.</Alert>;
  }

  return <Chart type={"matrix" as any} data={chartData} options={options} />;
};

export default ChangeHeatmap;