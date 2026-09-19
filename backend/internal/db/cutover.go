// Package db — DSN cutover strategy (last architectural decision of the MCP security arc).
//
// # Staged MCP-first (chosen)
//
// Flip only the MCP Server connection identity to uisce_mcp_app (or SET ROLE
// on MCP tool transactions) once MCP-read tables are FORCE'd and tool paths
// go through ApplyTenantGUCs / service choke points. Leave the broader HTTP
// api.go surface on postgres until the BeginTx wave classifies and fences
// remaining tenant-table writers.
//
// Reasoning: the SL migration already narrowed MCP blast radius to
// boread/pagestudio/trading/mdmread/driftread. Waiting for ~60 unfenced
// BeginTx files would delay "two-layer binding in production" for the surface
// this arc was about. Intermediate dual-identity (MCP app role / HTTP
// postgres) is explicit and time-boxed by TestBeginTxInventory.
//
// Implementation (landed): OpenMCPAppDB pins SET ROLE uisce_mcp_app on the
// MCP-only pool (api.go SetupRouter). Prefer UISCE_APP_DSN when pg_hba allows;
// otherwise SET ROLE on DATABASE_URL connections. pagestudio/boread wrap reads
// in ApplyTenantGUCs so FORCE policies see gold GUCs.
//
// Still before claiming production binding: Infisical home for UISCE_APP_DSN
// (do not leave only .env), optional pg_hba for direct role login, restart
// server and triple-receipt via pg_stat_activity usename + MCP IDOR through
// that pool + standing flip checklist.
//
// Rejected alternative: single fleet DATABASE_URL flip after full BeginTx
// migration — cleaner ops story, longer wait, couples MCP headline claim to
// unrelated HTTP paths.
package db
