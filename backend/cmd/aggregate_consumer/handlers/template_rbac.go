package handlers

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hondyman/uisce/backend/cmd/aggregate_consumer/dedupe"
)

// TemplateRBAC handles semantic_query_templates CDC events:
//
//   op=c (insert) → seed 3 RBAC rows (viewer/editor/admin) into
//                   semantic_query_template_permissions. Mirrors the old
//                   vend.create_default_template_permissions trigger.
//
//   op=u (update) → if any versioned field changed (name/description/
//                   semantic_query/parameters), insert a row into
//                   semantic_query_template_versions. Mirrors the old
//                   vend.create_template_version_on_update trigger BUT
//                   fixes the "version on every update" bug with
//                   IS DISTINCT FROM gating.
//
//   op=d (delete) → no action (the original triggers didn't handle deletes).
//   op=r (snapshot read) → no action.
type TemplateRBAC struct {
	pool   *pgxpool.Pool
	dedupe *dedupe.Store
}

const templateRBACHandler = "template_rbac"

func NewTemplateRBAC(pool *pgxpool.Pool, d *dedupe.Store) *TemplateRBAC {
	return &TemplateRBAC{pool: pool, dedupe: d}
}

func (h *TemplateRBAC) Topic() string { return "alpha_trg.public.semantic_query_templates" }

func (h *TemplateRBAC) Handle(ctx context.Context, evt Event) error {
	switch evt.Op {
	case "c":
		return h.seedRBAC(ctx, evt)
	case "u":
		return h.maybeVersion(ctx, evt)
	default:
		return nil
	}
}

func (h *TemplateRBAC) seedRBAC(ctx context.Context, evt Event) error {
	if err := h.dedupe.MarkIfNew(ctx, templateRBACHandler, evt.LSN, evt.Table, evt.Op); err != nil {
		if errors.Is(err, dedupe.ErrAlreadyProcessed) {
			return nil
		}
		return fmt.Errorf("dedupe: %w", err)
	}

	templateID := stringOf(evt.After["id"])
	if templateID == "" {
		return fmt.Errorf("seedRBAC: empty template id")
	}

	const seed = `
INSERT INTO semantic_query_template_permissions
       (template_id, role, can_run, can_edit, can_delete, can_promote)
VALUES ($1::uuid, 'viewer', true,  false, false, false),
       ($1::uuid, 'editor', true,  true,  false, false),
       ($1::uuid, 'admin',  true,  true,  true,  true)
ON CONFLICT DO NOTHING
`
	if _, err := h.pool.Exec(ctx, seed, templateID); err != nil {
		return fmt.Errorf("seed RBAC for template %s: %w", templateID, err)
	}
	return nil
}

func (h *TemplateRBAC) maybeVersion(ctx context.Context, evt Event) error {
	// Compare versioned fields. IS DISTINCT FROM in SQL matches jsonb_object_field
	// inequality including null. Here we use jsonb-aware equality in Go.
	if jsonEqual(evt.Before["name"], evt.After["name"]) &&
		jsonEqual(evt.Before["description"], evt.After["description"]) &&
		jsonEqual(evt.Before["semantic_query"], evt.After["semantic_query"]) &&
		jsonEqual(evt.Before["parameters"], evt.After["parameters"]) {
		return nil
	}

	if err := h.dedupe.MarkIfNew(ctx, templateRBACHandler, evt.LSN, evt.Table, evt.Op); err != nil {
		if errors.Is(err, dedupe.ErrAlreadyProcessed) {
			return nil
		}
		return fmt.Errorf("dedupe: %w", err)
	}

	templateID := stringOf(evt.After["id"])
	if templateID == "" {
		return fmt.Errorf("maybeVersion: empty template id")
	}

	const nextVersion = `
INSERT INTO semantic_query_template_versions
       (template_id, version_number, name, description, semantic_query, parameters,
        change_message, created_by)
SELECT $1::uuid,
       COALESCE((SELECT MAX(version_number) FROM semantic_query_template_versions WHERE template_id = $1::uuid), 0) + 1,
       $2, $3, $4::jsonb, $5::jsonb,
       '', $6
`
	name := stringOf(evt.After["name"])
	description := stringOf(evt.After["description"])
	semanticQuery := mustJSON(evt.After["semantic_query"])
	parameters := mustJSON(evt.After["parameters"])
	updatedBy := stringOf(evt.After["updated_by"])

	if _, err := h.pool.Exec(ctx, nextVersion, templateID, name, description, semanticQuery, parameters, updatedBy); err != nil {
		return fmt.Errorf("version template %s: %w", templateID, err)
	}
	return nil
}
