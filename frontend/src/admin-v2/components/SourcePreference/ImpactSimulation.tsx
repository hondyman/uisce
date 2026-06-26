import React, { useState } from "react";
import { Card } from "../Card";
import { useImpactSimulation, type SourcePreference } from "../../hooks/useSourcePreference";
import "./ImpactSimulation.css";

interface Props {
  preference: SourcePreference;
  onClose?: () => void;
}

const IMPACT_COLOR: Record<string, string> = {
  none: "var(--color-success)",
  low: "var(--color-info)",
  moderate: "var(--color-warning)",
  high: "var(--color-danger)",
};

export function ImpactSimulation({ preference, onClose }: Props) {
  const [newSystem, setNewSystem] = useState(preference.source_system);
  const [newConf, setNewConf] = useState(preference.confidence);
  const { result, loading, error, simulate, reset } = useImpactSimulation();

  const handleRun = () => simulate(preference, newSystem, newConf);

  return (
    <div className="impact-sim">
      <div className="impact-sim__header">
        <h3>What-If Simulation</h3>
        {onClose && <button className="btn btn-ghost btn-sm" onClick={() => { reset(); onClose(); }}>✕</button>}
      </div>

      <p className="impact-sim__subtitle">
        Simulate how changing the source preference for{" "}
        <strong>{preference.business_object} › {preference.semantic_term}</strong> would affect coverage and confidence.
      </p>

      {/* Controls */}
      <Card className="impact-sim__controls">
        <div className="impact-row">
          <div className="impact-col">
            <span className="impact-field-label">Current Source</span>
            <div className="impact-current">
              <strong>{preference.source_system}</strong>
              <span className="conf-chip conf-chip--current">{preference.confidence}% confidence</span>
            </div>
          </div>
          <div className="impact-arrow">→</div>
          <div className="impact-col">
            <label className="impact-field-label">New Source System</label>
            <input
              className="pref-input"
              value={newSystem}
              onChange={(e) => setNewSystem(e.target.value)}
              placeholder="e.g. TradingHours"
            />
            <label className="impact-field-label mt-2">New Confidence (%)</label>
            <input
              type="number" min={0} max={100}
              className="pref-input"
              value={newConf}
              onChange={(e) => setNewConf(+e.target.value)}
            />
          </div>
        </div>
        <button
          className="btn btn-primary mt-4"
          onClick={handleRun}
          disabled={loading || !newSystem}
        >
          {loading ? "Running…" : "Run Simulation"}
        </button>
      </Card>

      {/* Error */}
      {error && <div className="impact-error">{error}</div>}

      {/* Results */}
      {result && (
        <div className="impact-results">
          <Card className="impact-result-card">
            <div className="impact-result-header">
              <span className="impact-badge" style={{ background: IMPACT_COLOR[result.business_impact] ?? "var(--color-primary)" }}>
                {result.business_impact.toUpperCase()} IMPACT
              </span>
            </div>

            <div className="impact-stats-grid">
              {/* Affected Dates */}
              <div className="impact-stat">
                <div className="impact-stat__value">{result.affected_dates.toLocaleString()}</div>
                <div className="impact-stat__label">Affected Dates</div>
              </div>

              {/* Confidence Before */}
              <div className="impact-stat">
                <div className="impact-stat__value">{result.confidence_before}%</div>
                <div className="impact-stat__label">Before</div>
              </div>

              {/* Confidence After */}
              <div className="impact-stat">
                <div
                  className="impact-stat__value"
                  style={{ color: result.confidence_delta >= 0 ? "var(--color-success)" : "var(--color-danger)" }}
                >
                  {result.confidence_after}%
                </div>
                <div className="impact-stat__label">After</div>
              </div>

              {/* Delta */}
              <div className="impact-stat">
                <div
                  className="impact-stat__value impact-delta"
                  style={{ color: result.confidence_delta >= 0 ? "var(--color-success)" : "var(--color-danger)" }}
                >
                  {result.confidence_delta >= 0 ? "+" : ""}{result.confidence_delta}%
                </div>
                <div className="impact-stat__label">Delta</div>
              </div>
            </div>

            {/* Visual confidence bar comparison */}
            <div className="impact-conf-compare">
              <div className="impact-conf-row">
                <span>{preference.source_system}</span>
                <div className="impact-conf-track">
                  <div className="impact-conf-bar impact-conf-bar--before" style={{ width: `${result.confidence_before}%` }} />
                </div>
                <span>{result.confidence_before}%</span>
              </div>
              <div className="impact-conf-row">
                <span>{newSystem}</span>
                <div className="impact-conf-track">
                  <div
                    className="impact-conf-bar impact-conf-bar--after"
                    style={{
                      width: `${result.confidence_after}%`,
                      background: result.confidence_delta >= 0 ? "var(--color-success)" : "var(--color-danger)",
                    }}
                  />
                </div>
                <span>{result.confidence_after}%</span>
              </div>
            </div>

            {result.changed_dates.length > 0 && (
              <div className="impact-dates">
                <h4>Changed Dates</h4>
                <div className="impact-dates-list">
                  {result.changed_dates.map((d) => (
                    <div key={d.date} className="impact-date-row">
                      <span className="impact-date-value">{d.date}</span>
                      <span>{d.old_source} <span className="impact-arrow">→</span> {d.new_source}</span>
                      <span className="conf-chip">{d.old_confidence}% → {d.new_confidence}%</span>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </Card>
        </div>
      )}
    </div>
  );
}
