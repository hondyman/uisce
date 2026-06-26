// Admin Layout - Main shell for admin UI with navigation

import React, { useState } from "react";
import { Outlet, useNavigate, useLocation } from "react-router-dom";
import "./AdminLayout.css";

interface NavItem {
  id: string;
  label: string;
  path: string;
  icon: string;
}

const NAV_ITEMS: NavItem[] = [
  { id: "dashboard", label: "Dashboard", path: "/admin", icon: "📊" },
  { id: "tenants", label: "Tenants", path: "/admin/tenants", icon: "🏢" },
  { id: "api-keys", label: "API Keys", path: "/admin/api-keys", icon: "🔑" },
  {
    id: "usage-analytics",
    label: "Usage Analytics",
    path: "/admin/usage",
    icon: "📈",
  },
];

export const AdminLayout: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const [sidebarOpen, setSidebarOpen] = useState(true);

  const isActive = (path: string) => location.pathname === path;

  return (
    <div className="admin-layout">
      {/* Sidebar */}
      <aside className={`admin-sidebar ${sidebarOpen ? "open" : "closed"}`}>
        <div className="sidebar-header">
          <h1 className="logo">SemLayer Admin</h1>
          <button
            className="sidebar-toggle"
            onClick={() => setSidebarOpen(!sidebarOpen)}
            aria-label="Toggle sidebar"
          >
            {sidebarOpen ? "<<" : ">>"}
          </button>
        </div>

        <nav className="sidebar-nav">
          {NAV_ITEMS.map((item) => (
            <button
              key={item.id}
              className={`nav-item ${isActive(item.path) ? "active" : ""}`}
              onClick={() => navigate(item.path)}
            >
              <span className="nav-icon">{item.icon}</span>
              {sidebarOpen && <span className="nav-label">{item.label}</span>}
            </button>
          ))}
        </nav>

        <div className="sidebar-footer">
          <button
            className="logout-btn"
            onClick={() => {
              localStorage.removeItem("token");
              navigate("/login");
            }}
          >
            {sidebarOpen ? "Logout" : "🚪"}
          </button>
        </div>
      </aside>

      {/* Main Content */}
      <main className="admin-main">
        <header className="admin-header">
          <h2>Admin Dashboard</h2>
          <div className="header-info">
            <span className="user-name">Administrator</span>
            <time className="current-time">{new Date().toLocaleTimeString()}</time>
          </div>
        </header>

        <div className="admin-content">
          <Outlet />
        </div>
      </main>
    </div>
  );
};

export default AdminLayout;
