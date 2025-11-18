# TIER 1 & 2 Implementation Plan
# WhatsApp Meta API Adapter - Complete Feature Set

**Date**: 2025-11-18
**Goal**: Implement 98% feature coverage (from current 95%)
**Effort**: TIER 1 + TIER 2 = ~75-112 hours of work
**Timeline**: Implementing foundational structure for all features

---

## 🎯 IMPLEMENTATION STRATEGY

### Priorities
1. **Security First**: Every feature must maintain tenant isolation
2. **Production Ready**: No TODOs, no placeholders
3. **Meta API Compatible**: Request/response formats match official API
4. **Incremental**: Each feature independent and testable

### Order of Implementation
1. **Message Editing** (Simplest - validates approach)
2. **Ephemeral Messages** (Simple - tests settings)
3. **Polls** (Medium - tests new message types)
4. **Groups** (Complex - most endpoints)
5. **Newsletters** (Complex - independent system)

---

## 📐 ARCHITECTURE DESIGN

### New File Structure
```
internal/
├── whatsapp/
│   ├── manager.go (existing - add event handlers)
│   ├── groups.go (NEW - group operations)
│   ├── polls.go (NEW - poll operations)
│   └── newsletters.go (NEW - newsletter operations)
├── api/
│   ├── groups.go (NEW - group endpoints)
│   ├── polls.go (NEW - poll endpoints)
│   ├── newsletters.go (NEW - newsletter endpoints)
│   └── chats.go (NEW - chat settings/ephemeral)
├── models/
│   ├── group.go (NEW - group model)
│   └── newsletter.go (NEW - newsletter model)
└── repository/
    ├── group.go (NEW - group repository)
    └── newsletter.go (NEW - newsletter repository)
```

---

## 🔧 TIER 1 - CRITICAL FEATURES

### 1. MESSAGE EDITING ⭐⭐⭐⭐⭐

#### Backend (whatsapp/manager.go)
```go
// EditMessage edits a previously sent message
func (m *Manager) EditMessage(
    ctx context.Context,
    tenantID, instanceID string,
    chat types.JID,
    messageID types.MessageID,
    newText string,
) (string, error)
```

#### API (Extend messages.go)
```
PATCH /v1/{phone_number_id}/messages/{message_id}
Body: { "text": { "body": "edited text" } }
Response: { "success": true, "message_id": "..." }
```

#### Event Handler
- Already supported via `IsEdit` flag in events.Message
- Update handleIncomingMessage to detect edited messages

#### Database
- Use existing messages table
- Track edit history in content JSON

---

### 2. EPHEMERAL MESSAGES ⭐⭐⭐⭐

#### Backend (whatsapp/manager.go)
```go
// SetChatDisappearingTimer sets disappearing timer for a chat
func (m *Manager) SetChatDisappearingTimer(
    ctx context.Context,
    tenantID, instanceID string,
    chat types.JID,
    timer time.Duration,
) error

// SetDefaultDisappearingTimer sets default timer for new chats
func (m *Manager) SetDefaultDisappearingTimer(
    ctx context.Context,
    tenantID, instanceID string,
    timer time.Duration,
) error
```

#### API (NEW: api/chats.go)
```
PATCH /v1/{phone_number_id}/chats/{chat_jid}/disappearing
Body: { "duration": 86400 } // seconds (0 = off, 86400 = 1 day, 604800 = 7 days)
Response: { "success": true, "duration": 86400 }

PATCH /v1/{phone_number_id}/settings/disappearing
Body: { "duration": 604800 }
Response: { "success": true, "duration": 604800 }
```

#### Database
- Add chat_settings table (or use JSONB in instance settings)
- Track disappearing_timer per chat

---

### 3. GROUPS ⭐⭐⭐⭐⭐

#### Backend (NEW: whatsapp/groups.go)
```go
// Group creation and management
CreateGroup(ctx, tenantID, instanceID, req CreateGroupRequest) (*types.GroupInfo, error)
GetGroupInfo(ctx, tenantID, instanceID, groupJID types.JID) (*types.GroupInfo, error)
GetJoinedGroups(ctx, tenantID, instanceID) ([]*types.GroupInfo, error)
LeaveGroup(ctx, tenantID, instanceID, groupJID types.JID) error
JoinGroupWithLink(ctx, tenantID, instanceID, code string) (types.JID, error)

// Participant management
AddParticipants(ctx, tenantID, instanceID, groupJID types.JID, participants []types.JID) error
RemoveParticipants(ctx, tenantID, instanceID, groupJID types.JID, participants []types.JID) error
PromoteParticipants(ctx, tenantID, instanceID, groupJID types.JID, participants []types.JID) error
DemoteParticipants(ctx, tenantID, instanceID, groupJID types.JID, participants []types.JID) error

// Group settings
SetGroupName(ctx, tenantID, instanceID, groupJID types.JID, name string) error
SetGroupDescription(ctx, tenantID, instanceID, groupJID types.JID, description string) error
SetGroupPhoto(ctx, tenantID, instanceID, groupJID types.JID, imageData []byte) (string, error)
SetGroupAnnounce(ctx, tenantID, instanceID, groupJID types.JID, announce bool) error
SetGroupLocked(ctx, tenantID, instanceID, groupJID types.JID, locked bool) error

// Invite links
GetGroupInviteLink(ctx, tenantID, instanceID, groupJID types.JID, reset bool) (string, error)
```

#### API (NEW: api/groups.go)
```
POST   /v1/groups
GET    /v1/groups
GET    /v1/groups/:group_id
PATCH  /v1/groups/:group_id
DELETE /v1/groups/:group_id (leave)
POST   /v1/groups/:group_id/photo
GET    /v1/groups/:group_id/invite
POST   /v1/groups/join
POST   /v1/groups/:group_id/participants
DELETE /v1/groups/:group_id/participants/:phone
POST   /v1/groups/:group_id/admins
DELETE /v1/groups/:group_id/admins/:phone
PATCH  /v1/groups/:group_id/settings
```

#### Models (NEW: models/group.go)
```go
type Group struct {
    TenantID    string
    InstanceID  string
    GroupJID    string
    Name        string
    Description string
    PhotoURL    string
    IsAnnounce  bool
    IsLocked    bool
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type GroupParticipant struct {
    GroupJID    string
    ParticipantJID string
    IsAdmin     bool
    JoinedAt    time.Time
}
```

#### Database (NEW: repository/group.go)
- Use existing groups table from migrations
- Use existing group_participants table
- Implement CRUD operations

#### Event Handlers
```go
// In manager.go eventHandler
case *events.JoinedGroup:
    // Save group to database
case *events.GroupInfo:
    // Update group info (name, description, participants)
```

---

## 🔧 TIER 2 - HIGH PRIORITY FEATURES

### 4. POLLS (Enquetes) ⭐⭐⭐⭐

#### Backend (NEW: whatsapp/polls.go)
```go
// CreatePoll creates a poll message
func (m *Manager) CreatePoll(
    ctx context.Context,
    tenantID, instanceID string,
    to types.JID,
    question string,
    options []string,
    selectableCount int,
) (string, error)

// VotePoll votes on a poll
func (m *Manager) VotePoll(
    ctx context.Context,
    tenantID, instanceID string,
    pollInfo *types.MessageInfo,
    selectedOptions []string,
) error
```

#### API (Extend messages.go + NEW: api/polls.go)
```
POST /v1/{phone_number_id}/messages
Body: {
    "type": "poll",
    "to": "5511999999999",
    "poll": {
        "question": "What's your favorite color?",
        "options": ["Red", "Blue", "Green"],
        "selectable_count": 1
    }
}

POST /v1/{phone_number_id}/polls/{message_id}/vote
Body: {
    "options": ["Blue"]
}
```

#### Event Handler
```go
case msg.PollCreationMessage != nil:
    // Save poll to database
case msg.PollUpdateMessage != nil:
    // Update poll votes
```

---

### 5. NEWSLETTERS (Canais) ⭐⭐⭐⭐

#### Backend (NEW: whatsapp/newsletters.go)
```go
// Newsletter management
CreateNewsletter(ctx, tenantID, instanceID, params CreateNewsletterParams) (*types.NewsletterMetadata, error)
GetNewsletterInfo(ctx, tenantID, instanceID, jid types.JID) (*types.NewsletterMetadata, error)
GetSubscribedNewsletters(ctx, tenantID, instanceID) ([]*types.NewsletterMetadata, error)
FollowNewsletter(ctx, tenantID, instanceID, jid types.JID) error
UnfollowNewsletter(ctx, tenantID, instanceID, jid types.JID) error

// Newsletter messages
GetNewsletterMessages(ctx, tenantID, instanceID, jid types.JID, limit, offset int) ([]*types.NewsletterMessage, error)
SendNewsletterMessage(ctx, tenantID, instanceID, jid types.JID, message *waE2E.Message) (string, error)
NewsletterReact(ctx, tenantID, instanceID, jid types.JID, serverID, reaction, messageID) error
NewsletterMarkViewed(ctx, tenantID, instanceID, jid types.JID, serverIDs []types.MessageServerID) error
```

#### API (NEW: api/newsletters.go)
```
POST   /v1/newsletters
GET    /v1/newsletters
GET    /v1/newsletters/:id
POST   /v1/newsletters/:id/follow
DELETE /v1/newsletters/:id/unfollow
GET    /v1/newsletters/:id/messages
POST   /v1/newsletters/:id/messages
POST   /v1/newsletters/:id/messages/:message_id/react
POST   /v1/newsletters/:id/messages/:message_id/view
PATCH  /v1/newsletters/:id/mute
```

#### Models (NEW: models/newsletter.go)
```go
type Newsletter struct {
    TenantID      string
    InstanceID    string
    NewsletterJID string
    Name          string
    Description   string
    Verified      bool
    Role          string // owner, admin, subscriber
    Muted         bool
    SubscribedAt  time.Time
    CreatedAt     time.Time
    UpdatedAt     time.Time
}
```

#### Database (NEW: repository/newsletter.go)
- Create newsletters table
- Create newsletter_subscriptions table
- Implement CRUD operations

#### Event Handlers
```go
case *events.NewsletterJoin:
    // Save subscription
case *events.NewsletterLeave:
    // Remove subscription
case *events.NewsletterLiveUpdate:
    // Update newsletter messages
```

---

## 🔒 SECURITY CHECKLIST

### For All Features
- [ ] Tenant isolation enforced (all queries filter by tenant_id)
- [ ] Input validation (sanitize all user inputs)
- [ ] Permission checks (verify user can perform action)
- [ ] Rate limiting (per-tenant quotas)
- [ ] Audit logging (important actions logged)
- [ ] Error handling (no sensitive data in errors)
- [ ] SQL injection prevention (parameterized queries)

### Group-Specific
- [ ] Verify user is group member before actions
- [ ] Verify user is admin before admin actions (promote, settings)
- [ ] Prevent privilege escalation
- [ ] Limit group name length (25 chars per WhatsApp)
- [ ] Limit participants per request (prevent DoS)

### Newsletter-Specific
- [ ] Verify user is owner/admin before creating
- [ ] Prevent spam (rate limit message sends)
- [ ] Validate newsletter JID format

---

## 📊 API IMPROVEMENTS (UI/UX)

### Consistency
1. **Standard Response Format**:
```json
{
    "success": true,
    "data": { ... },
    "meta": {
        "timestamp": "2025-11-18T10:00:00Z",
        "request_id": "req_abc123"
    }
}
```

2. **Error Format** (already good):
```json
{
    "error": {
        "message": "Group not found",
        "type": "NotFoundError",
        "code": 404,
        "details": { ... }
    }
}
```

3. **Pagination** (for lists):
```json
{
    "data": [...],
    "paging": {
        "total": 150,
        "limit": 25,
        "offset": 0,
        "next": 25,
        "prev": null
    }
}
```

### Developer Experience
1. **Descriptive Errors**:
   - "Group not found" instead of "Not found"
   - Include possible solutions: "Group may have been deleted or you don't have access"

2. **Request ID Tracing**:
   - Generate unique request ID for debugging
   - Include in all responses and logs

3. **Rate Limit Headers**:
   - `X-RateLimit-Limit`
   - `X-RateLimit-Remaining`
   - `X-RateLimit-Reset`

---

## 🧪 TESTING STRATEGY

### Unit Tests
- Test each WhatsApp method independently
- Mock whatsmeow Client
- Verify correct parameters passed

### Integration Tests
- Test API endpoints end-to-end
- Use test database
- Verify database state after operations

### Security Tests
- Attempt cross-tenant access (should fail)
- Attempt privilege escalation (should fail)
- SQL injection attempts (should be prevented)

---

## 📈 ROLLOUT PLAN

### Phase 1: Message Editing & Ephemeral (Week 1)
- Low risk, simple features
- Validates approach
- Can be deployed independently

### Phase 2: Polls (Week 1)
- Medium risk
- Tests new message type handling
- Independent of other features

### Phase 3: Groups (Week 2-3)
- High complexity
- Most endpoints
- Phased rollout:
  1. Read operations (get info, list)
  2. Write operations (create, join)
  3. Admin operations (settings, participants)

### Phase 4: Newsletters (Week 3-4)
- High complexity
- Independent system
- Can be beta-tested separately

---

## 📋 IMPLEMENTATION CHECKLIST

### For Each Feature
- [ ] WhatsApp methods implemented
- [ ] API endpoints created
- [ ] Request/response models defined
- [ ] Input validation added
- [ ] Database operations (if needed)
- [ ] Event handlers updated
- [ ] Error handling complete
- [ ] Security checks in place
- [ ] Logging added
- [ ] Documentation updated (BMAD)
- [ ] Postman/curl examples
- [ ] Unit tests written
- [ ] Integration tests written

---

## 🎯 SUCCESS METRICS

### Coverage
- **Before**: 95% (individual messaging)
- **After TIER 1**: 97% (+ groups, editing, ephemeral)
- **After TIER 2**: 98% (+ polls, newsletters)

### API Endpoints
- **Before**: 15 endpoints
- **After**: 45+ endpoints

### Feature Completeness
- **Individual Messaging**: 100% ✅
- **Groups**: 90% ✅
- **Newsletters**: 85% ✅
- **Polls**: 80% ✅
- **Advanced Features**: 95% ✅

---

## 🚀 DEPLOYMENT

### Environment Variables (New)
```bash
# Group settings
MAX_GROUP_PARTICIPANTS=256
MAX_GROUP_NAME_LENGTH=25
GROUP_LINK_EXPIRY_HOURS=72

# Newsletter settings
MAX_NEWSLETTER_SUBSCRIBERS=10000
NEWSLETTER_MESSAGE_RATE_LIMIT=100  # per hour

# Poll settings
MAX_POLL_OPTIONS=12
MAX_POLL_SELECTABLE=12
```

### Database Migrations
- Groups tables already exist ✅
- Need newsletter tables (new migration)
- Need chat_settings table (new migration)

---

## 📚 DOCUMENTATION UPDATES

### API Reference
- Document all new endpoints
- Add request/response examples
- Document error codes
- Add webhook payloads

### BMAD Updates
- Update feature coverage percentage
- Add new endpoints to API matrix
- Update architecture diagrams
- Add deployment instructions

---

## ⚡ PERFORMANCE CONSIDERATIONS

### Database
- Index group_jid in groups table
- Index participant_jid in group_participants
- Partition messages table by tenant_id (already done)

### Caching
- Cache group info (5 minutes TTL)
- Cache newsletter metadata (10 minutes TTL)
- Cache invite links (until reset)

### Rate Limiting
- Group operations: 10/minute per tenant
- Newsletter operations: 5/minute per tenant
- Poll votes: 100/minute per tenant

---

**This plan provides the foundation for implementing TIER 1 & 2 features while maintaining production quality and security standards.**
