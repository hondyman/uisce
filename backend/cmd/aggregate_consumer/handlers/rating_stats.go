package handlers

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hondyman/uisce/backend/cmd/aggregate_consumer/dedupe"
)

// RatingStats transposes the trigger `update_template_rating_stats` from
// public.template_ratings into a CDC consumer. The original trigger recomputed
// process_templates.rating_average / rating_count on every INSERT/UPDATE/DELETE,
// filtering on moderation_status = 'approved'.
//
// Differences from the trigger:
//   - Dedupe on (source_lsn, template_ratings, op) so the same event can't be
//     double-applied on rebalance/restart.
//   - REPLICA IDENTITY FULL is required on template_ratings so the consumer
//     can detect moderation_status transitions (the trigger logic filters
//     by status='approved' but recomputes on any I/U/D; the consumer does
//     the same).
type RatingStats struct {
	pool   *pgxpool.Pool
	dedupe *dedupe.Store
}

const ratingHandler = "rating_stats"

func NewRatingStats(pool *pgxpool.Pool, d *dedupe.Store) *RatingStats {
	return &RatingStats{pool: pool, dedupe: d}
}

func (h *RatingStats) Topic() string { return "alpha_trg.public.template_ratings" }

func (h *RatingStats) Handle(ctx context.Context, evt Event) error {
	if err := h.dedupe.MarkIfNew(ctx, ratingHandler, evt.LSN, evt.Table, evt.Op); err != nil {
		if errors.Is(err, dedupe.ErrAlreadyProcessed) {
			return nil
		}
		return fmt.Errorf("dedupe: %w", err)
	}

	templateID, ok := evt.After["template_id"].(string)
	if !ok {
		if evt.Before != nil {
			templateID, _ = evt.Before["template_id"].(string)
		}
	}
	if templateID == "" {
		return nil
	}

	const recompute = `
UPDATE process_templates pt
SET    rating_average = COALESCE(s.avg_rating, 0.0)::numeric(3,2),
       rating_count   = s.total_count,
       updated_at     = NOW()
FROM (
    SELECT AVG(rating)::numeric(3,2) AS avg_rating,
           COUNT(*)::integer          AS total_count
    FROM   template_ratings
    WHERE  template_id = $1
      AND  moderation_status = 'approved'
) s
WHERE  pt.id = $1::uuid
`
	if _, err := h.pool.Exec(ctx, recompute, templateID); err != nil {
		return fmt.Errorf("recompute rating stats for template_id=%s: %w", templateID, err)
	}
	return nil
}

