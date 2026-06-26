import React, { useState } from "react";
import { SourcePreferenceEditor } from "../components/SourcePreference/SourcePreferenceEditor";
import { SourceAnalyticsDashboard } from "../components/SourcePreference/SourceAnalyticsDashboard";
import { ExceptionManager } from "../components/SourcePreference/ExceptionManager";
import "./SourcePreferencePage.css";

type Tab = "preferences" | "analytics" | "exceptions";

export function SourcePreferencePage() {
  const [tab, setTab] = useState<Tab>("preferences");

  return (
    <div className="sp-page">
      <div className="sp-page__header">
        <h1>Source Preference Management</h1>
        <p className="sp-page__subtitle">
          Manage preferred data sources, view analytics, and handle exceptions across your semantic layer.
        </p>
      </div>

      <div className="sp-tabs">
        {(["preferences", "analytics", "exceptions"] as Tab[]).map((t) => (
          <button
            key={t}
            className={`sp-tab ${tab === t ? "sp-tab--active" : ""}`}
            onClick={() => setTab(t)}
          >
            {t === "preferences" && "📋 "}
            {t === "analytics" && "📊 "}
            {t === "exceptions" && "⚡ "}
            {t.charAt(0).toUpperCase() + t.slice(1)}
          </button>
        ))}
      </div>

      <div className="sp-tab-content">
        {tab === "preferences" && <SourcePreferenceEditor />}
        {tab === "analytics" && <SourceAnalyticsDashboard />}
        {tab === "exceptions" && <ExceptionManager />}
      </div>
    </div>
  );
}
