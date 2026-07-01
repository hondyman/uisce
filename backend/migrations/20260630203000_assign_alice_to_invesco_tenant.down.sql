-- Revert alice.a@example.com tenant assignment.

DO $$
DECLARE
    alice_id text;
    invesco_tenant_id uuid := 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11';
BEGIN
    SELECT id INTO alice_id FROM public.app_user WHERE email = 'alice.a@example.com';

    IF alice_id IS NOT NULL THEN
        DELETE FROM public.user_tenant
        WHERE user_id = alice_id AND tenant_id = invesco_tenant_id;

        UPDATE public.app_user
        SET tenant_id = NULL,
            updated_at = NOW()
        WHERE id = alice_id;
    END IF;
END $$;
