import React, { useState } from "react";
import { Card } from "../Card";
import {
  useSourcePreferences,
  useCreatePreference,
  useRequestOverride,
  useApproveOverride,
  usePromoteStage,
  useImpactSimulation,
  type SourcePreference,
} from "../../hooks/useSourcePreference";
import "./SourcePreferenceEditor.css";

const STAGE_ORDER = ["draft", "testing", "staging", "production"] as const;
const IMPACT_COLOR: Record<string, string> = {
  none: "var(--color-success)",
  low: "var(--color-info)",
  moderate: "var(--color-warning)",
  high: "var(--color-danger)",
};

export function SourcePreferenceEditor({ businessObject }: { businessObject?: string }) {
  const [filterBO, setFilterBO] = useState(businessObject ?? "");
  const [showCreate, setShowCreate] = useState(false);
  const [showOverride, setShowOverride] = useState<string | null>(null);

  const { data: prefs = [], isLoading } = useSourcePreferences(filterBO || undefined);
  const createMutation = useCreatePreference();
  const overrideMutation = useRequestOverride();
  const approveMutation = useApproveOverride();
  const promoteMutation = usePromoteStage();
  const sim = useImpactSimulation();

  const [newPref, setNewPref] = useState({
    business_object: businessObject ?? "",
    semantic_term: "",
    region: "GLOBAL",
    priority: 1,
    source_system: "",
    confidence: 80,
  });

  const [overrideForm, setOverrideForm] = useState({ reason: "", valid_to: "" });

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    await createMutation.mutateAsync(newPref);
    setShowCreate(false);
  };

  const handleOverride = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!showOverride) return;
    await overrideMutation.mutateAsync({ id: showOverride, ...overrideForm });
    setShowOverride(null);
    setOverrideForm({ reason: "", valid_to: "" });
    sim.reset();
  };

  const stageIdx = (s: string) => STAGE_ORDER.indexOf(s as typeof STAGE_ORDER[number]);
  const canAdvance = (p: SourcePreference) => p.status !== "production";

  return (
    <div className="pref-editor">
      {/* ---- Header ---- */}
      <div className="pref-editor__header">
        <h2>Source Preferences</h2>
        <div className="pref-editor__controls">
          <input
            className="pref-input"
            placeholder="Filter by business object…"
            value={filterBO}
            onChange={(e) => setFilterBO(e.target.value)}
          />
          <button className="btn btn-primary" onClick={() => setShowCreate(true)}>
            + New Preference
          </button>
        </div>
      </div>

      {/* ---- Create modal ---- */}
      {showCreate && (
        <div className="pref-modal-overlay" onClick={() => setShowCreate(false)}>
          <div className="pref-modal" onClick={(e) => e.stopPropagation()}>
            <h3>Create Source Preference</h3>
            <form className="pref-form" onSubmit={handleCreate}>
              {(["business_object", "semantic_term", "region", "source_system"] as const).map((f) => (
                <label key={f}>
                  <span>{f.replace(/_/g, " ")}</span>
                  <input
                    required
                    value={(newPref as any)[f]}
                    onChange={(e) => setNewPref((p) => ({ ...p, [f]: e.target.value }))}
                  />
                </label>
              ))}
              <label>
                <span>Priority</span>
                <input
                  type="number" min={1} max={10}
                  value={newPref.priority}
                  onChange={(e) => setNewPref((p) => ({ ...p, priority: +e.target.value }))}
                />
              </label>
              <label>
                <span>Confidence (0–100)</span>
                <input
                  type="number" min={0} max={100}
                  value={newPref.confidence}
                  onChange={(e) => setNewPref((p) => ({ ...p, confidence: +e.target.value }))}
                />
              </label>
              <div className="pref-form__actions">
                <button type="button" className="btn btn-outline" onClick={() => setShowCreate(false)}>
                  Cancel
                </button>
                <button type="submit" className="btn btn-primary" disabled={createMutation.isPending}>
                  {createMutation.isPending ? "Saving…" : "Create"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* ---- Override modal ---- */}
      {showOverride && (
        <div className="pref-modal-overlay" onClick={() => setShowOverride(null)}>
          <div className="pref-modal" onClick={(e) => e.stopPropagation()}>
            <h3>Request Override</h3>
            {sim.result && (
              <div className="impact-preview">
                <span className="impact-label">Estimated Impact</span>
                <div className="impact-stats">
                  <span>Affected dates: <strong>{sim.result.affected_dates}</strong></span>
                  <span>Confidence Δ: <strong style={{ color: sim.result.confidence_delta >= 0 ? "var(--color-success)" : "var(--color-danger)" }}>
                    {sim.result.confidence_delta >= 0 ? "+" : ""}{sim.result.confidence_delta}%
                  </strong></span>
                  <span className="impact-badge" style={{ background: IMPACT_COLOR[sim.result.business_impact] || "var(--color-primary)" }}>
                    {sim.result.business_impact.toUpperCase()} IMPACT
                  </span>
                </div>
              </div>
            )}
            <form className="pref-form" onSubmit={handleOverride}>
              <label>
                <span>Reason</span>
                <textarea
                  required rows={3}
                  value={overrideForm.reason}
                  onChange={(e) => setOverrideForm((f) => ({ ...f, reason: e.target.value }))}
                  placeholder="Justify the override…"
                />
              </label>
              <label>
                <span>Valid To</span>
                <input
                  type="date" required
                  value={overrideForm.valid_to}
                  onChange={(e) => setOverrideForm((f) => ({ ...f, valid_to: e.target.value }))}
                />
              </label>
              <div className="pref-form__actions">
                <button type="button" className="btn btn-outline" onClick={() => setShowOverride(null)}>Cancel</button>
                <button type="submit" className="btn btn-primary" disabled={overrideMutation.isPending}>
                  {overrideMutation.isPending ? "Submitting…" : "Submit Override"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* ---- Preference list ---- */}
      {isLoading ? (
        <div className="pref-loading">Loading preferences…</div>
      ) : prefs.length === 0 ? (
        <Card className="pref-empty">
          <p>No source preferences found. Create one to get started.</p>
        </Card>
      ) : (
        <div className="pref-list">
          {prefs.map((p) => (
            <Card key={p.id} className="pref-card">
              <div className="pref-card__top">
                <div className="pref-card__meta">
                  <span className="pref-source">{p.source_system}</span>
                  <span className="pref-term">{p.business_object} › {p.semantic_term}</span>
                  <span className="pref-region">{p.region}</span>
                </div>
                <div className="pref-card__badges">
                  <span className={`stage-badge stage-badge--${p.status}`}>{p.status}</span>
                  <span className="priority-badge">P{p.priority}</span>
                  <span className="confidence-badge">{p.confidence}% conf.</span>
                </div>
              </div>

              {/* Workflow timeline */}
              <div className="stage-timeline">
                {STAGE_ORDER.map((s, i) => (
                  <React.Fragment key={s}>
                    <div className={`stage-step ${i <= stageIdx(p.status) ? "stage-step--done" : ""} ${p.status === s ? "stage-step--active" : ""}`}>
                      <div className="stage-dot" />
                      <span>{s}</span>
                    </div>
                    {i < STAGE_ORDER.length - 1 && <div className={`stage-line ${i < stageIdx(p.status) ? "stage-line--done" : ""}`} />}
                  </React.Fragment>
                ))}
              </div>

              {/* Impact summary */}
              {p.impact_analysis && p.impact_analysis.affected_dates > 0 && (
                <div className="impact-mini">
                  <span>Impact: <strong>{p.impact_analysis.affected_dates} dates</strong></span>
                  <span className="impact-badge" style={{ background: IMPACT_COLOR[p.impact_analysis.business_impact] }}>
                    {p.impact_analysis.business_impact}
                  </span>
                </div>
              )}

              {p.override_reason && (
                <div className="override-reason">
                  <span className="override-reason__label">Override reason:</span> {p.override_reason}
                </div>
              )}

              {/* Actions */}
              <div className="pref-card__actions">
                {p.status === "production" && (
                  <button
                    className="btn btn-sm btn-outline"
                    onClick={() => { setShowOverride(p.id); sim.simulate(p, p.source_system, p.confidence + 5); }}
                  >
                    Request Override
                  </button>
                )}
                {canAdvance(p) && (
                  <button
                    className="btn btn-sm btn-primary"
                    disabled={approveMutation.isPending}
                    onClick={() => approveMutation.mutate({ id: p.id })}
                  >
                    Approve →
                  </button>
                )}
                {canAdvance(p) && (
                  <button
                    className="btn btn-sm btn-ghost"
                    disabled={promoteMutation.isPending}
                    onClick={() => promoteMutation.mutate(p.id)}
                  >
                    Promote
                  </button>
                )}
              </div>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
