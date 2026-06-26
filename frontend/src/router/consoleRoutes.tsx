import React from 'react';
import { Routes, Route } from 'react-router-dom';
import { ConsoleLayout } from '../layout';
import {
  DashboardHome,
  ETLRunsPage,
  WASMVersionsPage,
  RuleLineagePage,
  ScenarioLineagePage,
} from '../pages/console';

/**
 * Risk & Compliance Console Routes
 * 
 * All routes are wrapped in ConsoleLayout which provides:
 * - Sidebar navigation
 * - Top bar with search and tenant switcher
 * - Responsive layout
 * - Multi-tenant context
 * 
 * Site Structure:
 * /console/
 *   dashboard/ - Main dashboard with KPIs and alerts
 *   etl/
 *     runs/ - ETL run list
 *     runs/:runId - ETL run detail
 *     wasm/ - WASM version registry
 *   compliance/
 *     rules/:ruleId/lineage - Rule evaluation lineage
 *   risk/
 *     scenarios/:scenarioId/lineage - Scenario P&L lineage
 */

export const consoleRoutes = (
  <Routes>
    {/* Dashboard */}
    <Route path="/console/dashboard" element={<DashboardHome />} />

    {/* ETL & Execution */}
    <Route path="/console/etl/runs" element={<ETLRunsPage />} />
    <Route path="/console/etl/runs/:runId" element={<ETLRunsPage />} />
    <Route path="/console/etl/wasm" element={<WASMVersionsPage />} />

    {/* Compliance - Lineage */}
    <Route path="/console/compliance/rules/:ruleId/lineage" element={<RuleLineagePage />} />

    {/* Risk - Lineage */}
    <Route path="/console/risk/scenarios/:scenarioId/lineage" element={<ScenarioLineagePage />} />

    {/* Fallback - redirect to dashboard */}
    <Route path="/console" element={<DashboardHome />} />
  </Routes>
);

/**
 * Usage in App.tsx:
 * 
 * import { BrowserRouter } from 'react-router-dom';
 * import { consoleRoutes } from './router/consoleRoutes';
 * import { ConsoleLayout } from './layout';
 * 
 * function App() {
 *   return (
 *     <BrowserRouter>
 *       <ConsoleLayout>
 *         {consoleRoutes}
 *       </ConsoleLayout>
 *     </BrowserRouter>
 *   );
 * }
 */
