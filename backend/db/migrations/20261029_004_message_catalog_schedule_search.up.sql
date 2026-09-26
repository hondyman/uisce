-- Scheduler message 9200-17: a bad date in the run history filter, en/es/fr.
INSERT INTO public.message_catalog (set_nbr, message_nbr, language_cd, severity, message_text, description, user_action) VALUES
  (9200, 17, 'en', 'Error', '''%1'' is not a date. Use YYYY-MM-DD.', 'run history from/to filter failed to parse', NULL),
  (9200, 17, 'es', 'Error', '''%1'' no es una fecha. Use AAAA-MM-DD.', NULL, NULL),
  (9200, 17, 'fr', 'Error', '« %1 » n''est pas une date. Utilisez AAAA-MM-JJ.', NULL, NULL)
ON CONFLICT (set_nbr, message_nbr, language_cd) DO NOTHING;
