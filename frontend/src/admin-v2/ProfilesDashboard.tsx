// Self-Service Studio — Screen 1
// Profiles Dashboard & Identity Ingestion Hub
// Spec: PART 5 § Screen 1

import React, { useEffect, useState } from "react";
import { studioApi, StudioApiError } from "./studioApi";
import type {
  CloneProfileRequest,
  IdpBroker,
  TenantProfile,
} from "./types";

// In a real implementation the system profile list comes from the
// bootstrap migration (000062) + keycloak realm config. We hard-code
// the canonical three here for the prototype.
const SYSTEM_PROFILES = [
  { profileKey: "platform_analyst", profileName: "Platform Analyst", origin: "system" as const, ruleCount: 14 },
  { profileKey: "platform_trader", profileName: "Platform Trader", origin: "system" as const, ruleCount: 32 },
  { profileKey: "platform_accountant", profileName: "Platform Accountant", origin: "system" as const, ruleCount: 22 },
];

export const ProfilesDashboard: React.FC = () => {
  const [tenantProfiles, setTenantProfiles] = useState<TenantProfile[]>([]);
  const [idpBrokers, setIdpBrokers] = useState<IdpBroker[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [cloneOpen, setCloneOpen] = useState(false);
  const [cloneSource, setCloneSource] = useState<string>("");
  const [cloneTarget, setCloneTarget] = useState<string>("");
  const [cloneTargetName, setCloneTargetName] = useState<string>("");
  const [idpOpen, setIdpOpen] = useState(false);
  const [idpAlias, setIdpAlias] = useState<string>("");
  const [idpProvider, setIdpProvider] = useState<string>("oidc");
  const [targetRealm, setTargetRealm] = useState<string>("uisce");

  const refresh = async () => {
    setLoading(true);
    setError(null);
    try {
      const tenantRes = await fetch(
        `${process.env.REACT_APP_API_URL || "http://localhost:8082/api"}/v1/tenant/profiles`,
        { credentials: "include" }
      );
      if (tenantRes.ok) {
        const data = await tenantRes.json();
        setTenantProfiles(data.profiles || []);
      }
      try {
        const brokers = await studioApi.listIdpBrokers(targetRealm);
        setIdpBrokers(brokers.brokers);
      } catch {
        // IdP broker service may be unconfigured in dev — non-fatal.
        setIdpBrokers([]);
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    refresh();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleClone = async () => {
    setError(null);
    try {
      const req: CloneProfileRequest = {
        sourceProfileKey: cloneSource,
        targetProfileKey: cloneTarget,
        targetProfileName: cloneTargetName,
      };
      await studioApi.cloneProfile(req);
      setCloneOpen(false);
      setCloneSource("");
      setCloneTarget("");
      setCloneTargetName("");
      await refresh();
    } catch (e) {
      const msg = e instanceof StudioApiError ? `${e.message}: ${e.detail || ""}` : String(e);
      setError(msg);
    }
  };

  const handleRegisterIdp = async () => {
    setError(null);
    try {
      await studioApi.registerIdpBroker(targetRealm, idpAlias, idpProvider);
      setIdpOpen(false);
      setIdpAlias("");
      await refresh();
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      setError(msg);
    }
  };

  return (
    <div className="studio-screen">
      <header className="studio-header">
        <h2>uisce_os — Security &amp; Entitlements Studio</h2>
        <input
          className="realm-input"
          placeholder="Target Realm (e.g. uisce)"
          value={targetRealm}
          onChange={(e) => setTargetRealm(e.target.value)}
        />
      </header>

      {error && <div className="studio-error">{error}</div>}
      {loading && <div className="studio-loading">Loading…</div>}

      {/* Section 1: Corporate IdP Link */}
      <section className="studio-section">
        <header>
          <h3>1. Corporate Identity Provider Link (Bring-Your-Own-IdP)</h3>
        </header>
        {idpBrokers.length === 0 ? (
          <p className="studio-muted">No IdP brokers linked. Click + Register IdP to connect Azure AD / Okta / Ping.</p>
        ) : (
          <ul className="studio-broker-list">
            {idpBrokers.map((b) => (
              <li key={b.alias} className={b.enabled ? "broker enabled" : "broker disabled"}>
                <span className="broker-alias">{b.alias}</span>
                <span className="broker-provider">{b.providerId}</span>
                <span className="broker-status">{b.enabled ? "Active" : "Disabled"}</span>
              </li>
            ))}
          </ul>
        )}
        {!idpOpen && (
          <button className="studio-btn primary" onClick={() => setIdpOpen(true)}>
            + Register IdP
          </button>
        )}
        {idpOpen && (
          <div className="studio-form">
            <label>
              Alias
              <input value={idpAlias} onChange={(e) => setIdpAlias(e.target.value)} placeholder="acme-oidc" />
            </label>
            <label>
              Provider
              <select value={idpProvider} onChange={(e) => setIdpProvider(e.target.value)}>
                <option value="oidc">OpenID Connect</option>
                <option value="saml">SAML</option>
                <option value="google">Google</option>
              </select>
            </label>
            <div className="studio-form-actions">
              <button onClick={() => setIdpOpen(false)}>Cancel</button>
              <button className="primary" onClick={handleRegisterIdp}>Register</button>
            </div>
          </div>
        )}
      </section>

      {/* Section 2: Active Directory Group Mapping */}
      <section className="studio-section">
        <header>
          <h3>2. AD Group Mapping</h3>
        </header>
        <p className="studio-muted">
          Configure corporate group → internal profile mappings via the
          <code>security.identity_profile_mappings</code> catalog. Cardinal
          Rule 7.4: group IDs are passed through verbatim from the IdP; resolution
          is the database plane.
        </p>
      </section>

      {/* Section 3: Platform Profile Register */}
      <section className="studio-section">
        <header>
          <h3>3. Platform Profile Register</h3>
        </header>
        <table className="studio-table">
          <thead>
            <tr>
              <th>Profile Name</th>
              <th>Type</th>
              <th>Status</th>
              <th>Rules</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {SYSTEM_PROFILES.map((p) => (
              <tr key={p.profileKey} className="system-profile">
                <td>
                  <span className="lock-icon" aria-label="system">🔒</span> {p.profileName}
                  <div className="profile-key">{p.profileKey}</div>
                </td>
                <td><span className="badge system">SYSTEM</span></td>
                <td><span className="status-pill baseline">Baseline</span></td>
                <td>{p.ruleCount}</td>
                <td>
                  <button className="studio-btn small" onClick={() => { setCloneSource(p.profileKey); setCloneOpen(true); }}>
                    ⚡ Clone
                  </button>
                </td>
              </tr>
            ))}
            {tenantProfiles.map((p) => (
              <tr key={p.profileId} className="tenant-profile">
                <td>
                  <span aria-label="tenant">👤</span> {p.profileName}
                  <div className="profile-key">{p.profileKey}</div>
                </td>
                <td><span className="badge tenant">CUSTOM</span></td>
                <td><span className="status-pill active">Active</span></td>
                <td>{p.ruleCount}</td>
                <td>
                  <a href={`/admin/entitlements/profiles/${p.profileKey}/components`} className="studio-btn small">
                    ✏️ Edit
                  </a>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {!cloneOpen && (
          <button className="studio-btn primary" onClick={() => setCloneOpen(true)} disabled={!cloneSource}>
            + Clone System Profile
          </button>
        )}
        {cloneOpen && (
          <div className="studio-form">
            <label>
              Source Profile
              <input value={cloneSource} onChange={(e) => setCloneSource(e.target.value)} readOnly={!!cloneSource} />
            </label>
            <label>
              Target Profile Key
              <input
                value={cloneTarget}
                onChange={(e) => setCloneTarget(e.target.value)}
                placeholder="inv_senior_analyst"
              />
            </label>
            <label>
              Target Profile Name
              <input
                value={cloneTargetName}
                onChange={(e) => setCloneTargetName(e.target.value)}
                placeholder="InvestCo Senior Analyst"
              />
            </label>
            <div className="studio-form-actions">
              <button onClick={() => setCloneOpen(false)}>Cancel</button>
              <button className="primary" onClick={handleClone}>Clone</button>
            </div>
          </div>
        )}
      </section>
    </div>
  );
};

export default ProfilesDashboard;
