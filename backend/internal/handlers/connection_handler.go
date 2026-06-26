package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/metadata"
)

type TestConnectionHandler struct {
	scanService *metadata.CatalogScanService
}

func NewTestConnectionHandler(scanService *metadata.CatalogScanService) *TestConnectionHandler {
	return &TestConnectionHandler{scanService: scanService}
}

// RegisterRoutes mounts connection test routes
func (h *TestConnectionHandler) RegisterRoutes(r chi.Router) {
	r.Post("/api/test-connection", h.HandleTestConnection)
}

type TestConnectionRequest struct {
	// Flattened fields to support both {id: "..."} and {input: {connection_details: "..."}}
	ID                string `json:"id"`
	ConnectionDetails string `json:"connection_details"` // Direct field from GraphQL mutation
	Input             struct {
		ConnectionDetails string            `json:"connection_details"`
		Type              string            `json:"type"`   // postgres, starrocks, iceberg
		Config            map[string]string `json:"config"` // generic config map
	} `json:"input"`
}

type TestConnectionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func (h *TestConnectionHandler) HandleTestConnection(w http.ResponseWriter, r *http.Request) {
	var req TestConnectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "invalid request body", "details": err.Error()})
		return
	}

	h.processTestConnection(w, r, req)
}

// Unified processing logic
func (h *TestConnectionHandler) processTestConnection(w http.ResponseWriter, r *http.Request, req TestConnectionRequest) {
	var err error

	// If ID is provided, use ID-based lookup
	if req.ID != "" {
		id, parseErr := uuid.Parse(req.ID)
		if parseErr != nil {
			h.jsonResponse(w, http.StatusBadRequest, TestConnectionResponse{Success: false, Message: "Invalid ID format"})
			return
		}
		err = h.scanService.TestConnectionByID(r.Context(), id)
	} else if req.ConnectionDetails != "" {
		// Direct connection_details field (from GraphQL mutation)
		err = h.scanService.TestConnection(r.Context(), req.ConnectionDetails)
	} else if req.Input.ConnectionDetails != "" {
		// Hasura-wrapped connection string test
		err = h.scanService.TestConnection(r.Context(), req.Input.ConnectionDetails)
	} else if req.Input.Type != "" {
		// Structured connection test (Phase 6)
		switch req.Input.Type {
		case "starrocks":
			// Test StarRocks (MySQL protocol)
			// req.Input.Config should contain host, port, user, password
			// For MVP, we pass the DSN
			err = h.scanService.TestConnection(r.Context(), req.Input.Config["dsn"])
		case "iceberg":
			// Test Iceberg (Nessie/S3)
			// Check if Nessie is reachable
			url := req.Input.Config["catalog_uri"]
			resp, httpErr := http.Get(url)
			if httpErr != nil || resp.StatusCode >= 400 {
				err = fmt.Errorf("failed to connect to iceberg catalog: %v", httpErr)
			}
			if resp != nil {
				resp.Body.Close()
			}
		default:
			err = fmt.Errorf("unsupported connection type: %s", req.Input.Type)
		}
	} else {
		h.jsonResponse(w, http.StatusBadRequest, TestConnectionResponse{Success: false, Message: "Missing id, connection_details (root or input.connection_details), or type/config"})
		return
	}

	if err != nil {
		h.jsonResponse(w, http.StatusOK, TestConnectionResponse{Success: false, Message: err.Error()})
		return
	}

	h.jsonResponse(w, http.StatusOK, TestConnectionResponse{Success: true, Message: "Connection successful"})
}

func (h *TestConnectionHandler) jsonResponse(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
