package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	dbpkg "github.com/hondyman/uisce/backend/internal/db"
	"github.com/jmoiron/sqlx"

	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
)

const (
	edgeTypeMapsTo     = "87a92dd1-51fc-442c-9a59-1037ccb03d24"
	edgeTypeFK         = "9ca50284-8dc1-435f-86bd-92093b078589"
	edgeTypeBORel      = "d5fd8908-96ad-4ac5-b2e0-f86bc666f6bd"
	semanticTermTypeID = "820b942a-9c9e-4abc-acdc-84616db33098"
)

type BOWizardHandler struct {
	db *sqlx.DB
}

func NewBOWizardHandler(db *sqlx.DB) *BOWizardHandler {
	return &BOWizardHandler{db: db}
}

func (h *BOWizardHandler) RegisterRoutes(r chi.Router) {
	r.Get("/bo-wizard/context/{tableId}", h.GetDrivingTableContext)
	r.Post("/bo-wizard/save", h.SaveBusinessObject)
}

type wizardTerm struct {
	TermID      string  `json:"termId" db:"term_id"`
	TermName    string  `json:"termName" db:"term_name"`
	DisplayName string  `json:"displayName" db:"display_name"`
	ColumnID    string  `json:"columnId" db:"column_id"`
	ColumnName  string  `json:"columnName" db:"column_name"`
	DataType    *string `json:"dataType,omitempty" db:"data_type"`
	Selected    bool    `json:"selected" db:"selected"`
}

type wizardRelated struct {
	TableID        string `json:"tableId" db:"related_table_id"`
	TableName      string `json:"tableName" db:"related_table_name"`
	FKName         string `json:"fkName" db:"fk_name"`
	ExistingBOID   *string `json:"existingBOId,omitempty" db:"existing_bo_id"`
	ExistingBOName *string `json:"existingBOName,omitempty" db:"existing_bo_name"`
	LinkType       string `json:"linkType"`
}

type saveWizardRequest struct {
	BOKey         string   `json:"bo_key"`
	Name          string   `json:"name"`
	DisplayName   string   `json:"display_name"`
	Description   string   `json:"description"`
	DriverTableID string   `json:"driver_table_id"`
	SelectedTerms []string `json:"selected_terms"`
}

func (h *BOWizardHandler) tenantAndDS(r *http.Request) (tenantID, datasourceID string, ok bool) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims != nil {
		tenantID = claims.TenantID
	}
	if tenantID == "" {
		tenantID = strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	}
	datasourceID = strings.TrimSpace(r.Header.Get("X-Tenant-Datasource-ID"))
	if datasourceID == "" {
		datasourceID = strings.TrimSpace(r.Header.Get("X-Datasource-Id"))
	}
	return tenantID, datasourceID, tenantID != "" && datasourceID != ""
}

func (h *BOWizardHandler) GetDrivingTableContext(w http.ResponseWriter, r *http.Request) {
	tableID := chi.URLParam(r, "tableId")
	tenantID, datasourceID, ok := h.tenantAndDS(r)
	if !ok || tableID == "" {
		http.Error(w, "tableId, tenant, and datasource are required", http.StatusBadRequest)
		return
	}

	var driving struct {
		ID            string `db:"id" json:"id"`
		Name          string `db:"node_name" json:"name"`
		QualifiedPath string `db:"qualified_path" json:"qualifiedPath"`
	}
	err := h.db.GetContext(r.Context(), &driving, `
		SELECT id, node_name, COALESCE(qualified_path, node_name) AS qualified_path
		FROM catalog_node WHERE id = $1::uuid
	`, tableID)
	if err != nil {
		http.Error(w, "driving table not found: "+err.Error(), http.StatusNotFound)
		return
	}

	var terms []wizardTerm
	err = h.db.SelectContext(r.Context(), &terms, `
		SELECT DISTINCT
			st.id::text AS term_id,
			st.node_name AS term_name,
			COALESCE(st.properties->>'title', st.node_name) AS display_name,
			col.id::text AS column_id,
			col.node_name AS column_name,
			col.properties->>'data_type' AS data_type,
			EXISTS (
				SELECT 1 FROM business_object_fields f
				WHERE f.term_node_id = st.id AND f.tenant_id = $2::uuid
			) AS selected
		FROM catalog_node col
		JOIN catalog_edge e ON e.target_node_id = col.id AND e.edge_type_id = $4::uuid
		JOIN catalog_node st ON st.id = e.source_node_id
		JOIN catalog_node_type stt ON stt.id = st.node_type_id AND stt.catalog_type_name = 'semantic_term'
		JOIN catalog_node_type ct ON ct.id = col.node_type_id AND ct.catalog_type_name = 'column'
		WHERE col.qualified_path LIKE $1 || '/%'
		  AND col.tenant_id = $2::uuid
		  AND (col.tenant_datasource_id = $3::uuid OR $3 = '')
		ORDER BY st.node_name
	`, driving.QualifiedPath, tenantID, datasourceID, edgeTypeMapsTo)
	if err != nil {
		http.Error(w, "failed to load terms: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if terms == nil {
		terms = []wizardTerm{}
	}

	var related []wizardRelated
	_ = h.db.SelectContext(r.Context(), &related, `
		SELECT DISTINCT
			rt.id::text AS related_table_id,
			rt.node_name AS related_table_name,
			COALESCE(e.properties->>'constraint_name', e.relationship_type, 'FK') AS fk_name,
			bo.id::text AS existing_bo_id,
			bo.bo_name AS existing_bo_name
		FROM catalog_edge e
		JOIN catalog_node rt ON rt.id = CASE WHEN e.source_node_id = $1::uuid THEN e.target_node_id ELSE e.source_node_id END
		LEFT JOIN business_objects bo ON bo.driver_table_id = rt.id AND bo.tenant_id = $2::uuid
		WHERE (e.source_node_id = $1::uuid OR e.target_node_id = $1::uuid)
		  AND e.edge_type_id IN ($3::uuid, $4::uuid)
		  AND rt.id <> $1::uuid
		ORDER BY rt.node_name
	`, tableID, tenantID, edgeTypeFK, edgeTypeBORel)
	if related == nil {
		related = []wizardRelated{}
	}
	for i := range related {
		if related[i].ExistingBOID != nil {
			related[i].LinkType = "link_bo"
		} else {
			related[i].LinkType = "include_terms"
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"drivingTable": map[string]interface{}{
			"id": driving.ID, "name": driving.Name, "qualifiedPath": driving.QualifiedPath,
			"columnCount": 0, "termCount": len(terms), "relatedCount": len(related),
		},
		"semanticTerms": terms,
		"relatedTables": related,
	})
}

func (h *BOWizardHandler) SaveBusinessObject(w http.ResponseWriter, r *http.Request) {
	tenantID, datasourceID, ok := h.tenantAndDS(r)
	if !ok {
		http.Error(w, "tenant and datasource are required", http.StatusBadRequest)
		return
	}
	var req saveWizardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	boKey := strings.ToLower(strings.TrimSpace(req.BOKey))
	if boKey == "" || req.DriverTableID == "" {
		http.Error(w, "bo_key and driver_table_id are required", http.StatusBadRequest)
		return
	}
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		http.Error(w, "invalid tenant", http.StatusBadRequest)
		return
	}
	tableID, err := uuid.Parse(req.DriverTableID)
	if err != nil {
		http.Error(w, "invalid driver_table_id", http.StatusBadRequest)
		return
	}

	var tablePath string
	if err := h.db.GetContext(r.Context(), &tablePath, `SELECT COALESCE(qualified_path, node_name) FROM catalog_node WHERE id = $1`, tableID); err != nil {
		http.Error(w, "driving table not found", http.StatusBadRequest)
		return
	}

	tx, err := h.db.BeginTxx(r.Context(), nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Predicate delta: none — SQL already binds tid; GUC for FORCE RLS.
	if err := dbpkg.ApplyTenantGUCs(r.Context(), tx.Tx, tid.String(), ""); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	boID := uuid.New()
	_, err = tx.ExecContext(r.Context(), `
		INSERT INTO business_objects (
			id, tenant_id, model_id, bo_key, bo_name, description, bo_type,
			classification_node_id, business_key_node_id, semantic_id_node_id, grain_node_id,
			is_active, is_core, driver_table_id, driver_table_name
		) VALUES ($1,$2,$1,$3,$4,$5,'ENTITY',$6,$6,$6,$6,true,false,$6,$7)
		ON CONFLICT (tenant_id, bo_key) DO UPDATE SET
			bo_name = EXCLUDED.bo_name,
			driver_table_id = EXCLUDED.driver_table_id,
			driver_table_name = EXCLUDED.driver_table_name,
			updated_at = NOW()
		RETURNING id
	`, boID, tid, boKey, coalesce(req.Name, boKey), req.Description, tableID, tablePath)
	if err != nil {
		http.Error(w, "failed to save BO: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tx.GetContext(r.Context(), &boID, `SELECT id FROM business_objects WHERE tenant_id = $1 AND bo_key = $2`, tid, boKey); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var backendID uuid.UUID
	_ = tx.GetContext(r.Context(), &backendID, `
		SELECT backend_id FROM business_object_binding WHERE tenant_id = $1 LIMIT 1
	`, tid)
	if backendID != uuid.Nil {
		_, _ = tx.ExecContext(r.Context(), `
			INSERT INTO business_object_binding (tenant_id, bo_id, backend_id, driving_node_id, is_default, temporal_override, is_active)
			VALUES ($1,$2,$3,$4,true,'NONE',true)
			ON CONFLICT DO NOTHING
		`, tid, boID, backendID, tableID)
	}

	for _, termID := range req.SelectedTerms {
		tidTerm, err := uuid.Parse(termID)
		if err != nil {
			continue
		}
		var termName, colName string
		_ = tx.QueryRowContext(r.Context(), `
			SELECT st.node_name, COALESCE(col.node_name, st.node_name)
			FROM catalog_node st
			LEFT JOIN catalog_edge e ON e.source_node_id = st.id AND e.edge_type_id = $2::uuid
			LEFT JOIN catalog_node col ON col.id = e.target_node_id
			WHERE st.id = $1
			LIMIT 1
		`, tidTerm, edgeTypeMapsTo).Scan(&termName, &colName)
		if termName == "" {
			continue
		}
		_, err = tx.ExecContext(r.Context(), `
			INSERT INTO business_object_fields (
				tenant_id, bo_id, term_node_id, field_name, field_role, eligibility_source,
				is_exposed, display_name, technical_name, is_required
			) VALUES ($1,$2,$3,$4,'DIMENSION','DIRECT',true,$4,$5,false)
			ON CONFLICT ON CONSTRAINT uq_bo_field_tenant DO NOTHING
		`, tid, boID, tidTerm, termName, colName)
		if err != nil {
			http.Error(w, "failed to bind term: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = datasourceID
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": boID.String(), "bo_key": boKey})
}

func coalesce(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

// silence unused if Scan uses sql.NullString later
var _ = sql.ErrNoRows
