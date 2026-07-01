-- Assign alice.a@example.com to the Invesco tenant so she only sees that tenant
-- on the /fabric/tenants page and other tenant-scoped views.

DO $$
DECLARE
    alice_id text;
    invesco_tenant_id uuid := 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11';
BEGIN
    -- Look up Alice's user id in the canonical app_user table.
    SELECT id INTO alice_id FROM public.app_user WHERE email = 'alice.a@example.com';

    IF alice_id IS NULL THEN
        RAISE NOTICE 'alice.a@example.com not found in public.app_user; no assignment made';
    ELSE
        -- Many-to-many tenant assignment (used by /api/tenants/accessible).
        INSERT INTO public.user_tenant (user_id, tenant_id, access_role)
        VALUES (alice_id, invesco_tenant_id, 'tenant_admin')
        ON CONFLICT (user_id, tenant_id) DO UPDATE
        SET access_role = EXCLUDED.access_role,
            updated_at = NOW();

        -- Also keep the legacy users.tenant_id fallback in sync.
        UPDATE public.app_user
        SET tenant_id = invesco_tenant_id::text,
            updated_at = NOW()
        WHERE id = alice_id;
    END IF;
END $$;
