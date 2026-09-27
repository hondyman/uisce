-- Message set 9400 (mastering): a golden record version that does not exist.
INSERT INTO public.message_catalog (set_nbr, message_nbr, language_cd, severity, message_text, description, user_action) VALUES
  (9400, 41, 'en', 'Error', 'Golden record %1 has no version %2.', NULL, NULL),
  (9400, 41, 'es', 'Error', 'El registro maestro %1 no tiene la versión %2.', NULL, NULL),
  (9400, 41, 'fr', 'Error', 'L''enregistrement de référence %1 n''a pas de version %2.', NULL, NULL)
ON CONFLICT (set_nbr, message_nbr, language_cd) DO NOTHING;
