# Workspace Configuration

## pnpm Monorepo Structure

This repository uses a **hybrid monorepo** with both **Node.js and Go packages**. The `pnpm-workspace.yaml` configuration has been updated to properly separate these:

### Node.js Packages (Managed by pnpm)

**Frontend:**
- `frontend` - React + Vite application

**Services:**
- `services/fabric-builder` - Semantic model and business process designer
- `services/wealth-management` - Wealth management service

**Libraries:**
- `libs/shared-types` - Shared TypeScript types
- `libs/hasura-client` - Hasura GraphQL client
- `libs/ai-sdk` - AI SDK utilities

### Go Packages (Not in pnpm workspace)

These are Go modules managed separately at the root and service levels:

**Libraries:**
- `libs/temporal-client/` - Temporal workflow client (Go)
- `libs/abac-client/` - ABAC authorization client (Go)

**Services:**
- `services/ai-builder/` - AI model builder (Go)
- `services/compliance-engine/` - Compliance engine (Go)
- `services/event-router/` - Event routing service (Go)
- `services/governance/` - Governance service (Go)
- `services/semantic-engine/` - Semantic engine (Go)
- Additional microservices (various Go services)

## Why They're Separated

Go packages use `go.mod` for dependency management, while Node.js packages use `package.json` and `pnpm`. Mixing them in the same workspace would cause:

1. ❌ `ERR_PNPM_WORKSPACE_PKG_NOT_FOUND` errors
2. ❌ Incompatible dependency resolution
3. ❌ Build system conflicts

## Installation & Development

### Install Node.js Dependencies
```bash
pnpm install
```

### Build Go Services
```bash
# Backend (from root or service directory)
cd backend && go build ./...

# Specific service
cd services/semantic-engine && go build
```

### Start Development

**Frontend:**
```bash
cd frontend && npm run dev
```

**Backend (Go):**
```bash
cd backend && PORT=8081 go run ./cmd/server/main.go
```

## Adding New Packages

- **Node.js packages**: Add to `pnpm-workspace.yaml` and create `package.json`
- **Go packages**: Create with `go.mod`, no need to add to `pnpm-workspace.yaml`

## References

- [pnpm Workspaces](https://pnpm.io/workspaces)
- [Go Modules](https://go.dev/blog/using-go-modules)
