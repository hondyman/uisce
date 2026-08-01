# Uisce Backend Roadmap & Deferred Features

## Deferred Frontend Components (Tracked via UI Roadmap)

1. **Survivorship Rule Studio (`src/components/mdm/SurvivorshipRuleStudio.tsx`):**
   - Connected Endpoint: `POST /api/v1/mdm/survivorship/merge`
   - Purpose: Multi-source field resolution strategies (`SOURCE_PRIORITY`, `MOST_RECENT`, `CONSERVATIVE_MIN/MAX`).

2. **Rule Performance HUD (`src/components/rules/RulePerformanceHUD.tsx`):**
   - Connected Endpoint: `GET /api/v1/compliance/telemetry`
   - Purpose: Real-time nanosecond VM latencies (`p50`, `p95`, `p99`) and 1-click StarRocks Materialized View DDL execution.

3. **Agent Approval Inbox (`src/components/governance/AgentApprovalInbox.tsx`):**
   - Connected Endpoint: `GET/PUT /api/approvals`
   - Purpose: Four-Eyes review inbox for self-healing schema drift repair proposals.

## Strategic Epics (Post-Compliance Hardening)

1. **AI Model Copilot (Feature 13):**
   - Natural language prompt-to-`RuleNode` compilation via BYOK LLM Gateway.
   - Outputs compiled ASTs directly into the Maker-Checker review queue.

2. **GraphRAG & Omnibox Search (Feature 16):**
   - `Cmd+K` command palette via Model Context Protocol (MCP) server for zero-hallucination semantic discovery over Business Objects.

3. **Cross-Region Active-Active Replication:**
   - Multi-region SHA-256 cryptographic audit ledger synchronization for G-SIFI disaster recovery compliance.
