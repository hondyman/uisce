#!/bin/bash

echo "=================================================="
echo "   TITAN EVENT-DRIVEN ARCHITECTURE DEMO"
echo "=================================================="

# 1. Simulate Price Drop (Should trigger 'bp_margin_call_protocol')
echo ""
echo "[Event] 📉 CRITICAL PRICE DROP DETECTED: AAPL -15%"
curl -X POST http://localhost:8090/api/events/market \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: demo-tenant" \
  -d '{
    "eventId": "EVT-001",
    "type": "PRICE_DROP",
    "symbol": "AAPL",
    "timestamp": 1704067200,
    "payload": {
        "price": 145.00,
        "prev_close": 170.00,
        "drop_pct": 0.147
    }
}'

echo ""
echo ""

# 2. Simulate Normal Info (Should be Ignored)
echo "[Event] ℹ️ DIVIDEND UPDATE: MSFT"
curl -X POST http://localhost:8090/api/events/market \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: demo-tenant" \
  -d '{
    "eventId": "EVT-002",
    "type": "DIVIDEND",
    "symbol": "MSFT",
    "timestamp": 1704067200,
    "payload": {
        "amount": 0.75
    }
}'

echo ""
echo ""
echo "✅ Demo Complete. Check server logs for workflow triggers."
