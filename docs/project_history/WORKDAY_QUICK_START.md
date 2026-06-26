# 🎯 Workday UI System - 5-Minute Setup

## Pre-Check: What's Working

✅ **Backend Go Code**: Fully implemented and compiling  
✅ **Database Schema**: Ready to deploy  
✅ **Example Data**: Ready to load  
❌ **Previous Command**: Exit code 1 (database connection issue)

---

## Step 1: Fix Database Connection (2 minutes)

### Is PostgreSQL running?

```bash
# Check if PostgreSQL is running
ps aux | grep postgres | grep -v grep

# If you see a process, it's running. If not, start it:
brew services start postgresql

# Verify it's actually running on port 5432
netstat -an | grep 5432
```

### Create the database

```bash
# Connect as postgres user and create database
createdb -U postgres alpha

# Verify it exists
psql -U postgres -l | grep alpha

# Should show: alpha | postgres | UTF8 | ...
```

---

## Step 2: Deploy the Schema (1 minute)

```bash
# From semlayer root
cd /Users/eganpj/GitHub/semlayer

# Run the schema
psql -U postgres -d alpha -f backend/db/migrations/workday_metadata_schema.sql

# Watch for CREATE TABLE and GRANT messages (no errors)
```

### Verify schema created

```bash
# List all tables
psql -U postgres -d alpha -c "\dt"

# Should show 11 tables
```

---

## Step 3: Load Example Data (1 minute)

```bash
# Load the Employee example
psql -U postgres -d alpha -f backend/db/migrations/example_hire_employee_setup.sql

# Watch for INSERT messages
```

### Verify data loaded

```bash
# Check Business Objects
psql -U postgres -d alpha -c "SELECT bo_name FROM business_objects;"

# Should show: Employee
```

---

## Step 4: Start Backend (1 minute)

```bash
# Build
cd /Users/eganpj/GitHub/semlayer
go build -o ./bin/api ./cmd/api

# Start the API
./bin/api

# Expected output:
# Starting Semlayer API...
# Listening on :8080
```

---

## Step 5: Test the System (1 minute)

### In another terminal:

```bash
# Get a layout ID
LAYOUT_ID=$(psql -U postgres -d alpha -t -c "SELECT id FROM page_layouts LIMIT 1;" | xargs)

echo "Layout ID: $LAYOUT_ID"

# Test the form definition endpoint
curl -s \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -H "X-Tenant-Datasource-ID: 11111111-1111-1111-1111-111111111111" \
  "http://localhost:8080/api/ui/forms/$LAYOUT_ID" | jq . | head -30

# Expected: JSON with form definition
```

---

## ✅ Success Checklist

- [ ] PostgreSQL running on :5432
- [ ] Database `alpha` created
- [ ] Schema deployed (11 tables)
- [ ] Example data loaded
- [ ] Backend compiled and running on :8080
- [ ] GET /api/ui/forms/:layoutId returns FormDefinition
- [ ] POST /api/ui/validate works
- [ ] POST /api/ui/submit works

---

## 🎨 Next: Build React Frontend

See `REACT_FRONTEND_IMPLEMENTATION.md` for complete React code (ready to copy/paste).

---

## 🐛 Troubleshooting

| Issue | Solution |
|-------|----------|
| `psql: command not found` | `brew install postgresql` |
| `could not connect to server` | `brew services start postgresql` |
| `database "alpha" does not exist` | `createdb -U postgres alpha` |
| `permission denied` | Verify postgres user and password |
| Backend won't start | Check port 8080 is free: `lsof -i :8080` |

---

That's it! You're ready to build. 🚀
