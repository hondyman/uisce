import LazyECharts from './components/LazyECharts';
import type { ColumnMeta, VizConfig, Filter } from './types';

type Props = {
  rows: any[];
  columns: ColumnMeta[];
  viz: VizConfig;
  onCrossFilter: (filter: Filter) => void;
};

function buildOption(type: VizConfig['type'], xField: string, yFields: string[], data: any[]) {
  if (type === 'pie') {
    return {
      tooltip: { trigger: 'item' },
      legend: { orient: 'vertical', left: 'left' },
      series: [
        {
          name: yFields[0],
          type: 'pie',
          radius: '50%',
          data: data.map(d => ({ name: d[xField], value: d[yFields[0]] }))
        }
      ]
    };
  }

  const categories = data.map(d => d[xField]);
  const series = yFields.map(y => ({
    name: y,
    type: type === 'auto' ? 'bar' : type,
    data: data.map(d => d[y])
  }));

  return {
    tooltip: { trigger: 'axis' },
    legend: { data: yFields },
    xAxis: { type: 'category', data: categories },
    yAxis: { type: 'value' },
    series
  };
}

export default function VisualizationPanel({ rows, columns, viz, onCrossFilter }: Props) {
  if (!rows.length || !columns.length) {
    return <div className="viz-panel empty">No data to visualize</div>;
  }

  // Auto-detect: pick first dimension as x, first numeric as y
  let xField = viz.x;
  let yFields = viz.y;
  if (viz.type === 'auto') {
    const dimCol = columns.find(c => c.type === 'string' || c.type === 'time') || columns[0];
    const numCol = columns.find(c => c.type === 'number') || columns[1];
    xField = dimCol?.name;
    yFields = numCol ? [numCol.name] : [];
  }

  if (!xField || !yFields || yFields.length === 0) {
    return <div className="viz-panel empty">Could not auto-detect fields for visualization.</div>;
  }

  const option = buildOption(viz.type, xField, yFields, rows);

  const onEvents = {
    click: (params: any) => {
      if (params.seriesName && params.name) {
        onCrossFilter({ field: xField!, op: 'in', values: [params.name] });
      }
    }
  };

  return (
    <div className="viz-panel">
  <div className="visualization-echart"><LazyECharts option={option} onEvents={onEvents} /></div>
    </div>
  );
}