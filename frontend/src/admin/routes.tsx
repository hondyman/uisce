// Admin Routes Configuration

import { RouteObject } from "react-router-dom";
import { AdminLayout } from "../layout/AdminLayout";
import { DashboardPage } from "../pages/DashboardPage";
import { TenantsPage } from "../pages/TenantsPage";
import { APIKeysPage } from "../pages/APIKeysPage";
import { UsageAnalyticsPage } from "../pages/UsageAnalyticsPage";

export const adminRoutes: RouteObject[] = [
  {
    path: "admin",
    element: <AdminLayout />,
    children: [
      {
        index: true,
        element: <DashboardPage />,
      },
      {
        path: "tenants",
        element: <TenantsPage />,
      },
      {
        path: "api-keys",
        element: <APIKeysPage />,
      },
      {
        path: "usage",
        element: <UsageAnalyticsPage />,
      },
    ],
  },
];
