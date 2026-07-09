// Self-Service Studio — Screen 3
// Entitlement Matrix Control Grid (Pages / Workflow Steps / Public APIs)
// Spec: PART 5 § Screen 3

import React, { useEffect, useState } from "react";
import { studioApi, StudioApiError } from "./studioApi";
import type {
  ComponentEntitlement,
  EntitlementType,
  OverrideState,
} from "./types";

interface EntitlementMatrixProps {
  profileKey: string;
  isCustom: boolean;
}

// Demo catalog of platform components. In a real build this comes from
// the platform's registered UI tree / workflow registry / API routes
// catalog — Cardinal Rule 1.3: the catalog is data, not code.
const PLATFORM_NODES: Record<EntitlementType, Array<{ path: string; label: string; baseline: OverrideState }>> = {
  MENU_PAGE: [
    { path: "gl.trial_balance", label: "Trial Balance Workspace", baseline: "INHERIT_BASELINE" },
    { path: "gl.fee_accrual", label: "Fee Accrual Parameters", baseline: "INHERIT_BASELINE" },
    { path: "trade_ops.manual_allocation", label: "Manual Allocation Trigger", baseline: "INHERIT_BASELINE" },
    { path: "trade_ops.block_settlement", label: "Block-Trade Settlement Lock", baseline: "INHERIT_BASELINE" },
  ],
  WORKFLOW_STEP: [
    { path: "trade_ops.workflow.allocate", label: "Allocate Step", baseline: "INHERIT_BASELINE" },
    { path: "trade_ops.workflow.clear", label: "Clear Step", baseline: "INHERIT_BASELINE" },
  ],
  PUBLIC_API: [
    { path: "/api/v1/public/trades/fetch", label: "GET /trades/fetch", baseline: "INHERIT_BASELINE" },
    { path: "/api/v1/public/orders/submit", label: "POST /orders/submit", baseline: "INHERIT_BASELINE" },
  ],
};

export const EntitlementMatrix: React.FC<EntitlementMatrixProps> = ({ profileKey, isCustom }) => {
  const [tab, setTab] = useState<EntitlementType>("MENU_PAGE");
  const [entitlements, setEntitlements] = useState<ComponentEntitlement[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [editingNode, setEditingNode] = useState<string | null>(null);
  const [editState, setEditState] = useState<OverrideState>("INHERIT_BASELINE");
  const [editCondition, setEditCondition] = useState<string>("");

  const refresh = async () => {
    setError(null);
    try {
      const res = await studioApi.listEntitlements(profileKey, tab);
      setEntitlements(res.entitlements);
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      setError(msg);
    }
  };

  useEffect(() => {
    refresh();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [profileKey, tab]);

  const stateFor = (nodePath: string): OverrideState => {
    const e = entitlements.find((x) => x.nodePath === nodePath);
    return e ? e.overrideState : "INHERIT_BASELINE";
  };

  const handleSave = async (nodePath: string) => {
    setError(null);
    try {
      await studioApi.upsertEntitlement({
        targetProfileKey: profileKey,
        entitlementType: tab,
        nodePath,
        overrideState: editState,
        conditionalDsl: editCondition || undefined,
      });
      setEditingNode(null);
      setEditCondition("");
      await refresh();
    } catch (e) {
      const msg = e instanceof StudioApiError ? `${e.message}: ${e.detail || ""}` : String(e);
      setError(msg);
    }
  };

  const overrideBadge = (state: OverrideState) => {
    switch (state) {
      case "EXPLICIT_ALLOW":
        return <span className="status-pill active">🟢 EXPLICIT ALLOW</span>;
      case "FORCE_DENY":
        return <span className="status-pill denied">🚫 FORCE DENY</span>;
      default:
        return <span className="status-pill baseline">Inherit Baseline</span>;
    }
  };

  const nodes = PLATFORM_NODES[tab];

  return (
    <div className="studio-screen">
      <header className="studio-header">
        <h2>Functional Scope Matrix: {profileKey}</h2>
        <p className="studio-muted">
          Profile Type: {isCustom ? "Custom Tenant-Scoped Instance" : "System Baseline"}
        </p>
      </header>

      {error && <div className="studio-error">{error}</div>}

      <div className="studio-tabs">
        <button
          className={tab === "MENU_PAGE" ? "active" : ""}
          onClick={() => setTab("MENU_PAGE")}
        >
          📂 Pages/Menus
        </button>
        <button
          className={tab === "WORKFLOW_STEP" ? "active" : ""}
          onClick={() => setTab("WORKFLOW_STEP")}
        >
          ⚙️ Workflow Steps
        </button>
        <button
          className={tab === "PUBLIC_API" ? "active" : ""}
          onClick={() => setTab("PUBLIC_API")}
        >
          🔌 Public Edge API
        </button>
      </div>

      <section className="studio-section">
        <table className="studio-table">
          <thead>
            <tr>
              <th>Component Node Target Path</th>
              <th>Ambient Blueprint State</th>
              <th>Enforced Override State</th>
            </tr>
          </thead>
          <tbody>
            {nodes.map((n) => {
              const state = stateFor(n.path);
              const isEditing = editingNode === n.path;
              return (
                <tr key={n.path} className={state === "FORCE_DENY" ? "denied" : state === "EXPLICIT_ALLOW" ? "allowed" : ""}>
                  <td>
                    <code>{n.path}</code>
                    <div className="profile-key">{n.label}</div>
                  </td>
                  <td>
                    <span className="status-pill baseline">{n.baseline === "INHERIT_BASELINE" ? "🟢 ALLOW" : n.baseline === "FORCE_DENY" ? "🚫 DENY" : "ALLOW"}</span>
                  </td>
                  <td>
                    {!isEditing && (
                      <div className="override-row">
                        {overrideBadge(state)}
                        <button className="studio-btn small" onClick={() => {
                          setEditingNode(n.path);
                          setEditState(state);
                          setEditCondition(entitlements.find((x) => x.nodePath === n.path)?.conditionalDsl || "");
                        }}>🔧</button>
                      </div>
                    )}
                    {isEditing && (
                      <div className="override-form">
                        <select
                          value={editState}
                          onChange={(e) => setEditState(e.target.value as OverrideState)}
                        >
                          <option value="INHERIT_BASELINE">Inherit Baseline</option>
                          <option value="EXPLICIT_ALLOW">🟢 EXPLICIT ALLOW</option>
                          <option value="FORCE_DENY">🚫 FORCE DENY</option>
                        </select>
                        <input
                          placeholder="resource.value <= 5000000"
                          value={editCondition}
                          onChange={(e) => setEditCondition(e.target.value)}
                        />
                        <button className="primary" onClick={() => handleSave(n.path)}>Save</button>
                        <button onClick={() => setEditingNode(null)}>Cancel</button>
                      </div>
                    )}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </section>
    </div>
  );
};

export default EntitlementMatrix;
