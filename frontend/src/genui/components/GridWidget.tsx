import React from "react";
import { gql, useQuery } from "@apollo/client";
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
  const { data: queryData, loading } = useQuery(gql(def.binding?.gql || ""), {
    variables: def.binding?.variables || {},
    skip: !def.binding,
  });

  if (loading) {
    return <GridSkeleton title={def.title} />;
  }

  const rowData = def.binding
    ? extractDataFromPath(queryData, def.binding.dataPath)
    : [];

  // Convert schema columns to AG-Grid column definitions
  const columnDefs: ColDef[] = def.columns.map((col) => ({
    field: col.field,
    headerName: col.headerName,
    width: col.width,
    sortable: col.sortable !== false,
    filter: col.filterable !== false,
    valueFormatter: getValueFormatter(col.type),
  }));

  // Add action column if actions are defined
  if (def.actions && def.actions.length > 0) {
    columnDefs.push({
      headerName: "Actions",
      cellRenderer: (params: any) => (
        <div className="flex gap-2">
          {def.actions!.map((action) => (
            <button
              key={action.id}
              onClick={() => handleAction(action.action, params.data)}
              className="text-blue-600 hover:underline text-sm"
            >
              {action.label}
            </button>
          ))}
        </div>
      ),
      pinned: "right",
      width: 120,
    });
  }

  return (
    <div className="bg-white rounded-lg shadow p-4">
      {def.title && <h3 className="text-lg font-semibold mb-4">{def.title}</h3>}
      {def.subtitle && <p className="text-sm text-gray-600 mb-2">{def.subtitle}</p>}

      <div className="ag-theme-alpine" style={{ height: 400, width: "100%" }}>
        <AgGridReact
          rowData={rowData}
          columnDefs={columnDefs}
          pagination={def.pagination?.enabled}
          paginationPageSize={def.pagination?.pageSize || 20}
          domLayout="autoHeight"
        />
      </div>
    </div>
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
  // TODO: Dispatch action to parent or emit event
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
    <div className="bg-white rounded-lg shadow p-4 animate-pulse">
      {title && <div className="h-6 bg-gray-200 rounded w-1/3 mb-4"></div>}
      <div className="space-y-3">
        {[...Array(5)].map((_, i) => (
          <div key={i} className="h-12 bg-gray-100 rounded"></div>
        ))}
      </div>
    </div>
  );
}