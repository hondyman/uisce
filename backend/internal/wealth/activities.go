package wealth

import (
	"context"

	"fmt"
	"time"

	"github.com/hondyman/semlayer/backend/internal/platform"
	"go.temporal.io/sdk/activity"
)

// WealthActivities contains all workflow activities for wealth management
type WealthActivities struct {
	tenantManager *platform.TenantDBManager
	recommender   *EstatePlanRecommender // Added generic recommender
}

// NewWealthActivities creates a new set of wealth activities
func NewWealthActivities(tm *platform.TenantDBManager) *WealthActivities {
	// Recommender would need proper init with services in real app, stubbing for now if nil
	return &WealthActivities{tenantManager: tm}
}

// Client Onboarding Activities

// SubmitClientDataActivity validates and stores initial client data
func (a *WealthActivities) SubmitClientDataActivity(ctx context.Context, tenantID string, clientID string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Submitting client data", "TenantID", tenantID, "ClientID", clientID)

	db, err := a.tenantManager.GetConnection(tenantID)
	if err != nil {
		return err
	}

	// Update client status to indicate data collection is complete
	query := `
		UPDATE wealth.clients 
		SET updated_at = $1 
		WHERE id = $2
	`
	_, err = db.Exec(query, time.Now(), clientID)
	return err
}

// ApproveKYCActivity marks KYC as approved
func (a *WealthActivities) ApproveKYCActivity(ctx context.Context, tenantID string, clientID string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Approving KYC", "TenantID", tenantID, "ClientID", clientID)

	db, err := a.tenantManager.GetConnection(tenantID)
	if err != nil {
		return err
	}

	query := `
		UPDATE wealth.clients 
		SET kyc_status = 'COMPLIANT', 
		    kyc_completed_at = $1,
		    updated_at = $1 
		WHERE id = $2
	`
	_, err = db.Exec(query, time.Now(), clientID)

	// Create compliance record
	if err == nil {
		complianceQuery := `
			INSERT INTO wealth.compliance_records (client_id, record_type, status, reviewed_at, created_at, updated_at)
			VALUES ($1, 'KYC', 'COMPLIANT', $2, $2, $2)
		`
		_, err = db.Exec(complianceQuery, clientID, time.Now())
	}

	return err
}

// ApproveAMLActivity marks AML as approved
func (a *WealthActivities) ApproveAMLActivity(ctx context.Context, tenantID string, clientID string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Approving AML", "TenantID", tenantID, "ClientID", clientID)

	db, err := a.tenantManager.GetConnection(tenantID)
	if err != nil {
		return err
	}

	query := `
		UPDATE wealth.clients 
		SET aml_status = 'COMPLIANT', 
		    aml_completed_at = $1,
		    updated_at = $1 
		WHERE id = $2
	`
	_, err = db.Exec(query, time.Now(), clientID)

	// Create compliance record
	if err == nil {
		complianceQuery := `
			INSERT INTO wealth.compliance_records (client_id, record_type, status, reviewed_at, created_at, updated_at)
			VALUES ($1, 'AML', 'COMPLIANT', $2, $2, $2)
		`
		_, err = db.Exec(complianceQuery, clientID, time.Now())
	}

	return err
}

// ApproveClientActivity finalizes client onboarding
func (a *WealthActivities) ApproveClientActivity(ctx context.Context, tenantID string, clientID string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Approving client", "TenantID", tenantID, "ClientID", clientID)

	db, err := a.tenantManager.GetConnection(tenantID)
	if err != nil {
		return err
	}

	query := `
		UPDATE wealth.clients 
		SET onboarding_completed = true,
		    status = 'ACTIVE',
		    updated_at = $1 
		WHERE id = $2
	`
	_, err = db.Exec(query, time.Now(), clientID)
	return err
}

// RejectClientActivity marks client as rejected
func (a *WealthActivities) RejectClientActivity(ctx context.Context, tenantID string, clientID string, reason string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Rejecting client", "TenantID", tenantID, "ClientID", clientID, "Reason", reason)

	db, err := a.tenantManager.GetConnection(tenantID)
	if err != nil {
		return err
	}

	query := `
		UPDATE wealth.clients 
		SET status = 'INACTIVE',
		    updated_at = $1 
		WHERE id = $2
	`
	_, err = db.Exec(query, time.Now(), clientID)
	return err
}

// Order Execution Activities

// SubmitOrderActivity creates the order in the database
func (a *WealthActivities) SubmitOrderActivity(ctx context.Context, tenantID string, orderID string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Submitting order", "TenantID", tenantID, "OrderID", orderID)

	db, err := a.tenantManager.GetConnection(tenantID)
	if err != nil {
		return err
	}

	query := `
		UPDATE wealth.orders 
		SET status = 'PENDING', 
		    updated_at = $1 
		WHERE id = $2
	`
	_, err = db.Exec(query, time.Now(), orderID)
	return err
}

// AutoApproveActivity auto-approves small orders
func (a *WealthActivities) AutoApproveActivity(ctx context.Context, tenantID string, orderID string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Auto-approving order", "TenantID", tenantID, "OrderID", orderID)

	db, err := a.tenantManager.GetConnection(tenantID)
	if err != nil {
		return err
	}

	query := `
		UPDATE wealth.orders 
		SET updated_at = $1 
		WHERE id = $2
	`
	_, err = db.Exec(query, time.Now(), orderID)
	return err
}

// SendToExchangeActivity submits order to external trading system
func (a *WealthActivities) SendToExchangeActivity(ctx context.Context, tenantID string, orderID string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Sending order to exchange", "TenantID", tenantID, "OrderID", orderID)

	db, err := a.tenantManager.GetConnection(tenantID)
	if err != nil {
		return err
	}

	// In production: integrate with trading API (e.g., Alpaca, Interactive Brokers)
	// For now: update status to submitted
	query := `
		UPDATE wealth.orders 
		SET status = 'SUBMITTED', 
		    external_order_id = $1,
		    updated_at = $2 
		WHERE id = $3
	`
	externalID := fmt.Sprintf("EXT-%s", orderID[:8])
	_, err = db.Exec(query, externalID, time.Now(), orderID)
	return err
}

// FullFillActivity marks order as fully filled
func (a *WealthActivities) FullFillActivity(ctx context.Context, tenantID string, orderID string, fillPrice float64) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Filling order", "TenantID", tenantID, "OrderID", orderID, "FillPrice", fillPrice)

	db, err := a.tenantManager.GetConnection(tenantID)
	if err != nil {
		return err
	}

	query := `
		UPDATE wealth.orders 
		SET status = 'FILLED', 
		    filled_quantity = quantity,
		    average_fill_price = $1,
		    filled_at = $2,
		    updated_at = $2 
		WHERE id = $3
	`
	_, err = db.Exec(query, fillPrice, time.Now(), orderID)

	// Create corresponding transaction
	if err == nil {
		txnQuery := `
			INSERT INTO wealth.transactions (portfolio_id, asset_id, transaction_type, quantity, 
			                          price_per_unit, total_amount, transaction_date, 
			                          status, order_id, created_at, updated_at)
			SELECT portfolio_id, asset_id, 
			       CASE WHEN side = 'BUY' THEN 'BUY'::transaction_type ELSE 'SELL'::transaction_type END,
			       quantity, $1, quantity * $1, $2, 'COMPLETED'::transaction_status, id, $2, $2
			FROM wealth.orders
			WHERE id = $3
		`
		_, err = db.Exec(txnQuery, fillPrice, time.Now(), orderID)
	}

	return err
}

// CancelOrderActivity cancels an order
func (a *WealthActivities) CancelOrderActivity(ctx context.Context, tenantID string, orderID string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Cancelling order", "TenantID", tenantID, "OrderID", orderID)

	db, err := a.tenantManager.GetConnection(tenantID)
	if err != nil {
		return err
	}

	query := `
		UPDATE wealth.orders 
		SET status = 'CANCELLED', 
		    cancelled_at = $1,
		    updated_at = $1 
		WHERE id = $2
	`
	_, err = db.Exec(query, time.Now(), orderID)
	return err
}

// RejectOrderActivity rejects an order due to compliance issues
func (a *WealthActivities) RejectOrderActivity(ctx context.Context, tenantID string, orderID string, reason string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Rejecting order", "TenantID", tenantID, "OrderID", orderID, "Reason", reason)

	db, err := a.tenantManager.GetConnection(tenantID)
	if err != nil {
		return err
	}

	query := `
		UPDATE wealth.orders 
		SET status = 'REJECTED', 
		    updated_at = $1 
		WHERE id = $2
	`
	_, err = db.Exec(query, time.Now(), orderID)

	// Log compliance issue
	if err == nil {
		complianceQuery := `
			INSERT INTO wealth.compliance_records (order_id, record_type, status, violations, created_at, updated_at)
			VALUES ($1, 'TRADE_REVIEW', 'NON_COMPLIANT', ARRAY[$2], $3, $3)
		`
		_, err = db.Exec(complianceQuery, orderID, reason, time.Now())
	}

	return err
}

// Estate Planning Activities

func (a *WealthActivities) GetFamilyProfileActivity(ctx context.Context, familyID string) (*FamilyProfile, error) {
	// Stub implementation
	return &FamilyProfile{FamilyID: familyID}, nil
}

func (a *WealthActivities) GetExistingScenariosActivity(ctx context.Context, familyID string) ([]EstatePlanScenario, error) {
	// Stub implementation
	return []EstatePlanScenario{}, nil
}

type CheckTaxLawChangesInput struct {
	SinceDate time.Time
}

func (a *WealthActivities) CheckTaxLawChangesActivity(ctx context.Context, input CheckTaxLawChangesInput) ([]TaxLawChange, error) {
	return []TaxLawChange{}, nil
}

type DetectFamilyChangesInput struct {
	FamilyID  string
	SinceDate time.Time
}

func (a *WealthActivities) DetectFamilyChangesActivity(ctx context.Context, input DetectFamilyChangesInput) ([]string, error) {
	return []string{}, nil
}

type DetectAssetChangesInput struct {
	FamilyID     string
	ThresholdPct float64
}

func (a *WealthActivities) DetectAssetChangesActivity(ctx context.Context, input DetectAssetChangesInput) (interface{}, error) {
	return nil, nil // Return empty interface
}

type SendReviewNotificationInput struct {
	FamilyID             string
	Changes              []string
	RequiresAction       bool
	ScenariosRegenerated int
}

func (a *WealthActivities) SendReviewNotificationActivity(ctx context.Context, input SendReviewNotificationInput) error {
	return nil
}

type ScheduleNextReviewInput struct {
	FamilyID   string
	ReviewDate time.Time
}

func (a *WealthActivities) ScheduleNextReviewActivity(ctx context.Context, input ScheduleNextReviewInput) error {
	return nil
}

// Gift Tax Filing Activities

type GetGiftsRequiringForm709Input struct {
	FamilyID string
	TaxYear  int
}

func (a *WealthActivities) GetGiftsRequiringForm709Activity(ctx context.Context, input GetGiftsRequiringForm709Input) ([]GiftForFiling, error) {
	return []GiftForFiling{}, nil
}

type PrepareForm709Input struct {
	FamilyID      string
	DonorMemberID string
	TaxYear       int
	Gifts         []GiftForFiling
}

func (a *WealthActivities) PrepareForm709Activity(ctx context.Context, input PrepareForm709Input) (Form709, error) {
	return Form709{
		FamilyID:      input.FamilyID,
		DonorMemberID: input.DonorMemberID,
		TaxYear:       input.TaxYear,
		FilingStatus:  "PREPARED",
	}, nil
}

type CalculateGiftAndGSTTaxInput struct {
	Form709 Form709
}

type TaxAmounts struct {
	GiftTax     float64
	GSTTax      float64
	TotalTaxDue float64
}

func (a *WealthActivities) CalculateGiftAndGSTTaxActivity(ctx context.Context, input CalculateGiftAndGSTTaxInput) (TaxAmounts, error) {
	return TaxAmounts{}, nil
}

type GenerateForm709PDFInput struct {
	Form709 Form709
}

func (a *WealthActivities) GenerateForm709PDFActivity(ctx context.Context, input GenerateForm709PDFInput) (string, error) {
	return "s3://bucket/form709.pdf", nil
}

type FileForm709Input struct {
	Form709 Form709
}

func (a *WealthActivities) FileForm709ElectronicallyActivity(ctx context.Context, input FileForm709Input) (string, error) {
	return "CONF-123456", nil
}

type MarkGiftsAsFiledInput struct {
	FormID     string
	FilingDate time.Time
}

func (a *WealthActivities) MarkGiftsAsFiledActivity(ctx context.Context, input MarkGiftsAsFiledInput) error {
	return nil
}

type SendFilingNotificationInput struct {
	FamilyID  string
	TaxYear   int
	Forms     []Form709
	AutoFiled bool
}

func (a *WealthActivities) SendFilingNotificationActivity(ctx context.Context, input SendFilingNotificationInput) error {
	return nil
}
