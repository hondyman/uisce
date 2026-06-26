# Entity Manager System

A production-grade entity management system with strong typing, polymorphic validation, integrated workflows, and comprehensive REST API.

## 🚀 Features

- **Strong TypeScript Typing**: Full type safety with strict mode compilation
- **Polymorphic Entity System**: Abstract base classes with concrete implementations (Personal, IRA, Trust accounts)
- **Integrated Workflows**: Entity → Validation → Approval → Execution pipeline
- **REST API**: Complete CRUD operations with proper error handling
- **Compliance Engine**: Configurable rules and validation
- **Approval Workflows**: Temporal-based workflow orchestration
- **Database Integration**: PostgreSQL with Hasura GraphQL support
- **Caching**: Redis for performance optimization
- **Event-Driven**: RabbitMQ for workflow notifications
- **Security**: Helmet, CORS, input validation
- **Monitoring**: Winston structured logging

## 📁 Project Structure

```
entity-manager/
├── src/
│   ├── entities/           # Core business entities
│   │   ├── Entity.ts       # Base entity class
│   │   ├── Client.ts       # Client with KYC validation
│   │   ├── Account.ts      # Abstract account class
│   │   ├── PersonalAccount.ts
│   │   ├── IRAAccount.ts
│   │   └── TrustAccount.ts
│   ├── services/           # Business logic services
│   │   ├── EntityManager.ts    # CRUD operations & caching
│   │   ├── ValidationEngine.ts # Rules engine
│   │   ├── ApprovalWorkflowEngine.ts # Temporal workflows
│   │   ├── UnifiedValidator.ts # Orchestrator
│   │   ├── database.ts     # PostgreSQL connection
│   │   ├── redis.ts        # Redis caching
│   │   ├── temporal.ts     # Workflow orchestration
│   │   └── rabbitmq.ts     # Event messaging
│   ├── api/                # REST API routes
│   │   ├── routes.ts       # Route configuration
│   │   ├── accounts.ts     # Account CRUD endpoints
│   │   ├── trades.ts       # Trade validation/execution
│   │   ├── approvals.ts    # Approval workflow management
│   │   ├── compliance.ts   # Compliance checking
│   │   └── demo.ts         # Demo data & testing
│   ├── utils/              # Utilities
│   │   ├── logger.ts       # Winston logging
│   │   └── types.ts        # Shared type definitions
│   └── server.ts           # Express server setup
├── dist/                   # Compiled JavaScript
├── package.json
├── tsconfig.json
└── README.md
```

## 🛠️ Installation

```bash
# Install dependencies
npm install

# Build the project
npm run build

# Start development server
npm run dev

# Or start production server
npm start
```

## 📡 API Endpoints

### Accounts
- `POST /api/accounts/personal` - Create personal account
- `POST /api/accounts/ira` - Create IRA account
- `POST /api/accounts/trust` - Create trust account
- `GET /api/accounts/:id` - Get account details
- `PUT /api/accounts/:id` - Update account
- `DELETE /api/accounts/:id` - Delete account

### Trades
- `POST /api/trades/validate` - Validate trade request
- `POST /api/trades/execute` - Execute validated trade
- `GET /api/trades/history/:accountId` - Get trade history

### Approvals
- `POST /api/approvals/start` - Start approval workflow
- `GET /api/approvals/status/:workflowId` - Get workflow status
- `POST /api/approvals/approve/:workflowId` - Approve workflow
- `POST /api/approvals/reject/:workflowId` - Reject workflow

### Compliance
- `GET /api/compliance/validate-all` - Validate all accounts
- `GET /api/compliance/account/:accountId` - Get account compliance

### Demo
- `POST /api/demo/create-sample-accounts` - Create sample accounts
- `POST /api/demo/validate-trade` - Demo trade validation
- `GET /api/demo/accounts` - List demo accounts

## 🧪 Demo Usage

```bash
# Create sample accounts
curl -X POST http://localhost:4000/api/demo/create-sample-accounts

# Validate a trade
curl -X POST http://localhost:4000/api/demo/validate-trade \
  -H "Content-Type: application/json" \
  -d '{
    "accountId": "demo-personal-1",
    "symbol": "AAPL",
    "quantity": 100,
    "price": 150.00
  }'

# Check compliance
curl http://localhost:4000/api/compliance/validate-all
```

## 🔧 Configuration

Create a `.env` file:

```env
# Server
PORT=4000
NODE_ENV=development
CORS_ORIGIN=http://localhost:3000

# Database
DATABASE_URL=postgresql://user:password@localhost:5432/entity_manager

# Redis
REDIS_URL=redis://localhost:6379

# Temporal
TEMPORAL_ADDRESS=localhost:7233
TEMPORAL_NAMESPACE=default

# RabbitMQ
RABBITMQ_URL=amqp://localhost:5672

# Hasura
HASURA_ENDPOINT=http://localhost:8080/v1/graphql
HASURA_ADMIN_SECRET=your-secret
```

## 🏗️ Architecture

### Entity Hierarchy
```
Entity (abstract)
├── Client
└── Account (abstract)
    ├── PersonalAccount
    ├── IRAAccount
    └── TrustAccount
```

### Workflow Pipeline
1. **Entity Creation**: Validate and persist entity
2. **Trade Request**: Check account rules and compliance
3. **Validation**: Apply business rules and concentration limits
4. **Approval**: Route through configurable approval chains
5. **Execution**: Process trade and update positions

### Service Layer
- **EntityManager**: CRUD operations with Redis caching
- **ValidationEngine**: Configurable rules engine
- **ApprovalWorkflowEngine**: Temporal workflow orchestration
- **UnifiedValidator**: Main orchestrator service

## 🧪 Testing

```bash
# Run tests
npm test

# Run linting
npm run lint
```

## 📊 Monitoring

- Health check: `GET /health`
- Structured logging with Winston
- Error tracking and metrics

## 🚀 Deployment

```bash
# Build for production
npm run build

# Start production server
npm start
```

## 🤝 Contributing

1. Follow TypeScript strict mode
2. Add tests for new features
3. Update documentation
4. Use conventional commits

## 📄 License

MIT License