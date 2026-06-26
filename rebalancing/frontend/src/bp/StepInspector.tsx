import React from "react";
import { ConditionBuilder } from "./ConditionBuilder";
import { ValidationRulePicker } from "./ValidationRulePicker";
import { ApprovalChainFlow, ApprovalChain } from "./ApprovalChainFlow";
import { RoutingRulesEditor, RoutingRules } from "./RoutingRulesEditor";
import { TimingEditor } from "./TimingEditor";

export type StepType = "task" | "approval" | "validation" | "notification" | "integration" | "wait_signal";

// Matching backend model partially for UI editing
export interface BPStepData {
  id: string;
  stepKey: string;
  type: string;
  conditionExpr: string;
  conditionExprType: "json";
  preValidationRuleIds: string[];
  postValidationRuleIds: string[];
  approvalChain?: ApprovalChain;
  routingRules?: RoutingRules;
  delayExpr: string;
  slaExpr: string;
  // ... other fields
}

interface Props {
  step: BPStepData;
  onChange: (step: BPStepData) => void;
}

export const StepInspector: React.FC<Props> = ({ step, onChange }) => {
  if (!step) return <div className="p-4 text-gray-400">Select a step to edit</div>;

  const handleChange = (patch: Partial<BPStepData>) => {
    onChange({ ...step, ...patch });
  };

  // Rule-aware settings visibility logic
  const show = {
    conditions: true, // Always allow entry conditions
    validations: ["task", "approval", "integration", "validation"].includes(step.type),
    approvalChain: step.type === "approval",
    routing: ["task", "bpf_user_task"].includes(step.type) || (step.type === "approval" && false), // Approvals handle their own routing via chain, usually
    timing: ["task", "approval", "wait_signal"].includes(step.type),
    integration: step.type === "integration",
    signal: step.type === "wait_signal",
  };

  return (
    <div className="step-inspector p-4 bg-gray-50 border-l h-full overflow-y-auto w-96">
      <h3 className="text-xl font-bold mb-4">{step.stepKey}</h3>
      
      <section className="mb-6">
        <label className="block text-sm font-bold mb-1">Step Type</label>
        <select 
          className="w-full border p-2 rounded"
          value={step.type} 
          onChange={(e) => handleChange({ type: e.target.value })}
        >
          <option value="task">User Task</option>
          <option value="approval">Approval Chain</option>
          <option value="validation">Validation Rule</option>
          <option value="notification">Notification</option>
          <option value="integration">Integration (API)</option>
          <option value="wait_signal">Wait for Signal</option>
        </select>
      </section>

      {show.conditions && (
        <section className="mb-6">
          <h4 className="text-sm font-bold mb-2 uppercase text-gray-500">Entry Conditions</h4>
          <ConditionBuilder
            value={step.conditionExpr}
            mode={step.conditionExprType}
            onChange={(json) => handleChange({ conditionExpr: json })}
          />
        </section>
      )}

      {show.validations && (
        <section className="mb-6">
          <h4 className="text-sm font-bold mb-2 uppercase text-gray-500">Pre-Validations</h4>
          <ValidationRulePicker
            selectedRuleIds={step.preValidationRuleIds || []}
            onChange={(ids) => handleChange({ preValidationRuleIds: ids })}
          />
        </section>
      )}

      {show.approvalChain && (
        <section className="mb-6">
          <h4 className="text-sm font-bold mb-2 uppercase text-gray-500">Approval Chain</h4>
          <ApprovalChainFlow
            value={step.approvalChain || { levels: [] }}
            onChange={(chain) => handleChange({ approvalChain: chain })}
          />
        </section>
      )}

      {show.routing && (
        <section className="mb-6">
          <h4 className="text-sm font-bold mb-2 uppercase text-gray-500">Routing & Assignment</h4>
          <p className="text-xs text-gray-500 mb-2">Condition-based role assignment.</p>
          <RoutingRulesEditor
            value={step.routingRules || { routes: [], fallbackRole: "" }}
            onChange={(rules) => handleChange({ routingRules: rules })}
          />
        </section>
      )}

      {show.integration && (
        <section className="mb-6">
          <h4 className="text-sm font-bold mb-2 uppercase text-gray-500">Integration Config</h4>
          <div className="p-3 border rounded bg-white text-sm text-gray-400 text-center border-dashed">
            Select API Endpoint... (IntegrationEditor Placeholder)
          </div>
        </section>
      )}

      {show.timing && (
        <section className="mb-6">
          <h4 className="text-sm font-bold mb-2 uppercase text-gray-500">Timing & SLA</h4>
          <TimingEditor
            delayExpr={step.delayExpr}
            slaExpr={step.slaExpr}
            onChange={(timing) =>
              handleChange({ delayExpr: timing.delayExpr, slaExpr: timing.slaExpr })
            }
          />
        </section>
      )}
      
      {show.validations && (
        <section className="mb-6">
          <h4 className="text-sm font-bold mb-2 uppercase text-gray-500">Post-Validations</h4>
          <ValidationRulePicker
            selectedRuleIds={step.postValidationRuleIds || []}
            onChange={(ids) => handleChange({ postValidationRuleIds: ids })}
          />
        </section>
      )}
    </div>
  );
};
