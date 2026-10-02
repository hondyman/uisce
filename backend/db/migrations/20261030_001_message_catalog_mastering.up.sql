-- Message set 9400: entity mastering (internal/mastering errors.go), en/es/fr.
INSERT INTO public.message_sets (set_nbr, set_name, module, set_type, description)
VALUES (9400, 'Mastering', 'mastering', 'core', 'Entity mastering messages')
ON CONFLICT (set_nbr) DO NOTHING;

INSERT INTO public.message_catalog (set_nbr, message_nbr, language_cd, severity, message_text, description, user_action) VALUES
  (9400, 1, 'en', 'Error', 'There is no mastering profile for %1.', NULL, NULL),
  (9400, 1, 'es', 'Error', 'No existe un perfil de maestrización para %1.', NULL, NULL),
  (9400, 1, 'fr', 'Error', 'Il n''existe pas de profil de mastering pour %1.', NULL, NULL),
  (9400, 2, 'en', 'Error', '%1 is not bound to %2. Propose a staging binding first.', NULL, NULL),
  (9400, 2, 'es', 'Error', '%1 no está vinculada a %2. Proponga primero un vínculo de staging.', NULL, NULL),
  (9400, 2, 'fr', 'Error', '%1 n''est pas liée à %2. Proposez d''abord une liaison de staging.', NULL, NULL),
  (9400, 3, 'en', 'Error', 'Load run %1 was not found.', NULL, NULL),
  (9400, 3, 'es', 'Error', 'No se encontró la carga %1.', NULL, NULL),
  (9400, 3, 'fr', 'Error', 'Le chargement %1 est introuvable.', NULL, NULL),
  (9400, 4, 'en', 'Error', 'The binding for %1 has no source record key. Bind @source_key so each record can be linked to its golden record.', NULL, NULL),
  (9400, 4, 'es', 'Error', 'El vínculo de %1 no tiene clave de registro de origen. Vincule @source_key para enlazar cada registro con su registro maestro.', NULL, NULL),
  (9400, 4, 'fr', 'Error', 'La liaison de %1 n''a pas de clé d''enregistrement source. Liez @source_key pour relier chaque enregistrement à son enregistrement de référence.', NULL, NULL),
  (9400, 5, 'en', 'Error', 'Mastering is not available: the data plane is not configured.', NULL, NULL),
  (9400, 5, 'es', 'Error', 'La maestrización no está disponible: el plano de datos no está configurado.', NULL, NULL),
  (9400, 5, 'fr', 'Error', 'Le mastering n''est pas disponible : le plan de données n''est pas configuré.', NULL, NULL),
  (9400, 6, 'en', 'Error', 'Only administrators and data stewards can run mastering.', NULL, NULL),
  (9400, 6, 'es', 'Error', 'Solo los administradores y los responsables de datos pueden ejecutar la maestrización.', NULL, NULL),
  (9400, 6, 'fr', 'Error', 'Seuls les administrateurs et les responsables des données peuvent lancer le mastering.', NULL, NULL),
  (9400, 7, 'en', 'Error', 'Mastering run %1 was not found.', NULL, NULL),
  (9400, 7, 'es', 'Error', 'No se encontró la ejecución de maestrización %1.', NULL, NULL),
  (9400, 7, 'fr', 'Error', 'L''exécution de mastering %1 est introuvable.', NULL, NULL),
  (9400, 8, 'en', 'Error', 'The mastering profile for %1 is not valid: %2', NULL, NULL),
  (9400, 8, 'es', 'Error', 'El perfil de maestrización de %1 no es válido: %2', NULL, NULL),
  (9400, 8, 'fr', 'Error', 'Le profil de mastering de %1 n''est pas valide : %2', NULL, NULL),
  (9400, 9, 'en', 'Error', 'Golden record %1 was not found.', NULL, NULL),
  (9400, 9, 'es', 'Error', 'No se encontró el registro maestro %1.', NULL, NULL),
  (9400, 9, 'fr', 'Error', 'L''enregistrement de référence %1 est introuvable.', NULL, NULL),
  (9400, 10, 'en', 'Error', 'Source system %1 is not registered.', NULL, NULL),
  (9400, 10, 'es', 'Error', 'El sistema de origen %1 no está registrado.', NULL, NULL),
  (9400, 10, 'fr', 'Error', 'Le système source %1 n''est pas enregistré.', NULL, NULL),
  (9400, 11, 'en', 'Error', 'Choose a staging table and a load to master.', NULL, NULL),
  (9400, 11, 'es', 'Error', 'Elija una tabla de staging y una carga para maestrizar.', NULL, NULL),
  (9400, 11, 'fr', 'Error', 'Choisissez une table de staging et un chargement à traiter.', NULL, NULL)
ON CONFLICT (set_nbr, message_nbr, language_cd) DO NOTHING;
