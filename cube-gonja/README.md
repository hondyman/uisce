# Cube Gonja Service

A Go-native service for rendering Jinja-like templates into Cube YAML models with hard data source binding and validation.

## Features

- **Go-native Gonja rendering** - No Python dependencies
- **Hard data source binding** - Enforces explicit data sources for each cube/view
- **Validation** - Pre-apply checks for compliance
- **Atomic file writes** - Safe updates to model files
- **Cube reload triggering** - Automatic reload after rendering
- **REST API** - Full control via HTTP endpoints
- **Template Management** - List and validate templates
- **Context Versioning** - History and rollback capabilities
- **Health Monitoring** - Detailed health checks and metrics
- **Configuration Validation** - Pre-flight config validation
- **Prometheus Metrics** - Service monitoring and alerting

## Quick Start

1. Build the service:
   ```bash
   make build
   ```

2. Run the service:
   ```bash
   ./gonja-service
   ```

3. Update context (data sources and dimensions):
   ```bash
   curl -X POST http://localhost:5001/update-context \
     -H "Content-Type: application/json" \
     -d '{
       "data_sources": {"orders": "default", "customers": "default"},
       "dimensions": {
         "orders": [{"name": "id", "sql": "id", "type": "number"}],
         "customers": [{"name": "id", "sql": "id", "type": "number"}]
       }
     }'
   ```

4. Render all templates:
   ```bash
   curl -X POST http://localhost:5001/render-all
   ```

5. Validate rendered models:
   ```bash
   make validate
   ```

## API Endpoints

### Core Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/healthz` | Simple health check |
| `GET` | `/health` | Detailed health status |
| `GET` | `/metrics` | Prometheus metrics |
| `GET` | `/templates` | List available templates |
| `POST` | `/validate-template` | Validate a specific template |
| `POST` | `/validate-config` | Validate configuration before applying |

### Context Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/update-context` | Update data sources and dimensions |
| `GET` | `/context/stats` | Get context statistics |
| `GET` | `/context/history` | Get version history |
| `POST` | `/context/rollback` | Rollback to previous version |

### Rendering

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/render` | Render a single template |
| `POST` | `/render-all` | Render all templates |
| `POST` | `/validate-dry-run` | Validate templates before applying |
| `POST` | `/preview` | Preview rendered template output |

## New Features

### 🏥 Health & Monitoring

**Detailed Health Check:**
```bash
curl http://localhost:5001/health
```
Returns uptime, template count, context stats, and last error.

**Prometheus Metrics:**
```bash
curl http://localhost:5001/metrics
```
Exposes metrics for monitoring:
- Template count
- Data source count
- Cube count
- Service uptime
- Version history size

### 📋 Template Management

**List Templates:**
```bash
curl http://localhost:5001/templates
```
Returns all available templates with metadata.

**Validate Template:**
```bash
curl -X POST http://localhost:5001/validate-template \
  -H "Content-Type: application/json" \
  -d '{"template_name": "orders"}'
```

### 🔄 Context Versioning

**View History:**
```bash
curl http://localhost:5001/context/history
```
Returns all previous context versions.

**Rollback:**
```bash
curl -X POST http://localhost:5001/context/rollback \
  -H "Content-Type: application/json" \
  -d '{"version_id": "1640995200"}'
```

**Context Statistics:**
```bash
curl http://localhost:5001/context/stats
```
Returns counts of data sources, cubes, dimensions, etc.

### ✅ Validation

**Configuration Validation:**
```bash
curl -X POST http://localhost:5001/validate-config \
  -H "Content-Type: application/json" \
  -d '{
    "data_sources": {"orders": "default"},
    "dimensions": {"orders": [{"name": "id", "sql": "id", "type": "number"}]}
  }'
```

**Template Validation:**
```bash
curl -X POST http://localhost:5001/validate-template \
  -H "Content-Type: application/json" \
  -d '{"template_name": "orders"}'
```

- `GET /healthz` - Health check
- `POST /update-context` - Update rendering context
- `POST /render` - Render single template
- `POST /render-all` - Render all templates
- `POST /validate-dry-run` - Validate and apply all templates atomically
- `POST /preview` - Preview rendered YAML without writing

## 🚀 New Features

### `/render-all`
Renders every `.yml.gonja` template in `/templates` and writes to `model-out/`.

```bash
curl -X POST http://localhost:5001/render-all
```

### `/validate-dry-run`
**Safe deployment workflow:**
1. Renders all templates to a temporary directory
2. Validates each rendered YAML for hard data source binding
3. If all pass, atomically replaces `model-out/` with validated models
4. Triggers Cube reload

```bash
curl -X POST http://localhost:5001/validate-dry-run
```

**Response:** `{"status":"validated and applied"}` or validation error

### Hard Data Source Binding
Every cube must have a `data_source` from the allowed set:
- `ALLOWED_DATA_SOURCES` environment variable
- Default: `default`
- Validation fails if any cube has invalid or missing data source

## CI/CD Integration

Add to your pipeline:

```yaml
- name: Validate data sources
  run: make validate

- name: Deploy models safely
  run: |
    curl -X POST http://localhost:5001/validate-dry-run
```

## CI/CD Integration

Add to your CI pipeline:

```yaml
- name: Validate data sources
  run: make validate
```

## Templates

Templates use Gonja syntax (Jinja-compatible):

```gonja
cubes:
  - name: {{ cube_name }}
    sql_table: {{ sql_table }}
    data_source: {{ get_data_source(cube_name) }}
    title: {{ title }}
```

## Wiring into Cube

1. Mount `model-out/` into your Cube container's model directory
2. Set multiple data sources in Cube configuration
3. Point Cube's model directory to the mounted path

## Development

- `make run` - Run in development mode
- `make tidy` - Clean up dependencies
- `make clean` - Clean build artifacts
