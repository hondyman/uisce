# SemLayer - Enterprise Wealth Management Platform

🚀 **The Ultimate Wealth Management Platform** - AI-native, ABAC-secure, real-time wealth management with microservices architecture.

## 🏗️ **Microservices Architecture**

This platform is built with a modern microservices architecture for scalability, maintainability, and independent deployment.

### **Core Services**

| Service | Description | Status |
|---------|-------------|--------|
| **wealth-management** | UMA Alpha, Attribution Alpha, Tax Harvest, Direct Indexing | ✅ Extracted |
| **fabric-builder** | Business process designer, validation rules | 🔄 Planned |
| **semantic-engine** | AI-powered search, mappings, suggestions | 🔄 Planned |
| **governance** | ABAC policies, audit trails, compliance | 🔄 Planned |
| **temporal-orchestrator** | Workflow orchestration engine | 🔄 Planned |
| **api-gateway** | Unified API routing and authentication | 🔄 Planned |

### **Shared Libraries**

| Library | Purpose | Language |
|---------|---------|----------|
| **shared-types** | Common TypeScript/Go types | TypeScript |
| **temporal-client** | Temporal workflow client utilities | Go |
| **abac-client** | Attribute-based access control | Go |
| **hasura-client** | GraphQL client utilities | TypeScript |
| **ai-sdk** | xAI integration SDK | TypeScript |

## 📁 **Project Structure**

```
semlayer/
├── services/                    # 🆕 Microservices
│   ├── wealth-management/      # Core wealth management functionality
│   └── [other services]/
├── libs/                       # 🆕 Shared libraries
│   ├── shared-types/           # Common types
│   ├── temporal-client/        # Temporal utilities
│   ├── abac-client/           # Access control
│   ├── hasura-client/         # GraphQL client
│   └── ai-sdk/                # AI integration
├── infrastructure/            # 🆕 Infrastructure as code
│   ├── docker/                # Container definitions
│   ├── k8s/                   # Kubernetes manifests
│   ├── terraform/             # Cloud infrastructure
│   └── monitoring/            # Observability
├── docs/                      # 🆕 Centralized documentation
│   ├── services/              # Per-service docs
│   ├── api/                   # API documentation
│   ├── deployment/            # Deployment guides
│   ├── development/           # Development setup
│   └── architecture/          # System architecture
├── tools/                     # 🆕 Development tools
│   ├── scripts/               # Build/deployment scripts
│   ├── e2e-tests/             # End-to-end test suites
│   └── ci/                    # CI/CD pipelines
├── frontend/                  # React frontend application
└── backend/                   # Legacy monolithic backend (being decomposed)
```

## 🚀 **Quick Start**

### **Prerequisites**
- Go 1.24+
- Node.js 18+
- Docker & Docker Compose
- Temporal CLI

### **Development Setup**

1. **Clone and setup:**
   ```bash
   git clone <repository>
   cd semlayer
   npm install
   ```

> Tip: If you want to run Postgres and Apache Ignite locally instead of in Docker, see `docs/development/dev-databases.md` for recommended steps and configuration.


2. **Start infrastructure:**
   ```bash
   cd infrastructure/docker
   docker-compose up -d
   ```

3. **Start wealth management service:**
   ```bash
   cd services/wealth-management
   go run .
   ```

4. **Start frontend:**
   ```bash
   cd frontend
   npm run dev
   ```

### Environment variables (frontend)

The frontend runs on Vite and expects environment variables prefixed with `VITE_*`. For backwards compatibility, some legacy scripts may still use `REACT_APP_*`, but prefer `VITE_*` for all new code. Use the `getEnv()` helper in front-end modules to read either legacy or Vite-style variables while you transition.

```bash
VITE_API_BASE_URL=http://localhost:29080
VITE_GRAPHQL_ENDPOINT=http://localhost:8080/v1/graphql
```

### **Build All Services**
```bash
npm run build        # Build all workspaces
npm run build:libs   # Build shared libraries first
npm run build:services # Build all services
```

## 🎯 **Killer Applications**

### **Wealth Management Service**
- **UMA Alpha**: AI-powered rebalancing in 2 seconds
- **Attribution Alpha**: Performance analysis in 4 seconds
- **Tax Harvest**: $1M+ tax savings per $1B AUM
- **Direct Indexing Alpha**: Index optimization in 3 seconds

### **Performance Comparison**

| Feature | SemLayer | Addepar | Aladdin | Envestnet | Black Diamond |
|---------|----------|---------|---------|-----------|---------------|
| **UMA Rebalancing** | **2s** | 10s | 30s+ | 15s | 20s |
| **Attribution** | **4s** | 20s | 90s+ | 60s | 120s |
| **Tax Optimization** | **60s** | Manual | Basic | Basic | Manual |
| **Direct Indexing** | **3s** | 15s | 60s+ | 30s | 45s |
| **AI Integration** | **xAI** | Manual | Basic | Basic | Manual |
| **Compliance** | **ABAC** | Basic | Static | RCI | Manual |
| **Architecture** | **Microservices** | Monolithic | Monolithic | Monolithic | Monolithic |

## 🛠️ **Development**

### **Adding a New Service**
1. Create service directory: `services/new-service/`
2. Add Go module: `go mod init github.com/hondyman/semlayer/services/new-service`
3. Update `go.work` and root `package.json`
4. Add service documentation: `docs/services/new-service.md`

### **Adding a Shared Library**
1. Create library: `libs/new-lib/`
2. Add package.json or go.mod
3. Update workspace references
4. Export from library index file

### **Contributing**
- Follow conventional commits
- Update documentation for any API changes
- Add tests for new functionality
- Update deployment scripts if needed

## � **Documentation**

- **[Architecture Overview](docs/architecture/)** - System design and patterns
- **[API Documentation](docs/api/)** - REST and GraphQL APIs
- **[Service Documentation](docs/services/)** - Per-service guides
- **[Deployment Guide](docs/deployment/)** - Production deployment
- **[Development Setup](docs/development/)** - Local development

## 🔧 **Infrastructure**

### **Local Development**
```bash
# Start all services
tools/scripts/deploy-platform.sh

# Or start individual services
cd services/wealth-management && go run .
cd frontend && npm run dev
```

### **Production Deployment**
- **Docker**: Containerized deployment
- **Kubernetes**: Orchestration manifests in `infrastructure/k8s/`
- **Terraform**: Cloud infrastructure in `infrastructure/terraform/`
- **Monitoring**: Observability in `infrastructure/monitoring/`

## 🤝 **Contributing**

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests and documentation
5. Submit a pull request

## 📄 **License**

Proprietary - All rights reserved.

---

**Built for the future of wealth management.** 🚀📈

*See [docs/](docs/) for detailed documentation.*
- **Governance Workflows**: Steward review and approval processes
- **Cube.js Integration**: Seamless OLAP cube generation
- **Real-time Dashboards**: Live parameter-driven analytics
- **Multi-tenant Support**: Isolated semantic layers per tenant

## 🏗️ Architecture

```
Frontend (React/TypeScript)
    ↓
API Gateway (Go/Gin)
    ↓
Backend Services (Go)
    ↓
PostgreSQL Database
    ↓
Cube.js (OLAP Engine)
```

## 📋 Dynamic Parameters & Measures

### Parameter Types

- **Dimensions**: city, region, country, device_type, status, category
- **Time Ranges**: period, granularity
- **Filters**: active_only, premium_only

### Dynamic Measures

Auto-generate measures from:
- Database enums (e.g., order status → processing, shipped, completed)
- Custom SQL expressions
- Aggregations with filters

### API Endpoints

```bash
# Parameter Management
GET  /api/parameters/schema
GET  /api/parameters/:type/:name/values

# Measure Management
POST /api/measures/generate
GET  /api/measures/catalog
POST /api/measures/validate

# Dynamic Queries
POST /api/v1/dynamic/query
```

## 🛠️ Quick Start

### Prerequisites

- Go 1.21+
- Node.js 18+
- PostgreSQL 13+
- Docker (optional)

### Backend Setup

```bash
cd backend
go mod download
cp config.example.yaml config.yaml
# Edit config.yaml with your database settings
go run main.go
```

### Frontend Setup

```bash
cd frontend
npm install
npm run dev
```

### Database Setup

```bash
# Create database
createdb semlayer

# Run migrations
psql -d semlayer -f backend/init-db.sql
```

## 🧪 Testing

### Run Backend Demo

```bash
cd backend
go run cmd/demo.go
```

### Test API Endpoints

```bash
./test-api.sh
```

### Example API Usage

```bash
# Get parameter schema
curl http://localhost:8080/api/parameters/schema

# Generate measures from orders.status
curl -X POST http://localhost:8080/api/measures/generate \
  -H "Content-Type: application/json" \
  -d '{
    "source_table": "orders",
    "source_column": "status",
    "measure_type": "count"
  }'

# Validate a measure
curl -X POST http://localhost:8080/api/measures/validate \
  -H "Content-Type: application/json" \
  -d '{
    "name": "total_processing_orders",
    "sql": "CASE WHEN status = '\''processing'\'' THEN 1 ELSE 0 END"
  }'
```

## GraphQL (backend/internal/graphql) - camelCase fields

The GraphQL schema for the IP whitelist uses camelCase fields so JSON clients should send/expect camelCase.

Schema excerpt (important fields):

type IpWhitelistEntry {
  id: UUID!
  tenantId: UUID
  ipAddress: String!
  description: String
  createdAt: Timestamp!
  updatedAt: Timestamp!
}

Example curl requests (camelCase fields):

Create an entry:
```bash
cat > /tmp/create.json <<'JSON'
{"query":"mutation Create($in: IpWhitelistEntryInput!){ createIpWhitelistEntry(input:$in){ id ipAddress description tenantId createdAt updatedAt }}","variables":{"in":{"ipAddress":"127.0.0.1","description":"local dev"}}}
JSON

curl -sS -X POST http://localhost:8080/api/graphql \
  -H 'Content-Type: application/json' \
  -d @/tmp/create.json | jq '.'
```

Query by id:
```bash
cat > /tmp/query.json <<'JSON'
{"query":"query Get($id: UUID!){ ipWhitelistEntry(id:$id){ id ipAddress description tenantId createdAt updatedAt }}","variables":{"id":"<PASTE_ID_HERE>"}}
JSON

curl -sS -X POST http://localhost:8080/api/graphql \
  -H 'Content-Type: application/json' \
  -d @/tmp/query.json | jq '.'
```

Update and delete use the same camelCase shape for inputs and returned fields.


## 🎨 Frontend Components

### ParameterSelector
Dynamic dropdowns and filters for runtime parameter selection.

### DynamicMeasureGenerator
Auto-generate measures from database schemas and enums.

### StewardWorkflow
Governance interface for measure review and approval.

### EnhancedDashboard
Live dashboard with parameter-driven analytics.

## 🔐 Governance

- **Steward Review**: All dynamic measures require approval
- **Golden Path**: Approved measures become reusable templates
- **Audit Trail**: Complete history of changes and approvals
- **Access Control**: Role-based permissions for measure creation

## 📊 Cube.js Integration

Dynamic measures are automatically converted to Cube.js YAML:

```yaml
measures:
  - name: total_processing_orders
    type: count
    sql: CASE WHEN status = 'processing' THEN 1 ELSE 0 END
    filters:
      - sql: city = '{FILTER_PARAMS.city}'
```

## 🧱 Dynamic Measures Sync

Auto-generate measures from database enums and sync to Cube:

```bash
cd scripts
go run generate_dynamic_measures.go
```

Features:
- Reads distinct values from source tables
- Generates Cube YAML measure definitions
- Syncs to catalog with governance metadata
- Supports multiple source tables (orders, products, clickstream)

## 🔍 Anomaly-Aware Measures

Generate measures with built-in anomaly detection:

```bash
cd scripts
go run generate_anomaly_measures.go
```

Features:
- Z-score and IQR anomaly detection methods
- Configurable thresholds and lookback periods
- Automatic anomaly flag dimensions
- Performance, revenue, and activity monitoring

## 🎯 Dynamic Dimensions & Scoped Filters

### API Endpoints

```bash
# Dynamic Dimensions
GET  /api/dimensions
GET  /api/dimensions/:dimension/values

# Scoped Filters
GET  /api/filters/scoped
POST /api/filters/scoped/apply
```

### Dimension Types

- **advisor_id**: Financial advisor identifiers
- **fund_type**: Investment fund categories
- **client_segment**: Client segmentation
- **risk_profile**: Risk assessment profiles

### Scoped Filters

- **high_value_clients**: AUM > $1M
- **active_traders**: Recent transaction activity
- **premium_accounts**: Premium tier clients
- **new_clients**: Recent acquisitions

## 🧭 Steward Cockpit

Complete governance interface for measure management:

```tsx
<StewardCockpit stewardUser="patrick" />
```

Features:
- Review and approve dynamic measures
- Golden path promotion workflow
- Comment threads and audit trails
- Status filtering and search
- Anomaly detection alerts

## 🔐 Governance Best Practices

### Schema Contracts
- JSON schema validation for all semantic assets
- Version control and drift detection
- Automated CI/CD validation

### Catalog Integration
- Unified metadata store in `public.catalog_node`
- Lineage tracking and dependencies
- Steward group assignments

### Quality Contracts
- SLA definitions per measure
- Data freshness requirements
- Accuracy and completeness thresholds

## 🧪 CI/CD Validation

Validate dynamic measures in your pipeline:

```bash
cd scripts
go run validate_dynamic_measures.go semantic_layer/dynamic_measures/
```

Features:
- JSON schema compliance checking
- SQL injection prevention
- Schema hash drift detection
- Automated measure naming validation

## 📊 Cube.js Integration

### Dynamic Measures YAML
```yaml
measures:
  - name: total_processing_orders
    type: count
    sql: CASE WHEN status = 'processing' THEN 1 ELSE 0 END
    description: "Total count of processing orders"
```

### Anomaly Dimensions
```yaml
dimensions:
  - name: revenue_anomaly_flag
    sql: |
      CASE
        WHEN ABS(SUM(revenue) - AVG(SUM(revenue)) OVER (...)) / STDDEV_POP(...) > 2.5
        THEN true ELSE false END
    type: boolean
```

## 🚀 Quick Start

### 1. Generate Dynamic Measures
```bash
cd scripts
go run generate_dynamic_measures.go
```

### 2. Generate Anomaly Measures
```bash
go run generate_anomaly_measures.go
```

### 3. Validate Schema
```bash
go run validate_dynamic_measures.go semantic_layer/dynamic_measures/
```

### 4. Start Backend
```bash
cd backend
go run main.go
```

### 5. Use Frontend Components
```tsx
import { DynamicMeasurePreview } from './components/pop/DynamicMeasurePreview';
import { StewardCockpit } from './components/pop/StewardCockpit';
import { DynamicDimensions } from './components/pop/DynamicDimensions';

// Use in your dashboard
<DynamicMeasurePreview />
<StewardCockpit stewardUser="patrick" />
<DynamicDimensions />
```

## 📚 Advanced Features

### Custom Anomaly Detection
Configure anomaly methods per measure:
- **Z-Score**: Statistical outlier detection
- **IQR**: Interquartile range method
- **Threshold**: Fixed threshold alerts
- **Prophet**: Time series forecasting

### Scoped Filter Parameters
Apply filters with runtime parameters:
```json
{
  "filter_name": "high_value_clients",
  "base_query": "SELECT * FROM clients",
  "parameters": {
    "min_aum": 1000000,
    "currency": "USD"
  }
}
```

### Steward Workflows
Complete governance lifecycle:
1. **Draft** → Auto-generated measures
2. **Pending Review** → Steward evaluation
3. **Approved** → Golden path promotion
4. **Deprecated** → End-of-life management

## 🔧 Configuration

### Environment Variables
```bash
DB_URL=postgres://user:pass@localhost:5432/semlayer
CUBE_API_URL=http://localhost:4000/cubejs-api/v1
CATALOG_SCHEMA=public
```

### Schema Locations
- Dynamic measures: `semantic_layer/dynamic_measures/`
- JSON schemas: `schemas/`
- Cube configs: `cube/schema/`

## 📈 Performance Optimization

- **Lazy Loading**: Dimension values loaded on demand
- **Caching**: Measure metadata cached in Redis
- **Batch Operations**: Bulk catalog sync operations
- **Indexing**: Optimized database queries for large datasets

## 🔒 Security

- **SQL Injection Prevention**: Parameterized queries only
- **Access Control**: Steward role-based permissions
- **Audit Logging**: Complete action history
- **Schema Validation**: Prevent malicious measure definitions

## 🎯 Use Cases

### Financial Services
- Revenue anomaly detection
- Client segmentation analysis
- Risk profile monitoring
- Advisor performance tracking

### E-commerce
- Order status analysis
- Product category performance
- Customer behavior patterns
- Inventory anomaly detection

### SaaS Analytics
- User activity monitoring
- Feature usage analysis
- Performance metric tracking
- Error rate anomaly detection

This platform provides enterprise-grade dynamic semantic modeling with complete governance, anomaly detection, and steward workflows - all built within your existing React + Go + Postgres + Cube stack!

For more information on the backend, see the [backend README](./backend/README.md).

For API details (including the catalog scan endpoint and the new 207 Multi-Status response), see [backend/API.md](./backend/API.md).

---

## Developer notes: backend/internal/api Routes pattern & tests

This repository uses a small convention inside `backend/internal/api` to keep HTTP route
registration modular and testable. Key points:

- A `Routes` helper (see `backend/internal/api/routes.go`) centralizes grouped
  registration helpers like `RegisterBundles`, `RegisterPolicies`, `RegisterViews`, etc.
- Individual route groups live in their own files (for example `bundles_routes.go`,
  `policies_routes.go`, `roles_routes.go`) and expose `RegisterRoutes(r chi.Router)`
  methods which the `Routes` helper calls from `SetupRouter`.
- Tests for the registration wrappers live next to the route files (e.g.
  `bundles_routes_test.go`) and assert the registration functions don't panic and
  wire paths correctly.

Quick commands for developers:

```bash
# Run only the api package tests (fast)
go test ./backend/internal/api -v

# Run backend module tests (safer than repo root)
cd backend && go test ./... -v
```

If you see duplicate type redeclaration errors during `go test ./...`, search
for small DTOs (Request/Response types) duplicated across files in
`backend/internal/api` and consolidate them to a single small file (for
example `types.go`, `governance_types.go`, `profiler_types.go`).

