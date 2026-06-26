# Confidence Breakdown Quick Reference

## Overview
The confidence breakdown feature provides transparency into how AI-generated business term suggestions are scored, allowing users to make informed decisions about accepting or rejecting mappings.

## Confidence Score Components

### 1. Name Similarity (50% weight)
**What it measures**: How closely the semantic term name matches the business term name

**Scoring factors**:
- **Exact match**: 1.0 (perfect)
- **Jaccard similarity**: Compares word sets
  - Example: "cust_name" vs "Customer Name" = 0.66
- **Levenshtein distance**: Character-level similarity
  - "firstname" vs "first_name" = 0.90
- **Abbreviation expansion**: Database-driven expansions
  - "acct_num" → "account_number" → match with "Account Number"
  - Adds 0.1-0.3 bonus for expansions

**Example breakdown**:
```json
{
  "label": "Name similarity",
  "score": 0.92,
  "weight": 0.5,
  "details": "Expanded 2 variations, Jaccard: 0.85, Levenshtein: 0.90"
}
```

**Interpretation**:
- **0.9-1.0**: Strong match (likely same concept)
- **0.7-0.9**: Good match (probably related)
- **0.5-0.7**: Moderate match (possibly related)
- **< 0.5**: Weak match (likely different concepts)

---

### 2. Profile Alignment (35% weight)
**What it measures**: How similar the data characteristics are between the semantic term and business term

**Scoring factors**:
- **Frequent values overlap**: Common values between columns
  - Example: Both columns have ["John", "Jane", "Michael"] = high overlap
- **Pattern matching**: Similar data patterns
  - Email pattern: `.*@.*\..*`
  - Phone pattern: `\d{3}-\d{3}-\d{4}`
  - Date pattern: `\d{4}-\d{2}-\d{2}`
- **Cardinality similarity**: Similar uniqueness
  - Both columns have ~1000 unique values out of 1000 rows = high similarity
- **Statistical properties**: Distribution similarity (optional)

**Example breakdown**:
```json
{
  "label": "Profile alignment",
  "score": 0.78,
  "weight": 0.35,
  "details": "65% value overlap, 80% pattern overlap, similar cardinality"
}
```

**Interpretation**:
- **0.8-1.0**: Very similar data characteristics
- **0.6-0.8**: Similar data characteristics
- **0.4-0.6**: Some overlap in characteristics
- **< 0.4**: Different data characteristics

---

### 3. Data Type Alignment (15% weight)
**What it measures**: Whether the data types are compatible

**Compatibility matrix**:

| Semantic Type | Compatible Business Types | Score |
|---------------|---------------------------|-------|
| VARCHAR       | VARCHAR, TEXT, CHAR       | 1.0   |
| INTEGER       | INTEGER, BIGINT, SMALLINT | 1.0   |
| DECIMAL       | DECIMAL, NUMERIC, FLOAT   | 1.0   |
| DATE          | DATE, TIMESTAMP           | 1.0   |
| BOOLEAN       | BOOLEAN, BIT              | 1.0   |
| VARCHAR       | INTEGER                   | 0.3   |
| INTEGER       | VARCHAR                   | 0.5   |

**Example breakdown**:
```json
{
  "label": "Data type alignment",
  "score": 1.0,
  "weight": 0.15,
  "details": "Compatible: VARCHAR → TEXT"
}
```

**Interpretation**:
- **1.0**: Perfect type match
- **0.5-0.9**: Compatible with implicit conversion
- **< 0.5**: Incompatible types (mapping may be incorrect)

---

## Final Confidence Calculation

**Formula**:
```
Final Confidence = (Name Similarity × 0.5) + (Profile Alignment × 0.35) + (Data Type × 0.15)
```

**Example**:
```
Name Similarity:     0.92 × 0.5  = 0.460
Profile Alignment:   0.78 × 0.35 = 0.273
Data Type:           1.0  × 0.15 = 0.150
                                 -------
Final Confidence:                 0.883
```

---

## Color-Coded Confidence Levels

| Range | Color | Label | Recommendation |
|-------|-------|-------|----------------|
| 0.85 - 1.0 | 🟢 Dark Green (`#00966b`) | High | Accept with confidence |
| 0.70 - 0.85 | 🟢 Light Green (`#4caf50`) | Good | Likely correct, review |
| 0.50 - 0.70 | 🟡 Yellow (`#ffeb3b`) | Moderate | Review carefully |
| 0.30 - 0.50 | 🟠 Orange (`#ff9800`) | Low | Probably incorrect |
| 0.0 - 0.30 | 🔴 Red (`#f44336`) | Very Low | Likely incorrect |

---

## UI Visualization Examples

### Badge Display
```
┌─────────────────────────────────────┐
│ Customer First Name                 │
│ ┌─────────┐                         │
│ │ 88% 🟢  │  Strong match           │
│ └─────────┘                         │
└─────────────────────────────────────┘
```

### Hover Tooltip
```
┌───────────────────────────────────┐
│ Confidence: 88%                   │
├───────────────────────────────────┤
│ Name Similarity:    92% (46pts)   │
│ Profile Alignment:  78% (27pts)   │
│ Data Type:         100% (15pts)   │
└───────────────────────────────────┘
```

### Detailed Modal
```
╔══════════════════════════════════════╗
║  Confidence Breakdown                ║
╠══════════════════════════════════════╣
║                                      ║
║  Overall Score: 88%                  ║
║                                      ║
║  ┌────────────────────────────────┐  ║
║  │ Name Similarity (50% weight)   │  ║
║  │ Score: 92%                     │  ║
║  │ ████████████████████░░ 46pts   │  ║
║  │                                │  ║
║  │ Details:                       │  ║
║  │ • Expanded 2 abbreviations     │  ║
║  │ • Jaccard: 85%                 │  ║
║  │ • Levenshtein: 90%             │  ║
║  └────────────────────────────────┘  ║
║                                      ║
║  ┌────────────────────────────────┐  ║
║  │ Profile Alignment (35% weight) │  ║
║  │ Score: 78%                     │  ║
║  │ ██████████████░░░░░░ 27pts     │  ║
║  │                                │  ║
║  │ Details:                       │  ║
║  │ • 65% value overlap            │  ║
║  │ • 80% pattern overlap          │  ║
║  │ • Similar cardinality          │  ║
║  └────────────────────────────────┘  ║
║                                      ║
║  ┌────────────────────────────────┐  ║
║  │ Data Type (15% weight)         │  ║
║  │ Score: 100%                    │  ║
║  │ ████████████████████ 15pts     │  ║
║  │                                │  ║
║  │ Details:                       │  ║
║  │ • Compatible: VARCHAR → TEXT   │  ║
║  └────────────────────────────────┘  ║
║                                      ║
║        [Close]                       ║
╚══════════════════════════════════════╝
```

---

## API Response Example

```json
{
  "suggestions": [
    {
      "business_term_id": "123e4567-e89b-12d3-a456-426614174000",
      "term_name": "Customer First Name",
      "confidence": 0.883,
      "reason": "Strong name similarity with abbreviation expansion, Good profile alignment",
      "description": "The first name of a customer",
      "categories": ["Customer", "Personal Information"],
      "confidence_breakdown": [
        {
          "label": "Name similarity",
          "score": 0.92,
          "weight": 0.5,
          "details": "Expanded 2 variations (cust→customer, nm→name), Jaccard similarity: 0.85, Levenshtein distance: 0.90"
        },
        {
          "label": "Profile alignment",
          "score": 0.78,
          "weight": 0.35,
          "details": "65% frequent value overlap, 80% pattern overlap (name pattern detected), similar cardinality: 985 vs 1003 unique values"
        },
        {
          "label": "Data type alignment",
          "score": 1.0,
          "weight": 0.15,
          "details": "Compatible types: VARCHAR(50) → TEXT"
        }
      ]
    }
  ]
}
```

---

## Debugging Low Confidence Scores

### Scenario 1: Low Name Similarity (< 0.5)
**Possible causes**:
- Term names are completely different
- Abbreviations not in database
- Spelling differences

**Solutions**:
- Add abbreviations to database
- Use synonyms table
- Manual review and accept if data profiles align

---

### Scenario 2: Low Profile Alignment (< 0.4)
**Possible causes**:
- Different data distributions
- Different cardinalities
- No common values

**Solutions**:
- Check if columns represent same concept differently
- Review sample values
- May be incorrect mapping

---

### Scenario 3: Low Data Type Score (< 0.5)
**Possible causes**:
- Type mismatch (e.g., VARCHAR vs INTEGER)
- Incompatible types

**Solutions**:
- Likely incorrect mapping
- Consider transformation logic
- Reject suggestion

---

## Weight Tuning Recommendations

Current weights (Name: 50%, Profile: 35%, Type: 15%) work well for most cases, but can be adjusted:

### Increase Name Weight (60-70%)
**When**: You have clean, consistent naming conventions
**Effect**: Rewards exact/similar names more

### Increase Profile Weight (40-50%)
**When**: Names are inconsistent but data patterns are reliable
**Effect**: Focuses more on data characteristics

### Increase Type Weight (20-30%)
**When**: Type compatibility is critical for your use case
**Effect**: Penalizes type mismatches more heavily

---

## Future Enhancements

1. **Machine Learning Weight Adjustment**
   - Learn optimal weights from user feedback
   - Per-tenant weight customization

2. **Historical Confidence Tracking**
   - Track accuracy of predictions over time
   - Adjust algorithm based on acceptance rates

3. **Domain-Specific Scoring**
   - Finance-specific term matching
   - Healthcare-specific term matching
   - Custom industry rules

4. **Contextual Boosting**
   - Related terms already mapped
   - Table context (e.g., "customer" table → boost customer terms)
   - Schema proximity

---

## Best Practices

1. **Accept High Confidence (> 0.85)**
   - Review but usually correct
   - Quick wins for bulk mapping

2. **Review Moderate (0.5 - 0.85)**
   - Check breakdown details
   - Look at data samples
   - Use judgment

3. **Reject Low (< 0.5)**
   - Rarely correct
   - May need manual mapping
   - Provide rejection reason for learning

4. **Monitor Patterns**
   - Track which component drives decisions
   - Identify areas for improvement
   - Update abbreviation database

---

## Support & Troubleshooting

**Issue**: Confidence always 0
- Check if abbreviation service initialized
- Verify database connection
- Check logs: `docker compose logs backend | grep confidence`

**Issue**: Breakdown not showing
- Verify API response includes `confidence_breakdown`
- Check frontend type definitions
- Inspect network tab in browser

**Issue**: Colors not displaying
- Check CSS classes applied
- Verify color mapping function
- Check Material-UI theme

**Questions?**
- Review `BUSINESS_TERM_ENHANCEMENTS.md`
- Check implementation in `semantic_matching_enhancements.go`
- Test with `TESTING_BUSINESS_TERM_UPDATES.md`
