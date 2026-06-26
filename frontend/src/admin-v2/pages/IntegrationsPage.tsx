import React, { useState } from "react";
import { SSOConfig } from "../components/Integrations/SSOConfig";
import { TeamManager } from "../components/Integrations/TeamManager";
import { AppDirectory } from "../components/Integrations/AppDirectory";
import "./IntegrationsPage.css";

export function IntegrationsPage() {
  const [activeTab, setActiveTab] = useState("sso");

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <h1>Integrations & Enterprise</h1>
          <p className="page-subtitle">Manage SSO, Teams, and Connected Apps</p>
        </div>
      </div>

      <div className="tabs-container">
        <div className="tabs-list">
          <button 
            className={`tab-trigger ${activeTab === 'sso' ? 'active' : ''}`}
            onClick={() => setActiveTab('sso')}
          >
            SSO & Auth
          </button>
          <button 
            className={`tab-trigger ${activeTab === 'teams' ? 'active' : ''}`}
            onClick={() => setActiveTab('teams')}
          >
            Teams & Workspace
          </button>
          <button 
            className={`tab-trigger ${activeTab === 'apps' ? 'active' : ''}`}
            onClick={() => setActiveTab('apps')}
          >
            App Directory
          </button>
        </div>

        <div className="tab-content">
          {activeTab === 'sso' && <SSOConfig />}
          {activeTab === 'teams' && <TeamManager />}
          {activeTab === 'apps' && <AppDirectory />}
        </div>
      </div>
    </div>
  );
}
