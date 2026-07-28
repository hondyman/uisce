package boresolver_test

import (
	"testing"
	"time"

	"github.com/hondyman/uisce/backend/internal/boresolver"
	"github.com/stretchr/testify/assert"
)

func TestBitemporalScoping(t *testing.T) {
	repo := &boresolver.MockBORepository{
		BODefinitions: map[string]*boresolver.BODefinition{
			"bo_trades": {
				ID:           "bo_trades",
				DrivingTable: "trades",
				Fields: []boresolver.BOField{
					{ID: "f_id", Name: "id", PhysicalColumn: "id"},
				},
			},
		},
	}

	generator, _ := boresolver.NewBOSQLGenerator(repo, "postgres")
	now := time.Now()

	req := boresolver.SQLGenerationRequest{
		BusinessObjectID: "bo_trades",
		SelectedFields:   []string{"id"},
		TenantID:         "tenant-123",
		KnowledgeDate:    now,
	}

	sql, args, err := generator.GenerateSQL(req)
	assert.NoError(t, err)
	assert.Contains(t, sql, "system_valid_from")
	assert.Contains(t, sql, "system_valid_to")
	assert.Len(t, args, 2) // tenant_id ($1) and knowledgeDate ($2)
}

func TestFXResolver_InjectsJoin(t *testing.T) {
	fxResolver := boresolver.NewFXResolver("USD", time.Now())
	ctx := &boresolver.GenerationContext{
		Request: boresolver.SQLGenerationRequest{TenantID: "tenant-123"},
		Aliases: make(map[string]string),
		Joins:   make([]boresolver.JoinStep, 0),
	}

	convertedExpr, fxAlias := fxResolver.InjectFXConversionJoin(ctx, boresolver.PostgresDialect{}, "EUR", "t0.amount")

	assert.Equal(t, "fx_eur", fxAlias)
	assert.Contains(t, convertedExpr, "t0.amount * COALESCE(fx_eur.rate, 1.0)")
	assert.Len(t, ctx.Joins, 1)
	assert.Equal(t, "public.fx_rates", ctx.Joins[0].ToTable)

}

func TestCEPStreamingCompiler_FlinkSQL(t *testing.T) {
	generator, _ := boresolver.NewBOSQLGenerator(&boresolver.MockBORepository{}, "postgres")
	ctx := &boresolver.GenerationContext{
		Aliases: make(map[string]string),
	}

	req := boresolver.StreamingGenerationRequest{
		SQLGenerationRequest: boresolver.SQLGenerationRequest{
			TenantID: "tenant-999",
		},
		Fields:         []string{"account_id", "margin_requirement"},
		TopicName:      "redpanda.tenant_999.market_ticks",
		WindowType:     "TUMBLE",
		WindowInterval: "5 MINUTE",
	}

	flinkSQL, err := generator.CompileFlinkStreamingSQL(ctx, req)
	assert.NoError(t, err)
	assert.Contains(t, flinkSQL, "JSON_VALUE(payload, '$.account_id') AS account_id")
	assert.Contains(t, flinkSQL, "JSON_VALUE(payload, '$.margin_requirement') AS margin_requirement")
	assert.Contains(t, flinkSQL, "TUMBLE(TABLE redpanda.tenant_999.market_ticks, DESCRIPTOR(proctime), INTERVAL '5 MINUTE')")
	assert.Contains(t, flinkSQL, "JSON_VALUE(payload, '$.tenant_id') = 'tenant-999'")
}

