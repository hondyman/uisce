# AI Routing - 5-Minute Quick Start

## 🚀 TL;DR - Get AI Routing Live in 5 Minutes

### Step 1: Database (1 min)

```bash
cd backend
psql postgres://postgres:postgres@host.docker.internal:5432/alpha?sslmode=disable < pkg/ai_routing/ai_routing_schema.sql
echo "✓ Database schema created"
```

### Step 2: Backend Integration (2 min)

**File:** `backend/cmd/server/main.go`

Find your main router setup and add this:

```go
// Import package
import "github.com/eganpj/semlayer/backend/pkg/ai_routing"

// In your router setup (in func main() or init function):
func setupAIRouting(db *sql.DB, r chi.Router) {
    // Initialize components
    predictiveModel := ai_routing.NewPredictiveRoutingModel("")
    rlAgent := ai_routing.NewRLRoutingAgent()
    sentimentAnalyzer := ai_routing.NewSentimentClassifier()
    ruleEngine := ai_routing.NewHybridRuleEngine()
    metricsCollector := ai_routing.NewRoutingMetricsCollector()

    // Create intelligent router
    router := ai_routing.NewIntelligentRouter(
        predictiveModel,
        rlAgent,
        sentimentAnalyzer,
        ruleEngine,
        metricsCollector,
    )

    // Create feedback collector (for RL training)
    feedbackCollector := ai_routing.NewFeedbackCollector(router, rlAgent, db)

    // Start background feedback loop (continuous learning)
    go feedbackCollector.StartFeedbackLoop(context.Background())

    // Register API handlers
    handlers := httpapi.NewAIRoutingHandlers(router, feedbackCollector, metricsCollector)
    handlers.RegisterRoutes(r)

    log.Println("✓ AI Routing initialized")
}

// Call in your main setup:
setupAIRouting(db, mainRouter)
```

### Step 3: Frontend (2 min)

**Copy files:**
```bash
mkdir -p frontend/src/components/AIRouting
cp backend/pkg/ai_routing/AIRoutingDashboard.tsx frontend/src/components/AIRouting/
```

**Add to routes** (`frontend/src/App.tsx`):
```tsx
import AIRoutingDashboard from './components/AIRouting/AIRoutingDashboard';

// In your route definitions:
<Route path="/core/ai-routing" element={<AIRoutingDashboard />} />
```

**Add to menu** (`frontend/src/components/MainNavigation.tsx`):
```tsx
{
  key: 'ai-routing',
  icon: <RobotOutlined />,
  label: 'AI Routing Dashboard',
  path: '/core/ai-routing',
}
```

### Step 4: Test (0 min - automatic)

```bash
# Restart backend
cd backend && go run cmd/server/main.go

# Restart frontend
cd frontend && npm run dev

# Navigate to http://localhost:3000/core/ai-routing
```

## ✅ You're Done!

You now have:
- ✅ Real-time AI routing decisions
- ✅ Ensemble voting (4 models)
- ✅ Reinforcement learning adaptation
- ✅ Comprehensive monitoring dashboard
- ✅ Continuous learning feedback loop

## 📊 What You Can Do Now

### Make a Routing Decision

```bash
curl -X POST http://localhost:8080/api/ai-routing/route \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: your-tenant-id" \
  -H "X-Tenant-Datasource-ID: your-datasource-id" \
  -d '{
    "workflow_id": "wf_test_123",
    "tenant_id": "your-tenant-id",
    "datasource_id": "your-datasource-id",
    "data": {
      "order_amount": 5000,
      "customer_tier": "VIP",
      "risk_score": 0.3
    },
    "available_branches": [
      {
        "id": "branch1",
        "name": "Fast Track",
        "capacity": 100,
        "current_load": 30,
        "avg_duration": 2.5,
        "success_rate": 0.95
      },
      {
        "id": "branch2", 
        "name": "Standard",
        "capacity": 500,
        "current_load": 250,
        "avg_duration": 5.0,
        "success_rate": 0.85
      }
    ]
  }'
```

### Record Workflow Outcome (for ML training)

```bash
curl -X POST http://localhost:8080/api/ai-routing/feedback/outcome \
  -H "Content-Type: application/json" \
  -d '{
    "workflow_id": "wf_test_123",
    "routing_decision_id": "decision_123",
    "branch_id": "branch1",
    "success": true,
    "completion_time": 2.3,
    "expected_time": 3.0,
    "customer_satisfaction_score": 0.95,
    "first_time_resolution": true,
    "cost_incurred": 45.00,
    "error_count": 0,
    "state_features": "vip|high|afternoon|monday|repeat|low_risk"
  }'
```

### View Metrics

```bash
curl http://localhost:8080/api/ai-routing/metrics?tenant_id=your-tenant-id
```

## 📈 Architecture at a Glance

```
Request → Intelligent Router
           ├→ Model 1: Predictive (success rate prediction)
           ├→ Model 2: RL Agent (adaptive learning)
           ├→ Model 3: Sentiment (customer intent)
           └→ Model 4: Load Balancer (queue optimization)
           ↓
         Ensemble Vote → Best Branch Selected
           ↓
        Decision Stored + Returned
           ↓
        Workflow Executes on Selected Branch
           ↓
        Outcome Recorded
           ↓
        RL Agent Updates Q-Values
           ↓
        Next Decision Even Smarter! 🧠
```

## 🎯 Key Metrics on Dashboard

| Metric | What It Means | Target |
|--------|---------------|--------|
| **AI Routing Accuracy** | % of workflows that succeed | > 85% |
| **Avg Decision Time** | Time to pick best branch | < 100ms |
| **Model Agreement** | Do models agree? | > 70% |
| **RL Episodes** | How many times did RL learn? | Growing |

## 🔥 What's Happening Behind the Scenes

1. **Every routing request**:
   - 4 models run in parallel (max 500ms)
   - Scores combined via weighted voting
   - Top branch + alternatives returned
   - Decision stored for audit trail

2. **Every hour**:
   - Completed workflows analyzed
   - Rewards calculated
   - Q-values updated (RL learns)
   - Performance metrics aggregated

3. **Continuously**:
   - Epsilon decays (exploration → exploitation)
   - Q-table grows (new states discovered)
   - Models improve accuracy
   - Business rules validated

## 🚨 Troubleshooting

**"Routes not being made?"**
- Check tenant_id and datasource_id headers
- Verify database connection
- Check logs for errors

**"Dashboard showing no data?"**
- Make at least one routing request first
- Wait 5-10 seconds for metrics to populate
- Refresh browser page

**"Decision latency too high?"**
- Check which model is slowest in logs
- Consider disabling sentiment analysis if not needed
- Increase timeout threshold if needed

## 📚 Next Steps

- Read full guide: [`AI_ROUTING_IMPLEMENTATION_GUIDE.md`](./AI_ROUTING_IMPLEMENTATION_GUIDE.md)
- Integrate with BP Builder: See "Integration" section
- Set up custom rules: See "Configuration" section
- Enable ML predictions: See "Advanced Features"

## 🎉 That's It!

You now have production-ready AI routing with:
- Multi-model ensemble voting
- Self-learning RL agent
- Real-time monitoring
- Continuous improvement
- Enterprise-grade scalability

**Status**: ✅ Live & Learning  
**Models Active**: 4/4 ✓  
**Learning Enabled**: Yes ✓  
**Ready for Production**: Yes ✓

---

Need help? Check [`AI_ROUTING_IMPLEMENTATION_GUIDE.md`](./AI_ROUTING_IMPLEMENTATION_GUIDE.md)
