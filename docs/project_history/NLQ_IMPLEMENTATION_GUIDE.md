# Natural Language Q&A System - Complete Implementation Guide

## Overview

This implementation provides a tenant-scoped, AI-powered Natural Language Q&A system for querying your data catalog. The system combines:

1. **Graph-based catalog navigation** via PostgreSQL recursive CTEs
2. **Semantic search** via pgvector embeddings
3. **Model-agnostic LLM integration** (Gemini by default, switchable via admin UI)
4. **Calculation DAG tracing** with lineage, data quality, and SLA metadata
5. **Beautiful, interactive frontend** with chat-style interface

## Architecture

### Backend Components

#### 1. LLM Provider (`backend/internal/llm/provider.go`)
- **Interface**: `LLMProvider` with methods `GenerateResponse()` and `Embed()`
- **Implementation**: `GeminiProvider` for Google Gemini (text-embedding-004 for embeddings)
- **Extensibility**: Easy to add OpenAI, Claude, or custom providers

#### 2. NLQ Service (`backend/internal/services/nlq_service.go`)
- **Hybrid Retrieval**: Combines semantic search (pgvector) with graph traversal
- **Auto-discovery**: If no target entity specified, finds most relevant node via embedding similarity
- **Structured Responses**: Returns answer + sources + calculation breakdown + caveats + confidence

#### 3. Catalog Embedding Service (`backend/internal/services/catalog_embedding_service.go`)
- **Background Job**: Generates embeddings for all catalog nodes
- **Incremental Updates**: Can regenerate embeddings for individual nodes
- **Smart Text Representation**: Combines node type, name, path, description, and properties

#### 4. API Handler (`backend/internal/api/api.go`)
- **Endpoint**: `POST /api/nlq/ask`
- **Tenant-Scoped**: Requires `X-Tenant-ID` and `X-Tenant-Datasource-ID` headers
- **Request**: `{ "question": "...", "target_entity_path": "..." }` (path optional)
- **Response**: Structured answer with sources, caveats, and metadata

### Database Layer

#### 1. SQL Migration (`backend/internal/migrations/nlq_support.sql`)
- **pgvector Extension**: Enables semantic search
- **Embedding Column**: Adds `vector(768)` column to `catalog_node`
- **Indexes**: IVFFlat index for fast cosine similarity search
- **Functions**:
  - `get_calc_dag(start_path, tenant)` - Basic DAG builder
  - `get_calc_dag_with_metadata(start_path, tenant)` - Enhanced DAG with lineage/DQ/SLA
  - `resolve_node(ref, tenant)` - Alias resolution

#### 2. Recursive CTEs for Graph Traversal
```sql
WITH RECURSIVE dag AS (
    SELECT ... FROM catalog_node WHERE qualified_path = start_path
    UNION ALL
    SELECT ... FROM dag JOIN catalog_edge ON ... JOIN catalog_node ON ...
)
```

### Frontend Components

#### 1. NLQ Page (`frontend/src/pages/nlq/NLQPage.tsx`)
- **Chat Interface**: Real-time Q&A with message history
- **Auto-scroll**: Smooth scrolling to latest messages
- **Loading States**: Animated dots while processing
- **Rich Responses**: Displays sources, caveats, calculation breakdowns
- **Example Questions**: Quick-start buttons for common queries

#### 2. LLM Config Page (`frontend/src/pages/admin/LLMConfigPage.tsx`)
- **Provider Selection**: Switch between Gemini, OpenAI, Claude
- **Model Selection**: Choose specific model version
- **Parameter Tuning**: Temperature, max tokens
- **API Key Management**: Secure credential storage
- **Test Function**: Verify configuration before saving

## Setup Instructions

### 1. Database Setup

Run the migration:

```bash
psql postgres://postgres:postgres@host.docker.internal:5432/alpha?sslmode=disable < backend/internal/migrations/nlq_support.sql
```

This will:
- Enable `pgvector` extension
- Add `embedding` column to `catalog_node`
- Create similarity search index
- Add calculation DAG functions

### 2. Environment Configuration

Set your Gemini API key (or other provider):

```bash
export GEMINI_API_KEY="your-api-key-here"
```

Alternatively, configure via the admin UI after startup.

### 3. Generate Embeddings

Use the embedding service to generate vectors for existing catalog nodes:

```go
import "github.com/hondyman/semlayer/backend/internal/services"

embeddingService := services.NewCatalogEmbeddingService(db, llmProvider)
err := embeddingService.GenerateEmbeddingsForTenant(ctx, tenantID, datasourceID)
```

Or create a CLI tool:

```bash
go run cmd/generate-embeddings/main.go --tenant=<TENANT_ID> --datasource=<DATASOURCE_ID>
```

### 4. Start the Backend

The NLQ service is automatically initialized in `api.go`:

```go
llmProvider := llm.NewGeminiProvider("", "") // Reads from env
srv.NLQService = services.NewNLQService(sqlxDB, llmProvider)
```

The endpoint is registered at `/api/nlq/ask`.

### 5. Frontend Integration

Add routes to your app:

```tsx
import NLQPage from './pages/nlq/NLQPage';
import LLMConfigPage from './pages/admin/LLMConfigPage';

// In your router:
<Route path="/nlq" element={<NLQPage />} />
<Route path="/admin/llm" element={<LLMConfigPage />} />
```

## Usage

### Basic Q&A Flow

1. User navigates to `/nlq`
2. Tenant picker ensures scope is selected
3. User asks: "How is monthly revenue calculated?"
4. Frontend calls: `POST /api/nlq/ask { "question": "..." }`
5. Backend:
   - Generates embedding for question
   - Finds closest catalog node via similarity search
   - Builds calculation DAG via `get_calc_dag_with_metadata()`
   - Constructs LLM prompt with DAG JSON
   - Sends to Gemini API
   - Parses response and enriches with metadata
6. Frontend displays:
   - Natural language answer
   - List of source nodes with paths and types
   - Data quality caveats (freshness, null rates)
   - Calculation breakdown (DAG structure)
   - Confidence level

### Example API Request

```bash
curl -X POST http://localhost:8080/api/nlq/ask \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000000" \
  -H "X-Tenant-Datasource-ID: 11111111-1111-1111-1111-111111111111" \
  -d '{
    "question": "How is monthly revenue calculated?"
  }'
```

### Example API Response

```json
{
  "answer": "Monthly Revenue is calculated as the sum of order_amount from the orders_clean view, after excluding refunds. The metric aggregates daily order totals into monthly buckets using the transaction_date field.",
  "calculation_breakdown": {
    "components": [
      {
        "id": "...",
        "name": "Monthly Revenue",
        "path": "finance.metrics.revenue.monthly",
        "type": "metric"
      },
      {
        "id": "...",
        "name": "Orders Clean",
        "path": "sales.views.orders_clean",
        "type": "view"
      }
    ],
    "dependencies": [
      {
        "source": "...",
        "target": "...",
        "relationship": "depends_on"
      }
    ]
  },
  "sources": [
    {
      "path": "finance.metrics.revenue.monthly",
      "name": "Monthly Revenue",
      "type": "metric",
      "metadata": {
        "data_quality": {
          "freshness": "3h",
          "null_rate": "0.5%"
        },
        "sla": {
          "availability": "99.9%"
        }
      }
    },
    {
      "path": "sales.views.orders_clean",
      "name": "Orders Clean",
      "type": "view",
      "metadata": {
        "lineage": {
          "upstream": ["raw.orders"]
        }
      }
    }
  ],
  "confidence": "High",
  "resolved_entity_path": "finance.metrics.revenue.monthly",
  "caveats": [
    "Data freshness: 3h",
    "Null rate: 0.5%"
  ]
}
```

## Admin Configuration

### LLM Provider Management

Navigate to `/admin/llm` to:

1. **Select Provider**: Gemini, OpenAI, Anthropic
2. **Choose Model**: `gemini-2.0-flash-exp`, `gpt-4`, `claude-3-opus`, etc.
3. **Set API Key**: Securely stored server-side
4. **Tune Parameters**: Temperature (creativity), max tokens (length)
5. **Test Configuration**: Verify setup before going live

## Extending the System

### Adding a New LLM Provider

1. Implement the `LLMProvider` interface in `backend/internal/llm/`:

```go
type OpenAIProvider struct {
    APIKey    string
    ModelName string
    Client    *http.Client
}

func (o *OpenAIProvider) GenerateResponse(ctx context.Context, prompt string) (string, error) {
    // Call OpenAI API
}

func (o *OpenAIProvider) Embed(ctx context.Context, text string) ([]float32, error) {
    // Call OpenAI embedding API
}
```

2. Update `api.go` to support provider selection:

```go
providerType := os.Getenv("LLM_PROVIDER") // "gemini", "openai", "anthropic"
var llmProvider llm.LLMProvider
switch providerType {
case "openai":
    llmProvider = llm.NewOpenAIProvider(...)
case "anthropic":
    llmProvider = llm.NewAnthropicProvider(...)
default:
    llmProvider = llm.NewGeminiProvider(...)
}
```

3. Add to frontend provider list in `LLMConfigPage.tsx`

### Customizing the Prompt

Edit `buildEnhancedPrompt()` in `nlq_service.go`:

```go
func (s *NLQService) buildEnhancedPrompt(question, dagJSON string) (string, error) {
    systemPrompt := `
SYSTEM: You are a [your custom persona].

INSTRUCTIONS:
- [Your custom instructions]
- [Additional rules]

CONTEXT:
%s

USER QUESTION:
%s

ASSISTANT RESPONSE:
`
    return fmt.Sprintf(systemPrompt, dagJSON, question), nil
}
```

### Adding More Metadata to Responses

Enhance `parseEnhancedLLMResponse()` in `nlq_service.go`:

```go
// Extract custom fields from catalog_node
if customField, ok := n["custom_field"].(string); ok && customField != "" {
    source.Metadata["custom_field"] = customField
}
```

## Monitoring & Observability

### Logging

The system logs:
- Embedding generation progress
- Semantic search queries and results
- LLM API calls and response times
- Errors and failures

### Metrics (Future)

Consider tracking:
- Questions per minute
- Average response time
- LLM token usage
- Cache hit rate (if caching responses)
- User satisfaction (thumbs up/down)

## Troubleshooting

### "No embedding found" Errors

**Cause**: Catalog nodes haven't been embedded yet.

**Fix**: Run the embedding generation job:

```go
embeddingService.GenerateEmbeddingsForTenant(ctx, tenantID, datasourceID)
```

### "Target entity not found" Errors

**Cause**: The qualified_path doesn't exist in catalog_node for that tenant.

**Fix**: 
1. Verify the path exists: `SELECT * FROM catalog_node WHERE qualified_path = '...' AND tenant_id = '...'`
2. Check spelling and case sensitivity
3. Ensure tenant/datasource scope is correct

### Slow Semantic Search

**Cause**: Missing or inefficient vector index.

**Fix**: Rebuild the index with more lists:

```sql
DROP INDEX idx_catalog_node_embedding;
CREATE INDEX idx_catalog_node_embedding 
ON catalog_node USING ivfflat (embedding vector_cosine_ops)
WITH (lists = 200);
```

Or use HNSW for better accuracy (slower build, faster query):

```sql
CREATE INDEX idx_catalog_node_embedding_hnsw
ON catalog_node USING hnsw (embedding vector_cosine_ops);
```

### LLM API Rate Limits

**Cause**: Too many requests to Gemini/OpenAI/etc.

**Fix**:
1. Implement response caching (store question hash → answer)
2. Add rate limiting in the backend
3. Use a lower-tier model for less critical questions
4. Batch embedding generation with delays

## Security Considerations

1. **API Keys**: Store in environment variables or secret management service, never commit to Git
2. **Tenant Isolation**: All queries are scoped by tenant_id and datasource_id
3. **Input Sanitization**: Frontend validates questions before submission
4. **LLM Output Validation**: Responses are parsed and filtered before display
5. **Access Control**: Admin config page should require elevated permissions

## Performance Optimization

1. **Embedding Cache**: Store embeddings in Redis for frequently asked questions
2. **DAG Cache**: Cache calculation DAGs (they rarely change)
3. **LLM Response Cache**: Cache answer for (question, entity_path, tenant) tuple
4. **Index Tuning**: Adjust pgvector `lists` parameter based on catalog size
5. **Async Embeddings**: Generate embeddings in background job, not on-demand

## Next Steps

1. **Feedback Loop**: Add thumbs up/down for answers → retrain or adjust prompts
2. **Conversation History**: Store chat sessions for follow-up questions
3. **Multi-turn Dialogue**: Support context from previous messages
4. **Voice Input**: Integrate speech-to-text for voice queries
5. **Export**: Allow users to export Q&A sessions as PDF reports
6. **Analytics Dashboard**: Show most asked questions, popular entities, etc.

## References

- [pgvector Documentation](https://github.com/pgvector/pgvector)
- [Gemini API Documentation](https://ai.google.dev/docs)
- [PostgreSQL Recursive Queries](https://www.postgresql.org/docs/current/queries-with.html)
- [Cube.dev Documentation](https://cube.dev/docs)

## Support

For issues or questions:
1. Check the troubleshooting section above
2. Review logs in `backend/logs/`
3. Open an issue in the repo with:
   - Error message
   - Tenant/datasource IDs
   - Question that caused the error
   - Relevant log snippets
