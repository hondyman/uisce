import React, { useState } from "react";
import { Card } from "../Card";
import { useSourceAnalytics, type SourceRanking } from "../../hooks/useSourcePreference";
import "./SourceAnalyticsDashboard.css";

const RANK_COLORS = [
  "var(--color-primary)",
  "var(--color-info)",
  "var(--color-success)",
  "var(--color-warning)",
];

function ConfidenceBar({ value }: { value: number }) {
  const color = value >= 90 ? "var(--color-success)" : value >= 70 ? "var(--color-warning)" : "var(--color-danger)";
  return (
    <div className="conf-bar-wrap">
      <div className="conf-bar" style={{ width: `${value}%`, background: color }} />
      <span className="conf-bar-label">{value.toFixed(1)}%</span>
    </div>
  );
}

function RankingBar({ label, count, total, color }: { label: string; count: number; total: number; color: string }) {
  const pct = total > 0 ? (count / total) * 100 : 0;
  return (
    <div className="rank-bar-row">
      <span className="rank-bar-label">{label}</span>
      <div className="rank-bar-track">
        <div className="rank-bar-fill" style={{ width: `${pct}%`, background: color }} />
      </div>
      <span className="rank-bar-count">{count}</span>
    </div>
  );
}

export function SourceAnalyticsDashboard() {
  const [bo, setBo] = useState("");
  const [term, setTerm] = useState("");
  const [region, setRegion] = useState("");

  const { data: report, isLoading, refetch } = useSourceAnalytics(
    bo || undefined,
    term || undefined,
    region || undefined,
  );

  const rankings: SourceRanking[] = report?.rankings ?? [];
  const totalSelections = rankings.reduce((s, r) => s + r.total_selections, 0);

  return (
    <div className="analytics-dash">
      <div className="analytics-dash__header">
        <h2>Source Analytics</h2>
        <div className="analytics-filters">
          <input className="pref-input" placeholder="Business Object" value={bo} onChange={(e) => setBo(e.target.value)} />
          <input className="pref-input" placeholder="Semantic Term" value={term} onChange={(e) => setTerm(e.target.value)} />
          <input className="pref-input" placeholder="Region" value={region} onChange={(e) => setRegion(e.target.value)} />
          <button className="btn btn-outline" onClick={() => refetch()}>Refresh</button>
        </div>
      </div>

      {isLoading ? (
        <div className="analytics-loading">Crunching analytics…</div>
      ) : rankings.length === 0 ? (
        <Card className="analytics-empty"><p>No analytics data available yet.</p></Card>
      ) : (
        <>
          {/* Summary cards */}
          <div className="analytics-summary">
            <Card className="analytics-stat-card">
              <div className="stat-value">{rankings.length}</div>
              <div className="stat-label">Source Systems</div>
            </Card>
            <Card className="analytics-stat-card">
              <div className="stat-value">{totalSelections.toLocaleString()}</div>
              <div className="stat-label">Total Selections</div>
            </Card>
            <Card className="analytics-stat-card">
              <div className="stat-value">
                {rankings.length > 0
                  ? (rankings.reduce((s, r) => s + r.avg_confidence, 0) / rankings.length).toFixed(1) + "%"
                  : "—"}
              </div>
              <div className="stat-label">Avg. Confidence</div>
            </Card>
          </div>

          {/* Ranking table */}
          <Card title="Preference Rankings" className="analytics-table-card">
            <div className="analytics-table">
              <div className="analytics-table__head">
                <span>Source</span>
                <span>1st Pref.</span>
                <span>2nd Pref.</span>
                <span>3rd Pref.</span>
                <span>Confidence</span>
              </div>
              {rankings.map((r, idx) => (
                <div key={r.source_system} className="analytics-table__row">
                  <div className="source-name">
                    <div className="source-dot" style={{ background: RANK_COLORS[idx % RANK_COLORS.length] }} />
                    <span>{r.source_system}</span>
                    {idx === 0 && <span className="top-badge">Top</span>}
                  </div>
                  <div>
                    <RankingBar
                      label=""
                      count={r.first_preference_count}
                      total={r.total_selections}
                      color={RANK_COLORS[0]}
                    />
                  </div>
                  <div>
                    <RankingBar
                      label=""
                      count={r.second_preference_count}
                      total={r.total_selections}
                      color={RANK_COLORS[1]}
                    />
                  </div>
                  <div>
                    <RankingBar
                      label=""
                      count={r.third_preference_count}
                      total={r.total_selections}
                      color={RANK_COLORS[2]}
                    />
                  </div>
                  <div><ConfidenceBar value={r.avg_confidence} /></div>
                </div>
              ))}
            </div>
            <div className="analytics-footer">
              Generated: {report?.generated_at ? new Date(report.generated_at).toLocaleString() : "—"}
            </div>
          </Card>

          {/* Distribution chart */}
          <Card title="First Preference Distribution" className="analytics-dist-card">
            <div className="dist-chart">
              {rankings.map((r, idx) => {
                const pct = totalSelections > 0 ? (r.first_preference_count / totalSelections) * 100 : 0;
                return (
                  <div key={r.source_system} className="dist-bar-wrap">
                    <div className="dist-bar" style={{ height: `${Math.max(pct, 4)}%`, background: RANK_COLORS[idx % RANK_COLORS.length] }} />
                    <span className="dist-bar-label">{r.source_system}</span>
                    <span className="dist-bar-pct">{pct.toFixed(1)}%</span>
                  </div>
                );
              })}
            </div>
          </Card>
        </>
      )}
    </div>
  );
}
