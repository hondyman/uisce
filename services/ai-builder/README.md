# AI Workflow Builder Service

A microservice that provides AI-powered workflow generation capabilities for the Temporal + Redpanda (Kafka) platform.

## Features

- **AI Workflow Suggestions**: Generate workflow suggestions using xAI API
- **Temporal Integration**: Execute and monitor AI-generated workflows
- **ABAC Security**: Attribute-based access control for workflow operations
- **REST API**: Clean REST endpoints for workflow management

## API Endpoints

### POST /workflows/suggest
Generate AI-powered workflow suggestions.

**Request Body:**
```json
{
  "description": "Rebalance portfolio for high-net-worth client",
  "context": {
    "industry": "wealth-management",
    "compliance": "FINRA"
  }
}
```

**Response:**
```json
{
  "suggestion": {
    "description": "Rebalance portfolio for high-net-worth client",
    "elements": [...],
    "yaml": "...",
    "metadata": {...}
  },
  "workflow_id": "ai-workflow-suggestion-1234567890"
}
```

### GET /workflows/{id}/status
Get the status of a workflow suggestion processing.

### GET /health
Health check endpoint.

## Running the Service

### As API Server (default)
```bash
./ai-builder
```

### As Temporal Worker
```bash
./ai-builder worker
```

## Temporal Workflows

### ProcessWorkflowSuggestion
Main workflow that processes AI-generated suggestions:
1. Validates the workflow structure
2. Stores the suggestion
3. Notifies stakeholders

## Dependencies

- Gin (HTTP framework)
- Temporal SDK (workflow orchestration)
- xAI API (AI suggestions)

## Configuration

Set the `XAI_API_KEY` environment variable for xAI API access.

## Integration

This service integrates with:
- **Temporal**: For workflow execution and monitoring
- **RabbitMQ**: For event-driven processing (future)
- **ABAC Engine**: For security and access control
- **Frontend**: Via REST API for workflow suggestions</content>
<parameter name="filePath">/Users/eganpj/GitHub/semlayer/services/ai-builder/README.md