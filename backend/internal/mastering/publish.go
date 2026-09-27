package mastering

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// matchOne links a valid record to a golden record: its cross-reference if
// the source sent it before, else a shared identifier, else a fuzzy match,
// else a new golden record. Each record is its own savepoint, so one bad
// record becomes an exception, not a failed run.
func (r *runner) matchOne(rec *Record) error {
	if _, err := r.tx.ExecContext(r.ctx, `SAVEPOINT match_one`); err != nil {
		return err
	}
	saved := *r.counts
	golden, err := r.link(rec)
	if err != nil {
		if _, rbErr := r.tx.ExecContext(r.ctx, `ROLLBACK TO SAVEPOINT match_one`); rbErr != nil {
			return rbErr
		}
		*r.counts = saved
		r.counts.Invalid++
		r.counts.Valid--
		return r.exception("", rec.SourceKey, Issue{Code: "MASTERING_FAILED", Severity: SevError, Message: err.Error()})
	}
	if _, err := r.tx.ExecContext(r.ctx, `RELEASE SAVEPOINT match_one`); err != nil {
		return err
	}
	if golden != "" {
		r.touched[golden] = true
	}
	return nil
}

// link returns the golden record the record now feeds ("" when it was
// held back, e.g. an identifier conflict).
func (r *runner) link(rec *Record) (string, error) {
	ctx, tx := r.ctx, r.tx

	// 1. Seen before.
	var golden string
	err := tx.GetContext(ctx, &golden, `SELECT golden_id::text FROM mdm.entity_xref
		WHERE entity_cd = $1 AND source_system_id::text = $2 AND source_key = $3 AND status = 'ACTIVE'`,
		r.p.EntityCd, r.sourceID, rec.SourceKey)
	if err == nil {
		r.counts.Xref++
		if _, err := tx.ExecContext(ctx, `UPDATE mdm.entity_xref SET last_run_id = $2::uuid, updated_at = now()
			WHERE entity_cd = $1 AND source_system_id::text = $3 AND source_key = $4 AND status = 'ACTIVE'`,
			r.p.EntityCd, r.run.ID, r.sourceID, rec.SourceKey); err != nil {
			return "", err
		}
		return golden, r.ensureIdentifiers(golden, rec)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}

	method, score, ruleCd := MatchNew, 0.0, ""
	var matched []string
	var best string

	// 2. A shared identifier.
	for _, mr := range r.cfg.match {
		if len(mr.DeterministicKeys) == 0 || !r.identifiers {
			continue
		}
		hits, keys, err := r.identifierHits(mr.DeterministicKeys, rec)
		if err != nil {
			return "", err
		}
		if len(hits) > 1 {
			r.counts.Conflicts++
			r.counts.Valid--
			r.counts.Invalid++
			return "", r.exception("", rec.SourceKey, Issue{Code: "IDENTIFIER_CONFLICT", Severity: SevError,
				Message: fmt.Sprintf("identifiers %s point at %d different records: %s", strings.Join(keys, ", "), len(hits), strings.Join(hits, ", "))})
		}
		if len(hits) == 1 {
			method, score, ruleCd, matched, golden = MatchDeterministic, 1, mr.RuleCd, keys, hits[0]
			break
		}
	}

	// 3. A fuzzy match.
	if method == MatchNew {
		for _, mr := range r.cfg.match {
			if len(mr.FuzzyKeys) == 0 {
				continue
			}
			cand, s, err := r.bestFuzzy(mr, rec)
			if err != nil {
				return "", err
			}
			if cand == "" {
				continue
			}
			switch mr.Classify(s) {
			case MatchFuzzy:
				method, score, ruleCd, golden = MatchFuzzy, s, mr.RuleCd, cand
				matched = fuzzyFields(mr)
			case MatchReview:
				if s > score {
					method, score, ruleCd, best = MatchReview, s, mr.RuleCd, cand
					matched = fuzzyFields(mr)
				}
			}
			if method == MatchFuzzy {
				break
			}
		}
	}

	// 4. New (a review match is new too, with a candidate pair to decide).
	xrefMethod := method
	switch method {
	case MatchDeterministic:
		r.counts.Deterministic++
	case MatchFuzzy:
		r.counts.Fuzzy++
	case MatchReview, MatchNew:
		if method == MatchReview {
			r.counts.Review++
		} else {
			r.counts.New++
		}
		xrefMethod = MatchNew
		if golden, err = r.mint(rec); err != nil {
			return "", err
		}
		if method == MatchReview {
			if err := r.candidate(best, golden, score, ruleCd, matched); err != nil {
				return "", err
			}
		}
	}

	keys, _ := json.Marshal(matched)
	var scoreArg any
	if method != MatchNew {
		scoreArg = score
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO mdm.entity_xref
		(tenant_id, entity_cd, source_system_id, source_key, golden_id, match_method, match_score, match_rule_cd, matched_keys, first_run_id, last_run_id)
		VALUES ($1::uuid, $2, $3::uuid, $4, $5::uuid, $6, $7, NULLIF($8, ''), $9, $10::uuid, $10::uuid)`,
		r.tenant, r.p.EntityCd, r.sourceID, rec.SourceKey, golden, xrefMethod, scoreArg, ruleCd, keys, r.run.ID); err != nil {
		return "", err
	}
	return golden, r.ensureIdentifiers(golden, rec)
}

func fuzzyFields(mr MatchRule) []string {
	out := make([]string, 0, len(mr.FuzzyKeys))
	for _, k := range mr.FuzzyKeys {
		out = append(out, k.Field)
	}
	return out
}

// identifierHits finds the golden records that hold any of the record's
// identifiers of the given types. A PROVIDER_CODE is the provider's own id,
// so it only matches the same provider's.
func (r *runner) identifierHits(types []string, rec *Record) ([]string, []string, error) {
	var (
		conds []string
		args  = []any{}
		keys  []string
	)
	id := r.p.ident()
	for _, t := range types {
		v, ok := rec.Identifiers[t]
		if !ok {
			continue
		}
		args = append(args, t, v)
		c := fmt.Sprintf("(%s = $%d AND %s = $%d", qi(id.TypeColumn), len(args)-1, qi(id.ValueColumn), len(args))
		if t == "PROVIDER_CODE" {
			args = append(args, r.identSource())
			c += fmt.Sprintf(" AND %s::text = $%d", qi(id.SourceColumn), len(args))
		}
		conds = append(conds, c+")")
		keys = append(keys, t)
	}
	if len(conds) == 0 {
		return nil, nil, nil
	}
	var hits []string
	err := r.tx.SelectContext(r.ctx, &hits, fmt.Sprintf(`SELECT DISTINCT %s::text FROM %s
		WHERE %s AND (%s) ORDER BY 1`, qi(id.KeyColumn), qi(*r.p.IdentifierTable), id.identActive(""), strings.Join(conds, " OR ")), args...)
	return hits, keys, err
}

// identSource is how the identifier table records the reporting source:
// its system id or its code.
func (r *runner) identSource() string {
	if r.p.ident().SourceIsID {
		return r.sourceID
	}
	return r.sourceCd
}

// bestFuzzy scores the record against the closest golden records by name
// (pg_trgm) and returns the best.
func (r *runner) bestFuzzy(mr MatchRule, rec *Record) (string, float64, error) {
	name := text(rec.Attrs[r.p.Settings.NameAttribute])
	if name == "" {
		return "", 0, nil
	}
	sel := []string{qi(r.p.entityCol()) + "::text AS id"}
	var trigram, exact []string
	for _, k := range mr.FuzzyKeys {
		if _, ok := r.anchorCols[k.Field]; !ok {
			continue
		}
		if k.Method == "trigram" {
			trigram = append(trigram, k.Field)
		} else {
			exact = append(exact, k.Field)
		}
	}
	args := []any{name}
	for _, f := range trigram {
		args = append(args, text(rec.Attrs[f]))
		sel = append(sel, fmt.Sprintf("similarity(%s::text, $%d) AS %s", qi(f), len(args), qi("sim_"+f)))
	}
	for _, f := range exact {
		sel = append(sel, fmt.Sprintf("%s::text AS %s", qi(f), qi("val_"+f)))
	}
	// A record merged into another is no longer a match for anything.
	live := ""
	if _, ok := r.anchorCols["merged_into_id"]; ok {
		live = " AND merged_into_id IS NULL"
	}
	live += " AND " + r.p.current("")
	q := fmt.Sprintf(`SELECT %s FROM %s WHERE %s %% $1%s ORDER BY similarity(%s::text, $1) DESC LIMIT 5`,
		strings.Join(sel, ", "), qi(r.p.AnchorTable), qi(r.p.Settings.NameAttribute), live, qi(r.p.Settings.NameAttribute))
	rows, err := r.tx.QueryxContext(r.ctx, q, args...)
	if err != nil {
		return "", 0, err
	}
	defer rows.Close()
	best, bestScore := "", 0.0
	for rows.Next() {
		m := map[string]any{}
		if err := rows.MapScan(m); err != nil {
			return "", 0, err
		}
		cand, sims := map[string]any{}, map[string]float64{}
		for _, f := range trigram {
			cand[f] = "present"
			if v, ok := number(normalize(m["sim_"+f], "real")); ok {
				sims[f] = v
			}
		}
		for _, f := range exact {
			cand[f] = normalize(m["val_"+f], "")
		}
		recView := map[string]any{}
		for _, k := range mr.FuzzyKeys {
			recView[k.Field] = rec.Attrs[k.Field]
		}
		if s := FuzzyScore(mr.FuzzyKeys, recView, cand, sims); s > bestScore {
			best, bestScore = text(m["id"]), s
		}
	}
	return best, bestScore, rows.Err()
}

// mint creates a golden record's anchor row from the record's attributes
// and the profile's defaults, with a minted code.
func (r *runner) mint(rec *Record) (string, error) {
	id := uuid.NewString()
	attrs := map[string]any{}
	for k, v := range r.p.Settings.Defaults {
		attrs[k] = v
	}
	for k, v := range rec.Attrs {
		attrs[k] = v
	}
	for _, a := range r.p.Settings.Required {
		if v, ok := attrs[a]; !ok || v == nil || text(v) == "" {
			return "", fmt.Errorf("%s is required to create a golden record and no source supplied it", a)
		}
	}
	set, err := r.project(attrs, true)
	if err != nil {
		return "", err
	}
	// The stable id (the row id too, unless the anchor versions rows).
	set[r.p.entityCol()] = id
	set["tenant_id"] = r.tenant
	code := r.p.CodePrefix + strings.ToUpper(strings.ReplaceAll(id, "-", "")[:10])
	set[r.p.AnchorCodeColumn] = code
	// Derived columns: the first available of the listed sources.
	for col, from := range r.p.Settings.Derived {
		if _, ok := r.anchorCols[col]; !ok {
			continue
		}
		for _, f := range from {
			var v any
			switch {
			case f == "@code":
				v = code
			case strings.HasPrefix(f, "id:"):
				if s := rec.Identifiers[strings.TrimPrefix(f, "id:")]; s != "" {
					v = s
				}
			default:
				v = attrs[f]
			}
			if v != nil && text(v) != "" {
				set[col] = v
				break
			}
		}
	}
	cols := make([]string, 0, len(set))
	for c := range set {
		cols = append(cols, c)
	}
	sort.Strings(cols)
	ph, quoted, args := make([]string, len(cols)), make([]string, len(cols)), make([]any, len(cols))
	for i, c := range cols {
		ph[i], quoted[i], args[i] = fmt.Sprintf("$%d", i+1), qi(c), set[c]
	}
	_, err = r.tx.ExecContext(r.ctx, fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s)`, qi(r.p.AnchorTable), strings.Join(quoted, ", "), strings.Join(ph, ", ")), args...)
	return id, err
}

// project maps golden attributes onto the anchor's columns: a reference's
// code becomes its foreign key, attributes the anchor lacks are left in the
// golden record only. A required reference is checked when minting; on an
// update a reference no source supplies (e.g. a status a steward set) is
// left as it is.
func (r *runner) project(attrs map[string]any, minting bool) (map[string]any, error) {
	protected := map[string]bool{"id": true, "tenant_id": true, r.p.AnchorCodeColumn: true, r.p.entityCol(): true,
		"created_at": true, "updated_at": true, "valid_from": true, "valid_to": true}
	set := map[string]any{}
	for a, v := range attrs {
		isRef := false
		for _, ref := range r.p.Settings.References {
			if ref.Attribute == a {
				isRef = true
				id, ok := r.refIDs[a][normCode(text(v))]
				if !ok {
					return nil, fmt.Errorf("%s %q is not in %s", a, text(v), ref.RefTable)
				}
				set[ref.Column] = id
			}
		}
		if isRef || protected[a] {
			continue
		}
		if _, ok := r.anchorCols[a]; ok {
			set[a] = v
		}
	}
	for _, ref := range r.p.Settings.References {
		if _, ok := set[ref.Column]; !ok && ref.Required && minting {
			return nil, fmt.Errorf("%s is required and no source supplied %s", ref.Column, ref.Attribute)
		}
	}
	return set, nil
}

// ensureIdentifiers records the record's identifiers on its golden record.
// One already held by another golden record is a conflict for a steward,
// not a silent move.
func (r *runner) ensureIdentifiers(golden string, rec *Record) error {
	if !r.identifiers || len(rec.Identifiers) == 0 {
		return nil
	}
	id := r.p.ident()
	types := make([]string, 0, len(rec.Identifiers))
	for t := range rec.Identifiers {
		types = append(types, t)
	}
	sort.Strings(types)
	values := make([]string, len(types))
	for i, t := range types {
		values[i] = rec.Identifiers[t]
	}
	// Who already holds each identifier, in one read.
	var held []struct {
		Type  string `db:"t"`
		Owner string `db:"owner"`
	}
	if err := r.tx.SelectContext(r.ctx, &held, fmt.Sprintf(`SELECT DISTINCT ON (k.t) k.t, i.%s::text AS owner
		FROM unnest($1::text[], $2::text[]) AS k(t, v)
		JOIN %s i ON i.%s = k.t AND i.%s = k.v AND %s AND (k.t <> 'PROVIDER_CODE' OR i.%s::text = $3)
		ORDER BY k.t, (i.%s::text = $4) DESC`, qi(id.KeyColumn), qi(*r.p.IdentifierTable), qi(id.TypeColumn), qi(id.ValueColumn),
		id.identActive("i"), qi(id.SourceColumn), qi(id.KeyColumn)), pq.Array(types), pq.Array(values), r.identSource(), golden); err != nil {
		return err
	}
	owners := make(map[string]string, len(held))
	for _, h := range held {
		owners[h.Type] = h.Owner
	}
	var fresh []string
	for _, t := range types {
		switch owner, ok := owners[t]; {
		case !ok:
			fresh = append(fresh, t)
		case owner != golden:
			if err := r.exception(golden, rec.SourceKey, Issue{Code: "IDENTIFIER_CONFLICT", Severity: SevWarning, Attribute: t,
				Message: fmt.Sprintf("%s %s is already held by %s", t, rec.Identifiers[t], owner)}); err != nil {
				return err
			}
		}
	}
	if len(fresh) == 0 {
		return nil
	}
	// All the new ones at once; if one is refused, record them one by one
	// so the rest are kept.
	if len(fresh) > 1 {
		ok, err := r.inSavepoint("ident", func() error {
			for _, t := range fresh {
				if _, err := r.tx.ExecContext(r.ctx, r.identInsert(), r.identInsertArgs(golden, t, rec.Identifiers[t], t == "ISIN")...); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil || ok {
			return err
		}
	}
	for _, t := range fresh {
		v := rec.Identifiers[t]
		var insErr error
		ok, err := r.inSavepoint("ident", func() error {
			_, insErr = r.tx.ExecContext(r.ctx, r.identInsert(), r.identInsertArgs(golden, t, v, t == "ISIN")...)
			return insErr
		})
		if err != nil {
			return err
		}
		if !ok {
			// e.g. a type the identifier table doesn't accept: keep mastering.
			if err := r.exception(golden, rec.SourceKey, Issue{Code: "IDENTIFIER_REJECTED", Severity: SevWarning, Attribute: t,
				Message: fmt.Sprintf("%s %s was not recorded: %v", t, v, insErr)}); err != nil {
				return err
			}
		}
	}
	return nil
}

// inSavepoint runs fn under a savepoint: false (and nothing kept) when fn
// fails; an error only when the savepoint itself does.
func (r *runner) inSavepoint(name string, fn func() error) (bool, error) {
	if _, err := r.tx.ExecContext(r.ctx, "SAVEPOINT "+name); err != nil {
		return false, err
	}
	if fn() != nil {
		_, err := r.tx.ExecContext(r.ctx, "ROLLBACK TO SAVEPOINT "+name)
		return false, err
	}
	_, err := r.tx.ExecContext(r.ctx, "RELEASE SAVEPOINT "+name)
	return err == nil, err
}

// identInsert records an identifier for a golden record, in the table's
// layout (args: tenant, golden, type, value, source[, primary][, active]).
func (r *runner) identInsert() string {
	id := r.p.ident()
	cols := []string{"tenant_id", qi(id.KeyColumn), qi(id.TypeColumn), qi(id.ValueColumn), qi(id.SourceColumn)}
	vals := []string{"$1::uuid", "$2::uuid", "$3", "$4", "$5"}
	if id.SourceIsID {
		vals[4] = "$5::uuid"
	}
	n := 5
	if id.PrimaryColumn != "" {
		n++
		cols, vals = append(cols, qi(id.PrimaryColumn)), append(vals, fmt.Sprintf("$%d", n))
	}
	if id.ActiveIsFlag {
		cols, vals = append(cols, qi(id.ActiveColumn)), append(vals, "true")
	}
	return fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s)`, qi(*r.p.IdentifierTable), strings.Join(cols, ", "), strings.Join(vals, ", "))
}

func (r *runner) identInsertArgs(golden, typ, value string, primary bool) []any {
	args := []any{r.tenant, golden, typ, value, r.identSource()}
	if r.p.ident().PrimaryColumn != "" {
		args = append(args, primary)
	}
	return args
}

// candidate raises a possible duplicate for a steward: the new record and
// the existing one it scored close to.
func (r *runner) candidate(existing, created string, score float64, ruleCd string, matched []string) error {
	keys, _ := json.Marshal(matched)
	custom, _ := json.Marshal(map[string]any{"run_id": r.run.ID, "source": r.sourceCd})
	_, err := r.tx.ExecContext(r.ctx, fmt.Sprintf(`INSERT INTO %s (tenant_id, match_rule_id, %s, %s, overall_score, deterministic_match, matched_keys, custom_attributes)
		SELECT $1::uuid, id, $2::uuid, $3::uuid, $4, false, $5, $6 FROM %s WHERE rule_cd = $7
		ORDER BY (tenant_id = $1::uuid) DESC LIMIT 1`, r.p.table("match_candidate"), qi(r.p.TablePrefix+"_id_a"), qi(r.p.TablePrefix+"_id_b"), r.p.table("match_rule")),
		r.tenant, existing, created, score, keys, custom, ruleCd)
	return err
}

// masterOne survives a golden record from every source linked to it and
// publishes a new version when anything changed. BLOCK rules (and held
// anomalies) keep the version in REVIEW: recorded, not current.
func (r *runner) masterOne(golden string) error {
	ctx, tx := r.ctx, r.tx
	var contribs []struct {
		SourceID  string          `db:"source_system_id"`
		SourceCd  string          `db:"source_system_cd"`
		SourceKey string          `db:"source_key"`
		Payload   json.RawMessage `db:"canonical_payload"`
		MappedAt  time.Time       `db:"mapped_at"`
	}
	err := tx.SelectContext(ctx, &contribs, fmt.Sprintf(`SELECT DISTINCT ON (x.source_system_id, x.source_key)
			x.source_system_id::text, i.source_system_cd, x.source_key, i.canonical_payload, i.mapped_at
		FROM mdm.entity_xref x
		JOIN %s i ON i.source_system_id = x.source_system_id AND i.source_row_id = x.source_key AND i.is_valid
		WHERE x.entity_cd = $1 AND x.golden_id::text = $2 AND x.status = 'ACTIVE'
		ORDER BY x.source_system_id, x.source_key, i.mapped_at DESC`, qi(r.p.IncomingTable)), r.p.EntityCd, golden)
	if err != nil {
		return err
	}
	cs := make([]Contribution, 0, len(contribs))
	for _, c := range contribs {
		var p struct {
			Attrs map[string]any `json:"attrs"`
			AsOf  time.Time      `json:"as_of"`
		}
		if err := json.Unmarshal(c.Payload, &p); err != nil {
			return err
		}
		if p.AsOf.IsZero() {
			p.AsOf = c.MappedAt
		}
		cs = append(cs, Contribution{SourceID: c.SourceID, SourceCd: strings.ToUpper(c.SourceCd), SourceKey: c.SourceKey, AsOf: p.AsOf, Attrs: p.Attrs})
	}

	// One read for the record's state: its latest and published versions,
	// its code (minted by mastering, so no source supplies it) and how sure
	// its linking is (the weakest active link; certain when none is scored).
	var st struct {
		Version  sql.NullInt64   `db:"golden_version"`
		Attrs    []byte          `db:"golden_attributes"`
		Status   sql.NullString  `db:"status"`
		Current  []byte          `db:"current_attributes"`
		Code     sql.NullString  `db:"code"`
		Identity sql.NullFloat64 `db:"identity"`
	}
	gr, key := r.p.table("golden_record"), r.p.keyColumn()
	if err := tx.GetContext(ctx, &st, fmt.Sprintf(`SELECT p.golden_version, p.golden_attributes, p.status,
			(SELECT c.golden_attributes FROM %[1]s c WHERE c.%[2]s::text = $1 AND c.is_current LIMIT 1) AS current_attributes,
			(SELECT a.%[3]s::text FROM %[4]s a WHERE a.%[5]s::text = $1 AND %[6]s LIMIT 1) AS code,
			(SELECT MIN(COALESCE(x.match_score, 1)) FROM mdm.entity_xref x
				WHERE x.entity_cd = $2 AND x.golden_id::text = $1 AND x.status = 'ACTIVE') AS identity
		FROM (SELECT 1) d LEFT JOIN LATERAL (SELECT golden_version, golden_attributes, status FROM %[1]s
			WHERE %[2]s::text = $1 ORDER BY golden_version DESC LIMIT 1) p ON true`,
		gr, key, qi(r.p.AnchorCodeColumn), qi(r.p.AnchorTable), qi(r.p.entityCol()), r.p.current("a")), golden, r.p.EntityCd); err != nil {
		return err
	}
	hasPrev := st.Version.Valid
	var previous, current map[string]any // current: the published values anomalies compare to
	if hasPrev {
		_ = json.Unmarshal(st.Attrs, &previous)
		if len(st.Current) > 0 {
			_ = json.Unmarshal(st.Current, &current)
		}
	}
	prev := struct {
		Version int
		Status  string
	}{int(st.Version.Int64), st.Status.String}
	identity := 1.0
	if st.Identity.Valid {
		identity = st.Identity.Float64
	}

	now := time.Now().UTC()
	if r.e.Now != nil {
		now = r.e.Now()
	}
	decisions, anomalies := Survive(cs, r.cfg.survival, current, now,
		&SurviveOptions{Hierarchy: r.cfg.rankingFor, Select: selector(r.selRules)})
	// A steward's active override wins over the sources (and is never an
	// anomaly: it was decided, not reported).
	if err := r.applyOverrides(golden, decisions); err != nil {
		return err
	}
	kept := anomalies[:0]
	for _, a := range anomalies {
		if decisions[a.Attribute].Strategy != "OVERRIDE" {
			kept = append(kept, a)
		}
	}
	anomalies = kept
	attrs := make(map[string]any, len(decisions))
	winners := map[string]string{}
	for a, d := range decisions {
		if d.Value == nil {
			// Held with nothing to keep: the attribute has no value yet.
			delete(decisions, a)
			continue
		}
		attrs[a] = d.Value
		winners[a] = d.Winner.SourceCd
	}
	// Rules see the record as the BO does, including its own code (minted
	// by mastering, so no source supplies it - e.g. "SecId is present").
	ruleView := make(map[string]any, len(attrs)+1)
	for k, v := range attrs {
		ruleView[k] = v
	}
	if st.Code.Valid {
		ruleView[r.p.AnchorCodeColumn] = st.Code.String
	}
	issues := append(anomalies, r.rules.Golden(inFields(ruleView, r.attrField))...)
	status := "PUBLISHED"
	for _, is := range issues {
		// A held value (an anomaly, or nothing passing a selection rule)
		// waits for a steward: the version is recorded, not current.
		if is.Severity == SevError || is.Code == IssueAnomaly || is.Code == IssueSelectionHold {
			status = "REVIEW"
		}
	}

	// Idempotent: the same attributes, status and contributing source
	// values publish nothing new. A source that newly contributes (or
	// changes what it says) is a new version even when it doesn't win, so
	// the provenance shows every value that was considered.
	if hasPrev && !r.force && prev.Status == status && sameJSON(previous, attrs) {
		same, err := r.sameContributions(golden, prev.Version, decisions)
		if err != nil {
			return err
		}
		if same {
			r.counts.Unchanged++
			return nil
		}
	}

	dq := dqScore(attrs, r.cfg.survival, issues)
	version := 1
	if hasPrev {
		version = prev.Version + 1
	}
	attrsJSON, _ := json.Marshal(attrs)
	winnersJSON, _ := json.Marshal(winners)
	recordCols, recordArgs := r.recordColumns(attrs)
	publish := status == "PUBLISHED"
	if publish {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`UPDATE %s SET is_current = false,
				status = CASE WHEN status = 'PUBLISHED' THEN 'SUPERSEDED' ELSE status END
			WHERE %s::text = $1 AND is_current`, r.p.table("golden_record"), r.p.keyColumn()), golden); err != nil {
			return err
		}
	}
	args := []any{r.tenant, golden, version, publish, attrsJSON, winnersJSON, dq, identity, status, nullUUID(r.req.StartedByID)}
	extra := ""
	for i, c := range recordCols {
		args = append(args, recordArgs[i])
		extra += fmt.Sprintf(", %s", qi(c))
	}
	ph := ""
	for i := 11; i <= len(args); i++ {
		ph += fmt.Sprintf(", $%d", i)
	}
	var recordID string
	if err := tx.GetContext(ctx, &recordID, fmt.Sprintf(`INSERT INTO %s (tenant_id, %s, golden_version, is_current, effective_date, knowledge_timestamp,
			golden_attributes, winning_sources, overall_dq_score, identity_confidence, status, published_at, published_by%s)
		VALUES ($1::uuid, $2::uuid, $3, $4, CURRENT_DATE, now(), $5, $6, $7, $8, $9, CASE WHEN $4 THEN now() END, $10::uuid%s)
		RETURNING id::text`, r.p.table("golden_record"), r.p.keyColumn(), extra, ph), args...); err != nil {
		return err
	}
	if err := r.provenance(recordID, golden, version, decisions); err != nil {
		return err
	}
	if publish {
		r.counts.Published++
		if err := r.updateAnchor(golden, attrs, dq); err != nil {
			return err
		}
	} else {
		r.counts.HeldForReview++
	}
	for _, is := range issues {
		if err := r.exception(golden, "", is); err != nil {
			return err
		}
	}
	return nil
}

// sameContributions: the competing values recorded for the previous
// version are the ones considered now (source, record and value per
// attribute).
func (r *runner) sameContributions(golden string, version int, decisions map[string]Decision) (bool, error) {
	var before []string
	if err := r.tx.SelectContext(r.ctx, &before, fmt.Sprintf(`SELECT l.field_name || '|' || COALESCE(c->>'source', '') || '|' || COALESCE(c->>'source_key', '') || '|' || COALESCE(c->>'value', '')
		FROM %s l CROSS JOIN LATERAL jsonb_array_elements(CASE WHEN jsonb_typeof(l.competing_values) = 'array' THEN l.competing_values ELSE '[]'::jsonb END) c
		WHERE l.%s::text = $1 AND l.golden_version = $2`, r.p.table("survivorship_log"), r.p.keyColumn()), golden, version); err != nil {
		return false, err
	}
	now := map[string]bool{}
	for a, d := range decisions {
		for _, c := range d.Competing {
			now[a+"|"+c.SourceCd+"|"+c.SourceKey+"|"+jsonText(c.Value)] = true
		}
	}
	if len(before) != len(now) {
		return false, nil
	}
	for _, k := range before {
		if !now[k] {
			return false, nil
		}
	}
	return true, nil
}

// jsonText renders a value as Postgres's ->> would (strings bare,
// everything else as JSON).
func jsonText(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	b, _ := json.Marshal(v)
	return string(b)
}

// recordColumns fills golden-record columns the profile says come from an
// attribute (e.g. product_type_cd on product_golden_record).
func (r *runner) recordColumns(attrs map[string]any) ([]string, []any) {
	cols := make([]string, 0, len(r.p.Settings.RecordColumns))
	for c := range r.p.Settings.RecordColumns {
		cols = append(cols, c)
	}
	sort.Strings(cols)
	args := make([]any, len(cols))
	for i, c := range cols {
		v := text(attrs[r.p.Settings.RecordColumns[c]])
		if v == "" {
			v = "UNKNOWN"
		}
		args[i] = v
	}
	return cols, args
}

func (r *runner) provenance(recordID, golden string, version int, decisions map[string]Decision) error {
	attrs := make([]string, 0, len(decisions))
	for a := range decisions {
		attrs = append(attrs, a)
	}
	sort.Strings(attrs)
	if len(attrs) == 0 {
		return nil
	}
	// One statement per table for all attributes: provenance is most of a
	// golden version's writes, and each round trip costs.
	null := func(s string, ok bool) sql.NullString { return sql.NullString{String: s, Valid: ok} }
	var names, values, sources, keys, rules, reasons, competing []string
	var nums, dates, confs []sql.NullString
	for _, a := range attrs {
		d := decisions[a]
		n, isNum := number(d.Value)
		date, isDate := "", false
		if s, ok := d.Value.(string); ok && len(s) == 10 {
			if _, err := time.Parse("2006-01-02", s); err == nil {
				date, isDate = s, true
			}
		}
		comp := d.Competing
		if comp == nil {
			comp = []Candidate{}
		}
		cj, _ := json.Marshal(comp)
		names = append(names, a)
		values = append(values, text(d.Value))
		nums = append(nums, null(strconv.FormatFloat(n, 'f', -1, 64), isNum))
		dates = append(dates, null(date, isDate))
		sources = append(sources, d.Winner.SourceID)
		keys = append(keys, truncate(d.Winner.SourceKey, 150))
		confs = append(confs, null(strconv.FormatFloat(d.Confidence, 'f', -1, 64), true))
		rules = append(rules, d.RuleID)
		reasons = append(reasons, truncate(d.Strategy+": "+d.Reason, 255))
		competing = append(competing, string(cj))
	}
	if _, err := r.tx.ExecContext(r.ctx, fmt.Sprintf(`INSERT INTO %s (tenant_id, golden_record_id, field_name, field_value, field_value_numeric,
			field_value_date, source_system_id, source_field, confidence, rule_applied)
		SELECT $1::uuid, $2::uuid, f.name, f.value, f.num::numeric, f.dt::date, NULLIF(f.src, '')::uuid, f.skey, f.conf::numeric, NULLIF(f.rule, '')::uuid
		FROM unnest($3::text[], $4::text[], $5::text[], $6::text[], $7::text[], $8::text[], $9::text[], $10::text[])
			AS f(name, value, num, dt, src, skey, conf, rule)`, r.p.table("golden_field")),
		r.tenant, recordID, pq.Array(names), pq.Array(values), pq.Array(nums), pq.Array(dates), pq.Array(sources), pq.Array(keys),
		pq.Array(confs), pq.Array(rules)); err != nil {
		return err
	}
	_, err := r.tx.ExecContext(r.ctx, fmt.Sprintf(`INSERT INTO %s (tenant_id, %s, golden_version, field_name, winning_source_id,
			winning_value, competing_values, decision_reason)
		SELECT $1::uuid, $2::uuid, $3, f.name, NULLIF(f.src, '')::uuid, f.value, f.comp::jsonb, f.reason
		FROM unnest($4::text[], $5::text[], $6::text[], $7::text[], $8::text[]) AS f(name, src, value, comp, reason)`,
		r.p.table("survivorship_log"), r.p.keyColumn()),
		r.tenant, golden, version, pq.Array(names), pq.Array(sources), pq.Array(values), pq.Array(competing), pq.Array(reasons))
	return err
}

func (r *runner) updateAnchor(golden string, attrs map[string]any, dq float64) error {
	set, err := r.project(attrs, false)
	if err != nil {
		return err
	}
	if _, ok := r.anchorCols["dq_score"]; ok {
		set["dq_score"] = dq
	}
	if _, ok := r.anchorCols["updated_at"]; ok {
		set["updated_at"] = time.Now().UTC()
	}
	return r.writeAnchor(golden, set)
}

// writeAnchor applies set to the golden record's anchor row. In place: an
// update. Bitemporal: the current row is closed (valid_to) and a new
// version inserted - a copy of it with set applied - so history is kept.
func (r *runner) writeAnchor(golden string, set map[string]any) error {
	if len(set) == 0 {
		return nil
	}
	cols := make([]string, 0, len(set))
	for c := range set {
		cols = append(cols, c)
	}
	sort.Strings(cols)
	ent := qi(r.p.entityCol())
	if !r.p.bitemporal() {
		args := []any{golden}
		parts := make([]string, len(cols))
		for i, c := range cols {
			args = append(args, set[c])
			parts[i] = fmt.Sprintf("%s = $%d", qi(c), i+2)
		}
		_, err := r.tx.ExecContext(r.ctx, fmt.Sprintf(`UPDATE %s SET %s WHERE %s::text = $1`, qi(r.p.AnchorTable), strings.Join(parts, ", "), ent), args...)
		return err
	}
	// Nothing actually changing (ignoring updated_at) writes no version.
	var cmp []string
	cargs := []any{golden}
	for _, c := range cols {
		if c == "updated_at" {
			continue
		}
		cargs = append(cargs, jsonText(set[c]))
		cmp = append(cmp, fmt.Sprintf("%s::text IS NOT DISTINCT FROM $%d", qi(c), len(cargs)))
	}
	if len(cmp) > 0 {
		var same bool
		if err := r.tx.GetContext(r.ctx, &same, fmt.Sprintf(`SELECT COALESCE(bool_and(%s), false) FROM %s WHERE %s::text = $1 AND %s`,
			strings.Join(cmp, " AND "), qi(r.p.AnchorTable), ent, r.p.current("")), cargs...); err == nil && same {
			return nil
		}
	}
	// Close the current version, then insert its successor from it.
	var rowID string
	if err := r.tx.GetContext(r.ctx, &rowID, fmt.Sprintf(`UPDATE %s SET "valid_to" = now() WHERE %s::text = $1 AND %s RETURNING id::text`,
		qi(r.p.AnchorTable), ent, r.p.current("")), golden); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return msgNoGolden(golden)
		}
		return err
	}
	all := make([]string, 0, len(r.anchorCols))
	for c := range r.anchorCols {
		all = append(all, c)
	}
	sort.Strings(all)
	args := []any{rowID}
	var into, from []string
	for _, c := range all {
		switch c {
		case "id":
			continue // a new row id (the column's default)
		case "valid_from":
			into, from = append(into, qi(c)), append(from, "now()")
			continue
		case "valid_to":
			into, from = append(into, qi(c)), append(from, "NULL")
			continue
		}
		into = append(into, qi(c))
		if v, ok := set[c]; ok {
			args = append(args, v)
			from = append(from, fmt.Sprintf("$%d", len(args)))
		} else {
			from = append(from, "a."+qi(c))
		}
	}
	_, err := r.tx.ExecContext(r.ctx, fmt.Sprintf(`INSERT INTO %s (%s) SELECT %s FROM %s a WHERE a.id::text = $1`,
		qi(r.p.AnchorTable), strings.Join(into, ", "), strings.Join(from, ", "), qi(r.p.AnchorTable)), args...)
	return err
}

// dqScore: how complete the golden record is against the attributes the
// entity masters (those with a survivorship rule, plus any a source sent),
// less 10 points per warning and 25 per blocking failure.
func dqScore(attrs map[string]any, rules map[string]SurvivalRule, issues []Issue) float64 {
	expected := map[string]bool{}
	for a := range rules {
		expected[a] = true
	}
	for a := range attrs {
		expected[a] = true
	}
	if len(expected) == 0 {
		return 0
	}
	filled := 0
	for a := range expected {
		if v, ok := attrs[a]; ok && v != nil && text(v) != "" {
			filled++
		}
	}
	score := 100 * float64(filled) / float64(len(expected))
	for _, is := range issues {
		if is.Severity == SevError {
			score -= 25
		} else {
			score -= 10
		}
	}
	return math.Round(math.Max(0, score)*100) / 100
}

func sameJSON(a, b map[string]any) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return string(x) == string(y)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func nullUUID(s string) any {
	if _, err := uuid.Parse(s); err != nil {
		return nil
	}
	return s
}
