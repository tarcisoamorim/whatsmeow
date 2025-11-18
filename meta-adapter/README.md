# WhatsApp Meta API Adapter

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MPL%202.0-blue.svg)](https://opensource.org/licenses/MPL-2.0)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=flat&logo=docker)](https://www.docker.com/)

> **⚠️ DISCLAIMER**: Este projeto é uma implementação não oficial e experimental. Não é afiliado, aprovado ou suportado pela Meta/WhatsApp. O uso de automação no WhatsApp Web pode violar os Termos de Serviço do WhatsApp e resultar em banimento da conta. Use por sua própria conta e risco, apenas para fins de desenvolvimento e teste.

## 📋 Índice

- [Sobre](#sobre)
- [Características](#características)
- [Arquitetura](#arquitetura)
- [Início Rápido](#início-rápido)
- [Uso](#uso)
- [API Reference](#api-reference)
- [Docker](#docker)
- [Configuração](#configuração)
- [Limitações](#limitações)
- [Desenvolvimento](#desenvolvimento)
- [Roadmap](#roadmap)
- [Contribuindo](#contribuindo)
- [Licença](#licença)

## 🎯 Sobre

O **WhatsApp Meta API Adapter** é uma camada de compatibilidade que permite que sistemas projetados para a [WhatsApp Business Cloud API oficial da Meta](https://developers.facebook.com/docs/whatsapp/cloud-api/) possam funcionar usando o WhatsApp Web através da biblioteca [whatsmeow](https://github.com/tulir/whatsmeow).

### Por que usar?

- **💰 Economia**: Evite custos por mensagem da API oficial da Meta
- **🔓 Sem aprovação**: Não requer aprovação comercial da Meta
- **✨ Features extras**: Acesso a funcionalidades não disponíveis na API oficial (grupos completos, reações, edições)
- **🏠 Self-hosted**: Controle total sobre infraestrutura e dados
- **🔌 Compatibilidade**: Drop-in replacement para sistemas existentes que usam Meta API

### Quando NÃO usar?

- ❌ Aplicações críticas de negócio (bancos, saúde)
- ❌ Alto volume (>50k mensagens/dia)
- ❌ Requisitos de SLA estritos
- ❌ Necessidade de compliance regulatório
- ❌ Marketing em massa / spam

## ✨ Características

### API Compatível

- ✅ **Envio de mensagens de texto**
- ✅ **Mídia** (imagens, vídeos, áudio, documentos)
- ✅ **Localização**
- ✅ **Contatos**
- ✅ **Reações**
- ✅ **Webhooks** para mensagens recebidas e status de entrega
- ✅ **Autenticação via QR Code**
- ✅ **Respostas (replies)**

### Features Técnicas

- 🚀 **Alta Performance** com Go e Fiber
- 🔒 **Segurança** com autenticação via API Key
- 📊 **Rate Limiting** configurável
- 🐳 **Docker-ready** com docker-compose
- 📝 **Logging estruturado** com zerolog
- 💾 **Persistência** com SQLite
- 🔄 **Auto-reconnect** WhatsApp
- 📡 **Webhooks** com retry automático
- 🎯 **CORS** configurável

## 🏗️ Arquitetura

```
┌─────────────────────────────────────┐
│   Cliente (App/Sistema existente)  │
└──────────────┬──────────────────────┘
               │ HTTP/JSON (Meta API compatible)
               ↓
┌──────────────────────────────────────────────┐
│         🌐 API Gateway (Fiber)               │
│  • POST /api/v1/messages                     │
│  • GET  /api/v1/session/qr                   │
│  • GET  /api/v1/session/status               │
└──────────────┬───────────────────────────────┘
               │
               ↓
┌──────────────────────────────────────────────┐
│      🔄 Adapter Layer (Translator)           │
│  • Meta JSON → whatsmeow Protobuf            │
│  • Media upload/download                     │
│  • Webhook formatting                        │
└──────────────┬───────────────────────────────┘
               │
               ↓
┌──────────────────────────────────────────────┐
│       📚 whatsmeow Client                    │
│  • WhatsApp Web protocol                     │
│  • Signal Protocol E2E encryption            │
│  • Multi-device support                      │
└──────────────┬───────────────────────────────┘
               │ WebSocket + Noise Protocol
               ↓
┌──────────────────────────────────────────────┐
│            ☁️  WhatsApp Servers               │
└──────────────────────────────────────────────┘
```

## 🚀 Início Rápido

### Pré-requisitos

- Go 1.24+ (para desenvolvimento)
- Docker & Docker Compose (para produção)
- Uma conta WhatsApp (não WhatsApp Business)

### Opção 1: Docker (Recomendado)

```bash
# 1. Clone o repositório
git clone https://github.com/tarcisoamorim/whatsmeow.git
cd whatsmeow/meta-adapter

# 2. Configure variáveis de ambiente
cp .env.example .env
# Edite .env e configure API_KEY

# 3. Inicie com Docker Compose
make quickstart

# Ou manualmente:
docker-compose -f docker/docker-compose.yml up -d
```

### Opção 2: Build Local

```bash
# 1. Clone e entre no diretório
cd whatsmeow/meta-adapter

# 2. Instale dependências
make install

# 3. Configure .env
cp .env.example .env

# 4. Execute
make run
```

### Primeiro Uso

1. **Obtenha o QR Code**:

```bash
curl http://localhost:8080/api/v1/session/qr
```

2. **Escaneie com WhatsApp**:
   - Abra WhatsApp no celular
   - Vá em Configurações → Aparelhos Conectados
   - Toque em "Conectar aparelho"
   - Escaneie o QR code retornado pela API

3. **Envie sua primeira mensagem**:

```bash
curl -X POST http://localhost:8080/api/v1/messages \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "messaging_product": "whatsapp",
    "to": "5521999999999",
    "type": "text",
    "text": {
      "body": "Hello from Meta Adapter! 🚀"
    }
  }'
```

## 📖 Uso

### Enviar Mensagem de Texto

```bash
POST /api/v1/messages

{
  "messaging_product": "whatsapp",
  "to": "5521999999999",
  "type": "text",
  "text": {
    "body": "Olá! Esta é uma mensagem de teste."
  }
}
```

### Enviar Imagem

```bash
POST /api/v1/messages

{
  "messaging_product": "whatsapp",
  "to": "5521999999999",
  "type": "image",
  "image": {
    "link": "https://example.com/image.jpg",
    "caption": "Confira esta imagem!"
  }
}
```

### Enviar Localização

```bash
POST /api/v1/messages

{
  "messaging_product": "whatsapp",
  "to": "5521999999999",
  "type": "location",
  "location": {
    "latitude": -23.5505,
    "longitude": -46.6333,
    "name": "São Paulo",
    "address": "São Paulo, SP, Brasil"
  }
}
```

### Enviar Reação

```bash
POST /api/v1/messages

{
  "messaging_product": "whatsapp",
  "to": "5521999999999",
  "type": "reaction",
  "reaction": {
    "message_id": "MESSAGE_ID_AQUI",
    "emoji": "👍"
  }
}
```

### Responder Mensagem

```bash
POST /api/v1/messages

{
  "messaging_product": "whatsapp",
  "to": "5521999999999",
  "type": "text",
  "text": {
    "body": "Esta é uma resposta"
  },
  "context": {
    "message_id": "MESSAGE_ID_PARA_RESPONDER"
  }
}
```

## 📚 API Reference

### Autenticação

Todas as requisições (exceto `/health`) requerem autenticação via header:

```
Authorization: Bearer YOUR_API_KEY
```

Ou:

```
X-API-Key: YOUR_API_KEY
```

### Endpoints

| Método | Endpoint | Descrição |
|--------|----------|-----------|
| GET | `/health` | Health check (sem auth) |
| GET | `/api/v1/session/qr` | Gera QR code para pareamento |
| GET | `/api/v1/session/status` | Status da sessão atual |
| POST | `/api/v1/session/logout` | Desconecta sessão |
| POST | `/api/v1/messages` | Envia mensagem |
| POST | `/api/v1/media` | Upload de mídia |
| GET | `/api/v1/:media_id` | Download de mídia |

### Webhooks

Configure `WEBHOOK_URL` no `.env` para receber eventos:

#### Mensagem Recebida

```json
{
  "object": "whatsapp_business_account",
  "entry": [{
    "id": "PHONE_NUMBER_ID",
    "changes": [{
      "value": {
        "messaging_product": "whatsapp",
        "metadata": {
          "display_phone_number": "5521999999999",
          "phone_number_id": "PHONE_NUMBER_ID"
        },
        "contacts": [{
          "profile": {"name": "Nome do Contato"},
          "wa_id": "5521888888888"
        }],
        "messages": [{
          "from": "5521888888888",
          "id": "wamid.XXX",
          "timestamp": "1234567890",
          "type": "text",
          "text": {"body": "Olá!"}
        }]
      },
      "field": "messages"
    }]
  }]
}
```

#### Status de Entrega

```json
{
  "object": "whatsapp_business_account",
  "entry": [{
    "changes": [{
      "value": {
        "statuses": [{
          "id": "wamid.XXX",
          "status": "delivered",
          "timestamp": "1234567890",
          "recipient_id": "5521999999999"
        }]
      }
    }]
  }]
}
```

## 🐳 Docker

### Build da Imagem

```bash
make docker-build
```

### Executar com Docker Compose

```bash
# Iniciar
make docker-up

# Ver logs
make docker-logs

# Parar
make docker-down

# Reiniciar
make docker-restart

# Limpar tudo
make docker-clean
```

### Volumes

O Docker Compose cria um volume persistente para:
- Sessões do WhatsApp (`/app/data/sessions`)
- Banco de dados SQLite (`/app/data/adapter.db`)
- Arquivos temporários (`/app/data/temp`)

## ⚙️ Configuração

Todas as configurações via variáveis de ambiente (arquivo `.env`):

### Server

- `PORT` - Porta do servidor (padrão: 8080)
- `HOST` - Host do servidor (padrão: 0.0.0.0)
- `ENVIRONMENT` - Ambiente (development/production)
- `API_VERSION` - Versão da API (padrão: v1)
- `API_BASE_PATH` - Caminho base da API (padrão: /api/v1)

### WhatsApp

- `WHATSAPP_STORE_PATH` - Caminho para armazenamento de sessões
- `WHATSAPP_AUTO_RECONNECT` - Auto-reconectar (true/false)

### Webhook

- `WEBHOOK_URL` - URL para receber webhooks
- `WEBHOOK_VERIFY_TOKEN` - Token de verificação
- `WEBHOOK_TIMEOUT` - Timeout para webhooks (ex: 30s)
- `WEBHOOK_RETRY_ATTEMPTS` - Tentativas de retry (padrão: 3)

### Segurança

- `API_KEY` - Chave de API única
- `API_KEYS` - Múltiplas chaves separadas por vírgula (multi-tenant)

### Rate Limiting

- `RATE_LIMIT_ENABLED` - Ativar rate limiting (true/false)
- `RATE_LIMIT_REQUESTS_PER_MINUTE` - Requisições por minuto
- `RATE_LIMIT_BURST` - Burst permitido

### Logging

- `LOG_LEVEL` - Nível de log (debug/info/warn/error)
- `LOG_FORMAT` - Formato (json/console)

### Outros

- `DB_PATH` - Caminho do banco SQLite
- `CORS_ENABLED` - Ativar CORS
- `CORS_ALLOWED_ORIGINS` - Origens permitidas (* para todas)
- `MEDIA_MAX_SIZE_MB` - Tamanho máximo de mídia (MB)

## ⚠️ Limitações

### Funcionalidades Não Suportadas

- ❌ **Templates de mensagens** (com aprovação Meta)
- ❌ **Catálogos e Carrinhos** (Commerce API)
- ❌ **Pagamentos integrados**
- ❌ **Analytics oficiais**
- ❌ **Tier system / billing** da Meta
- ❌ **Botões interativos complexos** (parcialmente suportado)

### Limitações Técnicas

- ⚠️ **Protocolo pode mudar** sem aviso (depende de reverse engineering)
- ⚠️ **Risco de ban** (uso não autorizado do WhatsApp Web)
- ⚠️ **Não recomendado para alto volume** (>10k msgs/dia)
- ⚠️ **Sem SLA oficial** ou suporte da Meta
- ⚠️ **Rate limits não documentados** (podem variar)

### Diferenças da API Oficial

| Feature | API Meta Oficial | Este Adapter |
|---------|------------------|--------------|
| Custo | Pago por mensagem | Gratuito (risco de ban) |
| Aprovação | Requerida | Não requerida |
| Templates | Aprovação Meta | Não suportado |
| Grupos | Envio limitado | Gestão completa |
| Reações | ✅ | ✅ |
| Edições | ❌ | ✅ |
| Polls | ❌ | ✅ |
| Catálogos | ✅ | ❌ |
| SLA | Oficial | Nenhum |
| Estabilidade | Alta | Média (dependente do protocolo) |

## 🛠️ Desenvolvimento

### Estrutura do Projeto

```
meta-adapter/
├── cmd/
│   └── server/          # Entry point
├── internal/
│   ├── api/            # HTTP handlers & routes
│   ├── adapter/        # Message translation layer
│   ├── whatsapp/       # whatsmeow wrapper
│   ├── webhook/        # Webhook dispatcher
│   ├── storage/        # Persistence layer
│   └── config/         # Configuration
├── pkg/
│   ├── models/         # Data models (Meta API structs)
│   └── errors/         # Error handling
├── docker/             # Docker files
├── docs/               # Documentation
└── api/                # OpenAPI spec
```

### Comandos Make

```bash
make help           # Mostra ajuda
make build          # Build binário
make run            # Executa localmente
make test           # Roda testes
make coverage       # Gera relatório de cobertura
make clean          # Limpa artefatos
make fmt            # Formata código
make lint           # Roda linter
make docker-build   # Build imagem Docker
make docker-up      # Inicia containers
make docker-down    # Para containers
make docker-logs    # Mostra logs
```

### Executar Testes

```bash
make test
```

### Gerar Coverage

```bash
make coverage
open coverage.html
```

## 🗺️ Roadmap

### MVP (Atual) ✅

- [x] Mensagens de texto
- [x] Mídia (imagem, vídeo, áudio, documento)
- [x] Localização
- [x] Contatos
- [x] Reações
- [x] Webhooks básicos
- [x] QR Code authentication
- [x] Docker support

### v1.1 (Próxima Release)

- [ ] Suporte a botões interativos
- [ ] Suporte a listas
- [ ] Templates locais (cache sem aprovação Meta)
- [ ] Gerenciamento de grupos via API
- [ ] Polls (enquetes)
- [ ] Status messages
- [ ] Rate limiting com Redis
- [ ] Métricas (Prometheus)

### v2.0 (Futuro)

- [ ] Multi-tenant (múltiplas sessões)
- [ ] Admin dashboard
- [ ] High Availability
- [ ] Backup automático de mensagens
- [ ] Analytics dashboard
- [ ] Suporte a WhatsApp Business API oficial (modo híbrido)

## 🤝 Contribuindo

Contribuições são bem-vindas! Por favor:

1. Fork o projeto
2. Crie uma branch para sua feature (`git checkout -b feature/AmazingFeature`)
3. Commit suas mudanças (`git commit -m 'Add some AmazingFeature'`)
4. Push para a branch (`git push origin feature/AmazingFeature`)
5. Abra um Pull Request

## 📄 Licença

Este projeto está licenciado sob a Mozilla Public License 2.0 - veja o arquivo [LICENSE](../LICENSE) para detalhes.

A biblioteca whatsmeow subjacente também está sob MPL 2.0.

## 🙏 Agradecimentos

- [whatsmeow](https://github.com/tulir/whatsmeow) por Tulir Asokan - biblioteca Go incrível para WhatsApp
- [Fiber](https://gofiber.io/) - framework web rápido para Go
- Comunidade Go

## ⚖️ Disclaimer Legal

Este projeto é uma implementação não oficial e **não é afiliado, aprovado ou suportado pela Meta Platforms, Inc. ou WhatsApp LLC**.

O uso de automação no WhatsApp Web pode violar os [Termos de Serviço do WhatsApp](https://www.whatsapp.com/legal/terms-of-service). O WhatsApp pode banir contas que usam clientes não oficiais.

**Use este projeto apenas para:**
- ✅ Desenvolvimento e testes
- ✅ Projetos pessoais de baixo volume
- ✅ Provas de conceito
- ✅ Ambientes de staging

**NÃO use para:**
- ❌ Spam ou marketing em massa
- ❌ Violação de privacidade
- ❌ Aplicações críticas de produção
- ❌ Atividades ilegais

Os desenvolvedores deste projeto não se responsabilizam por:
- Banimento de contas WhatsApp
- Perda de dados
- Violações de ToS
- Questões legais decorrentes do uso

**USE POR SUA PRÓPRIA CONTA E RISCO.**

---

**Made with ❤️ and Go**

Para suporte ou questões, abra uma [issue](https://github.com/tarcisoamorim/whatsmeow/issues).
