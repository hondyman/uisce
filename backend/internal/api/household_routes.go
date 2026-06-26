package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hondyman/semlayer/backend/internal/reports"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// ============================================================================
// HANDLERS SETUP
// ============================================================================

// RegisterHouseholdRoutes registers all household-related routes
func RegisterHouseholdRoutes(r chi.Router, db *gorm.DB) {
	r.Route("/api/households", func(r chi.Router) {
		r.Get("/", getTenantHouseholds(db))
		r.Post("/", createHousehold(db))
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", getHousehold(db))
			r.Put("/", updateHousehold(db))
			r.Delete("/", deleteHousehold(db))
			r.Get("/members", getHouseholdMembers(db))
			r.Post("/members", addHouseholdMember(db))
			r.Post("/preview-cube", previewSemanticCube(db))
		})
	})

	r.Route("/api/reports/household", func(r chi.Router) {
		r.Post("/", generateHouseholdReport(db))
		r.Get("/", listHouseholdReports(db))
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", getHouseholdReport(db))
			r.Get("/pdf", downloadHouseholdReportPDF(db))
			r.Delete("/", deleteHouseholdReport(db))
		})
	})
}

// ============================================================================
// REQUEST/RESPONSE TYPES
// ============================================================================

type CreateHouseholdRequest struct {
	Name                string `json:"name" binding:"required"`
	Description         string `json:"description"`
	HeadOfHouseholdName string `json:"head_of_household_name"`
	HouseholdType       string `json:"household_type" binding:"required"`
}

type CreateMemberRequest struct {
	MemberType string    `json:"member_type" binding:"required"` // 'alt', 'sma', 'advisor', 'beneficiary'
	MemberID   uuid.UUID `json:"member_id" binding:"required"`
	MemberName string    `json:"member_name"`
	IsPrimary  bool      `json:"is_primary"`
}

type GenerateReportRequest struct {
	HouseholdID    uuid.UUID              `json:"household_id" binding:"required"`
	ReportName     string                 `json:"report_name" binding:"required"`
	ReportType     string                 `json:"report_type" binding:"required"` // 'summary', 'detailed', etc.
	Parameters     map[string]interface{} `json:"parameters"`
	SemanticViewID *uuid.UUID             `json:"semantic_view_id"`
	GenerateNow    bool                   `json:"generate_now" default:"true"`
}

type HouseholdReportResponse struct {
	ID               uuid.UUID              `json:"id"`
	HouseholdID      uuid.UUID              `json:"household_id"`
	ReportName       string                 `json:"report_name"`
	ReportType       string                 `json:"report_type"`
	Status           string                 `json:"status"`
	PageCount        int                    `json:"page_count"`
	GeneratedAt      *time.Time             `json:"generated_at"`
	CreatedAt        time.Time              `json:"created_at"`
	SemanticCubeData map[string]interface{} `json:"semantic_cube_data,omitempty"`
}

// ============================================================================
// HOUSEHOLD HANDLERS
// ============================================================================

// getTenantHouseholds retrieves all households for the authenticated tenant
func getTenantHouseholds(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
		if tenantID == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "tenant scope required"})
			return
		}

		var households []reports.Household
		if err := db.Where("tenant_id = ?", tenantID).Find(&households).Error; err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to fetch households"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"households": households,
			"count":      len(households),
		})
	}
}

// createHousehold creates a new household
func createHousehold(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
		if tenantID == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "tenant scope required"})
			return
		}

		var req CreateHouseholdRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		household := reports.Household{
			ID:                  uuid.New(),
			TenantID:            uuid.MustParse(tenantID),
			Name:                req.Name,
			Description:         req.Description,
			HeadOfHouseholdName: req.HeadOfHouseholdName,
			HouseholdType:       req.HouseholdType,
			Status:              "active",
			IsPublished:         false,
		}

		if err := db.Create(&household).Error; err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to create household"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(household)
	}
}

// getHousehold retrieves a specific household
func getHousehold(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
		householdID := chi.URLParam(r, "id")

		var household reports.Household
		if err := db.Where("id = ? AND tenant_id = ?", householdID, tenantID).First(&household).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]string{"error": "household not found"})
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to fetch household"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(household)
	}
}

// updateHousehold updates household metadata
func updateHousehold(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
		householdID := chi.URLParam(r, "id")

		var req CreateHouseholdRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		if err := db.Model(&reports.Household{}).
			Where("id = ? AND tenant_id = ?", householdID, tenantID).
			Updates(map[string]interface{}{
				"name":                   req.Name,
				"description":            req.Description,
				"head_of_household_name": req.HeadOfHouseholdName,
				"household_type":         req.HouseholdType,
				"updated_at":             time.Now(),
			}).Error; err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to update household"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "household updated successfully"})
	}
}

// deleteHousehold deletes a household and its related data
func deleteHousehold(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
		householdID := chi.URLParam(r, "id")

		// Delete household (cascade will handle related records)
		if err := db.Where("id = ? AND tenant_id = ?", householdID, tenantID).
			Delete(&reports.Household{}).Error; err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to delete household"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "household deleted successfully"})
	}
}

// ============================================================================
// HOUSEHOLD MEMBERS HANDLERS
// ============================================================================

// getHouseholdMembers retrieves all members of a household
func getHouseholdMembers(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
		householdID := chi.URLParam(r, "id")

		var members []reports.HouseholdMember
		if err := db.Where("household_id = ? AND tenant_id = ?", householdID, tenantID).
			Find(&members).Error; err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to fetch members"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"members": members})
	}
}

// addHouseholdMember adds a member to a household
func addHouseholdMember(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
		householdID := chi.URLParam(r, "id")

		var req CreateMemberRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		member := reports.HouseholdMember{
			ID:          uuid.New(),
			HouseholdID: uuid.MustParse(householdID),
			TenantID:    uuid.MustParse(tenantID),
			MemberType:  req.MemberType,
			MemberID:    req.MemberID,
			MemberName:  req.MemberName,
			IsPrimary:   req.IsPrimary,
			IsActive:    true,
		}

		if err := db.Create(&member).Error; err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to add member"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(member)
	}
}

// ============================================================================
// HOUSEHOLD REPORTS HANDLERS
// ============================================================================

// generateHouseholdReport generates a new household report
func generateHouseholdReport(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
		if tenantID == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "tenant scope required"})
			return
		}

		var req GenerateReportRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		// Get household (verify access)
		var household reports.Household
		if err := db.Where("id = ? AND tenant_id = ?", req.HouseholdID, tenantID).
			First(&household).Error; err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{"error": "household not found or access denied"})
			return
		}

		// Create report engine
		engine := reports.NewHouseholdReportEngine(db)

		// Get household data
		_, _, mappings, err := engine.GetHouseholdData(r.Context(), req.HouseholdID, uuid.MustParse(tenantID))
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("failed to get household data: %v", err)})
			return
		}

		if len(mappings) == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "household has no semantic view mappings"})
			return
		}

		// Generate semantic cube from first mapping
		cube, err := engine.GenerateSemanticCube(r.Context(), req.HouseholdID, uuid.MustParse(tenantID), &mappings[0])
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("failed to generate semantic cube: %v", err)})
			return
		}

		// Build report from cube
		pages, err := engine.BuildReportFromCube(r.Context(), cube, req.ReportType)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("failed to build report: %v", err)})
			return
		}

		// Save report to database
		reportConfig, _ := json.Marshal(req.Parameters)
		report, err := engine.SaveReport(
			r.Context(),
			req.HouseholdID,
			uuid.MustParse(tenantID),
			req.ReportName,
			req.ReportType,
			reportConfig,
			cube,
			pages,
		)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("failed to save report: %v", err)})
			return
		}

		// Return structured response
		response := HouseholdReportResponse{
			ID:          report.ID,
			HouseholdID: report.HouseholdID,
			ReportName:  report.ReportName,
			ReportType:  report.ReportType,
			Status:      report.Status,
			PageCount:   report.PageCount,
			GeneratedAt: report.GeneratedAt,
			CreatedAt:   report.CreatedAt,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)
	}
}

// listHouseholdReports lists reports for a household
func listHouseholdReports(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
		householdID := r.URL.Query().Get("household_id")

		if householdID == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "household_id query parameter required"})
			return
		}

		var hsreports []reports.HouseholdReport
		query := db.Where("tenant_id = ? AND household_id = ?", tenantID, householdID).
			Order("created_at DESC")

		if err := query.Find(&hsreports).Error; err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to fetch reports"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"reports": hsreports,
			"count":   len(hsreports),
		})
	}
}

// getHouseholdReport retrieves a specific report
func getHouseholdReport(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
		reportID := chi.URLParam(r, "id")

		engine := reports.NewHouseholdReportEngine(db)
		report, err := engine.GetReport(r.Context(), uuid.MustParse(reportID), uuid.MustParse(tenantID))
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "report not found"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(report)
	}
}

// downloadHouseholdReportPDF downloads the PDF for a report
func downloadHouseholdReportPDF(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
		reportID := chi.URLParam(r, "id")

		engine := reports.NewHouseholdReportEngine(db)
		report, err := engine.GetReport(r.Context(), uuid.MustParse(reportID), uuid.MustParse(tenantID))
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "report not found"})
			return
		}

		if report.PDFData == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "PDF not yet generated"})
			return
		}

		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", report.PDFFileName))
		w.WriteHeader(http.StatusOK)
		w.Write(report.PDFData)
	}
}

// deleteHouseholdReport deletes a report
func deleteHouseholdReport(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
		reportID := chi.URLParam(r, "id")

		if err := db.Where("id = ? AND tenant_id = ?", reportID, tenantID).
			Delete(&reports.HouseholdReport{}).Error; err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to delete report"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "report deleted successfully"})
	}
}

// ============================================================================
// SEMANTIC CUBE PREVIEW
// ============================================================================

// previewSemanticCube generates a preview of the semantic cube for a household
func previewSemanticCube(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
		householdID := chi.URLParam(r, "id")

		engine := reports.NewHouseholdReportEngine(db)

		// Get household data
		_, _, mappings, err := engine.GetHouseholdData(
			r.Context(),
			uuid.MustParse(householdID),
			uuid.MustParse(tenantID),
		)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("failed to get household data: %v", err)})
			return
		}

		if len(mappings) == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "no semantic view mappings found"})
			return
		}

		// Generate semantic cube
		cube, err := engine.GenerateSemanticCube(
			r.Context(),
			uuid.MustParse(householdID),
			uuid.MustParse(tenantID),
			&mappings[0],
		)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("failed to generate cube: %v", err)})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cube)
	}
}

// ============================================================================
// EOF
// ============================================================================
