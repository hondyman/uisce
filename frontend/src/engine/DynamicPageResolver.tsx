import React, { useState, useEffect } from 'react';
import { DynamicBODataGrid, FieldMeta } from './DynamicBODataGrid';
import { CircularProgress } from '@mui/material';
import { Error as ErrorIcon } from '@mui/icons-material';

export interface PageLayoutBlueprint {
  page_key: string;
  page_name: string;
  bo_key: string;
  layout_type: 'GRID' | 'FORM' | 'SPLIT_MDM_STUDIO';
  fields: FieldMeta[];
}

export const DynamicPageResolver: React.FC<{ pageKey: string; tenantId: string }> = ({ pageKey, tenantId }) => {
  const [layout, setLayout] = useState<PageLayoutBlueprint | null>(null);
  const [data, setData] = useState<any[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadDynamicPage();
  }, [pageKey, tenantId]);

  const loadDynamicPage = async () => {
    setLoading(true);
    setError(null);
    try {
      // 1. Fetch Dynamic Layout Schema for Page
      const layoutRes = await fetch(`/api/v1/layout/resolve?pageKey=${pageKey}`, {
        headers: { 'X-Tenant-ID': tenantId },
      });
      if (!layoutRes.ok) throw new Error('Failed to resolve page metadata');
      const layoutData: PageLayoutBlueprint = await layoutRes.json();
      setLayout(layoutData);

      // 2. Fetch Data Hydration Payload from BO Endpoint
      const dataRes = await fetch(`/api/v1/bo/data/${layoutData.bo_key}`, {
        headers: { 'X-Tenant-ID': tenantId },
      });
      const records = await dataRes.json();
      setData(records.data || []);
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64 text-slate-400 gap-2 font-sans">
        <CircularProgress size={20} /> Resolving Metadata Layout...
      </div>
    );
  }

  if (error || !layout) {
    return (
      <div className="p-6 bg-red-950/30 border border-red-500/30 rounded-xl text-red-300 flex items-center gap-3 font-sans">
        <ErrorIcon /> Failed loading dynamic layout: {error}
      </div>
    );
  }

  return (
    <div className="p-6 bg-slate-900 min-h-screen text-slate-100 font-sans">
      <div className="mb-6 flex justify-between items-center">
        <div>
          <span className="text-xs font-mono bg-sky-900/50 text-sky-400 px-2 py-0.5 rounded uppercase tracking-wider">
            BO: {layout.bo_key}
          </span>
          <h1 className="text-2xl font-bold text-white mt-1">{layout.page_name}</h1>
        </div>
        <span className="text-xs text-slate-400 font-mono">
          Dynamic Attributes Rendered: {layout.fields.length}
        </span>
      </div>

      {/* Renders Dynamic Data Grid generated 100% from BO field metadata */}
      <DynamicBODataGrid fields={layout.fields} data={data} />
    </div>
  );
};
