# Client Self-Service Portal (Architecture Spec)

**Status:** Planned (Phase 7.4)
**Design Date:** 2025-12-31

## Overview
The Client Portal is a "White-Label" entry point for Titan's external clients (LPs, GPs, Stewards). Unlike the internal Admin Console, this portal is read-only by default and scoped strictly to the authenticated user's Tenant.

## Architecture

### 1. Frontend (`src/portal`)
- **Independent Entry Point**: A separate React root to ensure lightweight loading (no Admin-heavy libs) and strict security boundaries.
- **Components**:
    - `PortalLayout`: Simplified sidebar (Dashboard, Documents, Settings).
    - `LiveWorkflowView`: Read-only virtualization of the DAG.
    - `SandboxedBuilder`: A restricted version of `UisceBuilder` allowing only safe activities (Notification, Approval).

### 2. Security (Backend)
- **Role Enforcement**: All endpoints accessed via the Portal must have `X-Titan-Role: CLIENT`.
- **Tenant Isolation**: Middleware `RequireTenantID(ctx)` is mandatory.
- **ABAC**: OPA Policies (`policy/portal.rego`) restrict access to specific sensitive Data Objects (e.g., PII hidden).

## Feature Roadmap
1.  **Dashboard**: Real-time view of "My Trades" and "Settlement Risk Scores".
2.  **Documents**: Secure retrieval of generated PDFs (Audit Reports, Tax Forms).
3.  **Alerts**: "Notify me when Dividends Distribute" (Self-service automation).
