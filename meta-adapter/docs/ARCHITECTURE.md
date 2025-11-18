# Arquitetura do WhatsApp Meta API Adapter

## Visão Geral

O WhatsApp Meta API Adapter é um proxy/adapter que traduz requisições HTTP/JSON compatíveis com a Meta WhatsApp Business Cloud API para o protocolo nativo do WhatsApp Web usando a biblioteca whatsmeow.

## Diagrama de Componentes

```
┌────────────────────────────────────────────────────────────────┐
│                       CLIENTE EXTERNO                          │
│  (Aplicação, Sistema, Bot, CRM, etc.)                         │
└───────────────────────────┬────────────────────────────────────┘
                            │
                            │ HTTP/JSON REST
                            │ (Meta API Format)
                            ↓
┌────────────────────────────────────────────────────────────────┐
│                     API GATEWAY (Fiber)                        │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │  Middlewares:                                            │ │
│  │  • Authentication (API Key)                              │ │
│  │  • CORS                                                  │ │
│  │  • Rate Limiting                                         │ │
│  │  • Logging                                               │ │
│  │  • Error Handling                                        │ │
│  └──────────────────────────────────────────────────────────┘ │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │  Routes:                                                 │ │
│  │  • POST /api/v1/messages       → Send Message           │ │
│  │  • GET  /api/v1/session/qr     → Get QR Code            │ │
│  │  • GET  /api/v1/session/status → Session Status         │ │
│  │  • POST /api/v1/media          → Upload Media           │ │
│  │  • GET  /api/v1/:media_id      → Download Media         │ │
│  └──────────────────────────────────────────────────────────┘ │
└───────────────────────────┬────────────────────────────────────┘
                            │
                            ↓
┌────────────────────────────────────────────────────────────────┐
│                    ADAPTER LAYER                               │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │  Message Translator:                                     │ │
│  │  • Meta JSON → whatsmeow Protobuf                        │ │
│  │  • Text, Image, Video, Audio, Document                   │ │
│  │  • Location, Contacts, Reactions                         │ │
│  │  • Context Info (replies)                                │ │
│  └──────────────────────────────────────────────────────────┘ │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │  Media Handler:                                          │ │
│  │  • Download from URL                                     │ │
│  │  • Upload to WhatsApp                                    │ │
│  │  • Cache management                                      │ │
│  └──────────────────────────────────────────────────────────┘ │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │  Webhook Formatter:                                      │ │
│  │  • whatsmeow events → Meta webhook format                │ │
│  │  • Message events                                        │ │
│  │  • Receipt events (delivered/read)                       │ │
│  └──────────────────────────────────────────────────────────┘ │
└───────────────────────────┬────────────────────────────────────┘
                            │
                            ↓
┌────────────────────────────────────────────────────────────────┐
│                 WHATSAPP CLIENT WRAPPER                        │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │  whatsmeow.Client:                                       │ │
│  │  • Connection management                                 │ │
│  │  • Event handling                                        │ │
│  │  • Session management                                    │ │
│  │  • QR Code generation                                    │ │
│  └──────────────────────────────────────────────────────────┘ │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │  Message Builders:                                       │ │
│  │  • BuildTextMessage()                                    │ │
│  │  • BuildImageMessage()                                   │ │
│  │  • BuildLocationMessage()                                │ │
│  │  • etc...                                                │ │
│  └──────────────────────────────────────────────────────────┘ │
└───────────────────────────┬────────────────────────────────────┘
                            │
                            │ WebSocket + Noise Protocol
                            │ Binary XML + Protobuf
                            │ Signal Protocol (E2E)
                            ↓
┌────────────────────────────────────────────────────────────────┐
│                   WHATSAPP WEB SERVERS                         │
│                    (web.whatsapp.com)                          │
└────────────────────────────────────────────────────────────────┘

                            ↓
         ┌──────────────────────────────────────┐
         │   WEBHOOK DISPATCHER (Async)         │
         │  • HTTP POST to configured URL       │
         │  • Retry with exponential backoff    │
         │  • Events: messages, receipts        │
         └──────────────────────────────────────┘
                            ↓
         ┌──────────────────────────────────────┐
         │        EXTERNAL WEBHOOK URL          │
         │    (Sistema do cliente recebe)       │
         └──────────────────────────────────────┘

                    STORAGE LAYER
         ┌──────────────────────────────────────┐
         │        SQLite Database               │
         │  • Sessions                          │
         │  • Media cache                       │
         └──────────────────────────────────────┘
         ┌──────────────────────────────────────┐
         │   whatsmeow Store (SQLite)           │
         │  • Device credentials                │
         │  • Signal Protocol keys              │
         │  • Identity keys                     │
         │  • PreKeys                           │
         └──────────────────────────────────────┘
```

## Fluxo de Dados

### 1. Envio de Mensagem

```
Cliente → POST /api/v1/messages (Meta JSON)
    ↓
API Handler (autenticação, validação)
    ↓
Message Translator (JSON → Protobuf)
    ↓
    ├─→ [Se mídia com URL]
    │       ↓
    │   Media Handler (download)
    │       ↓
    │   WhatsApp Client (upload)
    │       ↓
    │   (continua...)
    │
whatsmeow Client (criptografia E2E)
    ↓
WhatsApp Web Servers
    ↓
Resposta (message_id, timestamp)
    ↓
Meta JSON Response → Cliente
```

### 2. Recebimento de Mensagem

```
WhatsApp Web Servers
    ↓
whatsmeow Client (evento Message)
    ↓
Event Handler
    ↓
Webhook Formatter (Protobuf → Meta JSON)
    ↓
Webhook Dispatcher
    ↓
HTTP POST → WEBHOOK_URL (com retry)
```

### 3. Autenticação (QR Code)

```
Cliente → GET /api/v1/session/qr
    ↓
whatsmeow Client.GetQRCode()
    ↓
Gera chaves efêmeras
    ↓
Conecta ao WhatsApp
    ↓
Recebe QR code string
    ↓
Retorna para cliente
    ↓
Usuário escaneia com WhatsApp mobile
    ↓
WhatsApp envia credenciais
    ↓
whatsmeow armazena no store
    ↓
Evento Connected
```

## Componentes Detalhados

### API Gateway (Fiber)

**Responsabilidades:**
- Receber requisições HTTP
- Autenticação via API Key
- Rate limiting
- CORS
- Roteamento
- Serialização/deserialização JSON
- Error handling

**Tecnologia:** gofiber/fiber v2

### Adapter Layer

**Message Translator:**
- Mapeia tipos de mensagens Meta → whatsmeow
- Valida campos obrigatórios
- Traduz estruturas de dados
- Gerencia context info (replies)

**Media Handler:**
- Download de mídia via HTTP(S)
- Upload para WhatsApp
- Cache de URLs
- Validação de tamanho/tipo

**Webhook Formatter:**
- Converte eventos whatsmeow para formato Meta
- Formata timestamps
- Preenche metadata

### WhatsApp Client Wrapper

**Funcionalidades:**
- Abstração sobre whatsmeow
- Gerenciamento de conexão
- Event dispatching
- Helpers para construir mensagens
- Session management

### Webhook Dispatcher

**Características:**
- Assíncrono (goroutines)
- Retry com exponential backoff
- Timeout configurável
- Logging de falhas
- Queue de eventos (futuro: Redis)

### Storage Layer

**SQLite para Adapter:**
- Sessions metadata
- Media cache
- Configurações

**whatsmeow Store:**
- Device credentials
- Signal Protocol keys
- PreKeys pool
- Identity keys
- Session data

## Segurança

### Camadas de Segurança

1. **API Authentication:**
   - API Keys validadas em middleware
   - Suporte a múltiplas keys (multi-tenant)

2. **Transport:**
   - HTTPS recomendado em produção
   - CORS configurável

3. **WhatsApp Protocol:**
   - Noise Protocol (handshake)
   - Signal Protocol (E2E encryption)
   - Certificate pinning

4. **Storage:**
   - Keys criptografadas pelo whatsmeow
   - File permissions (0600)
   - SQLite em filesystem protegido

### Considerações

- ⚠️ API Keys em variáveis de ambiente (não hardcode)
- ⚠️ HTTPS obrigatório em produção
- ⚠️ Webhook signatures (TODO: implementar HMAC)
- ⚠️ Input validation em todos os endpoints
- ⚠️ Sanitização de dados sensíveis em logs

## Performance

### Otimizações

1. **Concurrency:**
   - Fiber (FastHTTP) - alta performance
   - Goroutines para webhooks
   - Event handling assíncrono

2. **Caching:**
   - Media cache (24h TTL)
   - Session info cache
   - (Futuro: Redis distributed cache)

3. **Connection Pooling:**
   - HTTP client reutilizado
   - WebSocket persistente

### Bottlenecks Potenciais

- ❗ Download de mídia (depende de latência externa)
- ❗ Upload para WhatsApp (limited by bandwidth)
- ❗ SQLite (não é ideal para high concurrency writes)
- ❗ Single WhatsApp connection (não distribuível)

### Escalabilidade

**Vertical Scaling:**
- ✅ Aumentar CPU/RAM do container
- ✅ Go gerencia goroutines eficientemente

**Horizontal Scaling:**
- ❌ **NÃO SUPORTADO** - cada instância = uma sessão WhatsApp
- Para múltiplas sessões: deploy múltiplas instâncias independentes

## Confiabilidade

### Auto-Recovery

1. **Auto-Reconnect:**
   - whatsmeow tenta reconectar automaticamente
   - Exponential backoff

2. **Graceful Shutdown:**
   - Signal handling (SIGTERM, SIGINT)
   - Desconecta WhatsApp corretamente
   - Fecha database connections

3. **Error Handling:**
   - Todos os erros logados
   - Errors compatíveis com Meta API format
   - HTTP status codes corretos

### Monitoramento

**Health Checks:**
- `/health` endpoint
- Verifica conexão WhatsApp
- Docker healthcheck integrado

**Logs:**
- Structured logging (zerolog)
- JSON format em produção
- Console format em desenvolvimento
- Log levels configuráveis

**Metrics (TODO):**
- Prometheus metrics
- Grafana dashboards
- Alertas

## Limitações Arquiteturais

1. **Single Session:**
   - Uma instância = uma conta WhatsApp
   - Não há shared state entre instâncias

2. **SQLite:**
   - Não é ideal para high write concurrency
   - Arquivo único (não distribuível)
   - Migração para PostgreSQL recomendada para produção

3. **In-Memory State:**
   - Restart = perde state não persistido
   - Webhooks em fila podem ser perdidos

4. **Protocol Dependence:**
   - whatsmeow depende de reverse engineering
   - Mudanças no protocolo WhatsApp podem quebrar

## Próximas Evoluções

### v1.1

- [ ] Redis para queue de webhooks
- [ ] Metrics com Prometheus
- [ ] PostgreSQL support
- [ ] Health checks avançados

### v2.0

- [ ] Multi-tenant (múltiplas sessões)
- [ ] Distributed tracing (OpenTelemetry)
- [ ] Circuit breaker pattern
- [ ] Event sourcing para audit
- [ ] GraphQL API (além de REST)

## Referências

- [whatsmeow](https://github.com/tulir/whatsmeow)
- [Meta WhatsApp Cloud API](https://developers.facebook.com/docs/whatsapp/cloud-api/)
- [Fiber Framework](https://gofiber.io/)
- [Signal Protocol](https://signal.org/docs/)
- [Noise Protocol Framework](https://noiseprotocol.org/)
