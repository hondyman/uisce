package regulatory

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/hondyman/uisce/backend/internal/compliance"
	"github.com/hondyman/uisce/backend/internal/compliance/canonical"
	"github.com/hondyman/uisce/backend/internal/compliance/drift"
)

type IntakeRequest struct {
	CaseCode        string        `json:"case_code"`
	Source          CaseSource    `json:"source"`
	SourceReference string        `json:"source_reference"`
	Title           string        `json:"title"`
	Description     string        `json:"description"`
	CreatedBy       string        `json:"created_by"`
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
	RuleID              uuid.UUID                `json:"rule_id"`
	NewAST              map[string]interface{}   `json:"new_ast"`
	NewThresholds       map[string]interface{}   `json:"new_thresholds"`
	NewCitation         string                   `json:"new_citation"`
	EffectiveFrom       time.Time                `json:"effective_from"`
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
		return nil, fmt.Errorf("insert case event: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return s.GetCase(ctx, caseID)
}

// GetCase retrieves a single regulatory change case
func (s *Service) GetCase(ctx context.Context, caseID uuid.UUID) (*RegulatoryChangeCase, error) {
	var c RegulatoryChangeCase
	var classStr sql.NullString
	var pubVerJSON []byte
	var affUUIDs []string

	err := s.db.QueryRowContext(ctx, `
		SELECT 
			id, case_code, source, coalesce(source_reference, ''), title, description,
			affected_rule_ids, classification, coalesce(triage_notes, ''), coalesce(triaged_by, ''),
			triaged_at, status, is_escalated, escalation_count, escalated_at, last_escalated_at,
			published_rule_versions, due_at, created_by, created_at, updated_at
		FROM compliance.regulatory_change_case
		WHERE id = $1
	`, caseID).Scan(
		&c.ID, &c.CaseCode, &c.Source, &c.SourceReference, &c.Title, &c.Description,
		pq.Array(&affUUIDs), &classStr, &c.TriageNotes, &c.TriagedBy,
		&c.TriagedAt, &c.Status, &c.IsEscalated, &c.EscalationCount, &c.EscalatedAt, &c.LastEscalatedAt,
		&pubVerJSON, &c.DueAt, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if classStr.Valid {
		cl := CaseClassification(classStr.String)
		c.Classification = &cl
	}

	c.AffectedRuleIDs = make([]uuid.UUID, 0, len(affUUIDs))
	for _, idStr := range affUUIDs {
		if parsed, err := uuid.Parse(idStr); err == nil {
			c.AffectedRuleIDs = append(c.AffectedRuleIDs, parsed)
		}
	}

	if len(pubVerJSON) > 0 {
		_ = json.Unmarshal(pubVerJSON, &c.PublishedRuleVersions)
	}

	return &c, nil
}

// TriageCase records classification and affected rules
func (s *Service) TriageCase(ctx context.Context, req TriageRequest) error {
	if req.Classification == "" {
		return errors.New("classification is required for triage")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	affUUIDs := make([]string, len(req.AffectedRuleIDs))
	for i, id := range req.AffectedRuleIDs {
		affUUIDs[i] = id.String()
	}

	targetStatus := StatusTriaged
	if req.Classification == ClassificationNewRuleRequired {
		targetStatus = StatusNewRuleBacklog
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE compliance.regulatory_change_case
		SET 
			classification = $1,
			triage_notes = $2,
			triaged_by = $3,
			triaged_at = now(),
			affected_rule_ids = $4::uuid[],
			status = $5,
			updated_at = now()
		WHERE id = $6
	`, req.Classification, req.TriageNotes, req.TriagedBy, pq.Array(affUUIDs), targetStatus, req.CaseID)
	if err != nil {
		return fmt.Errorf("update case for triage: %w", err)
	}

	payloadJSON, _ := json.Marshal(map[string]interface{}{
		"classification":    req.Classification,
		"triage_notes":      req.TriageNotes,
		"triaged_by":        req.TriagedBy,
		"affected_rule_ids": req.AffectedRuleIDs,
		"status":            targetStatus,
	})

	eventType := EventTriaged
	if req.Classification == ClassificationNewRuleRequired {
		eventType = EventNewRuleRouted
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO compliance.regulatory_case_event (
			id, case_id, event_type, actor, payload, created_at
		) VALUES (
			gen_random_uuid(), $1, $2, $3, $4::jsonb, now()
		)
	`, req.CaseID, eventType, req.TriagedBy, string(payloadJSON))
	if err != nil {
		return fmt.Errorf("insert triage event: %w", err)
	}

	return tx.Commit()
}

// StartReview transitions case to UNDER_REVIEW
func (s *Service) StartReview(ctx context.Context, caseID uuid.UUID, stewardID string, diffViews []RuleDiffView) error {
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
		"steward_id": stewardID,
		"diff_views": diffViews,
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

// SaveDrafts persists durable draft rules with computed RFC 8785 content hashes
func (s *Service) SaveDrafts(ctx context.Context, caseID uuid.UUID, drafts []RuleDraft) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	for _, d := range drafts {
		astBytes, _ := json.Marshal(d.NewAST)
		paramBytes, _ := json.Marshal(d.NewThresholds)
		contentHash, err := canonical.ComputeRuleContentHashFromRaw(astBytes, paramBytes, d.NewCitation)
		if err != nil {
			return fmt.Errorf("compute draft content hash for rule %s: %w", d.RuleID, err)
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO compliance.regulatory_draft_rule (
				id, case_id, rule_id, proposed_ast, proposed_parameter_thresholds,
				proposed_citation, proposed_content_hash, updated_at
			) VALUES (
				gen_random_uuid(), $1, $2, $3::jsonb, $4::jsonb,
				$5, $6, now()
			)
			ON CONFLICT (case_id, rule_id) DO UPDATE SET
				proposed_ast = EXCLUDED.proposed_ast,
				proposed_parameter_thresholds = EXCLUDED.proposed_parameter_thresholds,
				proposed_citation = EXCLUDED.proposed_citation,
				proposed_content_hash = EXCLUDED.proposed_content_hash,
				updated_at = now()
		`, caseID, d.RuleID, string(astBytes), string(paramBytes), d.NewCitation, contentHash)
		if err != nil {
			return fmt.Errorf("upsert draft rule %s: %w", d.RuleID, err)
		}
	}

	return tx.Commit()
}

// ExecuteCorpusGate runs scenario tests, asserts active status (no PROVISIONAL on semantic changes), and records results
func (s *Service) ExecuteCorpusGate(ctx context.Context, caseID uuid.UUID, stewardID string, drafts []RuleDraft) (*drift.CorpusRunResult, error) {
	repinSvc := drift.NewRepinService(s.db)
	totalResult := &drift.CorpusRunResult{
		AllPassed: true,
	}

	// Persist drafts into durable storage first
	if err := s.SaveDrafts(ctx, caseID, drafts); err != nil {
		return nil, fmt.Errorf("save drafts: %w", err)
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

		// Update corpus results on draft rule
		corpusJSON, _ := json.Marshal(res)
		_, _ = s.db.ExecContext(ctx, `
			UPDATE compliance.regulatory_draft_rule
			SET corpus_results = $1::jsonb, updated_at = now()
			WHERE case_id = $2 AND rule_id = $3
		`, string(corpusJSON), caseID, d.RuleID)
	}

	draftHashes := make(map[string]string)
	for _, d := range drafts {
		astBytes, _ := json.Marshal(d.NewAST)
		paramBytes, _ := json.Marshal(d.NewThresholds)
		h, err := canonical.ComputeRuleContentHashFromRaw(astBytes, paramBytes, d.NewCitation)
		if err == nil {
			draftHashes[d.RuleID.String()] = h
		}
	}

	payloadJSON, _ := json.Marshal(map[string]interface{}{
		"corpus_result": totalResult,
		"steward_id":    stewardID,
		"draft_hashes":  draftHashes,
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

// ApproveCase transitions UNDER_REVIEW -> APPROVED_FOR_PUBLISH and cryptographically binds approved draft hashes
func (s *Service) ApproveCase(ctx context.Context, caseID uuid.UUID, stewardID string, notes string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Verify drafts exist in compliance.regulatory_draft_rule
	rows, err := tx.QueryContext(ctx, `
		SELECT rule_id, proposed_content_hash
		FROM compliance.regulatory_draft_rule
		WHERE case_id = $1
	`, caseID)
	if err != nil {
		return fmt.Errorf("query draft rules: %w", err)
	}
	defer rows.Close()

	approvedHashes := make(map[string]string)
	for rows.Next() {
		var rID uuid.UUID
		var pHash string
		if err := rows.Scan(&rID, &pHash); err != nil {
			return fmt.Errorf("scan draft hash: %w", err)
		}
		approvedHashes[rID.String()] = pHash
	}
	rows.Close()

	if len(approvedHashes) == 0 {
		return errors.New("approval blocked: no draft rules found in regulatory_draft_rule for this case")
	}

	// Verify that corpus run occurred and draft hashes match
	var corpusEventPayloadJSON []byte
	err = tx.QueryRowContext(ctx, `
		SELECT payload
		FROM compliance.regulatory_case_event
		WHERE case_id = $1 AND event_type = 'CORPUS_RUN'
		ORDER BY created_at DESC
		LIMIT 1
	`, caseID).Scan(&corpusEventPayloadJSON)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("approval blocked: case requires test corpus execution before approval")
		}
		return fmt.Errorf("check corpus run event: %w", err)
	}

	var corpusEventData struct {
		DraftHashes map[string]string `json:"draft_hashes"`
		CorpusResult struct {
			AllPassed bool `json:"allPassed"`
		} `json:"corpus_result"`
	}
	_ = json.Unmarshal(corpusEventPayloadJSON, &corpusEventData)

	if !corpusEventData.CorpusResult.AllPassed {
		return errors.New("approval blocked: latest test corpus execution did not pass")
	}

	for rID, pHash := range approvedHashes {
		corpusHash, ok := corpusEventData.DraftHashes[rID]
		if !ok || corpusHash != pHash {
			return fmt.Errorf("approval blocked: draft content for rule %s has changed since last corpus execution (corpus_hash=%s, current_draft_hash=%s); re-run corpus gate before approving", rID, corpusHash, pHash)
		}
	}

	// Mark drafts as approved
	_, err = tx.ExecContext(ctx, `
		UPDATE compliance.regulatory_draft_rule
		SET is_approved = true, approved_by = $1, approved_at = now(), updated_at = now()
		WHERE case_id = $2
	`, stewardID, caseID)
	if err != nil {
		return fmt.Errorf("mark drafts approved: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE compliance.regulatory_change_case
		SET status = 'APPROVED_FOR_PUBLISH', updated_at = now()
		WHERE id = $1
	`, caseID)
	if err != nil {
		return fmt.Errorf("update case to APPROVED_FOR_PUBLISH: %w", err)
	}

	payloadJSON, _ := json.Marshal(map[string]interface{}{
		"steward_id":             stewardID,
		"notes":                  notes,
		"approved_draft_hashes": approvedHashes,
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

// PublishRelease performs atomic Core release with cryptographic draft binding assertion
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

		// JCS canonical content hash computation of what is about to be published
		astBytes, _ := json.Marshal(d.NewAST)
		paramBytes, _ := json.Marshal(d.NewThresholds)
		publishedContentHash, err := canonical.ComputeRuleContentHashFromRaw(astBytes, paramBytes, d.NewCitation)
		if err != nil {
			return nil, fmt.Errorf("compute content hash for rule %s: %w", ruleCode, err)
		}
		bytecodeHash := canonical.ComputeBytecodeHash(nil)

		// Cryptographic Binding Assertion: draft must exist in regulatory_draft_rule, be approved, and match hash
		var draftApprovedHash string
		var isApproved bool
		err = tx.QueryRowContext(ctx, `
			SELECT proposed_content_hash, is_approved
			FROM compliance.regulatory_draft_rule
			WHERE case_id = $1 AND rule_id = $2
		`, req.CaseID, d.RuleID).Scan(&draftApprovedHash, &isApproved)
		if err != nil {
			return nil, fmt.Errorf("publish release blocked: no draft found in regulatory_draft_rule for case %s rule %s: %w", req.CaseID, ruleCode, err)
		}

		if !isApproved {
			return nil, fmt.Errorf("publish release blocked: draft for rule %s in case %s has not been approved", ruleCode, req.CaseID)
		}

		if draftApprovedHash != publishedContentHash {
			return nil, fmt.Errorf("publish release blocked: published content hash (%s) does not match approved draft hash (%s) for rule %s", publishedContentHash, draftApprovedHash, ruleCode)
		}

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
		`, d.RuleID, newVer, tenantID, string(astBytes), string(paramBytes), d.NewCitation, d.EffectiveFrom, publishedContentHash, bytecodeHash, req.StewardID)
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
			RuleID:               d.RuleID,
			RuleCode:             ruleCode,
			FromVersion:          currentVer,
			ToVersion:            newVer,
			ApprovedContentHash:  draftApprovedHash,
			PublishedContentHash: publishedContentHash,
		})

		// 5. Fan-out notifications to tenants
		notifPayloadInherit, _ := json.Marshal(map[string]interface{}{
			"case_code":              caseCode,
			"rule_code":              ruleCode,
			"from_version":           currentVer,
			"to_version":             newVer,
			"effective_from":         d.EffectiveFrom,
			"citation":               d.NewCitation,
			"approved_content_hash":  draftApprovedHash,
			"published_content_hash": publishedContentHash,
		})
		_, err = tx.ExecContext(ctx, `
			INSERT INTO compliance.compliance_notification (
				id, tenant_id, kind, title, payload, is_read, created_at
			)
			SELECT 
				gen_random_uuid(),
				id,
				'INHERIT_ADVANCE',
				$1,
				$2::jsonb,
				false,
				now()
			FROM public.tenants
			WHERE is_active = true AND gold_copy = false
		`, fmt.Sprintf("Core Rule %s Advanced to Version %d", ruleCode, newVer), string(notifPayloadInherit))
		if err != nil {
			return nil, fmt.Errorf("fan-out inherit notifications: %w", err)
		}

		notifPayloadDrift, _ := json.Marshal(map[string]interface{}{
			"case_code":              caseCode,
			"rule_code":              ruleCode,
			"new_core_version":       newVer,
			"action_required":        "Review and repin or merge custom overrides",
			"effective_from":         d.EffectiveFrom,
			"approved_content_hash":  draftApprovedHash,
			"published_content_hash": publishedContentHash,
		})
		_, err = tx.ExecContext(ctx, `
			INSERT INTO compliance.compliance_notification (
				id, tenant_id, kind, title, payload, is_read, created_at
			)
			SELECT 
				gen_random_uuid(),
				r.tenant_id,
				'DRIFT_FLAG',
				$1,
				$2::jsonb,
				false,
				now()
			FROM compliance.compliance_rule r
			WHERE r.core_rule_id = $3 AND r.inherit_mode = 'extend' AND r.valid_to IS NULL
		`, fmt.Sprintf("Core Version Updated on Extended Rule %s: Action Required", ruleCode), string(notifPayloadDrift), d.RuleID)
		if err != nil {
			return nil, fmt.Errorf("fan-out drift notifications: %w", err)
		}

		// 6. Record in compliance.governance_audit_event
		_, err = tx.ExecContext(ctx, `
			INSERT INTO compliance.governance_audit_event (
				id, tenant_id, rule_id, event_type, old_pinned_version,
				new_pinned_version, steward_id, steward_notes, ast_diff,
				corpus_run_results, created_at
			) VALUES (
				gen_random_uuid(), $1, $2, 'REGULATORY_CHANGE_PUBLISHED', $3,
				$4, $5, $6, $7::jsonb, $8::jsonb, now()
			)
		`, tenantID, d.RuleID, currentVer, newVer, req.StewardID,
			fmt.Sprintf("Published via Case %s: approved_hash=%s, published_hash=%s", caseCode, draftApprovedHash, publishedContentHash),
			`{"status":"PUBLISHED"}`, `{"status":"PASSED"}`)
		if err != nil {
			return nil, fmt.Errorf("insert governance audit event for %s: %w", ruleCode, err)
		}
	}

	// 7. Update case status to PUBLISHED
	pubJSON, _ := json.Marshal(publishedEntries)
	_, err = tx.ExecContext(ctx, `
		UPDATE compliance.regulatory_change_case
		SET status = 'PUBLISHED', published_rule_versions = $1::jsonb, updated_at = now()
		WHERE id = $2
	`, string(pubJSON), req.CaseID)
	if err != nil {
		return nil, fmt.Errorf("update case status to PUBLISHED: %w", err)
	}

	// 8. Record PUBLISHED event
	eventPayload, _ := json.Marshal(map[string]interface{}{
		"steward_id":        req.StewardID,
		"published_entries": publishedEntries,
	})
	_, err = tx.ExecContext(ctx, `
		INSERT INTO compliance.regulatory_case_event (
			id, case_id, event_type, actor, payload, created_at
		) VALUES (
			gen_random_uuid(), $1, 'PUBLISHED', $2, $3::jsonb, now()
		)
	`, req.CaseID, req.StewardID, string(eventPayload))
	if err != nil {
		return nil, fmt.Errorf("insert published case event: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit publish release: %w", err)
	}

	return publishedEntries, nil
}

// EscalateCase marks case as escalated, increments counter, and dispatches high-priority paging alert without closing case
func (s *Service) EscalateCase(ctx context.Context, caseID uuid.UUID, reason string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var caseCode, title string
	var currentStatus string
	var dueAt time.Time
	var count int
	err = tx.QueryRowContext(ctx, `
		SELECT case_code, title, status, due_at, escalation_count
		FROM compliance.regulatory_change_case
		WHERE id = $1
	`, caseID).Scan(&caseCode, &title, &currentStatus, &dueAt, &count)
	if err != nil {
		return fmt.Errorf("query case for escalation: %w", err)
	}

	newCount := count + 1

	_, err = tx.ExecContext(ctx, `
		UPDATE compliance.regulatory_change_case
		SET 
			is_escalated = true,
			escalation_count = $1,
			escalated_at = COALESCE(escalated_at, now()),
			last_escalated_at = now(),
			updated_at = now()
		WHERE id = $2
	`, newCount, caseID)
	if err != nil {
		return fmt.Errorf("update case escalation state: %w", err)
	}

	payloadJSON, _ := json.Marshal(map[string]interface{}{
		"escalation_count": newCount,
		"reason":           reason,
		"status":           currentStatus,
		"due_at":           dueAt,
	})

	_, err = tx.ExecContext(ctx, `
		INSERT INTO compliance.regulatory_case_event (
			id, case_id, event_type, actor, payload, created_at
		) VALUES (
			gen_random_uuid(), $1, 'ESCALATED', 'system_ttl_sweeper', $2::jsonb, now()
		)
	`, caseID, string(payloadJSON))
	if err != nil {
		return fmt.Errorf("insert escalated event: %w", err)
	}

	// Insert high-priority blotter notification
	loader := compliance.NewMultiTenantRuleLoader(s.db)
	goldTenant, err := loader.GetGoldCopyTenantID(ctx)
	if err != nil {
		goldTenant = uuid.MustParse("99e99e99-99e9-49e9-89e9-99e99e99e999")
	}

	notifPayload, _ := json.Marshal(map[string]interface{}{
		"case_code":        caseCode,
		"case_id":          caseID,
		"escalation_count": newCount,
		"status":           currentStatus,
		"due_at":           dueAt,
		"reason":           reason,
	})

	_, err = tx.ExecContext(ctx, `
		INSERT INTO compliance.compliance_notification (
			id, tenant_id, kind, title, payload, is_read, created_at
		) VALUES (
			gen_random_uuid(), $1, 'CASE_ESCALATED', $2, $3::jsonb, false, now()
		)
	`, goldTenant, fmt.Sprintf("URGENT SLA ESCALATION (%d): Case %s (%s) Unaddressed", newCount, caseCode, title), string(notifPayload))
	if err != nil {
		return fmt.Errorf("insert escalation notification: %w", err)
	}

	return tx.Commit()
}

// CloseCase closes a case with NO_IMPACT
func (s *Service) CloseCase(ctx context.Context, caseID uuid.UUID, actor string, reason string) error {
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
		return fmt.Errorf("insert closed event: %w", err)
	}

	return tx.Commit()
}

// RejectCase rejects a case
func (s *Service) RejectCase(ctx context.Context, caseID uuid.UUID, actor string, reason string) error {
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
		return fmt.Errorf("insert rejected event: %w", err)
	}

	return tx.Commit()
}

// RouteToNewRuleBacklog transitions case to NEW_RULE_BACKLOG
func (s *Service) RouteToNewRuleBacklog(ctx context.Context, caseID uuid.UUID, stewardID string, notes string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		UPDATE compliance.regulatory_change_case
		SET status = 'NEW_RULE_BACKLOG', updated_at = now()
		WHERE id = $1
	`, caseID)
	if err != nil {
		return fmt.Errorf("update case to NEW_RULE_BACKLOG: %w", err)
	}

	payloadJSON, _ := json.Marshal(map[string]interface{}{
		"steward_id": stewardID,
		"notes":      notes,
	})

	_, err = tx.ExecContext(ctx, `
		INSERT INTO compliance.regulatory_case_event (
			id, case_id, event_type, actor, payload, created_at
		) VALUES (
			gen_random_uuid(), $1, 'NEW_RULE_ROUTED', $2, $3::jsonb, now()
		)
	`, caseID, stewardID, string(payloadJSON))
	if err != nil {
		return fmt.Errorf("insert new rule routed event: %w", err)
	}

	return tx.Commit()
}

// GetUnaddressedCases queries operational SLA metrics from compliance.v_unaddressed_regulatory_cases
func (s *Service) GetUnaddressedCases(ctx context.Context) ([]UnaddressedRegulatoryCase, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT 
			case_id, case_code, title, source, coalesce(source_reference, ''),
			status, coalesce(classification, ''), is_escalated, escalation_count,
			due_at, created_at, triaged_at, published_rule_versions,
			overdue_seconds, is_overdue, intake_to_triage_seconds
		FROM compliance.v_unaddressed_regulatory_cases
		ORDER BY overdue_seconds DESC, created_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query unaddressed cases view: %w", err)
	}
	defer rows.Close()

	var cases []UnaddressedRegulatoryCase
	for rows.Next() {
		var c UnaddressedRegulatoryCase
		var pubVerJSON []byte
		err := rows.Scan(
			&c.CaseID, &c.CaseCode, &c.Title, &c.Source, &c.SourceReference,
			&c.Status, &c.Classification, &c.IsEscalated, &c.EscalationCount,
			&c.DueAt, &c.CreatedAt, &c.TriagedAt, &pubVerJSON,
			&c.OverdueSeconds, &c.IsOverdue, &c.IntakeToTriageSeconds,
		)
		if err != nil {
			return nil, fmt.Errorf("scan unaddressed case: %w", err)
		}
		if len(pubVerJSON) > 0 {
			_ = json.Unmarshal(pubVerJSON, &c.PublishedRuleVersions)
		}
		cases = append(cases, c)
	}
	return cases, nil
}

// GetStewardTriageView builds structured presentation model for Page Designer components
func (s *Service) GetStewardTriageView(ctx context.Context, caseID uuid.UUID) (*StewardTriageView, error) {
	c, err := s.GetCase(ctx, caseID)
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
		Classification:  c.Classification,
		IsEscalated:     c.IsEscalated,
		EscalationCount: c.EscalationCount,
		AffectedRules:   make([]AffectedRuleSummary, 0, len(c.AffectedRuleIDs)),
		DiffViews:       make([]RuleDiffView, 0),
		AvailableSignals: []string{
			"StartReview()",
			"CloseNoImpact(reason)",
			"Reject(reason)",
		},
	}

	if c.Status == StatusApprovedForPublish {
		view.AvailableSignals = []string{"Publish(steward_id, drafts)"}
	} else if c.Status == StatusUnderReview {
		view.AvailableSignals = []string{"Approve(steward_id, notes)", "Reject(reason)"}
	}

	// Fetch drafts from compliance.regulatory_draft_rule if available
	draftsByRuleID := make(map[uuid.UUID]RegulatoryDraftRule)
	dRows, err := s.db.QueryContext(ctx, `
		SELECT rule_id, proposed_ast, proposed_parameter_thresholds, proposed_citation, proposed_content_hash, corpus_results, is_approved
		FROM compliance.regulatory_draft_rule
		WHERE case_id = $1
	`, caseID)
	if err == nil {
		defer dRows.Close()
		for dRows.Next() {
			var dr RegulatoryDraftRule
			var astBytes, paramBytes, corpusBytes []byte
			if err := dRows.Scan(&dr.RuleID, &astBytes, &paramBytes, &dr.ProposedCitation, &dr.ProposedContentHash, &corpusBytes, &dr.IsApproved); err == nil {
				_ = json.Unmarshal(astBytes, &dr.ProposedAST)
				_ = json.Unmarshal(paramBytes, &dr.ProposedParameterThresholds)
				if len(corpusBytes) > 0 {
					_ = json.Unmarshal(corpusBytes, &dr.CorpusResults)
				}
				draftsByRuleID[dr.RuleID] = dr
			}
		}
	}

	for _, ruleID := range c.AffectedRuleIDs {
		var rCode, rName, libStatus, citation string
		var curVer int
		var astBytes, paramBytes []byte
		err := s.db.QueryRowContext(ctx, `
			SELECT rule_code, name, coalesce(library_status, 'ACTIVE'), current_version, ast_condition, parameter_thresholds, coalesce(citation, '')
			FROM compliance.compliance_rule
			WHERE id = $1
		`, ruleID).Scan(&rCode, &rName, &libStatus, &curVer, &astBytes, &paramBytes, &citation)
		if err == nil {
			var curAST, curParams map[string]interface{}
			_ = json.Unmarshal(astBytes, &curAST)
			_ = json.Unmarshal(paramBytes, &curParams)

			affSum := AffectedRuleSummary{
				RuleID:         ruleID,
				RuleCode:       rCode,
				Name:           rName,
				CurrentVersion: curVer,
				LibraryStatus:  libStatus,
				Citation:       citation,
				Thresholds:     curParams,
			}

			if draft, hasDraft := draftsByRuleID[ruleID]; hasDraft {
				affSum.ProposedThresholds = draft.ProposedParameterThresholds
				affSum.ProposedCitation = draft.ProposedCitation

				diff := drift.CompareAST(curAST, draft.ProposedAST)
				corpusPassed := draft.CorpusResults != nil && draft.CorpusResults.AllPassed

				// Enforce threshold-aware diff summary: when proposed_thresholds != current_thresholds,
				// format explicit summary text even if the AST is invariant.
				thresholdChanges := make([]string, 0)
				for k, v1 := range curParams {
					v2, ok := draft.ProposedParameterThresholds[k]
					if !ok {
						thresholdChanges = append(thresholdChanges, fmt.Sprintf("Removed param '%s'", k))
					} else if fmt.Sprintf("%v", v1) != fmt.Sprintf("%v", v2) {
						thresholdChanges = append(thresholdChanges, fmt.Sprintf("Param '%s': '%v' -> '%v'", k, v1, v2))
					}
				}
				for k, v2 := range draft.ProposedParameterThresholds {
					if _, ok := curParams[k]; !ok {
						thresholdChanges = append(thresholdChanges, fmt.Sprintf("Added param '%s': '%v'", k, v2))
					}
				}

				if len(thresholdChanges) > 0 {
					sort.Strings(thresholdChanges)
					thresholdSummary := "Threshold modifications: " + strings.Join(thresholdChanges, ", ")
					if diff == nil {
						diff = &drift.ASTSemanticDiff{
							SummaryText: thresholdSummary,
						}
					} else if diff.SummaryText == "" || strings.HasPrefix(diff.SummaryText, "No semantic changes") {
						diff.SummaryText = thresholdSummary
					} else {
						diff.SummaryText = diff.SummaryText + " | " + thresholdSummary
					}
				}

				view.DiffViews = append(view.DiffViews, RuleDiffView{
					RuleID:              ruleID,
					RuleCode:            rCode,
					RuleName:            rName,
					Diff:                diff,
					CurrentAST:          curAST,
					ProposedAST:         draft.ProposedAST,
					CurrentThresholds:   curParams,
					ProposedThresholds:  draft.ProposedParameterThresholds,
					CurrentCitation:     citation,
					ProposedCitation:    draft.ProposedCitation,
					ProposedContentHash: draft.ProposedContentHash,
					CorpusPassed:        corpusPassed,
					HasBreakingChanges:  diff != nil && diff.HasBreakingChanges,
				})
			}

			view.AffectedRules = append(view.AffectedRules, affSum)
		}
	}

	return view, nil
}

// GetStewardReviewView retrieves the complete steward diff presentation model for review and triage screens
func (s *Service) GetStewardReviewView(ctx context.Context, caseID uuid.UUID) (*StewardTriageView, error) {
	return s.GetStewardTriageView(ctx, caseID)
}

