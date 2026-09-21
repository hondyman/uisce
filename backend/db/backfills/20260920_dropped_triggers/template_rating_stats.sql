-- Backfill: process_templates.rating_average + rating_count
--
-- Replaces the deleted `update_template_rating_stats` trigger. After 002c
-- drops the trigger, this is the one-shot script that brings rating stats
-- up to date by recomputing from current template_ratings rows.
--
-- Run once, after the alpha-trg connector snapshot completes (otherwise
-- CDC events for any new ratings would race this update).
--   psql "$ALPHA_DB_URL" -f this_file.sql

BEGIN;

UPDATE process_templates pt
SET    rating_average = s.avg_rating,
       rating_count   = s.total_count,
       updated_at     = NOW()
FROM (
    SELECT template_id,
           AVG(rating)::numeric(3,2) AS avg_rating,
           COUNT(*)::integer          AS total_count
    FROM   template_ratings
    WHERE  moderation_status = 'approved'
    GROUP  BY template_id
) s
WHERE  pt.id = s.template_id
  AND (pt.rating_average IS DISTINCT FROM s.avg_rating
       OR pt.rating_count   IS DISTINCT FROM s.total_count);

-- Templates with zero approved ratings: zero out the stats so we don't
-- show stale averages from the deleted-trigger era.
UPDATE process_templates pt
SET    rating_average = 0,
       rating_count   = 0,
       updated_at     = NOW()
WHERE  pt.rating_count > 0
  AND  NOT EXISTS (
        SELECT 1 FROM template_ratings tr
        WHERE  tr.template_id = pt.id
          AND  tr.moderation_status = 'approved'
  );

COMMIT;
