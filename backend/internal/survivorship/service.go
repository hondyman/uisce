package survivorship

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

var (
	ErrNotFound       = errors.New("survivorship rule not found")
	ErrInvalidInput   = errors.New("invalid survivorship input")
	ErrMissingTerm    = errors.New("semantic_term_id is required and must reference an existing semantic_term catalog node")
	ErrUnknownStrategy = errors.New("unknown survivorship strategy")
)

var allowedStrategies = map[string]bool{
	"SOURCE_PRIORITY":      true,
	"MOST_RECENT":          true,
	"CONSERVATIVE_MIN":     true,
	"CONSERVATIVE_MAX":     true,
	"WEIGHTED_CONFIDENCE":  true,
}

// Service is the alpha control-plane CRUD for semantic survivorship rules.
type Service struct {
	db *sqlx.DB
}

func NewService(db *sqlx.DB) *Service {
	return &Service{db: db}
}

func (s *Service) withTenant(ctx context.Context, tenantID uuid.UUID, fn func(tx *sqlx.Tx) error) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`SELECT set_config('app.current_tenant', $1, true)`,
		tenantID.String(),
	); err != nil {
		return fmt.Errorf("set tenant guc: %w", err)
	}
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

// ListSourceSystems returns the platform MDM source registry.
func (s *Service) ListSourceSystems(ctx context.Context) ([]SourceSystem, error) {
	var out []SourceSystem
	err := s.db.SelectContext(ctx, &out, `
		SELECT code, display_name, default_rank, COALESCE(description, '') AS description,
		       is_active, created_at
		FROM public.mdm_source_systems
		WHERE is_active
		ORDER BY default_rank, code`)
	if err != nil {
		return nil, err
	}
	if out == nil {
		out = []SourceSystem{}
	}
	return out, nil
}

// ListRules returns active (or all) rules for an entity type.
func (s *Service) ListRules(ctx context.Context, tenantID uuid.UUID, entityType string, includeInactive bool) ([]Rule, error) {
	var out []Rule
	err := s.withTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
		q := `
			SELECT r.id, r.tenant_id, r.entity_type, r.semantic_term_id, r.strategy,
			       r.priority_order, r.max_stale_seconds, r.is_active, r.created_at, r.updated_at,
			       COALESCE(cn.node_name, '') AS semantic_term_name,
			       COALESCE(cn.qualified_path, '') AS semantic_term_path
			FROM public.semantic_survivorship_rules r
			LEFT JOIN public.catalog_node cn ON cn.id = r.semantic_term_id
			WHERE r.entity_type = $1`
		if !includeInactive {
			q += ` AND r.is_active`
		}
		q += ` ORDER BY cn.node_name NULLS LAST, r.created_at`
		rows, err := tx.QueryxContext(ctx, q, entityType)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var r Rule
			if err := rows.StructScan(&r); err != nil {
				return err
			}
			out = append(out, r)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	if out == nil {
		out = []Rule{}
	}
	return out, nil
}

func normalizeStrategy(s string) (string, error) {
	s = strings.TrimSpace(strings.ToUpper(s))
	if s == "" {
		s = "SOURCE_PRIORITY"
	}
	if !allowedStrategies[s] {
		return "", ErrUnknownStrategy
	}
	return s, nil
}

func (s *Service) assertSemanticTerm(ctx context.Context, tx *sqlx.Tx, termID uuid.UUID) error {
	var ok bool
	err := tx.GetContext(ctx, &ok, `
		SELECT EXISTS (
			SELECT 1
			FROM public.catalog_node cn
			JOIN public.catalog_node_types t ON t.id = cn.node_type_id
			WHERE cn.id = $1
			  AND cn.is_active
			  AND t.catalog_type_name IN ('semantic_term', 'SEMANTIC_TERM')
		)`, termID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrMissingTerm
	}
	return nil
}

// CreateRule inserts a new rule; refuses orphan field_cd / missing terms.
func (s *Service) CreateRule(ctx context.Context, tenantID uuid.UUID, in CreateRuleInput) (*Rule, error) {
	entityType := strings.TrimSpace(strings.ToUpper(in.EntityType))
	if entityType == "" {
		return nil, fmt.Errorf("%w: entity_type required", ErrInvalidInput)
	}
	termID, err := uuid.Parse(strings.TrimSpace(in.SemanticTermID))
	if err != nil || termID == uuid.Nil {
		return nil, ErrMissingTerm
	}
	strategy, err := normalizeStrategy(in.Strategy)
	if err != nil {
		return nil, err
	}
	if in.MaxStaleSeconds < 0 {
		return nil, fmt.Errorf("%w: max_stale_seconds must be >= 0", ErrInvalidInput)
	}
	priority := in.PriorityOrder
	if priority == nil {
		priority = []string{}
	}

	var out Rule
	err = s.withTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
		if err := s.assertSemanticTerm(ctx, tx, termID); err != nil {
			return err
		}
		now := time.Now().UTC()
		return tx.GetContext(ctx, &out, `
			INSERT INTO public.semantic_survivorship_rules (
				tenant_id, entity_type, semantic_term_id, strategy,
				priority_order, max_stale_seconds, is_active, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, true, $7, $7)
			RETURNING id, tenant_id, entity_type, semantic_term_id, strategy,
			          priority_order, max_stale_seconds, is_active, created_at, updated_at`,
			tenantID, entityType, termID, strategy, pq.Array(priority), in.MaxStaleSeconds, now,
		)
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateRule patches strategy / priority / active flag.
func (s *Service) UpdateRule(ctx context.Context, tenantID, id uuid.UUID, in UpdateRuleInput) (*Rule, error) {
	var out Rule
	err := s.withTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
		var cur Rule
		if err := tx.GetContext(ctx, &cur, `
			SELECT id, tenant_id, entity_type, semantic_term_id, strategy,
			       priority_order, max_stale_seconds, is_active, created_at, updated_at
			FROM public.semantic_survivorship_rules
			WHERE id = $1`, id); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}

		strategy := cur.Strategy
		if in.Strategy != nil {
			ns, err := normalizeStrategy(*in.Strategy)
			if err != nil {
				return err
			}
			strategy = ns
		}
		priority := []string(cur.PriorityOrder)
		if in.PriorityOrder != nil {
			priority = in.PriorityOrder
		}
		stale := cur.MaxStaleSeconds
		if in.MaxStaleSeconds != nil {
			if *in.MaxStaleSeconds < 0 {
				return fmt.Errorf("%w: max_stale_seconds must be >= 0", ErrInvalidInput)
			}
			stale = *in.MaxStaleSeconds
		}
		active := cur.IsActive
		if in.IsActive != nil {
			active = *in.IsActive
		}

		return tx.GetContext(ctx, &out, `
			UPDATE public.semantic_survivorship_rules
			   SET strategy = $2,
			       priority_order = $3,
			       max_stale_seconds = $4,
			       is_active = $5,
			       updated_at = now()
			 WHERE id = $1
			 RETURNING id, tenant_id, entity_type, semantic_term_id, strategy,
			           priority_order, max_stale_seconds, is_active, created_at, updated_at`,
			id, strategy, pq.Array(priority), stale, active,
		)
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// DeactivateRule soft-deletes a rule.
func (s *Service) DeactivateRule(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.withTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
		res, err := tx.ExecContext(ctx, `
			UPDATE public.semantic_survivorship_rules
			   SET is_active = false, updated_at = now()
			 WHERE id = $1`, id)
		if err != nil {
			return err
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return ErrNotFound
		}
		return nil
	})
}
