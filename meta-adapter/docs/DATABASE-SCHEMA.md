# Database Schema Documentation

**Version:** 1.0.0
**Database:** PostgreSQL 15+
**Last Updated:** 2025-11-18

## Table of Contents

1. [Overview](#overview)
2. [Schema Design Principles](#schema-design-principles)
3. [Core Tables](#core-tables)
4. [Table Definitions](#table-definitions)
5. [Indexes Strategy](#indexes-strategy)
6. [Partitioning Strategy](#partitioning-strategy)
7. [Data Retention Policies](#data-retention-policies)
8. [Migration Scripts](#migration-scripts)
9. [Performance Optimization](#performance-optimization)
10. [Backup and Recovery](#backup-and-recovery)

---

## 1. Overview

This document defines the complete database schema for the WhatsApp Meta API Adapter. The schema is designed to support:

- **Multi-tenancy** with complete data isolation
- **High scalability** with table partitioning
- **Audit compliance** with comprehensive logging
- **High performance** with optimized indexes
- **Data integrity** with foreign keys and constraints

### Database Statistics (Expected)

| Metric | Value |
|--------|-------|
| Total Tables | 13 |
| Partitioned Tables | 3 |
| Total Indexes | 45+ |
| Foreign Keys | 18 |
| Check Constraints | 25+ |

---

## 2. Schema Design Principles

### 2.1 Multi-Tenancy

All user data tables include `tenant_id` as the first column in the primary key for:
- Natural data partitioning
- Efficient tenant isolation
- Index optimization

```sql
-- Pattern for multi-tenant tables
CREATE TABLE example (
    tenant_id UUID NOT NULL,
    id UUID NOT NULL,
    -- other columns
    PRIMARY KEY (tenant_id, id)
) PARTITION BY HASH (tenant_id);
```

### 2.2 Audit Trail

Critical tables include audit columns:
```sql
created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
created_by UUID,
updated_by UUID
```

### 2.3 Soft Deletes

Important records use soft deletes instead of hard deletes:
```sql
deleted_at TIMESTAMP WITH TIME ZONE,
deleted_by UUID
```

### 2.4 Optimistic Locking

Tables with concurrent updates use version numbers:
```sql
version INTEGER DEFAULT 1 NOT NULL
```

---

## 3. Core Tables

### Table Relationship Diagram

```
tenants (1) ─┬─ (*) instances
             ├─ (*) oauth_clients
             ├─ (*) api_logs
             ├─ (*) rate_limits
             └─ (*) webhook_configs

instances (1) ─┬─ (*) messages
               ├─ (*) webhook_logs
               ├─ (*) contacts
               ├─ (*) groups
               ├─ (*) media_files
               ├─ (*) templates
               └─ (1) instance_stats

oauth_clients (1) ─── (*) oauth_tokens
```

---

## 4. Table Definitions

### 4.1 Tenants Table

Stores customer accounts (organizations/businesses).

```sql
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,

    -- Authentication
    password_hash VARCHAR(255), -- For admin login (bcrypt)
    email_verified BOOLEAN DEFAULT FALSE NOT NULL,
    email_verification_token VARCHAR(255),
    email_verification_expires_at TIMESTAMP WITH TIME ZONE,

    -- OAuth2 Configuration
    oauth_client_id VARCHAR(255) UNIQUE,
    oauth_client_secret_hash VARCHAR(255),
    oauth_redirect_uris TEXT[], -- Array of allowed redirect URIs

    -- API Key (Legacy/Fallback)
    api_key_hash VARCHAR(255) UNIQUE,
    api_key_prefix VARCHAR(16), -- First 8 chars for identification

    -- Subscription & Limits
    tier VARCHAR(50) DEFAULT 'free' NOT NULL,
    max_instances INTEGER DEFAULT 1 NOT NULL,
    max_messages_per_day INTEGER DEFAULT 100 NOT NULL,
    max_api_requests_per_minute INTEGER DEFAULT 30 NOT NULL,
    max_webhook_retries INTEGER DEFAULT 5 NOT NULL,

    -- Billing
    stripe_customer_id VARCHAR(255),
    subscription_status VARCHAR(50) DEFAULT 'active' NOT NULL,
    subscription_started_at TIMESTAMP WITH TIME ZONE,
    subscription_ends_at TIMESTAMP WITH TIME ZONE,
    trial_ends_at TIMESTAMP WITH TIME ZONE,

    -- Metadata
    company_name VARCHAR(255),
    company_document VARCHAR(50), -- CPF/CNPJ/Tax ID
    country_code CHAR(2) DEFAULT 'BR',
    timezone VARCHAR(50) DEFAULT 'America/Sao_Paulo',
    language VARCHAR(10) DEFAULT 'pt_BR',

    -- Contact
    phone VARCHAR(20),
    address JSONB, -- Flexible address structure

    -- Preferences
    preferences JSONB DEFAULT '{}' NOT NULL,
    feature_flags JSONB DEFAULT '{}' NOT NULL,

    -- Security
    two_factor_enabled BOOLEAN DEFAULT FALSE NOT NULL,
    two_factor_secret VARCHAR(255),
    allowed_ip_ranges INET[], -- Array of allowed IP ranges
    webhook_secret VARCHAR(255), -- For HMAC signatures

    -- Status
    status VARCHAR(50) DEFAULT 'active' NOT NULL,
    suspended_at TIMESTAMP WITH TIME ZONE,
    suspended_reason TEXT,

    -- Audit
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    last_login_at TIMESTAMP WITH TIME ZONE,
    deleted_at TIMESTAMP WITH TIME ZONE,

    CONSTRAINT tenants_tier_check CHECK (tier IN ('free', 'starter', 'business', 'enterprise')),
    CONSTRAINT tenants_status_check CHECK (status IN ('active', 'suspended', 'cancelled')),
    CONSTRAINT tenants_subscription_status_check CHECK (subscription_status IN ('active', 'trialing', 'past_due', 'cancelled', 'unpaid'))
);

-- Indexes
CREATE INDEX idx_tenants_email ON tenants(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenants_api_key_prefix ON tenants(api_key_prefix) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenants_oauth_client_id ON tenants(oauth_client_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenants_status ON tenants(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenants_tier ON tenants(tier);
CREATE INDEX idx_tenants_stripe_customer_id ON tenants(stripe_customer_id) WHERE stripe_customer_id IS NOT NULL;

-- Trigger for updated_at
CREATE TRIGGER update_tenants_updated_at BEFORE UPDATE ON tenants
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE tenants IS 'Customer accounts (organizations/businesses)';
COMMENT ON COLUMN tenants.webhook_secret IS 'Secret for signing webhook payloads sent to tenant';
COMMENT ON COLUMN tenants.allowed_ip_ranges IS 'IP whitelist for API access (optional)';
```

### 4.2 Instances Table

Stores WhatsApp phone number instances.

```sql
CREATE TABLE instances (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    id UUID NOT NULL DEFAULT gen_random_uuid(),

    -- WhatsApp Identity
    phone_number VARCHAR(20) UNIQUE, -- E.164 format: +5511999999999
    phone_number_id VARCHAR(255) UNIQUE NOT NULL, -- Our internal ID
    display_name VARCHAR(255),
    business_profile JSONB, -- Business account profile

    -- Connection State
    status VARCHAR(50) DEFAULT 'disconnected' NOT NULL,
    connection_state VARCHAR(50) DEFAULT 'disconnected' NOT NULL,
    qr_code TEXT, -- Base64 QR code for pairing
    qr_code_expires_at TIMESTAMP WITH TIME ZONE,
    pairing_code VARCHAR(20), -- 8-digit code for pairing
    pairing_code_expires_at TIMESTAMP WITH TIME ZONE,

    -- WhatsApp Session Data
    session_jid VARCHAR(255), -- WhatsApp JID (user@s.whatsapp.net)
    device_id INTEGER, -- WhatsApp device ID
    registration_id INTEGER,
    noise_key_public BYTEA,
    noise_key_private BYTEA, -- Encrypted
    identity_key_public BYTEA,
    identity_key_private BYTEA, -- Encrypted
    signed_pre_key BYTEA,
    signed_pre_key_id INTEGER,

    -- Instance Manager Assignment
    manager_id VARCHAR(100), -- Which Instance Manager process owns this
    manager_host VARCHAR(255),
    manager_assigned_at TIMESTAMP WITH TIME ZONE,

    -- Configuration
    webhook_url TEXT, -- Override tenant's default webhook
    webhook_events TEXT[], -- Array of subscribed events
    auto_reconnect BOOLEAN DEFAULT TRUE NOT NULL,
    max_reconnect_attempts INTEGER DEFAULT 10,

    -- Rate Limiting (Instance-level)
    max_messages_per_second DECIMAL(5,2) DEFAULT 1.0,
    max_messages_per_minute INTEGER DEFAULT 30,

    -- Statistics (Cached)
    total_messages_sent INTEGER DEFAULT 0 NOT NULL,
    total_messages_received INTEGER DEFAULT 0 NOT NULL,
    last_message_sent_at TIMESTAMP WITH TIME ZONE,
    last_message_received_at TIMESTAMP WITH TIME ZONE,

    -- Health
    last_seen_at TIMESTAMP WITH TIME ZONE,
    last_health_check_at TIMESTAMP WITH TIME ZONE,
    health_check_failures INTEGER DEFAULT 0,

    -- Metadata
    labels JSONB DEFAULT '{}', -- Custom labels for organization
    metadata JSONB DEFAULT '{}', -- Additional custom data

    -- Audit
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    connected_at TIMESTAMP WITH TIME ZONE,
    disconnected_at TIMESTAMP WITH TIME ZONE,
    deleted_at TIMESTAMP WITH TIME ZONE,

    PRIMARY KEY (tenant_id, id),

    CONSTRAINT instances_status_check CHECK (status IN (
        'disconnected', 'connecting', 'connected', 'failed', 'suspended'
    )),
    CONSTRAINT instances_connection_state_check CHECK (connection_state IN (
        'disconnected', 'connecting', 'connected', 'logged_out'
    ))
) PARTITION BY HASH (tenant_id);

-- Create 8 partitions for instances
CREATE TABLE instances_p0 PARTITION OF instances FOR VALUES WITH (MODULUS 8, REMAINDER 0);
CREATE TABLE instances_p1 PARTITION OF instances FOR VALUES WITH (MODULUS 8, REMAINDER 1);
CREATE TABLE instances_p2 PARTITION OF instances FOR VALUES WITH (MODULUS 8, REMAINDER 2);
CREATE TABLE instances_p3 PARTITION OF instances FOR VALUES WITH (MODULUS 8, REMAINDER 3);
CREATE TABLE instances_p4 PARTITION OF instances FOR VALUES WITH (MODULUS 8, REMAINDER 4);
CREATE TABLE instances_p5 PARTITION OF instances FOR VALUES WITH (MODULUS 8, REMAINDER 5);
CREATE TABLE instances_p6 PARTITION OF instances FOR VALUES WITH (MODULUS 8, REMAINDER 6);
CREATE TABLE instances_p7 PARTITION OF instances FOR VALUES WITH (MODULUS 8, REMAINDER 7);

-- Indexes
CREATE INDEX idx_instances_phone_number ON instances(phone_number) WHERE deleted_at IS NULL;
CREATE INDEX idx_instances_phone_number_id ON instances(phone_number_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_instances_status ON instances(tenant_id, status);
CREATE INDEX idx_instances_manager_id ON instances(manager_id) WHERE manager_id IS NOT NULL;
CREATE INDEX idx_instances_last_seen ON instances(last_seen_at DESC);

-- Trigger for updated_at
CREATE TRIGGER update_instances_updated_at BEFORE UPDATE ON instances
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE instances IS 'WhatsApp phone number instances';
COMMENT ON COLUMN instances.session_jid IS 'WhatsApp user JID (Jabber ID)';
COMMENT ON COLUMN instances.manager_id IS 'ID of Instance Manager process handling this instance';
```

### 4.3 Messages Table

Audit log of all messages sent and received.

```sql
CREATE TABLE messages (
    tenant_id UUID NOT NULL,
    instance_id UUID NOT NULL,
    id UUID NOT NULL DEFAULT gen_random_uuid(),

    -- Message Identity
    message_id VARCHAR(255) NOT NULL, -- WhatsApp message ID
    wamid VARCHAR(255), -- Meta-compatible message ID format

    -- Direction & Type
    direction VARCHAR(10) NOT NULL,
    type VARCHAR(50) NOT NULL,
    status VARCHAR(50) DEFAULT 'pending' NOT NULL,

    -- Participants
    from_jid VARCHAR(255) NOT NULL, -- WhatsApp JID
    from_phone VARCHAR(20), -- E.164 normalized
    to_jid VARCHAR(255) NOT NULL,
    to_phone VARCHAR(20),

    -- Context
    context_message_id VARCHAR(255), -- Reply/quote context
    context_from VARCHAR(255),
    forwarded BOOLEAN DEFAULT FALSE,
    frequently_forwarded BOOLEAN DEFAULT FALSE,

    -- Content (Encrypted at rest)
    content JSONB NOT NULL, -- Full message content
    content_text TEXT, -- Extracted text for search (denormalized)

    -- Media
    media_id VARCHAR(255),
    media_url TEXT,
    media_mime_type VARCHAR(100),
    media_sha256 VARCHAR(64),
    media_size BIGINT,

    -- Delivery Status
    sent_at TIMESTAMP WITH TIME ZONE,
    delivered_at TIMESTAMP WITH TIME ZONE,
    read_at TIMESTAMP WITH TIME ZONE,
    failed_at TIMESTAMP WITH TIME ZONE,
    error_code VARCHAR(50),
    error_message TEXT,

    -- Metadata
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    received_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,

    -- Billing
    billable BOOLEAN DEFAULT TRUE NOT NULL,
    billed_at TIMESTAMP WITH TIME ZONE,

    PRIMARY KEY (tenant_id, instance_id, id),
    FOREIGN KEY (tenant_id, instance_id) REFERENCES instances(tenant_id, id) ON DELETE CASCADE,

    CONSTRAINT messages_direction_check CHECK (direction IN ('inbound', 'outbound')),
    CONSTRAINT messages_type_check CHECK (type IN (
        'text', 'image', 'video', 'audio', 'document', 'sticker',
        'location', 'contacts', 'reaction', 'interactive', 'template',
        'system', 'unknown'
    )),
    CONSTRAINT messages_status_check CHECK (status IN (
        'pending', 'sent', 'delivered', 'read', 'failed', 'deleted'
    ))
) PARTITION BY HASH (tenant_id);

-- Create 16 partitions for messages (high volume)
CREATE TABLE messages_p00 PARTITION OF messages FOR VALUES WITH (MODULUS 16, REMAINDER 0);
CREATE TABLE messages_p01 PARTITION OF messages FOR VALUES WITH (MODULUS 16, REMAINDER 1);
CREATE TABLE messages_p02 PARTITION OF messages FOR VALUES WITH (MODULUS 16, REMAINDER 2);
CREATE TABLE messages_p03 PARTITION OF messages FOR VALUES WITH (MODULUS 16, REMAINDER 3);
CREATE TABLE messages_p04 PARTITION OF messages FOR VALUES WITH (MODULUS 16, REMAINDER 4);
CREATE TABLE messages_p05 PARTITION OF messages FOR VALUES WITH (MODULUS 16, REMAINDER 5);
CREATE TABLE messages_p06 PARTITION OF messages FOR VALUES WITH (MODULUS 16, REMAINDER 6);
CREATE TABLE messages_p07 PARTITION OF messages FOR VALUES WITH (MODULUS 16, REMAINDER 7);
CREATE TABLE messages_p08 PARTITION OF messages FOR VALUES WITH (MODULUS 16, REMAINDER 8);
CREATE TABLE messages_p09 PARTITION OF messages FOR VALUES WITH (MODULUS 16, REMAINDER 9);
CREATE TABLE messages_p10 PARTITION OF messages FOR VALUES WITH (MODULUS 16, REMAINDER 10);
CREATE TABLE messages_p11 PARTITION OF messages FOR VALUES WITH (MODULUS 16, REMAINDER 11);
CREATE TABLE messages_p12 PARTITION OF messages FOR VALUES WITH (MODULUS 16, REMAINDER 12);
CREATE TABLE messages_p13 PARTITION OF messages FOR VALUES WITH (MODULUS 16, REMAINDER 13);
CREATE TABLE messages_p14 PARTITION OF messages FOR VALUES WITH (MODULUS 16, REMAINDER 14);
CREATE TABLE messages_p15 PARTITION OF messages FOR VALUES WITH (MODULUS 16, REMAINDER 15);

-- Indexes
CREATE INDEX idx_messages_message_id ON messages(tenant_id, instance_id, message_id);
CREATE INDEX idx_messages_wamid ON messages(wamid) WHERE wamid IS NOT NULL;
CREATE INDEX idx_messages_timestamp ON messages(tenant_id, instance_id, timestamp DESC);
CREATE INDEX idx_messages_from_phone ON messages(tenant_id, instance_id, from_phone);
CREATE INDEX idx_messages_to_phone ON messages(tenant_id, instance_id, to_phone);
CREATE INDEX idx_messages_status ON messages(tenant_id, instance_id, status);
CREATE INDEX idx_messages_direction ON messages(tenant_id, instance_id, direction);
CREATE INDEX idx_messages_content_text ON messages USING gin(to_tsvector('portuguese', content_text)) WHERE content_text IS NOT NULL;
CREATE INDEX idx_messages_unbilled ON messages(tenant_id, instance_id) WHERE billable = TRUE AND billed_at IS NULL;

COMMENT ON TABLE messages IS 'Audit log of all messages sent and received';
COMMENT ON COLUMN messages.wamid IS 'Meta WhatsApp Message ID format (wamid.*)';
COMMENT ON COLUMN messages.content IS 'Full message payload (JSON)';
COMMENT ON COLUMN messages.billable IS 'Whether this message counts toward billing quota';
```

### 4.4 Webhook Logs Table

Tracks webhook delivery attempts.

```sql
CREATE TABLE webhook_logs (
    tenant_id UUID NOT NULL,
    instance_id UUID NOT NULL,
    id UUID NOT NULL DEFAULT gen_random_uuid(),

    -- Message Reference
    message_id UUID, -- Reference to messages table

    -- Webhook Details
    event_type VARCHAR(50) NOT NULL,
    webhook_url TEXT NOT NULL,

    -- Request
    payload JSONB NOT NULL,
    signature VARCHAR(255), -- HMAC signature
    attempt INTEGER DEFAULT 1 NOT NULL,

    -- Response
    status_code INTEGER,
    response_body TEXT,
    response_time_ms INTEGER,

    -- Status
    status VARCHAR(50) DEFAULT 'pending' NOT NULL,
    sent_at TIMESTAMP WITH TIME ZONE,
    next_retry_at TIMESTAMP WITH TIME ZONE,
    failed_at TIMESTAMP WITH TIME ZONE,
    error TEXT,

    -- Audit
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,

    PRIMARY KEY (tenant_id, instance_id, id),
    FOREIGN KEY (tenant_id, instance_id) REFERENCES instances(tenant_id, id) ON DELETE CASCADE,

    CONSTRAINT webhook_logs_status_check CHECK (status IN (
        'pending', 'sent', 'delivered', 'failed', 'abandoned'
    )),
    CONSTRAINT webhook_logs_attempt_check CHECK (attempt > 0 AND attempt <= 20)
) PARTITION BY HASH (tenant_id);

-- Create 8 partitions for webhook_logs
CREATE TABLE webhook_logs_p0 PARTITION OF webhook_logs FOR VALUES WITH (MODULUS 8, REMAINDER 0);
CREATE TABLE webhook_logs_p1 PARTITION OF webhook_logs FOR VALUES WITH (MODULUS 8, REMAINDER 1);
CREATE TABLE webhook_logs_p2 PARTITION OF webhook_logs FOR VALUES WITH (MODULUS 8, REMAINDER 2);
CREATE TABLE webhook_logs_p3 PARTITION OF webhook_logs FOR VALUES WITH (MODULUS 8, REMAINDER 3);
CREATE TABLE webhook_logs_p4 PARTITION OF webhook_logs FOR VALUES WITH (MODULUS 8, REMAINDER 4);
CREATE TABLE webhook_logs_p5 PARTITION OF webhook_logs FOR VALUES WITH (MODULUS 8, REMAINDER 5);
CREATE TABLE webhook_logs_p6 PARTITION OF webhook_logs FOR VALUES WITH (MODULUS 8, REMAINDER 6);
CREATE TABLE webhook_logs_p7 PARTITION OF webhook_logs FOR VALUES WITH (MODULUS 8, REMAINDER 7);

-- Indexes
CREATE INDEX idx_webhook_logs_status ON webhook_logs(tenant_id, status, next_retry_at) WHERE status IN ('pending', 'failed');
CREATE INDEX idx_webhook_logs_message_id ON webhook_logs(tenant_id, instance_id, message_id) WHERE message_id IS NOT NULL;
CREATE INDEX idx_webhook_logs_created_at ON webhook_logs(created_at DESC);

COMMENT ON TABLE webhook_logs IS 'Webhook delivery attempts and responses';
COMMENT ON COLUMN webhook_logs.attempt IS 'Retry attempt number (1-based)';
```

### 4.5 API Logs Table

Audit log of all API requests.

```sql
CREATE TABLE api_logs (
    tenant_id UUID NOT NULL,
    id UUID NOT NULL DEFAULT gen_random_uuid(),

    -- Request Identity
    request_id VARCHAR(100) NOT NULL,
    correlation_id VARCHAR(100), -- For tracing across services

    -- Authentication
    auth_method VARCHAR(50), -- 'oauth2', 'api_key', 'none'
    auth_client_id VARCHAR(255),
    auth_scopes TEXT[],

    -- Request Details
    method VARCHAR(10) NOT NULL,
    path TEXT NOT NULL,
    query_params JSONB,
    headers JSONB,
    body JSONB,
    ip_address INET NOT NULL,
    user_agent TEXT,

    -- Response
    status_code INTEGER,
    response_body JSONB,
    response_time_ms INTEGER,

    -- Error
    error_code VARCHAR(100),
    error_message TEXT,

    -- Resource Access
    instance_id UUID,

    -- Metadata
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,

    PRIMARY KEY (tenant_id, id),
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
) PARTITION BY HASH (tenant_id);

-- Create 4 partitions for api_logs
CREATE TABLE api_logs_p0 PARTITION OF api_logs FOR VALUES WITH (MODULUS 4, REMAINDER 0);
CREATE TABLE api_logs_p1 PARTITION OF api_logs FOR VALUES WITH (MODULUS 4, REMAINDER 1);
CREATE TABLE api_logs_p2 PARTITION OF api_logs FOR VALUES WITH (MODULUS 4, REMAINDER 2);
CREATE TABLE api_logs_p3 PARTITION OF api_logs FOR VALUES WITH (MODULUS 4, REMAINDER 3);

-- Indexes
CREATE INDEX idx_api_logs_timestamp ON api_logs(tenant_id, timestamp DESC);
CREATE INDEX idx_api_logs_request_id ON api_logs(request_id);
CREATE INDEX idx_api_logs_path ON api_logs(tenant_id, path, timestamp DESC);
CREATE INDEX idx_api_logs_status_code ON api_logs(tenant_id, status_code) WHERE status_code >= 400;
CREATE INDEX idx_api_logs_ip_address ON api_logs(ip_address);

COMMENT ON TABLE api_logs IS 'Audit log of all API requests';
COMMENT ON COLUMN api_logs.correlation_id IS 'For distributed tracing';
```

### 4.6 OAuth Clients Table

OAuth2 client registrations.

```sql
CREATE TABLE oauth_clients (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    id UUID NOT NULL DEFAULT gen_random_uuid(),

    -- Client Identity
    client_id VARCHAR(255) UNIQUE NOT NULL,
    client_secret_hash VARCHAR(255) NOT NULL,
    client_name VARCHAR(255) NOT NULL,
    client_type VARCHAR(50) DEFAULT 'confidential' NOT NULL,

    -- Configuration
    redirect_uris TEXT[] NOT NULL,
    allowed_scopes TEXT[] DEFAULT ARRAY['messages.send', 'messages.read'] NOT NULL,
    grant_types TEXT[] DEFAULT ARRAY['authorization_code', 'refresh_token'] NOT NULL,

    -- Token Lifetimes (seconds)
    access_token_lifetime INTEGER DEFAULT 3600 NOT NULL, -- 1 hour
    refresh_token_lifetime INTEGER DEFAULT 2592000 NOT NULL, -- 30 days
    authorization_code_lifetime INTEGER DEFAULT 600 NOT NULL, -- 10 minutes

    -- PKCE
    require_pkce BOOLEAN DEFAULT TRUE NOT NULL,

    -- Metadata
    logo_uri TEXT,
    client_uri TEXT,
    policy_uri TEXT,
    tos_uri TEXT,

    -- Status
    status VARCHAR(50) DEFAULT 'active' NOT NULL,

    -- Audit
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    last_used_at TIMESTAMP WITH TIME ZONE,

    PRIMARY KEY (tenant_id, id),

    CONSTRAINT oauth_clients_type_check CHECK (client_type IN ('confidential', 'public')),
    CONSTRAINT oauth_clients_status_check CHECK (status IN ('active', 'revoked'))
);

-- Indexes
CREATE UNIQUE INDEX idx_oauth_clients_client_id ON oauth_clients(client_id);
CREATE INDEX idx_oauth_clients_tenant ON oauth_clients(tenant_id, status);

-- Trigger
CREATE TRIGGER update_oauth_clients_updated_at BEFORE UPDATE ON oauth_clients
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE oauth_clients IS 'OAuth2 client applications registered by tenants';
COMMENT ON COLUMN oauth_clients.require_pkce IS 'Require PKCE (Proof Key for Code Exchange)';
```

### 4.7 OAuth Tokens Table

OAuth2 access and refresh tokens.

```sql
CREATE TABLE oauth_tokens (
    tenant_id UUID NOT NULL,
    client_id UUID NOT NULL,
    id UUID NOT NULL DEFAULT gen_random_uuid(),

    -- Token Identity
    token_type VARCHAR(50) NOT NULL,
    token_hash VARCHAR(255) UNIQUE NOT NULL,

    -- Scopes & Context
    scopes TEXT[] NOT NULL,
    instance_id UUID, -- Specific instance access (optional)

    -- Lifecycle
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    revoked BOOLEAN DEFAULT FALSE NOT NULL,
    revoked_at TIMESTAMP WITH TIME ZONE,
    revoked_reason VARCHAR(255),

    -- Refresh Token Chain
    refresh_token_id UUID, -- Link to refresh token

    -- Audit
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    last_used_at TIMESTAMP WITH TIME ZONE,

    PRIMARY KEY (tenant_id, client_id, id),
    FOREIGN KEY (tenant_id, client_id) REFERENCES oauth_clients(tenant_id, id) ON DELETE CASCADE,

    CONSTRAINT oauth_tokens_type_check CHECK (token_type IN ('access_token', 'refresh_token', 'authorization_code'))
);

-- Indexes
CREATE UNIQUE INDEX idx_oauth_tokens_hash ON oauth_tokens(token_hash) WHERE NOT revoked;
CREATE INDEX idx_oauth_tokens_expires ON oauth_tokens(expires_at) WHERE NOT revoked;
CREATE INDEX idx_oauth_tokens_client ON oauth_tokens(tenant_id, client_id) WHERE NOT revoked;

COMMENT ON TABLE oauth_tokens IS 'OAuth2 access tokens, refresh tokens, and authorization codes';
COMMENT ON COLUMN oauth_tokens.instance_id IS 'Optional: restrict token to specific instance';
```

### 4.8 Rate Limits Table

Real-time rate limiting state (supplement to Redis).

```sql
CREATE TABLE rate_limits (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    id UUID NOT NULL DEFAULT gen_random_uuid(),

    -- Limit Scope
    scope VARCHAR(50) NOT NULL, -- 'api', 'messages', 'webhooks'
    resource_id VARCHAR(255), -- Optional: instance_id for granular limits

    -- Window
    window_start TIMESTAMP WITH TIME ZONE NOT NULL,
    window_duration_seconds INTEGER NOT NULL,

    -- Counters
    request_count INTEGER DEFAULT 0 NOT NULL,
    limit_threshold INTEGER NOT NULL,

    -- Violations
    exceeded BOOLEAN DEFAULT FALSE NOT NULL,
    first_exceeded_at TIMESTAMP WITH TIME ZONE,

    -- Audit
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,

    PRIMARY KEY (tenant_id, id),

    CONSTRAINT rate_limits_scope_check CHECK (scope IN ('api', 'messages', 'webhooks'))
);

-- Indexes
CREATE INDEX idx_rate_limits_window ON rate_limits(tenant_id, scope, window_start DESC);
CREATE INDEX idx_rate_limits_exceeded ON rate_limits(tenant_id, exceeded) WHERE exceeded = TRUE;

-- Trigger
CREATE TRIGGER update_rate_limits_updated_at BEFORE UPDATE ON rate_limits
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE rate_limits IS 'Rate limiting state (persistent fallback for Redis)';
COMMENT ON COLUMN rate_limits.resource_id IS 'Optional granular resource (e.g., instance_id)';
```

### 4.9 Contacts Table

WhatsApp contact book per instance.

```sql
CREATE TABLE contacts (
    tenant_id UUID NOT NULL,
    instance_id UUID NOT NULL,
    id UUID NOT NULL DEFAULT gen_random_uuid(),

    -- Identity
    jid VARCHAR(255) NOT NULL, -- WhatsApp JID
    phone VARCHAR(20), -- E.164 format

    -- Profile
    display_name VARCHAR(255),
    push_name VARCHAR(255), -- Name from WhatsApp
    business_name VARCHAR(255),
    verified_name VARCHAR(255), -- Meta verified business name

    -- Contact Info
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    email VARCHAR(255),

    -- Metadata
    profile_picture_url TEXT,
    about TEXT,
    labels TEXT[], -- Custom tags
    notes TEXT,

    -- Interaction Stats
    total_messages_sent INTEGER DEFAULT 0,
    total_messages_received INTEGER DEFAULT 0,
    last_message_at TIMESTAMP WITH TIME ZONE,

    -- Status
    blocked BOOLEAN DEFAULT FALSE NOT NULL,

    -- Audit
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,

    PRIMARY KEY (tenant_id, instance_id, id),
    FOREIGN KEY (tenant_id, instance_id) REFERENCES instances(tenant_id, id) ON DELETE CASCADE,

    CONSTRAINT contacts_jid_instance_unique UNIQUE (instance_id, jid)
);

-- Indexes
CREATE INDEX idx_contacts_jid ON contacts(tenant_id, instance_id, jid);
CREATE INDEX idx_contacts_phone ON contacts(phone) WHERE phone IS NOT NULL;
CREATE INDEX idx_contacts_display_name ON contacts(tenant_id, instance_id, display_name);

-- Trigger
CREATE TRIGGER update_contacts_updated_at BEFORE UPDATE ON contacts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE contacts IS 'WhatsApp contacts per instance';
```

### 4.10 Groups Table

WhatsApp group information.

```sql
CREATE TABLE groups (
    tenant_id UUID NOT NULL,
    instance_id UUID NOT NULL,
    id UUID NOT NULL DEFAULT gen_random_uuid(),

    -- Identity
    jid VARCHAR(255) NOT NULL, -- Group JID

    -- Group Info
    name VARCHAR(255) NOT NULL,
    description TEXT,

    -- Participants
    participant_count INTEGER DEFAULT 0,
    participants JSONB DEFAULT '[]', -- Array of {jid, role, joined_at}

    -- Metadata
    created_by_jid VARCHAR(255),
    created_at_whatsapp TIMESTAMP WITH TIME ZONE,

    -- Settings
    announce BOOLEAN DEFAULT FALSE, -- Only admins can send
    locked BOOLEAN DEFAULT FALSE, -- Only admins can edit info
    ephemeral_duration INTEGER, -- Disappearing messages (seconds)

    -- Audit
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    left_at TIMESTAMP WITH TIME ZONE,

    PRIMARY KEY (tenant_id, instance_id, id),
    FOREIGN KEY (tenant_id, instance_id) REFERENCES instances(tenant_id, id) ON DELETE CASCADE,

    CONSTRAINT groups_jid_instance_unique UNIQUE (instance_id, jid)
);

-- Indexes
CREATE INDEX idx_groups_jid ON groups(tenant_id, instance_id, jid);
CREATE INDEX idx_groups_name ON groups(tenant_id, instance_id, name);

-- Trigger
CREATE TRIGGER update_groups_updated_at BEFORE UPDATE ON groups
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE groups IS 'WhatsApp group chats';
```

### 4.11 Media Files Table

Media file metadata and storage.

```sql
CREATE TABLE media_files (
    tenant_id UUID NOT NULL,
    instance_id UUID NOT NULL,
    id UUID NOT NULL DEFAULT gen_random_uuid(),

    -- File Identity
    media_id VARCHAR(255) UNIQUE NOT NULL, -- Our media ID (Meta-compatible)
    whatsapp_media_key VARCHAR(255), -- WhatsApp's media key

    -- File Info
    filename VARCHAR(255),
    mime_type VARCHAR(100) NOT NULL,
    file_size BIGINT NOT NULL,
    sha256_hash VARCHAR(64),

    -- Storage
    storage_type VARCHAR(50) DEFAULT 's3' NOT NULL,
    storage_path TEXT NOT NULL,
    storage_url TEXT,

    -- Encryption
    encryption_key BYTEA, -- Encrypted storage key
    encrypted BOOLEAN DEFAULT FALSE NOT NULL,

    -- Media Details
    width INTEGER,
    height INTEGER,
    duration_seconds INTEGER,

    -- Status
    upload_status VARCHAR(50) DEFAULT 'pending' NOT NULL,
    downloaded BOOLEAN DEFAULT FALSE NOT NULL,

    -- Lifecycle
    expires_at TIMESTAMP WITH TIME ZONE,

    -- Audit
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    accessed_at TIMESTAMP WITH TIME ZONE,

    PRIMARY KEY (tenant_id, instance_id, id),
    FOREIGN KEY (tenant_id, instance_id) REFERENCES instances(tenant_id, id) ON DELETE CASCADE,

    CONSTRAINT media_files_storage_check CHECK (storage_type IN ('s3', 'local', 'gcs', 'azure')),
    CONSTRAINT media_files_status_check CHECK (upload_status IN ('pending', 'uploading', 'completed', 'failed'))
);

-- Indexes
CREATE UNIQUE INDEX idx_media_files_media_id ON media_files(media_id);
CREATE INDEX idx_media_files_sha256 ON media_files(sha256_hash);
CREATE INDEX idx_media_files_expires ON media_files(expires_at) WHERE expires_at IS NOT NULL;

COMMENT ON TABLE media_files IS 'Media file metadata and storage references';
COMMENT ON COLUMN media_files.media_id IS 'Public media ID for API access';
```

### 4.12 Templates Table

Message templates (similar to Meta's templates).

```sql
CREATE TABLE templates (
    tenant_id UUID NOT NULL,
    instance_id UUID NOT NULL,
    id UUID NOT NULL DEFAULT gen_random_uuid(),

    -- Template Identity
    name VARCHAR(255) NOT NULL,
    language VARCHAR(10) DEFAULT 'pt_BR' NOT NULL,

    -- Category
    category VARCHAR(50) NOT NULL,

    -- Content
    header JSONB, -- {type: 'text|image|video|document', content: '...'}
    body TEXT NOT NULL,
    footer TEXT,
    buttons JSONB, -- Array of button definitions

    -- Variables
    variables TEXT[], -- Array of variable names: ['name', 'date']

    -- Status
    status VARCHAR(50) DEFAULT 'draft' NOT NULL,
    approved_at TIMESTAMP WITH TIME ZONE,
    rejected_at TIMESTAMP WITH TIME ZONE,
    rejection_reason TEXT,

    -- Usage Stats
    sent_count INTEGER DEFAULT 0,
    last_used_at TIMESTAMP WITH TIME ZONE,

    -- Audit
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,

    PRIMARY KEY (tenant_id, instance_id, id),
    FOREIGN KEY (tenant_id, instance_id) REFERENCES instances(tenant_id, id) ON DELETE CASCADE,

    CONSTRAINT templates_name_instance_unique UNIQUE (instance_id, name, language),
    CONSTRAINT templates_category_check CHECK (category IN ('marketing', 'utility', 'authentication')),
    CONSTRAINT templates_status_check CHECK (status IN ('draft', 'pending', 'approved', 'rejected'))
);

-- Indexes
CREATE INDEX idx_templates_name ON templates(tenant_id, instance_id, name);
CREATE INDEX idx_templates_status ON templates(tenant_id, instance_id, status);

-- Trigger
CREATE TRIGGER update_templates_updated_at BEFORE UPDATE ON templates
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE templates IS 'Message templates for broadcasting';
```

### 4.13 Instance Stats Table

Real-time statistics per instance (1:1 with instances).

```sql
CREATE TABLE instance_stats (
    tenant_id UUID NOT NULL,
    instance_id UUID NOT NULL,

    -- Message Counters
    messages_sent_today INTEGER DEFAULT 0 NOT NULL,
    messages_received_today INTEGER DEFAULT 0 NOT NULL,
    messages_sent_this_month INTEGER DEFAULT 0 NOT NULL,
    messages_received_this_month INTEGER DEFAULT 0 NOT NULL,

    -- Total Counters
    total_messages_sent BIGINT DEFAULT 0 NOT NULL,
    total_messages_received BIGINT DEFAULT 0 NOT NULL,

    -- Rate Limiting
    requests_this_minute INTEGER DEFAULT 0 NOT NULL,
    requests_this_hour INTEGER DEFAULT 0 NOT NULL,

    -- Error Counters
    failed_messages_today INTEGER DEFAULT 0 NOT NULL,
    webhook_failures_today INTEGER DEFAULT 0 NOT NULL,

    -- Performance Metrics
    avg_response_time_ms DECIMAL(10,2),
    p95_response_time_ms INTEGER,

    -- Uptime
    total_uptime_seconds BIGINT DEFAULT 0 NOT NULL,
    connected_since TIMESTAMP WITH TIME ZONE,

    -- Reset Timestamps
    daily_reset_at TIMESTAMP WITH TIME ZONE DEFAULT DATE_TRUNC('day', NOW()) NOT NULL,
    monthly_reset_at TIMESTAMP WITH TIME ZONE DEFAULT DATE_TRUNC('month', NOW()) NOT NULL,
    minute_reset_at TIMESTAMP WITH TIME ZONE DEFAULT DATE_TRUNC('minute', NOW()) NOT NULL,

    -- Audit
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,

    PRIMARY KEY (tenant_id, instance_id),
    FOREIGN KEY (tenant_id, instance_id) REFERENCES instances(tenant_id, id) ON DELETE CASCADE
);

-- Indexes
CREATE INDEX idx_instance_stats_updated ON instance_stats(updated_at DESC);

-- Trigger
CREATE TRIGGER update_instance_stats_updated_at BEFORE UPDATE ON instance_stats
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE instance_stats IS 'Real-time statistics and metrics per instance';
COMMENT ON COLUMN instance_stats.daily_reset_at IS 'Timestamp when daily counters were last reset';
```

---

## 5. Indexes Strategy

### 5.1 Index Types Used

| Type | Usage | Example |
|------|-------|---------|
| **B-Tree** | Primary keys, foreign keys, range queries | `CREATE INDEX idx_name ON table(column)` |
| **Hash Partitioning** | Even data distribution across partitions | `PARTITION BY HASH(tenant_id)` |
| **Partial Index** | Filtered indexes for specific conditions | `WHERE deleted_at IS NULL` |
| **GIN** | Full-text search | `USING gin(to_tsvector('portuguese', text))` |
| **Unique** | Enforce uniqueness constraints | `UNIQUE INDEX idx_email ON tenants(email)` |

### 5.2 Index Maintenance

```sql
-- Reindex all tables (monthly maintenance)
REINDEX DATABASE whatsapp_adapter;

-- Analyze tables for query planner
ANALYZE;

-- Update table statistics
VACUUM ANALYZE;
```

---

## 6. Partitioning Strategy

### 6.1 Partitioning Rationale

| Table | Partitions | Strategy | Reason |
|-------|-----------|----------|--------|
| `instances` | 8 | Hash(tenant_id) | Moderate write volume, tenant isolation |
| `messages` | 16 | Hash(tenant_id) | Very high volume, even distribution |
| `webhook_logs` | 8 | Hash(tenant_id) | High volume, tenant queries |
| `api_logs` | 4 | Hash(tenant_id) | Moderate volume, audit queries |

### 6.2 Partition Monitoring

```sql
-- Check partition sizes
SELECT
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE tablename LIKE 'messages_p%'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
```

---

## 7. Data Retention Policies

### 7.1 Retention Rules

| Table | Retention Period | Policy |
|-------|-----------------|--------|
| `messages` | 90 days (Free tier), 1 year (Paid) | Archive to cold storage |
| `webhook_logs` | 30 days | Hard delete |
| `api_logs` | 90 days | Hard delete |
| `oauth_tokens` | 90 days after expiry | Hard delete |
| `media_files` | 30 days after last access | Delete from storage |

### 7.2 Cleanup Queries

```sql
-- Archive old messages (run daily)
INSERT INTO messages_archive
SELECT * FROM messages
WHERE created_at < NOW() - INTERVAL '90 days'
  AND tenant_id IN (SELECT id FROM tenants WHERE tier = 'free');

DELETE FROM messages
WHERE created_at < NOW() - INTERVAL '90 days'
  AND tenant_id IN (SELECT id FROM tenants WHERE tier = 'free');

-- Delete old webhook logs (run daily)
DELETE FROM webhook_logs
WHERE created_at < NOW() - INTERVAL '30 days';

-- Delete old API logs (run weekly)
DELETE FROM api_logs
WHERE timestamp < NOW() - INTERVAL '90 days';

-- Delete expired OAuth tokens (run daily)
DELETE FROM oauth_tokens
WHERE expires_at < NOW() - INTERVAL '90 days';

-- Delete expired media files (run daily)
DELETE FROM media_files
WHERE expires_at IS NOT NULL
  AND expires_at < NOW();
```

---

## 8. Migration Scripts

### 8.1 Migration 001: Initial Schema

```sql
-- migration_001_initial_schema.up.sql

BEGIN;

-- Create update_updated_at_column function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create all tables (in dependency order)
-- ... (All CREATE TABLE statements from section 4)

COMMIT;
```

```sql
-- migration_001_initial_schema.down.sql

BEGIN;

DROP TABLE IF EXISTS instance_stats CASCADE;
DROP TABLE IF EXISTS templates CASCADE;
DROP TABLE IF EXISTS media_files CASCADE;
DROP TABLE IF EXISTS groups CASCADE;
DROP TABLE IF EXISTS contacts CASCADE;
DROP TABLE IF EXISTS rate_limits CASCADE;
DROP TABLE IF EXISTS oauth_tokens CASCADE;
DROP TABLE IF EXISTS oauth_clients CASCADE;
DROP TABLE IF EXISTS api_logs CASCADE;
DROP TABLE IF EXISTS webhook_logs CASCADE;
DROP TABLE IF EXISTS messages CASCADE;
DROP TABLE IF EXISTS instances CASCADE;
DROP TABLE IF EXISTS tenants CASCADE;

DROP FUNCTION IF EXISTS update_updated_at_column();

COMMIT;
```

### 8.2 Migration 002: Add Full-Text Search

```sql
-- migration_002_add_fulltext_search.up.sql

BEGIN;

-- Add tsvector column for messages
ALTER TABLE messages ADD COLUMN content_search_vector tsvector;

-- Populate tsvector
UPDATE messages
SET content_search_vector = to_tsvector('portuguese', COALESCE(content_text, ''));

-- Create GIN index
CREATE INDEX idx_messages_search_vector ON messages USING gin(content_search_vector);

-- Create trigger to maintain tsvector
CREATE TRIGGER messages_search_vector_update BEFORE INSERT OR UPDATE ON messages
FOR EACH ROW EXECUTE FUNCTION
tsvector_update_trigger(content_search_vector, 'pg_catalog.portuguese', content_text);

COMMIT;
```

### 8.3 Migration 003: Add Tenant Tiers

```sql
-- migration_003_add_tier_features.up.sql

BEGIN;

-- Add feature flags to tenants
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS max_webhook_urls INTEGER DEFAULT 1;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS analytics_enabled BOOLEAN DEFAULT FALSE;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS priority_support BOOLEAN DEFAULT FALSE;

-- Update based on tier
UPDATE tenants SET
    max_webhook_urls = CASE tier
        WHEN 'free' THEN 1
        WHEN 'starter' THEN 3
        WHEN 'business' THEN 10
        WHEN 'enterprise' THEN 100
    END,
    analytics_enabled = tier IN ('business', 'enterprise'),
    priority_support = tier = 'enterprise';

COMMIT;
```

---

## 9. Performance Optimization

### 9.1 Query Performance Tips

```sql
-- Use prepared statements for frequently executed queries
PREPARE get_instance_messages AS
SELECT * FROM messages
WHERE tenant_id = $1 AND instance_id = $2
ORDER BY timestamp DESC
LIMIT $3;

EXECUTE get_instance_messages('tenant-uuid', 'instance-uuid', 100);

-- Use connection pooling (configured in application)
-- Recommended: pgBouncer with pool_mode = transaction

-- Enable query logging for slow queries
ALTER DATABASE whatsapp_adapter SET log_min_duration_statement = 1000; -- 1 second
```

### 9.2 Table Statistics

```sql
-- Gather extended statistics for multi-column queries
CREATE STATISTICS messages_tenant_instance_stats
ON tenant_id, instance_id FROM messages;

ANALYZE messages;
```

### 9.3 Vacuum Strategy

```sql
-- Configure autovacuum for high-churn tables
ALTER TABLE messages SET (
    autovacuum_vacuum_scale_factor = 0.05,
    autovacuum_analyze_scale_factor = 0.02
);

ALTER TABLE webhook_logs SET (
    autovacuum_vacuum_scale_factor = 0.1,
    autovacuum_analyze_scale_factor = 0.05
);
```

---

## 10. Backup and Recovery

### 10.1 Backup Strategy

| Component | Frequency | Retention | Method |
|-----------|-----------|-----------|--------|
| Full Database | Daily | 30 days | `pg_dump` |
| Incremental WAL | Continuous | 7 days | WAL archiving |
| Tenant Data | On-demand | 90 days | Selective export |
| Media Files | Daily | 30 days | S3 versioning |

### 10.2 Backup Scripts

```bash
#!/bin/bash
# backup-database.sh

DATABASE="whatsapp_adapter"
BACKUP_DIR="/backups/postgres"
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="${BACKUP_DIR}/${DATABASE}_${DATE}.sql.gz"

# Full backup
pg_dump -U postgres -d ${DATABASE} \
  --format=custom \
  --compress=9 \
  --file=${BACKUP_FILE}

# Upload to S3
aws s3 cp ${BACKUP_FILE} s3://backups-bucket/postgres/

# Cleanup old backups (keep 30 days)
find ${BACKUP_DIR} -name "${DATABASE}_*.sql.gz" -mtime +30 -delete
```

### 10.3 Recovery Procedures

```bash
# Restore from backup
pg_restore -U postgres -d whatsapp_adapter \
  --clean \
  --if-exists \
  --verbose \
  /backups/postgres/whatsapp_adapter_20251118_120000.sql.gz

# Point-in-time recovery
pg_ctl stop -D /var/lib/postgresql/data
# Copy base backup
# Configure recovery.conf with restore_command
pg_ctl start -D /var/lib/postgresql/data
```

---

## Appendix A: Database Configuration

### PostgreSQL Configuration (postgresql.conf)

```ini
# Connection settings
max_connections = 200
shared_buffers = 4GB
effective_cache_size = 12GB
work_mem = 64MB
maintenance_work_mem = 512MB

# WAL settings
wal_level = replica
max_wal_size = 4GB
min_wal_size = 1GB
checkpoint_completion_target = 0.9

# Query planning
random_page_cost = 1.1  # For SSD
effective_io_concurrency = 200

# Logging
log_min_duration_statement = 1000
log_line_prefix = '%t [%p]: [%l-1] user=%u,db=%d,app=%a,client=%h '
log_checkpoints = on
log_connections = on
log_disconnections = on
log_lock_waits = on

# Autovacuum
autovacuum = on
autovacuum_max_workers = 4
autovacuum_naptime = 10s
```

---

## Appendix B: Sample Data

### Insert Sample Tenant

```sql
INSERT INTO tenants (
    id, name, email, tier, max_instances,
    oauth_client_id, webhook_secret
) VALUES (
    '01234567-89ab-cdef-0123-456789abcdef',
    'Acme Corp',
    'admin@acme.com',
    'business',
    10,
    'acme_oauth_client',
    encode(gen_random_bytes(32), 'hex')
);
```

### Insert Sample Instance

```sql
INSERT INTO instances (
    tenant_id, id, phone_number_id, phone_number,
    status, connection_state
) VALUES (
    '01234567-89ab-cdef-0123-456789abcdef',
    gen_random_uuid(),
    '1234567890',
    '+5511999999999',
    'connected',
    'connected'
);
```

---

## Document Version History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0.0 | 2025-11-18 | Claude | Initial schema documentation |

---

**End of Database Schema Documentation**
