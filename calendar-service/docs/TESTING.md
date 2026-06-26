# Test Suite Documentation

## Overview

The Calendar Service includes a comprehensive test suite covering:
- **Repository Layer Tests** - Unit tests for all data access operations
- **Integration Tests** - Full database tests with test database setup/teardown
- **Cache Tests** - Redis caching validation
- **Availability Checker Tests** - Business logic validation

## Test Structure

```
calendar-service/
├── internal/
│   ├── repository/
│   │   ├── tenant_test.go (6 tests)
│   │   ├── profile_test.go (6 tests)
│   │   ├── holiday_blackout_test.go (11 tests)
│   │   └── metadata_audit_test.go (8 tests)
│   ├── availability/
│   │   └── checker_test.go (7 tests)
│   └── testutil/
│       ├── database.go (TestDB + migrations)
│       └── fixtures.go (Factory methods)
```

## Running Tests

### Prerequisites

1. **PostgreSQL** running on localhost:5432
2. **Database credentials** (can use environment variables):
   ```bash
   export TEST_DB_USER=calendar_user
   export TEST_DB_PASSWORD=calendar_password
   ```

3. **Go** 1.23+

### Run All Tests

```bash
cd calendar-service

# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific package
go test -v ./internal/repository

# Run specific test
go test -v ./internal/repository -run TestTenantCreate
```

### Run Only Unit Tests (Fast)

```bash
# Skip integration tests (marked with testing.Short())
go test -short ./...
```

### Run Only Integration Tests

```bash
# Run integration tests only
go test -v ./internal/repository
go test -v ./internal/availability
```

### Run with Coverage

```bash
# Generate coverage profile
go test -coverage ./...

# Generate detailed coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Coverage for specific package
go test -coverprofile=coverage.out ./internal/repository
go tool cover -html=coverage.out
```

### Run Tests in Parallel

```bash
# Run up to N tests in parallel
go test -parallel 4 ./...
```

### Run with Custom Timeout

```bash
# Increase timeout for slow systems
go test -timeout 10m ./...
```

## Test Utilities

### TestDB

Provides automatic test database creation and cleanup:

```go
func TestSomething(t *testing.T) {
    tdb := testutil.NewTestDB(t)
    defer tdb.Close(t)
    
    ctx := tdb.Context()
    
    // Use tdb.Repos for repository access
    err := tdb.Repos.Tenant.Create(ctx, tenant)
}
```

Features:
- Auto-creates isolated test database
- Runs migrations automatically
- Provides repository layer access
- Cleans up after test completes

### Fixtures

Factory methods for creating test data:

```go
fixtures, err := testutil.NewFixtures(ctx, tdb.Repos)

// Create test entities
holiday, err := fixtures.NewHoliday(ctx, tdb.Repos, date, "Christmas")
blackout, err := fixtures.NewBlackout(ctx, tdb.Repos, start, end, "Maintenance")
```

Methods:
- `NewHoliday()` - Create holiday
- `NewBlackout()` - Create blackout
- `NewRecurringBlackout()` - Create recurring blackout with RRULE
- `NewMetadata()` - Create resolved calendar metadata
- `NewAuditLog()` - Create audit log entry

## Test Coverage

### Repository Tests (31 tests)

#### Tenant Repository (6 tests)
- ✅ Create tenant
- ✅ Get by ID
- ✅ Get by name
- ✅ List all
- ✅ Update tenant
- ✅ Delete (soft delete)

#### Calendar Profile Repository (6 tests)
- ✅ Create profile
- ✅ Get by ID
- ✅ Get by name (tenant-scoped)
- ✅ List by tenant with pagination
- ✅ Update profile and metadata
- ✅ Soft delete and cache invalidation

#### Holiday Repository (5 tests)
- ✅ Create holiday
- ✅ List by profile
- ✅ List by date range
- ✅ Upsert (insert or update on unique constraint)
- ✅ Delete by profile

#### Blackout Window Repository (6 tests)
- ✅ Create blackout
- ✅ List by profile
- ✅ List by date range
- ✅ Create recurring blackout with RRULE
- ✅ Update blackout
- ✅ Delete blackout

#### Metadata Repository (3 tests)
- ✅ Upsert metadata
- ✅ Get by profile
- ✅ Invalidate by profile (clear cache markers)

#### Audit Log Repository (5 tests)
- ✅ Log audit entry
- ✅ Get by entity type and ID
- ✅ Get by tenant
- ✅ JSON serialization of changes
- ✅ Query ordering and pagination

### Availability Checker Tests (7 tests)

- ✅ Resolve profile with caching
- ✅ Check availability (basic)
- ✅ Check availability with holidays
- ✅ Check availability with blackouts
- ✅ Find next available slot
- ✅ Same day comparison logic
- ✅ Cache get/set operations

## Database Test Isolation

Each test gets an isolated database:
1. Temporary database created with unique name
2. Schema migrations applied
3. Fixtures created as needed
4. Database dropped after test completes

This ensures:
- ✅ No test interference
- ✅ Parallel test execution safety
- ✅ Clean database state per test
- ✅ Automatic cleanup

## Environment Configuration

Tests use environment variables for database configuration:

```bash
# Optional: override defaults
export TEST_DB_USER=custom_user
export TEST_DB_PASSWORD=custom_pass
export TEST_DB_HOST=localhost        # Default
export TEST_DB_PORT=5432            # Default
export TEST_DB_NAME=postgres        # Default (for creating test DB)
```

## Continuous Integration

### GitHub Actions Example

```yaml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    
    services:
      postgres:
        image: postgres:16-alpine
        env:
          POSTGRES_USER: calendar_user
          POSTGRES_PASSWORD: calendar_password
          POSTGRES_DB: postgres
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 5432:5432
    
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.23'
      
      - name: Run tests
        run: go test -v -coverprofile=coverage.out ./...
```

## Test Performance

### Benchmark Results (Approximate)

On modern hardware with local PostgreSQL:

| Test Suite | Time | Tests |
|-----------|------|-------|
| Repository (all) | 5-10s | 31 |
| Availability | 2-5s | 7 |
| Total | 10-15s | 38 |

### Optimization Tips

1. **Run short tests only** during development:
   ```bash
   go test -short ./...
   ```

2. **Run specific test** to debug:
   ```bash
   go test -v ./internal/repository -run TestName
   ```

3. **Use parallel execution** on multi-core systems:
   ```bash
   go test -parallel 8 ./...
   ```

## CI/CD Integration

### Pre-commit Hook

```bash
#!/bin/bash
go test -short ./...
if [ $? -ne 0 ]; then
    echo "Tests failed"
    exit 1
fi
```

### Build Pipeline

```bash
# Build commands
make build              # Compile service
make test              # Run all tests
make test-coverage     # Generate coverage report
make test-integration  # Run integration tests only
```

## Troubleshooting

### Tests Skipping

```
--- SKIP: TestSomething (0.00s)
    database.go:XX: Cannot connect to postgres for test setup: ...
```

**Solution**: Ensure PostgreSQL is running:
```bash
brew services start postgresql@16  # macOS
sudo systemctl start postgresql     # Linux
```

### Database Connection Errors

**Problem**: `dial tcp localhost:5432: connect: connection refused`

**Solution**: 
1. Check PostgreSQL is running
2. Verify credentials in environment variables
3. Check port is 5432

### Slow Tests

**Problem**: Tests taking too long

**Solution**:
1. Use `-short` flag for quick iteration
2. Run specific test module with `-run`
3. Increase timeout if system is slow: `-timeout 20m`

## Best Practices

1. **Always use fixtures** for test data creation
2. **Use TestDB for isolation** - never share test databases
3. **Include both positive and negative** test cases
4. **Test boundary conditions** (empty results, large datasets)
5. **Clean up resources** properly (defer Close())
6. **Use table-driven tests** for multiple scenarios
7. **Document complex test logic** with comments

## Next Steps

- [ ] Add handler/API tests with mock HTTP
- [ ] Add performance benchmarking
- [ ] Add contract testing with GraphQL client
- [ ] Add scenario/acceptance tests
- [ ] Add load testing suite
- [ ] Add database constraint testing
