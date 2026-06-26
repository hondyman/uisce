import type { FC } from 'react';
import { Chart } from 'react-chartjs-2';
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Tooltip,
  Legend,
  ChartOptions,
} from 'chart.js';
import { format } from 'date-fns';

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Tooltip, Legend);

const decisionToValue = (decision: string): number | null => {
  switch (decision) {
    case 'allow':
      return 0;
    case 'block':
      return 2;
    default:
      return 1; // For any other state like 'warn'
  }
};

const DecisionReplayChart: FC<{ timeline: any[] }> = ({ timeline }) => {
  const labels = timeline.map((t) => format(new Date(t.timestamp), 'MM/dd/yy'));

  const dataA = timeline.map((t) => decisionToValue(t.decision_a));
  const dataB = timeline.map((t) => decisionToValue(t.decision_b));

  // Create a separate dataset to highlight the points where decisions differ
  const diffData = dataA.map((valA, i) => (valA !== dataB[i] ? dataB[i] : null));

  const chartData = {
    labels,
    datasets: [
      {
        label: 'Version A',
        data: dataA,
        borderColor: 'rgb(54, 162, 235)',
        backgroundColor: 'rgba(54, 162, 235, 0.5)',
        tension: 0.1,
        pointRadius: 3,
      },
      {
        label: 'Version B',
        data: dataB,
        borderColor: 'rgb(255, 159, 64)',
        backgroundColor: 'rgba(255, 159, 64, 0.5)',
        tension: 0.1,
        pointRadius: 3,
      },
      {
        label: 'Decision Changed',
        data: diffData,
        borderColor: 'rgb(255, 99, 132)',
        backgroundColor: 'rgb(255, 99, 132)',
        pointRadius: 6,
        pointHoverRadius: 8,
        showLine: false,
        type: 'scatter' as const,
      },
    ],
  };

  const options: ChartOptions = {
    responsive: true,
    maintainAspectRatio: false,
    plugins: {
      legend: {
        position: 'top' as const,
      },
      tooltip: {
        callbacks: {
          label: (context: any) => {
            const label = context.dataset.label || '';
            if (label === 'Decision Changed') return; // Return undefined to hide the label
            const value = context.parsed.y;
            const decision = ['Allow', 'Warn', 'Block'][value];
            return `${label}: ${decision}`;
          },
          afterBody: (context: any) => {
            const idx = context[0].dataIndex;
            const t = timeline[idx];
            const violationsA = t.violations_a.map((v: any) => v.rule_id).join(', ') || 'None';
            const violationsB = t.violations_b.map((v: any) => v.rule_id).join(', ') || 'None';
            return [`Run: ${t.change_id.substring(0, 8)}`, `Violations A: ${violationsA}`, `Violations B: ${violationsB}`];
          },
        },
      },
    },
    scales: {
      y: {
        ticks: {
          callback: (value: any) => ['Allow', 'Warn', 'Block'][value],
          stepSize: 1,
        },
        min: -0.5,
        max: 2.5,
        title: {
          display: true,
          text: 'Decision',
        },
      },
    },
  };

  return <Chart type="line" data={chartData} options={options} />;
};

export default DecisionReplayChart;