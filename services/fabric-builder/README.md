# Fabric Builder Service

The Fabric Builder Service provides semantic model management and business process design capabilities for the SemLayer platform.

## Overview

This service handles:
- **Semantic Models**: Creation and management of data models with validation and extensions
- **Business Processes**: Visual workflow design with step types, validation rules, and execution
- **Extensions**: Custom functionality that extends semantic models

## Architecture

### Backend (Go)
- **API Layer**: RESTful endpoints for CRUD operations
- **Service Layer**: Business logic for models, processes, and extensions
- **Data Layer**: Database interactions (to be implemented)

### Frontend (TypeScript/React)
- **FabricBuilder**: Main component for semantic model management
- **BusinessProcessDesigner**: Visual workflow designer
- **SemanticModelEditor**: Form-based model editor

## API Endpoints

### Fabric Models
- `GET /api/fabric/models` - List models
- `POST /api/fabric/models` - Create model
- `GET /api/fabric/models/{id}` - Get model
- `PUT /api/fabric/models/{id}` - Update model
- `DELETE /api/fabric/models/{id}` - Delete model

### Extensions
- `GET /api/fabric/extensions` - List extensions
- `POST /api/fabric/extensions` - Create extension
- `GET /api/fabric/extensions/{id}` - Get extension
- `PUT /api/fabric/extensions/{id}` - Update extension
- `DELETE /api/fabric/extensions/{id}` - Delete extension

### Business Processes
- `GET /api/business-process/` - List processes
- `POST /api/business-process/` - Create process
- `GET /api/business-process/{id}` - Get process
- `PUT /api/business-process/{id}` - Update process
- `DELETE /api/business-process/{id}` - Delete process
- `POST /api/business-process/{id}/execute` - Execute process

## Development

### Running the Service
```bash
# Backend
cd services/fabric-builder
go run main.go

# Frontend (from monorepo root)
npm run dev
```

### Building
```bash
# Backend
go build -o fabric-builder-service .

# Frontend
npm run build
```

## TODO

- [ ] Implement database layer for persistence
- [ ] Add Temporal workflow integration for process execution
- [ ] Implement extension compatibility checking
- [ ] Add comprehensive validation for models and processes
- [ ] Create visual process designer canvas
- [ ] Add testing framework
- [ ] Implement authentication and authorization