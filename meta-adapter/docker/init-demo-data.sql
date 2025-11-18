-- Demo Data Initialization Script
-- Creates a demo tenant and OAuth client for quick testing

-- Create demo tenant (Free tier)
INSERT INTO tenants (id, name, email, tier, max_instances, created_at, updated_at)
VALUES (
    'demo-tenant-id-00000000-0000-0000-0000-000000000001',
    'Demo Tenant',
    'demo@example.com',
    'free',
    1,
    NOW(),
    NOW()
) ON CONFLICT (id) DO NOTHING;

-- Create demo OAuth client
INSERT INTO oauth_clients (id, tenant_id, client_secret, name, redirect_uris, scopes, created_at)
VALUES (
    'demo-client-id-00000000-0000-0000-0000-000000000001',
    'demo-tenant-id-00000000-0000-0000-0000-000000000001',
    'demo_secret_do_not_use_in_production',
    'Demo OAuth Client',
    ARRAY['http://localhost:3000/callback']::VARCHAR[],
    ARRAY['messages.send', 'messages.read', 'instances.manage']::VARCHAR[],
    NOW()
) ON CONFLICT (id) DO NOTHING;

-- Output success message
DO $$
BEGIN
    RAISE NOTICE 'Demo data initialized successfully!';
    RAISE NOTICE 'Tenant ID: demo-tenant-id-00000000-0000-0000-0000-000000000001';
    RAISE NOTICE 'Client ID: demo-client-id-00000000-0000-0000-0000-000000000001';
    RAISE NOTICE 'Client Secret: demo_secret_do_not_use_in_production';
    RAISE NOTICE '';
    RAISE NOTICE 'Test OAuth token generation:';
    RAISE NOTICE 'curl -X POST http://localhost:8080/v1/oauth/token';
    RAISE NOTICE '';
END $$;
