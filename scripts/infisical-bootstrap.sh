#!/bin/bash
set -euo pipefail

INFISICAL_PROJECT="${INFISICAL_PROJECT:-uisce}"
INFISICAL_ENV="${INFISICAL_ENV:-dev}"
INFISICAL_TOKEN="${INFISICAL_TOKEN:-}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

usage() {
    cat <<EOF
Usage: $0 [OPTIONS]

Bootstrap secrets from Infisical for local development.

OPTIONS:
  -e, --env ENV         Environment: dev, staging, prod (default: dev)
  -p, --project PROJECT Infisical project name (default: uisce)
  -t, --token TOKEN     Infisical service token (or set INFISICAL_TOKEN env var)
  --dry-run             Show what would be generated without writing files
  -h, --help            Show this help message

EXAMPLES:
  # Use default (dev environment)
  ./infisical-bootstrap.sh

  # Generate for staging
  ./infisical-bootstrap.sh -e staging

  # Use a specific token
  ./infisical-bootstrap.sh -t ifs_xxxxx

ENVIRONMENT VARIABLES:
  INFISICAL_TOKEN    Infisical service token (required)
  INFISICAL_PROJECT  Project name (default: uisce)
  INFISICAL_ENV      Environment (default: dev)

INFISICAL SETUP:
  1. Install Infisical CLI: brew install infisical
  2. Run: infisical login
  3. Create a service token at: https://app.infisical.com
  4. Export: export INFISICAL_TOKEN=ifs_xxxxx

EOF
    exit 1
}

# log() writes to stderr so callers that capture stdout (e.g. `tmp_path=$(fn)`)
# don't accidentally absorb log lines into their captured value. warn() and
# die() already go to stderr; making log() consistent means callers don't
# have to think about whether stdout is safe to capture.
log() { echo "[bootstrap] $*" >&2; }
warn() { echo "[bootstrap] WARNING: $*" >&2; }
die() { echo "[bootstrap] ERROR: $*" >&2; exit 1; }

check_infisical() {
    if ! command -v infisical &> /dev/null; then
        die "Infisical CLI not found. Install: brew install infisical"
    fi
}

parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -e|--env) INFISICAL_ENV="$2"; shift 2 ;;
            -p|--project) INFISICAL_PROJECT="$2"; shift 2 ;;
            -t|--token) INFISICAL_TOKEN="$2"; shift 2 ;;
            --dry-run) DRY_RUN=true; shift ;;
            -h|--help) usage ;;
            *) die "Unknown option: $1" ;;
        esac
    done
}

pull_secrets_at_path() {
    local path="$1"
    local domain_arg="--domain=http://100.84.50.65:8085/api"
    # Project ID for the post-recovery northwind tenant — replaced on
    # 2026-09-15 because the prior project (9af25976-...) was wiped
    # during the 2026-09-13 encryption-key rotation (see
    # backend/docs/ORDERS_END_TO_END.md Post-session audit). The org
    # ID (4db162a6-90b2-46c4-9164-5ebf58b53390) is not needed here —
    # projectId resolves through the org context.
    local proj_arg="--projectId=860e3163-8e2d-410b-a9e5-c7dd44d1e343"
    local output=""

    # Run non-interactively with /dev/null stdin so infisical will not prompt
    if [ -n "$INFISICAL_TOKEN" ]; then
        output=$(infisical secrets $domain_arg $proj_arg --env="$INFISICAL_ENV" --path="$path" --recursive -o json --token="$INFISICAL_TOKEN" --silent </dev/null 2>/dev/null || true)
    else
        output=$(infisical secrets $domain_arg $proj_arg --env="$INFISICAL_ENV" --path="$path" --recursive -o json --silent </dev/null 2>/dev/null || true)
    fi

    if [ -z "$output" ] || ! echo "$output" | jq . >/dev/null 2>&1; then
        output="{}"
    fi
    echo "$output"
}

# Merges two Infisical JSON secret payloads (each may be an array of
# {secretKey, secretValue} or an object) into a single deduped array,
# with entries from $2 winning over $1 on key collisions.
merge_secrets_json() {
    local a b
    a=$(echo "$1" | jq 'if type=="array" then . elif type=="object" and length>0 then (to_entries | map({secretKey:.key, secretValue:.value})) else [] end' 2>/dev/null) || a='[]'
    b=$(echo "$2" | jq 'if type=="array" then . elif type=="object" and length>0 then (to_entries | map({secretKey:.key, secretValue:.value})) else [] end' 2>/dev/null) || b='[]'
    jq -n --argjson a "$a" --argjson b "$b" '
        ($a + $b) as $all
        | reduce $all[] as $item ({}; if $item.secretKey != null and $item.secretKey != "" then .[$item.secretKey] = $item.secretValue else . end)
        | to_entries | map({secretKey: .key, secretValue: .value})
    '
}

pull_secrets() {
    # Secrets live both at the project root and inside a "core" folder
    # (e.g. KEYCLOAK_JWKS_URL, DATABASE_URL). --recursive from "/" is
    # supposed to cover both, but fetch /core explicitly too so a
    # folder-scoped token or a CLI version that doesn't honor --recursive
    # for nested folders still picks these up.
    local root_output core_output
    root_output=$(pull_secrets_at_path "/")
    core_output=$(pull_secrets_at_path "/core")

    if [ "$root_output" = "{}" ] && [ "$core_output" = "{}" ]; then
        warn "Could not fetch valid JSON secrets from Infisical (not logged in or server unreachable). Using default/existing environment settings."
    fi

    merge_secrets_json "$root_output" "$core_output"
}

generate_env_file() {
    local secrets_json="$1"
    local output_path="$2"
    local section="$3"

    if [ -n "${DRY_RUN:-}" ]; then
        log "[DRY-RUN] Would generate $output_path"
        return
    fi

    log "Generating $output_path ($section)"

    # Write to a temp file in the same directory as the target and print
    # its path on stdout so the caller (typically generate_composite_secrets)
    # can keep appending to the same temp file before the final atomic
    # mv. The previous form opened $output_path with `>` *before* checking
    # that the Infisical pull had produced anything usable - so a
    # failed/empty pull truncated the target down to just the header.
    # Any subsequent step that errored (missing API_TOKEN_ENCRYPTION_KEY
    # composite, for example) then aborted the script mid-write, leaving
    # the target in a skeleton state with no rollback. The temp-file form:
    # never touch the real target until every step that contributes to
    # it has succeeded.
    #
    # Two guards:
    # (a) Bail with rm -f (no .env touched) if Infisical returned nothing
    #     parseable - a successful run with an empty pull would otherwise
    #     atomic-mv over a fully-populated existing file with a sparse
    #     skeleton (only the composites land). The previous failure of
    #     this kind clobbered a 115-line .env down to 14 lines. The pull
    #     MUST produce non-empty content for the script to write anything
    #     at all - if it doesn't, the existing file is preserved and the
    #     caller gets a clear "no secrets pulled" signal to handle.
    # (b) Bail with rm -f if the JSON parse fails (the file would have the
    #     header but no secrets - same risk as a).
    local tmp_path
    tmp_path=$(mktemp "${output_path}.tmp.XXXXXX") || {
        warn "Failed to create temp file for $output_path"
        return 1
    }

    {
        echo "# Generated by scripts/infisical-bootstrap.sh"
        echo "# Environment: $INFISICAL_ENV | Project: $INFISICAL_PROJECT"
        echo "# DO NOT COMMIT THIS FILE"
        echo ""
    } > "$tmp_path"

    if [ -z "$secrets_json" ] || [ "$secrets_json" = "null" ]; then
        warn "No secrets returned for $section - leaving $output_path untouched"
        rm -f "$tmp_path"
        return 2  # distinct exit code so main() can treat "no pull" vs "real error" differently if needed
    fi

    # Write the secrets to tmp_path. If the JSON is empty (e.g. {} or []),
    # jq produces zero output and the file would contain just the header -
    # we count lines after writing and bail if zero secrets landed, so the
    # existing target isn't mv-overwritten with a sparse skeleton.
    local secret_count=0
    if ! secret_count=$(echo "$secrets_json" | jq -r '
        if type == "array" then
            [.[] | select(.secretKey != null and .secretKey != "")] | length
        elif type == "object" then
            [.[] | select(.key != null and .key != "")] | length
        else
            0
        end
    ' 2>/dev/null); then
        warn "Failed to parse JSON for $section"
        rm -f "$tmp_path"
        return 1
    fi

    if [ "$secret_count" -eq 0 ]; then
        warn "Pulled zero secrets for $section - leaving $output_path untouched"
        rm -f "$tmp_path"
        return 2
    fi

    if ! echo "$secrets_json" | jq -r '
        if type == "array" then
            .[] | select(.secretKey != null and .secretKey != "") | "\(.secretKey)=\"\(.secretValue)\""
        elif type == "object" then
            to_entries[] | "\(.key)=\"\(.value)\""
        end
    ' >> "$tmp_path" 2>/dev/null; then
        warn "Failed to write secrets to $tmp_path"
        rm -f "$tmp_path"
        return 1
    fi

    # Print the tmp path so generate_composite_secrets can append to it.
    echo "$tmp_path"
}

generate_composite_secrets() {
    local output_path="$1"
    local tmp_path="$2"
    local prev_database_url="$3"
    local prev_postgres_dsn="$4"
    local db_pass="${POSTGRES_PASSWORD:-postgres}"
    local db_user="${POSTGRES_USER:-postgres}"
    local db_host="${DB_HOST:-100.84.50.65}"
    local db_port="${DB_PORT:-5432}"
    local db_name="${DB_NAME:-alpha}"

    if [ -n "${DRY_RUN:-}" ]; then
        log "[DRY-RUN] Would generate composite secrets in $output_path"
        return
    fi

    # We write to $tmp_path (the file generate_env_file created) and only
    # mv into $output_path at the very end. If any composite step fails,
    # $tmp_path is removed and $output_path is left untouched - the
    # previous failure mode had `>>` writing directly to $output_path
    # with no rollback, so a partial composite step left the real file
    # truncated to header + secrets + first-few-composites, then aborted.

    # generate_env_file populates DATABASE_URL/POSTGRES_DSN from Infisical
    # when the pull succeeds. Before the prev-capture guard existed, an
    # unreachable Infisical server silently clobbered a working
    # TLS-configured DSN (sslmode=verify-full plus client certs) with a
    # plaintext sslmode=disable default on every single run - see
    # backend/docs/INCIDENT_REPORT_20260906.md. Prefer whatever DSN was
    # already in this file before this run started; only fall back to the
    # plaintext default, loudly, if there was nothing to preserve.
    if ! grep -q "^DATABASE_URL=" "$tmp_path"; then
        if [ -n "$prev_database_url" ]; then
            # Quoted: the DSN contains "&", which an unquoted assignment
            # in a sourced .env file hands to the shell as a job-control
            # operator, silently truncating the value to empty the moment
            # anything sources this file with plain `source`/`. `.
            echo "DATABASE_URL=\"${prev_database_url}\"" >> "$tmp_path"
        else
            warn "No DATABASE_URL from Infisical or a prior $output_path - writing an insecure sslmode=disable default. This should only happen on a genuine first-time bootstrap."
            echo "DATABASE_URL=postgres://${db_user}:${db_pass}@${db_host}:${db_port}/${db_name}?sslmode=disable" >> "$tmp_path"
        fi
    fi
    if ! grep -q "^POSTGRES_DSN=" "$tmp_path"; then
        if [ -n "$prev_postgres_dsn" ]; then
            echo "POSTGRES_DSN=\"${prev_postgres_dsn}\"" >> "$tmp_path"
        else
            warn "No POSTGRES_DSN from Infisical or a prior $output_path - writing an insecure sslmode=disable default. This should only happen on a genuine first-time bootstrap."
            echo "POSTGRES_DSN=postgresql://${db_user}:${db_pass}@${db_host}:${db_port}/${db_name}?sslmode=disable" >> "$tmp_path"
        fi
    fi
    if ! grep -q "^REDIS_URL=" "$tmp_path"; then
        echo "REDIS_URL=redis://${db_host}:6379" >> "$tmp_path"
    fi
    # API_TOKEN_ENCRYPTION_KEY_DEV_FALLBACK removed — the previous default value
    # was committed to origin/main and represents a known credential leak. Bootstrap
    # callers must now set API_TOKEN_ENCRYPTION_KEY explicitly via .env (gitignored)
    # before invoking this script. Adding DEV_FALLBACK=true here would re-introduce
    # the silent-leak path.
    #
    # Resolution order: (1) Infisical pull (preferred - secrets in Infisical
    # are the source of truth across machines), (2) shell env (fallback -
    # set by sourcing backend/.env in main() before this runs; this is
    # how local-only setups work since the key isn't stored in Infisical
    # after the 2026-09-13 rotation). Either path is accepted; only when
    # neither has the value do we fail-fast.
    if ! grep -q "^API_TOKEN_ENCRYPTION_KEY=" "$tmp_path"; then
        if [ -n "${API_TOKEN_ENCRYPTION_KEY:-}" ]; then
            echo "API_TOKEN_ENCRYPTION_KEY=\"${API_TOKEN_ENCRYPTION_KEY}\"" >> "$tmp_path"
        else
            echo "ERROR: API_TOKEN_ENCRYPTION_KEY not set in environment. Refusing to fall back to a value that was committed to origin/main (de336a41af)." >&2
            rm -f "$tmp_path"
            return 1
        fi
    fi

    # Keycloak JWKS/issuer defaults. These are not currently stored as
    # Infisical secrets, so without this fallback the backend silently
    # ends up with no RSA public key configured and rejects every
    # user-issued JWT with "missing or invalid JWT token" (401s across
    # the app). Prefer real Infisical secrets when present; only fill in
    # gaps here.
    local kc_host="${KEYCLOAK_HOST:-100.84.50.65}"
    local kc_port="${KEYCLOAK_PORT:-8443}"
    local kc_realm="${KEYCLOAK_REALM:-uisce}"
    if ! grep -q "^KEYCLOAK_HOST=" "$tmp_path"; then
        echo "KEYCLOAK_HOST=${kc_host}" >> "$tmp_path"
    fi
    if ! grep -q "^KEYCLOAK_PORT=" "$tmp_path"; then
        echo "KEYCLOAK_PORT=${kc_port}" >> "$tmp_path"
    fi
    if ! grep -q "^KEYCLOAK_REALM=" "$tmp_path"; then
        echo "KEYCLOAK_REALM=${kc_realm}" >> "$tmp_path"
    fi
    if ! grep -q "^KEYCLOAK_JWKS_URL=" "$tmp_path"; then
        echo "KEYCLOAK_JWKS_URL=https://${kc_host}:${kc_port}/realms/${kc_realm}/protocol/openid-connect/certs" >> "$tmp_path"
    fi
    if ! grep -q "^KEYCLOAK_ISSUER=" "$tmp_path"; then
        echo "KEYCLOAK_ISSUER=https://${kc_host}:${kc_port}/realms/${kc_realm}" >> "$tmp_path"
    fi
    if ! grep -q "^KEYCLOAK_INSECURE_SKIP_VERIFY=" "$tmp_path"; then
        echo "KEYCLOAK_INSECURE_SKIP_VERIFY=true" >> "$tmp_path"
    fi

    # All composites appended. Atomic swap. If mv fails (cross-device,
    # permissions), leave the temp file in place for forensics rather
    # than partial-overwrite.
    if ! mv -f "$tmp_path" "$output_path"; then
        warn "Failed to mv $tmp_path -> $output_path (target left untouched)"
        rm -f "$tmp_path"
        return 1
    fi
}

# capture_env_value prints the current value of KEY from a .env file
# before generate_env_file truncates it, so generate_composite_secrets can
# preserve it instead of clobbering it with an insecure default. Strips a
# surrounding pair of double quotes if present (generate_env_file writes
# values quoted; hand-edited files may not be).
capture_env_value() {
    local path="$1"
    local key="$2"
    [ -f "$path" ] || return 0
    local line
    line=$(grep "^${key}=" "$path" | tail -n1)
    [ -n "$line" ] || return 0
    line="${line#${key}=}"
    line="${line%\"}"
    line="${line#\"}"
    echo "$line"
}

main() {
    parse_args "$@"
    check_infisical

    log "Pulling secrets for project=$INFISICAL_PROJECT env=$INFISICAL_ENV"

    local all_secrets
    all_secrets=$(pull_secrets)

    local root_env="$ROOT_DIR/.env"
    local backend_env="$ROOT_DIR/backend/.env"
    local frontend_env="$ROOT_DIR/frontend/.env.local"
    local calendar_env="$ROOT_DIR/calendar-service/.env"
    local rebalancing_env="$ROOT_DIR/rebalancing/.env"


    local env_vars
    env_vars=$(echo "$all_secrets" | jq -r 'if type == "array" then .[] | "\(.key)=\(.value)" elif type == "object" and length > 0 then to_entries[] | "\(.key)=\(.value)" else empty end' 2>/dev/null) || true
    if [ -n "$env_vars" ]; then
        eval "$(echo "$env_vars" | while read -r line; do [ -n "$line" ] && echo "export $line"; done)"
    fi

    # Best-effort source of pre-existing local .env files so backend-specific
    # secrets (notably API_TOKEN_ENCRYPTION_KEY, which is NOT in Infisical
    # because of the historical leak in origin/main:de336a41af and therefore
    # won't come back via the pull) are visible to
    # generate_composite_secrets' "live env" tier. Without this, the root
    # .env generation aborts with an ERROR even though the key is sitting
    # one file over in backend/.env - a misleading failure because set -e
    # makes the script abort before reaching the backend/.env block, so
    # nothing is actually broken. Idempotent: missing files just mean the
    # env-var tier finds nothing, same as before this fix. Re-adding
    # here because the previous edit dropped this block.
    if [ -f "$backend_env" ]; then
        set -a
        # shellcheck disable=SC1090
        source "$backend_env"
        set +a
    fi
    if [ -f "$rebalancing_env" ]; then
        set -a
        # shellcheck disable=SC1090
        source "$rebalancing_env"
        set +a
    fi

    # SUBSTANTIAL_TARGET_THRESHOLD: a freshly bootstrapped target has the
    # composites only (~15-20 keys, ~600 bytes). The empty-pull guard fires
    # only when the existing target has more than this many keys — meaning
    # the file was already populated from a prior bootstrap run (or
    # manually), and an empty Infisical pull would clobber that prior work.
    # A fresh-install target with this few keys is expected to receive an
    # empty pull (the project is empty), so the guard allows it through.
    SUBSTANTIAL_TARGET_THRESHOLD=50

    # Helper: did generate_env_file return the "empty pull" signal (rc=2)?
    # If so, decide whether the existing target is substantial enough to
    # refuse to clobber, or a fresh install that should be left alone.
    handle_empty_pull() {
        local target="$1"
        local section="$2"
        if [ -f "$target" ]; then
            local keys
            keys=$(grep -cE '^[A-Z_]+=' "$target" 2>/dev/null || echo 0)
            if [ "$keys" -gt "$SUBSTANTIAL_TARGET_THRESHOLD" ]; then
                warn "REFUSING TO CLOBBER $target ($keys keys > threshold $SUBSTANTIAL_TARGET_THRESHOLD)"
                warn "  Infisical pull returned zero secrets for $section"
                warn "  Existing target has substantial content - refusing to regenerate"
                warn "  Add secrets to Infisical project before re-running bootstrap"
                return 1  # signal abort
            fi
        fi
        # No file OR sparse file - empty pull is expected, leave target as-is
        log "Empty pull for $section - $target left untouched"
        return 0  # signal "skip this file, continue"
    }

    local prev_db_url prev_dsn tmp_path gen_rc=0 _bootstrap_tmp
    _bootstrap_tmp=$(mktemp) || { echo "FATAL: mktemp failed" >&2; exit 1; }
    prev_db_url=$(capture_env_value "$root_env" "DATABASE_URL")
    prev_dsn=$(capture_env_value "$root_env" "POSTGRES_DSN")
    generate_env_file "$all_secrets" "$root_env" "root" > "$_bootstrap_tmp" || gen_rc=$?
    if [ "$gen_rc" -eq 2 ]; then
        handle_empty_pull "$root_env" "root" || exit 1
    elif [ "$gen_rc" -ne 0 ]; then
        echo "FATAL: generate_env_file failed for root (rc=$gen_rc)" >&2
        exit 1
    else
        tmp_path=$(<"$_bootstrap_tmp")
        generate_composite_secrets "$root_env" "$tmp_path" "$prev_db_url" "$prev_dsn" || exit 1
    fi

    if [ -d "$ROOT_DIR/backend" ]; then
        prev_db_url=$(capture_env_value "$backend_env" "DATABASE_URL")
        prev_dsn=$(capture_env_value "$backend_env" "POSTGRES_DSN")
        generate_env_file "$all_secrets" "$backend_env" "backend" > "$_bootstrap_tmp" || gen_rc=$?
        if [ "$gen_rc" -eq 2 ]; then
            handle_empty_pull "$backend_env" "backend" || exit 1
        elif [ "$gen_rc" -ne 0 ]; then
            echo "FATAL: generate_env_file failed for backend (rc=$gen_rc)" >&2
            exit 1
        else
            tmp_path=$(<"$_bootstrap_tmp")
            generate_composite_secrets "$backend_env" "$tmp_path" "$prev_db_url" "$prev_dsn" || exit 1
        fi
    fi

    if [ -d "$ROOT_DIR/frontend" ]; then
        generate_env_file "$all_secrets" "$frontend_env" "frontend" > "$_bootstrap_tmp" || gen_rc=$?
        if [ "$gen_rc" -eq 2 ]; then
            handle_empty_pull "$frontend_env" "frontend" || exit 1
        elif [ "$gen_rc" -ne 0 ]; then
            echo "FATAL: generate_env_file failed for frontend (rc=$gen_rc)" >&2
            exit 1
        fi
    fi

    if [ -d "$ROOT_DIR/calendar-service" ]; then
        prev_db_url=$(capture_env_value "$calendar_env" "DATABASE_URL")
        prev_dsn=$(capture_env_value "$calendar_env" "POSTGRES_DSN")
        generate_env_file "$all_secrets" "$calendar_env" "calendar-service" > "$_bootstrap_tmp" || gen_rc=$?
        if [ "$gen_rc" -eq 2 ]; then
            handle_empty_pull "$calendar_env" "calendar-service" || exit 1
        elif [ "$gen_rc" -ne 0 ]; then
            echo "FATAL: generate_env_file failed for calendar-service (rc=$gen_rc)" >&2
            exit 1
        else
            tmp_path=$(<"$_bootstrap_tmp")
            generate_composite_secrets "$calendar_env" "$tmp_path" "$prev_db_url" "$prev_dsn" || exit 1
        fi
    fi

    if [ -d "$ROOT_DIR/rebalancing" ]; then
        prev_db_url=$(capture_env_value "$rebalancing_env" "DATABASE_URL")
        prev_dsn=$(capture_env_value "$rebalancing_env" "POSTGRES_DSN")
        generate_env_file "$all_secrets" "$rebalancing_env" "rebalancing" > "$_bootstrap_tmp" || gen_rc=$?
        if [ "$gen_rc" -eq 2 ]; then
            handle_empty_pull "$rebalancing_env" "rebalancing" || exit 1
        elif [ "$gen_rc" -ne 0 ]; then
            echo "FATAL: generate_env_file failed for rebalancing (rc=$gen_rc)" >&2
            exit 1
        else
            tmp_path=$(<"$_bootstrap_tmp")
            generate_composite_secrets "$rebalancing_env" "$tmp_path" "$prev_db_url" "$prev_dsn" || exit 1
        fi
    fi


    log "Done. Generated .env files:"
    [ -f "$root_env" ] && echo "  - $root_env"
    [ -f "$backend_env" ] && echo "  - $backend_env"
    [ -f "$frontend_env" ] && echo "  - $frontend_env"
    [ -f "$calendar_env" ] && echo "  - $calendar_env"
    [ -f "$rebalancing_env" ] && echo "  - $rebalancing_env"

    echo ""
    log "Next steps:"
    echo "  1. Review generated .env files (they contain secrets)"
    echo "  2. Add .env files to .gitignore if not already present"
    echo "  3. Run: docker-compose up -d"
}

main "$@"
