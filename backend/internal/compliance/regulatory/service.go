package regulatory

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/hondyman/uisce/backend/internal/compliance/canonical"
	"github.com/hondyman/uisce/backend/internal/compliance/drift"
)

type IntakeRequest struct {
	CaseCode        string     `json:"case_code"`
	Source          CaseSource `json:"source"`
	SourceReference string     `json:"source_reference"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	CreatedBy       string     `json:"created_by"`
	TTL             time.Duration `json:"ttl"` // default 30 days
}

type TriageRequest struct {
	CaseID          uuid.UUID          `json:"case_id"`
	Classification  CaseClassification `json:"classification"`
	TriageNotes     string             `json:"triage_notes"`
	TriagedBy       string             `json:"triaged_by"`
	AffectedRuleIDs []uuid.UUID        `json:"affected_rule_ids"`
}

type RuleDraft struct {
	RuleID              uuid.UUID              `json:"rule_id"`
	NewAST              map[string]interface{} `json:"new_ast"`
	NewThresholds       map[string]interface{} `json:"new_thresholds"`
	NewCitation         string                 `json:"new_citation"`
	EffectiveFrom       time.Time              `json:"effective_from"`
	TestCorpus          []drift.ScenarioTestCase `json:"test_corpus"`
}

type PublishRequest struct {
	CaseID    uuid.UUID   `json:"case_id"`
	StewardID string      `json:"steward_id"`
	Drafts    []RuleDraft `json:"drafts"`
}

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// CreateCase opens a new regulatory change case
func (s *Service) CreateCase(ctx context.Context, req IntakeRequest) (*RegulatoryChangeCase, error) {
	if req.CaseCode == "" {
		return nil, errors.New("case_code is required")
	}
	if req.Title == "" || req.Description == "" {
		return nil, errors.New("title and description are required")
	}
	if req.CreatedBy == "" {
		req.CreatedBy = "system"
	}
	ttl := req.TTL
	if ttl <= 0 {
		ttl = 30 * 24 * time.Hour
	}
	dueAt := time.Now().UTC().Add(ttl)

	caseID := uuid.New()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO compliance.regulatory_change_case (
			id, case_code, source, source_reference, title, description,
			status, due_at, created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			'INTAKED', $7, $8, now(), now()
		)
	`, caseID, req.CaseCode, req.Source, req.SourceReference, req.Title, req.Description, dueAt, req.CreatedBy)
	if err != nil {
		return nil, fmt.Errorf("insert case: %w", err)
	}

	payloadJSON, _ := json.Marshal(map[string]interface{}{
		"source":           req.Source,
		"source_reference": req.SourceReference,
		"title":            req.Title,
		"due_at":           dueAt,
	})

	_, err = tx.ExecContext(ctx, `
		INSERT INTO compliance.regulatory_case_event (
			id, case_id, event_type, actor, payload, created_at
		) VALUES (
			gen_random_uuid(), $1, 'CASE_OPENED', $2, $3::jsonb, now()
		)
	`, caseID, req.CreatedBy, string(payloadJSON))
	if err != nil {
		return nil, fmt.Errorf("insert case opened event: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return s.GetCaseByID(ctx, caseID)
}

// TriageCase processes stage 2 triage and transitions INTAKED -> TRIAGED
func (s *Service) TriageCase(ctx context.Context, req TriageRequest) error {
	if req.Classification == "" {
		return errors.New("classification is required for triage")
	}
	if req.TriagedBy == "" {
		return errors.New("triaged_by is required")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	affectedIDs := req.AffectedRuleIDs
	if affectedIDs == nil {
		affectedIDs = []uuid.UUID{}
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE compliance.regulatory_change_case
		SET 
			classification = $1,
			triage_notes = $2,
			triaged_by = $3,
			triaged_at = now(),
			affected_rule_ids = $4,
			status = 'TRIAGED',
			updated_at = now()
		WHERE id = $5
	`, req.Classification, req.TriageNotes, req.TriagedBy, pq.Array(affectedIDs), req.CaseID)
	if err != nil {
		return fmt.Errorf("update case to TRIAGED: %w", err)
	}

	payloadJSON, _ := json.Marshal(map[string]interface{}{
		"classification":    req.Classification,
		"triage_notes":      req.TriageNotes,
		"affected_rule_ids": affectedIDs,
	})

	_, err = tx.ExecContext(ctx, `
		INSERT INTO compliance.regulatory_case_event (
			id, case_id, event_type, actor, payload, created_at
		) VALUES (
			gen_random_uuid(), $1, 'TRIAGED', $2, $3::jsonb, now()
		)
	`, req.CaseID, req.TriagedBy, string(payloadJSON))
	if err != nil {
		return fmt.Errorf("insert case triaged event: %w", err)
	}

	return tx.Commit()
}

// StartReview transitions TRIAGED -> UNDER_REVIEW
func (s *Service) StartReview(ctx context.Context, caseID uuid.UUID, stewardID string, diffs []RuleDiffView) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		UPDATE compliance.regulatory_change_case
		SET status = 'UNDER_REVIEW', updated_at = now()
		WHERE id = $1
	`, caseID)
	if err != nil {
		return fmt.Errorf("update case to UNDER_REVIEW: %w", err)
	}

	payloadJSON, _ := json.Marshal(map[string]interface{}{
		"diff_views": diffs,
	})

	_, err = tx.ExecContext(ctx, `
		INSERT INTO compliance.regulatory_case_event (
			id, case_id, event_type, actor, payload, created_at
		) VALUES (
			gen_random_uuid(), $1, 'REVIEW_STARTED', $2, $3::jsonb, now()
		)
	`, caseID, stewardID, string(payloadJSON))
	if err != nil {
		return fmt.Errorf("insert review started event: %w", err)
	}

	return tx.Commit()
}

// ExecuteCorpusGate runs scenario tests, asserts active status (no PROVISIONAL on semantic changes), and records results
func (s *Service) ExecuteCorpusGate(ctx context.Context, caseID uuid.UUID, stewardID string, drafts []RuleDraft) (*drift.CorpusRunResult, error) {
	repinSvc := drift.NewRepinService(s.db)
	totalResult := &drift.CorpusRunResult{
		AllPassed: true,
	}

	for _, d := range drafts {
		// 1. Check rule library status
		var ruleCode, libStatus string
		err := s.db.QueryRowContext(ctx, `
			SELECT rule_code, coalesce(library_status, 'ACTIVE')
			FROM compliance.compliance_rule
			WHERE id = $1
		`, d.RuleID).Scan(&ruleCode, &libStatus)
		if err != nil {
			return nil, fmt.Errorf("fetch rule %s status: %w", d.RuleID, err)
		}

		if libStatus == "PROVISIONAL" {
			return nil, fmt.Errorf("corpus gate rejected: rule %s (%s) has PROVISIONAL library status and cannot undergo regulatory modification without gold-copy promotion", ruleCode, d.RuleID)
		}

		// 2. Execute test corpus
		res := repinSvc.ExecuteTestCorpus(d.TestCorpus)
		totalResult.TotalCases += res.TotalCases
		totalResult.PassedCases += res.PassedCases
		totalResult.FailedCases += res.FailedCases
		if !res.AllPassed {
			totalResult.AllPassed = false
			totalResult.FailureNotes = append(totalResult.FailureNotes, fmt.Sprintf("Rule %s failed %d/%d test cases", ruleCode, res.FailedCases, res.TotalCases))
		}
	}

	payloadJSON, _ := json.Marshal(map[string]interface{}{
		"corpus_result": totalResult,
		"steward_id":    stewardID,
	})

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO compliance.regulatory_case_event (
			id, case_id, event_type, actor, payload, created_at
		) VALUES (
			gen_random_uuid(), $1, 'CORPUS_RUN', $2, $3::jsonb, now()
		)
	`, caseID, stewardID, string(payloadJSON))
	if err != nil {
		return nil, fmt.Errorf("insert corpus run event: %w", err)
	}

	if !totalResult.AllPassed {
		return totalResult, fmt.Errorf("corpus gate validation failed: %s", strings.Join(totalResult.FailureNotes, "; "))
	}

	return totalResult, nil
}

// ApproveCase transitions UNDER_REVIEW -> APPROVED_FOR_PUBLISH
func (s *Service) ApproveCase(ctx context.Context, caseID uuid.UUID, stewardID string, notes string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		UPDATE compliance.regulatory_change_case
		SET status = 'APPROVED_FOR_PUBLISH', updated_at = now()
		WHERE id = $1
	`, caseID)
	if err != nil {
		return fmt.Errorf("update case to APPROVED_FOR_PUBLISH: %w", err)
	}

	payloadJSON, _ := json.Marshal(map[string]interface{}{
		"steward_id": stewardID,
		"notes":      notes,
	})

	_, err = tx.ExecContext(ctx, `
		INSERT INTO compliance.regulatory_case_event (
			id, case_id, event_type, actor, payload, created_at
		) VALUES (
			gen_random_uuid(), $1, 'APPROVED', $2, $3::jsonb, now()
		)
	`, caseID, stewardID, string(payloadJSON))
	if err != nil {
		return fmt.Errorf("insert approved event: %w", err)
	}

	return tx.Commit()
}

// PublishRelease performs atomic Core release: snapshots + rule updates + current_version + drift flags + governance audit + fanout notifications
func (s *Service) PublishRelease(ctx context.Context, req PublishRequest) ([]PublishedRuleVersionEntry, error) {
	if req.StewardID == "" {
		return nil, errors.New("steward_id is required for publish release")
	}
	if len(req.Drafts) == 0 {
		return nil, errors.New("no rule drafts provided for release")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// 1. Fetch case details
	var caseCode string
	err = tx.QueryRowContext(ctx, `
		SELECT case_code
		FROM compliance.regulatory_change_case
		WHERE id = $1
	`, req.CaseID).Scan(&caseCode)
	if err != nil {
		return nil, fmt.Errorf("fetch case code: %w", err)
	}

	publishedEntries := make([]PublishedRuleVersionEntry, 0, len(req.Drafts))

	for _, d := range req.Drafts {
		// Fetch current rule state
		var currentVer int
		var ruleCode string
		var tenantID uuid.UUID
		err = tx.QueryRowContext(ctx, `
			SELECT rule_code, current_version, tenant_id
			FROM compliance.compliance_rule
			WHERE id = $1
		`, d.RuleID).Scan(&ruleCode, &currentVer, &tenantID)
		if err != nil {
			return nil, fmt.Errorf("fetch current rule %s: %w", d.RuleID, err)
		}

		newVer := currentVer + 1

		// JCS canonical content hash computation
		astBytes, _ := json.Marshal(d.NewAST)
		paramBytes, _ := json.Marshal(d.NewThresholds)
		contentHash, err := canonical.ComputeRuleContentHashFromRaw(astBytes, paramBytes, d.NewCitation)
		if err != nil {
			return nil, fmt.Errorf("compute content hash for rule %s: %w", ruleCode, err)
		}
		bytecodeHash := canonical.ComputeBytecodeHash(nil)

		// 2. Insert new version snapshot into compliance_rule_version
		_, err = tx.ExecContext(ctx, `
			INSERT INTO compliance.compliance_rule_version (
				rule_id, version, tenant_id, resolved_ast, parameter_thresholds,
				citation, effective_from, effective_to, content_hash, compiled_bytecode_hash,
				created_by, created_at
			) VALUES (
				$1, $2, $3, $4::jsonb, $5::jsonb,
				$6, $7, null, $8, $9,
				$10, now()
			)
		`, d.RuleID, newVer, tenantID, string(astBytes), string(paramBytes), d.NewCitation, d.EffectiveFrom, contentHash, bytecodeHash, req.StewardID)
		if err != nil {
			return nil, fmt.Errorf("insert snapshot v%d for rule %s: %w", newVer, ruleCode, err)
		}

		// 3. Update compliance.compliance_rule with new current_version and thresholds
		_, err = tx.ExecContext(ctx, `
			UPDATE compliance.compliance_rule
			SET 
				current_version = $1,
				ast_condition = $2::jsonb,
				parameter_thresholds = $3::jsonb,
				citation = $4,
				effective_from = $5,
				updated_at = now()
			WHERE id = $6
		`, newVer, string(astBytes), string(paramBytes), d.NewCitation, d.EffectiveFrom, d.RuleID)
		if err != nil {
			return nil, fmt.Errorf("update rule %s to v%d: %w", ruleCode, newVer, err)
		}

		// 4. Update extend-mode tenant rules pointing to this core rule -> drift_status = CORE_VERSION_UPDATED
		_, err = tx.ExecContext(ctx, `
			UPDATE compliance.compliance_rule
			SET drift_status = 'CORE_VERSION_UPDATED',
			    updated_at = now()
			WHERE core_rule_id = $1 AND inherit_mode = 'extend' AND valid_to IS NULL
		`, d.RuleID)
		if err != nil {
			return nil, fmt.Errorf("flag extend tenants for core rule %s: %w", ruleCode, err)
		}

		publishedEntries = append(publishedEntries, PublishedRuleVersionEntry{
			RuleID:      d.RuleID,
			RuleCode:    ruleCode,
			FromVersion: currentVer,
			ToVersion:   newVer,
			ContentHash: contentHash,
		})

		// 5. Fan-out notifications to tenants
		// For inherit tenants: INHERIT_ADVANCE
		// For extend tenants: DRIFT_FLAG
		notifPayloadInherit, _ := json.Marshal(map[string]interface{}{
			"case_code":      caseCode,
			"rule_code":      ruleCode,
			"from_version":   currentVer,
			"to_version":     newVer,
			"effective_from": d.EffectiveFrom,
			"citation":       d.NewCitation,
		})
		_, err = tx.ExecContext(ctx, `
			INSERT INTO compliance.compliance_notification (
				id, tenant_id, kind, title, payload, is_read, created_at
			)
			SELECT 
				gen_random_uuid(),
				t.id,
				'INHERIT_ADVANCE',
				format('Rule %s Advanced to Version %s (Effective %s)', $1::text, $2::text, $3::text),
				$4::jsonb,
				false,
				now()
			FROM public.tenants t
			WHERE t.is_active = true AND (t.gold_copy IS NULL OR t.gold_copy = false)
		`, ruleCode, fmt.Sprintf("%d", newVer), d.EffectiveFrom.Format("2006-01-02"), string(notifPayloadInherit))
		if err != nil {
			return nil, fmt.Errorf("fanout inherit notifications: %w", err)
		}

		notifPayloadExtend, _ := json.Marshal(map[string]interface{}{
			"case_code":      caseCode,
			"rule_code":      ruleCode,
			"pinned_version": currentVer,
			"new_version":    newVer,
			"effective_from": d.EffectiveFrom,
			"stale_deadline": d.EffectiveFrom.Add(90 * 24 * time.Hour),
		})
		_, err = tx.ExecContext(ctx, `
			INSERT INTO compliance.compliance_notification (
				id, tenant_id, kind, title, payload, is_read, created_at
			)
			SELECT DISTINCT
				gen_random_uuid(),
				r.tenant_id,
				'DRIFT_FLAG',
				format('Core Version Updated on Extended Rule %s: Action Required', $1::text),
				$2::jsonb,
				false,
				now()
			FROM compliance.compliance_rule r
			WHERE r.core_rule_id = $3 AND r.inherit_mode = 'extend' AND r.valid_to IS NULL
		`, ruleCode, string(notifPayloadExtend), d.RuleID)
		// 8. Record Governance Audit Event (REGULATORY_CHANGE_PUBLISHED)
		_, err = tx.ExecContext(ctx, `
			INSERT INTO compliance.governance_audit_event (
				id, tenant_id, rule_id, event_type, old_pinned_version, new_pinned_version,
				steward_id, steward_notes, ast_diff, corpus_run_results, created_at
			) VALUES (
				gen_random_uuid(), public.uisce_gold_copy_tenant_id(), $1, 'REGULATORY_CHANGE_PUBLISHED',
				$2, $3, $4, $5, $6::jsonb, '{}'::jsonb, now()
			)
		`, d.RuleID, currentVer, newVer, req.StewardID, fmt.Sprintf("Case %s: %s", caseCode, d.NewCitation), string(astBytes))
		if err != nil {
			return nil, fmt.Errorf("insert governance audit event for rule %s: %w", ruleCode, err)
		}
	}

	// 6. Transition case to PUBLISHED
	publishedVersionsJSON, _ := json.Marshal(publishedEntries)
	_, err = tx.ExecContext(ctx, `
		UPDATE compliance.regulatory_change_case
		SET 
			status = 'PUBLISHED',
			published_rule_versions = $1::jsonb,
			updated_at = now()
		WHERE id = $2
	`, string(publishedVersionsJSON), req.CaseID)
	if err != nil {
		return nil, fmt.Errorf("transition case to PUBLISHED: %w", err)
	}

	// 7. Record Case Event
	_, err = tx.ExecContext(ctx, `
		INSERT INTO compliance.regulatory_case_event (
			id, case_id, event_type, actor, payload, created_at
		) VALUES (
			gen_random_uuid(), $1, 'PUBLISHED', $2, $3::jsonb, now()
		)
	`, req.CaseID, req.StewardID, string(publishedVersionsJSON))
	if err != nil {
		return nil, fmt.Errorf("insert case published event: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit publish release tx: %w", err)
	}

	return publishedEntries, nil
}

// EscalateCase writes ESCALATED event and emits notification
func (s *Service) EscalateCase(ctx context.Context, caseID uuid.UUID) error {
	var caseCode, title string
	var dueAt time.Time
	err := s.db.QueryRowContext(ctx, `
		SELECT case_code, title, due_at
		FROM compliance.regulatory_change_case
		WHERE id = $1
	`, caseID).Scan(&caseCode, &title, &dueAt)
	if err != nil {
		return fmt.Errorf("fetch case for escalation: %w", err)
	}

	payloadJSON, _ := json.Marshal(map[string]interface{}{
		"case_code":    caseCode,
		"due_at":       dueAt,
		"escalated_at": time.Now().UTC(),
		"reason":       "Case exceeded SLA TTL without resolution",
	})

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO compliance.regulatory_case_event (
			id, case_id, event_type, actor, payload, created_at
		) VALUES (
			gen_random_uuid(), $1, 'ESCALATED', 'temporal_ttl_monitor', $2::jsonb, now()
		)
	`, caseID, string(payloadJSON))
	if err != nil {
		return fmt.Errorf("insert case escalated event: %w", err)
	}

	// Insert platform notification
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO compliance.compliance_notification (
			id, tenant_id, kind, title, payload, is_read, created_at
		) VALUES (
			gen_random_uuid(), public.uisce_gold_copy_tenant_id(), 'CASE_ESCALATED',
			format('Regulatory Case %s Escalated: SLA Exceeded', $1),
			$2::jsonb, false, now()
		)
	`, caseCode, string(payloadJSON))
	if err != nil {
		return fmt.Errorf("insert escalation notification: %w", err)
	}

	return nil
}

// RejectCase transitions case to REJECTED
func (s *Service) RejectCase(ctx context.Context, caseID uuid.UUID, actor, reason string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		UPDATE compliance.regulatory_change_case
		SET status = 'REJECTED', updated_at = now()
		WHERE id = $1
	`, caseID)
	if err != nil {
		return fmt.Errorf("update case to REJECTED: %w", err)
	}

	payloadJSON, _ := json.Marshal(map[string]interface{}{
		"actor":  actor,
		"reason": reason,
	})

	_, err = tx.ExecContext(ctx, `
		INSERT INTO compliance.regulatory_case_event (
			id, case_id, event_type, actor, payload, created_at
		) VALUES (
			gen_random_uuid(), $1, 'REJECTED', $2, $3::jsonb, now()
		)
	`, caseID, actor, string(payloadJSON))
	if err != nil {
		return fmt.Errorf("insert case rejected event: %w", err)
	}

	return tx.Commit()
}

// CloseCaseNoImpact transitions case to CLOSED_NO_IMPACT
func (s *Service) CloseCaseNoImpact(ctx context.Context, caseID uuid.UUID, actor, reason string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		UPDATE compliance.regulatory_change_case
		SET status = 'CLOSED_NO_IMPACT', updated_at = now()
		WHERE id = $1
	`, caseID)
	if err != nil {
		return fmt.Errorf("update case to CLOSED_NO_IMPACT: %w", err)
	}

	payloadJSON, _ := json.Marshal(map[string]interface{}{
		"actor":  actor,
		"reason": reason,
	})

	_, err = tx.ExecContext(ctx, `
		INSERT INTO compliance.regulatory_case_event (
			id, case_id, event_type, actor, payload, created_at
		) VALUES (
			gen_random_uuid(), $1, 'CLOSED', $2, $3::jsonb, now()
		)
	`, caseID, actor, string(payloadJSON))
	if err != nil {
		return fmt.Errorf("insert case closed event: %w", err)
	}

	return tx.Commit()
}

// GetCaseByID fetches a single case
func (s *Service) GetCaseByID(ctx context.Context, id uuid.UUID) (*RegulatoryChangeCase, error) {
	var c RegulatoryChangeCase
	var affectedRules pq.ByteaArray
	_ = affectedRules
	var class sql.NullString
	var triageNotes, triagedBy sql.NullString
	var triagedAt sql.NullTime
	var pubVersions []byte

	var rawRuleIDs []string
	err := s.db.QueryRowContext(ctx, `
		SELECT 
			id, case_code, source, source_reference, title, description,
			affected_rule_ids, classification, triage_notes, triaged_by, triaged_at,
			status, published_rule_versions, due_at, created_by, created_at, updated_at
		FROM compliance.regulatory_change_case
		WHERE id = $1
	`, id).Scan(
		&c.ID, &c.CaseCode, &c.Source, &c.SourceReference, &c.Title, &c.Description,
		pq.Array(&rawRuleIDs), &class, &triageNotes, &triagedBy, &triagedAt,
		&c.Status, &pubVersions, &c.DueAt, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	c.AffectedRuleIDs = make([]uuid.UUID, 0, len(rawRuleIDs))
	for _, rid := range rawRuleIDs {
		if parsed, err := uuid.Parse(rid); err == nil {
			c.AffectedRuleIDs = append(c.AffectedRuleIDs, parsed)
		}
	}

	if class.Valid {
		cc := CaseClassification(class.String)
		c.Classification = &cc
	}
	if triageNotes.Valid {
		c.TriageNotes = triageNotes.String
	}
	if triagedBy.Valid {
		c.TriagedBy = triagedBy.String
	}
	if triagedAt.Valid {
		c.TriagedAt = &triagedAt.Time
	}
	if len(pubVersions) > 0 {
		_ = json.Unmarshal(pubVersions, &c.PublishedRuleVersions)
	}

	return &c, nil
}

// GetStewardTriageView builds structured output formatted for Page Designer UI components
func (s *Service) GetStewardTriageView(ctx context.Context, caseCode string) (*StewardTriageView, error) {
	var caseID uuid.UUID
	err := s.db.QueryRowContext(ctx, `
		SELECT id FROM compliance.regulatory_change_case WHERE case_code = $1
	`, caseCode).Scan(&caseID)
	if err != nil {
		return nil, fmt.Errorf("find case by code %s: %w", caseCode, err)
	}

	c, err := s.GetCaseByID(ctx, caseID)
	if err != nil {
		return nil, err
	}

	view := &StewardTriageView{
		CaseCode:        c.CaseCode,
		Title:           c.Title,
		Source:          c.Source,
		SourceReference: c.SourceReference,
		Description:     c.Description,
		DueAt:           c.DueAt,
		Status:          c.Status,
		AffectedRules:   make([]AffectedRuleSummary, 0),
		DiffViews:       make([]RuleDiffView, 0),
	}

	for _, rid := range c.AffectedRuleIDs {
		var code, name, citation, libStatus string
		var curVer int
		var threshJSON []byte
		err := s.db.QueryRowContext(ctx, `
			SELECT rule_code, name, citation, current_version, coalesce(library_status, 'ACTIVE'), parameter_thresholds
			FROM compliance.compliance_rule
			WHERE id = $1
		`, rid).Scan(&code, &name, &citation, &curVer, &libStatus, &threshJSON)
		if err == nil {
			var thresh map[string]interface{}
			_ = json.Unmarshal(threshJSON, &thresh)
			view.AffectedRules = append(view.AffectedRules, AffectedRuleSummary{
				RuleID:         rid,
				RuleCode:       code,
				Name:           name,
				CurrentVersion: curVer,
				LibraryStatus:  libStatus,
				Citation:       citation,
				Thresholds:     thresh,
			})
		}
	}

	// Signals available based on state machine
	switch c.Status {
	case StatusIntaked:
		view.AvailableSignals = []string{"Triage(classification, notes)", "Reject(reason)"}
	case StatusTriaged:
		view.AvailableSignals = []string{"StartReview()", "CloseNoImpact(reason)", "Reject(reason)"}
	case StatusUnderReview:
		view.AvailableSignals = []string{"ExecuteCorpusGate()", "ApproveForPublish()", "Reject(reason)"}
	case StatusApprovedForPublish:
		view.AvailableSignals = []string{"PublishRelease()", "Reject(reason)"}
	default:
		view.AvailableSignals = []string{}
	}

	return view, nil
}
