package boresolver

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestDataFusionIcebergDialect_Compilation(t *testing.T) {
	dialect := DataFusionIcebergDialect{}

	// Test 1: Identifier Quoting
	quotedCol := dialect.QuoteIdent("customer_id")
	if quotedCol != `"customer_id"` {
		t.Errorf("expected \"customer_id\", got %s", quotedCol)
	}

	// Test 2: Identifier Quoting (already quoted)
	quotedCol2 := dialect.QuoteIdent(`"customer_id"`)
	if quotedCol2 != `"customer_id"` {
		t.Errorf("expected \"customer_id\", got %s", quotedCol2)
	}

	// Test 3: Qualified 3-Tier Catalog Path
	parts := []string{"tenant_alpha", "default", "orders"}
	formattedPath := dialect.FormatQualifiedPath(parts)
	expectedPath := `"tenant_alpha"."default"."orders"`
	if formattedPath != expectedPath {
		t.Errorf("expected %s, got %s", expectedPath, formattedPath)
	}

	// Test 4: Placeholder Binds
	placeholder := dialect.BindPlaceholder(1)
	if placeholder != "?" {
		t.Errorf("expected '?', got %s", placeholder)
	}

	// Test 5: Time Travel Snapshot Format
	timeTravelSQL := dialect.FormatTableSnapshot(`"tenant_alpha"."default"."orders"`, "2026-01-01T00:00:00Z")
	expectedTimeTravel := `"tenant_alpha"."default"."orders" FOR SYSTEM_TIME AS OF TIMESTAMP '2026-01-01T00:00:00Z'`
	if timeTravelSQL != expectedTimeTravel {
		t.Errorf("expected %s, got %s", expectedTimeTravel, timeTravelSQL)
	}

	// Test 6: Time Travel with empty timestamp
	noTimeTravel := dialect.FormatTableSnapshot(`"tenant_alpha"."default"."orders"`, "")
	if noTimeTravel != `"tenant_alpha"."default"."orders"` {
		t.Errorf("expected unquoted path, got %s", noTimeTravel)
	}

	// Test 7: Limit/Offset
	limitOffset := dialect.BuildLimitOffset(100, 20)
	expectedLO := "LIMIT 100 OFFSET 20"
	if limitOffset != expectedLO {
		t.Errorf("expected %s, got %s", expectedLO, limitOffset)
	}

	// Test 8: Limit only
	limitOnly := dialect.BuildLimitOffset(50, 0)
	expectedL := "LIMIT 50"
	if limitOnly != expectedL {
		t.Errorf("expected %s, got %s", expectedL, limitOnly)
	}

	// Test 9: No pagination
	noPag := dialect.BuildLimitOffset(0, 0)
	if noPag != "" {
		t.Errorf("expected empty string, got %s", noPag)
	}

	// Test 10: Dialect Name
	if dialect.Name() != "datafusion_iceberg" {
		t.Errorf("expected datafusion_iceberg, got %s", dialect.Name())
	}

	// Test 11: RequiresOrderByForLimit
	if dialect.RequiresOrderByForLimit() != false {
		t.Error("expected RequiresOrderByForLimit to be false")
	}

	// Test 12: SupportsBitemporalAsOf
	if dialect.SupportsBitemporalAsOf() != true {
		t.Error("expected SupportsBitemporalAsOf to be true")
	}

	// Test 13: SafeDiv
	safeDiv := dialect.SafeDiv("orders.amount", "orders.quantity")
	if safeDiv != "(orders.amount / NULLIF(orders.quantity, 0))" {
		t.Errorf("unexpected SafeDiv result: %s", safeDiv)
	}
}

func TestGetDialect_DataFusionAliases(t *testing.T) {
	aliases := []string{"datafusion", "datafusion_iceberg", "iceberg"}
	for _, alias := range aliases {
		d, err := GetDialect(alias)
		if err != nil {
			t.Fatalf("failed to resolve dialect alias '%s': %v", alias, err)
		}
		if d.Name() != "datafusion_iceberg" {
			t.Errorf("alias '%s' resolved to wrong dialect name '%s'", alias, d.Name())
		}
	}
}

func TestGetDialect_UnknownAlias(t *testing.T) {
	_, err := GetDialect("nonexistent_dialect")
	if err == nil {
		t.Error("expected error for unknown dialect, got nil")
	}
}

func TestBOSQLGenerator_ResolveEffectiveDialect_Override(t *testing.T) {
	gen := &BOSQLGenerator{
		Dialect: PostgresDialect{},
	}

	req := &SQLGenerationRequest{
		DialectOverride: "datafusion",
	}

	dialect, err := gen.ResolveEffectiveDialect(req, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dialect.Name() != "datafusion_iceberg" {
		t.Errorf("expected datafusion_iceberg, got %s", dialect.Name())
	}
}

func TestBOSQLGenerator_ResolveEffectiveDialect_Watermark(t *testing.T) {
	watermarkTime, _ := time.Parse(time.RFC3339, "2026-01-15T00:00:00Z")
	asOfTime, _ := time.Parse(time.RFC3339, "2026-01-10T00:00:00Z")

	gen := &BOSQLGenerator{
		Dialect: PostgresDialect{},
		WatermarkResolver: &mockWatermarkResolver{watermark: watermarkTime},
	}

	req := &SQLGenerationRequest{
		TenantID: "tenant-123",
		AsOfDate: asOfTime, // before watermark -> cold tier
	}

	dialect, err := gen.ResolveEffectiveDialect(req, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dialect.Name() != "datafusion_iceberg" {
		t.Errorf("expected datafusion_iceberg (cold tier), got %s", dialect.Name())
	}
}

func TestBOSQLGenerator_ResolveEffectiveDialect_HotTier(t *testing.T) {
	watermarkTime, _ := time.Parse(time.RFC3339, "2026-01-15T00:00:00Z")
	asOfTime, _ := time.Parse(time.RFC3339, "2026-07-01T00:00:00Z") // after watermark -> hot tier

	gen := &BOSQLGenerator{
		Dialect: PostgresDialect{},
		WatermarkResolver: &mockWatermarkResolver{watermark: watermarkTime},
	}

	req := &SQLGenerationRequest{
		TenantID: "tenant-123",
		AsOfDate: asOfTime, // after watermark -> hot tier
	}

	dialect, err := gen.ResolveEffectiveDialect(req, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dialect.Name() != "postgres" {
		t.Errorf("expected postgres (hot tier), got %s", dialect.Name())
	}
}

func TestBOSQLGenerator_ResolveEffectiveDialect_BindingFallback(t *testing.T) {
	gen := &BOSQLGenerator{
		Dialect: PostgresDialect{},
	}

	binding := &BOBinding{
		DialectName: "snowflake",
	}

	req := &SQLGenerationRequest{
		TenantID: "tenant-123",
	}

	dialect, err := gen.ResolveEffectiveDialect(req, binding)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dialect.Name() != "snowflake" {
		t.Errorf("expected snowflake, got %s", dialect.Name())
	}
}

func TestBOSQLGenerator_ResolveEffectiveDialect_DefaultFallback(t *testing.T) {
	gen := &BOSQLGenerator{
		Dialect: PostgresDialect{},
	}

	req := &SQLGenerationRequest{}

	dialect, err := gen.ResolveEffectiveDialect(req, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dialect.Name() != "postgres" {
		t.Errorf("expected postgres, got %s", dialect.Name())
	}
}

// mockWatermarkResolver for testing
type mockWatermarkResolver struct {
	watermark time.Time
}

func (m *mockWatermarkResolver) GetHotColdWatermark(tenantID string) time.Time {
	return m.watermark
}

type mockTelemetryRouter struct {
	flavor string
	err    error
}

func (m *mockTelemetryRouter) GetOptimalFlavor(ctx context.Context, tenantID, boKey, defaultFlavor string) (string, error) {
	return m.flavor, m.err
}

func TestBOSQLGenerator_ResolveEffectiveDialect_CBOOverridesToIceberg(t *testing.T) {
	gen := &BOSQLGenerator{
		Dialect: PostgresDialect{},
		TelemetryRouter: &mockTelemetryRouter{
			flavor: "ICEBERG",
			err:    nil,
		},
	}

	req := &SQLGenerationRequest{
		TenantID:         "tenant-123",
		BusinessObjectID: "bo_orders",
	}

	dialect, err := gen.ResolveEffectiveDialect(req, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dialect.Name() != "datafusion_iceberg" {
		t.Errorf("expected datafusion_iceberg (CBO override), got %s", dialect.Name())
	}
}

func TestBOSQLGenerator_ResolveEffectiveDialect_CBOOverridesToSnowflake(t *testing.T) {
	gen := &BOSQLGenerator{
		Dialect: PostgresDialect{},
		TelemetryRouter: &mockTelemetryRouter{
			flavor: "SNOWFLAKE",
			err:    nil,
		},
	}

	req := &SQLGenerationRequest{
		TenantID:         "tenant-123",
		BusinessObjectID: "bo_orders",
	}

	dialect, err := gen.ResolveEffectiveDialect(req, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dialect.Name() != "snowflake" {
		t.Errorf("expected snowflake (CBO override), got %s", dialect.Name())
	}
}

func TestBOSQLGenerator_ResolveEffectiveDialect_CBOWithExplicitOverride(t *testing.T) {
	gen := &BOSQLGenerator{
		Dialect: PostgresDialect{},
		TelemetryRouter: &mockTelemetryRouter{
			flavor: "ICEBERG",
			err:    nil,
		},
	}

	req := &SQLGenerationRequest{
		TenantID:         "tenant-123",
		BusinessObjectID: "bo_orders",
		DialectOverride:   "snowflake",
	}

	dialect, err := gen.ResolveEffectiveDialect(req, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dialect.Name() != "snowflake" {
		t.Errorf("expected snowflake (explicit override wins), got %s", dialect.Name())
	}
}

func TestBOSQLGenerator_ResolveEffectiveDialect_CBOReturnsSameFlavor(t *testing.T) {
	gen := &BOSQLGenerator{
		Dialect: PostgresDialect{},
		TelemetryRouter: &mockTelemetryRouter{
			flavor: "POSTGRES",
			err:    nil,
		},
	}

	req := &SQLGenerationRequest{
		TenantID:         "tenant-123",
		BusinessObjectID: "bo_orders",
	}

	dialect, err := gen.ResolveEffectiveDialect(req, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dialect.Name() != "postgres" {
		t.Errorf("expected postgres (CBO no-op), got %s", dialect.Name())
	}
}

func TestBOSQLGenerator_ResolveEffectiveDialect_CBOFallbackOnError(t *testing.T) {
	gen := &BOSQLGenerator{
		Dialect: PostgresDialect{},
		TelemetryRouter: &mockTelemetryRouter{
			flavor: "",
			err:    fmt.Errorf("telemetry unavailable"),
		},
	}

	req := &SQLGenerationRequest{
		TenantID:         "tenant-123",
		BusinessObjectID: "bo_orders",
	}

	dialect, err := gen.ResolveEffectiveDialect(req, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dialect.Name() != "postgres" {
		t.Errorf("expected postgres (CBO error fallback), got %s", dialect.Name())
	}
}
