#!/bin/bash
# Test wealth transfer API endpoints

set -e

echo "========================================="
echo "Wealth Transfer API Testing"
echo "=========================================
"

# Base URL
BASE_URL="http://localhost:8080"
TENANT_ID="00000000-0000-0000-0000-000000000001"
FAMILY_ID="00000000-0000-0000-0000-000000000001"
MEMBER_ID_JOHN="00000000-0000-0000-0001-000000000001"
MEMBER_ID_MARY="00000000-0000-0000-0001-000000000002"

echo "1. Testing Family Office endpoints..."
echo ""

# Get family office
echo "GET /api/wealth-transfer/families/${FAMILY_ID}"
curl -s "${BASE_URL}/api/wealth-transfer/families/${FAMILY_ID}" | jq '.' || echo "Failed"
echo ""

# List families for tenant
echo "GET /api/wealth-transfer/families?tenant_id=${TENANT_ID}"
curl -s "${BASE_URL}/api/wealth-transfer/families?tenant_id=${TENANT_ID}" | jq '.' || echo "Failed"
echo ""

# Get family members
echo "GET /api/wealth-transfer/families/${FAMILY_ID}/members"
curl -s "${BASE_URL}/api/wealth-transfer/families/${FAMILY_ID}/members" | jq '.' || echo "Failed"
echo ""

# Get family tree
echo "GET /api/wealth-transfer/families/${FAMILY_ID}/tree"
curl -s "${BASE_URL}/api/wealth-transfer/families/${FAMILY_ID}/tree" | jq '.' || echo "Failed"
echo ""

# Get family profile
echo "GET /api/wealth-transfer/families/${FAMILY_ID}/profile"
curl -s "${BASE_URL}/api/wealth-transfer/families/${FAMILY_ID}/profile" | jq '.' || echo "Failed"
echo ""

echo "========================================="
echo "2. Testing Tax Calculation endpoints..."
echo "========================================="
echo ""

# Calculate federal estate tax
echo "POST /api/wealth-transfer/tax/estate/federal"
curl -s -X POST "${BASE_URL}/api/wealth-transfer/tax/estate/federal" \
  -H "Content-Type: application/json" \
  -d '{
    "gross_estate": "25000000",
    "prior_exemption_used": "0"
  }' | jq '.' || echo "Failed"
echo ""

# Calculate state tax (California - no estate tax)
echo "POST /api/wealth-transfer/tax/estate/state (CA)"
curl -s -X POST "${BASE_URL}/api/wealth-transfer/tax/estate/state" \
  -H "Content-Type: application/json" \
  -d '{
    "state_code": "CA",
    "gross_estate": "25000000"
  }' | jq '.' || echo "Failed"
echo ""

# Calculate state tax (New York - has estate tax)
echo "POST /api/wealth-transfer/tax/estate/state (NY)"
curl -s -X POST "${BASE_URL}/api/wealth-transfer/tax/estate/state" \
  -H "Content-Type: application/json" \
  -d '{
    "state_code": "NY",
    "gross_estate": "25000000"
  }' | jq '.' || echo "Failed"
echo ""

# Calculate combined tax
echo "POST /api/wealth-transfer/tax/estate/combined"
curl -s -X POST "${BASE_URL}/api/wealth-transfer/tax/estate/combined" \
  -H "Content-Type: application/json" \
  -d '{
    "state_code": "NY",
    "gross_estate": "25000000",
    "prior_federal_exemption_used": "0"
  }' | jq '.' || echo "Failed"
echo ""

# Calculate gift tax
echo "POST /api/wealth-transfer/tax/gift"
curl -s -X POST "${BASE_URL}/api/wealth-transfer/tax/gift" \
  -H "Content-Type: application/json" \
  -d '{
    "gift_value": "100000",
    "annual_exclusion_used_this_year": "0",
    "lifetime_exemption_used_prior": "0",
    "spousal_split": false
  }' | jq '.' || echo "Failed"
echo ""

# Calculate GST tax
echo "POST /api/wealth-transfer/tax/gst"
curl -s -X POST "${BASE_URL}/api/wealth-transfer/tax/gst" \
  -H "Content-Type: application/json" \
  -d '{
    "transfer_value": "5000000",
    "gst_exemption_used_prior": "0",
    "generations_skipped": 2
  }' | jq '.' || echo "Failed"
echo ""

echo "========================================="
echo "3. Testing Gift History endpoints..."
echo "========================================="
echo ""

# Record a gift
echo "POST /api/wealth-transfer/gifts"
GIFT_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/wealth-transfer/gifts" \
  -H "Content-Type: application/json" \
  -d "{
    \"family_id\": \"${FAMILY_ID}\",
    \"donor_member_id\": \"${MEMBER_ID_JOHN}\",
    \"recipient_member_id\": \"${MEMBER_ID_MARY}\",
    \"gift_date\": \"2025-01-15T00:00:00Z\",
    \"gift_type\": \"ANNUAL_EXCLUSION\",
    \"asset_description\": \"Cash gift\",
    \"fair_market_value\": \"18500\",
    \"valuation_method\": \"MARKET_PRICE\",
    \"valuation_discount_pct\": \"0\",
    \"spousal_split_election\": false,
    \"is_generation_skipping\": false
  }")

echo "$GIFT_RESPONSE" | jq '.' || echo "Failed"
GIFT_ID=$(echo "$GIFT_RESPONSE" | jq -r '.gift_id')
echo ""

# Get gift history
echo "GET /api/wealth-transfer/families/${FAMILY_ID}/gifts"
curl -s "${BASE_URL}/api/wealth-transfer/families/${FAMILY_ID}/gifts" | jq '.' || echo "Failed"
echo ""

# Get exemption summary
echo "GET /api/wealth-transfer/families/${FAMILY_ID}/members/${MEMBER_ID_JOHN}/exemptions"
curl -s "${BASE_URL}/api/wealth-transfer/families/${FAMILY_ID}/members/${MEMBER_ID_JOHN}/exemptions" | jq '.' || echo "Failed"
echo ""

# Get pending Form 709 filings
echo "GET /api/wealth-transfer/families/${FAMILY_ID}/gifts/pending-form-709"
curl -s "${BASE_URL}/api/wealth-transfer/families/${FAMILY_ID}/gifts/pending-form-709" | jq '.' || echo "Failed"
echo ""

echo "========================================="
echo "4. Testing Trust Entity endpoints..."
echo "========================================="
echo ""

# Create a trust
echo "POST /api/wealth-transfer/trusts"
TRUST_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/wealth-transfer/trusts" \
  -H "Content-Type: application/json" \
  -d "{
    \"family_id\": \"${FAMILY_ID}\",
    \"entity_type\": \"SLAT\",
    \"entity_name\": \"Smith Family SLAT\",
    \"formation_date\": \"2025-01-01T00:00:00Z\",
    \"formation_state\": \"CA\",
    \"grantor_member_ids\": [\"${MEMBER_ID_JOHN}\"],
    \"trustee_member_ids\": [\"${MEMBER_ID_MARY}\"],
    \"beneficiary_member_ids\": [\"${MEMBER_ID_MARY}\"],
    \"terms\": {
      \"distribution_standard\": \"HEMS\",
      \"spendthrift_clause\": true
    }
  }")

echo "$TRUST_RESPONSE" | jq '.' || echo "Failed"
TRUST_ID=$(echo "$TRUST_RESPONSE" | jq -r '.entity_id')
echo ""

# Get trust
echo "GET /api/wealth-transfer/trusts/${TRUST_ID}"
curl -s "${BASE_URL}/api/wealth-transfer/trusts/${TRUST_ID}" | jq '.' || echo "Failed"
echo ""

# List trusts for family
echo "GET /api/wealth-transfer/families/${FAMILY_ID}/trusts"
curl -s "${BASE_URL}/api/wealth-transfer/families/${FAMILY_ID}/trusts" | jq '.' || echo "Failed"
echo ""

# Validate trust compliance
echo "GET /api/wealth-transfer/trusts/${TRUST_ID}/compliance"
curl -s "${BASE_URL}/api/wealth-transfer/trusts/${TRUST_ID}/compliance" | jq '.' || echo "Failed"
echo ""

# Calculate trust value
echo "GET /api/wealth-transfer/trusts/${TRUST_ID}/value"
curl -s "${BASE_URL}/api/wealth-transfer/trusts/${TRUST_ID}/value" | jq '.' || echo "Failed"
echo ""

echo "========================================="
echo "Testing Complete!"
echo "========================================="
echo ""
echo "Summary:"
echo "  - Family Office API: ✓ Tested"
echo "  - Tax Calculation API: ✓ Tested"
echo "  - Gift History API: ✓ Tested"
echo "  - Trust Entity API: ✓ Tested"
echo ""
echo "Next steps:"
echo "  1. Review API responses above"
echo "  2. Test with Postman/Insomnia collection"
echo "  3. Integrate with frontend UI"
echo ""
