-- Message set 9000: business-object record API errors (internal/api
-- bo_errors.go). English, Spanish and French; other languages fall back to
-- English until translated in the catalog. ON CONFLICT DO NOTHING: edits
-- made through the catalog win.
INSERT INTO public.message_sets (set_nbr, set_name, module, set_type, description)
VALUES (9000, 'Business Objects', 'bo', 'core', 'Business-object record API messages')
ON CONFLICT (set_nbr) DO NOTHING;

INSERT INTO public.message_catalog (set_nbr, message_nbr, language_cd, severity, message_text, description, user_action) VALUES
  (9000, 1, 'en', 'Error', 'Business object %1 was not found.', 'BO key/id unknown to this tenant or gold copy', NULL),
  (9000, 1, 'es', 'Error', 'No se encontró el objeto de negocio %1.', NULL, NULL),
  (9000, 1, 'fr', 'Error', 'L''objet métier %1 est introuvable.', NULL, NULL),
  (9000, 2, 'en', 'Error', 'The %1 record was not found, or it belongs to another organization.', 'no row for id (tenant-scoped)', NULL),
  (9000, 2, 'es', 'Error', 'No se encontró el registro de %1 o pertenece a otra organización.', NULL, NULL),
  (9000, 2, 'fr', 'Error', 'L''enregistrement %1 est introuvable ou appartient à une autre organisation.', NULL, NULL),
  (9000, 3, 'en', 'Error', 'Unknown field ''%1''.', 'payload key is not a column of the driving table', 'Check the field name against the business object''s fields.'),
  (9000, 3, 'es', 'Error', 'Campo desconocido: ''%1''.', NULL, 'Compruebe el nombre del campo con los campos del objeto de negocio.'),
  (9000, 3, 'fr', 'Error', 'Champ inconnu : ''%1''.', NULL, 'Vérifiez le nom du champ par rapport aux champs de l''objet métier.'),
  (9000, 4, 'en', 'Error', 'No fields to save were provided.', 'payload had no writable columns', NULL),
  (9000, 4, 'es', 'Error', 'No se indicó ningún campo para guardar.', NULL, NULL),
  (9000, 4, 'fr', 'Error', 'Aucun champ à enregistrer n''a été fourni.', NULL, NULL),
  (9000, 5, 'en', 'Error', 'The record was rejected by validation rules: %1.', 'BLOCK rule violation with enforcement on', 'Correct the values the named rules check, then save again.'),
  (9000, 5, 'es', 'Error', 'Las reglas de validación rechazaron el registro: %1.', NULL, 'Corrija los valores que comprueban las reglas indicadas y vuelva a guardar.'),
  (9000, 5, 'fr', 'Error', 'L''enregistrement a été rejeté par les règles de validation : %1.', NULL, 'Corrigez les valeurs vérifiées par les règles indiquées, puis enregistrez à nouveau.'),
  (9000, 6, 'en', 'Error', 'Required fields are missing: %1.', 'is_required field empty in the written row', 'Fill in the listed fields, then save again.'),
  (9000, 6, 'es', 'Error', 'Faltan campos obligatorios: %1.', NULL, 'Complete los campos indicados y vuelva a guardar.'),
  (9000, 6, 'fr', 'Error', 'Des champs obligatoires sont manquants : %1.', NULL, 'Renseignez les champs indiqués, puis enregistrez à nouveau.'),
  (9000, 7, 'en', 'Error', 'At most %1 records can be sent in one request.', 'bulk request over maxBulkRecords', 'Split the records into smaller batches.'),
  (9000, 7, 'es', 'Error', 'Se pueden enviar como máximo %1 registros por solicitud.', NULL, 'Divida los registros en lotes más pequeños.'),
  (9000, 7, 'fr', 'Error', 'Au plus %1 enregistrements peuvent être envoyés par requête.', NULL, 'Répartissez les enregistrements en lots plus petits.'),
  (9000, 8, 'en', 'Error', 'Mode must be create or upsert.', 'bulk mode not create/upsert', NULL),
  (9000, 8, 'es', 'Error', 'El modo debe ser create o upsert.', NULL, NULL),
  (9000, 8, 'fr', 'Error', 'Le mode doit être create ou upsert.', NULL, NULL),
  (9000, 9, 'en', 'Error', 'Upsert needs key fields to find existing records.', 'upsert without key_fields', NULL),
  (9000, 9, 'es', 'Error', 'La operación upsert necesita campos clave para localizar los registros existentes.', NULL, NULL),
  (9000, 9, 'fr', 'Error', 'L''upsert nécessite des champs clés pour retrouver les enregistrements existants.', NULL, NULL),
  (9000, 10, 'en', 'Error', 'Field %1 can''t be used as a key.', 'key field not a writable column, or tenant_id', NULL),
  (9000, 10, 'es', 'Error', 'El campo %1 no se puede usar como clave.', NULL, NULL),
  (9000, 10, 'fr', 'Error', 'Le champ %1 ne peut pas servir de clé.', NULL, NULL),
  (9000, 11, 'en', 'Error', 'The records of %1 are stored in a datasource that isn''t available right now.', 'BO''s bound datasource could not be resolved or connected (never falls back to another DB)', 'Try again later. If it keeps happening, contact support with the reference.'),
  (9000, 11, 'es', 'Error', 'Los registros de %1 están en un origen de datos que no está disponible en este momento.', NULL, 'Vuelva a intentarlo más tarde. Si el problema continúa, contacte con el soporte indicando la referencia.'),
  (9000, 11, 'fr', 'Error', 'Les enregistrements de %1 se trouvent dans une source de données actuellement indisponible.', NULL, 'Réessayez plus tard. Si le problème persiste, contactez le support en indiquant la référence.'),
  (9000, 12, 'en', 'Error', '%2 has no relationship named %1.', 'relKey unknown for the parent BO', NULL),
  (9000, 12, 'es', 'Error', '%2 no tiene ninguna relación llamada %1.', NULL, NULL),
  (9000, 12, 'fr', 'Error', '%2 n''a aucune relation nommée %1.', NULL, NULL),
  (9000, 13, 'en', 'Error', 'Nothing was changed: the record doesn''t exist or belongs to another organization.', 'UPDATE/INSERT returned no row (ErrNoRowWritten)', NULL),
  (9000, 13, 'es', 'Error', 'No se modificó nada: el registro no existe o pertenece a otra organización.', NULL, NULL),
  (9000, 13, 'fr', 'Error', 'Rien n''a été modifié : l''enregistrement n''existe pas ou appartient à une autre organisation.', NULL, NULL),
  (9000, 14, 'en', 'Error', 'Writes are paused: validation rule enforcement is not available.', 'handler has no rule enforcer; refuses rather than bypassing the rules', 'Contact support with the reference.'),
  (9000, 14, 'es', 'Error', 'Las escrituras están en pausa: la aplicación de las reglas de validación no está disponible.', NULL, 'Contacte con el soporte indicando la referencia.'),
  (9000, 14, 'fr', 'Error', 'Les écritures sont suspendues : l''application des règles de validation n''est pas disponible.', NULL, 'Contactez le support en indiquant la référence.'),
  (9000, 15, 'en', 'Error', 'Key field %1 is missing from the record.', 'upsert row without a key field value', NULL),
  (9000, 15, 'es', 'Error', 'Falta el campo clave %1 en el registro.', NULL, NULL),
  (9000, 15, 'fr', 'Error', 'Le champ clé %1 est absent de l''enregistrement.', NULL, NULL),
  (9000, 16, 'en', 'Error', 'The related records can''t be linked to %1: no foreign key connects them.', 'child FK column could not be resolved', NULL),
  (9000, 16, 'es', 'Error', 'Los registros relacionados no se pueden vincular a %1: ninguna clave foránea los conecta.', NULL, NULL),
  (9000, 16, 'fr', 'Error', 'Les enregistrements liés ne peuvent pas être rattachés à %1 : aucune clé étrangère ne les relie.', NULL, NULL)
ON CONFLICT (set_nbr, message_nbr, language_cd) DO NOTHING;
