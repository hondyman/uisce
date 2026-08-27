import React from "react";
import { useTheme, Box, Typography, Skeleton } from "@mui/material";
import { useQuery } from "@tanstack/react-query";
import { apiFetch } from "../../lib/apiClient";
import { AgGridReact } from "ag-grid-react";
import "ag-grid-community/styles/ag-grid.css";
import "ag-grid-community/styles/ag-theme-alpine.css";
import type { GridComponent } from "../schema";
import type { ColDef } from "ag-grid-community";
import { devDebug } from '../../utils/devLogger';

interface GridWidgetProps {
  def: GridComponent;
}

export function GridWidget({ def }: GridWidgetProps) {
  const theme = useTheme();
  const binding = def.binding as any;
  const { data: queryData, isLoading } = useQuery({
    queryKey: ['genui-grid', binding?.endpoint, binding?.variables],
    queryFn: async () => {
      if (!binding?.endpoint) return null;
      const response = await apiFetch(binding.endpoint, {
        method: binding.method || 'GET',
      });
      return response.json();
    },
    enabled: !!binding?.endpoint,
  });

  if (isLoading) {
    return <GridSkeleton title={def.title} />;
  }

  const rowData = binding?.dataPath
    ? extractDataFromPath(queryData, binding.dataPath)
    : queryData || [];

  const columnDefs: ColDef[] = (def.columns || []).map((col: any) => ({
    field: col.field,
    headerName: col.headerName,
    width: col.width,
    sortable: col.sortable !== false,
    filter: col.filterable !== false,
    valueFormatter: getValueFormatter(col.type),
  }));

  if (def.actions && def.actions.length > 0) {
    columnDefs.push({
      headerName: "Actions",
      cellRenderer: (params: any) => (
        <Box sx={{ display: 'flex', gap: 1 }}>
          {def.actions!.map((action) => (
            <Box
              component="button"
              key={action.id}
              onClick={() => handleAction(action.action, params.data)}
              sx={{
                color: 'primary.main',
                textDecoration: 'underline',
                fontSize: '0.875rem',
                cursor: 'pointer',
                background: 'none',
                border: 'none',
                p: 0,
                '&:hover': { color: 'primary.dark' }
              }}
            >
              {action.label}
            </Box>
          ))}
        </Box>
      ),
      pinned: "right",
      width: 120,
    });
  }

  return (
    <Box sx={{ bgcolor: 'background.paper', borderRadius: 2, boxShadow: 1, p: 2 }}>
      {def.title && (
        <Typography variant="h6" sx={{ mb: 2 }}>
          {def.title}
        </Typography>
      )}
      {def.subtitle && (
        <Typography variant="body2" sx={{ color: 'grey.600', mb: 1 }}>
          {def.subtitle}
        </Typography>
      )}

      <div className="ag-theme-alpine" style={{ height: 400, width: "100%" }}>
        <AgGridReact
          rowData={rowData}
          columnDefs={columnDefs}
          pagination={def.pagination?.enabled}
          paginationPageSize={def.pagination?.pageSize || 20}
          domLayout="autoHeight"
        />
      </div>
    </Box>
  );
}

function getValueFormatter(type?: string) {
  switch (type) {
    case "currency":
      return (params: any) =>
        params.value != null
          ? new Intl.NumberFormat("en-US", {
              style: "currency",
              currency: "USD",
            }).format(params.value)
          : "";
    case "percentage":
      return (params: any) =>
        params.value != null
          ? new Intl.NumberFormat("en-US", {
              style: "percent",
              minimumFractionDigits: 2,
            }).format(params.value)
          : "";
    case "date":
      return (params: any) =>
        params.value ? new Date(params.value).toLocaleDateString() : "";
    default:
      return undefined;
  }
}

function handleAction(actionType: string, rowData: any) {
  devDebug(`Action ${actionType} triggered for:`, rowData);
}

function extractDataFromPath(obj: any, path: string): any[] {
  const parts = path.split(".");
  let current = obj;
  for (const part of parts) {
    if (current && typeof current === "object") {
      current = current[part];
    } else {
      return [];
    }
  }
  return Array.isArray(current) ? current : [];
}

function GridSkeleton({ title }: { title?: string }) {
  return (
    <Box sx={{ bgcolor: 'background.paper', borderRadius: 2, boxShadow: 1, p: 2 }}>
      {title && <Skeleton variant="text" width="30%" height={28} sx={{ mb: 2 }} />}
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
        {[...Array(5)].map((_, i) => (
          <Skeleton key={i} variant="rectangular" height={48} sx={{ borderRadius: 1 }} />
        ))}
      </Box>
    </Box>
  );
}
