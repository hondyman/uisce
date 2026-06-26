import type { FC } from 'react';
import { Scatter } from 'react-chartjs-2';
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  Tooltip,
  Legend,
  TimeScale,
  ChartOptions,
} from 'chart.js';
import 'chartjs-adapter-date-fns';

ChartJS.register(CategoryScale, LinearScale, PointElement, Tooltip, Legend, TimeScale);

const classifyChange = (a: string, b: string): string => `${a} → ${b}`;

const changeColor = (a: string, b: string): string => {
  if (a === 'allow' && b === 'block') return 'rgba(239, 68, 68, 0.7)'; // red
  if (a === 'block' && b === 'allow') return 'rgba(34, 197, 94, 0.7)'; // green
  return 'rgba(249, 115, 22, 0.7)'; // orange
};

const DifferenceOnlyTimeline: FC<{ diffs: any[] }> = ({ diffs }) => {
  const changeTypes = [...new Set(diffs.map((d) => classifyChange(d.decision_a, d.decision_b)))].sort();

  const chartData = {
    datasets: [
      {
        label: 'Decision Changes',
        data: diffs.map((d) => ({
          x: new Date(d.timestamp).getTime(),
          y: classifyChange(d.decision_a, d.decision_b),
          run: d,
        })),
        pointBackgroundColor: diffs.map((d) => changeColor(d.decision_a, d.decision_b)),
        pointRadius: 6,
        pointHoverRadius: 8,
      },
    ],
  };

  const options: ChartOptions<'scatter'> = {
    responsive: true,
    maintainAspectRatio: false,
    scales: {
      x: {
        type: 'time',
        time: {
          unit: 'day',
          tooltipFormat: 'MMM dd, yyyy',
        },
        title: {
          display: true,
          text: 'Date of Change',
        },
      },
      y: {
        type: 'category',
        labels: changeTypes,
        offset: true,
        title: {
          display: true,
          text: 'Decision Change',
        },
      },
    },
    plugins: {
      legend: {
        display: false,
      },
      tooltip: {
        callbacks: {
          title: (ctx: any) => {
            const d = ctx[0].raw.run;
            return `Run: ${d.change_id.substring(0, 8)}`;
          },
          label: (ctx: any) => {
            const r = ctx.raw.run;
            const added = r.violations_added?.map((v: any) => v.rule_id).join(', ') || 'None';
            const removed = r.violations_removed?.map((v: any) => v.rule_id).join(', ') || 'None';
            return [
              `Change: ${r.decision_a} → ${r.decision_b}`,
              `+ Violations: ${added}`,
              `- Violations: ${removed}`,
            ];
          },
        },
      },
    },
  };

  return <Scatter data={chartData} options={options} />;
};

export default DifferenceOnlyTimeline;