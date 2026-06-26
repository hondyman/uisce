# Semantic Mapper Improvements

## Overview
This document describes the improvements made to the semantic mapping system to provide intelligent, learning-based semantic term generation with better naming conventions.

## Backend Improvements (`semantic_mapping_service.go`)

### 1. **Prefix Removal**
- **Purpose**: Remove BI-specific prefixes that don't add semantic value
- **Prefixes Removed**: `DIM_`, `FCT_`, `FACT_`, `DIMENSION_`, `AGG_`, `TMP_`, `TEMP_`, `STG_`, `STAGE_`
- **Example**: `DIM_CUSTOMERS` → `CUSTOMER`

### 2. **Singularization**
- **Purpose**: Convert plural table/column names to singular form for consistency
- **Handles**:
  - Regular plurals: `EMPLOYEES` → `EMPLOYEE`
  - Special cases: `PEOPLE` → `PERSON`, `CHILDREN` → `CHILD`
  - IES endings: `COMPANIES` → `COMPANY`, `CATEGORIES` → `CATEGORY`
  - SES endings: `ADDRESSES` → `ADDRESS`, `STATUSES` → `STATUS`
  - XES endings: `BOXES` → `BOX`, `INDEXES` → `INDEX`
  - VES endings: `WIVES` → `WIFE`, `KNIVES` → `KNIFE`

### 3. **Underscore Separators**
- **Change**: Use underscores (`_`) instead of asterisks (`*`) as word separators
- **Reason**: More standard and readable
- **Example**: `CUSTOMER*NAME*ID` → `CUSTOMER_NAME_ID`

### 4. **Context for Generic Terms**
- **Purpose**: Add table context to generic column names for better semantic clarity
- **Generic Terms Enhanced**:
  - `BIRTH_DATE` → `EMPLOYEE_BIRTH_DATE`
  - `ADDRESS` → `CUSTOMER_ADDRESS`
  - `PHONE` → `SUPPLIER_PHONE`
  - `EMAIL` → `USER_EMAIL`
  - `CITY`, `STATE`, `ZIP`, `COUNTRY`, `FAX`, `REGION`, `DESCRIPTION`, `NOTES`

### 5. **Redundancy Removal**
- **Purpose**: Eliminate duplicate semantic components
- **Example**: `CUSTOMERS_CUSTOMER_CITY` → `CUSTOMER_CITY`
- **Logic**: Detects singular/plural forms and removes duplicates

### 6. **Uppercase Normalization**
- **Purpose**: All semantic terms are consistently uppercase
- **Example**: `customerName` → `CUSTOMER_NAME`

## Frontend Improvements (`SemanticMapper.tsx`)

### 1. **Better Text Visibility**
- **Fix**: Changed semantic term button from blue-on-blue to white text on blue background
- **Before**: `bgcolor: 'primary.light', color: 'primary.dark'`
- **After**: `bgcolor: 'primary.main', color: 'white'`

### 2. **Create New Semantic Terms**
- **Feature**: Add new semantic terms directly from the mapper
- **UI**: "➕ Create New: [TERM_NAME]" button appears when typing
- **API**: `POST /api/semantic-terms` to create new term and edge
- **Flow**:
  1. User types a term name in search box
  2. Click "Create New" button
  3. Term is created in database
  4. Mapping is updated with new term
  5. Success toast notification

### 3. **Improved Search**
- **Fix**: Added `credentials: 'include'` to search API calls
- **Feature**: Search works with tenant scope enforcement
- **UX**: Clear feedback when creating or selecting terms

## API Enhancements

### New Endpoint: Create Semantic Term
```http
POST /api/semantic-terms
Content-Type: application/json

{
  "term_name": "CUSTOMER_EMAIL",
  "description": "Custom semantic term: CUSTOMER_EMAIL"
}
```

**Response**:
```json
{
  "node_id": "uuid-here",
  "term_name": "CUSTOMER_EMAIL",
  "description": "Custom semantic term: CUSTOMER_EMAIL"
}
```

## Examples

### Before vs After

| Original Column | Old Semantic Term | New Semantic Term | Improvements |
|----------------|-------------------|-------------------|--------------|
| `employees.birth_date` | `EMPLOYEES*BIRTH*DATE` | `EMPLOYEE_BIRTH_DATE` | Singular, underscore, context |
| `dim_customers.customer_id` | `DIM*CUSTOMERS*CUSTOMER*ID` | `CUSTOMER_ID` | No prefix, singular, no redundancy |
| `fct_orders.order_date` | `FCT*ORDERS*ORDER*DATE` | `ORDER_DATE` | No prefix, singular |
| `customers.address` | `ADDRESS` | `CUSTOMER_ADDRESS` | Context added |
| `customers.customer_city` | `CUSTOMERS*CUSTOMER*CITY` | `CUSTOMER_CITY` | Singular, no redundancy |
| `employee_territories.territory_id` | `EMPLOYEE*TERRITORIES*TERRITORY*ID` | `EMPLOYEE_TERRITORY_ID` | Singular |

## Learning System (Future Enhancement)

The system is now structured to support learning from user overrides:

1. **Override Tracking**: When users manually change a semantic term, track:
   - Original suggested term
   - User's selected term
   - Column metadata (schema, table, column, data_type)

2. **Pattern Learning**: Build patterns like:
   - "Columns matching `%birth%` should use `{TABLE}_BIRTH_DATE`"
   - "Columns in tables starting with `DIM_` should have prefix removed"
   - "Plural table names should be singularized"

3. **Confidence Adjustment**: Boost confidence for patterns that match learned rules

4. **User Preferences**: Store per-tenant or per-user semantic naming preferences

## Testing

### Test Cases
1. ✅ `DIM_EMPLOYEES` → `EMPLOYEE`
2. ✅ `FCT_ORDERS` → `ORDER`
3. ✅ `employees.birth_date` → `EMPLOYEE_BIRTH_DATE`
4. ✅ `customers.address` → `CUSTOMER_ADDRESS`
5. ✅ `CUSTOMERS_CUSTOMER_CITY` → `CUSTOMER_CITY`
6. ✅ Semantic term text is white on blue (visible)
7. ✅ Can create new semantic terms from UI
8. ✅ Search works with tenant scope

### Rebuild & Test
```bash
# Backend
cd /Users/eganpj/GitHub/semlayer
docker compose up -d --build backend

# Frontend  
cd /Users/eganpj/GitHub/semlayer/frontend
npm run dev

# Test
open http://localhost:5173/semantic-mapper
```

## Configuration

No configuration required - all improvements are automatic based on:
- Column name analysis
- Table name analysis  
- Data type detection
- Built-in business rules

## Next Steps

1. **User Feedback Loop**: Add UI to collect feedback on semantic suggestions
2. **ML Model**: Train a model on approved mappings to improve suggestions
3. **Rule Engine**: Allow users to define custom naming rules
4. **Export/Import**: Share semantic term mappings between tenants
5. **Validation**: Add data profiling to validate semantic term assignments
