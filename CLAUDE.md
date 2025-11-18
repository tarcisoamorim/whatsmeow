# Claude AI Development Guide

**Project**: WhatsApp Meta API Adapter
**Version**: 1.0.0
**Last Updated**: 2025-11-18

---

## Purpose

This document provides instructions for Claude AI (or any AI coding assistant) working on this codebase. It ensures consistency, quality, and adherence to project standards across all AI-assisted development sessions.

---

## Core Directives

### 1. **ALWAYS Read Before Acting**

Before making ANY code changes:
1. ✅ Read `AGENTS.md` - Understand development standards
2. ✅ Read `docs/PRD.md` - Understand product requirements
3. ✅ Read `docs/ARCHITECTURE.md` - Understand system design
4. ✅ Read `docs/DATABASE-SCHEMA.md` - Understand data model
5. ✅ Read `docs/API-SPEC.md` - Understand API contracts
6. ✅ Read relevant source files - Understand existing patterns

### 2. **Think Senior, Code Senior**

Every decision should reflect **senior-level engineering**:
- ⚡ Performance-conscious (benchmark, optimize)
- 🔒 Security-first (OWASP, STRIDE, zero-trust)
- 🧪 Test-driven (write tests first when possible)
- 📝 Documentation-complete (code + docs updated together)
- 🏗️ Architecture-aware (follow established patterns)
- 🔮 Future-proof (extensible, maintainable)

### 3. **Production-Ready from Day One**

Every commit must be production-grade:
- ✅ No TODO comments (fix now or create issue)
- ✅ No hardcoded values (use config)
- ✅ No placeholder implementations
- ✅ No skipped error handling
- ✅ No missing tests
- ✅ No security vulnerabilities

---

## ULTRATHINK Protocol

When user includes `**ULTRATHINK**` in their message:

### Mandatory Deep Analysis Steps

1. **Context Gathering** (5 minutes)
   - Read all relevant documentation
   - Review existing code patterns
   - Identify dependencies and constraints
   - Check for similar implementations

2. **Problem Decomposition** (5 minutes)
   - Break down request into atomic tasks
   - Identify critical path and dependencies
   - Estimate complexity and time
   - Flag potential blockers

3. **Architecture Planning** (10 minutes)
   - Design solution following established patterns
   - Consider scalability and performance
   - Identify security implications
   - Plan testing strategy

4. **Implementation Strategy** (5 minutes)
   - Order tasks by dependency
   - Plan commit boundaries
   - Identify reusable components
   - Prepare rollback strategy

5. **Quality Checklist**
   - [ ] Follows architecture patterns
   - [ ] Security reviewed (OWASP)
   - [ ] Performance acceptable
   - [ ] Tests comprehensive
   - [ ] Documentation complete
   - [ ] Error handling robust
   - [ ] Logging structured
   - [ ] Monitoring instrumented

### Output Format

```markdown
# ULTRATHINK Analysis: [Feature Name]

## 1. Understanding
[What I understood from the request]

## 2. Architecture Impact
[How this affects existing architecture]

## 3. Implementation Plan
- [ ] Task 1
- [ ] Task 2
- [ ] Task 3

## 4. Security Considerations
[Potential vulnerabilities and mitigations]

## 5. Testing Strategy
[Unit, integration, e2e tests needed]

## 6. Estimated Complexity
[Simple/Medium/Complex + time estimate]

## 7. Risks & Mitigations
[What could go wrong and how to prevent]
```

---

## Development Patterns

### Pattern 1: Feature Implementation

```markdown
**Step-by-step for new features:**

1. Create feature branch
2. Write tests first (TDD)
3. Implement minimal viable version
4. Add comprehensive error handling
5. Add logging and metrics
6. Write documentation
7. Run security scan
8. Create migration (if DB changes)
9. Update CHANGELOG.md
10. Commit with conventional commit message
```

### Pattern 2: Bug Fix

```markdown
**Step-by-step for bugs:**

1. Reproduce bug with test
2. Identify root cause (don't just fix symptoms)
3. Fix minimal code necessary
4. Verify test now passes
5. Check for similar bugs elsewhere
6. Add regression test
7. Update documentation if behavior changed
8. Commit with fix: prefix
```

### Pattern 3: Refactoring

```markdown
**Step-by-step for refactoring:**

1. Ensure tests exist and pass
2. Make small, incremental changes
3. Run tests after each change
4. Keep commits atomic (one change per commit)
5. Don't mix refactoring with feature work
6. Update documentation
7. Commit with refactor: prefix
```

---

## Code Quality Standards

### Go Code Style

```go
// ✅ GOOD: Follows Go conventions
package service

import (
    "context"
    "fmt"

    "github.com/google/uuid"
    "go.uber.org/zap"

    "github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/models"
    "github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/repository"
)

// UserService handles user-related business logic.
// It provides methods for user CRUD operations with
// tenant isolation and caching.
type UserService struct {
    repo  repository.UserRepository
    cache cache.Cache
    log   *zap.Logger
}

// NewUserService creates a new UserService with injected dependencies.
func NewUserService(repo repository.UserRepository, cache cache.Cache, log *zap.Logger) *UserService {
    return &UserService{
        repo:  repo,
        cache: cache,
        log:   log,
    }
}

// CreateUser creates a new user with the given input.
// It validates input, checks for duplicates, and caches the result.
// Returns ErrUserExists if email already registered.
func (s *UserService) CreateUser(ctx context.Context, input *CreateUserInput) (*models.User, error) {
    // Input validation
    if err := input.Validate(); err != nil {
        return nil, fmt.Errorf("invalid input: %w", err)
    }

    // Check for existing user
    existing, err := s.repo.FindByEmail(ctx, input.TenantID, input.Email)
    if err != nil && !errors.Is(err, repository.ErrNotFound) {
        s.log.Error("failed to check existing user",
            zap.Error(err),
            zap.String("email", input.Email),
        )
        return nil, fmt.Errorf("failed to check existing user: %w", err)
    }

    if existing != nil {
        return nil, ErrUserExists
    }

    // Create user
    user := &models.User{
        ID:       uuid.New().String(),
        TenantID: input.TenantID,
        Email:    input.Email,
        Name:     input.Name,
    }

    if err := s.repo.Create(ctx, user); err != nil {
        s.log.Error("failed to create user",
            zap.Error(err),
            zap.String("email", input.Email),
        )
        return nil, fmt.Errorf("failed to create user: %w", err)
    }

    // Cache user
    if err := s.cache.Set(ctx, fmt.Sprintf("user:%s", user.ID), user, time.Hour); err != nil {
        s.log.Warn("failed to cache user",
            zap.Error(err),
            zap.String("user_id", user.ID),
        )
        // Don't fail on cache error
    }

    s.log.Info("user created",
        zap.String("user_id", user.ID),
        zap.String("email", user.Email),
    )

    return user, nil
}
```

### SQL Query Style

```sql
-- ✅ GOOD: Readable, parameterized, indexed
SELECT
    u.id,
    u.tenant_id,
    u.email,
    u.name,
    u.created_at,
    u.updated_at
FROM users u
WHERE u.tenant_id = $1
  AND u.email = $2
  AND u.deleted_at IS NULL
ORDER BY u.created_at DESC
LIMIT $3 OFFSET $4;

-- Index to support this query
CREATE INDEX idx_users_tenant_email ON users(tenant_id, email) WHERE deleted_at IS NULL;
```

### TypeScript/React Style

```typescript
// ✅ GOOD: TypeScript, hooks, error handling
import { useState, useEffect } from 'react';
import { useQuery, useMutation } from '@tanstack/react-query';
import { z } from 'zod';

// Schema validation
const createInstanceSchema = z.object({
  displayName: z.string().min(1).max(255),
  webhookUrl: z.string().url().optional(),
});

type CreateInstanceInput = z.infer<typeof createInstanceSchema>;

export function CreateInstanceForm() {
  const [error, setError] = useState<string | null>(null);

  const createInstance = useMutation({
    mutationFn: async (input: CreateInstanceInput) => {
      const validated = createInstanceSchema.parse(input);
      const response = await fetch('/v1/instances', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(validated),
      });

      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.message);
      }

      return response.json();
    },
    onSuccess: () => {
      // Invalidate and refetch
      queryClient.invalidateQueries({ queryKey: ['instances'] });
    },
    onError: (err) => {
      setError(err.message);
    },
  });

  return (
    <form onSubmit={(e) => {
      e.preventDefault();
      const formData = new FormData(e.currentTarget);
      createInstance.mutate({
        displayName: formData.get('displayName') as string,
        webhookUrl: formData.get('webhookUrl') as string || undefined,
      });
    }}>
      {/* Form fields */}
    </form>
  );
}
```

---

## Commit Strategy

### Granular Commits (MANDATORY)

**DO**: Make small, focused commits
```bash
git commit -m "feat(auth): add OAuth2 authorization endpoint"
git commit -m "feat(auth): add token exchange endpoint"
git commit -m "feat(auth): add refresh token rotation"
git commit -m "test(auth): add OAuth2 integration tests"
git commit -m "docs(auth): update API spec with OAuth2 endpoints"
```

**DON'T**: Make giant commits
```bash
# ❌ BAD
git commit -m "feat: add OAuth2 and webhooks and rate limiting"
```

### Commit After Each Logical Unit

Commit boundaries:
- ✅ After completing a single function/method
- ✅ After adding a test suite for a component
- ✅ After creating a migration file
- ✅ After implementing an API endpoint
- ✅ After adding configuration
- ✅ After updating documentation

### Conventional Commit Format

```
<type>(<scope>): <subject>

[optional body]

[optional footer]
```

**Types**:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation
- `style`: Formatting
- `refactor`: Code restructuring
- `perf`: Performance
- `test`: Tests
- `chore`: Maintenance
- `ci`: CI/CD

**Scopes** (use these):
- `auth`: Authentication/authorization
- `api`: API endpoints
- `db`: Database
- `whatsapp`: WhatsApp integration
- `webhook`: Webhook system
- `instance`: Instance management
- `message`: Message handling
- `media`: Media storage
- `admin`: Admin dashboard
- `deploy`: Deployment
- `config`: Configuration

**Examples**:
```
feat(auth): implement OAuth2 PKCE flow

- Add authorization endpoint with state validation
- Add token exchange with code verifier
- Add refresh token rotation for security
- Include comprehensive tests

Closes #42

---

fix(webhook): prevent infinite retry loop

Webhooks were retrying indefinitely on 4xx errors.
Now only retries on 5xx (server errors) and network failures.

Fixes #108

---

docs(api): add rate limiting section to API spec

Documents rate limits by tier and response headers.
Includes examples of 429 error responses.
```

---

## Error Handling Standards

### Go Errors

```go
// ✅ Define custom error types
var (
    ErrUserNotFound     = errors.New("user not found")
    ErrUserExists       = errors.New("user already exists")
    ErrInvalidInput     = errors.New("invalid input")
    ErrUnauthorized     = errors.New("unauthorized")
    ErrRateLimitExceeded = errors.New("rate limit exceeded")
)

// ✅ Wrap errors with context
func (s *UserService) GetUser(ctx context.Context, id string) (*models.User, error) {
    user, err := s.repo.FindByID(ctx, id)
    if err != nil {
        if errors.Is(err, repository.ErrNotFound) {
            return nil, ErrUserNotFound
        }
        return nil, fmt.Errorf("failed to get user: %w", err)
    }
    return user, nil
}

// ✅ HTTP error mapping
func mapErrorToHTTP(err error) (int, *ErrorResponse) {
    switch {
    case errors.Is(err, ErrUserNotFound):
        return 404, &ErrorResponse{
            Error: ErrorDetail{
                Message: "User not found",
                Type:    "NotFoundError",
                Code:    404,
            },
        }
    case errors.Is(err, ErrUnauthorized):
        return 401, &ErrorResponse{
            Error: ErrorDetail{
                Message: "Unauthorized",
                Type:    "AuthenticationError",
                Code:    401,
            },
        }
    default:
        return 500, &ErrorResponse{
            Error: ErrorDetail{
                Message: "Internal server error",
                Type:    "InternalError",
                Code:    500,
            },
        }
    }
}
```

---

## Testing Requirements

### Unit Test Coverage

**Minimum**: 80% coverage
**Target**: 90%+ coverage

```bash
# Run tests with coverage
go test ./... -cover -coverprofile=coverage.out

# View coverage report
go tool cover -html=coverage.out

# Enforce minimum coverage
go test ./... -cover | grep -q "coverage: [8-9][0-9]"
```

### Test Structure (Table-Driven)

```go
func TestUserService_CreateUser(t *testing.T) {
    tests := []struct {
        name      string
        input     *CreateUserInput
        setupMock func(*mocks.UserRepository)
        want      *models.User
        wantErr   error
    }{
        {
            name: "success",
            input: &CreateUserInput{
                TenantID: "tenant-1",
                Email:    "test@example.com",
                Name:     "Test User",
            },
            setupMock: func(m *mocks.UserRepository) {
                m.On("FindByEmail", mock.Anything, "tenant-1", "test@example.com").
                    Return(nil, repository.ErrNotFound)
                m.On("Create", mock.Anything, mock.AnythingOfType("*models.User")).
                    Return(nil)
            },
            want: &models.User{
                Email: "test@example.com",
                Name:  "Test User",
            },
            wantErr: nil,
        },
        {
            name: "user exists",
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
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Arrange
            repo := &mocks.UserRepository{}
            if tt.setupMock != nil {
                tt.setupMock(repo)
            }
            cache := &mocks.Cache{}
            svc := NewUserService(repo, cache, zap.NewNop())

            // Act
            got, err := svc.CreateUser(context.Background(), tt.input)

            // Assert
            if tt.wantErr != nil {
                assert.ErrorIs(t, err, tt.wantErr)
                return
            }
            assert.NoError(t, err)
            assert.NotEmpty(t, got.ID)
            assert.Equal(t, tt.want.Email, got.Email)
        })
    }
}
```

---

## Security Checklist

Before committing ANY code:

### Input Validation
- [ ] All user inputs validated
- [ ] SQL injection prevented (parameterized queries)
- [ ] XSS prevented (output escaping)
- [ ] Path traversal prevented
- [ ] File upload restrictions enforced

### Authentication
- [ ] OAuth2 properly implemented
- [ ] Tokens have expiration
- [ ] Refresh tokens rotated
- [ ] Sessions invalidated on logout

### Authorization
- [ ] All endpoints check permissions
- [ ] Tenant isolation enforced
- [ ] RBAC scopes validated
- [ ] No privilege escalation possible

### Data Protection
- [ ] Sensitive data encrypted at rest
- [ ] TLS 1.3 enforced in transit
- [ ] Secrets not in code/logs
- [ ] PII minimized

### API Security
- [ ] Rate limiting on all endpoints
- [ ] CORS properly configured
- [ ] CSRF tokens on mutations
- [ ] Webhook signatures verified

---

## Performance Guidelines

### Database Queries

```go
// ✅ GOOD: Efficient query with index
SELECT id, name, email
FROM users
WHERE tenant_id = $1
  AND status = 'active'
  AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT 100;

// Index to support
CREATE INDEX idx_users_tenant_status ON users(tenant_id, status, created_at DESC)
WHERE deleted_at IS NULL;

// ❌ BAD: N+1 query
for _, user := range users {
    profile, _ := getProfile(user.ID) // Don't do this!
}

// ✅ GOOD: Batch query
profiles, _ := getProfilesByUserIDs(userIDs)
```

### Caching Strategy

```go
// ✅ Cache-aside pattern
func (s *UserService) GetUser(ctx context.Context, id string) (*models.User, error) {
    // Try cache first
    cacheKey := fmt.Sprintf("user:%s", id)
    var user models.User
    if err := s.cache.Get(ctx, cacheKey, &user); err == nil {
        return &user, nil
    }

    // Cache miss, get from DB
    user, err := s.repo.FindByID(ctx, id)
    if err != nil {
        return nil, err
    }

    // Update cache (fire and forget)
    go func() {
        _ = s.cache.Set(context.Background(), cacheKey, user, time.Hour)
    }()

    return user, nil
}
```

### Rate Limiting

```go
// ✅ Token bucket implementation
func (rl *RateLimiter) Allow(ctx context.Context, tenantID string) (bool, error) {
    key := fmt.Sprintf("rate_limit:%s:%d", tenantID, time.Now().Unix()/60)

    count, err := rl.redis.Incr(ctx, key).Result()
    if err != nil {
        return false, err
    }

    if count == 1 {
        rl.redis.Expire(ctx, key, time.Minute)
    }

    quota := rl.getQuota(tenantID)
    return count <= int64(quota), nil
}
```

---

## Documentation Requirements

### Code Documentation

```go
// ✅ GOOD: Explains WHY and edge cases
// CreateUser creates a new user with email verification.
// It sends a verification email asynchronously and marks the user
// as unverified. The user cannot login until they verify their email.
//
// Returns ErrUserExists if a user with this email already exists
// in the same tenant. Email uniqueness is enforced per-tenant only.
//
// The verification token expires after 24 hours and can be resent
// via the ResendVerification endpoint.
func (s *UserService) CreateUser(ctx context.Context, input *CreateUserInput) (*models.User, error) {
    // Implementation
}
```

### API Documentation

Update OpenAPI spec for every endpoint change:
```yaml
paths:
  /v1/users:
    post:
      summary: Create user
      description: |
        Creates a new user and sends verification email.
        User cannot login until email is verified.
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateUserInput'
            examples:
              basic:
                value:
                  email: user@example.com
                  name: John Doe
      responses:
        '201':
          description: User created
        '409':
          description: User already exists
```

---

## Monitoring & Observability

### Metrics

```go
// Define metrics
var (
    requestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
        Name: "http_request_duration_seconds",
        Help: "HTTP request latency",
    }, []string{"method", "path", "status"})

    messagesProcessed = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "messages_processed_total",
        Help: "Total messages processed",
    }, []string{"direction", "status"})
)

// Instrument code
func (h *Handler) SendMessage(c *fiber.Ctx) error {
    start := time.Now()
    defer func() {
        requestDuration.WithLabelValues(
            c.Method(),
            c.Path(),
            strconv.Itoa(c.Response().StatusCode()),
        ).Observe(time.Since(start).Seconds())
    }()

    // Handler logic
    messagesProcessed.WithLabelValues("outbound", "sent").Inc()
    return c.JSON(response)
}
```

### Logging

```go
// ✅ Structured logging with context
log.Info("message sent",
    zap.String("tenant_id", tenantID),
    zap.String("instance_id", instanceID),
    zap.String("message_id", messageID),
    zap.String("to", to),
    zap.String("type", msgType),
    zap.Duration("duration", time.Since(start)),
)

log.Error("failed to send message",
    zap.Error(err),
    zap.String("tenant_id", tenantID),
    zap.String("instance_id", instanceID),
    zap.String("to", to),
)
```

---

## Emergency Procedures

### Rollback Procedure

```bash
# 1. Identify problematic deployment
kubectl get deployments -n whatsapp-adapter

# 2. Rollback to previous version
kubectl rollout undo deployment/api-server -n whatsapp-adapter

# 3. Verify rollback
kubectl rollout status deployment/api-server -n whatsapp-adapter

# 4. Check logs
kubectl logs -f deployment/api-server -n whatsapp-adapter

# 5. Notify team
# Post in #incidents channel
```

### Database Rollback

```bash
# Rollback migration
migrate -path migrations -database "$DATABASE_URL" down 1

# Verify schema
psql $DATABASE_URL -c "\d users"
```

---

## Claude-Specific Instructions

### When Starting a Session

1. Read all `docs/*.md` files
2. Read `AGENTS.md` and `CLAUDE.md`
3. Review recent commits (`git log --oneline -10`)
4. Check current branch and status
5. Ask for clarification if requirements unclear

### When Implementing Features

1. Start with tests (TDD when possible)
2. Implement minimal version first
3. Add error handling
4. Add logging/metrics
5. Update documentation
6. Commit granularly

### When User Says "ULTRATHINK"

1. Perform deep analysis (25+ minutes)
2. Document findings in markdown
3. Create detailed implementation plan
4. Identify all risks and dependencies
5. Estimate time realistically
6. Get user approval before coding

### When Unsure

**ASK QUESTIONS**. Don't assume. Better to clarify than implement wrong solution.

---

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0.0 | 2025-11-18 | Initial Claude guide |

---

**Remember**: You're building a production system that will handle real businesses' customer communications. Every line of code matters. Every security decision matters. Every performance optimization matters.

**Think senior. Code senior. Ship production-ready.**
