-- Message set 9400 (mastering): closing exceptions.
INSERT INTO public.message_catalog (set_nbr, message_nbr, language_cd, severity, message_text, description, user_action) VALUES
  (9400, 12, 'en', 'Error', '%1 is not a way to close an exception. Use RESOLVED, WAIVED or IN_REVIEW.', NULL, NULL),
  (9400, 12, 'es', 'Error', '%1 no es una forma de cerrar una excepción. Use RESOLVED, WAIVED o IN_REVIEW.', NULL, NULL),
  (9400, 12, 'fr', 'Error', '%1 n''est pas une façon de clore une exception. Utilisez RESOLVED, WAIVED ou IN_REVIEW.', NULL, NULL),
  (9400, 13, 'en', 'Error', 'Exception %1 was not found or is already closed.', NULL, NULL),
  (9400, 13, 'es', 'Error', 'No se encontró la excepción %1 o ya está cerrada.', NULL, NULL),
  (9400, 13, 'fr', 'Error', 'L''exception %1 est introuvable ou déjà close.', NULL, NULL)
ON CONFLICT (set_nbr, message_nbr, language_cd) DO NOTHING;
