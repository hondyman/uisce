package vectorized

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type IncrementalCalculationMaterializer struct {
	db      *sqlx.DB
	kernels *VectorizedFinancialKernels
}

func NewIncrementalCalculationMaterializer(db *sqlx.DB) *IncrementalCalculationMaterializer {
	return &IncrementalCalculationMaterializer{
		db:      db,
		kernels: NewVectorizedFinancialKernels(),
	}
}

// IngestTransactionEvent processes an incoming trade or cashflow event, updating the materialized state incrementally
func (m *IncrementalCalculationMaterializer) IngestTransactionEvent(
	ctx context.Context,
	tenantID, boID, fieldID uuid.UUID,
	entitySID string,
	eventTimestamp time.Time,
	amount float64,
) (float64, error) {
	tx, err := m.db.BeginTxx(ctx, nil)
	if err != nil {
		return 0.0, err
	}
	defer tx.Rollback()

	cacheKey := computeFormulaCacheKey(boID, fieldID, "XIRR")

	// 1. Fetch current cashflow vector from cache
	var existingPayload []byte
	var currentVal float64
	err = tx.QueryRowContext(ctx, `
		SELECT vector_payload, computed_value 
		FROM public.calc_cache 
		WHERE tenant_id = $1 AND bo_id = $2 AND field_id = $3 AND entity_sid = $4 AND cache_key = $5
		FOR UPDATE
	`, tenantID, boID, fieldID, entitySID, cacheKey).Scan(&existingPayload, &currentVal)

	var vec *PackedCashflowVector
	if err == nil && len(existingPayload) > 0 {
		vec, _ = DeserializeCashflowVector(existingPayload)
	} else {
		vec = &PackedCashflowVector{Dates: make([]int64, 0), Amounts: make([]float64, 0)}
	}

	// 2. Append new cashflow and maintain chronological sort
	eventUnix := eventTimestamp.Unix()
	vec.Dates = append(vec.Dates, eventUnix)
	vec.Amounts = append(vec.Amounts, amount)

	// In-memory sort by date
	type cfPair struct {
		d int64
		a float64
	}
	pairs := make([]cfPair, len(vec.Dates))
	for i := range vec.Dates {
		pairs[i] = cfPair{d: vec.Dates[i], a: vec.Amounts[i]}
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].d < pairs[j].d })
	for i := range pairs {
		vec.Dates[i] = pairs[i].d
		vec.Amounts[i] = pairs[i].a
	}

	// 3. Recompute XIRR on isolated slice (< 50µs)
	newIRR, calcErr := m.kernels.CalculateXIRR(vec.Dates, vec.Amounts, 100, 1e-7)
	if calcErr != nil {
		newIRR = currentVal // Retain previous value if calculation fails to converge
	}

	serialized := vec.Serialize()

	// 4. Update latest cache table
	var cacheID uuid.UUID
	upsertQuery := `
		INSERT INTO public.calc_cache (
			tenant_id, bo_id, field_id, entity_sid, cache_key,
			computed_value, vector_payload, last_event_timestamp, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		ON CONFLICT (tenant_id, bo_id, field_id, entity_sid, cache_key) DO UPDATE SET
			computed_value = EXCLUDED.computed_value,
			vector_payload = EXCLUDED.vector_payload,
			last_event_timestamp = EXCLUDED.last_event_timestamp,
			updated_at = NOW()
		RETURNING cache_id;
	`
	err = tx.QueryRowContext(ctx, upsertQuery,
		tenantID, boID, fieldID, entitySID, cacheKey,
		newIRR, serialized, eventTimestamp,
	).Scan(&cacheID)
	if err != nil {
		return 0.0, fmt.Errorf("failed updating calc cache: %w", err)
	}

	// 5. Expire active bitemporal history record and append new snapshot
	_, _ = tx.ExecContext(ctx, `
		UPDATE public.calc_cache_history 
		SET valid_to = $1 
		WHERE cache_id = $2 AND valid_to IS NULL;
	`, eventTimestamp, cacheID)

	_, _ = tx.ExecContext(ctx, `
		INSERT INTO public.calc_cache_history (
			tenant_id, cache_id, entity_sid, computed_value,
			valid_from, valid_to, vector_payload
		) VALUES ($1, $2, $3, $4, $5, NULL, $6);
	`, tenantID, cacheID, entitySID, newIRR, eventTimestamp, serialized)

	return newIRR, tx.Commit()
}

// GetAsOfMetric queries the point-in-time calculation state without scanning historical databases
func (m *IncrementalCalculationMaterializer) GetAsOfMetric(
	ctx context.Context,
	tenantID, boID, fieldID uuid.UUID,
	entitySID string,
	asOfDate time.Time,
) (float64, bool, error) {
	cacheKey := computeFormulaCacheKey(boID, fieldID, "XIRR")
	var val float64
	query := `
		SELECT h.computed_value 
		FROM public.calc_cache_history h
		JOIN public.calc_cache c ON c.cache_id = h.cache_id
		WHERE h.tenant_id = $1 AND c.bo_id = $2 AND c.field_id = $3 
		  AND h.entity_sid = $4 AND c.cache_key = $5
		  AND h.valid_from <= $6 AND (h.valid_to > $6 OR h.valid_to IS NULL)
		LIMIT 1;
	`
	err := m.db.GetContext(ctx, &val, query, tenantID, boID, fieldID, entitySID, cacheKey, asOfDate)
	if err != nil {
		return 0.0, false, nil // Cache miss; trigger fallback execution
	}
	return val, true, nil
}

func computeFormulaCacheKey(boID, fieldID uuid.UUID, formulaType string) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%s", boID, fieldID, formulaType)))
	return hex.EncodeToString(h[:16])
}
