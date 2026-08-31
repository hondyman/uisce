import React from "react";
import { Spinner, ErrorBanner } from "./Feedback";
import {
  useAutofixPolicies,
  useSetAutofixPolicy,
  ExceptionType,
} from "../hooks/useExceptions";
import "./ExceptionAutofixPolicyPanel.css";

// Every exception type the aggregator can emit, matching
// backend/internal/platform_intelligence/exceptions/aggregator.go. Types
// without a saved policy row yet render as "disabled" (the safe default).
const ALL_EXCEPTION_TYPES: ExceptionType[] = [
  "slo_breach",
  "semantic_drift",
  "security_anomaly",
  "data_quality",
  "residency_violation",
  "accessibility_violation",
  "api_inconsistency",
  "preagg_inconsistency",
  "tenant_anomaly",
  "pii_exposure",
];

/**
 * Per-tenant, per-exception-type autofix toggle panel. Intentionally has no
 * global on/off switch — each row is an independent policy write.
 */
export function ExceptionAutofixPolicyPanel() {
  const policiesQuery = useAutofixPolicies();
  const setPolicyMutation = useSetAutofixPolicy();

  const policyByType = new Map(
    (policiesQuery.data || []).map((p) => [p.exception_type, p])
  );

  if (policiesQuery.isLoading) {
    return <Spinner size="sm" />;
  }

  return (
    <div className="autofix-policy-panel">
      <div className="autofix-policy-header">
        <h4>Auto-fix Policy</h4>
        <span className="autofix-policy-hint">Per exception type, this tenant only</span>
      </div>

      {setPolicyMutation.isError && (
        <ErrorBanner message="Failed to update auto-fix policy" />
      )}

      <ul className="autofix-policy-list">
        {ALL_EXCEPTION_TYPES.map((type) => {
          const policy = policyByType.get(type);
          const enabled = policy?.enabled ?? false;
          return (
            <li key={type} className="autofix-policy-row">
              <span className="autofix-policy-type">{type}</span>
              <label className="autofix-policy-toggle">
                <input
                  type="checkbox"
                  checked={enabled}
                  disabled={setPolicyMutation.isPending}
                  onChange={(e) =>
                    setPolicyMutation.mutate({
                      exception_type: type,
                      enabled: e.target.checked,
                      requires_approval: policy?.requires_approval ?? false,
                    })
                  }
                />
                <span>{enabled ? "Enabled" : "Disabled"}</span>
              </label>
            </li>
          );
        })}
      </ul>
    </div>
  );
}
