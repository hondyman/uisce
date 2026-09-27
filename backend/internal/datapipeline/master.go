package datapipeline

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/survivorship"
	"github.com/jmoiron/sqlx"
)

// masterSink promotes pending staging rows into mdm masters via TwoPoolLoader.
// Inbound rows are a trigger only (content ignored). Work reads staging on the
// data plane and definitions / survivorship rules on alpha.
type masterSink struct {
	cfg    MasterSinkConfig
	alpha  *sql.DB
	data   *sql.DB
	ran    bool
	tenant uuid.UUID
}

func newMasterSink(n Node, d Deps) (Processor, error) {
	var c MasterSinkConfig
	if err := json.Unmarshal(n.Config, &c); err != nil {
		return nil, fmt.Errorf("master_sink config: %w", err)
	}
	c.EntityType = strings.ToUpper(strings.TrimSpace(c.EntityType))
	if c.EntityType == "" {
		return nil, fmt.Errorf("master_sink: entity_type is required")
	}
	switch c.EntityType {
	case "ACCOUNT", "SECURITY", "PARTY":
	default:
		return nil, fmt.Errorf("master_sink: entity_type %q not supported (ACCOUNT|SECURITY|PARTY)", c.EntityType)
	}
	if c.BatchSize <= 0 {
		c.BatchSize = 100
	}
	if c.StagingTable == "" {
		switch c.EntityType {
		case "SECURITY":
			c.StagingTable = "staging.security_data"
		case "PARTY":
			c.StagingTable = "staging.party_data"
		default:
			c.StagingTable = "staging.account_data"
		}
	}
	var raw map[string]json.RawMessage
	_ = json.Unmarshal(n.Config, &raw)
	if _, ok := raw["require_semantic_terms"]; !ok {
		c.RequireSemanticTerms = true
	}
	if d.AlphaDB == nil {
		return nil, fmt.Errorf("master_sink: alpha control-plane database is not configured")
	}
	if d.StagingDB == nil {
		return nil, fmt.Errorf("master_sink: staging / data-plane database is not configured")
	}
	return &masterSink{cfg: c, alpha: d.AlphaDB, data: d.StagingDB}, nil
}

func (p *masterSink) Open(_ context.Context, rc *RunContext) error {
	if rc == nil || rc.TenantID == "" {
		return fmt.Errorf("master_sink: run tenant id is required")
	}
	id, err := uuid.Parse(rc.TenantID)
	if err != nil {
		return fmt.Errorf("master_sink: invalid tenant id: %w", err)
	}
	p.tenant = id
	if rc.DryRun {
		p.cfg.DryRun = true
	}
	return nil
}

func (p *masterSink) Close(context.Context, error) error { return nil }

func (p *masterSink) Process(ctx context.Context, rows []Row) (Result, error) {
	if p.ran {
		return Result{Out: rows}, nil
	}
	p.ran = true

	if p.cfg.DryRun {
		return Result{
			Out: annotateMaster(rows, 0, 0),
			Warnings: []Reject{{
				Reason: fmt.Sprintf("master_sink dry_run: would master up to %d pending %s rows from %s",
					p.cfg.BatchSize, p.cfg.EntityType, p.cfg.StagingTable),
			}},
		}, nil
	}

	alpha := sqlx.NewDb(p.alpha, "postgres")
	data := sqlx.NewDb(p.data, "postgres")
	loader := &TwoPoolLoader{
		AlphaPool:            alpha,
		DataPool:             data,
		Surv:                 survivorship.NewService(alpha),
		RequireSemanticTerms: p.cfg.RequireSemanticTerms,
	}
	var loaded, warnings int
	var err error
	switch p.cfg.EntityType {
	case "SECURITY":
		loaded, warnings, err = loader.LoadSecurityBatch(ctx, p.tenant, p.cfg.BatchSize)
	case "PARTY":
		loaded, warnings, err = loader.LoadPartyBatch(ctx, p.tenant, p.cfg.BatchSize)
	default:
		loaded, warnings, err = loader.LoadAccountBatch(ctx, p.tenant, p.cfg.BatchSize)
	}
	if err != nil {
		return Result{}, err
	}

	var warns []Reject
	if warnings > 0 {
		warns = append(warns, Reject{
			Reason: fmt.Sprintf("master_sink: loaded=%d warnings=%d", loaded, warnings),
		})
	}
	return Result{Out: annotateMaster(rows, loaded, warnings), Warnings: warns}, nil
}

func annotateMaster(rows []Row, loaded, warnings int) []Row {
	if len(rows) == 0 {
		return []Row{{Num: 1, Data: map[string]any{
			"master_loaded":   loaded,
			"master_warnings": warnings,
		}}}
	}
	out := make([]Row, len(rows))
	for i, r := range rows {
		data := map[string]any{}
		for k, v := range r.Data {
			data[k] = v
		}
		data["master_loaded"] = loaded
		data["master_warnings"] = warnings
		out[i] = Row{Num: r.Num, Data: data}
	}
	return out
}
