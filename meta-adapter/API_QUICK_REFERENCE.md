# API Quick Reference Guide
# WhatsApp Meta API Adapter

**Version**: 1.0.0 (Current) + TIER 1 & 2 (Planned)
**Base URL**: `http://localhost:8080/v1`
**Authentication**: Bearer token (OAuth2)

---

## 🔑 AUTHENTICATION

### Get Access Token
```bash
curl -X POST http://localhost:8080/v1/oauth/token \
  -H "Content-Type: application/json" \
  -d '{
    "grant_type": "client_credentials",
    "client_id": "your-client-id",
    "client_secret": "your-client-secret"
  }'
```

**Response**:
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "token_type": "Bearer",
  "expires_in": 3600
}
```

---

## ✅ IMPLEMENTED (PRODUCTION READY)

### Instance Management

#### Create Instance
```bash
curl -X POST http://localhost:8080/v1/instances \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "display_name": "Customer Support",
    "webhook_url": "https://your-app.com/webhooks"
  }'
```

#### Get QR Code
```bash
curl -X GET "http://localhost:8080/v1/instances/550123456789/qrcode" \
  -H "Authorization: Bearer $TOKEN"
```

#### List Instances
```bash
curl -X GET "http://localhost:8080/v1/instances?limit=25&offset=0" \
  -H "Authorization: Bearer $TOKEN"
```

### Messaging

#### Send Text Message
```bash
curl -X POST http://localhost:8080/v1/550123456789/messages \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "messaging_product": "whatsapp",
    "to": "5511999999999",
    "type": "text",
    "text": {
      "body": "Hello, World!"
    }
  }'
```

#### Send Image
```bash
curl -X POST http://localhost:8080/v1/550123456789/messages \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "to": "5511999999999",
    "type": "image",
    "image": {
      "link": "https://example.com/image.jpg",
      "caption": "Check this out!"
    }
  }'
```

#### Send Video
```bash
curl -X POST http://localhost:8080/v1/550123456789/messages \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "to": "5511999999999",
    "type": "video",
    "video": {
      "link": "https://example.com/video.mp4",
      "caption": "Watch this"
    }
  }'
```

#### Send Audio
```bash
curl -X POST http://localhost:8080/v1/550123456789/messages \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "to": "5511999999999",
    "type": "audio",
    "audio": {
      "link": "https://example.com/audio.ogg"
    }
  }'
```

#### Send Document
```bash
curl -X POST http://localhost:8080/v1/550123456789/messages \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "to": "5511999999999",
    "type": "document",
    "document": {
      "link": "https://example.com/doc.pdf",
      "filename": "invoice.pdf",
      "caption": "Your invoice"
    }
  }'
```

#### List Messages
```bash
curl -X GET "http://localhost:8080/v1/550123456789/messages?limit=25&offset=0" \
  -H "Authorization: Bearer $TOKEN"
```

### Message Actions

#### Mark as Read
```bash
curl -X POST http://localhost:8080/v1/550123456789/messages/wamid.xxx/read \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "status": "read"
  }'
```

#### Delete Message
```bash
curl -X DELETE http://localhost:8080/v1/550123456789/messages/wamid.xxx \
  -H "Authorization: Bearer $TOKEN"
```

#### React to Message
```bash
curl -X POST http://localhost:8080/v1/550123456789/messages/wamid.xxx/react \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "emoji": "👍"
  }'
```

### Presence & Typing

#### Set Presence (Online/Offline)
```bash
curl -X PATCH http://localhost:8080/v1/550123456789/presence \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "status": "online"
  }'
```

#### Send Typing Indicator
```bash
curl -X POST http://localhost:8080/v1/550123456789/typing \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "to": "5511999999999",
    "state": "composing",
    "media": "text"
  }'
```

---

## 🔄 TIER 1 - PLANNED (Critical Features)

### Message Editing

#### Edit Message
```bash
curl -X PATCH http://localhost:8080/v1/550123456789/messages/wamid.xxx \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "text": {
      "body": "Corrected message text"
    }
  }'
```

### Ephemeral Messages

#### Set Chat Disappearing Timer
```bash
curl -X PATCH http://localhost:8080/v1/550123456789/chats/5511999999999@s.whatsapp.net/disappearing \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "duration": 86400
  }'
```
**Durations**: 0 (off), 86400 (1 day), 604800 (7 days), 7776000 (90 days)

#### Set Default Disappearing Timer
```bash
curl -X PATCH http://localhost:8080/v1/550123456789/settings/disappearing \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "duration": 604800
  }'
```

### Groups

#### Create Group
```bash
curl -X POST http://localhost:8080/v1/groups \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "instance_id": "550123456789",
    "name": "Team Chat",
    "participants": ["5511999999999", "5511888888888"]
  }'
```

#### List Groups
```bash
curl -X GET "http://localhost:8080/v1/groups?instance_id=550123456789" \
  -H "Authorization: Bearer $TOKEN"
```

#### Get Group Info
```bash
curl -X GET http://localhost:8080/v1/groups/120363123456789012@g.us \
  -H "Authorization: Bearer $TOKEN"
```

#### Update Group Name/Description
```bash
curl -X PATCH http://localhost:8080/v1/groups/120363123456789012@g.us \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "New Group Name",
    "description": "Updated description"
  }'
```

#### Upload Group Photo
```bash
curl -X POST http://localhost:8080/v1/groups/120363123456789012@g.us/photo \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: multipart/form-data" \
  -F "photo=@group_photo.jpg"
```

#### Add Participants
```bash
curl -X POST http://localhost:8080/v1/groups/120363123456789012@g.us/participants \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "participants": ["5511777777777"]
  }'
```

#### Remove Participant
```bash
curl -X DELETE http://localhost:8080/v1/groups/120363123456789012@g.us/participants/5511777777777 \
  -H "Authorization: Bearer $TOKEN"
```

#### Promote to Admin
```bash
curl -X POST http://localhost:8080/v1/groups/120363123456789012@g.us/admins \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "participants": ["5511999999999"]
  }'
```

#### Demote from Admin
```bash
curl -X DELETE http://localhost:8080/v1/groups/120363123456789012@g.us/admins/5511999999999 \
  -H "Authorization: Bearer $TOKEN"
```

#### Get Invite Link
```bash
curl -X GET http://localhost:8080/v1/groups/120363123456789012@g.us/invite \
  -H "Authorization: Bearer $TOKEN"
```

#### Join Group via Link
```bash
curl -X POST http://localhost:8080/v1/groups/join \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "instance_id": "550123456789",
    "invite_code": "DjsK3j2kLm9xYz"
  }'
```

#### Leave Group
```bash
curl -X DELETE http://localhost:8080/v1/groups/120363123456789012@g.us \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "instance_id": "550123456789"
  }'
```

#### Update Group Settings
```bash
curl -X PATCH http://localhost:8080/v1/groups/120363123456789012@g.us/settings \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "announce": true,
    "locked": true
  }'
```

---

## 🔄 TIER 2 - PLANNED (High Priority)

### Polls

#### Create Poll
```bash
curl -X POST http://localhost:8080/v1/550123456789/messages \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "to": "5511999999999",
    "type": "poll",
    "poll": {
      "question": "What is your favorite color?",
      "options": ["Red", "Blue", "Green", "Yellow"],
      "selectable_count": 1
    }
  }'
```

#### Vote on Poll
```bash
curl -X POST http://localhost:8080/v1/550123456789/polls/wamid.poll123/vote \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "options": ["Blue"]
  }'
```

### Newsletters

#### Create Newsletter
```bash
curl -X POST http://localhost:8080/v1/newsletters \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "instance_id": "550123456789",
    "name": "Tech News Daily",
    "description": "Daily tech news and updates"
  }'
```

#### List Subscribed Newsletters
```bash
curl -X GET "http://localhost:8080/v1/newsletters?instance_id=550123456789" \
  -H "Authorization: Bearer $TOKEN"
```

#### Get Newsletter Info
```bash
curl -X GET http://localhost:8080/v1/newsletters/120363456789012345@newsletter \
  -H "Authorization: Bearer $TOKEN"
```

#### Follow Newsletter
```bash
curl -X POST http://localhost:8080/v1/newsletters/120363456789012345@newsletter/follow \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "instance_id": "550123456789"
  }'
```

#### Unfollow Newsletter
```bash
curl -X DELETE http://localhost:8080/v1/newsletters/120363456789012345@newsletter/unfollow \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "instance_id": "550123456789"
  }'
```

#### Get Newsletter Messages
```bash
curl -X GET "http://localhost:8080/v1/newsletters/120363456789012345@newsletter/messages?limit=25" \
  -H "Authorization: Bearer $TOKEN"
```

#### Send Newsletter Message
```bash
curl -X POST http://localhost:8080/v1/newsletters/120363456789012345@newsletter/messages \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "instance_id": "550123456789",
    "type": "text",
    "text": {
      "body": "Breaking: New tech announcement!"
    }
  }'
```

#### React to Newsletter Message
```bash
curl -X POST http://localhost:8080/v1/newsletters/120363456789012345@newsletter/messages/msgid123/react \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "instance_id": "550123456789",
    "emoji": "👍"
  }'
```

#### Mark Newsletter Messages as Viewed
```bash
curl -X POST http://localhost:8080/v1/newsletters/120363456789012345@newsletter/messages/view \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "instance_id": "550123456789",
    "server_ids": [123, 124, 125]
  }'
```

#### Mute/Unmute Newsletter
```bash
curl -X PATCH http://localhost:8080/v1/newsletters/120363456789012345@newsletter/mute \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "instance_id": "550123456789",
    "mute": true
  }'
```

---

## 📊 RESPONSE FORMATS

### Success Response
```json
{
  "success": true,
  "data": {
    "id": "...",
    "...": "..."
  },
  "meta": {
    "timestamp": "2025-11-18T10:00:00Z",
    "request_id": "req_abc123"
  }
}
```

### Error Response
```json
{
  "error": {
    "message": "Resource not found",
    "type": "NotFoundError",
    "code": 404,
    "details": {
      "resource": "group",
      "id": "120363123456789012@g.us"
    }
  }
}
```

### Pagination Response
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

---

## 🔔 WEBHOOK EVENTS

### Message Received
```json
{
  "event": "message",
  "timestamp": "2025-11-18T10:00:00Z",
  "data": {
    "from": "5511999999999",
    "to": "550123456789",
    "type": "text",
    "text": {
      "body": "Hello!"
    },
    "message_id": "wamid.xxx"
  }
}
```

### Message Status Update
```json
{
  "event": "message.status",
  "timestamp": "2025-11-18T10:00:05Z",
  "data": {
    "message_id": "wamid.xxx",
    "status": "delivered"
  }
}
```

### Group Event
```json
{
  "event": "group.participant_added",
  "timestamp": "2025-11-18T10:00:00Z",
  "data": {
    "group_id": "120363123456789012@g.us",
    "participant": "5511777777777",
    "added_by": "5511999999999"
  }
}
```

---

## 🔧 RATE LIMITS

| Resource | Limit | Window |
|----------|-------|--------|
| Messages | 60 | 1 minute |
| Group Operations | 10 | 1 minute |
| Newsletter Operations | 5 | 1 minute |
| Poll Votes | 100 | 1 minute |

**Headers**:
```
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 45
X-RateLimit-Reset: 1700000060
```

---

## 🔒 AUTHENTICATION SCOPES

| Scope | Description |
|-------|-------------|
| `messages.send` | Send messages, presence, typing |
| `messages.read` | Read message history |
| `instances.manage` | Create, manage instances |
| `groups.manage` | Create, manage groups |
| `newsletters.manage` | Create, manage newsletters |

---

## 📚 ADDITIONAL RESOURCES

- **Full Documentation**: See `BMAD.md`
- **Architecture**: See `docs/ARCHITECTURE.md`
- **Implementation Plan**: See `TIER1_TIER2_IMPLEMENTATION_PLAN.md`
- **WhatsApp Analysis**: See `WHATSMEOW_COMPLETE_ANALYSIS.md`
- **Deployment**: See `QUICK_DEPLOY_GUIDE.md`

---

**Status Legend**:
- ✅ Implemented and Production Ready
- 🔄 Planned (TIER 1 & 2)
- ⏳ Future Enhancement (TIER 3+)
