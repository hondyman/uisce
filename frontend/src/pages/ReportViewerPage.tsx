import React, { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { ReportRenderer } from '@/components/reports/ReportRenderer';
import { Card, CardContent } from '@/components/ui/card';
import { Loader2 } from 'lucide-react';
import { useTenant } from '@/contexts/TenantContext';
import { getSelectedRegion } from '@/lib/region';

interface ReportTemplate {
  id: string;
  name: string;
  description: string;
  layout: any;
}

export const ReportViewerPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const { tenant, datasource } = useTenant();
  const [template, setTemplate] = useState<ReportTemplate | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchTemplate = async () => {
      if (!id) return;
      try {
        const response = await fetch(`/api/reports/templates/${id}`, {
          headers: {
            'X-Tenant-ID': tenant?.id || '',
            'X-Tenant-Datasource-ID': datasource?.id || '',
            'X-Tenant-Region': getSelectedRegion(),
          },
        });
        if (!response.ok) {
          throw new Error('Failed to fetch report template');
        }
        const data = await response.json();
        setTemplate(data);
      } catch (err: any) {
        setError(err.message);
      } finally {
        setLoading(false);
      }
    };

    fetchTemplate();
  }, [id]);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    );
  }

  if (error || !template) {
    return (
      <div className="p-8">
        <Card className="bg-red-50 border-red-200">
          <CardContent className="p-6 text-red-600">
            Error: {error || 'Report not found'}
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="p-8 max-w-7xl mx-auto">
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-slate-900 dark:text-slate-100">{template.name}</h1>
        <p className="text-slate-500 mt-2">{template.description}</p>
      </div>
      
      <ReportRenderer layout={template.layout} />
    </div>
  );
};
