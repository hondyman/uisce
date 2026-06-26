# Semantic Mapper Datamart Fix

## Issue
The semantic wizard was not working for the datamart datasource. The user reported that:
- Database columns were not being matched to semantic terms
- Semantic terms were not being matched to business terms
- Catalog and edge links were not being established

## Root Cause
When the "datamart" datasource is selected, there are no actual database column nodes stored under the datamart datasource ID. The columns exist under the "alpha_dwh" datasource. 

The semantic mapping wizard had special logic to handle this (mapping datamart → alpha_dwh), but the regular semantic mapper endpoint (`/api/semantic-mappings`) and the edge creation endpoint (`/api/semantic-mappings/edges`) did not have this logic.

## Changes Made

### 1. Fixed `GenerateMappings` function
**File**: `backend/internal/analytics/semantic_mapping_service.go` (lines 1704-1741)

Added datamart → alpha_dwh datasource resolution logic to the `GenerateMappings` function to match the wizard's behavior:
- Before fetching columns, check if the datasource is named "datamart"
- If so, resolve to the alpha_dwh datasource ID
- Use the resolved ID to fetch columns
- Maintains backward compatibility for other datasources

**Why this matters**: This ensures the semantic mapper page gets the correct columns for datamart instead of returning empty results.

### 2. Fixed `/api/semantic-mappings/edges` endpoint
**File**: `backend/internal/api/api.go` (lines 2056-2160)

Added datasource resolution for edge creation:
- Build a resolution map for datamart → alpha_dwh at the start of mapping processing
- For each mapping, check if the datasource is datamart and resolve it
- Use the resolved datasource ID when calling `CreateMappingEdge`
- Added logging to track datasource resolution

**Why this matters**: When the frontend sends database column node IDs from alpha_dwh, the edges must be created using the alpha_dwh datasource ID (not datamart) to ensure they link to the correct nodes in the catalog.

### 3. Added `DB()` public method
**File**: `backend/internal/analytics/semantic_mapping_service.go` (lines 28-30)

Exposed a public method to access the database connection from the SemanticMappingService. This was needed for the API endpoint to perform datasource lookups.

## How Business Terms Work
The semantic mapping flow creates business term edges through the `ApplyEnrichment` function:

1. When mappings are applied, both semantic terms and business terms are created (or found if they exist)
2. An edge is created from semantic term → business term with relationship type "HAS_BUSINESS_TERM"
3. An edge is created from column → semantic term with relationship type "MAPS_TO"

This ensures the complete catalog structure:
```
Database Column --[MAPS_TO]--> Semantic Term --[HAS_BUSINESS_TERM]--> Business Term
```

## Testing the Fix

To test the semantic mapper with datamart:

1. Navigate to `http://localhost:5173/core/semantic-mapper`
2. Select a tenant/product/datamart datasource via the tenant picker
3. Verify columns are loaded (should show columns from the database)
4. Select columns and click "Create Edges"
5. Verify edges are created in the catalog

The wizard should now properly:
- Discover database columns from the datamart datasource
- Generate semantic term suggestions
- Create catalog nodes and edges for semantic terms and business terms
- Establish the complete semantic chain: Column → Semantic Term → Business Term

## Related Code
- `SemanticMappingService.GenerateMappings()` - Fetches columns for semantic mapping
- `SemanticMappingService.CreateMappingEdge()` - Creates MAPS_TO edges between columns and semantic terms
- `SemanticMappingService.ApplyEnrichment()` - Creates both semantic and business term nodes and their edges
- `/api/semantic-mappings` - GET endpoint that returns suggested mappings
- `/api/semantic-mappings/edges` - POST endpoint that creates edges for selected mappings
- `/api/semantic-mapping/wizard/generate` - POST endpoint for AI-powered mapping generation
