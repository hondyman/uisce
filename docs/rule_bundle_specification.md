# Rule Bundle Specification & Portability Contract

## Overview

The Uisce Centralized Validation Rules Engine supports environment-to-environment rule portability and GitOps version control through self-contained, deterministic **Rule Bundles**.

A Rule Bundle is an environment-agnostic interchange format that contains validation rules authored against **semantic terms** (e.g. `TargetQuantity`, `AccountStatus`), stripped of environment-specific catalog UUIDs and physical database column mappings.

---

## 1. Interchange Format Contract

- **Canonical Format**: Strict JSON. While YAML is supported for authoring convenience, all hashing, validation, and storage operations normalize payloads to canonical JSON.
- **Topological Sorting**: Rules with `depends_on` relationships are topologically ordered prior to hashing and import. Cyclic dependencies are strictly rejected (`ERR_DEPENDENCY_CYCLE`).
- **Deterministic AST Serialization (`vm.Compact`)**:
  - Map keys are sorted lexicographically at all nesting depths.
  - IEEE 754 floats are formatted in their shortest unambiguous representation (`105.50` -> `105.5`).
  - Integers preserve exact precision without floating-point conversion.
  - All extraneous whitespace and formatting differences are stripped.
- **Length-Prefixed Checksum (`CanonicalBundleBytes`)**:
  - Rules are sorted by `rule_key` ASC.
  - Each property (`rule_key`, canonical AST, `severity`, `timing`, `domain`, `category`, `live`/`deleted`) is serialized with a 4-byte big-endian length prefix.
  - The SHA-256 digest of this byte stream forms `bundle.checksum`.
  - Bundle metadata (e.g., `created_at`, `exported_from`) does **not** participate in the checksum, ensuring identical rulesets exported from different environments produce identical checksums.

---

## 2. Bundle Schema (`models.RuleBundle`)

```json
{
  "bundle_version": "1.0",
  "created_at": "2026-10-01T12:00:00Z",
  "exported_from": "staging",
  "tenant_scope": "core",
  "origin": "core",
  "checksum": "0837ec32feba021c4de430bc52d2f7a965bd55e8399e8a2c44a95247aff15679",
  "rules": [
    {
      "rule_key": "account.balance_positive",
      "name": "Balance Must Be Positive",
      "bo_name": "account",
      "domain": "mdm",
      "severity": "BLOCK",
      "timing": "pre_write",
      "governance_status": "published",
      "depends_on": [],
      "rule_ast": {
        "type": "condition",
        "field": "balance",
        "fieldPath": "balance",
        "operator": ">",
        "value": 0,
        "valueType": "number"
      }
    }
  ]
}
```

### Tombstone Deletions (`deleted: true`)
Rules marked with `"deleted": true` represent explicit GitOps deletions.
Tombstones participate in the canonical checksum calculation (contributing an empty AST slot and the token `"deleted"`), preventing bundle collisions with prior active states.

---

## 3. Precedence & Governance Guardrails

1. **Precedence**: HTTP query parameters and CLI flags strictly override payload fields.
   - `?dry_run=true` forces preflight diff computation regardless of `bundle.dry_run`.
   - `?overwrite_policy=overwrite` overrides bundle-specified conflict policy.
2. **Core Shadow Protection**: Custom tenant rules cannot reuse rule keys or names of immutable core (gold copy) rules (`ERR_CORE_SHADOW`).
3. **Governance Coercion**: Rules in the `compliance` domain authored as `published` are automatically coerced to `submitted_for_review` unless the caller possesses `RULE_IMPORT_BYPASS_GOVERNANCE` or `is_core_admin`.

---

## 4. CLI Tool (`uisce-rules`)

### Subcommands
- `export`: Exports rules from remote server (`--tenant`, `--bo`, `--domain`, `--origin`, `--format`).
- `diff` / `preview`: Performs dry-run preflight diff against target environment.
- `import`: Atomically executes transactional rule import.
- `validate-bundle`: Validates AST syntax and dependency DAG offline. Use `--stamp` to recalculate and write checksums.

### Exit Codes for CI/CD
- `0`: Success (clean execution, valid bundle, zero errors)
- `1`: Validation / Preflight / Import Error (diff conflicts, rule AST errors, checksum mismatch)
- `2`: Transport / Configuration Error (network down, HTTP 500/401/403, missing required arguments)
