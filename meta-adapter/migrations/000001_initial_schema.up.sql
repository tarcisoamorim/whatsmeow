-- Initial schema for WhatsApp Meta API Adapter
-- Version: 1.0.0
-- Description: Creates core tables for multi-tenant WhatsApp API

BEGIN;

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Create update_updated_at_column function for triggers
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- TENANTS TABLE
-- =====================================================
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,

    -- Authentication
    password_hash VARCHAR(255),
    email_verified BOOLEAN DEFAULT FALSE NOT NULL,
    email_verification_token VARCHAR(255),
    email_verification_expires_at TIMESTAMP WITH TIME ZONE,

    -- OAuth2 Configuration
    oauth_client_id VARCHAR(255) UNIQUE,
    oauth_client_secret_hash VARCHAR(255),
    oauth_redirect_uris TEXT[],

    -- API Key (Legacy/Fallback)
    api_key_hash VARCHAR(255) UNIQUE,
    api_key_prefix VARCHAR(16),

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
    company_document VARCHAR(50),
    country_code CHAR(2) DEFAULT 'BR',
    timezone VARCHAR(50) DEFAULT 'America/Sao_Paulo',
    language VARCHAR(10) DEFAULT 'pt_BR',

    -- Contact
    phone VARCHAR(20),
    address JSONB,

    -- Preferences
    preferences JSONB DEFAULT '{}' NOT NULL,
    feature_flags JSONB DEFAULT '{}' NOT NULL,

    -- Security
    two_factor_enabled BOOLEAN DEFAULT FALSE NOT NULL,
    two_factor_secret VARCHAR(255),
    allowed_ip_ranges INET[],
    webhook_secret VARCHAR(255),

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

-- Indexes for tenants
CREATE INDEX idx_tenants_email ON tenants(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenants_api_key_prefix ON tenants(api_key_prefix) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenants_oauth_client_id ON tenants(oauth_client_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenants_status ON tenants(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenants_tier ON tenants(tier);
CREATE INDEX idx_tenants_stripe_customer_id ON tenants(stripe_customer_id) WHERE stripe_customer_id IS NOT NULL;

-- Trigger for tenants
CREATE TRIGGER update_tenants_updated_at BEFORE UPDATE ON tenants
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE tenants IS 'Customer accounts (organizations/businesses)';

-- =====================================================
-- INSTANCES TABLE (Partitioned by tenant_id)
-- =====================================================
CREATE TABLE instances (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    id UUID NOT NULL DEFAULT gen_random_uuid(),

    -- WhatsApp Identity
    phone_number VARCHAR(20) UNIQUE,
    phone_number_id VARCHAR(255) UNIQUE NOT NULL,
    display_name VARCHAR(255),
    business_profile JSONB,

    -- Connection State
    status VARCHAR(50) DEFAULT 'disconnected' NOT NULL,
    connection_state VARCHAR(50) DEFAULT 'disconnected' NOT NULL,
    qr_code TEXT,
    qr_code_expires_at TIMESTAMP WITH TIME ZONE,
    pairing_code VARCHAR(20),
    pairing_code_expires_at TIMESTAMP WITH TIME ZONE,

    -- WhatsApp Session Data (encrypted)
    session_jid VARCHAR(255),
    device_id INTEGER,
    registration_id INTEGER,
    noise_key_public BYTEA,
    noise_key_private BYTEA,
    identity_key_public BYTEA,
    identity_key_private BYTEA,
    signed_pre_key BYTEA,
    signed_pre_key_id INTEGER,

    -- Instance Manager Assignment
    manager_id VARCHAR(100),
    manager_host VARCHAR(255),
    manager_assigned_at TIMESTAMP WITH TIME ZONE,

    -- Configuration
    webhook_url TEXT,
    webhook_events TEXT[],
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
    labels JSONB DEFAULT '{}',
    metadata JSONB DEFAULT '{}',

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

-- Indexes for instances
CREATE INDEX idx_instances_phone_number ON instances(phone_number) WHERE deleted_at IS NULL;
CREATE INDEX idx_instances_phone_number_id ON instances(phone_number_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_instances_status ON instances(tenant_id, status);
CREATE INDEX idx_instances_manager_id ON instances(manager_id) WHERE manager_id IS NOT NULL;
CREATE INDEX idx_instances_last_seen ON instances(last_seen_at DESC);

-- Trigger for instances
CREATE TRIGGER update_instances_updated_at BEFORE UPDATE ON instances
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE instances IS 'WhatsApp phone number instances (multi-tenant)';

-- =====================================================
-- OAUTH CLIENTS TABLE
-- =====================================================
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
    access_token_lifetime INTEGER DEFAULT 3600 NOT NULL,
    refresh_token_lifetime INTEGER DEFAULT 2592000 NOT NULL,
    authorization_code_lifetime INTEGER DEFAULT 600 NOT NULL,

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

-- Indexes for oauth_clients
CREATE UNIQUE INDEX idx_oauth_clients_client_id ON oauth_clients(client_id);
CREATE INDEX idx_oauth_clients_tenant ON oauth_clients(tenant_id, status);

-- Trigger
CREATE TRIGGER update_oauth_clients_updated_at BEFORE UPDATE ON oauth_clients
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE oauth_clients IS 'OAuth2 client applications';

-- =====================================================
-- OAUTH TOKENS TABLE
-- =====================================================
CREATE TABLE oauth_tokens (
    tenant_id UUID NOT NULL,
    client_id UUID NOT NULL,
    id UUID NOT NULL DEFAULT gen_random_uuid(),

    -- Token Identity
    token_type VARCHAR(50) NOT NULL,
    token_hash VARCHAR(255) UNIQUE NOT NULL,

    -- Scopes & Context
    scopes TEXT[] NOT NULL,
    instance_id UUID,

    -- Lifecycle
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    revoked BOOLEAN DEFAULT FALSE NOT NULL,
    revoked_at TIMESTAMP WITH TIME ZONE,
    revoked_reason VARCHAR(255),

    -- Refresh Token Chain
    refresh_token_id UUID,

    -- Audit
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    last_used_at TIMESTAMP WITH TIME ZONE,

    PRIMARY KEY (tenant_id, client_id, id),
    FOREIGN KEY (tenant_id, client_id) REFERENCES oauth_clients(tenant_id, id) ON DELETE CASCADE,

    CONSTRAINT oauth_tokens_type_check CHECK (token_type IN ('access_token', 'refresh_token', 'authorization_code'))
);

-- Indexes for oauth_tokens
CREATE UNIQUE INDEX idx_oauth_tokens_hash ON oauth_tokens(token_hash) WHERE NOT revoked;
CREATE INDEX idx_oauth_tokens_expires ON oauth_tokens(expires_at) WHERE NOT revoked;
CREATE INDEX idx_oauth_tokens_client ON oauth_tokens(tenant_id, client_id) WHERE NOT revoked;

COMMENT ON TABLE oauth_tokens IS 'OAuth2 tokens (access, refresh, authorization codes)';

-- =====================================================
-- RATE LIMITS TABLE
-- =====================================================
CREATE TABLE rate_limits (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    id UUID NOT NULL DEFAULT gen_random_uuid(),

    -- Limit Scope
    scope VARCHAR(50) NOT NULL,
    resource_id VARCHAR(255),

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

COMMIT;
