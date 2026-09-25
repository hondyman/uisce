-- Removes only what 20261027_002 added that did not exist before it: the
-- catalog's own set and the set 1 translations. The ENG -> en conversion
-- is not reversed (en is the app's code; nothing reads ENG).
DELETE FROM public.message_catalog WHERE set_nbr = 9100;
DELETE FROM public.message_sets WHERE set_nbr = 9100;
DELETE FROM public.message_catalog WHERE set_nbr = 1 AND language_cd IN ('es', 'fr');
