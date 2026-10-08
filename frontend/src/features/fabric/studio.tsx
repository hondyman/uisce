import React from 'react';
import { registerDomainComponents } from '../../studio-core/components/registry';

import CalculationsLibraryPage from './pages/CalculationsLibraryPage';
import ManagementPage from './pages/preaggregations/ManagementPage';
import DashboardPage from './pages/DashboardPage';
import SettingsPage from './pages/SettingsPage';
import { ReportLibrary } from '../reporting/components/ReportLibrary';
import ReportBuilderPage from '../../pages/ReportBuilderPage';
import QueryLibrary from '../query-builder/pages/QueryLibrary';
import { SemanticModelManager } from '../semantic/components/SemanticModelManager';
import BundleExplorer from '../../components/BundleExplorer';
import ViewsCatalogPage from '../views/pages/ViewsCatalogPage';
import BusinessObjectsPage from '../../pages/BusinessObjectsPage';

const DOMAIN = 'fabric';

registerDomainComponents([
  {
    id: 'fabric.CalculationsLibrary',
    domain: DOMAIN,
    label: 'Calculations Library',
    description: 'Central registry of metrics, business formulas, and reusable calculations.',
    inputs: [],
    events: [],
    render: () => <CalculationsLibraryPage />,
  },
  {
    id: 'fabric.Preaggregations',
    domain: DOMAIN,
    label: 'Preaggregations Management',
    description: 'Pre-computed rollup cubes, partition refreshes, and acceleration telemetry.',
    inputs: [],
    events: [],
    render: () => <ManagementPage />,
  },
  {
    id: 'fabric.Dashboard',
    domain: DOMAIN,
    label: 'Fabric Dashboard',
    description: 'Unified semantic fabric overview, node topology, and performance monitors.',
    inputs: [],
    events: [],
    render: () => <DashboardPage />,
  },
  {
    id: 'fabric.Settings',
    domain: DOMAIN,
    label: 'Fabric Settings',
    description: 'System configurations, caching policies, and query engine routing rules.',
    inputs: [],
    events: [],
    render: () => <SettingsPage />,
  },
  {
    id: 'fabric.ReportLibrary',
    domain: DOMAIN,
    label: 'Report Library',
    description: 'Enterprise catalog of published executive reports, templates, and analytics.',
    inputs: [],
    events: [],
    render: () => <ReportLibrary />,
  },
  {
    id: 'fabric.ReportBuilder',
    domain: DOMAIN,
    label: 'Report Builder',
    description: 'Interactive visual designer for building reports and parameterized views.',
    inputs: [],
    events: [],
    render: () => <ReportBuilderPage />,
  },
  {
    id: 'fabric.QueryLibrary',
    domain: DOMAIN,
    label: 'Query Builder & Library',
    description: 'Visual query builder and saved query management repository.',
    inputs: [],
    events: [],
    render: () => <QueryLibrary />,
  },
  {
    id: 'fabric.SemanticModels',
    domain: DOMAIN,
    label: 'Semantic Models',
    description: 'Manage analytical semantic models, dimensions, and measures.',
    inputs: [],
    events: [],
    render: () => <SemanticModelManager />,
  },
  {
    id: 'fabric.BundleExplorer',
    domain: DOMAIN,
    label: 'Bundle Explorer',
    description: 'Inspect micro-bundles, metadata packages, and cross-tenant exports.',
    inputs: [],
    events: [],
    render: () => <BundleExplorer />,
  },
  {
    id: 'fabric.ViewsCatalog',
    domain: DOMAIN,
    label: 'Views Catalog',
    description: 'Catalog of materialized, logical, and dynamic semantic views.',
    inputs: [],
    events: [],
    render: () => <ViewsCatalogPage />,
  },
  {
    id: 'fabric.BusinessObjects',
    domain: DOMAIN,
    label: 'Business Objects Catalog',
    description: 'Enterprise business entities, driving tables, and bitemporal mappings.',
    inputs: [],
    events: [],
    render: () => <BusinessObjectsPage />,
  },
]);
