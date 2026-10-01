-- Message set 9400 (mastering): merge requests under the entity's policy.
INSERT INTO public.message_catalog (set_nbr, message_nbr, language_cd, severity, message_text, description, user_action) VALUES
  (9400, 29, 'en', 'Error', 'A merge of this pair is already waiting for approval.', NULL, NULL),
  (9400, 29, 'es', 'Error', 'Ya hay una unión de este par pendiente de aprobación.', NULL, NULL),
  (9400, 29, 'fr', 'Error', 'Une fusion de cette paire attend déjà une approbation.', NULL, NULL),
  (9400, 30, 'en', 'Error', 'Merge request %1 was not found.', NULL, NULL),
  (9400, 30, 'es', 'Error', 'No se encontró la solicitud de unión %1.', NULL, NULL),
  (9400, 30, 'fr', 'Error', 'La demande de fusion %1 est introuvable.', NULL, NULL),
  (9400, 31, 'en', 'Error', 'You cannot approve or reject your own merge request; another person must.', NULL, NULL),
  (9400, 31, 'es', 'Error', 'No puede aprobar ni rechazar su propia solicitud de unión; debe hacerlo otra persona.', NULL, NULL),
  (9400, 31, 'fr', 'Error', 'Vous ne pouvez pas approuver ni rejeter votre propre demande de fusion ; une autre personne doit le faire.', NULL, NULL),
  (9400, 32, 'en', 'Error', 'You have already decided on this merge request.', NULL, NULL),
  (9400, 32, 'es', 'Error', 'Ya ha decidido sobre esta solicitud de unión.', NULL, NULL),
  (9400, 32, 'fr', 'Error', 'Vous avez déjà statué sur cette demande de fusion.', NULL, NULL),
  (9400, 33, 'en', 'Error', 'Only the person who requested a pending merge can withdraw it.', NULL, NULL),
  (9400, 33, 'es', 'Error', 'Solo quien solicitó una unión pendiente puede retirarla.', NULL, NULL),
  (9400, 33, 'fr', 'Error', 'Seule la personne qui a demandé une fusion en attente peut la retirer.', NULL, NULL),
  (9400, 34, 'en', 'Error', 'This merge request has already been decided (%1).', NULL, NULL),
  (9400, 34, 'es', 'Error', 'Esta solicitud de unión ya se decidió (%1).', NULL, NULL),
  (9400, 34, 'fr', 'Error', 'Cette demande de fusion a déjà été traitée (%1).', NULL, NULL),
  (9400, 35, 'en', 'Error', 'Merges need approval under this entity''s policy, but merge requests are not set up yet (crims migration 0014).', NULL, NULL),
  (9400, 35, 'es', 'Error', 'Las uniones requieren aprobación según la política de esta entidad, pero las solicitudes de unión aún no están configuradas (migración crims 0014).', NULL, NULL),
  (9400, 35, 'fr', 'Error', 'Les fusions nécessitent une approbation selon la politique de cette entité, mais les demandes de fusion ne sont pas encore configurées (migration crims 0014).', NULL, NULL)
ON CONFLICT (set_nbr, message_nbr, language_cd) DO NOTHING;
