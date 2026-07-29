// Package contracts provides a Data Contract Gatekeeper for CI/CD schema-change validation.
//
// The gatekeeper intercepts proposed DDL changes (column drops, type changes, table drops)
// and validates them against the semantic catalog before they are applied upstream. It uses
// the lineage graph to find all downstream Business Objects, semantic terms, and exports
// that would be affected by a breaking change, and classifies each change as SAFE or
// CRITICAL.
//
// When CONTRACT_GATEWAY_OPEN_TICKETS=true, CRITICAL violations automatically open a
// MakerChecker compliance ticket for human data-steward sign-off.
//
// Usage:
//
//	gatekeeper := contracts.NewGatekeeper(db, mcService)
//	resp, err := gatekeeper.Validate(ctx, &contracts.ContractValidationRequest{...})
//
// Architectural constraints:
//   - All SQL queries are parameterized with tenant_id (Rule 7: Tenant Isolation)
//   - New packages only import from governance, lineage, agentic (Rule 3: No Package Cycles)
//   - All config read from env vars before struct construction (Rule 1: Config-Before-Code)
package contracts
