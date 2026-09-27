-- Message set 9400 (mastering): a profile's named business object binding is missing.
INSERT INTO public.message_catalog (set_nbr, message_nbr, language_cd, severity, message_text, description, user_action) VALUES
  (9400, 36, 'en', 'Error', '%1 has no binding named %2 - the mastering profile needs it to map fields to the master table.', NULL, NULL),
  (9400, 36, 'es', 'Error', '%1 no tiene ningún vínculo llamado %2; el perfil de maestrización lo necesita para asignar campos a la tabla maestra.', NULL, NULL),
  (9400, 36, 'fr', 'Error', '%1 n''a pas de liaison nommée %2 ; le profil de mastering en a besoin pour associer les champs à la table de référence.', NULL, NULL)
ON CONFLICT (set_nbr, message_nbr, language_cd) DO NOTHING;
