package datapipeline

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// RatingStagingRow is one pending staging.rating_incoming row.
type RatingStagingRow struct {
	ID                   uuid.UUID         `db:"id"`
	TenantID             uuid.UUID         `db:"tenant_id"`
	SourceSystemID       uuid.UUID         `db:"source_system_id"`
	SourceSystemCd       string            `db:"source_system_cd"`
	SourceRowID          string            `db:"source_row_id"`
	LoadRunID            uuid.UUID         `db:"load_run_id"`
	RatedPartyType       string            `db:"rated_party_type"`
	RatedPartyKey        *string           `db:"rated_party_key"`
	RatedPartyID         *uuid.UUID        `db:"rated_party_id"`
	AgencyCd             string            `db:"agency_cd"`
	RatingTypeCd         string            `db:"rating_type_cd"`
	RatingScaleCd        *string           `db:"rating_scale_cd"`
	RatingValue          string            `db:"rating_value"`
	OutlookCd            *string           `db:"outlook_cd"`
	WatchCd              *string           `db:"watch_cd"`
	CreditWatchDirection *string           `db:"credit_watch_direction"`
	IsCreditEvent        bool              `db:"is_credit_event"`
	RatingDate           time.Time         `db:"rating_date"`
	EffectiveFrom        time.Time         `db:"effective_from"`
	EffectiveTo          *time.Time        `db:"effective_to"`
	SourceDocument       *string           `db:"source_document"`
}

// LoadRatingBatch reads up to batchSize pending staging.rating_incoming rows
// for tenantID and inserts them into mdm.rating. A row is rejected (recorded
// as is_valid=false on the staging row with a structured reason) when:
//   - agency_cd does not resolve to mdm.rating_agency.id
//   - rating_type_cd does not resolve to mdm.rating_type.id
//   - rated_party_key does not resolve to mdm.party.id (rated_party_id is NOT NULL)
//   - rating_scale_cd is set but rating_value is not on that scale (VALUE_NOT_ON_SCALE)
//
// rating_rank is derived from mdm.rating_scale.rank_no (Shape B) at write
// time via batch-loaded scaleRows map keyed by (scale_cd, value). Single
// source of truth — staging rows do not carry rank.
//
// Shape B notes:
//   - mdm.rating_scale columns: scale_cd, value, rank_no, agency_id, agency_cd
//     UNIQUE (tenant_id, scale_cd, value) — agency_cd not in key
//   - mdm.rating_outlook.columns: outlook_cd (was code)
//   - mdm.rating_watch.columns: watch_cd (was code)
//   - mdm.rating_type.columns: type_cd (was code)
//   - mdm.rating_action_type.columns: action_cd (was code)
//     Direction values must be in Shape B's CHECK list:
//     UPGRADE, DOWNGRADE, AFFIRMATION, INITIAL, WITHDRAWN, PLACED_ON_WATCH,
//     REMOVED_FROM_WATCH, DEFAULT, CURE, OTHER.
//   - mdm.rating_action: shape-agnostic; loader writes the new model_value column.
func (l *TwoPoolLoader) LoadRatingBatch(ctx context.Context, tenantID uuid.UUID, batchSize int) (loaded int, rejected int, err error) {
	if batchSize <= 0 {
		batchSize = 100
	}
	if l.DataPool == nil {
		return 0, 0, fmt.Errorf("two-pool loader requires DataPool (crims)")
	}

	tx, err := l.DataPool.BeginTxx(ctx, nil)
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`SELECT set_config('app.current_tenant', $1, true), set_config('uisce.current_tenant', $1, true)`,
		tenantID.String()); err != nil {
		return 0, 0, fmt.Errorf("set tenant guc: %w", err)
	}

	// ── 1. Load pending rows ───────────────────────────────────────────
	var rows []RatingStagingRow
	if err := tx.SelectContext(ctx, &rows, `
		SELECT id, tenant_id, source_system_id, source_system_cd, source_row_id,
		       load_run_id,
		       rated_party_type, rated_party_key, rated_party_id,
		       agency_cd, rating_type_cd, rating_scale_cd, rating_value,
		       outlook_cd, watch_cd, credit_watch_direction, is_credit_event,
		       rating_date, effective_from, effective_to, source_document
		FROM staging.rating_incoming
		WHERE tenant_id = $1
		  AND loaded_at_master IS NULL
		  AND is_valid = true
		ORDER BY loaded_at, mapped_at
		LIMIT $2`, tenantID, batchSize); err != nil {
		return 0, 0, fmt.Errorf("select staging rows: %w", err)
	}
	if len(rows) == 0 {
		return 0, 0, tx.Commit()
	}

	// ── 2. Bulk-resolve reference IDs ──────────────────────────────────
	agencyCDs := uniqueStrings(rows, func(r RatingStagingRow) string { return r.AgencyCd })
	typeCDs := uniqueStrings(rows, func(r RatingStagingRow) string { return r.RatingTypeCd })
	outlookCDs := uniqueNonNilStrings(rows, func(r RatingStagingRow) *string { return r.OutlookCd })
	watchCDs := uniqueNonNilStrings(rows, func(r RatingStagingRow) *string { return r.WatchCd })
	partyKeys := uniqueNonNilStrings(rows, func(r RatingStagingRow) *string { return r.RatedPartyKey })

	agencyID, err := lookupAgencyIDs(ctx, tx, tenantID, agencyCDs)
	if err != nil {
		return 0, 0, fmt.Errorf("resolve agencies: %w", err)
	}
	typeID, err := lookupRatingTypeIDs(ctx, tx, tenantID, typeCDs)
	if err != nil {
		return 0, 0, fmt.Errorf("resolve rating_types: %w", err)
	}
	// Shape B scale lookup: one batch query, composite (scale_cd, value) key.
	// Replaces the per-scale map that collided on shape A's (scale_cd alone)
	// key — multiple values per scale silently lost.
	scaleRows, err := loadScaleRows(ctx, tx, tenantID)
	if err != nil {
		return 0, 0, fmt.Errorf("resolve rating_scale: %w", err)
	}
	outlookID, err := lookupOutlookIDs(ctx, tx, tenantID, outlookCDs)
	if err != nil {
		return 0, 0, fmt.Errorf("resolve outlooks: %w", err)
	}
	watchID, err := lookupWatchIDs(ctx, tx, tenantID, watchCDs)
	if err != nil {
		return 0, 0, fmt.Errorf("resolve watches: %w", err)
	}
	partyID, err := lookupPartyIDs(ctx, tx, tenantID, partyKeys)
	if err != nil {
		return 0, 0, fmt.Errorf("resolve parties: %w", err)
	}

	// Resolve action_type_ids once at batch start — three round-trips per
	// batch, regardless of batch size. The classifier picks one per row.
	// Shape B: WHERE action_cd = $2 (was code).
	actionTypeIDs := make(map[actionKind]uuid.UUID, 3)
	for _, kind := range []actionKind{
		actionAffirm, actionOverrideApplied, actionOverrideExpired,
	} {
		var id uuid.UUID
		if err := tx.GetContext(ctx, &id,
			`SELECT id FROM mdm.rating_action_type WHERE tenant_id = $1 AND action_cd = $2`,
			tenantID, string(kind)); err != nil {
			if err == sql.ErrNoRows {
				return 0, 0, fmt.Errorf("mdm.rating_action_type: %s missing for tenant %s (run 023_mdm_rating_type.sql and 025_mdm_rating_internal_override_action_types.sql)", kind, tenantID)
			}
			return 0, 0, fmt.Errorf("resolve action_type %q: %w", kind, err)
		}
		actionTypeIDs[kind] = id
	}

	// ── 3. Process each row ────────────────────────────────────────────
	runID := uuid.New()
	for _, row := range rows {
		reasons := []map[string]any{}

		agencyUUID, ok := agencyID[row.AgencyCd]
		if !ok {
			reasons = append(reasons, map[string]any{
				"field": "agency_cd", "value": row.AgencyCd,
				"reason": "AGENCY_NOT_FOUND",
			})
		}
		typeUUID, ok := typeID[row.RatingTypeCd]
		if !ok {
			reasons = append(reasons, map[string]any{
				"field": "rating_type_cd", "value": row.RatingTypeCd,
				"reason": "RATING_TYPE_NOT_FOUND",
			})
		}

		// Party: rated_party_id is NOT NULL in mdm.rating.
		var resolvedPartyID *uuid.UUID
		if row.RatedPartyID != nil {
			resolvedPartyID = row.RatedPartyID
		} else if row.RatedPartyKey != nil {
			if pid, ok := partyID[*row.RatedPartyKey]; ok {
				resolvedPartyID = &pid
			}
		}
		if resolvedPartyID == nil {
			reasons = append(reasons, map[string]any{
				"field": "rated_party_key",
				"value": strDeref(row.RatedPartyKey),
				"reason": "PARTY_NOT_FOUND",
			})
		}

		if len(reasons) > 0 {
			if err := markRowInvalid(ctx, tx, row.ID, reasons); err != nil {
				return loaded, rejected, fmt.Errorf("mark invalid: %w", err)
			}
			rejected++
			continue
		}

		// Scale lookup: composite (scale_cd, value) key. CSV value off
		// the named scale rejects the row — rank is load-bearing for
		// survivorship comparisons, so a NULL poisons every downstream
		// join that expects rank. Symmetric with the override-path rejection.
		var scaleUUID *uuid.UUID
		var rank *int
		if row.RatingScaleCd != nil {
			sr, ok := scaleRows[scaleKey{ScaleCd: *row.RatingScaleCd, Value: row.RatingValue}]
			if !ok {
				reasons := []map[string]any{{
					"field":    "rating_value",
					"value":    row.RatingValue,
					"scale_cd": *row.RatingScaleCd,
					"reason":   "VALUE_NOT_ON_SCALE",
				}}
				if err := markRowInvalid(ctx, tx, row.ID, reasons); err != nil {
					return loaded, rejected, fmt.Errorf("mark invalid (CSV value off scale): %w", err)
				}
				rejected++
				continue
			}
			scaleUUID = &sr.ID
			r := sr.RankNo
			rank = &r
		}
		var resolvedOutlook *uuid.UUID
		if row.OutlookCd != nil {
			if oid, ok := outlookID[*row.OutlookCd]; ok {
				resolvedOutlook = &oid
			}
		}
		var resolvedWatch *uuid.UUID
		if row.WatchCd != nil {
			if wid, ok := watchID[*row.WatchCd]; ok {
				resolvedWatch = &wid
			}
		}

		newRatingID, priorRatingID, priorValue, priorRank, err := upsertRatingMaster(ctx, tx, tenantID, row, agencyUUID, typeUUID, scaleUUID, rank, resolvedOutlook, resolvedWatch, resolvedPartyID)
		if err != nil {
			return loaded, rejected, fmt.Errorf("upsert mdm.rating: %w", err)
		}

		// Determine the effective rating value: model output, or override
		// applied on top. For INTERNAL rows only.
		modelValue := row.RatingValue
		effectiveValue := modelValue
		if row.AgencyCd == "INTERNAL" {
			override, oerr := lookupInternalOverride(ctx, tx, tenantID, row.RatedPartyType, row.RatedPartyKey)
			if oerr != nil {
				return loaded, rejected, fmt.Errorf("internal override lookup: %w", oerr)
			}
			if override != nil {
				effectiveValue = override.Value
			}
		}

		// Update the master row's rating_value AND rating_rank if the
		// override changed the value. Reuse the batch-loaded scaleRows map.
		// The CSV-path scale lookup (above) guarantees rank is set whenever
		// rating_scale_cd is set, so the override path always has a row to
		// find — but we still markRowInvalid if not (committee typo).
		if effectiveValue != row.RatingValue {
			var overrideRank *int
			if row.RatingScaleCd != nil {
				sr, ok := scaleRows[scaleKey{ScaleCd: *row.RatingScaleCd, Value: effectiveValue}]
				if !ok {
					reasons := []map[string]any{{
						"field":    "rating_value",
						"value":    effectiveValue,
						"scale_cd": *row.RatingScaleCd,
						"reason":   "OVERRIDE_VALUE_NOT_ON_SCALE",
					}}
					if err := markRowInvalid(ctx, tx, row.ID, reasons); err != nil {
						return loaded, rejected, fmt.Errorf("mark invalid (override scale miss): %w", err)
					}
					rejected++
					continue
				}
				overrideRank = &sr.RankNo
			}

			if _, eerr := tx.ExecContext(ctx, `
				UPDATE mdm.rating SET rating_value = $2, rating_rank = $3
				WHERE id = $1`, newRatingID, effectiveValue, nullableInt(overrideRank)); eerr != nil {
				return loaded, rejected, fmt.Errorf("apply override to master: %w", eerr)
			}
			rank = overrideRank
		}

		// Classify the action. priorWasOverride is read only when a prior
		// exists (priorRatingID != uuid.Nil); for first inserts the prior
		// branch never fires.
		var priorWasOverride bool
		if priorRatingID != uuid.Nil {
			priorWasOverride, err = lookupPriorWasOverride(ctx, tx, priorRatingID, actionTypeIDs[actionOverrideApplied])
			if err != nil {
				return loaded, rejected, fmt.Errorf("prior action lookup: %w", err)
			}
		}

		kind, modelOut := classifyAction(modelValue, effectiveValue, priorValue,
			priorRatingID != uuid.Nil, priorWasOverride)

		if err := insertRatingAction(ctx, tx, tenantID, runID, newRatingID, priorRatingID,
			actionTypeIDs[kind], priorValue, effectiveValue, priorRank, rank,
			row.OutlookCd, row.WatchCd, row.IsCreditEvent,
			strDeref(row.SourceDocument), row.SourceSystemCd, row.RatingDate, modelOut); err != nil {
			return loaded, rejected, fmt.Errorf("insert mdm.rating_action: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE staging.rating_incoming
			SET loaded_at_master = now()
			WHERE id = $1`, row.ID); err != nil {
			return loaded, rejected, fmt.Errorf("mark staging loaded_at_master: %w", err)
		}
		loaded++
	}

	if err := tx.Commit(); err != nil {
		return loaded, rejected, err
	}
	return loaded, rejected, nil
}

// ratingScaleRow is the projection of mdm.rating_scale the loader needs:
// id (FK target) and rank_no (master.rating_rank source).
type ratingScaleRow struct {
	ID     uuid.UUID
	RankNo int
}

// scaleKey is the composite lookup key for mdm.rating_scale. Shape B's
// UNIQUE (tenant_id, scale_cd, value) means the (scale_cd, value) pair
// uniquely identifies a row — keying only on scale_cd would collide
// across values within the same scale.
type scaleKey struct {
	ScaleCd string
	Value   string
}

// loadScaleRows loads the tenant's full mdm.rating_scale into a map keyed
// by (scale_cd, value). One round-trip per batch regardless of batch size.
func loadScaleRows(ctx context.Context, tx *sqlx.Tx, tenantID uuid.UUID) (map[scaleKey]ratingScaleRow, error) {
	out := make(map[scaleKey]ratingScaleRow)
	rows, err := tx.QueryContext(ctx, `
		SELECT id, scale_cd, value, rank_no
		FROM mdm.rating_scale
		WHERE tenant_id = $1`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("scale lookup: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var scaleCd, value string
		var rankNo int
		if err := rows.Scan(&id, &scaleCd, &value, &rankNo); err != nil {
			return nil, fmt.Errorf("scale scan: %w", err)
		}
		out[scaleKey{ScaleCd: scaleCd, Value: value}] = ratingScaleRow{
			ID:     id,
			RankNo: rankNo,
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scale rows: %w", err)
	}
	return out, nil
}

func lookupAgencyIDs(ctx context.Context, tx *sqlx.Tx, tenantID uuid.UUID, codes []string) (map[string]uuid.UUID, error) {
	out := map[string]uuid.UUID{}
	if len(codes) == 0 {
		return out, nil
	}
	var rows []struct {
		ID     uuid.UUID `db:"id"`
		Agency string    `db:"agency_cd"`
	}
	q, args, err := sqlx.In(`
		SELECT id, agency_cd FROM mdm.rating_agency
		WHERE tenant_id = ? AND agency_cd IN (?)`, tenantID, codes)
	if err != nil {
		return nil, err
	}
	if err := tx.SelectContext(ctx, &rows, tx.Rebind(q), args...); err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.Agency] = r.ID
	}
	return out, nil
}

func lookupRatingTypeIDs(ctx context.Context, tx *sqlx.Tx, tenantID uuid.UUID, codes []string) (map[string]uuid.UUID, error) {
	out := map[string]uuid.UUID{}
	if len(codes) == 0 {
		return out, nil
	}
	var rows []struct {
		ID      uuid.UUID `db:"id"`
		TypeCd  string    `db:"type_cd"`
	}
	q, args, err := sqlx.In(`
		SELECT id, type_cd FROM mdm.rating_type
		WHERE tenant_id = ? AND type_cd IN (?)`, tenantID, codes)
	if err != nil {
		return nil, err
	}
	if err := tx.SelectContext(ctx, &rows, tx.Rebind(q), args...); err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.TypeCd] = r.ID
	}
	return out, nil
}

func lookupOutlookIDs(ctx context.Context, tx *sqlx.Tx, tenantID uuid.UUID, codes []string) (map[string]uuid.UUID, error) {
	out := map[string]uuid.UUID{}
	if len(codes) == 0 {
		return out, nil
	}
	var rows []struct {
		ID        uuid.UUID `db:"id"`
		OutlookCd string    `db:"outlook_cd"`
	}
	q, args, err := sqlx.In(`
		SELECT id, outlook_cd FROM mdm.rating_outlook
		WHERE tenant_id = ? AND outlook_cd IN (?)`, tenantID, codes)
	if err != nil {
		return nil, err
	}
	if err := tx.SelectContext(ctx, &rows, tx.Rebind(q), args...); err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.OutlookCd] = r.ID
	}
	return out, nil
}

func lookupWatchIDs(ctx context.Context, tx *sqlx.Tx, tenantID uuid.UUID, codes []string) (map[string]uuid.UUID, error) {
	out := map[string]uuid.UUID{}
	if len(codes) == 0 {
		return out, nil
	}
	var rows []struct {
		ID       uuid.UUID `db:"id"`
		WatchCd  string    `db:"watch_cd"`
	}
	q, args, err := sqlx.In(`
		SELECT id, watch_cd FROM mdm.rating_watch
		WHERE tenant_id = ? AND watch_cd IN (?)`, tenantID, codes)
	if err != nil {
		return nil, err
	}
	if err := tx.SelectContext(ctx, &rows, tx.Rebind(q), args...); err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.WatchCd] = r.ID
	}
	return out, nil
}

func lookupPartyIDs(ctx context.Context, tx *sqlx.Tx, tenantID uuid.UUID, keys []string) (map[string]uuid.UUID, error) {
	out := map[string]uuid.UUID{}
	if len(keys) == 0 {
		return out, nil
	}
	// mdm.party (alpha shape, db/migrations/20261028_001_mdm_party_expand.up.sql)
	// has no party_cd column — the only universal business identifier
	// indexed for lookup is `lei`. rated_party_key on staging rows is
	// therefore interpreted as an LEI in this slice. If mdm.party gains
	// another business-key column, expand the SELECT below.
	var rows []struct {
		ID  uuid.UUID `db:"id"`
		Lei *string   `db:"lei"`
	}
	q, args, err := sqlx.In(`
		SELECT id, lei FROM mdm.party
		WHERE tenant_id = ? AND lei IN (?)`, tenantID, keys)
	if err != nil {
		return nil, err
	}
	if err := tx.SelectContext(ctx, &rows, tx.Rebind(q), args...); err != nil {
		return nil, err
	}
	for _, r := range rows {
		if r.Lei != nil {
			out[*r.Lei] = r.ID
		}
	}
	return out, nil
}

// upsertRatingMaster inserts a new mdm.rating row, demoting any prior latest
// row for the same (tenant, agency, rating_type, rated_party) tuple to
// is_latest=false. Returns the new rating_id, and (when applicable) the
// prior rating_id/value/rank for the action log.
func upsertRatingMaster(
	ctx context.Context, tx *sqlx.Tx,
	tenantID uuid.UUID, row RatingStagingRow,
	agencyID, typeID uuid.UUID, scaleID *uuid.UUID, rank *int,
	outlookID, watchID, partyID *uuid.UUID,
) (newID, priorID uuid.UUID, priorValue string, priorRank *int, err error) {

	// 1. Find any existing latest row for this key.
	var existing struct {
		ID          uuid.UUID `db:"id"`
		RatingValue string    `db:"rating_value"`
		RatingRank  *int      `db:"rating_rank"`
	}
	q := `
		SELECT id, rating_value, rating_rank
		FROM mdm.rating
		WHERE tenant_id = $1 AND agency_id = $2 AND rating_type_id = $3
		  AND rated_party_type = $4 AND rated_party_id = $5
		  AND is_latest = true`
	err = tx.GetContext(ctx, &existing, q,
		tenantID, agencyID, typeID, row.RatedPartyType, *partyID)
	switch {
	case err == sql.ErrNoRows:
		// first row for this key — no prior
		err = nil
	case err != nil:
		return uuid.Nil, uuid.Nil, "", nil, err
	default:
		priorID = existing.ID
		priorValue = existing.RatingValue
		priorRank = existing.RatingRank
	}

	// 2. Demote prior.
	if priorID != uuid.Nil {
		if _, derr := tx.ExecContext(ctx, `
			UPDATE mdm.rating SET is_latest = false, effective_to = $2
			WHERE id = $1 AND is_latest = true`,
			priorID, row.RatingDate); derr != nil {
			return uuid.Nil, uuid.Nil, "", nil, fmt.Errorf("demote prior: %w", derr)
		}
	}

	// 3. Insert new row.
	effectiveTo := row.EffectiveTo
	if effectiveTo == nil {
		// open-ended until withdrawn; the demote step above closes it on the
		// next load. No need to set here.
		effectiveTo = nil
	}
	insertQ := `
		INSERT INTO mdm.rating (
			tenant_id, agency_id, rating_type_id, rating_scale_id,
			rated_party_type, rated_party_id, rated_party_key,
			rating_value, rating_rank,
			outlook_id, watch_id, credit_watch_direction,
			is_active, is_latest,
			rating_date, effective_from, effective_to
		) VALUES (
			$1,$2,$3,$4,
			$5,$6,$7,
			$8,$9,
			$10,$11,$12,
			true, true,
			$13, $14, $15
		)
		RETURNING id`
	if err = tx.GetContext(ctx, &newID, insertQ,
		tenantID, agencyID, typeID, scaleID,
		row.RatedPartyType, *partyID, row.RatedPartyKey,
		row.RatingValue, rank,
		outlookID, watchID, row.CreditWatchDirection,
		row.RatingDate, row.EffectiveFrom, effectiveTo,
	); err != nil {
		return uuid.Nil, uuid.Nil, "", nil, fmt.Errorf("insert mdm.rating: %w", err)
	}
	return newID, priorID, priorValue, priorRank, nil
}

func insertRatingAction(
	ctx context.Context, tx *sqlx.Tx,
	tenantID, runID, ratingID, priorRatingID, actionTypeID uuid.UUID,
	priorValue, newValue string,
	priorRank, newRank *int,
	outlookCd, watchCd *string,
	isCreditEvent bool,
	sourceDocument, sourceSystemCd string,
	ratingDate time.Time,
	modelValue *string,
) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO mdm.rating_action (
			tenant_id, rating_id, action_type_id, prior_rating_id,
			prior_value, new_value, prior_rank, new_rank,
			action_date, reason, is_credit_event, source_document,
			model_value
		) VALUES (
			$1,$2,$3,$4,
			$5,$6,$7,$8,
			$9, $10, $11, $12, $13
		)`,
		tenantID, ratingID, actionTypeID, nullableUUID(priorRatingID),
		priorValue, newValue,
		nullableInt(priorRank), nullableInt(newRank),
		ratingDate,
		"source_system="+sourceSystemCd,
		isCreditEvent,
		nullableString(sourceDocument),
		nullableStringPtr(modelValue),
	)
	return err
}

func nullableStringPtr(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}

func markRowInvalid(ctx context.Context, tx *sqlx.Tx, rowID uuid.UUID, reasons []map[string]any) error {
	b, err := json.Marshal(reasons)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE staging.rating_incoming
		SET is_valid = false, validation_errors = $2::jsonb, loaded_at_master = now()
		WHERE id = $1`, rowID, string(b))
	return err
}

// ── helpers ───────────────────────────────────────────────────────────

func uniqueStrings(rows []RatingStagingRow, pick func(RatingStagingRow) string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, r := range rows {
		v := pick(r)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func uniqueNonNilStrings(rows []RatingStagingRow, pick func(RatingStagingRow) *string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, r := range rows {
		v := pick(r)
		if v == nil || *v == "" {
			continue
		}
		if _, ok := seen[*v]; ok {
			continue
		}
		seen[*v] = struct{}{}
		out = append(out, *v)
	}
	return out
}

func strDeref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func nullableUUID(id uuid.UUID) any {
	if id == uuid.Nil {
		return nil
	}
	return id
}

func nullableInt(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// (kept for forward compat with future batch telemetry)
var _ = strconv.Itoa

// ── action_kind + classifier ───────────────────────────────────────────

type actionKind string

const (
actionAffirm          actionKind = "AFFIRMATION"
actionOverrideApplied actionKind = "OVERRIDE_APPLIED"
actionOverrideExpired actionKind = "OVERRIDE_EXPIRED"
)

// classifyAction returns the action_kind that describes one load, plus the
// optional model_value to record. Pure function; no DB, no UUIDs.
//
// model_value: populated only when the model's raw output differs from
// BOTH the prior master value and the new master value. If model output
// equals prior, the override is the only thing that changed —
// prior_value already captures it. If model output equals new, override
// value equals model output and capturing is redundant. hasPrior guards
// against reading priorValue="" as a real change when no prior exists.
func classifyAction(
	modelValue, effectiveValue, priorValue string,
	hasPrior, priorWasOverride bool,
) (actionKind, *string) {
	overrideApplied := modelValue != effectiveValue
	masterChanged := priorValue != effectiveValue
	overrideExpired := !overrideApplied && priorWasOverride && masterChanged

	switch {
	case overrideApplied:
		if hasPrior && modelValue != priorValue {
			v := modelValue
			return actionOverrideApplied, &v
		}
		return actionOverrideApplied, nil

	case overrideExpired:
		return actionOverrideExpired, nil

	default:
		return actionAffirm, nil
	}
}

// internalOverrideValue is the projection of mdm.rating_internal_override
// the loader needs to apply an override.
type internalOverrideValue struct {
	Value  string `db:"rating_value"`
	Reason string `db:"reason"`
}

// lookupInternalOverride returns the at-most-one in-window approved override
// for the (tenant, rated_party_type, rated_party_key) tuple, or nil if none.
// Index uq_rating_internal_override_active guarantees at most one row, but
// we still assert len <= 1 — if the index is dropped or its predicate is
// altered, this is the canary.
func lookupInternalOverride(
	ctx context.Context, tx *sqlx.Tx,
	tenantID uuid.UUID, ratedPartyType string, ratedPartyKey *string,
) (*internalOverrideValue, error) {
	if ratedPartyKey == nil {
		return nil, nil
	}
	var rows []internalOverrideValue
	if err := tx.SelectContext(ctx, &rows, `
		SELECT rating_value, reason
		FROM mdm.rating_internal_override
		WHERE tenant_id = $1 AND rated_party_type = $2 AND rated_party_key = $3
		  AND is_active
		  AND approval_status = 'approved'
		  AND effective_from <= CURRENT_DATE
		  AND (effective_to IS NULL OR effective_to > CURRENT_DATE)`,
		tenantID, ratedPartyType, *ratedPartyKey); err != nil {
		return nil, err
	}
	if len(rows) > 1 {
		// Unreachable given uq_rating_internal_override_active (migration 0012).
		// If this fires: the index was dropped, its predicate was altered,
		// or is_active / approval_status was renamed. See migration 0012.
		return nil, fmt.Errorf(
			"internal override lookup returned %d rows for (%s, %s); "+
				"uq_rating_internal_override_active is broken",
			len(rows), ratedPartyType, *ratedPartyKey)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return &rows[0], nil
}

// lookupPriorWasOverride returns true if the most recent mdm.rating_action
// row for priorRatingID is OVERRIDE_APPLIED (i.e., the prior master value
// was written by an override, not by the model directly). Used by the
// classifier to distinguish "model updated, no override" from "override
// expired, model output restored."
//
// Returns an error if two action rows share the max action_timestamp —
// that's the loader invariant (one action per tx per rating_id) being
// violated, which would silently mis-resolve "most recent" for the
// classifier. id DESC is not a real tiebreak because id is a random UUID.
func lookupPriorWasOverride(
	ctx context.Context, tx *sqlx.Tx,
	priorRatingID, overrideAppliedActionTypeID uuid.UUID,
) (bool, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT action_type_id FROM mdm.rating_action
		WHERE rating_id = $1
		ORDER BY action_timestamp DESC
		LIMIT 2`, priorRatingID)
	if err != nil {
		return false, fmt.Errorf("prior action lookup: %w", err)
	}
	defer rows.Close()

	var collected []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return false, fmt.Errorf("prior action scan: %w", err)
		}
		collected = append(collected, id)
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("prior action rows: %w", err)
	}
	if len(collected) > 1 {
		return false, fmt.Errorf(
			"two mdm.rating_action rows for rating_id=%s share max timestamp; "+
				"loader invariant violated (one action per tx)", priorRatingID)
	}
	if len(collected) == 0 {
		return false, nil
	}
	return collected[0] == overrideAppliedActionTypeID, nil
}
