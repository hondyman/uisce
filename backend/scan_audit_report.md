# Scan Audit Report — rows.Scan safety

Date: 2025-10-27
Branch: chore/triage-u1000-shims

Summary
-------
I ran a repo-wide search under `backend/internal` for uses of `rows.Scan`, `QueryRow(...).Scan`, and other DB scan sites to find places that may be vulnerable to `sql: Scan error` when the DB returns NULL or a type that doesn't match the target Go variable. The goal: avoid injecting human-readable errors into HTTP responses and prevent JSON corruption by using safe scan targets (e.g., `sql.NullString`, `[]byte` for JSON columns, `sql.NullInt64`, `sql.NullTime`) and converting to the public type after scanning.

What I looked for
-----------------
- rows.Scan or QueryRow(...).Scan sites
- Scan targets that are plain `string` but the selected column name contains JSON-ish names (e.g., `_json`, `properties`, `view`, `implementation_json`, `config`, `events`, `filters`, `summary`, `long_description`) or that are known nullable columns (e.g., `created_by`, `updated_by`, `icon_emoji`, etc.)
- Sites that write scan errors to the HTTP response body (dangerous)

High-level findings
-------------------
- The original faulty site (`backend/internal/api/marketplace_routes.go`) has already been patched to scan `icon_emoji` into `sql.NullString`, and the handler was converted to fail-fast with `writeJSONError(...)` on scan errors. Unit tests were added.

- Many other scan sites already follow safe patterns:
  - JSON columns (condition_json, config, events, filters, properties, view, etc.) are commonly scanned into `[]byte` and then `json.Unmarshal`ed.
  - Nullable strings often use `sql.NullString` (several files), and nullable ints use `sql.NullInt64` where appropriate.

- Candidate locations for review (not exhaustive):
  - Files where a `var <name> string` is defined and immediately scanned: these are potential candidates if the DB column can be NULL. Examples found:
    - `backend/internal/api/api.go` (entity registry list) — created_at/updated_at scanned into `string` (lines ~820-880). Consider using `time.Time` or `sql.NullString` depending on desired JSON output.
    - `backend/internal/api/api.go` (var nodeID, qualifiedPath string) — small utility routes.
    - `backend/internal/services/semantic_mapping_service.go`, `abbreviation_service.go`, `handlers/dynamic_handlers.go`, and other handler files include simple `var value string` + `rows.Scan(&value)` patterns. Many of these are benign (single non-null text columns) but should be validated.

- No other handlers were found that write human-readable Scan errors directly into the HTTP response (the marketplace handler was the main offender and it was corrected).

Recommended next steps
----------------------
1. Conservative automated patches (safe):
   - For any `rows.Scan` that uses a target variable whose name indicates JSON (`*_json`, `properties`, `view`, `config`, `events`, `filters`, `implementation_json`, `condition_json`) but the declared target is `string` or `sql.NullString`, change the scan target to `[]byte` (or `sql.NullString` where appropriate) and explicitly `json.Unmarshal` afterwards. This is safe for JSON columns.
   - For any `rows.Scan` that scans `*_emoji`, `created_by`, `updated_by`, `changed_by` into `string`, change to `sql.NullString` and set the pointer/empty string after scanning.

2. Manual review/opt-in patches (recommended):
   - For smaller, loosely-typed handlers (admin/debug endpoints) where `created_at`/`updated_at` are scanned into `string`, decide whether you want `time.Time` or `string` in the JSON output. Converting to `time.Time` is idiomatic but may require updating response shaping in places that expect string.

3. Add a small unit/integration test pattern for critical endpoints that decode DB rows and encode JSON to catch regressions (sqlmock is already in use in tests). Consider adding a GitHub Action job to run these tests on PRs.

Candidate files (scan hits summary)
-----------------------------------
Below are the files that my search flagged. This is not an assertion that they're broken — only that they contain `rows.Scan` with plain `string` target variables or JSON-like column names. Each entry includes the filename and the approximate line where the scan occurs.

- backend/internal/api/validation_rules_routes.go — uses []byte for condition_json (safe) but was reviewed and left as-is.
- backend/internal/api/marketplace_routes.go — patched already (icon_emoji -> sql.NullString, fail-fast on scan_error).
- backend/internal/api/api.go — several occurrences; lines ~615, ~706, ~820, ~1789, ~2489, ~3384, ~4318, etc. Mostly use `[]byte` for properties.
- backend/internal/api/custom_components.go — JSON fields scanned into []byte (safe).
- backend/internal/api/layouts_handlers.go — view scanned into []byte (safe).
- backend/internal/handlers/wealth_management_handler.go — uses sql.NullString for createdAt/updatedAt (safe).
- backend/internal/services/semantic_mapping_service.go — uses sql.NullString for properties JSON (safe) but some places may benefit from scanning into []byte.
- backend/internal/services/pop_service.go — several scans — manual check recommended.
- backend/internal/scanner/*.go — many scans for schema inspection (typically strings) — low risk.

Applied changes so far
----------------------
- `backend/internal/api/marketplace_routes.go` was fixed (icon_emoji -> sql.NullString), handler now returns structured JSON error on scan failure and no longer writes plain-text error lines into the response.
- Unit tests added to ensure NULL icon_emoji and scan-error behavior.
- Frontend debug logs were gated behind a DEV logger.

Automation options
------------------
- I can apply the conservative automated patches for JSON and emoji-like columns now (safe, low-risk). This will:
  - Replace scanning of JSON-ish columns into `string` with `[]byte` temporary variables and call `json.Unmarshal`.
  - Replace scanning of likely-null text columns into `sql.NullString` and convert to `string` or `*string` afterward.
  - Run `go test ./...` (or a narrower package set) and report failures.

- Or I can produce a follow-up PR with the audit report and proposed patches for manual review.

Next action (automated)
-----------------------
If you want me to proceed automatically, I will:
1. Patch all identified JSON-ish scan sites to scan into `[]byte` and `json.Unmarshal` afterward.
2. Patch candidate nullable string sites (icon_emoji, *_by, label/display fields) to use `sql.NullString` and convert after scanning.
3. Run `go test ./internal/...` (or `./...`) and fix compilation issues where necessary.

Let me know whether you'd like me to apply the conservative automated patches now (I recommend yes). If you prefer, I can instead produce a PR with the suggested patches for review before merging.
