# Product Requirements Document (PRD)
# WhatsApp Meta API Adapter - Enterprise Edition

**Version:** 2.0.0
**Date:** 2025-01-18
**Author:** BMAD Product Team
**Status:** PLANNING

---

## Executive Summary

WhatsApp Meta API Adapter Enterprise é uma plataforma **multi-tenant, production-ready** que permite pequenos e médios negócios acessarem funcionalidades do WhatsApp Business através de uma API 100% compatível com a Meta WhatsApp Business Cloud API, mas utilizando WhatsApp Web como backend (via whatsmeow).

### Value Proposition

**Para Pequenos Negócios:**
- 💰 **Economia**: Sem custo por mensagem (vs $0.005-0.09/msg da Meta)
- 🚀 **Rápido Setup**: QR Code em 30 segundos (vs semanas de aprovação Meta)
- 🔓 **Sem Burocracia**: Não requer aprovação comercial
- 🤖 **AI-Ready**: Porta de entrada para atendimento com agentes IA

**Para Desenvolvedores:**
- 🔌 **Drop-in Replacement**: API idêntica à Meta (facilita migração futura)
- 📊 **Dashboard Visual**: Gerenciamento completo via GUI
- 🔐 **Enterprise Security**: OAuth2, audit logs, RBAC
- 📈 **Escalável**: Multi-tenant, PostgreSQL, Redis, Kubernetes-ready

---

## 1. Product Goals

### 1.1 Business Goals

1. **Democratizar WhatsApp Business** para PMEs sem budget Meta
2. **Servir como gateway** para adoção de atendimento com IA
3. **Encorajar migração** para Meta API oficial quando negócio crescer
4. **Gerar receita** via modelo freemium/SaaS

### 1.2 User Goals

**Persona 1: Dono de Pequeno Negócio (João)**
- Precisa atender clientes via WhatsApp
- Orçamento limitado (~R$ 200/mês)
- Sem conhecimento técnico profundo
- Quer começar rápido (< 1 dia)

**Persona 2: Desenvolvedor de Chatbot (Maria)**
- Desenvolve soluções de IA para atendimento
- Precisa API confiável e compatível com Meta
- Quer trocar de backend sem reescrever código
- Necessita webhooks configuráveis e logs

**Persona 3: Agência Digital (TechCorp)**
- Gerencia WhatsApp de múltiplos clientes
- Precisa isolamento entre contas
- Requer painel admin para operação
- Necessita billing por cliente

### 1.3 Technical Goals

- ✅ **100% Meta API Compatibility** (core endpoints)
- ✅ **Multi-Tenant** com isolamento completo
- ✅ **Production-Grade** security, monitoring, scaling
- ✅ **Self-Hosted** ou SaaS deployment options
- ✅ **Open Source** (MPL 2.0) com suporte comercial opcional

---

## 2. Features & Requirements

### 2.1 Core Features (MVP 2.0)

#### F1: Multi-Tenant Management

**User Story:**
> Como **agência digital**, quero **gerenciar múltiplos clientes** para que **cada um tenha sua instância isolada**.

**Requirements:**
- [ ] Criar tenant (signup)
- [ ] Autenticação OAuth2 + API Keys
- [ ] Isolamento de dados por tenant
- [ ] Rate limiting per tenant
- [ ] Billing/usage tracking per tenant

**Acceptance Criteria:**
- Tenant A não pode ver dados do Tenant B
- Rate limit de um tenant não afeta outros
- Logs de audit completos

---

#### F2: Instance Management (Phone Numbers)

**User Story:**
> Como **dono de negócio**, quero **conectar meu número WhatsApp** para que **possa enviar/receber mensagens via API**.

**Requirements:**
- [ ] Criar instância (número)
- [ ] QR Code generation
- [ ] Pair Code (8 dígitos)
- [ ] Status tracking (disconnected, qr_pending, connected)
- [ ] Reconnect automático
- [ ] Desconectar/deletar instância

**Acceptance Criteria:**
- QR code expira em 60 segundos
- Auto-reconnect funciona após desconexão
- Dados persistem após restart do servidor

---

#### F3: Send Messages API (Meta Compatible)

**User Story:**
> Como **desenvolvedor**, quero **enviar mensagens** usando **mesma API da Meta** para que **possa trocar de backend facilmente**.

**Requirements:**
- [ ] `POST /{phone_number_id}/messages` endpoint
- [ ] Suporte a: text, image, video, audio, document, location, contacts, reaction
- [ ] Context info (replies)
- [ ] Interactive messages (buttons básicos)
- [ ] Error handling idêntico à Meta
- [ ] Response format idêntico à Meta

**Acceptance Criteria:**
- Código escrito para Meta API funciona sem alteração
- Todos os campos da Meta API são suportados ou retornam erro claro
- Rate limiting respeita limites do WhatsApp

---

#### F4: Receive Messages (Webhooks)

**User Story:**
> Como **sistema de CRM**, quero **receber mensagens em tempo real** para que **possa processar com IA**.

**Requirements:**
- [ ] Webhook configurável por instância
- [ ] Webhook global por tenant (fallback)
- [ ] Webhook verification (hub.challenge)
- [ ] Retry com exponential backoff (até 7 dias)
- [ ] Webhook logs e status
- [ ] Payload idêntico à Meta API

**Acceptance Criteria:**
- Webhook entrega mensagens em < 1 segundo
- Retry funciona mesmo após restart
- Logs mostram todas as tentativas

---

#### F5: Admin Dashboard (GUI)

**User Story:**
> Como **operador**, quero **painel visual** para que **possa gerenciar instâncias sem usar API**.

**Requirements:**
- [ ] Dashboard responsivo (React/Vue)
- [ ] Listar/criar/deletar instâncias
- [ ] Ver QR code
- [ ] Monitorar status em tempo real
- [ ] Configurar webhooks
- [ ] Ver logs de mensagens
- [ ] Analytics básico (mensagens enviadas/recebidas)
- [ ] Configurações de conta

**Acceptance Criteria:**
- Interface intuitiva (< 5 min para conectar primeira instância)
- Atualização em tempo real via WebSocket
- Mobile-friendly

---

#### F6: OAuth2 Authentication

**User Story:**
> Como **plataforma**, quero **autenticação segura** para que **tenants tenham acesso isolado**.

**Requirements:**
- [ ] OAuth2 Authorization Code Flow
- [ ] Access tokens (1 hora)
- [ ] Refresh tokens (60 dias)
- [ ] Scopes: `whatsapp_business_messaging`, `whatsapp_business_management`
- [ ] API Keys como alternativa (para scripts)
- [ ] Token revocation

**Acceptance Criteria:**
- Compatível com OAuth2 RFC
- Tokens são JWT assinados
- Revogação imediata

---

### 2.2 Advanced Features (Post-MVP)

#### F7: Interactive Messages (Lists)
- Send list messages
- Receive list responses

#### F8: Template Management
- Local template cache
- Template validation
- (Sem aprovação Meta)

#### F9: Business Profile
- Get/Update business profile via API

#### F10: Analytics Dashboard
- Message volume graphs
- Success rate
- Response time metrics
- Cost savings vs Meta API

#### F11: WhatsApp Groups
- Create/manage groups via API
- Send messages to groups
- Add/remove participants

---

## 3. Non-Functional Requirements

### 3.1 Performance

| Metric | Target | Method |
|--------|--------|--------|
| API Response Time | < 200ms (p95) | Load testing |
| Message Delivery | < 2s (p99) | Monitoring |
| Concurrent Instances | 1000+ per server | Stress testing |
| Uptime | 99.5% | SLA monitoring |

### 3.2 Security

| Requirement | Implementation |
|-------------|----------------|
| Data Encryption at Rest | PostgreSQL encryption, encrypted volumes |
| Data Encryption in Transit | TLS 1.3+ mandatory |
| Authentication | OAuth2 + API Keys with bcrypt |
| Authorization | RBAC with scopes |
| Audit Logging | All API calls logged |
| Rate Limiting | Per tenant + global |
| IP Whitelisting | Optional per tenant |
| Secret Management | Vault or AWS Secrets Manager |
| GDPR Compliance | Data export, deletion |
| LGPD Compliance | Data residency, consent |

### 3.3 Scalability

- **Horizontal Scaling**: Stateless API servers behind load balancer
- **Database**: PostgreSQL with read replicas
- **Cache**: Redis cluster
- **Message Queue**: RabbitMQ/Kafka for webhooks
- **File Storage**: S3-compatible for media
- **CDN**: CloudFront/Cloudflare for admin dashboard

### 3.4 Reliability

- **Auto-Reconnect**: WhatsApp sessions reconnect automatically
- **Circuit Breaker**: Protect against downstream failures
- **Health Checks**: `/health` endpoint for load balancers
- **Graceful Shutdown**: Finish processing before shutdown
- **Data Backup**: Daily automated backups
- **Disaster Recovery**: RTO < 4h, RPO < 1h

### 3.5 Observability

- **Metrics**: Prometheus + Grafana
- **Logging**: Structured JSON logs, centralized (Loki/ELK)
- **Tracing**: Distributed tracing (Jaeger/Zipkin)
- **Alerting**: PagerDuty/Opsgenie integration
- **Status Page**: Public status.example.com

---

## 4. User Flows

### 4.1 Tenant Signup & First Instance

```
1. Visit dashboard.example.com
2. Click "Sign Up"
3. Enter: email, password, company name
4. Verify email
5. Login → Redirect to Dashboard
6. Click "+ New Instance"
7. See QR Code
8. Scan with WhatsApp mobile
9. Instance connects → Green status
10. Configure webhook URL (optional)
11. Test sending message via API
```

### 4.2 Send Message via API

```
1. Get access token (OAuth2 or API Key)
2. POST /{phone_number_id}/messages
   {
     "messaging_product": "whatsapp",
     "to": "5521999999999",
     "type": "text",
     "text": {"body": "Hello!"}
   }
3. Receive 200 OK with message_id
4. WhatsApp delivers message
5. Webhook receives delivery confirmation
```

### 4.3 Receive Message

```
1. User sends message to business number
2. whatsmeow receives event
3. Format as Meta API webhook payload
4. POST to tenant's webhook URL
5. If webhook fails, retry with backoff
6. Log webhook attempt
7. Show in dashboard "Webhook Logs"
```

---

## 5. API Endpoints (Complete Spec)

### 5.1 Authentication

```
POST /oauth/token
POST /oauth/revoke
POST /oauth/introspect
```

### 5.2 Tenants (Admin API)

```
POST   /admin/tenants           # Create tenant
GET    /admin/tenants           # List tenants
GET    /admin/tenants/:id       # Get tenant
PATCH  /admin/tenants/:id       # Update tenant
DELETE /admin/tenants/:id       # Delete tenant
```

### 5.3 Instances (Tenant API)

```
POST   /v1/instances            # Create instance
GET    /v1/instances            # List instances
GET    /v1/instances/:id        # Get instance
PATCH  /v1/instances/:id        # Update instance
DELETE /v1/instances/:id        # Delete instance
GET    /v1/instances/:id/qr     # Get QR code
POST   /v1/instances/:id/reconnect # Force reconnect
POST   /v1/instances/:id/logout # Logout
```

### 5.4 Messages (Meta API Compatible)

```
POST   /v1/:phone_number_id/messages      # Send message
GET    /v1/:phone_number_id/messages/:id  # Get message status (future)
```

### 5.5 Media (Meta API Compatible)

```
POST   /v1/:phone_number_id/media         # Upload media
GET    /v1/:media_id                      # Download media
DELETE /v1/:media_id                      # Delete media (future)
```

### 5.6 Webhooks

```
GET    /v1/webhook              # Webhook verification (hub.challenge)
POST   /v1/webhook              # Receive webhook (if global)
GET    /v1/webhook-logs         # Get webhook logs
```

### 5.7 Analytics (Future)

```
GET    /v1/analytics/messages   # Message volume
GET    /v1/analytics/usage      # API usage
```

---

## 6. Tech Stack

### 6.1 Backend

| Component | Technology | Justification |
|-----------|-----------|---------------|
| Language | Go 1.24+ | Performance, whatsmeow native |
| API Framework | Fiber v2 | FastHTTP, Express-like API |
| Database | PostgreSQL 15+ | ACID, scalability, JSON support |
| Cache | Redis 7+ | Sessions, rate limiting, queues |
| Message Queue | RabbitMQ | Webhook delivery, async jobs |
| Auth | OAuth2 (RFC 6749) | Industry standard |
| ORM | GORM | Go-friendly, migrations |
| Validation | go-playground/validator | Struct validation |

### 6.2 Frontend (Admin Dashboard)

| Component | Technology |
|-----------|-----------|
| Framework | React 18+ ou Vue 3+ |
| UI Library | Tailwind CSS + shadcn/ui |
| State | Zustand / Pinia |
| HTTP Client | Axios |
| WebSocket | Socket.io client |
| Charts | Recharts / Chart.js |
| Build | Vite |

### 6.3 Infrastructure

| Component | Technology |
|-----------|-----------|
| Containerization | Docker |
| Orchestration | Kubernetes (optional) ou Docker Swarm |
| Reverse Proxy | Nginx / Traefik |
| Load Balancer | Nginx / HAProxy |
| Monitoring | Prometheus + Grafana |
| Logging | Loki + Promtail |
| Secrets | Vault / AWS Secrets Manager |
| CI/CD | GitHub Actions / GitLab CI |

---

## 7. Security Considerations

### 7.1 Threat Model

| Threat | Mitigation |
|--------|-----------|
| Unauthorized Access | OAuth2, API Keys, RBAC |
| Data Breach | Encryption at rest/transit, audit logs |
| DDoS | Rate limiting, CloudFlare |
| SQL Injection | Parameterized queries (GORM) |
| XSS | React auto-escaping, CSP headers |
| CSRF | CSRF tokens, SameSite cookies |
| Session Hijacking | Short-lived tokens, secure cookies |
| Webhook Spoofing | HMAC signatures |
| Insider Threat | Audit logs, least privilege |

### 7.2 Compliance

- **GDPR**: Data export, right to deletion, consent management
- **LGPD**: Data residency (Brazil), privacy policy
- **SOC 2** (Future): Security controls, audit trail
- **ISO 27001** (Future): Information security management

---

## 8. Success Metrics

### 8.1 Product Metrics

| Metric | Target (6 months) |
|--------|-------------------|
| Active Tenants | 100+ |
| Active Instances | 500+ |
| Messages/day | 50k+ |
| Dashboard MAU | 200+ |
| API Uptime | 99.5%+ |
| NPS Score | 40+ |

### 8.2 Business Metrics

| Metric | Target |
|--------|--------|
| MRR (Monthly Recurring Revenue) | $5k+ |
| CAC (Customer Acquisition Cost) | < $50 |
| LTV (Lifetime Value) | > $500 |
| Churn Rate | < 5% |
| Conversion Free → Paid | 15%+ |

---

## 9. Pricing Tiers (SaaS Model)

### Free Tier
- 1 instance
- 1000 messages/month
- Community support
- **Price: $0**

### Starter
- 3 instances
- 10k messages/month
- Email support
- Webhook logs
- **Price: $29/month**

### Business
- 10 instances
- 100k messages/month
- Priority support
- Custom webhooks
- Analytics
- **Price: $99/month**

### Enterprise
- Unlimited instances
- Unlimited messages
- SLA 99.9%
- Dedicated support
- Custom deployment
- **Price: Custom**

---

## 10. Roadmap

### Phase 1: MVP 2.0 (8 weeks)

**Weeks 1-2: Foundation**
- Multi-tenant database schema
- OAuth2 implementation
- Instance manager refactor

**Weeks 3-4: API**
- Meta-compatible endpoints
- Webhook system redesign
- Rate limiting per tenant

**Weeks 5-6: Dashboard**
- React dashboard
- Instance management UI
- QR code display
- Webhook configuration

**Weeks 7-8: Testing & Deploy**
- Integration tests
- Security audit
- Load testing
- Production deployment

### Phase 2: Growth (12 weeks)

- Interactive messages (lists)
- Template management
- Business profile API
- Analytics dashboard
- Mobile app (React Native)

### Phase 3: Scale (16 weeks)

- Kubernetes deployment
- Multi-region support
- Advanced analytics
- White-label option
- Marketplace/plugins

---

## 11. Risks & Mitigations

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| WhatsApp bans accounts | MEDIUM | CRITICAL | Educate users, rate limiting, ToS acceptance |
| Protocol changes | MEDIUM | HIGH | Monitor whatsmeow updates, automated tests |
| Security breach | LOW | CRITICAL | Penetration testing, bug bounty, insurance |
| Scaling issues | MEDIUM | MEDIUM | Load testing, auto-scaling, caching |
| Legal issues (Meta) | LOW | CRITICAL | Disclaimer, encourage Meta migration, legal review |
| Competitor launch | HIGH | MEDIUM | Fast iteration, unique features (AI integration) |

---

## 12. Open Questions

1. **Webhook HMAC Signatures**: Implementar assinatura HMAC como Meta API?
2. **Media Storage**: S3 ou local filesystem? CDN?
3. **Session Persistence**: Redis ou PostgreSQL para whatsmeow sessions?
4. **Kubernetes**: Necessário no MVP ou pode usar Docker Swarm?
5. **Multi-Region**: Suporte desde o início ou Phase 2?
6. **AI Integration**: Incluir agent IA integrado ou apenas documentar como conectar?

---

## 13. Appendix

### A. References

- Meta WhatsApp Cloud API: https://developers.facebook.com/docs/whatsapp/cloud-api/
- whatsmeow: https://github.com/tulir/whatsmeow
- OAuth2 RFC: https://datatracker.ietf.org/doc/html/rfc6749
- GDPR: https://gdpr.eu/
- LGPD: https://www.gov.br/esporte/pt-br/acesso-a-informacao/lgpd

### B. Glossary

- **Tenant**: Cliente/empresa usando a plataforma
- **Instance**: Número WhatsApp conectado (phone_number_id)
- **Session**: Conexão whatsmeow persistente
- **Webhook**: HTTP callback para enviar eventos
- **QR Code**: Método de autenticação visual
- **Pair Code**: Código de 8 dígitos para autenticação

---

**Document Version History:**

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2025-01-18 | BMAD PM | Initial PRD |

**Approval:**

- [ ] Product Manager
- [ ] Engineering Lead
- [ ] Security Lead
- [ ] Legal Review
