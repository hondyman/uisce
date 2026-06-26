import React from "react";
import { Link, useLocation } from "react-router-dom";
import "./AdminLayout.css";

export interface AdminLayoutProps {
  children: React.ReactNode;
}

export function AdminLayout({ children }: AdminLayoutProps) {
  const location = useLocation();

  const navItems = [
    { path: "/admin", label: "Dashboard", icon: "📊" },
    { path: "/admin/ops-cockpit", label: "Ops Cockpit", icon: "⚡" },
    { path: "/admin/tenants", label: "Tenants", icon: "🏢" },
    { path: "/admin/api-keys", label: "API Keys", icon: "🔑" },
    { path: "/admin/telemetry/etl-runs", label: "ETL Telemetry", icon: "⏱️" },
    { path: "/admin/telemetry/wasm-registry", label: "WASM Registry", icon: "📦" },
  ];

  const isActive = (path: string) => {
    if (path === "/admin" && location.pathname === "/admin") return true;
    if (path !== "/admin" && location.pathname.startsWith(path)) return true;
    return false;
  };

  return (
    <div className="admin-layout">
      <aside className="sidebar">
        <div className="sidebar-header">
          <div className="logo">SemLayer</div>
          <span className="label">Admin</span>
        </div>

        <nav className="sidebar-nav">
          {navItems.map((item) => (
            <Link
              key={item.path}
              to={item.path}
              className={`nav-item ${isActive(item.path) ? "active" : ""}`}
            >
              <span className="nav-icon">{item.icon}</span>
              <span className="nav-label">{item.label}</span>
            </Link>
          ))}
        </nav>

        <div className="sidebar-footer">
          <div className="user-info">
            <div className="avatar">👤</div>
            <div className="user-details">
              <div className="user-name">Admin</div>
              <div className="user-role">System Admin</div>
            </div>
          </div>
        </div>
      </aside>

      <main className="main-content">
        {children}
      </main>
    </div>
  );
}
