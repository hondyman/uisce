ALTER TABLE public.private_markets_sessions DROP CONSTRAINT private_markets_sessions_user_id_fkey;
ALTER TABLE public.private_markets_sessions ADD CONSTRAINT private_markets_sessions_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;
