# Schema-Driven Upgrade Artifacts

This project uses a unified JSON Schema to define the contract for all upgrade artifacts, ensuring zero drift between backend (Go) and frontend (TypeScript) implementations.

## Architecture

- **Single Source of Truth**: `schemas/upgrade-artifacts.schema.json`
- **Generated Types**:
  - Go: `backend/internal/types/upgrade.go` (generated)
  - TypeScript: `frontend/src/types/upgrade-generated.ts` (generated)
- **Manual Types**: `frontend/src/types/upgrade.ts` (imports from generated + manual extensions)

## Key Benefits

1. **Zero Drift**: Backend and frontend always use identical type definitions
2. **Versioned Contracts**: Every artifact includes `schema_version` and `changelog`
3. **CI/CD Enforcement**: Automated type generation and validation
4. **Evolution Tracking**: Schema changes are tracked with version history

## Schema Structure

```json
{
  "schema_version": "1.0.0",
  "changelog": [...],
  "report": {...},
  "aliases": {...}
}
```

## Development Workflow

1. **Modify Schema**: Edit `schemas/upgrade-artifacts.schema.json`
2. **Regenerate Types**: Run `scripts/enforce-schema-contract.sh`
3. **Update Code**: Use the regenerated types in your code
4. **Bump Version**: Update `schema_version` for breaking changes

## Scripts

- `scripts/generate-go-types.sh`: Generate Go types from schema
- `scripts/generate-ts-types.sh`: Generate TypeScript types from schema
- `scripts/enforce-schema-contract.sh`: CI/CD validation and regeneration

## Type Usage

### Backend (Go)
```go
import "github.com/eganpj/semlayer/backend/internal/types"

// Use generated types directly
var artifact types.UpgradeArtifacts
```

### Frontend (TypeScript)
```typescript
import { DiffReport, AliasMap } from './types/upgrade';

// Use imported types
const report: DiffReport = {...};
```
