# Cube.js Multi-Tenant Deployment Guide

## Pre-Flight Checklist

Before deploying Cube.js to production, ensure:

- [ ] StarRocks cluster is running and healthy
- [ ] Trino is connected to Iceberg catalog
- [ ] Postgres database has tenant/datasource tables populated
- [ ] Environment variables are configured
- [ ] API secrets are generated and stored securely
- [ ] Pre-aggregation database is initialized

## Step-by-Step Deployment

### 1. Initialize StarRocks Pre-Aggregation Storage

```bash
# Connect to StarRocks FE
docker exec -i starrocks-fe mysql -uroot -h localhost -P 9030

# Run initialization script
source cube/init-starrocks-preaggs.sql

# Verify database created
SHOW DATABASES;
USE cube_preaggs;
SHOW TABLES;
```

### 2. Set Environment Variables

Create `.env` file in project root:

```bash
# Cube.js API Secret (generate with: openssl rand -hex 32)
CUBE_API_SECRET=your-production-secret-here

# StarRocks Connection (Hot Tier)
CUBEJS_DS_STARROCKS_DB_HOST=starrocks-fe
CUBEJS_DS_STARROCKS_DB_PORT=9030
CUBEJS_DS_STARROCKS_DB_USER=root
CUBEJS_DS_STARROCKS_DB_PASS=your-starrocks-password

# Trino Connection (Cold Tier)
CUBEJS_DS_TRINO_DB_HOST=trino
CUBEJS_DS_TRINO_DB_PORT=8080
CUBEJS_DS_TRINO_DB_USER=admin

# Pre-Aggregation Storage (StarRocks)
CUBEJS_EXT_DB_HOST=starrocks-fe
CUBEJS_EXT_DB_PORT=9030
CUBEJS_EXT_DB_NAME=cube_preaggs
CUBEJS_EXT_DB_USER=root
CUBEJS_EXT_DB_PASS=your-starrocks-password

# Security
CUBEJS_DEV_MODE=false
CUBEJS_ROLLUP_ONLY=true

# Performance
CUBEJS_CONCURRENCY=4
CUBEJS_CACHE_AND_QUEUE_DRIVER=memory
```

### 3. Start Cube.js Service

```bash
# Start all services
docker compose up -d

# Wait for Cube.js to be healthy
docker compose ps cube

# Check logs
docker logs -f cube-semantic-layer
```

### 4. Verify Connectivity

```bash
# Health check
curl http://localhost:4000/readyz

# Should return: {"health":"HEALTH_STATUS_SERVING"}
```

### 5. Test Tenant-Scoped Query

```bash
# Replace with actual tenant/datasource IDs from your database
export TENANT_ID="00000000-0000-0000-0000-000000000000"
export DATASOURCE_ID="11111111-1111-1111-1111-111111111111"
export API_SECRET="your-production-secret-here"

curl -X POST http://localhost:4000/cubejs-api/v1/load \
  -H "Content-Type: application/json" \
  -H "Authorization: ${API_SECRET}" \
  -H "X-Tenant-ID: ${TENANT_ID}" \
  -H "X-Tenant-Datasource-ID: ${DATASOURCE_ID}" \
  -d '{
    "query": {
      "measures": ["Trades.count"],
      "timeDimensions": [{
        "dimension": "Trades.event_time",
        "granularity": "day",
        "dateRange": "last 7 days"
      }]
    }
  }'
```

### 6. Configure Backend to Use Cube.js

Update `backend/cmd/server/main.go`:

```go
import (
    "github.com/hondyman/semlayer/backend/internal/cube"
    httpapi "github.com/hondyman/semlayer/backend/internal/api"
)

func main() {
    // ... existing setup ...
    
    // Initialize Cube.js client
    cubeClient := cube.NewClient(
        os.Getenv("CUBE_API_URL"),  // http://cube:4000
        os.Getenv("CUBE_API_SECRET"),
    )
    
    // Register Cube.js API handler
    cubeHandler := httpapi.NewCubeHandler(cubeClient)
    apiGroup := router.Group("/api")
    cubeHandler.RegisterCubeRoutes(apiGroup)
    
    // ... rest of setup ...
}
```

### 7. Enable Scheduled Refresh

Tenant-aware refresh contexts are now generated from the platform database. Run the sync script any time tenants, datasources, or schema overrides change:

```bash
# Ensure DATABASE_URL (or ALPHA_DB_URL) is populated
make sync-cube-tenants
```

This command runs `scripts/sync_cube_tenants.go`, which:

1. Reads `tenant_product_datasource` for every active tenant scope.
2. Produces `cube/generated/tenant-scopes.json`, which Cube loads at startup for `scheduledRefreshContexts`, QoS routing, and header validation.
3. Writes any `schema_overrides` JSON into `cube/schema/tenants/<tenant>/<datasource>/auto/*.yml`, so tenant overlays are automatically picked up by the `repositoryFactory`.

If you need to store the JSON elsewhere (for example, when packaging the service), set `CUBE_TENANT_CONFIG_PATH=/path/to/tenant-scopes.json` before starting Cube.

## Production Hardening

### 1. JWT Authentication

Replace header-based auth with JWT:

```javascript
// cube/cube.js
const jwt = require('jsonwebtoken');

checkAuth: async (req, authorization) => {
  if (!authorization) {
    throw new Error('Authorization header required');
  }
  
  const token = authorization.replace('Bearer ', '');
  
  try {
    const decoded = jwt.verify(token, process.env.JWT_SECRET);
    
    return {
      tenant_id: decoded.tenant_id,
      datasource_id: decoded.datasource_id,
      user_id: decoded.sub,
      role: decoded.role
    };
  } catch (err) {
    throw new Error('Invalid token');
  }
}
```

### 2. Resource Limits

Create StarRocks resource groups for each tenant tier:

```sql
-- Connect to StarRocks
USE cube_preaggs;

-- Premium tier
CREATE RESOURCE GROUP tenant_premium
WITH (
  cpu_weight = 10,
  mem_limit = '40%',
  concurrency_limit = 20,
  type = 'normal'
);

-- Standard tier
CREATE RESOURCE GROUP tenant_standard
WITH (
  cpu_weight = 5,
  mem_limit = '30%',
  concurrency_limit = 10,
  type = 'normal'
);

-- Assign tenants to resource groups
-- (Do this as part of tenant provisioning)
SET PROPERTY FOR 'tenant_a_user' 'resource_group' = 'tenant_premium';
```

### 3. CORS Configuration

Update `cube/cube.js` for production origins:

```javascript
http: {
  cors: {
    origin: [
      'https://your-production-domain.com',
      'https://app.your-domain.com'
    ],
    credentials: true,
    methods: ['GET', 'POST', 'OPTIONS']
  }
}
```

### 4. SSL/TLS

Use a reverse proxy (nginx/Traefik) to terminate SSL:

```nginx
# /etc/nginx/sites-available/cube
server {
  listen 443 ssl http2;
  server_name cube.your-domain.com;
  
  ssl_certificate /path/to/cert.pem;
  ssl_certificate_key /path/to/key.pem;
  
  location / {
    proxy_pass http://localhost:4000;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection 'upgrade';
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
  }
}
```

### 5. Monitoring

Set up Prometheus metrics:

```javascript
// cube/cube.js
const client = require('prom-client');

const queryCounter = new client.Counter({
  name: 'cube_queries_total',
  help: 'Total number of queries',
  labelNames: ['tenant_id', 'cube', 'status']
});

const queryDuration = new client.Histogram({
  name: 'cube_query_duration_seconds',
  help: 'Query duration in seconds',
  labelNames: ['tenant_id', 'cube']
});

// In middleware
http: {
  middleware: [
    (req, res, next) => {
      const start = Date.now();
      const tenantId = req.headers['x-tenant-id'];
      
      res.on('finish', () => {
        const duration = (Date.now() - start) / 1000;
        queryCounter.inc({ tenant_id: tenantId, status: res.statusCode });
        queryDuration.observe({ tenant_id: tenantId }, duration);
      });
      
      next();
    }
  ]
}
```

## Scaling Strategy

### Horizontal Scaling

Run multiple Cube.js instances behind a load balancer:

```yaml
# docker-compose.prod.yml
services:
  cube-1:
    image: cubejs/cube:latest
    environment: ...
    
  cube-2:
    image: cubejs/cube:latest
    environment: ...
    
  cube-lb:
    image: nginx:alpine
    volumes:
      - ./nginx-cube.conf:/etc/nginx/nginx.conf:ro
    ports:
      - "4000:80"
    depends_on:
      - cube-1
      - cube-2
```

### Vertical Scaling

Increase Cube.js concurrency:

```bash
CUBEJS_CONCURRENCY=8          # CPU cores
CUBEJS_DB_MAX_POOL=20         # Database connections
```

### Pre-Aggregation Optimization

Monitor and optimize pre-aggregations:

```sql
-- StarRocks: Check pre-aggregation usage
SELECT 
  cube_name,
  COUNT(*) as rollup_count,
  SUM(row_count) / 1000000 as million_rows,
  SUM(storage_bytes) / (1024*1024*1024) as storage_gb,
  AVG(TIMESTAMPDIFF(MINUTE, last_refresh, NOW())) as avg_age_minutes
FROM cube_preaggs.preagg_metadata
GROUP BY cube_name
ORDER BY million_rows DESC;
```

## Disaster Recovery

### Backup Strategy

```bash
#!/bin/bash
# backup-cube-metadata.sh

# Backup Cube schema definitions
tar -czf cube-schema-$(date +%Y%m%d).tar.gz cube/schema/

# Backup StarRocks pre-aggregations metadata
docker exec starrocks-fe mysql -uroot -e "
  USE cube_preaggs;
  SELECT * INTO OUTFILE '/tmp/preagg_metadata.csv'
  FIELDS TERMINATED BY ','
  FROM preagg_metadata;
"

# Upload to S3
aws s3 cp cube-schema-*.tar.gz s3://backups/cube/
```

### Recovery Procedure

1. Restore schema files
2. Recreate pre-aggregations (will auto-refresh)
3. Verify queries return expected results

## Performance Benchmarks

Expected query performance:

| Query Type | Without Pre-Agg | With Pre-Agg |
|------------|----------------|--------------|
| Simple aggregation | 2-5s | 50-200ms |
| Time series (7 days) | 5-10s | 100-300ms |
| Complex rollup | 30-60s | 200-500ms |

## Troubleshooting

### Issue: Pre-aggregations not building

**Solution**:
```bash
# Check Cube.js logs
docker logs cube-semantic-layer | grep -i "pre-aggregation"

# Manually trigger refresh via API
curl -X POST http://localhost:4000/cubejs-api/v1/pre-aggregations/jobs \
  -H "Authorization: ${API_SECRET}" \
  -H "X-Tenant-ID: ${TENANT_ID}" \
  -d '{"action": "refresh", "selector": {"contexts": [{"tenant_id": "..."}]}}'
```

### Issue: Queries timeout

**Possible causes**:
1. Missing pre-aggregation → Create one
2. Trino overload → Check Trino UI
3. StarRocks slow → Check resource groups

### Issue: Wrong tenant data returned

**Solution**: Verify `queryRewrite` is injecting filters:

```bash
# Enable debug logging
docker compose exec cube sh -c 'export DEBUG=cube* && cube server'

# Check generated SQL includes tenant_id filter
```

## Support & Resources

- Cube.js Community: https://slack.cube.dev
- StarRocks Forum: https://github.com/StarRocks/starrocks/discussions
- Internal Wiki: Link to your internal docs

## Next Steps

1. Load production data into StarRocks/Trino
2. Create tenant-specific cube overrides
3. Set up monitoring dashboards
4. Train users on SQL API for BI tools
5. Optimize pre-aggregation refresh schedules based on usage patterns
