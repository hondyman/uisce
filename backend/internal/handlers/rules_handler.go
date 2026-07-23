package handlers

import (
	"database/sql"

	"github.com/gorilla/mux"
)

// PriorityStep represents a single priority rule
type PriorityStep struct {
	ID          string    `json:"id"`
	Priority    int       `json:"priority"`
	Condition   Condition `json:"condition"`
	Action      Action    `json:"action"`
	Description string    `json:"description"`
}

// Condition defines the IF clause
type Condition struct {
	SemanticTerm string `json:"semanticTerm"`
	Operator     string `json:"operator"`
	Value        string `json:"value"`
}

// Action defines the THEN clause
type Action struct {
	UseField   string `json:"useField"`
	Confidence int    `json:"confidence"`
}

// Rule represents a semantic priority rule
type Rule struct {
	ID                 string         `json:"id"`
	BusinessObject     string         `json:"businessObject"`
	Name               string         `json:"name"`
	Description        string         `json:"description"`
	Version            int            `json:"version"`
	Status             string         `json:"status"` // draft, testing, staging, production
	Steps              []PriorityStep `json:"steps"`
	DefaultAction      string         `json:"defaultAction"`
	CreatedAt          string         `json:"createdAt"`
	UpdatedAt          string         `json:"updatedAt"`
	CreatedBy          string         `json:"createdBy"`
	TenantID           string         `json:"tenantId"`
	SemanticTerm       string         `json:"semanticTerm,omitempty"`       // For gold copy publishing
	RuleEngine         string         `json:"ruleEngine,omitempty"`         // For gold copy publishing
	ExpressionLanguage string         `json:"expressionLanguage,omitempty"` // For gold copy publishing
}

// RuleHandler handles rule-related HTTP requests
type RuleHandler struct {
	db                *sql.DB     // PostgreSQL connection pool
	cache             interface{} // Would be your cache layer (Redis, etc.)
	goldCopyPublisher interface{} // *services.GoldCopyPublisher - avoid import cycle
	executionEngine   interface{} // *mdm.ExecutionEngine - avoid import cycle
}

// NewRuleHandler creates a new rule handler (deprecated)
func NewRuleHandler() *RuleHandler {
	return &RuleHandler{}
}

// RegisterRoutes registers rule routes with the provided router
func (h *RuleHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/v1/rules", h.CreateRule).Methods("POST")
	router.HandleFunc("/api/v1/rules/{ruleId}", h.GetRule).Methods("GET")
	router.HandleFunc("/api/v1/rules/{ruleId}", h.UpdateRule).Methods("PUT")
	router.HandleFunc("/api/v1/rules/{ruleId}", h.DeleteRule).Methods("DELETE")
	router.HandleFunc("/api/v1/rules", h.ListRules).Methods("GET")
	router.HandleFunc("/api/v1/rules/{ruleId}/publish", h.PublishRule).Methods("POST")
	router.HandleFunc("/api/v1/rules/{ruleId}/promote", h.PromoteRule).Methods("POST")
	router.HandleFunc("/api/v1/rules/{ruleId}/simulate", h.SimulateRule).Methods("POST")
	router.HandleFunc("/api/v1/rules/{ruleId}/versions", h.GetVersions).Methods("GET")
	router.HandleFunc("/api/v1/rules/{ruleId}/diff", h.GetDiff).Methods("GET")
	router.HandleFunc("/api/v1/rules/{ruleId}/rollback", h.RollbackRule).Methods("POST")
	router.HandleFunc("/api/v1/rules/{ruleId}/approve", h.RequestApproval).Methods("POST")
	router.HandleFunc("/api/v1/approvals/pending", h.GetPendingApprovals).Methods("GET")
}
