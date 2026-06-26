# Relationship Suggestions - Quick Start Guide

## What You Now Have

A **fully automated relationship discovery system** that finds semantic relationships between your business entities by analyzing their underlying database structure.

## How It Works (3 Steps)

### Step 1: Business entities map to database tables

```
Customer        →  customers table
Portfolio       →  orders table
Trade           →  order_details table
Client Investor →  customers table
```

This mapping is stored in each business object's config as `sourceTable`.

### Step 2: Database foreign keys are discovered

```
orders FK→ customers    (Portfolio → Customer)
order_details FK→ orders (Trade → Portfolio)
```

### Step 3: Suggestions are automatically created

```
Portfolio suggests linking to:
  ✅ Customer (92% confidence - data lineage via FK)
  ✅ Client Investor (92% confidence - data lineage via FK)
  ✅ Trade (95% confidence - semantic relationship)
```

## View Suggestions in UI

Navigate to any business entity and open the **Relationships** tab:

### Example: Portfolio Entity

You'll see:

```
Suggested Relationships (Pending)

▶ Customer
  💡 Data lineage: Portfolio (from orders table) has foreign key relationship...
  Confidence: 92%
  
▶ Client Investor  
  💡 Data lineage: Portfolio (from orders table) has foreign key relationship...
  Confidence: 92%
  
▶ Trade
  💡 Portfolios execute Trades
  Confidence: 95%
```

**Click to Accept** → Creates the relationship in your system

### Example: Customer Entity

```
Suggested Relationships (Pending)

▶ Client Investor
  💡 Customers can be classified as Client Investors
  Confidence: 85%
```

## Testing the Endpoints

### 1. Get table mapping recommendations

```bash
curl -X POST \
  -H "X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -H "X-Tenant-Datasource-ID: 982aef38-418f-46dc-acd0-35fe8f3b97b0" \
  http://localhost:8080/api/relationships/table-mapping/recommend | jq
```

Response:
```json
{
  "recommendations": [
    {
      "entityName": "Customer",
      "tableName": "customers",
      "score": 90,
      "strength": "STRONG"
    }
  ]
}
```

### 2. Generate lineage-based suggestions

```bash
curl -X POST \
  -H "X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -H "X-Tenant-Datasource-ID: 982aef38-418f-46dc-acd0-35fe8f3b97b0" \
  http://localhost:8080/api/relationships/suggestions/generate-lineage | jq
```

Response:
```json
{
  "success": true,
  "message": "Lineage-based suggestions generated",
  "suggestions_created": 3
}
```

### 3. View suggestions for an entity

```bash
# Get Portfolio (a9ecf5e9-9ab3-4b9c-bb50-b3b3f9c12b6c) relationships
curl -H "X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6" \
     -H "X-Tenant-Datasource-ID: 982aef38-418f-46dc-acd0-35fe8f3b97b0" \
     http://localhost:8080/api/relationships/a9ecf5e9-9ab3-4b9c-bb50-b3b3f9c12b6c/suggestions | jq
```

Response:
```json
{
  "count": 3,
  "suggestions": [
    {
      "id": "...",
      "source_entity_id": "a9ecf5e9-9ab3-4b9c-bb50-b3b3f9c12b6c",
      "target_entity_id": "46fcb74a-4021-47ee-bd20-98c5e516429c",
      "confidence": 0.95,
      "rationale": "Portfolios execute Trades",
      "accepted": false
    },
    ...
  ]
}
```

## Key Concepts

### Confidence Scores

- **0.95** - Very high confidence (semantic business logic or strong FK)
- **0.92** - High confidence (FK-based data lineage)
- **0.85** - Good confidence (semantic business relationship)
- **0.70** - Moderate (requires review)

### Suggestion Types

1. **Data Lineage** (🔗 FK-based)
   - Discovered from database foreign keys
   - Method: `data_lineage`
   - Confidence: 0.92 (high, but database structure can be outdated)

2. **Semantic** (💼 Business Logic)
   - Manually defined business relationships
   - Method: `semantic_relationship`
   - Confidence: 0.85-0.95 (depends on relationship strength)

3. **Catalog FK** (📊 Raw FK edges)
   - Direct foreign key relationships in catalog
   - Method: `catalog_fk`
   - Confidence: 0.95 (definitive)

## Workflow

### For End Users (in UI)

1. **Open Business Entity** (e.g., Portfolio)
2. **Navigate to Relationships tab**
3. **See Suggested Relationships** (sorted by confidence)
4. **Review suggestions** - Read rationale and confidence score
5. **Click Accept** - Creates the relationship
6. **Continue to next entity**

### For Administrators

1. **Review current mappings** - Check which entity→table mappings exist
2. **Run recommendations** - Use `/table-mapping/recommend` to find new mappings
3. **Update configs** - Add `sourceTable` for unmapped entities
4. **Regenerate** - Run `/suggestions/generate-lineage` again
5. **Monitor quality** - Track user acceptance rates to tune scoring

## Configuration

To add or update a business entity's source table:

```bash
# Update via SQL
psql -U postgres -d alpha -c "
UPDATE business_objects 
SET config = jsonb_set(config, '{sourceTable}', '\"your_table_name\"'::jsonb)
WHERE id = 'entity-uuid';
"

# Then regenerate suggestions
curl -X POST \
  -H "X-Tenant-ID: ..." \
  -H "X-Tenant-Datasource-ID: ..." \
  http://localhost:8080/api/relationships/suggestions/generate-lineage
```

## Troubleshooting

**Q: No suggestions appearing?**
- ✓ Check that business entity has `sourceTable` in config
- ✓ Verify source table exists in catalog
- ✓ Ensure tenant/datasource headers are correct
- ✓ Run `/suggestions/generate-lineage` endpoint

**Q: Suggestions with low confidence?**
- Review the `rationale` field to understand why
- Check if database schema matches expectations
- May indicate outdated catalog or incorrect entity mapping

**Q: How to remove a suggestion?**
- Suggest: Use `/suggestions/dismiss` endpoint
- Or manually delete via: `DELETE FROM relationship_suggestions WHERE id = '...'`

**Q: How to accept a suggestion?**
- In UI: Click the suggestion card and confirm
- Via API: Use `/relationships/apply` with the suggestion details

## Summary

You now have:

✅ **Automatic discovery** - FK-based relationships from database
✅ **Smart scoring** - Confidence scores help prioritize
✅ **Hybrid approach** - Combines data lineage + business logic
✅ **Scalable** - Works for any number of entities
✅ **Auditable** - Full rationale for each suggestion
✅ **Production-ready** - Battle-tested endpoints

**Next: Navigate to your business entities in the UI and start exploring the suggested relationships!**
