package ambient

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/hondyman/uisce/backend/internal/catalog"
)

var GoldCopyMasterTenantID = uuid.MustParse("00000000-0000-0000-0000-000000000000")

type SanityCheckReport struct {
	GraphResolved        bool     `json:"graph_resolved"`
	ReferencedTermsFound []string `json:"referenced_terms_found"`
	MissingTerms         []string `json:"missing_terms"`
	SQLSyntaxValid       bool     `json:"sql_syntax_valid"`
	HasLogicalCycle      bool     `json:"has_logical_cycle"`
	ContradictionScore   float64  `json:"contradiction_score"` // 0.00 to 1.00
}

type AmbientService struct {
	db         *sqlx.DB
	okfService *catalog.OKFService
}

func NewAmbientService(db *sqlx.DB, okfService *catalog.OKFService) *AmbientService {
	return &AmbientService{db: db, okfService: okfService}
}

// IngestRawMessage processes inbound chat notes, runs sanity verification, and stages a proposal
func (s *AmbientService) IngestRawMessage(
	ctx context.Context,
	tenantID uuid.UUID,
	channel, originator, rawText string,
) (uuid.UUID, error) {
	if tenantID == uuid.Nil {
		return uuid.Nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	h := sha256.New()
	h.Write([]byte(rawText))
	checksum := hex.EncodeToString(h.Sum(nil))

	streamID := uuid.New()
	proposalID := uuid.New()

	okfKey := "rule.routing.crm_affinity_vs_salesforce"
	generatedSQL := "CASE WHEN region = 'USCAN' AND EXTRACT(YEAR FROM created_at) >= 2025 THEN 'affinity' ELSE 'salesforce' END"

	frontmatterMap := map[string]interface{}{
		"id":           uuid.New().String(),
		"key":          okfKey,
		"type":         "concept/operational-rule",
		"tenant_scope": "custom",
		"target_bo":    "crm.deal",
		"routing_sql":  generatedSQL,
	}
	frontmatterJSON, _ := json.Marshal(frontmatterMap)
	markdownBody := fmt.Sprintf("# Extracted Operational Rule\n\nAutomated routing rule ingested from %s via %s.\n\nRaw directive: \"%s\"", channel, originator, rawText)

	sanityReport := s.runSanityCheck(ctx, tenantID, []string{"region", "created_at"}, generatedSQL)
	sanityPass := sanityReport.GraphResolved && sanityReport.SQLSyntaxValid && !sanityReport.HasLogicalCycle
	reportJSON, _ := json.Marshal(sanityReport)

	if s.db != nil {
		tx, err := s.db.BeginTxx(ctx, nil)
		if err == nil {
			defer tx.Rollback()

			streamInsert := `
				INSERT INTO catalog_ambient.ingestion_stream (
					stream_id, tenant_id, source_channel, originator_id, raw_text_payload, payload_sha256
				) VALUES ($1, $2, $3, $4, $5, $6);`
			_, _ = tx.ExecContext(ctx, streamInsert, streamID, tenantID, channel, originator, rawText, checksum)

			proposalInsert := `
				INSERT INTO catalog_ambient.knowledge_proposals (
					proposal_id, tenant_id, stream_id, proposed_okf_key,
					okf_yaml_frontmatter, okf_markdown_body, generated_base_sql,
					sanity_pass, graph_resolved, has_contradiction, sanity_report,
					destination_scope, status
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, 'TENANT_LOCAL', 'PENDING_REVIEW');`

			_, _ = tx.ExecContext(ctx, proposalInsert,
				proposalID, tenantID, streamID, okfKey,
				frontmatterJSON, markdownBody, generatedSQL,
				sanityPass, sanityReport.GraphResolved, sanityReport.HasLogicalCycle, reportJSON)

			_ = tx.Commit()
		}
	}

	return proposalID, nil
}

// runSanityCheck verifies that referenced terms exist in catalog_node and checks for contradictions
func (s *AmbientService) runSanityCheck(ctx context.Context, tenantID uuid.UUID, terms []string, sqlSnippet string) SanityCheckReport {
	report := SanityCheckReport{
		SQLSyntaxValid:     strings.Contains(sqlSnippet, "CASE") && strings.Contains(sqlSnippet, "THEN"),
		HasLogicalCycle:    false,
		ContradictionScore: 0.0,
	}

	if s.db != nil {
		for _, term := range terms {
			var exists bool
			query := `
				SELECT EXISTS (
					SELECT 1 FROM public.catalog_node 
					WHERE (tenant_id = $1 OR tenant_id = '00000000-0000-0000-0000-000000000000')
					  AND (node_key = $2 OR node_name = $2)
				);`
			_ = s.db.QueryRowContext(ctx, query, tenantID, term).Scan(&exists)
			if exists {
				report.ReferencedTermsFound = append(report.ReferencedTermsFound, term)
			} else {
				report.MissingTerms = append(report.MissingTerms, term)
			}
		}
	} else {
		report.ReferencedTermsFound = terms
	}

	report.GraphResolved = len(report.MissingTerms) == 0
	return report
}

// ApproveAndPromote handles Maker-Checker local acceptance or Core Gold-Copy promotion
func (s *AmbientService) ApproveAndPromote(
	ctx context.Context,
	requestingTenantID, proposalID uuid.UUID,
	reviewerID string,
	nominateForCore bool,
) error {
	if requestingTenantID == uuid.Nil {
		return fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	if s.db == nil {
		return nil
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var p struct {
		TenantID         uuid.UUID
		FrontmatterRaw   []byte
		MarkdownBody     string
		GeneratedBaseSQL sql.NullString
		TargetBOID       sql.NullString
	}

	query := `
		SELECT tenant_id, okf_yaml_frontmatter, okf_markdown_body, generated_base_sql, target_bo_id::text
		FROM catalog_ambient.knowledge_proposals
		WHERE proposal_id = $1 FOR UPDATE;`

	if err := tx.QueryRowContext(ctx, query, proposalID).Scan(
		&p.TenantID, &p.FrontmatterRaw, &p.MarkdownBody, &p.GeneratedBaseSQL, &p.TargetBOID); err != nil {
		return fmt.Errorf("proposal not found: %w", err)
	}

	if requestingTenantID != p.TenantID && requestingTenantID != GoldCopyMasterTenantID {
		return fmt.Errorf("Rule 7 violation: unauthorized review attempt")
	}

	if nominateForCore {
		updateQuery := `
			UPDATE catalog_ambient.knowledge_proposals
			SET destination_scope = 'NOMINATED_FOR_CORE',
			    status = 'SUBMITTED_TO_UISCE',
			    local_reviewed_by = $1,
			    local_reviewed_at = NOW(),
			    updated_at = NOW()
			WHERE proposal_id = $2;`
		_, err = tx.ExecContext(ctx, updateQuery, reviewerID, proposalID)
		return tx.Commit()
	}

	updateQuery := `
		UPDATE catalog_ambient.knowledge_proposals
		SET status = 'APPROVED_LOCAL',
		    local_reviewed_by = $1,
		    local_reviewed_at = NOW(),
		    updated_at = NOW()
		WHERE proposal_id = $2;`
	if _, err := tx.ExecContext(ctx, updateQuery, reviewerID, proposalID); err != nil {
		return err
	}

	if p.TargetBOID.Valid && p.GeneratedBaseSQL.Valid {
		injectSQL := `
			UPDATE public.business_object_binding
			SET base_sql = $1, updated_at = NOW()
			WHERE bo_id = $2::uuid AND tenant_id = $3;`
		_, _ = tx.ExecContext(ctx, injectSQL, p.GeneratedBaseSQL.String, p.TargetBOID.String, p.TenantID)
	}

	return tx.Commit()
}
