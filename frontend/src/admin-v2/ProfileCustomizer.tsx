// Self-Service Studio — Screen 2
// Profile Customizer Matrix (The ABAC Shadowing Engine)
// Spec: PART 5 § Screen 2

import React, { useEffect, useState } from "react";
import { studioApi, StudioApiError } from "./studioApi";
import type {
  AbacPolicy,
  AppendPolicyOverrideRequest,
  PolicyEffect,
} from "./types";

interface ProfileCustomizerProps {
  profileKey: string;
  isSystem: boolean;
}

export const ProfileCustomizer: React.FC<ProfileCustomizerProps> = ({ profileKey, isSystem }) => {
  const [policies, setPolicies] = useState<AbacPolicy[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [adding, setAdding] = useState(false);
  const [action, setAction] = useState<string>("");
  const [effect, setEffect] = useState<PolicyEffect>("deny");
  const [priority, setPriority] = useState<number>(10);
  const [condition, setCondition] = useState<string>("");
  const [name, setName] = useState<string>("");
  const [description, setDescription] = useState<string>("");

  const refresh = async () => {
    setError(null);
    try {
      const res = await fetch(
        `${process.env.REACT_APP_API_URL || "http://localhost:8082/api"}/v1/tenant/policies?target_profile_key=${encodeURIComponent(profileKey)}`,
        { credentials: "include" }
      );
      if (res.ok) {
        const data = await res.json();
        setPolicies(data.policies || []);
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  useEffect(() => {
    refresh();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [profileKey]);

  const handleAdd = async () => {
    setError(null);
    try {
      const req: AppendPolicyOverrideRequest = {
        targetProfileKey: profileKey,
        actionAttribute: action,
        effect,
        priorityRank: priority,
        conditionDsl: condition || undefined,
        name: name || `Tenant override: ${action} on ${profileKey}`,
        description: description || undefined,
      };
      await studioApi.appendPolicyOverride(req);
      setAdding(false);
      setAction("");
      setCondition("");
      setName("");
      setDescription("");
      await refresh();
    } catch (e) {
      const msg = e instanceof StudioApiError ? `${e.message}: ${e.detail || ""}` : String(e);
      setError(msg);
    }
  };

  const systemPolicies = policies.filter((p) => p.origin === "system");
  const tenantPolicies = policies.filter((p) => p.origin === "tenant");

  return (
    <div className="studio-screen">
      <header className="studio-header">
        <h2>Profile Customizer: {profileKey}</h2>
        {isSystem && <p className="studio-muted">Extending Platform Core Blueprint (immutable)</p>}
      </header>

      {error && <div className="studio-error">{error}</div>}

      <section className="studio-section">
        <header>
          <h3>1. Shadowing Engine Matrix (Precedence Resolver)</h3>
        </header>
        <table className="studio-table">
          <thead>
            <tr>
              <th>Origin</th>
              <th>Target Capability</th>
              <th>Applied Restriction DSL</th>
              <th>Effect</th>
              <th>Priority</th>
            </tr>
          </thead>
          <tbody>
            {systemPolicies.length === 0 && tenantPolicies.length === 0 && (
              <tr>
                <td colSpan={5} className="studio-muted">No rules yet.</td>
              </tr>
            )}
            {systemPolicies.map((p) => (
              <tr key={p.policyId} className="system-rule">
                <td><span className="lock-icon" aria-label="system">🔒</span> SYSTEM</td>
                <td>{p.actionAttribute}</td>
                <td><code>{p.conditionDsl || "—"}</code></td>
                <td>
                  <span className={p.effect === "allow" ? "status-pill active" : "status-pill denied"}>
                    {p.effect === "allow" ? "🟢 ALLOW" : "🚫 DENY"}
                  </span>
                </td>
                <td>{p.priorityRank}</td>
              </tr>
            ))}
            {tenantPolicies.map((p) => (
              <tr key={p.policyId} className="tenant-rule">
                <td>👤 TENANT</td>
                <td>{p.actionAttribute}</td>
                <td><code>{p.conditionDsl || "—"}</code></td>
                <td>
                  <span className={p.effect === "allow" ? "status-pill active" : "status-pill denied"}>
                    {p.effect === "allow" ? "🟢 ALLOW" : "🚫 DENY"}
                  </span>
                </td>
                <td>{p.priorityRank} (Max)</td>
              </tr>
            ))}
          </tbody>
        </table>
        {!adding && (
          <button className="studio-btn primary" onClick={() => setAdding(true)}>
            + Append Local Rule
          </button>
        )}
        {adding && (
          <div className="studio-form">
            <h4>Policy Composer</h4>
            <label>
              Action Attribute
              <input value={action} onChange={(e) => setAction(e.target.value)} placeholder="read_ledger_data" />
            </label>
            <label>
              Effect
              <select value={effect} onChange={(e) => setEffect(e.target.value as PolicyEffect)}>
                <option value="deny">🚫 DENY (Deny-Overrides Enforced)</option>
                <option value="allow">🟢 ALLOW</option>
              </select>
            </label>
            <label>
              Priority Rank
              <select value={priority} onChange={(e) => setPriority(Number(e.target.value))}>
                <option value={10}>10 (Highest Override Precedence)</option>
                <option value={20}>20</option>
                <option value={50}>50</option>
              </select>
            </label>
            <label>
              Condition DSL
              <input
                value={condition}
                onChange={(e) => setCondition(e.target.value)}
                placeholder="resource.bo_key == 'sensitive_cash_gl'"
              />
            </label>
            <label>
              Rule Name
              <input
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="Block ledger access for sensitive GL"
              />
            </label>
            <label>
              Description
              <input
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="Optional"
              />
            </label>
            <div className="studio-form-actions">
              <button onClick={() => setAdding(false)}>Cancel</button>
              <button className="primary" onClick={handleAdd}>Apply Rule</button>
            </div>
          </div>
        )}
      </section>
    </div>
  );
};

export default ProfileCustomizer;
