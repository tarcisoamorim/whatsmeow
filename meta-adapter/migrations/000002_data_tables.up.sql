-- Data tables for messages, webhooks, contacts, etc.
-- Version: 1.0.0

BEGIN;

-- =====================================================
-- MESSAGES TABLE (Partitioned - high volume)
-- =====================================================
CREATE TABLE messages (
    tenant_id UUID NOT NULL,
    instance_id UUID NOT NULL,
    id UUID NOT NULL DEFAULT gen_random_uuid(),

    -- Message Identity
    message_id VARCHAR(255) NOT NULL,
    wamid VARCHAR(255),

    -- Direction & Type
    direction VARCHAR(10) NOT NULL,
    type VARCHAR(50) NOT NULL,
    status VARCHAR(50) DEFAULT 'pending' NOT NULL,

    -- Participants
    from_jid VARCHAR(255) NOT NULL,
    from_phone VARCHAR(20),
    to_jid VARCHAR(255) NOT NULL,
    to_phone VARCHAR(20),

    -- Context
    context_message_id VARCHAR(255),
    context_from VARCHAR(255),
    forwarded BOOLEAN DEFAULT FALSE,
    frequently_forwarded BOOLEAN DEFAULT FALSE,

    -- Content (Encrypted at rest)
    content JSONB NOT NULL,
    content_text TEXT,

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

-- Indexes for messages
CREATE INDEX idx_messages_message_id ON messages(tenant_id, instance_id, message_id);
CREATE INDEX idx_messages_wamid ON messages(wamid) WHERE wamid IS NOT NULL;
CREATE INDEX idx_messages_timestamp ON messages(tenant_id, instance_id, timestamp DESC);
CREATE INDEX idx_messages_from_phone ON messages(tenant_id, instance_id, from_phone);
CREATE INDEX idx_messages_to_phone ON messages(tenant_id, instance_id, to_phone);
CREATE INDEX idx_messages_status ON messages(tenant_id, instance_id, status);
CREATE INDEX idx_messages_direction ON messages(tenant_id, instance_id, direction);
CREATE INDEX idx_messages_unbilled ON messages(tenant_id, instance_id) WHERE billable = TRUE AND billed_at IS NULL;

COMMENT ON TABLE messages IS 'Message audit log (all messages sent/received)';

-- =====================================================
-- WEBHOOK LOGS TABLE (Partitioned)
-- =====================================================
CREATE TABLE webhook_logs (
    tenant_id UUID NOT NULL,
    instance_id UUID NOT NULL,
    id UUID NOT NULL DEFAULT gen_random_uuid(),

    -- Message Reference
    message_id UUID,

    -- Webhook Details
    event_type VARCHAR(50) NOT NULL,
    webhook_url TEXT NOT NULL,

    -- Request
    payload JSONB NOT NULL,
    signature VARCHAR(255),
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

-- Indexes for webhook_logs
CREATE INDEX idx_webhook_logs_status ON webhook_logs(tenant_id, status, next_retry_at) WHERE status IN ('pending', 'failed');
CREATE INDEX idx_webhook_logs_message_id ON webhook_logs(tenant_id, instance_id, message_id) WHERE message_id IS NOT NULL;
CREATE INDEX idx_webhook_logs_created_at ON webhook_logs(created_at DESC);

COMMENT ON TABLE webhook_logs IS 'Webhook delivery attempts and responses';

-- =====================================================
-- API LOGS TABLE (Partitioned)
-- =====================================================
CREATE TABLE api_logs (
    tenant_id UUID NOT NULL,
    id UUID NOT NULL DEFAULT gen_random_uuid(),

    -- Request Identity
    request_id VARCHAR(100) NOT NULL,
    correlation_id VARCHAR(100),

    -- Authentication
    auth_method VARCHAR(50),
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

-- Indexes for api_logs
CREATE INDEX idx_api_logs_timestamp ON api_logs(tenant_id, timestamp DESC);
CREATE INDEX idx_api_logs_request_id ON api_logs(request_id);
CREATE INDEX idx_api_logs_path ON api_logs(tenant_id, path, timestamp DESC);
CREATE INDEX idx_api_logs_status_code ON api_logs(tenant_id, status_code) WHERE status_code >= 400;
CREATE INDEX idx_api_logs_ip_address ON api_logs(ip_address);

COMMENT ON TABLE api_logs IS 'API request audit log';

-- =====================================================
-- CONTACTS TABLE
-- =====================================================
CREATE TABLE contacts (
    tenant_id UUID NOT NULL,
    instance_id UUID NOT NULL,
    id UUID NOT NULL DEFAULT gen_random_uuid(),

    -- Identity
    jid VARCHAR(255) NOT NULL,
    phone VARCHAR(20),

    -- Profile
    display_name VARCHAR(255),
    push_name VARCHAR(255),
    business_name VARCHAR(255),
    verified_name VARCHAR(255),

    -- Contact Info
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    email VARCHAR(255),

    -- Metadata
    profile_picture_url TEXT,
    about TEXT,
    labels TEXT[],
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

-- Indexes for contacts
CREATE INDEX idx_contacts_jid ON contacts(tenant_id, instance_id, jid);
CREATE INDEX idx_contacts_phone ON contacts(phone) WHERE phone IS NOT NULL;
CREATE INDEX idx_contacts_display_name ON contacts(tenant_id, instance_id, display_name);

-- Trigger for contacts
CREATE TRIGGER update_contacts_updated_at BEFORE UPDATE ON contacts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE contacts IS 'WhatsApp contacts per instance';

-- =====================================================
-- GROUPS TABLE
-- =====================================================
CREATE TABLE groups (
    tenant_id UUID NOT NULL,
    instance_id UUID NOT NULL,
    id UUID NOT NULL DEFAULT gen_random_uuid(),

    -- Identity
    jid VARCHAR(255) NOT NULL,

    -- Group Info
    name VARCHAR(255) NOT NULL,
    description TEXT,

    -- Participants
    participant_count INTEGER DEFAULT 0,
    participants JSONB DEFAULT '[]',

    -- Metadata
    created_by_jid VARCHAR(255),
    created_at_whatsapp TIMESTAMP WITH TIME ZONE,

    -- Settings
    announce BOOLEAN DEFAULT FALSE,
    locked BOOLEAN DEFAULT FALSE,
    ephemeral_duration INTEGER,

    -- Audit
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    left_at TIMESTAMP WITH TIME ZONE,

    PRIMARY KEY (tenant_id, instance_id, id),
    FOREIGN KEY (tenant_id, instance_id) REFERENCES instances(tenant_id, id) ON DELETE CASCADE,

    CONSTRAINT groups_jid_instance_unique UNIQUE (instance_id, jid)
);

-- Indexes for groups
CREATE INDEX idx_groups_jid ON groups(tenant_id, instance_id, jid);
CREATE INDEX idx_groups_name ON groups(tenant_id, instance_id, name);

-- Trigger for groups
CREATE TRIGGER update_groups_updated_at BEFORE UPDATE ON groups
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE groups IS 'WhatsApp group chats';

-- =====================================================
-- MEDIA FILES TABLE
-- =====================================================
CREATE TABLE media_files (
    tenant_id UUID NOT NULL,
    instance_id UUID NOT NULL,
    id UUID NOT NULL DEFAULT gen_random_uuid(),

    -- File Identity
    media_id VARCHAR(255) UNIQUE NOT NULL,
    whatsapp_media_key VARCHAR(255),

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
    encryption_key BYTEA,
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

-- Indexes for media_files
CREATE UNIQUE INDEX idx_media_files_media_id ON media_files(media_id);
CREATE INDEX idx_media_files_sha256 ON media_files(sha256_hash);
CREATE INDEX idx_media_files_expires ON media_files(expires_at) WHERE expires_at IS NOT NULL;

COMMENT ON TABLE media_files IS 'Media file metadata and storage references';

-- =====================================================
-- TEMPLATES TABLE
-- =====================================================
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
    header JSONB,
    body TEXT NOT NULL,
    footer TEXT,
    buttons JSONB,

    -- Variables
    variables TEXT[],

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

-- Indexes for templates
CREATE INDEX idx_templates_name ON templates(tenant_id, instance_id, name);
CREATE INDEX idx_templates_status ON templates(tenant_id, instance_id, status);

-- Trigger for templates
CREATE TRIGGER update_templates_updated_at BEFORE UPDATE ON templates
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE templates IS 'Message templates for broadcasting';

-- =====================================================
-- INSTANCE STATS TABLE
-- =====================================================
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

-- Indexes for instance_stats
CREATE INDEX idx_instance_stats_updated ON instance_stats(updated_at DESC);

-- Trigger for instance_stats
CREATE TRIGGER update_instance_stats_updated_at BEFORE UPDATE ON instance_stats
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE instance_stats IS 'Real-time statistics per instance';

COMMIT;
