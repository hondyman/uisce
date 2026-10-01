-- Message set 9400 (mastering): steward decisions on possible duplicates.
INSERT INTO public.message_catalog (set_nbr, message_nbr, language_cd, severity, message_text, description, user_action) VALUES
  (9400, 14, 'en', 'Error', 'Possible duplicate %1 was not found.', NULL, NULL),
  (9400, 14, 'es', 'Error', 'No se encontró el posible duplicado %1.', NULL, NULL),
  (9400, 14, 'fr', 'Error', 'Le doublon possible %1 est introuvable.', NULL, NULL),
  (9400, 15, 'en', 'Error', 'Possible duplicate %1 has already been decided (%2).', NULL, NULL),
  (9400, 15, 'es', 'Error', 'El posible duplicado %1 ya se decidió (%2).', NULL, NULL),
  (9400, 15, 'fr', 'Error', 'Le doublon possible %1 a déjà été traité (%2).', NULL, NULL)
ON CONFLICT (set_nbr, message_nbr, language_cd) DO NOTHING;
