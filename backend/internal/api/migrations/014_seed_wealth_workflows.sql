-- Seed Wealth Management Business Processes into Metadata Registry

-- 1. Client Onboarding Process
INSERT INTO meta_processes (id, name, version_major, version_minor, version_patch, status, payload)
VALUES ('bp_client_onboarding', 'Client Onboarding', 1, 0, 0, 'active', '{
  "meta": {
    "id": "bp_client_onboarding",
    "name": "Client Onboarding",
    "version": {"major": 1, "minor": 0, "patch": 0},
    "status": "active",
    "valid_from": "2025-01-01T00:00:00Z"
  },
  "states": ["DATA_COLLECTION", "KYC_REVIEW", "AML_REVIEW", "SUITABILITY_ANALYSIS", "APPROVED", "REJECTED"],
  "transitions": [
    {
      "from": "DATA_COLLECTION",
      "to": "KYC_REVIEW",
      "action_ref": "SubmitClientData",
      "sla": "PT24H"
    },
    {
      "from": "KYC_REVIEW",
      "to": "AML_REVIEW",
      "guard_expr": "kyc_status == COMPLIANT",
      "action_ref": "ApproveKYC",
      "sla": "PT48H"
    },
    {
      "from": "KYC_REVIEW",
      "to": "REJECTED",
      "guard_expr": "kyc_status == NON_COMPLIANT",
      "action_ref": "RejectClient"
    },
    {
      "from": "AML_REVIEW",
      "to": "SUITABILITY_ANALYSIS",
      "guard_expr": "aml_status == COMPLIANT",
      "action_ref": "ApproveAML",
      "sla": "PT48H"
    },
    {
      "from": "AML_REVIEW",
      "to": "REJECTED",
      "guard_expr": "aml_status == NON_COMPLIANT",
      "action_ref": "RejectClient"
    },
    {
      "from": "SUITABILITY_ANALYSIS",
      "to": "APPROVED",
      "guard_expr": "suitability_score >= 70",
      "action_ref": "ApproveClient"
    },
    {
      "from": "SUITABILITY_ANALYSIS",
      "to": "REJECTED",
      "guard_expr": "suitability_score < 70",
      "action_ref": "RejectClient"
    }
  ],
  "bindings": {
    "DATA_COLLECTION": "view_client_form",
    "KYC_REVIEW": "view_kyc_review",
    "AML_REVIEW": "view_aml_review",
    "SUITABILITY_ANALYSIS": "view_suitability_analysis"
  }
}');

-- 2. Order Execution Process
INSERT INTO meta_processes (id, name, version_major, version_minor, version_patch, status, payload)
VALUES ('bp_order_execution', 'Order Execution', 1, 0, 0, 'active', '{
  "meta": {
    "id": "bp_order_execution",
    "name": "Order Execution",
    "version": {"major": 1, "minor": 0, "patch": 0},
    "status": "active",
    "valid_from": "2025-01-01T00:00:00Z"
  },
  "states": ["CREATED", "PENDING_APPROVAL", "COMPLIANCE_CHECK", "SUBMITTED", "PARTIALLY_FILLED", "FILLED", "CANCELLED", "REJECTED"],
  "transitions": [
    {
      "from": "CREATED",
      "to": "PENDING_APPROVAL",
      "action_ref": "SubmitOrder"
    },
    {
      "from": "PENDING_APPROVAL",
      "to": "COMPLIANCE_CHECK",
      "guard_expr": "order_amount < approval_threshold",
      "action_ref": "AutoApprove"
    },
    {
      "from": "PENDING_APPROVAL",
      "to": "COMPLIANCE_CHECK",
      "guard_expr": "approved == true",
      "action_ref": "ManualApprove",
      "sla": "PT1H"
    },
    {
      "from": "COMPLIANCE_CHECK",
      "to": "SUBMITTED",
      "guard_expr": "compliance_status == COMPLIANT",
      "action_ref": "SendToExchange"
    },
    {
      "from": "COMPLIANCE_CHECK",
      "to": "REJECTED",
      "guard_expr": "compliance_status == NON_COMPLIANT",
      "action_ref": "RejectOrder"
    },
    {
      "from": "SUBMITTED",
      "to": "PARTIALLY_FILLED",
      "action_ref": "PartialFill"
    },
    {
      "from": "SUBMITTED",
      "to": "FILLED",
      "action_ref": "FullFill"
    },
    {
      "from": "PARTIALLY_FILLED",
      "to": "FILLED",
      "action_ref": "CompleteFill"
    },
    {
      "from": "SUBMITTED",
      "to": "CANCELLED",
      "action_ref": "CancelOrder"
    },
    {
      "from": "PARTIALLY_FILLED",
      "to": "CANCELLED",
      "action_ref": "CancelOrder"
    }
  ],
  "bindings": {
    "CREATED": "view_order_form",
    "PENDING_APPROVAL": "view_order_approval",
    "COMPLIANCE_CHECK": "view_compliance_check",
    "SUBMITTED": "view_order_status"
  }
}');

-- 3. Portfolio Rebalancing Process
INSERT INTO meta_processes (id, name, version_major, version_minor, version_patch, status, payload)
VALUES ('bp_portfolio_rebalancing', 'Portfolio Rebalancing', 1, 0, 0, 'active', '{
  "meta": {
    "id": "bp_portfolio_rebalancing",
    "name": "Portfolio Rebalancing",
    "version": {"major": 1, "minor": 0, "patch": 0},
    "status": "active",
    "valid_from": "2025-01-01T00:00:00Z"
  },
  "states": ["ANALYSIS", "PROPOSAL_GENERATED", "CLIENT_REVIEW", "APPROVED", "EXECUTING", "COMPLETED", "REJECTED"],
  "transitions": [
    {
      "from": "ANALYSIS",
      "to": "PROPOSAL_GENERATED",
      "action_ref": "GenerateRebalanceProposal"
    },
    {
      "from": "PROPOSAL_GENERATED",
      "to": "CLIENT_REVIEW",
      "action_ref": "SendToClient"
    },
    {
      "from": "CLIENT_REVIEW",
      "to": "APPROVED",
      "guard_expr": "client_approved == true",
      "action_ref": "ClientApprove",
      "sla": "P7D"
    },
    {
      "from": "CLIENT_REVIEW",
      "to": "REJECTED",
      "guard_expr": "client_approved == false",
      "action_ref": "ClientReject"
    },
    {
      "from": "APPROVED",
      "to": "EXECUTING",
      "action_ref": "ExecuteRebalance"
    },
    {
      "from": "EXECUTING",
      "to": "COMPLETED",
      "action_ref": "ConfirmCompletion"
    }
  ],
  "bindings": {
    "ANALYSIS": "view_rebalance_analysis",
    "PROPOSAL_GENERATED": "view_rebalance_proposal",
    "CLIENT_REVIEW": "view_client_approval"
  }
}');
