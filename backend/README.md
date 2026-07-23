
# SemLayer - Easy Configuration

SemLayer is now incredibly easy to configure! This guide shows you how to get started in minutes.

## 🚀 Quick Start (3 Steps)

### Step 1: Run the Setup Script
```bash
# Clone the repository and navigate to backend
cd semlayer/backend

# Run the automated setup (it detects your environment automatically)
./setup.sh
```

That's it! The setup script will:
- ✅ Detect your environment (local, Docker, CI)
- ✅ Create appropriate configuration files
- ✅ Set up environment variables
- ✅ Generate secure secrets
- ✅ Provide next steps

### Step 2: Review Configuration
The setup script creates several files:
- `.env` - Environment variables (recommended)
- `config.yaml` - YAML configuration file
- `docker-compose.yml` - For Docker environments
- `setup-db.sh` - Database setup helper

### Step 3: Start the Application
```bash
# With environment variables (recommended)
source .env && go run ./cmd/server

# Or with config file
go run ./cmd/server --config config.yaml

# Or with Docker
docker-compose up -d
```

## 📋 Configuration Methods

### Method 1: Environment Variables (Easiest)
```bash
# Your .env file (created by setup.sh)
ENVIRONMENT=development
DRIVER=postgres
DSN=host=localhost port=5432 user=postgres password=password dbname=semlayer sslmode=disable
# Or rely on DATABASE_URL (postgres:// or postgresql://); the app will normalize it automatically
DATABASE_URL=postgresql://postgres:postgres@localhost:5432/alpha?sslmode=disable
PORT=:8080
JWT_SECRET=your-generated-secret
LOG_LEVEL=debug
ENABLE_METRICS=true
ENABLE_CACHING=true
ENABLE_SECURITY=false
```

### Method 2: YAML Configuration File
```yaml
# config.yaml (also created by setup.sh)
yaml_dir: "./config"
driver: "postgres"
dsn: "host=localhost port=5432 user=postgres password=password dbname=semlayer sslmode=disable"
port: ":8080"
environment: "development"
log_level: "debug"
enable_metrics: true
enable_caching: true
enable_security: false
jwt_secret: "your-generated-secret"
```

### Method 3: Docker (Zero Configuration)
```bash
# Just run - everything is configured automatically
docker-compose up -d

# View logs
docker-compose logs -f semlayer
```

## 🔧 Configuration Options

### Core Settings
| Setting | Environment Variable | Default | Description |
|---------|---------------------|---------|-------------|
| `ENVIRONMENT` | `ENVIRONMENT` | `development` | Environment type |
| `DRIVER` | `DRIVER` | `postgres` | Database driver |
| `DSN` | `DSN` / `DATABASE_URL` | - | Database connection string |
| `PORT` | `PORT` | `:8080` | HTTP server port |
| `JWT_SECRET` | `JWT_SECRET` | - | JWT signing secret |

### Performance Settings
| Setting | Environment Variable | Default | Description |
|---------|---------------------|---------|-------------|
| `ENABLE_CACHING` | `ENABLE_CACHING` | `true` | Enable advanced caching |
| `ENABLE_METRICS` | `ENABLE_METRICS` | `true` | Enable metrics collection |
| `CACHE_NUM_SHARDS` | `CACHE_NUM_SHARDS` | `16` | Number of cache shards |
| `DB_MAX_OPEN_CONNS` | `DB_MAX_OPEN_CONNS` | `25` | Max database connections |

### Security Settings
| Setting | Environment Variable | Default | Description |
|---------|---------------------|---------|-------------|
| `ENABLE_SECURITY` | `ENABLE_SECURITY` | `true` | Enable security features |
| `RATE_LIMIT_ENABLED` | `RATE_LIMIT_ENABLED` | `true` | Enable rate limiting |
| `RATE_LIMIT_REQUESTS` | `RATE_LIMIT_REQUESTS` | `100` | Requests per window |

## 🌍 Environment-Specific Configurations

### Development
- **Security**: Disabled for easier development
- **Logging**: Debug level
- **Caching**: Basic local caching
- **Database**: Minimal connections

### Staging
- **Security**: Enabled with moderate rate limiting
- **Logging**: Info level
- **Caching**: Redis-backed caching
- **Database**: Moderate connection pooling

### Production
- **Security**: Full security with strict rate limiting
- **Logging**: Warn level only
- **Caching**: Distributed Redis caching
- **Database**: Optimized connection pooling

## 🐳 Docker Quick Start

```bash
# 1. Run setup (creates docker-compose.yml)
./setup.sh

# 2. Start everything
docker-compose up -d

# 3. Check logs
docker-compose logs -f semlayer

# 4. Access the application
open http://localhost:8080
```

## 🔍 Troubleshooting

### Configuration Validation
```bash
# The app validates configuration on startup
go run ./cmd/server

# Enable debug logging
LOG_LEVEL=debug go run ./cmd/server
```

### Common Issues

**Database Connection Failed**
```bash
# Check your DSN
export DSN="host=localhost port=5432 user=postgres password=password dbname=semlayer sslmode=disable"

# Or set DATABASE_URL and let the server normalize it
export DATABASE_URL="postgresql://postgres:postgres@localhost:5432/alpha?sslmode=disable"

# Or run the database setup script
./setup-db.sh
```

**JWT Secret Not Set**
```bash
# Generate a new secret
openssl rand -base64 32

# Update your .env file
echo "JWT_SECRET=your-new-secret" >> .env
```

**Port Already in Use**
```bash
# Find what's using the port
lsof -i :8080

# Use a different port
export PORT=:8081
```

## 📚 Advanced Configuration

For detailed configuration options, see:
- `CONFIGURATION_GUIDE.md` - Complete configuration reference
- `config.example.dev.yaml` - Development example
- `config.example.prod.yaml` - Production example
- `config.example.staging.yaml` - Staging example

## 🎯 Environment Variables Reference

```bash
# Quick reference of all environment variables
ENVIRONMENT=development                    # dev/staging/prod
DRIVER=postgres                           # postgres/snowflake/mssql
DSN=host=localhost...                     # Database connection
DATABASE_URL=postgres://...               # Alternate database connection (auto-normalized)
PORT=:8080                                # HTTP port
JWT_SECRET=your-secret                    # JWT signing key
LOG_LEVEL=debug                           # debug/info/warn/error
ENABLE_METRICS=true                       # true/false
ENABLE_CACHING=true                       # true/false
ENABLE_SECURITY=true                      # true/false
REDIS_ADDR=localhost:6379                 # Redis address
DB_MAX_OPEN_CONNS=25                      # Database connections
CACHE_NUM_SHARDS=16                       # Cache shards
RATE_LIMIT_REQUESTS=100                   # Rate limit per minute
```

## 🔍 Cube Parity Comparator Service

`cmd/cube-parity` runs the parity checks described in Phase 10. It exposes a simple HTTP API and persists results to `migration.parity_results` (DDL in `sql/parity_results.sql`).

### Run locally

```bash
go run ./cmd/cube-parity \
	-addr=:8090 \
	-tolerance=0.0001 \
	-dsn="postgresql://postgres:postgres@localhost:5432/alpha?sslmode=disable"
```

Configuration flags:

| Flag / Env | Description |
| --- | --- |
| `-addr` | HTTP listen address (default `:8090`). |
| `-tolerance` | Numeric tolerance before a mismatch is recorded (default `1e-6`). |
| `-dsn` / `PARITY_DATABASE_URL` | Optional Postgres/StarRocks DSN for persisting parity rows. |

### API

`POST /compare` (single payload) or `POST /compare/batch` (array) accept:

```json
{
	"tenant_id": "tenant-123",
	"query_id": "positions-q1",
	"legacy_payload": {"mv": 100.0},
	"cube_payload": {"mv": 100.0004},
	"tolerance": 0.001,
	"metadata": {"source_trace_id": "abc"}
}
```

Responses echo the computed `status`, `max_delta`, hashes, and diff summary. When a DSN is supplied each result is written to the parity table for downstream Grafana/Flagger gates.

## 🚀 Getting Help

- **Configuration Guide**: `CONFIGURATION_GUIDE.md`
- **Example Files**: `config.example.*.yaml`
- **Setup Script**: `./setup.sh --help`
- **Validation**: The app shows helpful error messages

---

**That's it!** SemLayer is now incredibly easy to configure. Just run `./setup.sh` and you're ready to go! 🎉

---

## Database migrations (development)

The backend image runs the migration runner during container startup. The migration runner now includes the seed file `migrations/000023_seed_private_markets_bundles.sql` which inserts example private markets bundles.

Local options:
- Run migrations via Makefile: `make migrate` (runs the migration runner)
- Run a single SQL migration directly: set `POSTGRES_URL` and run `make migrate-sql`

When running with Docker Compose, the backend container will execute migrations on startup (the image entrypoint runs the migration runner before launching the server).
