import React from "react";
import { Handle, Position, NodeProps } from "reactflow";

export interface ApprovalNodeData {
  label: string;
  stepKey: string;
  approvalChain?: {
    rules: Array<{
      id: string;
      label: string;
      condition: any;
      actorRole: string;
    }>;
    fallbackRole?: string;
  };
   // Keeping consistency with other nodes
   activityName?: string;
   signalName?: string;
   conditionExpr?: string;
   delayExpr?: string;
   slaExpr?: string;
   routingRules?: any;
   escalations?: Array<{
        id: string;
        stepNumber: number;
        delayAfterPreviousExpr: string;
        targetActorRole: string;
        notificationTemplate?: string;
        condition?: any;
    }>;
}

export const ApprovalNode: React.FC<NodeProps<ApprovalNodeData>> = ({ data, selected }) => {

  return (
    <div
      className={`approval-node ${selected ? "selected" : ""}`}
      style={{
        padding: "16px",
        border: selected ? "2px solid #0ea5e9" : "1px solid #cbd5e1",
        borderRadius: "8px",
        background: "#f0f9ff",
        minWidth: "160px",
        boxShadow: selected ? "0 0 0 3px rgba(14, 165, 233, 0.1)" : "none",
      }}
    >
      <div style={{ fontWeight: "600", marginBottom: "8px", color: "#0f172a" }}>
        ✅ {data.label || "Approval"}
      </div>
      <div style={{ fontSize: "12px", color: "#64748b", marginBottom: "8px" }}>
        {data.stepKey}
      </div>
      {data.approvalChain?.rules && (
        <div style={{ fontSize: "11px", color: "#334155" }}>
          <strong>Rules:</strong> {data.approvalChain.rules.length}
        </div>
      )}
      <Handle type="target" position={Position.Top} />
      <Handle type="source" position={Position.Bottom} />
    </div>
  );
};
