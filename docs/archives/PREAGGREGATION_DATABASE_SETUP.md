# Preaggregation Database Setup Guide

Since PostgreSQL is not installed locally, here are the options to set up the preaggregation database schema:

## Option 1: Docker Setup (Recommended)

### 1. Start PostgreSQL with Docker
```bash
# Create a PostgreSQL container for testing
docker run --name postgres-semlayer \
  -e POSTGRES_DB=semlayer \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -p 5432:5432 \
  -d postgres:13

# Wait for PostgreSQL to start
sleep 10

# Verify connection
docker exec -it postgres-semlayer psql -U postgres -d semlayer -c "SELECT version();"
```

### 2. Run the Migration
```bash
# Copy the migration file to the container
docker cp /Users/eganpj/GitHub/semlayer/backend/migrations/000015_preaggregation_schema.sql postgres-semlayer:/tmp/

# Run the migration
docker exec -it postgres-semlayer psql -U postgres -d semlayer -f /tmp/000015_preaggregation_schema.sql
```

### 3. Verify Setup
```bash
# Check that the schema was created
docker exec -it postgres-semlayer psql -U postgres -d semlayer -c "SELECT schema_name FROM information_schema.schemata WHERE schema_name = 'semantic_layer';"

# Check that tables were created
docker exec -it postgres-semlayer psql -U postgres -d semlayer -c "SELECT table_name FROM information_schema.tables WHERE table_schema = 'semantic_layer';"

# Check sample data
docker exec -it postgres-semlayer psql -U postgres -d semlayer -c "SELECT COUNT(*) FROM semantic_layer.preaggregated_metrics;"
```

## Option 2: Manual SQL Execution

If you have access to a PostgreSQL database through other means (pgAdmin, cloud database, etc.):

### 1. Connect to your database
- Use your preferred PostgreSQL client
- Connect to your target database

### 2. Run the migration SQL
- Open the file: `/Users/eganpj/GitHub/semlayer/backend/migrations/000015_preaggregation_schema.sql`
- Execute the entire SQL script in your database

### 3. Verify the setup
```sql
-- Check schema creation
SELECT schema_name FROM information_schema.schemata WHERE schema_name = 'semantic_layer';

-- Check table creation
SELECT table_name FROM information_schema.tables WHERE table_schema = 'semantic_layer';

-- Check sample data
SELECT COUNT(*) FROM semantic_layer.preaggregated_metrics;
```

## Option 3: Update Configuration

If you want to use the existing database setup from your project:

### 1. Update the config.yaml
The current config points to a Docker container. Update it to point to your available database:

```yaml
yaml_dir: ./models
driver: postgres
dsn: "postgres://your_user:your_password@your_host:5432/your_database?sslmode=disable"
port: :8080
pg_port: :5432
```

### 2. Run the Go migration
```bash
cd /Users/eganpj/GitHub/semlayer/backend
go run cmd/migration/main.go
```

## Testing the Setup

Once the database schema is created, test it with:

### 1. Run the Preaggregation Demo
```bash
cd /Users/eganpj/GitHub/semlayer/backend/cmd/preaggregation
go run main.go
```

### 2. Test Individual Functions
```sql
-- Test the helper function
SELECT * FROM semantic_layer.get_preaggregated_metric(
    'private_markets_net_irr',
    '{"fund_id": "FUND001", "month": "2024-09-01T00:00:00Z"}'::jsonb,
    24
);

-- Check data quality
SELECT * FROM semantic_layer.get_data_quality_summary(7);
```

## Automated Scheduling Setup

After database setup, configure automated preaggregation:

### 1. Daily Jobs (6 AM UTC)
```bash
# Add to crontab
0 6 * * * cd /Users/eganpj/GitHub/semlayer/backend/cmd/preaggregation && /usr/local/go/bin/go run main.go
```

### 2. Weekly Jobs (Monday 6 AM UTC)
```bash
# Add to crontab
0 6 * * 1 cd /Users/eganpj/GitHub/semlayer/backend/cmd/preaggregation && /usr/local/go/bin/go run main.go weekly
```

## Troubleshooting

### Common Issues:

1. **Connection Refused**
   - Ensure PostgreSQL is running
   - Check connection string and credentials
   - Verify network/firewall settings

2. **Permission Denied**
   - Grant necessary permissions to the database user
   - Check schema creation permissions

3. **Migration Fails**
   - Check for syntax errors in the SQL
   - Ensure dependent objects don't exist
   - Review PostgreSQL logs

### Verification Queries:

```sql
-- Check all preaggregation tables
SELECT schemaname, tablename
FROM pg_tables
WHERE schemaname = 'semantic_layer'
ORDER BY tablename;

-- Check all functions
SELECT routine_name
FROM information_schema.routines
WHERE routine_schema = 'semantic_layer'
ORDER BY routine_name;

-- Check indexes
SELECT indexname, tablename
FROM pg_indexes
WHERE schemaname = 'semantic_layer'
ORDER BY tablename, indexname;
```

## Next Steps

Once the database is set up:

1. ✅ **Database Schema**: Preaggregation tables and functions created
2. ⏳ **Run Demo**: Test the preaggregation system
3. ⏳ **Configure Automation**: Set up scheduled jobs
4. ⏳ **Monitor Performance**: Track query improvements
5. ⏳ **Production Deployment**: Move to production environment

The preaggregation system is now ready for deployment! 🚀
