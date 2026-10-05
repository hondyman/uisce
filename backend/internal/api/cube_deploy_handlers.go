package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.temporal.io/api/serviceerror"

	"github.com/hondyman/uisce/backend/internal/handlers"
	"github.com/hondyman/uisce/backend/internal/querybuilder"
)

// cubeDeployRequest is the optional body for POST /api/cubes/{id}/deploy|refresh.
type cubeDeployRequest struct {
	Grain []string `json:"grain,omitempty"` // when empty, all cube grains are started
	Force bool     `json:"force,omitempty"` // deploy defaults force=true; refresh defaults false
}

// HandleCubeDeploy starts CubeMaterializeWorkflow for one or all grains (force=true).
// Concurrent starts for the same grain return 409 already_running.
func (s *Server) HandleCubeDeploy(w http.ResponseWriter, r *http.Request) {
	s.handleCubeMaterializeStart(w, r, true)
}

// HandleCubeRefresh starts CubeMaterializeWorkflow allowing content_hash noop (force=false).
func (s *Server) HandleCubeRefresh(w http.ResponseWriter, r *http.Request) {
	s.handleCubeMaterializeStart(w, r, false)
}

func (s *Server) handleCubeMaterializeStart(w http.ResponseWriter, r *http.Request, defaultForce bool) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", s.SecurityContextDeps)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if secCtx == nil || secCtx.TenantID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	cubeID := strings.TrimSpace(chi.URLParam(r, "id"))
	if cubeID == "" {
		http.Error(w, "cube id required", http.StatusBadRequest)
		return
	}

	var body cubeDeployRequest
	if r.Body != nil && r.ContentLength != 0 {
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&body); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
	}
	force := defaultForce
	if r.URL.Path != "" && strings.HasSuffix(r.URL.Path, "/refresh") {
		force = body.Force // refresh: only force when explicitly requested
	} else if body.Force {
		force = true
	}

	grains, err := s.cubeGrainsForDeploy(r, secCtx.TenantID, cubeID, body.Grain)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(grains) == 0 {
		http.Error(w, "cube has no grains to materialize", http.StatusBadRequest)
		return
	}

	type startResult struct {
		Grain      []string `json:"grain"`
		GrainHash  string   `json:"grain_hash,omitempty"`
		WorkflowID string   `json:"workflow_id,omitempty"`
		AttemptID  string   `json:"attempt_id,omitempty"`
		Noop       bool     `json:"noop,omitempty"`
		NoopReason string   `json:"noop_reason,omitempty"`
		AlreadyRunning bool `json:"already_running,omitempty"`
		Error      string   `json:"error,omitempty"`
	}
	results := make([]startResult, 0, len(grains))
	anyConflict := false
	anyOK := false

	for _, grain := range grains {
		req := querybuilder.CubeMaterializeRequest{
			TenantID: secCtx.TenantID,
			CubeID:   cubeID,
			Grain:    grain,
			Force:    force,
		}
		wfID, plan, startErr := s.StartCubeMaterialize(r.Context(), req)
		res := startResult{Grain: grain}
		if plan != nil {
			res.GrainHash = plan.GrainHash
			res.AttemptID = plan.AttemptID
			res.Noop = plan.Noop
			res.NoopReason = plan.NoopReason
		}
		if startErr != nil {
			if isAlreadyRunning(startErr) {
				anyConflict = true
				res.AlreadyRunning = true
				res.WorkflowID = querybuilder.CubeMaterializeWorkflowID(
					secCtx.TenantID, cubeID,
					planContractVersion(plan),
					res.GrainHash,
				)
				res.Error = "already running"
			} else {
				res.Error = startErr.Error()
			}
			results = append(results, res)
			continue
		}
		anyOK = true
		res.WorkflowID = wfID
		results = append(results, res)
	}

	status := http.StatusAccepted
	if anyConflict && !anyOK {
		status = http.StatusConflict
	} else if !anyOK {
		status = http.StatusBadRequest
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"cube_id": cubeID,
		"force":   force,
		"starts":  results,
	})
}

func planContractVersion(plan *querybuilder.CubeMaterializePlan) int {
	if plan == nil {
		return 1
	}
	return plan.ContractVersion
}

func isAlreadyRunning(err error) bool {
	if err == nil {
		return false
	}
	var started *serviceerror.WorkflowExecutionAlreadyStarted
	if errors.As(err, &started) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "already started") || strings.Contains(msg, "already running")
}

func (s *Server) cubeGrainsForDeploy(r *http.Request, tenantID, cubeID string, want []string) ([][]string, error) {
	if s.CubeHandler == nil {
		return nil, errors.New("cubes: handler not configured")
	}
	cube, err := s.CubeHandler.GetCubeForTenant(r.Context(), tenantID, cubeID)
	if err != nil {
		return nil, err
	}
	if len(want) > 0 {
		return [][]string{want}, nil
	}
	return cube.Grains, nil
}
