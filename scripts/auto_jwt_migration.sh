#!/usr/bin/env bash
# Patches Go files to use jwtmiddleware instead of X-Tenant-ID header
# Usage: run from repo root
set -eu

echo "Searching for Go files with X-Tenant-ID.."
files=$(grep -R --line-number "X-Tenant-ID" . | grep '\\.go' | cut -d: -f1 | sort -u)

for f in $files; do
    echo "Processing $f"
    # add import if missing
    if grep -q "jwt-middleware" "$f"; then
        echo "  import already present"
    else
        # insert into import block
        # handle single-line and multi-line blocks
        perl -i -pe 's|(import \()|$1\n\t"github.com/hondyman/semlayer/libs/jwt-middleware"| if $.==1' "$f" || true
        # if above fails try alternative pattern
    fi

    # replace tenant header retrieval
    perl -i -0777 -pe 's|tenantID := r\.Header\.Get\("[xX]-Tenant-ID"\)|claims := jwtmiddleware.GetClaimsFromContext(r)
if claims == nil {
    http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
    return
}
tenantID := claims.TenantID|g' "$f"

    # also check for lower-case variable names
    perl -i -0777 -pe 's|r\.Header\.Get\("[xX]-Tenant-ID"\)|
claims := jwtmiddleware.GetClaimsFromContext(r)
if claims == nil {
    http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
    return
}
tenantID := claims.TenantID|g' "$f"
done

echo "Migration script completed. Please review patches and run go build/tests."
