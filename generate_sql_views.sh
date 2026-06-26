#!/bin/bash

# DAX to SQL View Generator for Financial Services Semantic Layer
# This script generates SQL CREATE VIEW statements from the DAX-to-SQL mapping

echo "🚀 Generating SQL Views from DAX-to-SQL Mapping"
echo "=============================================="

# Read the mapping file
MAPPING_FILE="/Users/eganpj/GitHub/semlayer/dax_to_sql_mapping.json"

# Output directory for generated SQL files
OUTPUT_DIR="/Users/eganpj/GitHub/semlayer/generated_views"
mkdir -p "$OUTPUT_DIR"

echo "📁 Output directory: $OUTPUT_DIR"

# Function to extract view definitions from JSON
generate_views() {
    local json_file="$1"
    local output_dir="$2"

    # Use jq to parse JSON and generate SQL files
    # Note: This assumes jq is installed. If not, you can install it with: brew install jq

    if ! command -v jq &> /dev/null; then
        echo "❌ jq is required but not installed. Please install jq first:"
        echo "   brew install jq"
        return 1
    fi

    # Extract each mapping and create individual SQL files
    jq -r '.dax_to_sql_mapping_guide.mappings[] | @base64' "$json_file" | while read -r encoded; do
        # Decode the JSON object
        local mapping
        mapping=$(echo "$encoded" | base64 --decode)

        # Extract fields
        local metric_id
        metric_id=$(echo "$mapping" | jq -r '.metric_id')
        local view_definition
        view_definition=$(echo "$mapping" | jq -r '.view_definition')
        local directquery_compatibility
        directquery_compatibility=$(echo "$mapping" | jq -r '.directquery_compatibility')

        # Create SQL file
        local sql_file="$output_dir/${metric_id}.sql"

        cat > "$sql_file" << EOF
-- =============================================
-- Metric: $metric_id
-- DirectQuery Compatibility: $directquery_compatibility
-- Generated from DAX-to-SQL mapping
-- =============================================

$view_definition;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON ${metric_id} TO reporting_users;

EOF

        echo "✅ Generated: $sql_file"
    done

    # Create a master script to run all views
    local master_script="$output_dir/create_all_views.sql"

    cat > "$master_script" << 'EOF'
-- =============================================
-- Master Script: Create All Financial Services Views
-- Generated from DAX-to-SQL mapping
-- =============================================

-- Drop existing views (uncomment if needed)
-- DROP VIEW IF EXISTS net_interest_margin;
-- ... add other DROP statements as needed

EOF

    # Add each view creation to master script
    jq -r '.dax_to_sql_mapping_guide.mappings[] | .view_definition' "$json_file" >> "$master_script"

    cat >> "$master_script" << 'EOF'

-- =============================================
-- Verification Queries
-- =============================================

-- Example: Check if views were created successfully
-- SELECT table_name FROM information_schema.views WHERE table_schema = 'public';

-- Example: Test a specific view
-- SELECT * FROM net_interest_margin LIMIT 5;

EOF

    echo "✅ Generated master script: $master_script"
}

# Generate the views
generate_views "$MAPPING_FILE" "$OUTPUT_DIR"

echo ""
echo "🎉 SQL View Generation Complete!"
echo "================================="
echo "📂 Generated files in: $OUTPUT_DIR"
echo ""
echo "📋 Next Steps:"
echo "1. Review the generated SQL files"
echo "2. Adjust table names and column references for your schema"
echo "3. Run the master script: create_all_views.sql"
echo "4. Test the views with sample queries"
echo "5. Create indexes on frequently queried columns"
echo ""
echo "🔧 Database-specific adaptations may be needed:"
echo "   - PostgreSQL: Use appropriate syntax"
echo "   - SQL Server: Use [schema].[table] notation"
echo "   - Oracle: Adjust view syntax as needed"
echo ""
echo "⚡ Performance Tips:"
echo "   - Create indexes on entity_id, as_of_date"
echo "   - Consider materialized views for complex calculations"
echo "   - Test query folding in Power BI DirectQuery"
