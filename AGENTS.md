# Development Agents Guide

**Project**: WhatsApp Meta API Adapter
**Version**: 1.0.0
**Last Updated**: 2025-11-18

---

## Overview

This document defines the specialized AI agents and their roles in maintaining and evolving this codebase. It ensures consistency, quality, and adherence to established patterns across development sessions.

---

## Core Development Principles

### 1. Code Quality Standards

**Senior-Level Requirements**:
- ✅ **No technical debt**: Refactor immediately, don't accumulate shortcuts
- ✅ **Production-ready from day one**: Every commit should be deployable
- ✅ **Comprehensive error handling**: Never panic, always return errors
- ✅ **Logging and observability**: Structured logging at every layer
- ✅ **Security-first**: OWASP, STRIDE, defense in depth
- ✅ **Performance-conscious**: Benchmark critical paths, optimize early

### 2. Architecture Patterns (Non-Negotiable)

```
meta-adapter/
├── cmd/                    # Application entry points
│   ├── server/            # Main API server
│   ├── worker/            # Background workers
│   └── migrate/           # Database migrations CLI
├── internal/              # Private application code
│   ├── api/              # HTTP handlers (Fiber)
│   ├── auth/             # OAuth2 + JWT
│   ├── models/           # Domain models
│   ├── repository/       # Data access layer
│   ├── service/          # Business logic
│   ├── whatsapp/         # WhatsApp client wrapper
│   ├── webhook/          # Webhook dispatcher
│   └── middleware/       # HTTP middleware
├── pkg/                   # Public libraries
│   ├── crypto/           # Encryption utilities
│   ├── logger/           # Structured logging
│   └── validator/        # Input validation
├── migrations/            # SQL migration files
├── web/                   # Frontend (React/Vue)
│   ├── src/
│   ├── public/
│   └── package.json
├── deployments/           # Infrastructure as Code
│   ├── docker/
│   ├── kubernetes/
│   └── terraform/
├── docs/                  # BMAD documentation
├── tests/                 # Test suites
│   ├── unit/
│   ├── integration/
│   └── e2e/
└── scripts/               # Automation scripts
```

### 3. Dependency Injection Pattern

**ALWAYS use DI for testability**:

```go
// ❌ BAD: Hard-coded dependencies
func NewUserService() *UserService {
    return &UserService{
        repo: postgres.NewUserRepository(),
        cache: redis.NewClient(),
    }
}

// ✅ GOOD: Injected dependencies
type UserService struct {
    repo  repository.UserRepository
    cache cache.Cache
    log   logger.Logger
}

func NewUserService(repo repository.UserRepository, cache cache.Cache, log logger.Logger) *UserService {
    return &UserService{repo: repo, cache: cache, log: log}
}
```

### 4. Error Handling Standard

```go
// ✅ ALWAYS return custom error types with context
type AppError struct {
    Code    string
    Message string
    Err     error
    Meta    map[string]interface{}
}

func (e *AppError) Error() string {
    if e.Err != nil {
        return fmt.Sprintf("%s: %v", e.Message, e.Err)
    }
    return e.Message
}

// Usage
func (s *UserService) GetUser(ctx context.Context, id string) (*models.User, error) {
    user, err := s.repo.FindByID(ctx, id)
    if err != nil {
        return nil, &AppError{
            Code:    "USER_NOT_FOUND",
            Message: "failed to retrieve user",
            Err:     err,
            Meta:    map[string]interface{}{"user_id": id},
        }
    }
    return user, nil
}
```

### 5. Logging Standard

```go
// ✅ Use structured logging with context
import "go.uber.org/zap"

log.Info("user created",
    zap.String("user_id", user.ID),
    zap.String("email", user.Email),
    zap.Duration("duration", time.Since(start)),
)

log.Error("database query failed",
    zap.Error(err),
    zap.String("query", query),
    zap.String("tenant_id", tenantID),
)
```

---

## Agent Roles & Responsibilities

### Agent 1: Code Reviewer

**Trigger**: After significant code changes
**Checklist**:
- [ ] Follows architecture patterns (layered, DI, repository)
- [ ] No hardcoded secrets or credentials
- [ ] Error handling comprehensive
- [ ] Logging at appropriate levels
- [ ] No SQL injection vulnerabilities
- [ ] Input validation on all user inputs
- [ ] Tests written for new code
- [ ] Documentation updated
- [ ] No commented-out code
- [ ] Consistent naming conventions

### Agent 2: Security Auditor

**Trigger**: Before merging to main
**Focus Areas**:
- [ ] OWASP Top 10 vulnerabilities
- [ ] SQL injection (use parameterized queries)
- [ ] XSS (sanitize outputs)
- [ ] CSRF (tokens on state-changing operations)
- [ ] Authentication bypass attempts
- [ ] Authorization checks on all endpoints
- [ ] Sensitive data encryption (at rest + transit)
- [ ] Rate limiting on public endpoints
- [ ] Webhook signature verification
- [ ] Session management security

**Tools**:
```bash
# Run security scanners
gosec ./...
govulncheck ./...
trivy fs .
```

### Agent 3: Performance Optimizer

**Trigger**: Before production deployment
**Benchmarks**:
- [ ] Database queries use indexes (EXPLAIN ANALYZE)
- [ ] N+1 queries eliminated
- [ ] Caching strategy implemented (Redis)
- [ ] Connection pooling configured
- [ ] API response times < 200ms (p95)
- [ ] Memory allocations minimized
- [ ] Goroutine leaks prevented

**Tools**:
```bash
# Profile application
go test -bench=. -benchmem -cpuprofile=cpu.prof
go tool pprof cpu.prof

# Load testing
k6 run tests/load/api_test.js
```

### Agent 4: Test Engineer

**Trigger**: Continuous (TDD)
**Coverage Targets**:
- Unit tests: **80%+**
- Integration tests: **Critical paths 100%**
- E2E tests: **User journeys 100%**

**Patterns**:

```go
// Table-driven tests
func TestUserService_CreateUser(t *testing.T) {
    tests := []struct {
        name    string
        input   *models.User
        want    *models.User
        wantErr bool
    }{
        {
            name:  "valid user",
            input: &models.User{Email: "test@example.com"},
            want:  &models.User{ID: "uuid", Email: "test@example.com"},
        },
        {
            name:    "duplicate email",
            input:   &models.User{Email: "duplicate@example.com"},
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Agent 5: DevOps Engineer

**Trigger**: Infrastructure changes
**Responsibilities**:
- [ ] Dockerfile optimized (multi-stage builds)
- [ ] Kubernetes manifests validated
- [ ] Health checks configured
- [ ] Resource limits set
- [ ] Secrets management (not in code)
- [ ] CI/CD pipeline functional
- [ ] Monitoring dashboards created
- [ ] Alerts configured

---

## Development Workflow

### 1. Feature Development

```bash
# 1. Create feature branch
git checkout -b feature/oauth2-implementation

# 2. Write tests first (TDD)
# tests/unit/auth/oauth2_test.go

# 3. Implement feature
# internal/auth/oauth2.go

# 4. Run tests
go test ./... -v -cover

# 5. Run linters
golangci-lint run

# 6. Security scan
gosec ./...

# 7. Commit with conventional commits
git commit -m "feat(auth): implement OAuth2 authorization code flow

- Add authorization endpoint with PKCE support
- Implement token exchange
- Add refresh token rotation
- Include comprehensive tests

Closes #123"

# 8. Push and create PR
git push -u origin feature/oauth2-implementation
```

### 2. Code Review Checklist

**Reviewer must verify**:
```markdown
- [ ] Code follows architecture patterns
- [ ] Tests pass (unit + integration)
- [ ] Coverage >= 80%
- [ ] No security vulnerabilities
- [ ] Documentation updated
- [ ] CHANGELOG.md updated
- [ ] Migration scripts included (if DB changes)
- [ ] Breaking changes noted
```

### 3. Commit Message Convention

**Format**: `<type>(<scope>): <subject>`

**Types**:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation only
- `style`: Formatting, no code change
- `refactor`: Code restructuring
- `perf`: Performance improvement
- `test`: Adding tests
- `chore`: Maintenance tasks
- `ci`: CI/CD changes

**Examples**:
```
feat(auth): add OAuth2 PKCE support
fix(webhook): prevent retry loop on permanent failures
docs(api): update OpenAPI spec with new endpoints
perf(db): add composite index on (tenant_id, created_at)
```

---

## Technology Stack Decisions

### Backend
- **Language**: Go 1.24+
- **Framework**: Fiber v2 (FastHTTP)
- **ORM**: None (raw SQL + sqlx for scanning)
- **Migration**: golang-migrate
- **Auth**: OAuth2 (RFC 6749) + JWT
- **Validation**: go-playground/validator
- **Logging**: zap (structured)
- **Config**: viper + environment variables

### Database
- **Primary**: PostgreSQL 15+
- **Cache**: Redis 7+
- **Queue**: RabbitMQ 3.12+
- **Search**: PostgreSQL Full-Text Search

### Frontend
- **Framework**: React 18+ (TypeScript)
- **State**: Zustand or Jotai
- **UI**: Tailwind CSS + shadcn/ui
- **Forms**: React Hook Form + Zod
- **API Client**: TanStack Query (React Query)
- **Build**: Vite

### Infrastructure
- **Container**: Docker
- **Orchestration**: Kubernetes
- **CI/CD**: GitHub Actions
- **Monitoring**: Prometheus + Grafana
- **Logging**: Loki + Promtail
- **Tracing**: OpenTelemetry + Jaeger
- **Cloud**: AWS (primary), GCP (alternative)

---

## Performance Targets

| Metric | Target | Measurement |
|--------|--------|-------------|
| API Response Time (p95) | < 200ms | Prometheus |
| API Response Time (p99) | < 500ms | Prometheus |
| Database Query Time | < 50ms | pg_stat_statements |
| Webhook Delivery | < 2s | Application logs |
| Instance Connection | < 10s | Application metrics |
| Uptime | 99.9% | External monitoring |
| Error Rate | < 0.1% | Application logs |

---

## Security Requirements

### Authentication
- ✅ OAuth2 with PKCE (prevent code interception)
- ✅ JWT with short expiry (1 hour)
- ✅ Refresh token rotation
- ✅ 2FA support (TOTP)

### Authorization
- ✅ RBAC with scopes
- ✅ Tenant isolation at database level
- ✅ Row-level security (RLS) in PostgreSQL

### Data Protection
- ✅ Encryption at rest (AES-256)
- ✅ Encryption in transit (TLS 1.3)
- ✅ Secrets in vault (not environment)
- ✅ PII data minimization

### Compliance
- ✅ GDPR (EU)
- ✅ LGPD (Brazil)
- ✅ Data retention policies
- ✅ Right to deletion

---

## Testing Strategy

### Unit Tests (80%+ coverage)
```go
// internal/service/user_service_test.go
func TestUserService_CreateUser(t *testing.T) {
    // Arrange
    repo := &mocks.UserRepository{}
    cache := &mocks.Cache{}
    svc := NewUserService(repo, cache, logger.NewNop())

    // Act
    user, err := svc.CreateUser(context.Background(), &CreateUserInput{
        Email: "test@example.com",
    })

    // Assert
    assert.NoError(t, err)
    assert.NotEmpty(t, user.ID)
}
```

### Integration Tests
```go
// tests/integration/api_test.go
func TestAPI_SendMessage(t *testing.T) {
    // Start test server with real database
    server := setupTestServer(t)
    defer server.Cleanup()

    // Send request
    resp := server.POST("/v1/123/messages", map[string]interface{}{
        "to": "5511999999999",
        "type": "text",
        "text": map[string]string{"body": "Test"},
    })

    assert.Equal(t, 200, resp.StatusCode)
}
```

### E2E Tests (Playwright/Cypress)
```typescript
// tests/e2e/instance-creation.spec.ts
test('should create instance and connect via QR code', async ({ page }) => {
  await page.goto('/instances');
  await page.click('button:text("New Instance")');
  await page.fill('input[name="name"]', 'Test Instance');
  await page.click('button:text("Create")');

  // Wait for QR code
  await expect(page.locator('img[alt="QR Code"]')).toBeVisible();
});
```

---

## Monitoring & Observability

### Metrics (Prometheus)
```go
// Instrument HTTP handlers
httpDuration := promauto.NewHistogramVec(prometheus.HistogramOpts{
    Name: "http_request_duration_seconds",
    Help: "HTTP request latency",
}, []string{"method", "path", "status"})

// Instrument business logic
messagesProcessed := promauto.NewCounterVec(prometheus.CounterOpts{
    Name: "messages_processed_total",
    Help: "Total messages processed",
}, []string{"direction", "type", "status"})
```

### Logging (Structured)
```go
log.Info("message sent",
    zap.String("tenant_id", tenantID),
    zap.String("instance_id", instanceID),
    zap.String("message_id", messageID),
    zap.String("to", to),
    zap.Duration("duration", time.Since(start)),
)
```

### Tracing (OpenTelemetry)
```go
ctx, span := tracer.Start(ctx, "UserService.CreateUser")
defer span.End()

span.SetAttributes(
    attribute.String("user.email", email),
    attribute.String("tenant.id", tenantID),
)
```

---

## Documentation Standards

### Code Comments
```go
// ❌ BAD: Obvious comments
// Get user by ID
func GetUser(id string) (*User, error) {}

// ✅ GOOD: Why, not what
// GetUser retrieves a user by ID from cache if available,
// falling back to database if cache miss. Returns ErrUserNotFound
// if user doesn't exist.
func GetUser(ctx context.Context, id string) (*User, error) {}
```

### API Documentation
- OpenAPI 3.0 spec auto-generated from code
- Examples for all endpoints
- Error codes documented

### Database Documentation
- Schema documented in DATABASE-SCHEMA.md
- Migration files include comments
- ER diagrams kept up to date

---

## Emergency Procedures

### Production Incident
1. **Acknowledge**: Post in #incidents Slack channel
2. **Assess**: Check monitoring dashboards
3. **Mitigate**: Rollback if recent deployment
4. **Communicate**: Update status page
5. **Resolve**: Fix root cause
6. **Postmortem**: Write incident report (no blame)

### Security Breach
1. **Isolate**: Disable affected services
2. **Notify**: Security team + legal
3. **Investigate**: Review audit logs
4. **Remediate**: Patch vulnerability
5. **Disclose**: Follow responsible disclosure timeline

---

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0.0 | 2025-11-18 | Initial agent guide |

---

**Maintained by**: WhatsApp Meta API Adapter Team
**Review Cycle**: Quarterly
