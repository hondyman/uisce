package main

// cmd/aiserver/main.go

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

// PageLayout matches frontend type
type PageLayout struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	PrimaryBO  string          `json:"primaryBO"`
	LayoutType string          `json:"layoutType"` // "detail" | "form" | "list"
	Sections   []LayoutSection `json:"sections"`
}

type LayoutSection struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Type           string   `json:"type"` // "fields" | "related_list" | "custom"
	Columns        int      `json:"columns"`
	Collapsible    bool     `json:"collapsible"`
	FieldIDs       []string `json:"fieldIds,omitempty"`
	RelationshipID string   `json:"relationshipId,omitempty"`
	RelatedBO      string   `json:"relatedBO,omitempty"`
	ColumnFieldIDs []string `json:"columnFieldIds,omitempty"`
}

type GenLayoutReq struct {
	Prompt    string `json:"prompt"`
	PrimaryBO string `json:"primaryBO"`
}

type GenLayoutResp struct {
	Generated    PageLayout   `json:"generatedLayout"`
	Confidence   float64      `json:"confidence"`
	Alternatives []PageLayout `json:"alternatives"`
	Explanation  string       `json:"explanation"`
	ModelVersion string       `json:"modelVersion"`
	GeneratedAt  time.Time    `json:"generatedAt"`
	DraftID      string       `json:"draftId"` // ID in ai_layouts table
}

type FieldRecReq struct {
	PrimaryBO        string                 `json:"primaryBO"`
	SectionContext   map[string]interface{} `json:"sectionContext"`
	ExistingFieldIDs []string               `json:"existingFieldIds"`
}

type FieldRecommendation struct {
	FieldID    string  `json:"fieldId"`
	FieldLabel string  `json:"fieldLabel"`
	UsageScore float64 `json:"usageScore"`
	Reason     string  `json:"reason"`
}

type FieldRecResp struct {
	Recommendations []FieldRecommendation `json:"recommendations"`
	GeneratedAt     time.Time             `json:"generatedAt"`
}

// Mock usage signals; replace with real analytics
var mockFieldUsage = map[string]float64{
	"f1": 0.94, "f2": 0.91, "f3": 0.88, "f4": 0.81, "f5": 0.62,
	"f6": 0.57, "f7": 0.49, "f8": 0.77, "f9": 0.66, "f10": 0.59,
}

var db *sql.DB

func init() {
	// connStr := os.Getenv("DATABASE_URL")
	// if connStr == "" {
	// 	connStr = "postgres://postgres:postgres@host.docker.internal:5432/alpha?sslmode=disable"
	// }

	// var err error
	// db, err = sql.Open("postgres", connStr)
	// if err != nil {
	// 	log.Fatalf("failed to open db: %v", err)
	// }

	// err = db.Ping()
	// if err != nil {
	// 	log.Fatalf("failed to ping db: %v", err)
	// }

	log.Printf("database connected: (mocked)")
}

// extractTenant retrieves X-Tenant-ID from headers; returns empty string if missing
func extractTenant(r *http.Request) string {
	return r.Header.Get("X-Tenant-ID")
}

// persistDraft saves generated layout to ai_layouts table with adopted=false
func persistDraft(tenantID, primaryBO, name, layoutType, modelVersion string, confidence float64, payload, alternatives interface{}, explanation string) (string, error) {
	payloadJSON, _ := json.Marshal(payload)
	altsJSON, _ := json.Marshal(alternatives)

	var draftID string
	err := db.QueryRow(`
		INSERT INTO ai_layouts (tenant_id, primary_bo, name, layout_type, payload, alternatives, model_version, confidence, explanation, adopted, created_by, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, FALSE, 'system', TRUE)
		RETURNING id
	`, tenantID, primaryBO, name, layoutType, payloadJSON, altsJSON, modelVersion, confidence, explanation).Scan(&draftID)

	return draftID, err
}

// markAdopted updates ai_layouts to set adopted=true when user applies layout
func markAdopted(draftID, userID string) error {
	_, err := db.Exec(`
		UPDATE ai_layouts SET adopted = TRUE, adopted_at = NOW(), adopted_by = $1 WHERE id = $2
	`, userID, draftID)
	return err
}

func main() {
	mux := http.NewServeMux()

	// POST /api/ai/generate-layout
	mux.HandleFunc("/api/ai/generate-layout", func(w http.ResponseWriter, r *http.Request) {
		tenantID := extractTenant(r)
		if tenantID == "" {
			http.Error(w, "missing X-Tenant-ID header", 400)
			return
		}

		var req GenLayoutReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		// Deterministic layout generation based on prompt
		name := "AI " + strings.Title(req.PrimaryBO) + " Detail"
		cols := 2
		if strings.Contains(strings.ToLower(req.Prompt), "three") {
			cols = 3
		}

		layout := PageLayout{
			ID:         "gen_" + time.Now().Format("150405"),
			Name:       name,
			PrimaryBO:  req.PrimaryBO,
			LayoutType: "detail",
			Sections: []LayoutSection{
				{
					ID:          "sec_basic",
					Title:       "Basic Information",
					Type:        "fields",
					Columns:     cols,
					Collapsible: false,
					FieldIDs:    []string{"f1", "f2", "f3", "f4"},
				},
				{
					ID:             "sec_related",
					Title:          "Related Records",
					Type:           "related_list",
					Columns:        1,
					Collapsible:    true,
					RelationshipID: "rel1",
					RelatedBO:      "Order",
					ColumnFieldIDs: []string{"o1", "o3", "o4"},
				},
			},
		}

		alts := []PageLayout{
			{
				ID:         "altA",
				Name:       name + " A",
				PrimaryBO:  req.PrimaryBO,
				LayoutType: "detail",
				Sections: []LayoutSection{
					{
						ID:          "secA",
						Title:       "Profile",
						Type:        "fields",
						Columns:     2,
						Collapsible: false,
						FieldIDs:    []string{"f1", "f2", "f8"},
					},
				},
			},
			{
				ID:         "altB",
				Name:       name + " B",
				PrimaryBO:  req.PrimaryBO,
				LayoutType: "detail",
				Sections: []LayoutSection{
					{
						ID:          "secB",
						Title:       "Overview",
						Type:        "fields",
						Columns:     1,
						Collapsible: true,
						FieldIDs:    []string{"f1", "f3", "f4", "f9"},
					},
				},
			},
		}

		explanation := "Matched prompt keywords and common patterns for this BO across tenants."

		// Persist to ai_layouts with adopted=false
		draftID, err := persistDraft(tenantID, req.PrimaryBO, layout.Name, layout.LayoutType, "rulebased-v1", 0.87, layout, alts, explanation)
		if err != nil {
			log.Printf("error persisting draft: %v", err)
			http.Error(w, fmt.Sprintf("error persisting draft: %v", err), 500)
			return
		}

		resp := GenLayoutResp{
			Generated:    layout,
			Confidence:   0.87,
			Alternatives: alts,
			Explanation:  explanation,
			ModelVersion: "rulebased-v1",
			GeneratedAt:  time.Now(),
			DraftID:      draftID,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	// POST /api/ai/field-recommendations
	mux.HandleFunc("/api/ai/field-recommendations", func(w http.ResponseWriter, r *http.Request) {
		tenantID := extractTenant(r)
		if tenantID == "" {
			http.Error(w, "missing X-Tenant-ID header", 400)
			return
		}

		var req FieldRecReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		existing := make(map[string]bool)
		for _, id := range req.ExistingFieldIDs {
			existing[id] = true
		}

		recs := make([]FieldRecommendation, 0, len(mockFieldUsage))
		for id, score := range mockFieldUsage {
			if existing[id] {
				continue
			}
			recs = append(recs, FieldRecommendation{
				FieldID:    id,
				FieldLabel: "Field " + strings.ToUpper(id),
				UsageScore: math.Round(score*1000) / 1000,
				Reason:     "High engagement across similar layouts.",
			})
		}

		// Sort by score descending
		for i := 0; i < len(recs); i++ {
			for j := i + 1; j < len(recs); j++ {
				if recs[j].UsageScore > recs[i].UsageScore {
					recs[i], recs[j] = recs[j], recs[i]
				}
			}
		}

		resp := FieldRecResp{Recommendations: recs, GeneratedAt: time.Now()}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	// POST /api/ai/mark-adopted (called after user publishes layout)
	mux.HandleFunc("/api/ai/mark-adopted", func(w http.ResponseWriter, r *http.Request) {
		tenantID := extractTenant(r)
		if tenantID == "" {
			http.Error(w, "missing X-Tenant-ID header", 400)
			return
		}

		var payload struct {
			DraftID string `json:"draftId"`
			UserID  string `json:"userId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		if err := markAdopted(payload.DraftID, payload.UserID); err != nil {
			http.Error(w, fmt.Sprintf("error marking adopted: %v", err), 500)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(204)
	})

	// GET /api/ai/layouts?primary_bo=X (list unadopted drafts for this tenant/BO)
	mux.HandleFunc("/api/ai/layouts", func(w http.ResponseWriter, r *http.Request) {
		tenantID := extractTenant(r)
		if tenantID == "" {
			http.Error(w, "missing X-Tenant-ID header", 400)
			return
		}

		primaryBO := r.URL.Query().Get("primary_bo")

		rows, err := db.Query(`
			SELECT id, name, layout_type, confidence, explanation, created_at, adopted
			FROM ai_layouts
			WHERE tenant_id = $1 AND primary_bo = $2 AND is_active = TRUE
			ORDER BY created_at DESC LIMIT 50
		`, tenantID, primaryBO)
		if err != nil {
			http.Error(w, fmt.Sprintf("error querying drafts: %v", err), 500)
			return
		}
		defer rows.Close()

		type DraftSummary struct {
			ID          string    `json:"id"`
			Name        string    `json:"name"`
			LayoutType  string    `json:"layoutType"`
			Confidence  float64   `json:"confidence"`
			Explanation string    `json:"explanation"`
			CreatedAt   time.Time `json:"createdAt"`
			Adopted     bool      `json:"adopted"`
		}

		var drafts []DraftSummary
		for rows.Next() {
			var d DraftSummary
			if err := rows.Scan(&d.ID, &d.Name, &d.LayoutType, &d.Confidence, &d.Explanation, &d.CreatedAt, &d.Adopted); err != nil {
				log.Printf("error scanning draft: %v", err)
				continue
			}
			drafts = append(drafts, d)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"drafts": drafts})
	})

	addr := ":8088"
	if v := os.Getenv("API_ADDR"); v != "" {
		addr = v
	}
	log.Printf("AI service listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
