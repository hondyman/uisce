package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"portfolio-management/internal/hierarchy"
)

// ============================================================================
// Hierarchy Handlers
// ============================================================================

// handleValidateHierarchy validates a parent-child entity relationship
func handleValidateHierarchy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request
	var req struct {
		TenantID        string `json:"tenant_id"`
		ParentModelType string `json:"parent_model_type"`
		ChildModelType  string `json:"child_model_type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	result, err := hierarchyService.ValidateHierarchy(ctx, req.TenantID, req.ParentModelType, req.ChildModelType)
	if err != nil {
		log.Printf("Validation error: %v", err)
		http.Error(w, "Validation failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleGetHierarchyRules retrieves all hierarchy rules for a tenant
func handleGetHierarchyRules(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		http.Error(w, "tenant_id parameter required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rules, err := hierarchyService.GetHierarchyRules(ctx, tenantID)
	if err != nil {
		log.Printf("Error fetching hierarchy rules: %v", err)
		http.Error(w, "Failed to fetch rules", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"rules": rules,
		"count": len(rules),
	})
}

// handleGetHierarchySummary retrieves a summary of hierarchy rules with active relationships
func handleGetHierarchySummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		http.Error(w, "tenant_id parameter required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	summary, err := hierarchyService.GetHierarchySummary(ctx, tenantID)
	if err != nil {
		log.Printf("Error fetching hierarchy summary: %v", err)
		http.Error(w, "Failed to fetch summary", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"summary": summary,
		"count":   len(summary),
	})
}

// handleGetHierarchyTree retrieves the entity hierarchy tree
func handleGetHierarchyTree(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rootID := r.URL.Query().Get("root_id")
	if rootID == "" {
		http.Error(w, "root_id parameter required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	tree, err := hierarchyService.GetEntityHierarchy(ctx, rootID, 50)
	if err != nil {
		log.Printf("Error fetching hierarchy tree: %v", err)
		http.Error(w, "Failed to fetch tree", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tree": tree,
	})
}

// handleGetHierarchyStats retrieves statistics about the hierarchy
func handleGetHierarchyStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		http.Error(w, "tenant_id parameter required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	stats, err := hierarchyService.GetHierarchyStats(ctx, tenantID)
	if err != nil {
		log.Printf("Error fetching hierarchy stats: %v", err)
		http.Error(w, "Failed to fetch stats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// handleImportHierarchy imports hierarchy rules from JSON
func handleImportHierarchy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		http.Error(w, "tenant_id parameter required", http.StatusBadRequest)
		return
	}

	var req hierarchy.HierarchyImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	result, err := hierarchyService.ImportHierarchyRules(ctx, tenantID, &req)
	if err != nil {
		log.Printf("Error importing hierarchy: %v", err)
		http.Error(w, "Import failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result)
}
