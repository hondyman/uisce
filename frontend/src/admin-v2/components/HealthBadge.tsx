import React from "react";
import type { HealthStatus } from "../types";
import "./HealthBadge.css";

export interface HealthBadgeProps {
  score: number;
  size?: "sm" | "md" | "lg";
}

export function HealthBadge({ score, size = "md" }: HealthBadgeProps) {
  if (score >= 80) {
    return (
      <div className={`health-badge health-healthy health-${size}`}>
        <span className="health-dot"></span>
        <span className="health-label">Healthy</span>
        <span className="health-score">{score}</span>
      </div>
    );
  }

  if (score >= 50) {
    return (
      <div className={`health-badge health-degraded health-${size}`}>
        <span className="health-dot"></span>
        <span className="health-label">Degraded</span>
        <span className="health-score">{score}</span>
      </div>
    );
  }

  return (
    <div className={`health-badge health-critical health-${size}`}>
      <span className="health-dot"></span>
      <span className="health-label">Critical</span>
      <span className="health-score">{score}</span>
    </div>
  );
}

export interface HealthComponentsProps {
  components: Record<string, number>;
}

export function HealthComponents({ components }: HealthComponentsProps) {
  const entries = Object.entries(components).sort(([, a], [, b]) => a - b);

  return (
    <div className="health-components">
      {entries.map(([name, value]) => (
        <div key={name} className="component-item">
          <span className="component-name">{name.replace(/_/g, " ")}</span>
          <div className="component-bar">
            <div className="component-fill" style={{ width: `${value}%` }}></div>
          </div>
          <span className="component-value">{Math.round(value)}</span>
        </div>
      ))}
    </div>
  );
}
