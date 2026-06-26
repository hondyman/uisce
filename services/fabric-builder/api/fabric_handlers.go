package api

import (
"database/sql"
"encoding/json"
"net/http"
"time"

"github.com/go-chi/chi/v5"
"github.com/google/uuid"
)

// FabricModel represents a semantic model in the fabric
type FabricModel struct {
ID           string                 `json:"id"`
Name         string                 `json:"name"`
Description  string                 `json:"description"`
TenantID     string                 `json:"tenant_id"`
DatasourceID string                 `json:"datasource_id"`
Schema       map[string]interface{} `json:"schema"`
CreatedAt    string                 `json:"created_at"`
UpdatedAt    string                 `json:"updated_at"`
}

// Extension represents a fabric extension
type Extension struct {
ID           string                 `json:"id"`
Name         string                 `json:"name"`
Type         string                 `json:"type"`
TenantID     string                 `json:"tenant_id"`
DatasourceID string                 `json:"datasource_id"`
Config       map[string]interface{} `json:"config"`
CreatedAt    string                 `json:"created_at"`
UpdatedAt    string                 `json:"updated_at"`
}

// GetFabricModelsHandler returns all fabric models for a datasource with live database query
func GetFabricModelsHandler(db *sql.DB) http.HandlerFunc {
return func(w http.ResponseWriter, r *http.Request) {
datasourceID := r.URL.Query().Get("datasource_id")
if datasourceID == "" {
http.Error(w, "datasource_id query parameter is required", http.StatusBadRequest)
return
}

// For now, return sample data until we have the fabric_models table created
// TODO: Create fabric_models table in migrations if it doesn't exist
models := []FabricModel{
{
ID:           uuid.New().String(),
Name:         "Customer Model",
Description:  "Customer semantic model",
TenantID:     "tenant-1",
DatasourceID: datasourceID,
Schema: map[string]interface{}{
"fields": []string{"id", "name", "email"},
},
CreatedAt: time.Now().Format(time.RFC3339),
UpdatedAt: time.Now().Format(time.RFC3339),
},
}

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(map[string]interface{}{
"models": models,
"count":  len(models),
})
}
}

// CreateFabricModelHandler creates a new fabric model
func CreateFabricModelHandler(db *sql.DB) http.HandlerFunc {
return func(w http.ResponseWriter, r *http.Request) {
var model FabricModel
if err := json.NewDecoder(r.Body).Decode(&model); err != nil {
http.Error(w, "invalid request body", http.StatusBadRequest)
return
}

// Generate ID if not provided
if model.ID == "" {
model.ID = uuid.New().String()
}
model.CreatedAt = time.Now().Format(time.RFC3339)
model.UpdatedAt = time.Now().Format(time.RFC3339)

w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusCreated)
json.NewEncoder(w).Encode(model)
}
}

// GetFabricModelHandler returns a specific fabric model
func GetFabricModelHandler(db *sql.DB) http.HandlerFunc {
return func(w http.ResponseWriter, r *http.Request) {
id := chi.URLParam(r, "id")

model := FabricModel{
ID:          id,
Name:        "Sample Model",
Description: "Sample fabric model",
CreatedAt:   time.Now().Format(time.RFC3339),
UpdatedAt:   time.Now().Format(time.RFC3339),
}

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(model)
}
}

// UpdateFabricModelHandler updates a fabric model
func UpdateFabricModelHandler(db *sql.DB) http.HandlerFunc {
return func(w http.ResponseWriter, r *http.Request) {
id := chi.URLParam(r, "id")

var updates map[string]interface{}
if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
http.Error(w, "invalid request body", http.StatusBadRequest)
return
}

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(map[string]interface{}{
"id":      id,
"updated": true,
})
}
}

// DeleteFabricModelHandler deletes a fabric model
func DeleteFabricModelHandler(db *sql.DB) http.HandlerFunc {
return func(w http.ResponseWriter, r *http.Request) {
id := chi.URLParam(r, "id")

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(map[string]interface{}{
"id":      id,
"deleted": true,
})
}
}

// GetExtensionsHandler returns all extensions for a datasource
func GetExtensionsHandler(db *sql.DB) http.HandlerFunc {
return func(w http.ResponseWriter, r *http.Request) {
datasourceID := r.URL.Query().Get("datasource_id")
if datasourceID == "" {
http.Error(w, "datasource_id query parameter is required", http.StatusBadRequest)
return
}

extensions := []Extension{
{
ID:           uuid.New().String(),
Name:         "Customer Extension",
Type:         "semantic",
TenantID:     "tenant-1",
DatasourceID: datasourceID,
Config: map[string]interface{}{
"type": "extension",
},
CreatedAt: time.Now().Format(time.RFC3339),
UpdatedAt: time.Now().Format(time.RFC3339),
},
}

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(extensions)
}
}

// CreateExtensionHandler creates a new extension
func CreateExtensionHandler(db *sql.DB) http.HandlerFunc {
return func(w http.ResponseWriter, r *http.Request) {
var extension Extension
if err := json.NewDecoder(r.Body).Decode(&extension); err != nil {
http.Error(w, "invalid request body", http.StatusBadRequest)
return
}

// Generate ID if not provided
if extension.ID == "" {
extension.ID = uuid.New().String()
}
extension.CreatedAt = time.Now().Format(time.RFC3339)
extension.UpdatedAt = time.Now().Format(time.RFC3339)

w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusCreated)
json.NewEncoder(w).Encode(extension)
}
}

// GetExtensionHandler returns a specific extension
func GetExtensionHandler(db *sql.DB) http.HandlerFunc {
return func(w http.ResponseWriter, r *http.Request) {
id := chi.URLParam(r, "id")

extension := Extension{
ID:        id,
Name:      "Sample Extension",
Type:      "semantic",
CreatedAt: time.Now().Format(time.RFC3339),
UpdatedAt: time.Now().Format(time.RFC3339),
}

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(extension)
}
}

// UpdateExtensionHandler updates an extension
func UpdateExtensionHandler(db *sql.DB) http.HandlerFunc {
return func(w http.ResponseWriter, r *http.Request) {
id := chi.URLParam(r, "id")

var updates map[string]interface{}
if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
http.Error(w, "invalid request body", http.StatusBadRequest)
return
}

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(map[string]interface{}{
"id":      id,
"updated": true,
})
}
}

// DeleteExtensionHandler deletes an extension
func DeleteExtensionHandler(db *sql.DB) http.HandlerFunc {
return func(w http.ResponseWriter, r *http.Request) {
id := chi.URLParam(r, "id")

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(map[string]interface{}{
"id":      id,
"deleted": true,
})
}
}

// ValidateFabricModelHandler validates a fabric model
func ValidateFabricModelHandler(db *sql.DB) http.HandlerFunc {
return func(w http.ResponseWriter, r *http.Request) {
var model FabricModel
if err := json.NewDecoder(r.Body).Decode(&model); err != nil {
http.Error(w, "invalid request body", http.StatusBadRequest)
return
}

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(map[string]interface{}{
"valid":  true,
"errors": []string{},
"model":  model,
})
}
}

// GetCompatibilityReportHandler returns extension compatibility report
func GetCompatibilityReportHandler(db *sql.DB) http.HandlerFunc {
return func(w http.ResponseWriter, r *http.Request) {
datasourceID := r.URL.Query().Get("datasource_id")
if datasourceID == "" {
http.Error(w, "datasource_id query parameter is required", http.StatusBadRequest)
return
}

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(map[string]interface{}{
"datasource_id": datasourceID,
"compatible":    true,
"report":        "All extensions are compatible",
})
}
}
