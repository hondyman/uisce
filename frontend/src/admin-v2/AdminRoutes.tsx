import React from "react";
import { Routes, Route, Navigate } from "react-router-dom";
import { AdminLayout } from "./layout/AdminLayout";
import { GlobalOpsDashboard } from "./pages/GlobalOpsDashboard";
import { IncidentDetailPage } from "./pages/IncidentDetailPage";
import { TenantsPage } from "./pages/TenantsPage";
import { APIKeysPage } from "./pages/APIKeysPage";
import { IntegrationsPage } from "./pages/IntegrationsPage";
import { ETLRunDashboardPage } from "./pages/Observability/ETLRunDashboardPage";
import { WasmVersionRegistryPage } from "./pages/Observability/WasmVersionRegistryPage";
import { RuleLineageExplorerPage } from "./pages/Observability/RuleLineageExplorerPage";
import { ScenarioLineageExplorerPage } from "./pages/Observability/ScenarioLineageExplorerPage";

export function AdminRoutes() {
  return (
    <Routes>
      <Route
        path="/"
        element={
          <AdminLayout>
            <GlobalOpsDashboard />
          </AdminLayout>
        }
      />
      <Route
        path="/ops/incidents/:incidentId"
        element={
          <AdminLayout>
            <IncidentDetailPage />
          </AdminLayout>
        }
      />
      <Route
        path="/tenants"
        element={
          <AdminLayout>
            <TenantsPage />
          </AdminLayout>
        }
      />
      <Route
        path="/api-keys"
        element={
          <AdminLayout>
            <APIKeysPage />
          </AdminLayout>
        }
      />
      <Route
        path="/integrations"
        element={
          <AdminLayout>
            <IntegrationsPage />
          </AdminLayout>
        }
      />
      <Route
        path="/telemetry/etl-runs"
        element={
          <AdminLayout>
            <ETLRunDashboardPage />
          </AdminLayout>
        }
      />
      <Route
        path="/telemetry/wasm-registry"
        element={
          <AdminLayout>
            <WasmVersionRegistryPage />
          </AdminLayout>
        }
      />
      <Route
        path="/telemetry/rules/:ruleId/lineage"
        element={
          <AdminLayout>
            <RuleLineageExplorerPage />
          </AdminLayout>
        }
      />
      <Route
        path="/telemetry/scenarios/:scenarioId/lineage"
        element={
          <AdminLayout>
            <ScenarioLineageExplorerPage />
          </AdminLayout>
        }
      />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
