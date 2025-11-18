# Testing Strategy

**Version**: 1.0.0
**Last Updated**: 2025-11-18

---

## Table of Contents

1. [Overview](#overview)
2. [Testing Pyramid](#testing-pyramid)
3. [Unit Testing](#unit-testing)
4. [Integration Testing](#integration-testing)
5. [End-to-End Testing](#end-to-end-testing)
6. [Load Testing](#load-testing)
7. [Security Testing](#security-testing)
8. [Test Coverage](#test-coverage)
9. [CI/CD Integration](#cicd-integration)

---

## 1. Overview

This document defines the comprehensive testing strategy for the WhatsApp Meta API Adapter.

### Testing Goals

- ✅ **80%+ code coverage** (unit tests)
- ✅ **100% critical path coverage** (integration tests)
- ✅ **99.9% uptime** (verified through load tests)
- ✅ **Zero security vulnerabilities** (OWASP compliance)
- ✅ **Sub-200ms p95 latency** (performance tests)

### Testing Tools

| Type | Tool | Purpose |
|------|------|---------|
| Unit | Go testing | Table-driven tests |
| Mocking | testify/mock | Interface mocking |
| Integration | Go testing + Docker | Database/API tests |
| E2E | Playwright | User journey tests |
| Load | k6 | Performance testing |
| Security | gosec, trivy | SAST, container scanning |
| API | Postman/Newman | API contract tests |

---

## 2. Testing Pyramid

```
       ┌──────────┐
       │   E2E    │ ← 10% (slow, brittle)
       │  Tests   │
       └──────────┘
      ┌────────────┐
      │ Integration │ ← 20% (moderate speed)
      │   Tests     │
      └─────────────┘
    ┌──────────────────┐
    │   Unit Tests     │ ← 70% (fast, reliable)
    └──────────────────┘
```

**Distribution**:
- **70%**: Unit tests (isolated, fast, many)
- **20%**: Integration tests (database, API, medium)
- **10%**: E2E tests (user flows, slow, few)

---

## 3. Unit Testing

### 3.1 Principles

- **FIRST**: Fast, Independent, Repeatable, Self-validating, Timely
- **Table-driven**: Test multiple scenarios with single test function
- **Mocking**: Use interfaces for dependencies
- **Coverage**: Minimum 80% per package

### 3.2 Test Structure

```go
// internal/service/user_service_test.go
package service

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "go.uber.org/zap"

    "github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/models"
    "github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/repository/mocks"
)

func TestUserService_CreateUser(t *testing.T) {
    tests := []struct {
        name      string
        input     *CreateUserInput
        setupMock func(*mocks.UserRepository)
        want      *models.User
        wantErr   error
    }{
        {
            name: "success - new user",
            input: &CreateUserInput{
                TenantID: "tenant-1",
                Email:    "test@example.com",
                Name:     "Test User",
            },
            setupMock: func(m *mocks.UserRepository) {
                m.On("FindByEmail", mock.Anything, "tenant-1", "test@example.com").
                    Return(nil, repository.ErrNotFound)
                m.On("Create", mock.Anything, mock.MatchedBy(func(u *models.User) bool {
                    return u.Email == "test@example.com" && u.TenantID == "tenant-1"
                })).Return(nil)
            },
            want: &models.User{
                Email:    "test@example.com",
                Name:     "Test User",
                TenantID: "tenant-1",
            },
            wantErr: nil,
        },
        {
            name: "error - user already exists",
            input: &CreateUserInput{
                TenantID: "tenant-1",
                Email:    "existing@example.com",
            },
            setupMock: func(m *mocks.UserRepository) {
                m.On("FindByEmail", mock.Anything, "tenant-1", "existing@example.com").
                    Return(&models.User{Email: "existing@example.com"}, nil)
            },
            wantErr: ErrUserExists,
        },
        {
            name: "error - invalid email",
            input: &CreateUserInput{
                TenantID: "tenant-1",
                Email:    "invalid-email",
            },
            setupMock: func(m *mocks.UserRepository) {},
            wantErr:   ErrInvalidInput,
        },
        {
            name: "error - database failure",
            input: &CreateUserInput{
                TenantID: "tenant-1",
                Email:    "test@example.com",
            },
            setupMock: func(m *mocks.UserRepository) {
                m.On("FindByEmail", mock.Anything, "tenant-1", "test@example.com").
                    Return(nil, repository.ErrNotFound)
                m.On("Create", mock.Anything, mock.Anything).
                    Return(errors.New("database connection failed"))
            },
            wantErr: errors.New("failed to create user"),
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Arrange
            mockRepo := &mocks.UserRepository{}
            if tt.setupMock != nil {
                tt.setupMock(mockRepo)
            }
            mockCache := &mocks.Cache{}
            logger := zap.NewNop()

            svc := NewUserService(mockRepo, mockCache, logger)

            // Act
            got, err := svc.CreateUser(context.Background(), tt.input)

            // Assert
            if tt.wantErr != nil {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.wantErr.Error())
                return
            }

            assert.NoError(t, err)
            assert.NotEmpty(t, got.ID)
            assert.Equal(t, tt.want.Email, got.Email)
            assert.Equal(t, tt.want.Name, got.Name)
            assert.Equal(t, tt.want.TenantID, got.TenantID)

            mockRepo.AssertExpectations(t)
        })
    }
}
```

### 3.3 Mock Generation

```go
// Generate mocks for interfaces
//go:generate mockery --name=UserRepository --output=./mocks --outpkg=mocks

type UserRepository interface {
    Create(ctx context.Context, user *models.User) error
    FindByID(ctx context.Context, id string) (*models.User, error)
    FindByEmail(ctx context.Context, tenantID, email string) (*models.User, error)
    Update(ctx context.Context, user *models.User) error
    Delete(ctx context.Context, id string) error
}
```

### 3.4 Running Unit Tests

```bash
# Run all tests
go test ./... -v

# Run with coverage
go test ./... -cover -coverprofile=coverage.out

# View coverage report
go tool cover -html=coverage.out

# Run specific package
go test ./internal/service -v

# Run specific test
go test ./internal/service -run TestUserService_CreateUser -v

# Parallel execution
go test ./... -parallel 4

# Short mode (skip long tests)
go test ./... -short
```

---

## 4. Integration Testing

### 4.1 Test Database Setup

```go
// tests/integration/setup_test.go
package integration

import (
    "context"
    "database/sql"
    "testing"

    "github.com/golang-migrate/migrate/v4"
    "github.com/golang-migrate/migrate/v4/database/postgres"
    _ "github.com/golang-migrate/migrate/v4/source/file"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/wait"
)

type TestDB struct {
    Container testcontainers.Container
    DB        *sql.DB
    DSN       string
}

func SetupTestDB(t *testing.T) *TestDB {
    ctx := context.Background()

    req := testcontainers.ContainerRequest{
        Image:        "postgres:15-alpine",
        ExposedPorts: []string{"5432/tcp"},
        Env: map[string]string{
            "POSTGRES_DB":       "test_db",
            "POSTGRES_USER":     "test_user",
            "POSTGRES_PASSWORD": "test_pass",
        },
        WaitingFor: wait.ForListeningPort("5432/tcp"),
    }

    container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: req,
        Started:          true,
    })
    if err != nil {
        t.Fatalf("failed to start container: %v", err)
    }

    host, _ := container.Host(ctx)
    port, _ := container.MappedPort(ctx, "5432")

    dsn := fmt.Sprintf("postgres://test_user:test_pass@%s:%s/test_db?sslmode=disable",
        host, port.Port())

    db, err := sql.Open("postgres", dsn)
    if err != nil {
        t.Fatalf("failed to connect to database: %v", err)
    }

    // Run migrations
    driver, _ := postgres.WithInstance(db, &postgres.Config{})
    m, _ := migrate.NewWithDatabaseInstance("file://../../migrations", "postgres", driver)
    m.Up()

    return &TestDB{
        Container: container,
        DB:        db,
        DSN:       dsn,
    }
}

func (tdb *TestDB) Cleanup(t *testing.T) {
    tdb.DB.Close()
    tdb.Container.Terminate(context.Background())
}
```

### 4.2 Integration Test Example

```go
// tests/integration/user_repository_test.go
package integration

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"

    "github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/models"
    "github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/repository"
)

func TestUserRepository_Create(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test in short mode")
    }

    testDB := SetupTestDB(t)
    defer testDB.Cleanup(t)

    repo := repository.NewUserRepository(testDB.DB)

    tests := []struct {
        name    string
        user    *models.User
        wantErr bool
    }{
        {
            name: "create valid user",
            user: &models.User{
                ID:       "user-1",
                TenantID: "tenant-1",
                Email:    "test@example.com",
                Name:     "Test User",
            },
            wantErr: false,
        },
        {
            name: "duplicate email",
            user: &models.User{
                ID:       "user-2",
                TenantID: "tenant-1",
                Email:    "test@example.com", // Same email
                Name:     "Another User",
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := repo.Create(context.Background(), tt.user)

            if tt.wantErr {
                assert.Error(t, err)
                return
            }

            assert.NoError(t, err)

            // Verify user was created
            found, err := repo.FindByID(context.Background(), tt.user.ID)
            assert.NoError(t, err)
            assert.Equal(t, tt.user.Email, found.Email)
        })
    }
}
```

### 4.3 API Integration Tests

```go
// tests/integration/api_test.go
package integration

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/stretchr/testify/assert"

    "github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/api"
)

func TestAPI_SendMessage(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    // Setup test server
    testDB := SetupTestDB(t)
    defer testDB.Cleanup(t)

    app := api.SetupApp(testDB.DSN)
    defer app.Shutdown()

    tests := []struct {
        name       string
        payload    map[string]interface{}
        wantStatus int
    }{
        {
            name: "send text message",
            payload: map[string]interface{}{
                "messaging_product": "whatsapp",
                "to":                "5511999999999",
                "type":              "text",
                "text": map[string]string{
                    "body": "Test message",
                },
            },
            wantStatus: 200,
        },
        {
            name: "invalid phone number",
            payload: map[string]interface{}{
                "messaging_product": "whatsapp",
                "to":                "invalid",
                "type":              "text",
            },
            wantStatus: 400,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            body, _ := json.Marshal(tt.payload)
            req := httptest.NewRequest("POST", "/v1/123/messages", bytes.NewReader(body))
            req.Header.Set("Content-Type", "application/json")
            req.Header.Set("Authorization", "Bearer test-token")

            resp, _ := app.Test(req)

            assert.Equal(t, tt.wantStatus, resp.StatusCode)
        })
    }
}
```

---

## 5. End-to-End Testing

### 5.1 Playwright Setup

```typescript
// tests/e2e/playwright.config.ts
import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './tests/e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: 'html',
  use: {
    baseURL: 'http://localhost:3000',
    trace: 'on-first-retry',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'firefox',
      use: { ...devices['Desktop Firefox'] },
    },
    {
      name: 'webkit',
      use: { ...devices['Desktop Safari'] },
    },
  ],
  webServer: {
    command: 'npm run dev',
    url: 'http://localhost:3000',
    reuseExistingServer: !process.env.CI,
  },
});
```

### 5.2 E2E Test Example

```typescript
// tests/e2e/instance-creation.spec.ts
import { test, expect } from '@playwright/test';

test.describe('Instance Management', () => {
  test.beforeEach(async ({ page }) => {
    // Login
    await page.goto('/login');
    await page.fill('input[name="email"]', 'test@example.com');
    await page.fill('input[name="password"]', 'password123');
    await page.click('button[type="submit"]');
    await expect(page).toHaveURL('/dashboard');
  });

  test('should create new instance', async ({ page }) => {
    // Navigate to instances
    await page.click('a[href="/instances"]');
    await expect(page).toHaveURL('/instances');

    // Click create button
    await page.click('button:text("New Instance")');

    // Fill form
    await page.fill('input[name="displayName"]', 'Test Instance');
    await page.fill('input[name="webhookUrl"]', 'https://example.com/webhook');

    // Submit
    await page.click('button:text("Create")');

    // Verify instance created
    await expect(page.locator('text=Test Instance')).toBeVisible();
  });

  test('should connect instance via QR code', async ({ page }) => {
    await page.goto('/instances/123');

    // Click connect button
    await page.click('button:text("Connect")');

    // QR code should appear
    await expect(page.locator('img[alt="QR Code"]')).toBeVisible({
      timeout: 10000,
    });

    // Verify instructions
    await expect(page.locator('text=Scan this QR code')).toBeVisible();
  });

  test('should send message', async ({ page }) => {
    await page.goto('/instances/123/messages');

    // Fill message form
    await page.fill('input[name="to"]', '+5511999999999');
    await page.fill('textarea[name="message"]', 'Hello from E2E test!');

    // Send
    await page.click('button:text("Send")');

    // Verify success
    await expect(page.locator('text=Message sent')).toBeVisible();

    // Verify message in history
    await expect(page.locator('text=Hello from E2E test!')).toBeVisible();
  });
});
```

---

## 6. Load Testing

### 6.1 k6 Configuration

```javascript
// tests/load/api-load-test.js
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

const errorRate = new Rate('errors');

export const options = {
  stages: [
    { duration: '2m', target: 100 },  // Ramp up to 100 users
    { duration: '5m', target: 100 },  // Stay at 100 users
    { duration: '2m', target: 200 },  // Ramp up to 200 users
    { duration: '5m', target: 200 },  // Stay at 200 users
    { duration: '2m', target: 0 },    // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<200', 'p(99)<500'], // 95% < 200ms, 99% < 500ms
    http_req_failed: ['rate<0.01'],                 // Error rate < 1%
    errors: ['rate<0.1'],                           // Custom error rate < 10%
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const ACCESS_TOKEN = __ENV.ACCESS_TOKEN;

export default function () {
  // Send message
  const payload = JSON.stringify({
    messaging_product: 'whatsapp',
    to: '5511999999999',
    type: 'text',
    text: {
      body: 'Load test message',
    },
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${ACCESS_TOKEN}`,
    },
  };

  const res = http.post(`${BASE_URL}/v1/123/messages`, payload, params);

  const success = check(res, {
    'status is 200': (r) => r.status === 200,
    'response time < 200ms': (r) => r.timings.duration < 200,
  });

  errorRate.add(!success);

  sleep(1);
}

export function handleSummary(data) {
  return {
    'summary.json': JSON.stringify(data),
    stdout: textSummary(data, { indent: ' ', enableColors: true }),
  };
}
```

### 6.2 Running Load Tests

```bash
# Install k6
brew install k6  # macOS
# or
wget https://github.com/grafana/k6/releases/download/v0.47.0/k6-v0.47.0-linux-amd64.tar.gz

# Run load test
k6 run tests/load/api-load-test.js

# Run with custom duration
k6 run --duration 10m --vus 100 tests/load/api-load-test.js

# Run with Prometheus remote write
k6 run --out experimental-prometheus-rw tests/load/api-load-test.js
```

---

## 7. Security Testing

### 7.1 Static Analysis (SAST)

```bash
# Go security scanner
gosec -fmt=json -out=gosec-report.json ./...

# Vulnerability check
govulncheck ./...

# License compliance
go-licenses check ./...
```

### 7.2 Container Scanning

```bash
# Trivy scan
trivy image whatsapp-adapter:latest

# Severity filter
trivy image --severity HIGH,CRITICAL whatsapp-adapter:latest

# Fail on vulnerabilities
trivy image --exit-code 1 --severity CRITICAL whatsapp-adapter:latest
```

### 7.3 Dynamic Analysis (DAST)

```bash
# OWASP ZAP baseline scan
docker run -t owasp/zap2docker-stable zap-baseline.py \
  -t https://api.whatsapp-adapter.example.com \
  -r zap-report.html

# Full scan (takes longer)
docker run -t owasp/zap2docker-stable zap-full-scan.py \
  -t https://api.whatsapp-adapter.example.com
```

---

## 8. Test Coverage

### 8.1 Coverage Targets

| Package | Target | Current |
|---------|--------|---------|
| `internal/service` | 90% | - |
| `internal/repository` | 85% | - |
| `internal/api` | 80% | - |
| `internal/auth` | 95% | - |
| `internal/whatsapp` | 75% | - |
| **Overall** | **80%** | - |

### 8.2 Coverage Report

```bash
# Generate coverage report
go test ./... -coverprofile=coverage.out -covermode=atomic

# View coverage by function
go tool cover -func=coverage.out

# HTML report
go tool cover -html=coverage.out -o coverage.html

# Upload to Codecov (CI)
bash <(curl -s https://codecov.io/bash) -f coverage.out
```

### 8.3 Enforcing Coverage

```bash
# Fail if coverage below 80%
go test ./... -cover | grep "coverage:" | awk '{if ($2 < 80.0) exit 1}'

# Or use go-test-coverage tool
go install github.com/vladopajic/go-test-coverage@latest
go-test-coverage --config=.testcoverage.yml
```

```yaml
# .testcoverage.yml
profile: coverage.out
local-prefix: github.com/tarcisoamorim/whatsmeow

threshold:
  file: 70
  package: 80
  total: 80

exclude:
  paths:
    - \.pb\.go$
    - mock_.*\.go$
    - .*_gen\.go$
  files:
    - cmd/server/main.go
```

---

## 9. CI/CD Integration

### 9.1 GitHub Actions Test Workflow

```yaml
# .github/workflows/test.yml
name: Tests

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4

    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.24'

    - name: Cache Go modules
      uses: actions/cache@v3
      with:
        path: ~/go/pkg/mod
        key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}

    - name: Install dependencies
      run: go mod download

    - name: Run unit tests
      run: go test ./... -v -race -coverprofile=coverage.out -covermode=atomic

    - name: Upload coverage to Codecov
      uses: codecov/codecov-action@v3
      with:
        file: ./coverage.out

  integration-tests:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_DB: test_db
          POSTGRES_USER: test_user
          POSTGRES_PASSWORD: test_pass
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
    - uses: actions/checkout@v4

    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.24'

    - name: Run integration tests
      env:
        DATABASE_URL: postgres://test_user:test_pass@localhost:5432/test_db?sslmode=disable
      run: go test ./tests/integration/... -v

  e2e-tests:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4

    - name: Set up Node.js
      uses: actions/setup-node@v3
      with:
        node-version: '20'

    - name: Install dependencies
      run: |
        cd web
        npm ci

    - name: Install Playwright
      run: npx playwright install --with-deps

    - name: Run E2E tests
      run: npx playwright test

    - name: Upload Playwright report
      if: always()
      uses: actions/upload-artifact@v3
      with:
        name: playwright-report
        path: playwright-report/

  security-scan:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4

    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.24'

    - name: Run Gosec
      run: |
        go install github.com/securego/gosec/v2/cmd/gosec@latest
        gosec -fmt sarif -out gosec-results.sarif ./...

    - name: Run govulncheck
      run: |
        go install golang.org/x/vuln/cmd/govulncheck@latest
        govulncheck ./...

    - name: Upload SARIF file
      uses: github/codeql-action/upload-sarif@v2
      with:
        sarif_file: gosec-results.sarif
```

### 9.2 Pre-commit Hooks

```bash
# .git/hooks/pre-commit
#!/bin/bash

echo "Running pre-commit checks..."

# Run tests
echo "Running unit tests..."
go test ./... -short || exit 1

# Run linter
echo "Running golangci-lint..."
golangci-lint run || exit 1

# Check formatting
echo "Checking formatting..."
if [ "$(gofmt -l . | wc -l)" -gt 0 ]; then
  echo "Code is not formatted. Run 'gofmt -w .'"
  exit 1
fi

# Security scan
echo "Running security scan..."
gosec -quiet ./... || exit 1

echo "All checks passed!"
```

---

## Appendix A: Test Data Fixtures

```go
// tests/fixtures/users.go
package fixtures

import "github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/models"

var (
    TestUser1 = &models.User{
        ID:       "user-1",
        TenantID: "tenant-1",
        Email:    "test1@example.com",
        Name:     "Test User 1",
    }

    TestUser2 = &models.User{
        ID:       "user-2",
        TenantID: "tenant-1",
        Email:    "test2@example.com",
        Name:     "Test User 2",
    }

    TestTenant1 = &models.Tenant{
        ID:    "tenant-1",
        Name:  "Test Tenant",
        Email: "tenant@example.com",
        Tier:  "business",
    }
)
```

---

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0.0 | 2025-11-18 | Initial testing strategy |
