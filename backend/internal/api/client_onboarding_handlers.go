package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// ============================================================================
// CLIENT ONBOARDING HANDLERS
// ============================================================================

type ClientOnboardingHandler struct {
	service *ClientOnboardingService
	db      *sqlx.DB
}

// NewClientOnboardingHandler creates a new handler instance
func NewClientOnboardingHandler(db *sqlx.DB) *ClientOnboardingHandler {
	return &ClientOnboardingHandler{
		service: NewClientOnboardingService(db),
		db:      db,
	}
}

// ============================================================================
// CLIENT MANAGEMENT
// ============================================================================

// CreateClientHandler creates a new client record (before onboarding starts)
// POST /api/clients
func (h *ClientOnboardingHandler) CreateClientHandler(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")
	userID := r.Header.Get("X-User-ID")

	if tenantID == "" || datasourceID == "" || userID == "" {
		http.Error(w, "Missing tenant or user context", http.StatusBadRequest)
		return
	}

	var req ClientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	client, err := h.service.CreateClient(r.Context(), tenantID, datasourceID, userID, &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create client: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(client)
}

// GetClientHandler retrieves a client
// GET /api/clients/{clientID}
func (h *ClientOnboardingHandler) GetClientHandler(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	clientID := chi.URLParam(r, "clientID")

	if tenantID == "" || clientID == "" {
		http.Error(w, "Missing tenant or client ID", http.StatusBadRequest)
		return
	}

	client, err := h.service.GetClient(r.Context(), tenantID, clientID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Client not found: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(client)
}

// ListClientsHandler lists all clients in onboarding
// GET /api/clients?limit=20&offset=0
func (h *ClientOnboardingHandler) ListClientsHandler(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "Missing tenant ID", http.StatusBadRequest)
		return
	}

	limit := 20
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	clients, total, err := h.service.ListClientsInOnboarding(r.Context(), tenantID, limit, offset)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list clients: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"clients": clients,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ============================================================================
// STEP 1: VALIDATE CLIENT DATA AGAINST REGULATORY REQUIREMENTS
// ============================================================================

// Step1ValidateClientHandler validates KYC/AML and client data
// POST /api/onboarding/step1/validate
func (h *ClientOnboardingHandler) Step1ValidateClientHandler(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")
	userID := r.Header.Get("X-User-ID")

	if tenantID == "" || datasourceID == "" || userID == "" {
		http.Error(w, "Missing tenant or user context", http.StatusBadRequest)
		return
	}

	var req ValidateClientDataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Get the client
	client, err := h.service.GetClient(r.Context(), tenantID, req.ClientID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Client not found: %v", err), http.StatusNotFound)
		return
	}

	// Create validation result
	validationErrors := make(map[string]interface{})
	validationPassed := true

	// Check required fields
	if client.IdentificationNumber == nil || *client.IdentificationNumber == "" {
		validationErrors["identification_number"] = "identification number is required"
		validationPassed = false
	}

	if client.DateOfBirth == nil || *client.DateOfBirth == "" {
		validationErrors["date_of_birth"] = "date of birth is required"
		validationPassed = false
	}

	// Check for high-net-worth requiring due diligence
	if req.RequiresDueDiligence {
		validationErrors["due_diligence_flag"] = req.DueDiligenceReason
	}

	// Create onboarding workflow if it doesn't exist
	workflowID := fmt.Sprintf("client-onboard-%s-%d", req.ClientID, time.Now().Unix())
	workflow, err := h.service.CreateOnboardingWorkflow(r.Context(), tenantID, datasourceID, req.ClientID, workflowID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create workflow: %v", err), http.StatusInternalServerError)
		return
	}

	// Update client with workflow ID
	query := `UPDATE clients SET temporal_workflow_id = $1, kyc_status = $2, aml_status = $3 WHERE id = $4`
	kyc := "pending"
	aml := "pending"

	if validationPassed && !req.RequiresDueDiligence {
		kyc = "approved"
		aml = "approved"
	}

	_, err = h.db.ExecContext(r.Context(), query, workflowID, kyc, aml, req.ClientID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update client: %v", err), http.StatusInternalServerError)
		return
	}

	// Record event
	event := &OnboardingEvent{
		TenantID:     tenantID,
		DatasourceID: datasourceID,
		WorkflowID:   workflow.ID,
		EventType:    "validation_started",
		EventData: map[string]interface{}{
			"validation_passed": validationPassed,
			"errors":            validationErrors,
			"verification_kyc":  req.VerifyKYC,
			"perform_aml":       req.PerformAMLScreening,
		},
		TriggeredBy: &userID,
		ActorType:   "system",
		StepNumber:  ptrInt(1),
	}
	h.service.RecordOnboardingEvent(r.Context(), event)

	response := map[string]interface{}{
		"client_id":           req.ClientID,
		"workflow_id":         workflowID,
		"validation_passed":   validationPassed,
		"errors":              validationErrors,
		"kyc_status":          kyc,
		"aml_status":          aml,
		"next_step":           "route_for_advisor_review",
		"requires_escalation": len(validationErrors) > 0,
	}

	w.Header().Set("Content-Type", "application/json")
	if validationPassed {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusUnprocessableEntity)
	}
	json.NewEncoder(w).Encode(response)
}

// ============================================================================
// STEP 2: ROUTE FOR ADVISOR REVIEW/APPROVAL
// ============================================================================

// Step2RouteForReviewHandler routes client to an advisor
// POST /api/onboarding/step2/route
func (h *ClientOnboardingHandler) Step2RouteForReviewHandler(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")
	userID := r.Header.Get("X-User-ID")

	if tenantID == "" || datasourceID == "" || userID == "" {
		http.Error(w, "Missing tenant or user context", http.StatusBadRequest)
		return
	}

	var req RouteForReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Update client with advisor assignment
	query := `
		UPDATE clients 
		SET assigned_advisor_id = $1, onboarding_status = $2, onboarding_stage = $3, updated_at = $4
		WHERE id = $5
	`
	_, err := h.db.ExecContext(r.Context(), query, req.AdvisorID, "pending_review", 2, time.Now(), req.ClientID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to assign advisor: %v", err), http.StatusInternalServerError)
		return
	}

	// Get workflow
	var workflowID string
	wfQuery := `SELECT workflow_id FROM onboarding_workflows WHERE client_id = $1`
	err = h.db.GetContext(r.Context(), &workflowID, wfQuery, req.ClientID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get workflow: %v", err), http.StatusInternalServerError)
		return
	}

	// Record event
	event := &OnboardingEvent{
		TenantID:     tenantID,
		DatasourceID: datasourceID,
		WorkflowID:   workflowID,
		EventType:    "review_assigned",
		EventData: map[string]interface{}{
			"advisor_id":   req.AdvisorID,
			"priority":     req.Priority,
			"review_notes": req.ReviewNotes,
		},
		TriggeredBy: &userID,
		ActorType:   "user",
		ActorRole:   ptrString("compliance_officer"),
		StepNumber:  ptrInt(2),
	}
	h.service.RecordOnboardingEvent(r.Context(), event)

	response := map[string]interface{}{
		"client_id":  req.ClientID,
		"advisor_id": req.AdvisorID,
		"status":     "pending_review",
		"next_step":  "send_agreements",
		"message":    "Client routed to advisor for review",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// ============================================================================
// STEP 3: GENERATE AND SEND AGREEMENTS FOR E-SIGNATURE
// ============================================================================

// Step3GenerateAgreementsHandler generates and sends agreements
// POST /api/onboarding/step3/agreements
func (h *ClientOnboardingHandler) Step3GenerateAgreementsHandler(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")
	userID := r.Header.Get("X-User-ID")

	if tenantID == "" || datasourceID == "" || userID == "" {
		http.Error(w, "Missing tenant or user context", http.StatusBadRequest)
		return
	}

	var req GenerateAgreementsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	client, err := h.service.GetClient(r.Context(), tenantID, req.ClientID)
	if err != nil {
		http.Error(w, "Client not found", http.StatusNotFound)
		return
	}

	// Create agreement documents
	for _, agreementType := range req.AgreementTypes {
		doc := &Document{
			DocumentType:        agreementType,
			DocumentName:        fmt.Sprintf("%s - %s %s", agreementType, client.FirstName, client.LastName),
			Status:              "pending_signature",
			VerificationStatus:  "unverified",
			ESignatureStatus:    ptrString("pending"),
			ESignatureRequestID: ptrString(fmt.Sprintf("docusign-%d", time.Now().Unix())),
		}

		_, err := h.service.CreateDocument(r.Context(), tenantID, datasourceID, req.ClientID, userID, doc)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to create agreement: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Update client status
	now := time.Now()
	query := `
		UPDATE clients 
		SET onboarding_status = $1, onboarding_stage = $2, agreements_sent_date = $3, updated_at = $4
		WHERE id = $5
	`
	_, err = h.db.ExecContext(r.Context(), query, "pending_agreements", 3, now, now, req.ClientID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update client: %v", err), http.StatusInternalServerError)
		return
	}

	// Get workflow for event recording
	var workflowID string
	wfQuery := `SELECT workflow_id FROM onboarding_workflows WHERE client_id = $1`
	err = h.db.GetContext(r.Context(), &workflowID, wfQuery, req.ClientID)

	// Record event
	if err == nil {
		event := &OnboardingEvent{
			TenantID:     tenantID,
			DatasourceID: datasourceID,
			WorkflowID:   workflowID,
			EventType:    "agreement_sent",
			EventData: map[string]interface{}{
				"agreement_types":    req.AgreementTypes,
				"e_signature_method": req.ESignatureMethod,
				"delivery_method":    req.DeliveryMethod,
			},
			TriggeredBy: &userID,
			ActorType:   "user",
			StepNumber:  ptrInt(3),
		}
		h.service.RecordOnboardingEvent(r.Context(), event)
	}

	response := map[string]interface{}{
		"client_id":          req.ClientID,
		"agreements_sent":    len(req.AgreementTypes),
		"e_signature_method": req.ESignatureMethod,
		"status":             "pending_agreements",
		"next_step":          "create_accounts",
		"message":            "Agreements sent for e-signature",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// ============================================================================
// STEP 4: CREATE LINKED ACCOUNTS AND PORTFOLIOS
// ============================================================================

// Step4CreateAccountsHandler creates investment accounts and portfolios
// POST /api/onboarding/step4/accounts
func (h *ClientOnboardingHandler) Step4CreateAccountsHandler(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")
	userID := r.Header.Get("X-User-ID")

	if tenantID == "" || datasourceID == "" || userID == "" {
		http.Error(w, "Missing tenant or user context", http.StatusBadRequest)
		return
	}

	var req CreateAccountsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	client, err := h.service.GetClient(r.Context(), tenantID, req.ClientID)
	if err != nil {
		http.Error(w, "Client not found", http.StatusNotFound)
		return
	}

	createdAccounts := []map[string]interface{}{}

	// Create accounts based on client risk profile
	for i, accountType := range req.AccountTypes {
		account := &Account{
			AccountNumber:  fmt.Sprintf("ACC-%s-%d", req.ClientID[:8], time.Now().Unix()+int64(i)),
			AccountType:    accountType,
			Status:         "pending_funding",
			InitialBalance: 0,
			CurrentBalance: 0,
			Currency:       "USD",
			CustodianName:  &req.Custodian,
			AllowsMargin:   false,
			AllowsOptions:  false,
			AllowsCrypto:   false,
		}

		if req.InitialFunding != nil && *req.InitialFunding > 0 {
			account.InitialBalance = *req.InitialFunding
			account.CurrentBalance = *req.InitialFunding
		}

		createdAccount, err := h.service.CreateAccount(r.Context(), tenantID, datasourceID, req.ClientID, userID, account)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to create account: %v", err), http.StatusInternalServerError)
			return
		}

		// Create initial portfolio based on risk profile
		portfolio := &Portfolio{
			PortfolioName:      fmt.Sprintf("%s - %s Portfolio", client.RiskProfile, accountType),
			PortfolioType:      ptrString("model"),
			Status:             "active",
			RiskLevel:          &client.RiskProfile,
			RebalanceFrequency: "quarterly",
			InceptionDate:      time.Now().Format("2006-01-02"),
			AllocationJSON:     getAllocationForRiskProfile(client.RiskProfile),
		}

		_, err = h.service.CreatePortfolio(r.Context(), tenantID, datasourceID, createdAccount.ID, userID, portfolio)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to create portfolio: %v", err), http.StatusInternalServerError)
			return
		}

		createdAccounts = append(createdAccounts, map[string]interface{}{
			"account_id":     createdAccount.ID,
			"account_number": createdAccount.AccountNumber,
			"type":           accountType,
		})
	}

	// Update client status
	now := time.Now()
	query := `
		UPDATE clients 
		SET onboarding_status = $1, onboarding_stage = $2, updated_at = $3
		WHERE id = $4
	`
	_, err = h.db.ExecContext(r.Context(), query, "pending_notification", 4, now, req.ClientID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update client: %v", err), http.StatusInternalServerError)
		return
	}

	// Get workflow for event recording
	var workflowID string
	wfQuery := `SELECT workflow_id FROM onboarding_workflows WHERE client_id = $1`
	err = h.db.GetContext(r.Context(), &workflowID, wfQuery, req.ClientID)

	if err == nil {
		event := &OnboardingEvent{
			TenantID:     tenantID,
			DatasourceID: datasourceID,
			WorkflowID:   workflowID,
			EventType:    "account_created",
			EventData: map[string]interface{}{
				"account_count": len(createdAccounts),
				"accounts":      createdAccounts,
			},
			TriggeredBy: &userID,
			ActorType:   "system",
			StepNumber:  ptrInt(4),
		}
		h.service.RecordOnboardingEvent(r.Context(), event)
	}

	response := map[string]interface{}{
		"client_id":        req.ClientID,
		"accounts_created": createdAccounts,
		"status":           "pending_notification",
		"next_step":        "notify_client",
		"message":          fmt.Sprintf("%d accounts created successfully", len(createdAccounts)),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// ============================================================================
// STEP 5: NOTIFY CLIENT UPON COMPLETION
// ============================================================================

// Step5NotifyClientHandler sends completion notification
// POST /api/onboarding/step5/notify
func (h *ClientOnboardingHandler) Step5NotifyClientHandler(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")
	userID := r.Header.Get("X-User-ID")

	if tenantID == "" || datasourceID == "" || userID == "" {
		http.Error(w, "Missing tenant or user context", http.StatusBadRequest)
		return
	}

	var req NotifyClientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	client, err := h.service.GetClient(r.Context(), tenantID, req.ClientID)
	if err != nil {
		http.Error(w, "Client not found", http.StatusNotFound)
		return
	}

	// Mark onboarding as complete
	query := `
		UPDATE clients 
		SET onboarding_status = $1, onboarding_stage = $2, updated_at = $3
		WHERE id = $4
	`
	_, err = h.db.ExecContext(r.Context(), query, "active", 5, time.Now(), req.ClientID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update client: %v", err), http.StatusInternalServerError)
		return
	}

	// Get workflow for event recording
	var workflowID string
	wfQuery := `SELECT workflow_id FROM onboarding_workflows WHERE client_id = $1`
	err = h.db.GetContext(r.Context(), &workflowID, wfQuery, req.ClientID)

	if err == nil {
		event := &OnboardingEvent{
			TenantID:     tenantID,
			DatasourceID: datasourceID,
			WorkflowID:   workflowID,
			EventType:    "onboarding_completed",
			EventData: map[string]interface{}{
				"notification_type": req.NotificationType,
				"portal_access_url": req.PortalAccessURL,
			},
			TriggeredBy: &userID,
			ActorType:   "system",
			StepNumber:  ptrInt(5),
		}
		h.service.RecordOnboardingEvent(r.Context(), event)
	}

	response := map[string]interface{}{
		"client_id":         req.ClientID,
		"client_name":       fmt.Sprintf("%s %s", client.FirstName, client.LastName),
		"email":             client.Email,
		"status":            "active",
		"notification_sent": true,
		"notification_type": req.NotificationType,
		"portal_access_url": req.PortalAccessURL,
		"message":           "Client onboarding completed successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// ============================================================================
// WORKFLOW STATUS & MANAGEMENT
// ============================================================================

// GetOnboardingStatusHandler returns current onboarding status
// GET /api/onboarding/status/{clientID}
func (h *ClientOnboardingHandler) GetOnboardingStatusHandler(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	clientID := chi.URLParam(r, "clientID")

	if tenantID == "" || clientID == "" {
		http.Error(w, "Missing tenant or client ID", http.StatusBadRequest)
		return
	}

	status, err := h.service.GetOnboardingStatus(r.Context(), tenantID, clientID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get status: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// ApproveOnboardingHandler approves client onboarding
// POST /api/onboarding/approve
func (h *ClientOnboardingHandler) ApproveOnboardingHandler(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	if tenantID == "" || datasourceID == "" {
		http.Error(w, "Missing tenant context", http.StatusBadRequest)
		return
	}

	var req ApproveOnboardingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	workflow, err := h.service.GetOnboardingWorkflow(r.Context(), tenantID, req.WorkflowID)
	if err != nil {
		http.Error(w, "Workflow not found", http.StatusNotFound)
		return
	}

	// Update workflow approval
	query := `
		UPDATE onboarding_workflows 
		SET approved_by = $1, approved_at = $2, overall_status = $3, updated_at = $4
		WHERE id = $5
	`
	_, err = h.db.ExecContext(r.Context(), query, req.AdvisorID, time.Now(), "in_progress", time.Now(), workflow.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to approve: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"workflow_id": req.WorkflowID,
		"approved_by": req.AdvisorID,
		"status":      "approved",
		"message":     "Onboarding approved and will proceed to next steps",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// ============================================================================
// ROUTE REGISTRATION
// ============================================================================

// RegisterClientOnboardingRoutes registers all onboarding routes
func RegisterClientOnboardingRoutes(router chi.Router, db *sqlx.DB) {
	handler := NewClientOnboardingHandler(db)

	// Client management
	router.Post("/api/clients", handler.CreateClientHandler)
	router.Get("/api/clients", handler.ListClientsHandler)
	router.Get("/api/clients/{clientID}", handler.GetClientHandler)

	// Onboarding workflow steps
	router.Post("/api/onboarding/step1/validate", handler.Step1ValidateClientHandler)
	router.Post("/api/onboarding/step2/route", handler.Step2RouteForReviewHandler)
	router.Post("/api/onboarding/step3/agreements", handler.Step3GenerateAgreementsHandler)
	router.Post("/api/onboarding/step4/accounts", handler.Step4CreateAccountsHandler)
	router.Post("/api/onboarding/step5/notify", handler.Step5NotifyClientHandler)

	// Status & Management
	router.Get("/api/onboarding/status/{clientID}", handler.GetOnboardingStatusHandler)
	router.Post("/api/onboarding/approve", handler.ApproveOnboardingHandler)
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func ptrString(s string) *string {
	return &s
}

// getAllocationForRiskProfile returns appropriate asset allocation based on risk profile
func getAllocationForRiskProfile(riskProfile string) map[string]interface{} {
	allocations := map[string]map[string]interface{}{
		"low": {
			"equities":     20,
			"fixed_income": 60,
			"alternatives": 10,
			"cash":         10,
		},
		"moderate": {
			"equities":     50,
			"fixed_income": 30,
			"alternatives": 10,
			"cash":         10,
		},
		"high": {
			"equities":     70,
			"fixed_income": 15,
			"alternatives": 10,
			"cash":         5,
		},
		"very_high": {
			"equities":     80,
			"fixed_income": 10,
			"alternatives": 10,
			"cash":         0,
		},
	}

	if alloc, ok := allocations[riskProfile]; ok {
		return alloc
	}

	// Default to moderate if not found
	return allocations["moderate"]
}
