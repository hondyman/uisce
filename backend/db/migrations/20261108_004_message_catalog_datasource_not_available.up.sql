-- 1-16: the X-Tenant-Datasource-ID a request named is unknown, inactive or
-- belongs to another tenant (security.ErrDatasourceNotAvailable). Answered
-- 403 so it is not mistaken for 1-3 "authentication is required".
INSERT INTO public.message_catalog (set_nbr, message_nbr, language_cd, severity, message_text, description, user_action) VALUES
  (1, 16, 'en', 'Error', 'The selected datasource is not available for this tenant.', 'X-Tenant-Datasource-ID is unknown, inactive or belongs to another tenant', 'Choose a datasource in Operating Scope and try again.'),
  (1, 16, 'es', 'Error', 'El origen de datos seleccionado no está disponible para esta organización.', NULL, 'Elija un origen de datos en el ámbito operativo y vuelva a intentarlo.'),
  (1, 16, 'fr', 'Error', 'La source de données sélectionnée n''est pas disponible pour cette organisation.', NULL, 'Choisissez une source de données dans le périmètre opérationnel et réessayez.')
ON CONFLICT (set_nbr, message_nbr, language_cd) DO NOTHING;
