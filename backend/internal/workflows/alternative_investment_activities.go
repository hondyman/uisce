package workflows

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hondyman/semlayer/backend/internal/services"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/types"
)

// AlternativeInvestmentActivities contains all Temporal activities for alternative investments
type AlternativeInvestmentActivities struct {
	db            *sql.DB
	altInvService *services.AlternativeInvestmentService
	perfService   *services.PerformanceService
	docService    *services.DocumentProcessingService
}

// NewAlternativeInvestmentActivities creates new activities
func NewAlternativeInvestmentActivities(
	db *sql.DB,
	altInvService *services.AlternativeInvestmentService,
	perfService *services.PerformanceService,
	docService *services.DocumentProcessingService,
) *AlternativeInvestmentActivities {
	return &AlternativeInvestmentActivities{
		db:            db,
		altInvService: altInvService,
		perfService:   perfService,
		docService:    docService,
	}
}

// ExtractDocumentData extracts structured data from a document using AI
func (a *AlternativeInvestmentActivities) ExtractDocumentData(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	documentType := input["documentType"].(string)

	// TODO: Integrate with Gemini API
	// For now, return mock data based on document type

	extractedData := make(map[string]interface{})

	switch documentType {
	case "K1":
		extractedData = map[string]interface{}{
			"tax_year":               2024,
			"fund_name":              "Mock Fund",
			"ordinary_income":        -15000.00,
			"long_term_capital_gain": 50000.00,
			"confidence_score":       0.85,
		}
	case "CAPITAL_CALL":
		extractedData = map[string]interface{}{
			"call_number":      5,
			"call_date":        "2024-11-27",
			"due_date":         "2024-12-27",
			"amount_requested": 250000.00,
			"confidence_score": 0.90,
		}
	default:
		return nil, fmt.Errorf("unsupported document type: %s", documentType)
	}

	return extractedData, nil
}

// StoreExtractedData stores extracted data in the database
func (a *AlternativeInvestmentActivities) StoreExtractedData(ctx context.Context, documentID string, extractedData map[string]interface{}) error {
	docUUID, err := uuid.Parse(documentID)
	if err != nil {
		return err
	}

	extractedJSON, err := json.Marshal(extractedData)
	if err != nil {
		return err
	}

	// TODO: Replace SQL with Hasura GraphQL mutation:
	// mutation UpdateDocument($id: uuid!, $data: jsonb!) {
	//   update_alternative_investment_documents_by_pk(
	//     pk_columns: {id: $id},
	//     _set: {extracted_data: $data, processing_status: "COMPLETED", processed_at: "now()"}
	//   ) {
	//     id
	//     processing_status
	//   }
	// }
	// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
	query := `
		UPDATE alternative_investment_documents
		SET 
			extracted_data = $1::jsonb,
			processing_status = 'COMPLETED',
			processed_at = NOW()
		WHERE id = $2
	`

	_, err = a.db.ExecContext(ctx, query, string(extractedJSON), docUUID)
	return err
}

// CheckIfReviewRequired determines if human review is needed based on confidence scores
func (a *AlternativeInvestmentActivities) CheckIfReviewRequired(ctx context.Context, documentID string, extractedData map[string]interface{}) (bool, error) {
	// Check confidence score
	confidenceScore, ok := extractedData["confidence_score"].(float64)
	if !ok {
		return true, nil // Require review if no confidence score
	}

	// Require review if confidence < 90%
	if confidenceScore < 0.90 {
		// Update document to mark as needs review
		// TODO: Replace SQL with Hasura GraphQL mutation:
		// mutation MarkForReview($id: uuid!) {
		//   update_alternative_investment_documents_by_pk(
		//     pk_columns: {id: $id},
		//     _set: {requires_review: true}
		//   ) {
		//     id
		//     requires_review
		//   }
		// }
		// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
		docUUID, _ := uuid.Parse(documentID)
		_, err := a.db.ExecContext(ctx, `
			UPDATE alternative_investment_documents
			SET requires_review = TRUE
			WHERE id = $1
		`, docUUID)
		return true, err
	}

	return false, nil
}

// MarkDocumentFailed marks a document as failed
func (a *AlternativeInvestmentActivities) MarkDocumentFailed(ctx context.Context, documentID string, errorMsg string) error {
	docUUID, err := uuid.Parse(documentID)
	if err != nil {
		return err
	}

	// TODO: Replace SQL with Hasura GraphQL mutation:
	// mutation MarkDocumentFailed($id: uuid!, $error: String!) {
	//   update_alternative_investment_documents_by_pk(
	//     pk_columns: {id: $id},
	//     _set: {processing_status: "FAILED", processing_error: $error}
	//   ) {
	//     id
	//     processing_status
	//   }
	// }
	// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
	query := `
		UPDATE alternative_investment_documents
		SET 
			processing_status = 'FAILED',
			processing_error = $1
		WHERE id = $2
	`

	_, err = a.db.ExecContext(ctx, query, errorMsg, docUUID)
	return err
}

// ApplyExtractedData applies extracted data to investment records
func (a *AlternativeInvestmentActivities) ApplyExtractedData(ctx context.Context, documentID string, documentType string) error {
	// Get the document and extracted data
	doc, err := a.docService.GetDocument(ctx, uuid.MustParse(documentID))
	if err != nil {
		return err
	}

	switch documentType {
	case "K1":
		return a.applyK1Data(ctx, doc)
	case "CAPITAL_CALL":
		return a.applyCapitalCallData(ctx, doc)
	case "QUARTERLY_STATEMENT":
		return a.applyQuarterlyStatementData(ctx, doc)
	default:
		return fmt.Errorf("unsupported document type for applying data: %s", documentType)
	}
}

func (a *AlternativeInvestmentActivities) applyK1Data(ctx context.Context, doc *types.AlternativeInvestmentDocument) error {
	// Update investment with K-1 received flag
	// TODO: Replace SQL with Hasura GraphQL mutation:
	// mutation UpdateK1Received($id: uuid!) {
	//   update_alternative_investments_by_pk(
	//     pk_columns: {id: $id},
	//     _set: {k1_received: true, k1_received_date: "now()"}
	//   ) {
	//     id
	//     k1_received
	//   }
	// }
	// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
	query := `
		UPDATE alternative_investments
		SET 
			k1_received = TRUE,
			k1_received_date = NOW()
		WHERE id = $1
	`

	_, err := a.db.ExecContext(ctx, query, doc.InvestmentID)
	return err
}

func (a *AlternativeInvestmentActivities) applyCapitalCallData(ctx context.Context, doc *types.AlternativeInvestmentDocument) error {
	// Extract capital call details from document
	callNumber := int(doc.ExtractedData["call_number"].(float64))
	callDate, _ := time.Parse("2006-01-02", doc.ExtractedData["call_date"].(string))
	dueDate, _ := time.Parse("2006-01-02", doc.ExtractedData["due_date"].(string))
	amount := doc.ExtractedData["amount_requested"].(float64)

	// Create capital call record
	call := &types.CapitalCall{
		InvestmentID:     doc.InvestmentID,
		CallNumber:       callNumber,
		CallDate:         callDate,
		DueDate:          dueDate,
		AmountRequested:  amount,
		NoticeDocumentID: &doc.ID,
	}

	return a.altInvService.RecordCapitalCall(ctx, call)
}

func (a *AlternativeInvestmentActivities) applyQuarterlyStatementData(ctx context.Context, doc *types.AlternativeInvestmentDocument) error {
	// Update NAV from quarterly statement
	nav := doc.ExtractedData["nav"].(float64)
	valuationDate, _ := time.Parse("2006-01-02", doc.ExtractedData["valuation_date"].(string))

	// TODO: Replace SQL with Hasura GraphQL mutation:
	// mutation UpdateInvestmentNav($id: uuid!, $nav: numeric!, $date: timestamptz!) {
	//   update_alternative_investments_by_pk(
	//     pk_columns: {id: $id},
	//     _set: {current_nav: $nav, last_valuation_date: $date, valuation_method: "GP_ESTIMATE"}
	//   ) {
	//     id
	//     current_nav
	//   }
	// }
	// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
	query := `
		UPDATE alternative_investments
		SET 
			current_nav = $1,
			last_valuation_date = $2,
			valuation_method = 'GP_ESTIMATE'
		WHERE id = $3
	`

	_, err := a.db.ExecContext(ctx, query, nav, valuationDate, doc.InvestmentID)
	return err
}

// GetActiveInvestmentIDs retrieves all active investment IDs for a tenant
func (a *AlternativeInvestmentActivities) GetActiveInvestmentIDs(ctx context.Context, tenantID string) ([]string, error) {
	// TODO: Replace SQL with Hasura GraphQL query:
	// query GetActiveInvestments($tenantId: uuid!) {
	//   alternative_investments(where: {tenant_id: {_eq: $tenantId}, deleted_at: {_is_null: true}}) {
	//     id
	//   }
	// }
	// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
	query := `
		SELECT id::text
		FROM alternative_investments
		WHERE tenant_id = $1 AND deleted_at IS NULL
	`

	rows, err := a.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	return ids, rows.Err()
}

// CalculateInvestmentPerformance calculates performance metrics for an investment
func (a *AlternativeInvestmentActivities) CalculatePerformanceActivity(ctx context.Context, input PerformanceInput) error {
	invID, err := uuid.Parse(input.InvestmentID)
	if err != nil {
		return err
	}

	tenantID, err := uuid.Parse(input.TenantID)
	if err != nil {
		return err
	}

	asOfDate := time.Now()
	_, err = a.perfService.CalculateAndSavePerformanceMetrics(ctx, tenantID, invID, asOfDate)
	return err
}

// GetInvestmentsWithUnfundedCommitments gets investments that still have unfunded commitments
func (a *AlternativeInvestmentActivities) GetInvestmentsWithUnfundedCommitments(ctx context.Context, tenantID string) ([]string, error) {
	// TODO: Replace SQL with Hasura GraphQL query:
	// query GetUnfundedInvestments($tenantId: uuid!) {
	//   alternative_investments(where: {
	//     tenant_id: {_eq: $tenantId},
	//     deleted_at: {_is_null: true},
	//     unfunded_commitment: {_gt: 0}
	//   }) {
	//     id
	//   }
	// }
	// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
	query := `
		SELECT id::text
		FROM alternative_investments
		WHERE tenant_id = $1 
		  AND deleted_at IS NULL
		  AND unfunded_commitment > 0
	`

	rows, err := a.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	return ids, rows.Err()
}

// GenerateCapitalCallForecast generates a forecast for upcoming capital calls
func (a *AlternativeInvestmentActivities) GenerateCapitalCallForecast(ctx context.Context, investmentID string) error {
	// TODO: Implement ML-based forecasting
	// For now, create a simple heuristic-based forecast

	invID, err := uuid.Parse(investmentID)
	if err != nil {
		return err
	}

	// Simple heuristic: assume remaining capital will be called over next 2 years
	var unfundedCommitment float64
	err = a.db.QueryRowContext(ctx, `
		SELECT unfunded_commitment FROM alternative_investments WHERE id = $1
	`, invID).Scan(&unfundedCommitment)
	if err != nil {
		return err
	}

	if unfundedCommitment <= 0 {
		return nil // No forecast needed
	}

	// Create 4 quarterly forecasts
	for i := 1; i <= 4; i++ {
		forecastDate := time.Now().AddDate(0, i*3, 0)
		estimatedAmount := unfundedCommitment / 4.0

		// TODO: Replace SQL with Hasura GraphQL mutation:
		// mutation InsertForecast($object: capital_call_forecasts_insert_input!) {
		//   insert_capital_call_forecasts_one(
		//     object: $object,
		//     on_conflict: {constraint: capital_call_forecasts_pkey, update_columns: []}
		//   ) {
		//     id
		//   }
		// }
		// Variables: {"object": {"investment_id": "...", "forecasted_call_date": "...",
		//   "estimated_amount": 25000.0, "confidence_score": 0.65, "model_type": "HEURISTIC"}}
		// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
		query := `
			INSERT INTO capital_call_forecasts (
				investment_id, forecasted_call_date, estimated_amount,
				confidence_score, model_type
			) VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT DO NOTHING
		`

		_, err = a.db.ExecContext(ctx, query, invID, forecastDate, estimatedAmount, 0.65, "HEURISTIC")
		if err != nil {
			return err
		}
	}

	return nil
}

// CheckUpcomingCapitalCallsAndAlert checks for upcoming capital calls and sends alerts
func (a *AlternativeInvestmentActivities) CheckUpcomingCapitalCallsAndAlert(ctx context.Context, tenantID string) error {
	// Get capital calls due in next 30 days that haven't been alerted
	// TODO: Replace SQL with Hasura GraphQL query:
	// query GetUpcomingCapitalCalls($tenantId: uuid!, $dueDate: timestamptz!) {
	//   capital_calls(where: {
	//     investment: {tenant_id: {_eq: $tenantId}},
	//     status: {_eq: "PENDING"},
	//     due_date: {_lte: $dueDate},
	//     alert_sent: {_eq: false}
	//   }) {
	//     id
	//     investment_id
	//     amount_requested
	//     due_date
	//     liquidity_check_status
	//   }
	// }
	// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
	query := `
		SELECT id, investment_id, amount_requested, due_date, liquidity_check_status
		FROM capital_calls
		WHERE investment_id IN (
			SELECT id FROM alternative_investments WHERE tenant_id = $1
		)
		AND status = 'PENDING'
		AND due_date <= NOW() + INTERVAL '30 days'
		AND alert_sent = FALSE
	`

	rows, err := a.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var callID uuid.UUID
		var investmentID uuid.UUID
		var amount float64
		var dueDate time.Time
		var liquidityStatus sql.NullString

		if err := rows.Scan(&callID, &investmentID, &amount, &dueDate, &liquidityStatus); err != nil {
			continue
		}

		// TODO: Send alert via notification system
		// For now, just mark as alerted
		// TODO: Replace SQL with Hasura GraphQL mutation:
		// mutation MarkAlertSent($id: uuid!) {
		//   update_capital_calls_by_pk(
		//     pk_columns: {id: $id},
		//     _set: {alert_sent: true, alert_sent_at: "now()"}
		//   ) {
		//     id
		//     alert_sent
		//   }
		// }
		// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
		_, err = a.db.ExecContext(ctx, `
			UPDATE capital_calls
			SET alert_sent = TRUE, alert_sent_at = NOW()
			WHERE id = $1
		`, callID)
		if err != nil {
			continue
		}
	}

	return rows.Err()
}
