import React, { useState } from "react";
import { Card } from "../Card";
import {
  useSourceExceptions,
  useResolveException,
  useCreateException,
  type SourceException,
} from "../../hooks/useSourcePreference";
import "./ExceptionManager.css";

const TYPE_ICON: Record<string, string> = {
  SOURCE_CONFLICT: "⚡",
  DATA_QUALITY: "📊",
  SYSTEM_ERROR: "🔴",
  COMPLIANCE_VIOLATION: "🛡️",
};

const LEVEL_LABEL = ["", "Minimal", "Low", "Moderate", "High", "Critical"];
const LEVEL_COLOR = ["", "var(--color-success)", "var(--color-info)", "var(--color-warning)", "var(--color-danger)", "var(--color-danger)"];

export function ExceptionManager() {
  const [statusFilter, setStatusFilter] = useState<string>("open");
  const [showCreate, setShowCreate] = useState(false);
  const [expandedId, setExpandedId] = useState<string | null>(null);

  const { data: exceptions = [], isLoading } = useSourceExceptions(statusFilter || undefined);
  const resolveMutation = useResolveException();
  const createMutation = useCreateException();

  const [newExc, setNewExc] = useState<Partial<SourceException>>({
    business_object: "",
    source_system: "",
    exception_type: "SOURCE_CONFLICT",
    description: "",
    impact_level: 2,
  });

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    await createMutation.mutateAsync(newExc);
    setShowCreate(false);
    setNewExc({ business_object: "", source_system: "", exception_type: "SOURCE_CONFLICT", description: "", impact_level: 2 });
  };

  const criticalCount = exceptions.filter((e) => e.critical_path).length;
  const openCount = exceptions.filter((e) => e.status === "open").length;

  return (
    <div className="exc-manager">
      <div className="exc-manager__header">
        <h2>Exception Manager</h2>
        <div className="exc-header-meta">
          {criticalCount > 0 && (
            <span className="exc-critical-badge">{criticalCount} Critical</span>
          )}
          <span className="exc-open-badge">{openCount} Open</span>
        </div>
        <div className="exc-controls">
          <select
            className="pref-input"
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
          >
            <option value="">All</option>
            <option value="open">Open</option>
            <option value="in_progress">In Progress</option>
            <option value="resolved">Resolved</option>
          </select>
          <button className="btn btn-primary" onClick={() => setShowCreate(true)}>
            + Report Exception
          </button>
        </div>
      </div>

      {/* Create Modal */}
      {showCreate && (
        <div className="pref-modal-overlay" onClick={() => setShowCreate(false)}>
          <div className="pref-modal" onClick={(e) => e.stopPropagation()}>
            <h3>Report Exception</h3>
            <form className="pref-form" onSubmit={handleCreate}>
              <label>
                <span>Business Object</span>
                <input required value={newExc.business_object ?? ""} onChange={(e) => setNewExc((p) => ({ ...p, business_object: e.target.value }))} />
              </label>
              <label>
                <span>Source System</span>
                <input required value={newExc.source_system ?? ""} onChange={(e) => setNewExc((p) => ({ ...p, source_system: e.target.value }))} />
              </label>
              <label>
                <span>Exception Type</span>
                <select value={newExc.exception_type} onChange={(e) => setNewExc((p) => ({ ...p, exception_type: e.target.value }))}>
                  <option value="SOURCE_CONFLICT">Source Conflict</option>
                  <option value="DATA_QUALITY">Data Quality</option>
                  <option value="SYSTEM_ERROR">System Error</option>
                  <option value="COMPLIANCE_VIOLATION">Compliance Violation</option>
                </select>
              </label>
              <label>
                <span>Description</span>
                <textarea required rows={3} value={newExc.description ?? ""} onChange={(e) => setNewExc((p) => ({ ...p, description: e.target.value }))} />
              </label>
              <label>
                <span>Impact Level (1–5)</span>
                <input type="number" min={1} max={5} value={newExc.impact_level} onChange={(e) => setNewExc((p) => ({ ...p, impact_level: +e.target.value }))} />
              </label>
              <div className="pref-form__actions">
                <button type="button" className="btn btn-outline" onClick={() => setShowCreate(false)}>Cancel</button>
                <button type="submit" className="btn btn-primary" disabled={createMutation.isPending}>
                  {createMutation.isPending ? "Reporting…" : "Submit"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {isLoading ? (
        <div className="pref-loading">Loading exceptions…</div>
      ) : exceptions.length === 0 ? (
        <Card className="analytics-empty"><p>No exceptions found for the selected status.</p></Card>
      ) : (
        <div className="exc-list">
          {exceptions.map((exc) => (
            <Card key={exc.id} className={`exc-card ${exc.critical_path ? "exc-card--critical" : ""}`}>
              <div className="exc-card__header" onClick={() => setExpandedId(expandedId === exc.id ? null : exc.id)}>
                <div className="exc-type-icon">{TYPE_ICON[exc.exception_type] ?? "⚠️"}</div>
                <div className="exc-meta">
                  <span className="exc-type">{exc.exception_type.replace(/_/g, " ")}</span>
                  <span className="exc-bo">{exc.business_object} · {exc.source_system}</span>
                </div>
                <div className="exc-badges">
                  {exc.critical_path && <span className="exc-badge exc-badge--critical">Critical Path</span>}
                  <span className="exc-badge" style={{ border: `1px solid ${LEVEL_COLOR[exc.impact_level]}`, color: LEVEL_COLOR[exc.impact_level] }}>
                    L{exc.impact_level} {LEVEL_LABEL[exc.impact_level]}
                  </span>
                  <span className={`exc-status exc-status--${exc.status}`}>{exc.status.replace(/_/g, " ")}</span>
                </div>
                <span className="exc-expand">{expandedId === exc.id ? "▲" : "▼"}</span>
              </div>

              {expandedId === exc.id && (
                <div className="exc-card__body">
                  <p className="exc-desc">{exc.description}</p>
                  <div className="exc-details">
                    <span>Semantic Term: <strong>{exc.semantic_term || "—"}</strong></span>
                    <span>Region: <strong>{exc.region || "—"}</strong></span>
                    <span>Reported: <strong>{new Date(exc.created_at).toLocaleString()}</strong></span>
                    {exc.resolved_at && <span>Resolved: <strong>{new Date(exc.resolved_at).toLocaleString()}</strong></span>}
                  </div>

                  {exc.status !== "resolved" && (
                    <div className="exc-actions">
                      <button
                        className="btn btn-sm btn-primary"
                        disabled={resolveMutation.isPending}
                        onClick={() => resolveMutation.mutate(exc.id)}
                      >
                        {resolveMutation.isPending ? "Resolving…" : "Mark Resolved"}
                      </button>
                    </div>
                  )}
                </div>
              )}
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
