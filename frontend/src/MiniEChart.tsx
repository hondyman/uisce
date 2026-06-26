import LazyECharts from './components/LazyECharts';

interface MiniEChartProps {
  data: any[];
  x: string;
  y: string[];
  type: 'bar' | 'line' | 'pie';
}

export default function MiniEChart({ data, x, y, type }: MiniEChartProps) {
  const option = type === 'pie'
    ? { series: [{ type: 'pie', radius: '60%', data: data.map(d => ({ name: d[x], value: d[y[0]] })) }] }
    : {
        grid: { top: 10, right: 10, bottom: 20, left: 30 },
        xAxis: { type: 'category', data: data.map(d => d[x]) },
        yAxis: { type: 'value' },
        series: y.map(name => ({ name, type, data: data.map(d => d[name]), smooth: type === 'line' }))
      };
  return <LazyECharts option={option} className="mini-echart" />;
}