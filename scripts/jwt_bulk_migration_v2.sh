#!/bin/bash

# JWT Bulk Migration Script v2
# Systematically patches all Go files with X-Tenant-ID header access
# to use JWT claims instead

set -e

WORK_DIR="/Users/eganpj/GitHub/semlayer"
cd "$WORK_DIR"

echo "Starting JWT bulk migration v2..."
echo "Target: Replace all r.Header.Get(\"X-Tenant-ID\") with JWT claims"

# Find all Go files (exclude test files and vendor)
GO_FILES=$(find backend internal mdm-service calendar-service -name "*.go" -type f | grep -v test | grep -v vendor | sort)

PROCESSED=0
UPDATED=0

for FILE in $GO_FILES; do
    # Check if file contains X-Tenant-ID
    if ! grep -q "X-Tenant-ID" "$FILE"; then
        continue
    fi
    
    PROCESSED=$((PROCESSED + 1))
    
    # Backup original
    cp "$FILE" "$FILE.backup"
    
    # Check if file already has jwt-middleware import
    if ! grep -q "github.com/hondyman/semlayer/libs/jwt-middleware" "$FILE"; then
        # Add import after "import ("
        # Handle both brace styles
        if grep -q "^import (" "$FILE"; then
            # Multi-line import block
            sed -i '' '/^import (/a\
\	"github.com/hondyman/semlayer/libs/jwt-middleware"
' "$FILE" || true
        elif grep -q "^import$" "$FILE"; then
            # Single resource imports - convert to block
            sed -i '' 's/^import$/import (\n\t"github.com\/hondyman\/semlayer\/libs\/jwt-middleware"\n)/' "$FILE" || true
        fi
    fi
    
    # Replace common patterns
    # Pattern 1: tenantID := r.Header.Get("X-Tenant-ID")
    sed -i '' 's/tenantID := r\.Header\.Get("X-Tenant-ID")/claims := jwtmiddleware.GetClaimsFromContext(r)\n\tif claims == nil {\n\t\thttp.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)\n\t\treturn\n\t}\n\ttenantID := claims.TenantID/' "$FILE" || true
    
    # Pattern 2: tenantID := c.GetHeader("X-Tenant-ID") (Gin context)
    sed -i '' 's/tenantID := c\.GetHeader("X-Tenant-ID")/claims := jwtmiddleware.GetGinClaimsFromContext(c)\n\tif claims == nil {\n\t\tc.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"})\n\t\treturn\n\t}\n\ttenantID := claims.TenantID/' "$FILE" || true
    
    # Pattern 3: var tenantID := r.Header.Get("X-Tenant-ID")
    sed -i '' 's/tenantID := r\.Header\.Get("X-Tenant-ID")/claims := jwtmiddleware.GetClaimsFromContext(r); tenantID := claims.TenantID/' "$FILE" || true
    
    # Pattern 4: if tenantID := r.Header.Get("X-Tenant-ID"); ...
    sed -i '' 's/if tenantID := r\.Header\.Get("X-Tenant-ID"); /if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil {\n\t\ttenantID := claims.TenantID\n\t\t/' "$FILE" || true
    
    UPDATED=$((UPDATED + 1))
    echo "Processed: $FILE"
done

echo ""
echo "Migration complete!"
echo "Files processed: $PROCESSED"
echo "Files updated: $UPDATED"
echo ""
echo "Next steps:"
echo "1. Review changes: git diff backend/ internal/ mdm-service/ calendar-service/"
echo "2. Run: cd backend && go mod tidy && go build ./..."
echo "3. Check for compilation errors and fix manually as needed"
