package agentic

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/libs/jwt-middleware"
	"github.com/jmoiron/sqlx"
)

type ProposalRequest struct {
	TenantID   string          `json:"tenant_id"`
	AgentID    string          `json:"agent_id"`
	TargetBOID string          `json:"target_bo_id"`
	ActionType string          `json:"action_type"`
	Payload    json.RawMessage `json:"proposed_payload"`
}

type ApprovalTicket struct {
	TicketID                    string          `json:"ticket_id" db:"ticket_id"`
	TenantID                    string          `json:"tenant_id" db:"tenant_id"`
	AgentID                     string          `json:"agent_id" db:"agent_id"`
	TargetBOID                  string          `json:"target_bo_id" db:"target_bo_id"`
	ActionType                  string          `json:"action_type" db:"action_type"`
	ProposedPayload             json.RawMessage `json:"proposed_payload" db:"proposed_payload"`
	ComplianceValidationResults json.RawMessage `json:"compliance_validation_results" db:"compliance_validation_results"`
	Status                      string          `json:"status" db:"status"`
	CreatedByAI                 bool            `json:"created_by_ai" db:"created_by_ai"`
	CheckedBy                   *string         `json:"checked_by,omitempty" db:"checked_by"`
	CheckedAt                   *time.Time      `json:"checked_at,omitempty" db:"checked_at"`
	CreatedAt                   time.Time       `json:"created_at" db:"created_at"`
}

type MakerCheckerService struct {
	db *sqlx.DB
}

func NewMakerCheckerService(db *sqlx.DB) *MakerCheckerService {
	return &MakerCheckerService{db: db}
}

// SubmitAgentProposal intercepts an AI tool action, runs compliance checks, and creates a pending ticket
func (s *MakerCheckerService) SubmitAgentProposal(ctx context.Context, req ProposalRequest) (string, error) {
	if req.TenantID == "" {
		req.TenantID = "core"
	}
	if req.AgentID == "" {
		req.AgentID = "AutonomousAI-Agent"
	}

	// 1. Run Pre-Trade / Pre-Action Compliance Engine rules against payload
	passed, violations := s.evaluateComplianceRules(req.TargetBOID, req.Payload)

	status := "PENDING_CHECKER"
	if !passed {
		status = "COMPLIANCE_FAILED"
	}

	violationsJSON, _ := json.Marshal(map[string]interface{}{"passed": passed, "violations": violations})
	ticketID := uuid.New().String()

	if s.db != nil {
		query := `
			INSERT INTO public.agent_approval_tickets 
			(ticket_id, tenant_id, agent_id, target_bo_id, action_type, proposed_payload, compliance_validation_results, status, created_by_ai)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, true)
		`
		_, err := s.db.ExecContext(ctx, query,
			ticketID, req.TenantID, req.AgentID, req.TargetBOID, req.ActionType, string(req.Payload), string(violationsJSON), status,
		)
		if err != nil {
			return "", fmt.Errorf("failed to create agent approval ticket: %w", err)
		}
	}

	if !passed {
		return ticketID, fmt.Errorf("proposal generated but blocked by pre-trade compliance rules: %v", violations)
	}

	return ticketID, nil
}

func (s *MakerCheckerService) ApproveTicket(ctx context.Context, ticketID, reviewerID string) error {
	if s.db == nil {
		return nil
	}
	query := `UPDATE public.agent_approval_tickets SET status = 'APPROVED', checked_by = $1, checked_at = NOW(), updated_at = NOW() WHERE ticket_id = $2`
	_, err := s.db.ExecContext(ctx, query, reviewerID, ticketID)
	return err
}

func (s *MakerCheckerService) RejectTicket(ctx context.Context, ticketID, reviewerID, reason string) error {
	if s.db == nil {
		return nil
	}
	query := `UPDATE public.agent_approval_tickets SET status = 'REJECTED', checked_by = $1, checked_at = NOW(), updated_at = NOW() WHERE ticket_id = $2`
	_, err := s.db.ExecContext(ctx, query, reviewerID, ticketID)
	return err
}

func (s *MakerCheckerService) ListTickets(ctx context.Context, tenantID string) ([]ApprovalTicket, error) {
	if s.db == nil {
		// Mock response if DB uninitialized
		now := time.Now()
		reviewer := "pm_jane_doe"
		return []ApprovalTicket{
			{
				TicketID:                    "tkt-991823",
				TenantID:                    tenantID,
				AgentID:                     "PortfolioRebalanceAgent-v3",
				TargetBOID:                  "customers",
				ActionType:                  "BULK_REBALANCE",
				ProposedPayload:             json.RawMessage(`{"portfolio_id": "PORT-991", "rebalance_orders": [{"symbol": "AAPL", "qty": 500}, {"symbol": "AGG", "qty": 1200}]}`),
				ComplianceValidationResults: json.RawMessage(`{"passed": true, "violations": []}`),
				Status:                      "PENDING_CHECKER",
				CreatedByAI:                 true,
				CreatedAt:                   now.Add(-10 * time.Minute),
			},
			{
				TicketID:                    "tkt-991824",
				TenantID:                    tenantID,
				AgentID:                     "RiskGuardAgent-v1",
				TargetBOID:                  "customers",
				ActionType:                  "STATUS_OVERRIDE",
				ProposedPayload:             json.RawMessage(`{"account_id": "ACC-8812", "new_status": "RESTRICTED"}`),
				ComplianceValidationResults: json.RawMessage(`{"passed": true, "violations": []}`),
				Status:                      "APPROVED",
				CreatedByAI:                 true,
				CheckedBy:                   &reviewer,
				CheckedAt:                   &now,
				CreatedAt:                   now.Add(-2 * time.Hour),
			},
		}, nil
	}

	query := `SELECT ticket_id, tenant_id, agent_id, target_bo_id, action_type, proposed_payload, compliance_validation_results, status, created_by_ai, checked_by, checked_at, created_at FROM public.agent_approval_tickets WHERE tenant_id = $1 ORDER BY created_at DESC`
	var tickets []ApprovalTicket
	err := s.db.SelectContext(ctx, &tickets, query, tenantID)
	return tickets, err
}

func (s *MakerCheckerService) evaluateComplianceRules(boID string, payload json.RawMessage) (bool, []string) {
	// Simple validation check against payload
	str := string(payload)
	if str == "" || str == "{}" {
		return false, []string{"Payload cannot be empty"}
	}
	return true, []string{}
}

// HTTP Handlers

func (s *MakerCheckerService) ListTicketsHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := "core"
	if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil && claims.TenantID != "" {
		tenantID = claims.TenantID
	}

	tickets, err := s.ListTickets(r.Context(), tenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch tickets: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"tickets": tickets})
}

func (s *MakerCheckerService) ReviewTicketHandler(w http.ResponseWriter, r *http.Request) {
	var body struct {
		TicketID string `json:"ticket_id"`
		Decision string `json:"decision"` // APPROVE, REJECT
		Reason   string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	reviewerID := "portfolio_manager_user"
	if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil && claims.UserID != "" {
		reviewerID = claims.UserID
	}

	var err error
	if body.Decision == "APPROVE" {
		err = s.ApproveTicket(r.Context(), body.TicketID, reviewerID)
	} else {
		err = s.RejectTicket(r.Context(), body.TicketID, reviewerID, body.Reason)
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to process decision: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "ticket_id": body.TicketID, "status": body.Decision})
}
