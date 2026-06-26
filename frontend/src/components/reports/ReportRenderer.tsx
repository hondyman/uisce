import React from 'react';
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

interface ReportComponent {
  type: 'text' | 'metric_card' | 'chart' | 'table';
  title?: string;
  content?: string; // For text
  metricId?: string; // For metric_card
  dataSource?: string; // For chart/table
  config?: any;
}

interface ReportSection {
  title: string;
  components: ReportComponent[];
}

interface ReportLayout {
  sections: ReportSection[];
}

interface ReportRendererProps {
  layout: ReportLayout;
}

export const ReportRenderer: React.FC<ReportRendererProps> = ({ layout }) => {
  if (!layout || !layout.sections) {
    return <div>Invalid report layout</div>;
  }

  return (
    <div className="space-y-8">
      {layout.sections.map((section, idx) => (
        <div key={idx} className="space-y-4">
          <h2 className="text-2xl font-bold text-slate-800 dark:text-slate-200">{section.title}</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {section.components.map((component, cIdx) => (
              <ReportComponentRenderer key={cIdx} component={component} />
            ))}
          </div>
        </div>
      ))}
    </div>
  );
};

const ReportComponentRenderer: React.FC<{ component: ReportComponent }> = ({ component }) => {
  switch (component.type) {
    case 'text':
      return (
        <Card className="col-span-full">
          <CardHeader>
            <CardTitle>{component.title}</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-slate-600 dark:text-slate-400">{component.content}</p>
          </CardContent>
        </Card>
      );
    case 'metric_card':
      return (
        <Card>
          <CardHeader>
            <CardTitle className="text-sm font-medium text-slate-500">{component.title}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {/* Placeholder for actual metric fetching */}
              $1,234.56
            </div>
            <p className="text-xs text-slate-500">+20.1% from last month</p>
          </CardContent>
        </Card>
      );
    case 'chart':
      return (
        <Card className="col-span-2">
          <CardHeader>
            <CardTitle>{component.title}</CardTitle>
          </CardHeader>
          <CardContent className="h-[300px] flex items-center justify-center bg-slate-50 dark:bg-slate-900 rounded-md">
            <span className="text-slate-400">Chart Placeholder ({component.dataSource})</span>
          </CardContent>
        </Card>
      );
    case 'table':
      return (
        <Card className="col-span-full">
          <CardHeader>
            <CardTitle>{component.title}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="h-[200px] flex items-center justify-center bg-slate-50 dark:bg-slate-900 rounded-md">
              <span className="text-slate-400">Table Placeholder ({component.dataSource})</span>
            </div>
          </CardContent>
        </Card>
      );
    default:
      return <div>Unknown component type: {component.type}</div>;
  }
};
