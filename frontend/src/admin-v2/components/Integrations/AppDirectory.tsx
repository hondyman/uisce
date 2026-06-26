import React from "react";
import { Card } from "../";

export function AppDirectory() {
  const apps = [
    { name: "Slack", description: "Receive calendar notifications in Slack", icon: "💬", connected: false },
    { name: "Zapier", description: "Automate workflows with 5000+ apps", icon: "⚡", connected: true },
    { name: "Microsoft Teams", description: "Integration for enterprise collaboration", icon: "👥", connected: false },
  ];

  return (
    <div className="app-directory">
      <div className="section-header">
        <h2 className="text-xl font-semibold">Available Apps & Integrations</h2>
      </div>

      <div className="apps-grid" style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))', gap: '1rem', marginTop: '1rem' }}>
        {apps.map((app) => (
          <Card key={app.name} className="app-card">
            <div style={{ display: 'flex', alignItems: 'center', gap: '1rem' }}>
              <span style={{ fontSize: '2rem' }}>{app.icon}</span>
              <div>
                <h3 style={{ margin: 0, fontWeight: 'bold' }}>{app.name}</h3>
                <p style={{ margin: 0, fontSize: '0.875rem', color: '#6b7280' }}>{app.description}</p>
              </div>
            </div>
            <div style={{ marginTop: '1rem', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <span className={`badge ${app.connected ? "badge-active" : "badge-outline"}`}>
                {app.connected ? "Connected" : "Not Connected"}
              </span>
              <button className={`btn btn-sm ${app.connected ? "btn-outline" : "btn-primary"}`}>
                {app.connected ? "Configure" : "Connect"}
              </button>
            </div>
          </Card>
        ))}
      </div>
    </div>
  );
}
