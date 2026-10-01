-- Enterprise schedulers (Tidal, Control-M, AutoSys, ...) drive Uisce
-- schedules: a schedule is either run on its own timetable or triggered
-- externally, and every external trigger is idempotent on a key the caller
-- supplies, so a retried call never starts a second run.

ALTER TABLE public.schedules
    ADD COLUMN IF NOT EXISTS trigger_mode text NOT NULL DEFAULT 'timetable';
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'schedules_trigger_mode_check') THEN
        ALTER TABLE public.schedules ADD CONSTRAINT schedules_trigger_mode_check
            CHECK (trigger_mode IN ('timetable', 'external'));
    END IF;
    -- An external schedule has no timetable; its calendar rule can only skip
    -- (the caller wants an answer now, never "wait for the next business day").
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'schedules_external_shape_check') THEN
        ALTER TABLE public.schedules ADD CONSTRAINT schedules_external_shape_check
            CHECK (trigger_mode = 'timetable' OR calendar_rule IN ('none', 'skip'));
    END IF;
END $$;

ALTER TABLE public.schedule_runs
    ADD COLUMN IF NOT EXISTS external_system text,
    ADD COLUMN IF NOT EXISTS external_ref text,
    ADD COLUMN IF NOT EXISTS idempotency_key text;
ALTER TABLE public.schedule_runs DROP CONSTRAINT IF EXISTS schedule_runs_trigger_check;
ALTER TABLE public.schedule_runs ADD CONSTRAINT schedule_runs_trigger_check
    CHECK (trigger IN ('schedule', 'manual', 'external'));
CREATE UNIQUE INDEX IF NOT EXISTS uq_schedule_runs_idempotency
    ON public.schedule_runs (tenant_id, schedule_id, idempotency_key) WHERE idempotency_key IS NOT NULL;

COMMENT ON COLUMN public.schedules.trigger_mode IS 'timetable: fired by its own cron; external: fired only by an enterprise scheduler through the trigger API';
COMMENT ON COLUMN public.schedule_runs.idempotency_key IS 'The external caller''s key: one run per (tenant, schedule, key), whatever the retries';

INSERT INTO public.message_catalog (set_nbr, message_nbr, language_cd, severity, message_text, description, user_action) VALUES
  (9200, 18, 'en', 'Error', 'Schedule %1 runs on its own timetable. Only externally triggered schedules can be triggered this way.', NULL, 'Switch the schedule to "Triggered by an external scheduler", or use Run now.'),
  (9200, 18, 'es', 'Error', 'La programación %1 se ejecuta con su propio horario. Solo las programaciones con disparo externo se pueden disparar así.', NULL, 'Cambie la programación a "Disparada por un planificador externo" o use Ejecutar ahora.'),
  (9200, 18, 'fr', 'Error', 'La planification %1 suit son propre horaire. Seules les planifications déclenchées de l''extérieur peuvent être déclenchées ainsi.', NULL, 'Passez la planification en « Déclenchée par un ordonnanceur externe » ou utilisez Exécuter maintenant.'),
  (9200, 19, 'en', 'Error', 'Schedule %1 is paused.', NULL, 'Resume it in the Schedules console.'),
  (9200, 19, 'es', 'Error', 'La programación %1 está en pausa.', NULL, 'Reanúdela en la consola de programaciones.'),
  (9200, 19, 'fr', 'Error', 'La planification %1 est en pause.', NULL, 'Reprenez-la dans la console des planifications.'),
  (9200, 20, 'en', 'Error', 'Send an idempotency key of 1 to 200 characters with every trigger.', 'external trigger without a usable idempotency key', NULL),
  (9200, 20, 'es', 'Error', 'Envíe una clave de idempotencia de 1 a 200 caracteres con cada disparo.', NULL, NULL),
  (9200, 20, 'fr', 'Error', 'Envoyez une clé d''idempotence de 1 à 200 caractères avec chaque déclenchement.', NULL, NULL),
  (9200, 21, 'en', 'Error', 'An externally triggered schedule can only skip days its calendar is closed.', NULL, NULL),
  (9200, 21, 'es', 'Error', 'Una programación con disparo externo solo puede omitir los días en que su calendario está cerrado.', NULL, NULL),
  (9200, 21, 'fr', 'Error', 'Une planification déclenchée de l''extérieur peut seulement ignorer les jours où son calendrier est fermé.', NULL, NULL),
  (9200, 22, 'en', 'Error', 'Service accounts can only read and trigger schedules.', NULL, NULL),
  (9200, 22, 'es', 'Error', 'Las cuentas de servicio solo pueden leer y disparar programaciones.', NULL, NULL),
  (9200, 22, 'fr', 'Error', 'Les comptes de service peuvent seulement lire et déclencher des planifications.', NULL, NULL),
  (9200, 23, 'en', 'Error', 'Nothing was triggered on schedule %1 with key %2.', NULL, NULL),
  (9200, 23, 'es', 'Error', 'No se disparó nada en la programación %1 con la clave %2.', NULL, NULL),
  (9200, 23, 'fr', 'Error', 'Rien n''a été déclenché sur la planification %1 avec la clé %2.', NULL, NULL)
ON CONFLICT (set_nbr, message_nbr, language_cd) DO NOTHING;
