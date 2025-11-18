# WhatsApp Meta API Adapter

> 🚀 Production-ready WhatsApp adapter with Meta API compatibility

A multi-tenant WhatsApp API adapter that provides a Meta (Facebook) API-compatible interface for sending and receiving WhatsApp messages using the [whatsmeow](https://github.com/tulir/whatsmeow) library.

## Features

✅ **Meta API Compatible** - Drop-in replacement for WhatsApp Business API  
✅ **Multi-Tenant** - Isolate instances and data per tenant  
✅ **OAuth2 + JWT** - Secure authentication and authorization  
✅ **QR Code Authentication** - Easy WhatsApp connection  
✅ **Real-time Messaging** - Send/receive messages via WhatsApp  
✅ **Message History** - Full audit trail in PostgreSQL  
✅ **Webhook Support** - Real-time event notifications (coming soon)  
✅ **Rate Limiting** - Per-tenant quotas (coming soon)  
✅ **Production Ready** - Docker, health checks, structured logging  

## Quick Start

### Prerequisites

- Docker & Docker Compose
- Go 1.24+ (for local development)

### Running with Docker

1. Clone and configure:
```bash
git clone <repository>
cd meta-adapter
cp .env.example .env
# Edit .env with your configuration
```

2. Start all services:
```bash
make docker-up
```

3. Check health:
```bash
curl http://localhost:8080/health
```

### Running Locally

1. Start dependencies:
```bash
docker-compose up -d postgres redis rabbitmq
```

2. Run migrations:
```bash
export DATABASE_URL="postgresql://whatsapp:password@localhost:5432/whatsapp_adapter?sslmode=disable"
make migrate-up
```

3. Start the server:
```bash
make run
```

## API Usage

### 1. Get Access Token

```bash
curl -X POST http://localhost:8080/v1/oauth/token \
  -H "Content-Type: application/json" \
  -d '{}'
```

Response:
```json
{
  "access_token": "eyJhbGc...",
  "token_type": "Bearer",
  "expires_in": 3600
}
```

### 2. Create WhatsApp Instance

```bash
curl -X POST http://localhost:8080/v1/instances \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "display_name": "My Bot",
    "webhook_url": "https://your-app.com/webhook"
  }'
```

### 3. Get QR Code

```bash
curl http://localhost:8080/v1/instances/PHONE_NUMBER_ID/qrcode \
  -H "Authorization: Bearer YOUR_TOKEN"
```

Scan the QR code with WhatsApp on your phone.

### 4. Send Message

```bash
curl -X POST http://localhost:8080/v1/PHONE_NUMBER_ID/messages \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "messaging_product": "whatsapp",
    "to": "5511999999999",
    "type": "text",
    "text": {
      "body": "Hello from the adapter!"
    }
  }'
```

## Architecture

```
┌─────────────┐     ┌──────────────┐     ┌───────────────┐
│   Client    │────▶│  Fiber HTTP  │────▶│  WhatsApp     │
│   (Your App)│     │  API Server  │     │  (whatsmeow)  │
└─────────────┘     └──────────────┘     └───────────────┘
                            │
                    ┌───────┴────────┐
                    │                │
              ┌─────▼─────┐   ┌─────▼─────┐
              │ PostgreSQL│   │   Redis   │
              │  (Data)   │   │  (Cache)  │
              └───────────┘   └───────────┘
```

## Database Schema

- **tenants** - Multi-tenant accounts
- **instances** - WhatsApp instances (partitioned by tenant_id)
- **messages** - Message history (partitioned 16-way)
- **oauth_clients** - OAuth2 client credentials
- **oauth_tokens** - Active access tokens

## Security

- 🔒 JWT authentication with scopes
- 🔒 Per-tenant data isolation (hash partitioning)
- 🔒 Non-root Docker containers
- 🔒 Read-only filesystem (where possible)
- 🔒 Health checks and graceful shutdown

## Development

```bash
# Run tests
make test

# Build binary
make build

# View logs
make docker-logs

# Stop all services
make docker-down
```

## Environment Variables

See `.env.example` for all configuration options.

Key variables:
- `DATABASE_URL` - PostgreSQL connection string
- `REDIS_URL` - Redis connection string  
- `JWT_SECRET` - Secret for signing JWT tokens
- `PORT` - HTTP server port (default: 8080)

## Production Deployment

1. Generate strong secrets:
```bash
# Generate JWT secret
openssl rand -base64 32

# Generate database password
openssl rand -base64 24
```

2. Update .env with production values

3. Deploy with Docker Compose or Kubernetes (see DEPLOYMENT.md)

## Contributing

See [AGENTS.md](AGENTS.md) for development standards and [CLAUDE.md](CLAUDE.md) for AI-assisted development guidelines.

## License

MIT License - see LICENSE file

## Support

For issues and questions, please open a GitHub issue.
