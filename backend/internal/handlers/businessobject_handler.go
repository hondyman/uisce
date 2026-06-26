//go:build ignore
// +build ignore

package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/services"
	kafka "github.com/segmentio/kafka-go"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// BusinessObjectHandler handles HTTP requests for business objects
type BusinessObjectHandler struct {
	boService      *services.BusinessObjectService
	eventPublisher *services.EventPublisher
	commandBus     *services.CommandPublisher
	enabled        bool
}

// NewBusinessObjectHandler creates a new BO handler
func NewBusinessObjectHandler(
	boService *services.BusinessObjectService,
	eventPublisher *services.EventPublisher,
	commandBus *services.CommandPublisher,
) *BusinessObjectHandler {
	handler := &BusinessObjectHandler{
		boService:      boService,
		eventPublisher: eventPublisher,
		commandBus:     commandBus,
	}

	// If command bus is enabled, setup reply queue
	if commandBus != nil && commandBus.IsEnabled() {
		// Channel will be set when needed
		handler.enabled = true
	}

	return handler
}

// waitForCommandResponse waits for a command response with a timeout
func (h *BusinessObjectHandler) waitForCommandResponse(ctx context.Context, correlationID string, timeout time.Duration) (*services.CommandResponse, error) {
	// If command bus is not enabled, return error
	if !h.enabled {
		return nil, fmt.Errorf("command bus not available")
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	brokers := strings.Split(os.Getenv("KAFKA_BROKERS"), ",")
	if len(brokers) == 0 || brokers[0] == "" {
		brokers = []string{"localhost:9092"}
	}

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		GroupID:     fmt.Sprintf("api-reply-%s", correlationID),
		Topic:       "semlayer.replies",
		MinBytes:    10e3,
		MaxBytes:    10e6,
		StartOffset: kafka.LastOffset,
	})
	defer r.Close()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("command response timeout")
		default:
			m, err := r.FetchMessage(ctx)
			if err != nil {
				continue
			}
			if string(m.Key) != correlationID {
				// Not our response; commit and continue
				r.CommitMessages(ctx, m)
				continue
			}

			var response services.CommandResponse
			if err := json.Unmarshal(m.Value, &response); err != nil {
				return nil, fmt.Errorf("failed to unmarshal response: %w", err)
			}
			r.CommitMessages(ctx, m)
			return &response, nil
		}
	}
}

// ============================================================================
// BUSINESS OBJECT ENDPOINTS (Command Bus Pattern)
// ============================================================================

// POST /api/business-objects - Create a new BO via command bus
func (h *BusinessObjectHandler) CreateBusinessObject(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	userID := r.Header.Get("X-User-ID")

	if tenantID == "" {
		http.Error(w, "Missing X-Tenant-ID header", http.StatusBadRequest)
		return
	}

	var req models.CreateBusinessObjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Extract datasource from header if not in request body
	if req.DatasourceID == "" {
		req.DatasourceID = r.Header.Get("X-Tenant-Datasource-ID")
		if req.DatasourceID == "" {
			req.DatasourceID = r.URL.Query().Get("datasource_id")
		}
	}

	// If command bus is disabled, fall back to direct service call
	if !h.enabled {
		bo, err := h.boService.CreateBusinessObject(r.Context(), tenantID, req, userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Publish event
		if h.eventPublisher != nil {
			h.eventPublisher.PublishBOCreated(r.Context(), bo, userID)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(bo)
		return
	}

	// Publish command to command bus
	correlationID, err := h.commandBus.PublishCommand(
		r.Context(),
		services.CommandCreateBO,
		tenantID,
		userID,
		req,
	)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to publish command: %v", err), http.StatusInternalServerError)
		return
	}

	// Wait for command response (with 10 second timeout)
	response, err := h.waitForCommandResponse(r.Context(), correlationID, 10*time.Second)
	if err != nil {
		http.Error(w, fmt.Sprintf("Command failed: %v", err), http.StatusInternalServerError)
		return
	}

	if response.Status != services.CommandStatusSuccess {
		http.Error(w, response.Error, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response.Data)
}

// GET /api/business-objects - List all BOs
func (h *BusinessObjectHandler) ListBusinessObjects(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	if tenantID == "" {
		http.Error(w, "Missing X-Tenant-ID header", http.StatusBadRequest)
		return
	}

	// List is a read-only operation, always call service directly
	bos, err := h.boService.ListBusinessObjects(r.Context(), tenantID, datasourceID)
	if err != nil {
		reqID := r.Header.Get("X-Request-ID")
		logging.GetLogger().Sugar().Errorf("ListBusinessObjects failed: request_id=%s tenant=%s datasource=%s err=%v", reqID, tenantID, datasourceID, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error":      "Internal server error",
			"request_id": reqID,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bos)
}

// GET /api/business-objects/{key} - Get a specific BO
func (h *BusinessObjectHandler) GetBusinessObject(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	key := chi.URLParam(r, "key")

	if tenantID == "" {
		http.Error(w, "Missing X-Tenant-ID header", http.StatusBadRequest)
		return
	}

	logging.GetLogger().Sugar().Infof("DEBUG: HTTP BusinessObject Get - tenant=%s key=%s", tenantID, key)
	// Get is a read-only operation, always call service directly
	bo, err := h.boService.GetBusinessObject(r.Context(), tenantID, key)
	if err != nil {
		logging.GetLogger().Sugar().Warnf("DEBUG: HTTP BusinessObject Get - service returned error: %v", err)
		http.Error(w, "BO not found", http.StatusNotFound)
		return
	}
	logging.GetLogger().Sugar().Infof("DEBUG: HTTP BusinessObject Get - service returned bo.id=%s subtypes=%d", bo.ID, len(bo.Subtypes))

	// If subtypes not populated, attach child BOs from list query
	if bo.Subtypes == nil || len(bo.Subtypes) == 0 {
		logging.GetLogger().Sugar().Infof("DEBUG: HTTP BusinessObject Get - subtypes empty, falling back to scanning all BOs")
		if tenantID != "" {
			all, err := h.boService.ListBusinessObjects(r.Context(), tenantID)
			if err == nil {
				for _, cand := range all {
					// Check if this BO is a child of the current parent (use .Valid and .String for sql.NullString)
					if cand.ParentID.Valid && cand.ParentID.String == bo.ID {
						logging.GetLogger().Sugar().Infof("DEBUG: Found subtype: candID=%s parentID=%s parentBOID=%s", cand.ID, cand.ParentID.String, bo.ID)
						// Build subtype shape
						sd := models.SubtypeDefinition{
							ID:            cand.ID,
							Key:           cand.Key,
							Name:          cand.Name,
							DisplayName:   cand.DisplayName,
							TechnicalName: cand.TechnicalName,
							Description:   cand.Description,
							IsCore:        cand.IsCore,
							SubtypeFields: append([]models.FieldDefinition{}, cand.CustomFields...),
						}
						if bo.Subtypes == nil {
							bo.Subtypes = make(map[string]models.SubtypeDefinition)
						}
						bo.Subtypes[sd.Key] = sd
					}
				}
				logging.GetLogger().Sugar().Infof("DEBUG: After fallback scan, bo.Subtypes count=%d", len(bo.Subtypes))
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bo)
}

// PUT /api/business-objects/{key} - Update a BO via command bus
func (h *BusinessObjectHandler) UpdateBusinessObject(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	userID := r.Header.Get("X-User-ID")
	key := chi.URLParam(r, "key")

	if tenantID == "" {
		http.Error(w, "Missing X-Tenant-ID header", http.StatusBadRequest)
		return
	}

	var req models.UpdateBusinessObjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// If command bus is disabled, fall back to direct service call
	if !h.enabled {
		bo, err := h.boService.UpdateBusinessObject(r.Context(), tenantID, key, req, userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Publish event
		if h.eventPublisher != nil {
			h.eventPublisher.PublishBOUpdated(r.Context(), bo, userID)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(bo)
		return
	}

	// Create update command with key
	updateCmd := map[string]interface{}{
		"key":  key,
		"data": req,
	}

	// Publish command to command bus
	correlationID, err := h.commandBus.PublishCommand(
		r.Context(),
		services.CommandUpdateBO,
		tenantID,
		userID,
		updateCmd,
	)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to publish command: %v", err), http.StatusInternalServerError)
		return
	}

	// Wait for command response
	response, err := h.waitForCommandResponse(r.Context(), correlationID, 10*time.Second)
	if err != nil {
		http.Error(w, fmt.Sprintf("Command failed: %v", err), http.StatusInternalServerError)
		return
	}

	if response.Status != services.CommandStatusSuccess {
		http.Error(w, response.Error, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response.Data)
}

// DELETE /api/business-objects/{key} - Delete a BO via command bus
func (h *BusinessObjectHandler) DeleteBusinessObject(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	userID := r.Header.Get("X-User-ID")
	key := chi.URLParam(r, "key")

	if tenantID == "" {
		http.Error(w, "Missing X-Tenant-ID header", http.StatusBadRequest)
		return
	}

	// If command bus is disabled, fall back to direct service call
	if !h.enabled {
		err := h.boService.DeleteBusinessObject(r.Context(), tenantID, key, userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Publish event
		if h.eventPublisher != nil {
			h.eventPublisher.PublishBODeleted(r.Context(), tenantID, key, userID)
		}

		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Publish command to command bus
	deleteCmd := map[string]interface{}{
		"key": key,
	}

	correlationID, err := h.commandBus.PublishCommand(
		r.Context(),
		services.CommandDeleteBO,
		tenantID,
		userID,
		deleteCmd,
	)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to publish command: %v", err), http.StatusInternalServerError)
		return
	}

	// Wait for command response
	response, err := h.waitForCommandResponse(r.Context(), correlationID, 10*time.Second)
	if err != nil {
		http.Error(w, fmt.Sprintf("Command failed: %v", err), http.StatusInternalServerError)
		return
	}

	if response.Status != services.CommandStatusSuccess {
		http.Error(w, response.Error, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// POST /api/business-objects/{key}/clone - Clone a BO via command bus
func (h *BusinessObjectHandler) CloneBusinessObject(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	userID := r.Header.Get("X-User-ID")
	key := chi.URLParam(r, "key")

	if tenantID == "" {
		http.Error(w, "Missing X-Tenant-ID header", http.StatusBadRequest)
		return
	}

	var req models.CloneBORequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req.SourceBOKey = key

	// If command bus is disabled, fall back to direct service call
	if !h.enabled {
		bo, err := h.boService.CloneBusinessObject(r.Context(), tenantID, req, userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Publish event
		if h.eventPublisher != nil {
			h.eventPublisher.PublishBOCloned(r.Context(), bo, key, userID)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(bo)
		return
	}

	// Publish command to command bus
	correlationID, err := h.commandBus.PublishCommand(
		r.Context(),
		services.CommandCloneBO,
		tenantID,
		userID,
		req,
	)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to publish command: %v", err), http.StatusInternalServerError)
		return
	}

	// Wait for command response
	response, err := h.waitForCommandResponse(r.Context(), correlationID, 10*time.Second)
	if err != nil {
		http.Error(w, fmt.Sprintf("Command failed: %v", err), http.StatusInternalServerError)
		return
	}

	if response.Status != services.CommandStatusSuccess {
		http.Error(w, response.Error, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response.Data)
}

// ============================================================================
// BO INSTANCES ENDPOINTS
// ============================================================================
// BO INSTANCES ENDPOINTS
// ============================================================================

// POST /api/bo/{boKey}/instances - Create a BO instance
func (h *BusinessObjectHandler) CreateInstance(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	userID := r.Header.Get("X-User-ID")

	if tenantID == "" || userID == "" {
		http.Error(w, "Missing X-Tenant-ID or X-User-ID headers", http.StatusBadRequest)
		return
	}

	var instance models.BusinessObjectInstance
	if err := json.NewDecoder(r.Body).Decode(&instance); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// If command bus is enabled, route through command bus
	if h.enabled && h.commandBus != nil {
		commandData := map[string]interface{}{
			"tenantID":          tenantID,
			"userID":            userID,
			"businessObjectKey": instance.BusinessObjectKey,
			"instance": map[string]interface{}{
				"businessObjectID":  instance.BusinessObjectID,
				"businessObjectKey": instance.BusinessObjectKey,
				"datasourceID":      instance.DatasourceID,
				"subtypeID":         instance.SubtypeID.String,
				"subtypeKey":        instance.SubtypeKey,
				"coreFieldValues":   instance.CoreFieldValues,
				"customFieldValues": instance.CustomFieldValues,
			},
		}

		// Publish command to message bus
		correlationID, err := h.commandBus.PublishCommand(r.Context(), services.CommandCreateInstance, tenantID, userID, commandData)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to publish command: %v", err), http.StatusInternalServerError)
			return
		}

		// Wait for response from command consumer
		response, err := h.waitForCommandResponse(r.Context(), correlationID, 10*time.Second)
		if err != nil {
			http.Error(w, fmt.Sprintf("Command timeout or failed: %v", err), http.StatusInternalServerError)
			return
		}

		if response.Status != services.CommandStatusSuccess {
			http.Error(w, response.Message, http.StatusInternalServerError)
			return
		}

		// Extract instance from response data
		if responseInstance, ok := response.Data.(map[string]interface{})["instance"].(models.BusinessObjectInstance); ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(responseInstance)
			return
		}

		// Fallback: if response parsing fails, marshal and unmarshal
		respJSON, _ := json.Marshal(response.Data.(map[string]interface{})["instance"])
		var created models.BusinessObjectInstance
		json.Unmarshal(respJSON, &created)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(created)
		return
	}

	// Fallback: direct service call if command bus is disabled
	created, err := h.boService.CreateInstance(r.Context(), tenantID, userID, &instance)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if h.eventPublisher != nil {
		h.eventPublisher.PublishInstanceCreated(r.Context(), created, userID)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

// GET /api/bo/{boKey}/instances - List instances
func (h *BusinessObjectHandler) ListInstances(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	boKey := chi.URLParam(r, "boKey")

	page := r.URL.Query().Get("page")
	pageSize := r.URL.Query().Get("page_size")
	pageNum := 1
	pageSz := 50

	if p, err := strconv.Atoi(page); err == nil && p > 0 {
		pageNum = p
	}
	if sz, err := strconv.Atoi(pageSize); err == nil && sz > 0 {
		pageSz = sz
	}

	if tenantID == "" {
		http.Error(w, "Missing X-Tenant-ID header", http.StatusBadRequest)
		return
	}

	offset := (pageNum - 1) * pageSz
	instances, total, err := h.boService.ListInstances(r.Context(), tenantID, boKey, offset, pageSz)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Total-Count", strconv.Itoa(total))
	w.Header().Set("X-Page", strconv.Itoa(pageNum))
	w.Header().Set("X-Page-Size", strconv.Itoa(pageSz))
	json.NewEncoder(w).Encode(instances)
}

// GET /api/bo/{boKey}/instances/{instanceID} - Get instance
func (h *BusinessObjectHandler) GetInstance(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	instanceID := chi.URLParam(r, "instanceID")

	if tenantID == "" {
		http.Error(w, "Missing X-Tenant-ID header", http.StatusBadRequest)
		return
	}

	instance, err := h.boService.GetInstance(r.Context(), tenantID, instanceID)
	if err != nil {
		http.Error(w, "Instance not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(instance)
}

// PUT /api/bo/{boKey}/instances/{instanceID} - Update instance
func (h *BusinessObjectHandler) UpdateInstance(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	userID := r.Header.Get("X-User-ID")
	instanceID := chi.URLParam(r, "instanceID")

	if tenantID == "" || userID == "" {
		http.Error(w, "Missing X-Tenant-ID or X-User-ID headers", http.StatusBadRequest)
		return
	}

	var req struct {
		CoreFields   map[string]interface{} `json:"coreFields,omitempty"`
		CustomFields map[string]interface{} `json:"customFields,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// If command bus is enabled, route through command bus
	if h.enabled && h.commandBus != nil {
		commandData := map[string]interface{}{
			"tenantID":           tenantID,
			"userID":             userID,
			"instanceID":         instanceID,
			"coreFieldUpdates":   req.CoreFields,
			"customFieldUpdates": req.CustomFields,
		}

		// Publish command to message bus
		correlationID, err := h.commandBus.PublishCommand(r.Context(), services.CommandUpdateInstance, tenantID, userID, commandData)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to publish command: %v", err), http.StatusInternalServerError)
			return
		}

		// Wait for response from command consumer
		response, err := h.waitForCommandResponse(r.Context(), correlationID, 10*time.Second)
		if err != nil {
			http.Error(w, fmt.Sprintf("Command timeout or failed: %v", err), http.StatusInternalServerError)
			return
		}

		if response.Status != services.CommandStatusSuccess {
			http.Error(w, response.Message, http.StatusInternalServerError)
			return
		}

		// Extract instance from response data
		if responseInstance, ok := response.Data.(map[string]interface{})["instance"].(models.BusinessObjectInstance); ok {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(responseInstance)
			return
		}

		// Fallback: if response parsing fails, marshal and unmarshal
		respJSON, _ := json.Marshal(response.Data.(map[string]interface{})["instance"])
		var updated models.BusinessObjectInstance
		json.Unmarshal(respJSON, &updated)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(updated)
		return
	}

	// Fallback: direct service call if command bus is disabled
	instance, err := h.boService.UpdateInstance(r.Context(), tenantID, instanceID, userID, req.CoreFields, req.CustomFields)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if h.eventPublisher != nil {
		h.eventPublisher.PublishInstanceUpdated(r.Context(), instance, userID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(instance)
}

// DELETE /api/bo/{boKey}/instances/{instanceID} - Delete instance
func (h *BusinessObjectHandler) DeleteInstance(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	userID := r.Header.Get("X-User-ID")
	instanceID := chi.URLParam(r, "instanceID")
	boKey := chi.URLParam(r, "boKey")

	if tenantID == "" || userID == "" {
		http.Error(w, "Missing X-Tenant-ID or X-User-ID headers", http.StatusBadRequest)
		return
	}

	// If command bus is enabled, route through command bus
	if h.enabled && h.commandBus != nil {
		commandData := map[string]interface{}{
			"tenantID":          tenantID,
			"userID":            userID,
			"instanceID":        instanceID,
			"businessObjectKey": boKey,
		}

		// Publish command to message bus
		correlationID, err := h.commandBus.PublishCommand(r.Context(), services.CommandDeleteInstance, tenantID, userID, commandData)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to publish command: %v", err), http.StatusInternalServerError)
			return
		}

		// Wait for response from command consumer
		response, err := h.waitForCommandResponse(r.Context(), correlationID, 10*time.Second)
		if err != nil {
			http.Error(w, fmt.Sprintf("Command timeout or failed: %v", err), http.StatusInternalServerError)
			return
		}

		if response.Status != services.CommandStatusSuccess {
			http.Error(w, response.Message, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Fallback: direct service call if command bus is disabled
	err := h.boService.DeleteInstance(r.Context(), tenantID, instanceID, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if h.eventPublisher != nil {
		h.eventPublisher.PublishInstanceDeleted(r.Context(), tenantID, boKey, instanceID, userID)
	}

	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// ROUTING
// ============================================================================

// RegisterRoutes registers all BO routes
func (h *BusinessObjectHandler) RegisterRoutes(router *chi.Mux) {
	// Business Objects
	router.Post("/business-objects", h.CreateBusinessObject)
	router.Get("/business-objects", h.ListBusinessObjects)
	router.Get("/business-objects/{key}", h.GetBusinessObject)
	router.Put("/business-objects/{key}", h.UpdateBusinessObject)
	router.Delete("/business-objects/{key}", h.DeleteBusinessObject)
	router.Post("/business-objects/{key}/clone", h.CloneBusinessObject)

	// Instances
	router.Post("/bo/{boKey}/instances", h.CreateInstance)
	router.Get("/bo/{boKey}/instances", h.ListInstances)
	router.Get("/bo/{boKey}/instances/{instanceID}", h.GetInstance)
	router.Put("/bo/{boKey}/instances/{instanceID}", h.UpdateInstance)
	router.Delete("/bo/{boKey}/instances/{instanceID}", h.DeleteInstance)
}
