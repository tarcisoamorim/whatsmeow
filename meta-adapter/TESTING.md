# Testing Guide

## Overview

This project follows production-grade testing practices with comprehensive test coverage across all layers:

- **Unit Tests**: Repository, JWT, Webhook components
- **Integration Tests**: API handlers with HTTP testing
- **Target Coverage**: 80%+ code coverage

## Prerequisites

### Test Database

Tests require a PostgreSQL test database. Set up using Docker:

```bash
docker run -d \
  --name whatsmeow-test-db \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=whatsmeow_test \
  -p 5432:5432 \
  postgres:15-alpine
```

### Run Migrations

```bash
# Set test database URL
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/whatsmeow_test?sslmode=disable"

# Run migrations
migrate -path migrations -database "$DATABASE_URL" up
```

## Running Tests

### Run All Tests

```bash
cd meta-adapter
go test ./... -v
```

### Run Tests with Coverage

```bash
go test ./... -v -coverprofile=coverage.out -covermode=atomic

# View coverage report
go tool cover -html=coverage.out
```

### Run Specific Package Tests

```bash
# Repository tests
go test ./internal/repository -v

# API handler tests
go test ./internal/api -v

# Auth tests
go test ./internal/auth -v

# Webhook tests
go test ./internal/webhook -v
```

## Test Structure

### Repository Tests (`internal/repository/*_test.go`)

Tests database operations with real PostgreSQL connections:

- **InstanceRepository**: CRUD, tenant isolation, status updates, QR code storage
- **MessageRepository**: CRUD, status transitions, pagination, JSONB content storage
- **TenantRepository**: Basic operations

**Key Features Tested**:
- Multi-tenant data isolation via partitioning
- Concurrent operations
- Transaction handling
- JSONB content storage

### API Handler Tests (`internal/api/*_test.go`)

Tests HTTP endpoints using Fiber's httptest:

- **InstanceHandler**: List, Create (with quota validation), Get, GetQRCode
- **MessageHandler**: Send (multiple types), List (with pagination)

**Key Features Tested**:
- Request validation
- Meta API compatible responses
- Error handling (404, 400, 403, 500)
- Tenant isolation at API layer
- Pagination (limit, offset, max caps)

### Auth Tests (`internal/auth/*_test.go`)

Tests JWT token generation and verification:

- Token generation with custom claims
- Token verification
- Expiration handling
- Scope validation
- Secret isolation

### Webhook Tests (`internal/webhook/*_test.go`)

Tests HMAC signature generation and verification:

- Payload signing with SHA256
- Signature verification
- Tampering detection
- Different secrets isolation

## Test Database Cleanup

After testing, clean up test data:

```bash
# Drop test database
docker exec -it whatsmeow-test-db psql -U postgres -c "DROP DATABASE whatsmeow_test;"
docker exec -it whatsmeow-test-db psql -U postgres -c "CREATE DATABASE whatsmeow_test;"

# Or restart container
docker restart whatsmeow-test-db
```

## Continuous Integration

For CI environments, use the following workflow:

```yaml
# Example GitHub Actions workflow
- name: Run tests
  env:
    DATABASE_URL: postgres://postgres:postgres@localhost:5432/whatsmeow_test?sslmode=disable
  run: |
    go test ./... -v -coverprofile=coverage.out -covermode=atomic

- name: Upload coverage
  uses: codecov/codecov-action@v3
  with:
    files: ./coverage.out
```

## Test Data Helpers

Tests use helper functions for setup:

```go
// Setup test database
db := setupTestDB(t)
defer db.Close()

// Setup test tenant
tenant := setupTestTenant(t, db)

// Setup test WhatsApp manager (mock for now)
waManager := setupTestWAManager(t, db)
```

**Note**: Tests will be skipped if the test database is not available, allowing development without PostgreSQL running locally.

## Performance Testing

For load testing the API:

```bash
# Install vegeta
go install github.com/tsenart/vegeta@latest

# Test instances endpoint
echo "GET http://localhost:8080/v1/instances" | \
  vegeta attack -duration=30s -rate=50 -header "Authorization: Bearer YOUR_TOKEN" | \
  vegeta report
```

## Benchmarking

Run benchmarks for performance-critical code:

```bash
# Run all benchmarks
go test ./... -bench=. -benchmem

# Run specific benchmark
go test ./internal/repository -bench=BenchmarkInstanceCreate -benchmem
```

## Test Coverage Goals

| Package | Target Coverage | Current Status |
|---------|----------------|----------------|
| `internal/repository` | 85% | ✅ Comprehensive tests |
| `internal/api` | 80% | ✅ Comprehensive tests |
| `internal/auth` | 90% | ✅ Comprehensive tests |
| `internal/webhook` | 85% | ✅ Comprehensive tests |
| `internal/middleware` | 75% | ⚠️  Needs tests |
| `internal/whatsapp` | 70% | ⚠️  Needs tests |
| **Overall** | **80%** | 🎯 On track |

## Known Limitations

1. **WhatsApp Manager Mock**: Tests currently skip WhatsApp integration. Future: implement proper mocks using interfaces.
2. **RabbitMQ Tests**: Webhook deliverer tests need RabbitMQ integration (use testcontainers).
3. **Redis Tests**: Rate limiter tests need Redis integration (use testcontainers).

## Next Steps

1. ✅ Repository tests
2. ✅ API handler tests
3. ✅ Auth tests
4. ✅ Webhook signer tests
5. ⏳ Middleware tests (rate limit, tenant enrichment)
6. ⏳ Integration tests with Testcontainers
7. ⏳ E2E tests with full stack

## Troubleshooting

### Tests Skip with "database not available"

Ensure PostgreSQL is running:
```bash
docker ps | grep whatsmeow-test-db
```

### Migration Errors

Reset migrations:
```bash
migrate -path migrations -database "$DATABASE_URL" down
migrate -path migrations -database "$DATABASE_URL" up
```

### Connection Refused

Check database connection:
```bash
psql "postgres://postgres:postgres@localhost:5432/whatsmeow_test" -c "SELECT 1;"
```

## References

- [Testing in Go](https://go.dev/doc/tutorial/add-a-test)
- [Table-Driven Tests](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
- [Testcontainers](https://testcontainers.com/modules/postgresql/)
- [Fiber Testing](https://docs.gofiber.io/api/app#test)
