#!/bin/bash
# Integration Checklist: Temporal Workflow Governance
# Run through this checklist to integrate the Temporal governance features into your platform

set -e

echo "╔════════════════════════════════════════════════════════════════╗"
echo "║  Temporal Workflow Governance - Integration Checklist         ║"
echo "║  Follow these steps in order                                  ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""

# Step 1: Backend API Routes
echo "[1/6] ✓ Backend Files Already Created"
echo "      Files:"
echo "        • backend/internal/temporal/search_attributes.go"
echo "        • backend/internal/temporal/workflow_admin.go"
echo "        • backend/internal/temporal/history_export.go"
echo "        • backend/internal/api/temporal_admin.go"
echo "      Status: READY (Copy files if not already in place)"
echo ""

# Step 2: Register Routes
echo "[2/6] NEXT: Register API Routes in Backend"
echo "      File: backend/internal/api/api.go"
echo "      Add this to your Server.RegisterRoutes() method, inside r.Route(\"/api\", func(r chi.Router) {...}):"
echo ""
echo '        import "go.temporal.io/sdk/client"'
echo '        import httpapi "github.com/eganpj/semlayer/backend/internal/api"'
echo ""
echo '        // In your r.Route("/api") block:'
echo '        httpapi.RegisterTemporalAdminRoutes(r, temporalClient)'
echo ""
echo "      Then rebuild: cd backend && go build -o server ./cmd/server"
echo ""

# Step 3: Frontend Route
echo "[3/6] Frontend Dashboard Already Created"
echo "      Files:"
echo "        • frontend/src/pages/TemporalAdminDashboard.tsx"
echo "        • frontend/src/pages/TemporalAdminDashboard.css"
echo "      Status: READY"
echo ""

# Step 4: Add frontend route
echo "[4/6] NEXT: Register Frontend Route"
echo "      File: frontend/src/AppRoutes.tsx (or your routing file)"
echo "      Add:"
echo ""
echo '        import TemporalAdminDashboard from "./pages/TemporalAdminDashboard";'
echo ""
echo '        // In your routes array:'
echo '        {'
echo '          path: "/temporal-admin",'
echo '          element: <TemporalAdminDashboard />,'
echo '          label: "Temporal Admin",'
echo '        }'
echo ""

# Step 5: Prometheus & Grafana
echo "[5/6] Monitoring Stack Configuration"
echo "      Files Updated:"
echo "        • prometheus/prometheus.yml (Temporal metrics scraping added)"
echo "        • grafana/provisioning/dashboards/temporal-workflows.json (Dashboard template added)"
echo "      Status: READY"
echo ""

# Step 6: Search Attributes
echo "[6/6] NEXT: Register Search Attributes in Temporal"
echo ""
echo "      Option A: Download and run setup script"
echo "        curl http://localhost:8080/api/temporal/setup-cli-script > setup.sh"
echo "        bash setup.sh"
echo ""
echo "      Option B: Run manually"
echo "        temporal operator search-attribute create --name BusinessUnit --type Keyword --yes"
echo "        temporal operator search-attribute create --name SlaDeadline --type Datetime --yes"
echo "        temporal operator search-attribute create --name Priority --type Int --yes"
echo "        temporal operator search-attribute create --name ProcessOwner --type Keyword --yes"
echo "        temporal operator search-attribute create --name CustomerID --type Keyword --yes"
echo "        temporal operator search-attribute create --name ProcessStatus --type Keyword --yes"
echo "        temporal operator search-attribute create --name ComplianceRisk --type Keyword --yes"
echo "        temporal operator search-attribute create --name EscalationLevel --type Int --yes"
echo "        temporal operator search-attribute create --name StartTime --type Datetime --yes"
echo "        temporal operator search-attribute create --name TenantID --type Keyword --yes"
echo ""

echo ""
echo "╔════════════════════════════════════════════════════════════════╗"
echo "║                    DEPLOYMENT STEPS                           ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""

echo "1. Update Backend Code"
echo "   └─ Edit backend/internal/api/api.go (add RegisterTemporalAdminRoutes)"
echo ""

echo "2. Update Frontend Code"
echo "   └─ Edit frontend/src/AppRoutes.tsx (add temporal-admin route)"
echo ""

echo "3. Rebuild Services"
echo "   ├─ Backend:  cd backend && go build -o server ./cmd/server"
echo "   └─ Frontend: cd frontend && npm install && npm run build"
echo ""

echo "4. Start Docker Services"
echo "   └─ docker-compose up -d"
echo ""

echo "5. Register Search Attributes"
echo "   └─ bash setup.sh  (or run manual commands above)"
echo ""

echo "6. Verify Setup"
echo "   ├─ API: curl http://localhost:8080/api/temporal/search-attributes"
echo "   ├─ Frontend: http://localhost:5173/temporal-admin"
echo "   ├─ Grafana: http://localhost:3000 (admin/admin)"
echo "   └─ Prometheus: http://localhost:9091"
echo ""

echo ""
echo "╔════════════════════════════════════════════════════════════════╗"
echo "║                    QUICK TEST COMMANDS                         ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""

echo "Test 1: Verify API Endpoints"
echo "  curl http://localhost:8080/api/temporal/search-attributes"
echo ""

echo "Test 2: Signal a Workflow"
echo "  curl -X POST http://localhost:8080/api/temporal/workflows/order-123/signal \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -d '{\"signal_name\":\"unblock\",\"reason\":\"test\"}'  "
echo ""

echo "Test 3: Access Admin Dashboard"
echo "  open http://localhost:5173/temporal-admin"
echo ""

echo "Test 4: View Grafana Dashboard"
echo "  open http://localhost:3000"
echo "  Login: admin / admin"
echo "  Select: Temporal Workflows - Real-time Monitor"
echo ""

echo "Test 5: Check Prometheus Targets"
echo "  open http://localhost:9090/targets"
echo "  Verify: temporal-server is UP"
echo ""

echo ""
echo "╔════════════════════════════════════════════════════════════════╗"
echo "║                    DOCUMENTATION                              ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""

echo "Full Implementation Guide:"
echo "  → Read: TEMPORAL_GOVERNANCE_IMPLEMENTATION.md"
echo ""

echo "Quick Start (5 minutes):"
echo "  → Read: TEMPORAL_QUICK_START.md"
echo ""

echo "Delivery Summary (overview):"
echo "  → Read: TEMPORAL_DELIVERY_SUMMARY.md"
echo ""

echo ""
echo "✅ All files created and ready for integration!"
echo "👉 Next: Follow deployment steps above"
echo ""
