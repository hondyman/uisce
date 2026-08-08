#!/bin/bash
# UTC Compliance Checker for Schema Files
# Run this script to validate that all timestamp columns use TIMESTAMPTZ (not bare TIMESTAMP)
# Exit code 0 = all good, non-zero = violations found

set -e

SCHEMA_FILES=$(find . -name "*.sql" -type f | grep -v node_modules | grep -v ".git")

echo "=== UTC Compliance Check ==="
echo "Checking schema files for non-UTC timestamp columns..."
echo ""

VIOLATIONS=0

for file in $SCHEMA_FILES; do
    # Check for bare 'timestamp' (not 'timestamp with time zone', 'timestamptz', or 'timestampz')
    # Pattern matches 'timestamp' followed by space, comma, newline, or end of line (not 'tz')
    if grep -HE 'timestamp[^a-z]?[^_]?' "$file" 2>/dev/null | grep -vE 'TIMESTAMP WITH TIME ZONE|TIMESTAMPTZ|timestamp with time zone|timestamptz|TZ\s|time_zone|user\.timezone|system_timezone|default_timezone|list_timezones|pg_timezone' | grep -vE '^[^:]*--' | grep -vE 'timestamp_' | grep -vE '::timestamp' > /dev/null 2>&1; then

        # More precise check for bare timestamp declarations
        BARE_TIMESTAMPS=$(grep -nE '^\s*[a-zA-Z_]+\s+(TIMESTAMP|timestamp)(\s+|,\s*\w|\))' "$file" 2>/dev/null || true)

        if [ -n "$BARE_TIMESTAMPS" ]; then
            echo "❌ VIOLATION: $file"
            echo "$BARE_TIMESTAMPS" | head -5
            echo ""
            VIOLATIONS=$((VIOLATIONS + 1))
        fi
    fi
done

# Check for explicit timestamp without timezone in column definitions
for file in $SCHEMA_FILES; do
    # Match patterns like: column_name timestamp NOT NULL
    VIOLATIONS_FOUND=$(grep -nE '\s+[a-z_]+\s+timestamp\s+(NOT\s+NULL|NULL|DEFAULT|\,|\))' "$file" 2>/dev/null || true)

    if [ -n "$VIOLATIONS_FOUND" ]; then
        echo "❌ VIOLATION: $file contains bare timestamp columns:"
        echo "$VIOLATIONS_FOUND"
        echo ""
        VIOLATIONS=$((VIOLATIONS + 1))
    fi
done

echo "=== Summary ==="
if [ $VIOLATIONS -eq 0 ]; then
    echo "✅ All schema files are UTC compliant"
    exit 0
else
    echo "❌ Found $VIOLATIONS violation(s) - schema files must use TIMESTAMPTZ for UTC compliance"
    exit 1
fi
