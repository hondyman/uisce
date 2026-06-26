package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/hondyman/semlayer/backend/internal/events"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type GoldCopyEventHandler struct {
	db        *sqlx.DB
	publisher *events.KafkaPublisher
	logger    *zap.Logger
}

func NewGoldCopyEventHandler(db *sqlx.DB, publisher *events.KafkaPublisher, logger *zap.Logger) *GoldCopyEventHandler {
	return &GoldCopyEventHandler{
		db:        db,
		publisher: publisher,
		logger:    logger,
	}
}

type GoldCopyEventPayload struct {
	Event struct {
		Op   string `json:"op"`
		Data struct {
			Old map[string]interface{} `json:"old"`
			New map[string]interface{} `json:"new"`
		} `json:"data"`
	} `json:"event"`
	Table struct {
		Schema string `json:"schema"`
		Name   string `json:"name"`
	} `json:"table"`
}

func (h *GoldCopyEventHandler) HandleEvent(w http.ResponseWriter, r *http.Request) {
	var payload GoldCopyEventPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.logger.Error("Failed to decode webhook payload", zap.Error(err))
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	// Determine data based on operation
	var record map[string]interface{}
	action := payload.Event.Op
	if action == "DELETE" {
		record = payload.Event.Data.Old
	} else {
		record = payload.Event.Data.New
	}

	if record == nil {
		h.logger.Warn("Received event with no data", zap.String("op", action))
		w.WriteHeader(http.StatusOK)
		return
	}

	// 1. Identify Entity Type from Table Name
	tableName := payload.Table.Name
	var entityType string
	switch tableName {
	case "connections":
		entityType = "connection"
	case "tenant_instance":
		entityType = "instance"
	case "tenant_product":
		entityType = "product"
	case "tenant_product_datasource":
		entityType = "datasource"
	default:
		h.logger.Warn("Received event for unknown table", zap.String("table", tableName))
		w.WriteHeader(http.StatusOK)
		return
	}

	// 2. Extract Tenant ID (varies by table)
	var tenantID string
	var err error

	if entityType == "connection" || entityType == "instance" || entityType == "product" {
		// These tables have direct tenant_id column
		if tid, ok := record["tenant_id"].(string); ok {
			tenantID = tid
		}
	} else if entityType == "datasource" {
		// Datasource links to tenant_product, need to resolve tenant_id?
		// Actually, Hasura payload might not contain joined data.
		// For simplicity, let's assume we can query it or it's in the record if augmented.
		// But wait, standard triggers just send row data.
		// tenant_product_datasource -> tenant_product_id -> tenant_product -> tenant_id
		// Ideally we query DB here to get tenant_id if missing.

		// For now, let's try to fetch it if missing.
		if tpdID, ok := record["tenant_product_id"].(string); ok {
			err = h.db.Get(&tenantID, `
				SELECT tp.tenant_id 
				FROM tenant_product tp 
				WHERE tp.id = $1
			`, tpdID)
			if err != nil {
				h.logger.Error("Failed to resolve tenant_id for datasource", zap.Error(err))
				// Can't validate gold copy without tenant_id
				w.WriteHeader(http.StatusOK)
				return
			}
		}
	}

	if tenantID == "" {
		h.logger.Warn("Could not determine tenant_id", zap.String("entity", entityType))
		w.WriteHeader(http.StatusOK)
		return
	}

	// 3. Check Gold Copy Status
	// Optimization: Cache result? For now, DB query.
	var isGoldCopy bool
	err = h.db.Get(&isGoldCopy, "SELECT gold_copy FROM tenants WHERE id = $1", tenantID)
	if err != nil {
		h.logger.Error("Failed to query tenant status", zap.Error(err), zap.String("tenant_id", tenantID))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if !isGoldCopy {
		// Log debug only
		// h.logger.Debug("Ignoring event for non-gold-copy tenant", zap.String("tenant_id", tenantID))
		w.WriteHeader(http.StatusOK)
		return
	}

	// 4. Construct Event
	entityID, _ := record["id"].(string)
	userID, _ := record["created_by"].(string) // Assuming created_by exists

	evt := &events.GoldCopyEntityEvent{
		EventType:  events.GoldCopyEntityChanged,
		TenantID:   tenantID,
		EntityType: entityType,
		EntityID:   entityID,
		Action:     action,
		Data:       record,
		Timestamp:  time.Now(),
	}
	if userID != "" {
		evt.UserID = &userID
	}

	// 5. Publish
	if err := h.publisher.PublishGoldCopyEntityEvent(r.Context(), evt); err != nil {
		h.logger.Error("Failed to publish Gold Copy entity event", zap.Error(err))
		http.Error(w, "Failed to publish event", http.StatusInternalServerError)
		return
	}

	h.logger.Info("Published Gold Copy entity event",
		zap.String("type", entityType),
		zap.String("action", action),
		zap.String("id", entityID))

	w.WriteHeader(http.StatusOK)
}
