package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/marketplace"
	"github.com/hondyman/uisce/backend/internal/middleware"
)

// MarketplaceListing is the public API shape served to React.
type MarketplaceListing struct {
	ID                string  `json:"id"`
	Title             string  `json:"title"`
	Kind              string  `json:"kind"`
	Category          string  `json:"category"`
	PublisherTenantID string  `json:"publisher_tenant_id"`
	PublisherName     string  `json:"publisher_name"`
	Description       string  `json:"description"`
	PriceCents        int     `json:"price_cents"`
	BillingPeriod     string  `json:"billing_period"`
	Rating            float64 `json:"rating"`
	InstallsCount     int     `json:"installs_count"`
	ArtifactType      string  `json:"artifact_type,omitempty"`
	ArtifactVersion   string  `json:"artifact_version,omitempty"`
	CreatedAt         string  `json:"created_at"`
}

type PublishListingRequest struct {
	Title              string   `json:"title"`
	Category          string   `json:"category"`
	Description       string   `json:"description"`
	PriceCents        int      `json:"price_cents"`
	BillingPeriod     string   `json:"billing_period"` // one_time | monthly | annual
	RoleKey           string   `json:"role_key"`
	RoleName          string   `json:"role_name"`
	ArtifactType      string   `json:"artifact_type"`    // rbac_role | abac_policy | ...
	ArtifactVersion   string   `json:"artifact_version"`  // semver
	MinPlatformVersion string  `json:"min_platform_version"`
	Permissions       []string `json:"permissions"` // for rbac_role
}

// browseMarketplace returns a paginated, filtered catalog of published listings.
func (h *RBACHandlers) browseMarketplace(w http.ResponseWriter, r *http.Request) {
	_, _, ok := middleware.MarketplaceAuthFromRequest(r)
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	params := marketplace.BrowseParams{
		Kind:     r.URL.Query().Get("kind"),
		Category: r.URL.Query().Get("category"),
		Search:   r.URL.Query().Get("search"),
		Status:   "published",
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			params.Limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			params.Offset = n
		}
	}

	svc := marketplace.NewService(h.db)
	result, err := svc.Browse(r.Context(), params)
	if err != nil {
		http.Error(w, `{"error":"internal_error","message":"`+err.Error()+`"}`,
			http.StatusInternalServerError)
		return
	}

	listings := make([]MarketplaceListing, 0, len(result.Listings))
	for _, l := range result.Listings {
		listings = append(listings, MarketplaceListing{
			ID:                l.ID,
			Title:             l.Title,
			Kind:              l.Kind,
			Category:          l.Category,
			PublisherTenantID: l.PublisherTenantID,
			PublisherName:     l.PublisherName,
			Description:       l.Description,
			PriceCents:        l.PriceCents,
			BillingPeriod:     l.BillingPeriod,
			Rating:            l.Rating,
			InstallsCount:     l.InstallsCount,
			ArtifactType:      l.ArtifactType,
			ArtifactVersion:   l.ArtifactVersion,
			CreatedAt:         l.CreatedAt,
		})
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"listings": listings,
		"total":    result.Total,
		"limit":    result.Limit,
		"offset":   result.Offset,
	})
}

// publishMarketplaceItem creates a listing + artifact from the tenant's existing role/policy.
func (h *RBACHandlers) publishMarketplaceItem(w http.ResponseWriter, r *http.Request) {
	actorID, _, ok := middleware.MarketplaceAuthFromRequest(r)
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	if !h.hasMarketplacePublishScope(r) {
		http.Error(w, `{"error":"forbidden","message":"marketplace:publish permission required"}`,
			http.StatusForbidden)
		return
	}

	raw, err := marketplace.ReadAllBounds(r.Body, marketplace.MaxBodyBytes)
	if err != nil {
		http.Error(w, `{"error":"bad_request","message":"`+err.Error()+`"}`,
			http.StatusBadRequest)
		return
	}

	var req PublishListingRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		http.Error(w, `{"error":"bad_request","message":"invalid JSON"}`,
			http.StatusBadRequest)
		return
	}

	if req.Title == "" || req.Category == "" {
		http.Error(w, `{"error":"bad_request","message":"title and category are required"}`,
			http.StatusBadRequest)
		return
	}
	if req.ArtifactType == "" {
		req.ArtifactType = string(marketplace.ArtifactTypeRBACRole)
	}
	if req.ArtifactVersion == "" {
		req.ArtifactVersion = "1.0.0"
	}

	actorUUID, _ := uuid.Parse(actorID) // validated by middleware
	// TenantID is needed; derive from JWT claims in context.
	tenantCtx, _, _ := middleware.MarketplaceAuthFromRequest(r)
	tenantUUID, err := uuid.Parse(tenantCtx)
	if err != nil {
		http.Error(w, `{"error":"unauthorized","message":"invalid tenant in token"}`,
			http.StatusUnauthorized)
		return
	}

	def := marketplace.ArtifactDef{
		"role_key":    req.RoleKey,
		"role_name":   req.RoleName,
		"description": req.Description,
	}
	if len(req.Permissions) > 0 {
		permAny := make([]any, len(req.Permissions))
		for i, p := range req.Permissions {
			permAny[i] = p
		}
		def["permissions"] = permAny
	}

	artifact := marketplace.Artifact{
		SchemaVersion:       "uisce.mp/v1",
		ArtifactType:        marketplace.ArtifactType(req.ArtifactType),
		ArtifactVersion:     req.ArtifactVersion,
		MinPlatformVersion:  req.MinPlatformVersion,
		Definition:          def,
	}

	billingPeriod := marketplace.BillingPeriod(req.BillingPeriod)
	if billingPeriod == "" {
		billingPeriod = marketplace.PeriodOneTime
	}

	svc := marketplace.NewService(h.db)
	result, idemRec, err := svc.Publish(r.Context(), marketplace.PublishRequest{
		ActorID:        actorUUID,
		TenantID:       tenantUUID,
		IdempotencyKey: r.Header.Get("Idempotency-Key"),
		RequestHash:    sha256Sum(raw),
		Title:          req.Title,
		Kind:           marketplace.KindRBAC,
		Category:       req.Category,
		Description:    req.Description,
		PriceCents:    req.PriceCents,
		BillingPeriod:  billingPeriod,
		Artifact:       artifact,
	})
	if err != nil {
		if idemRec != nil {
			http.Error(w, `{"error":"conflict","message":"Idempotency-Key reused with a different request body"}`,
				http.StatusConflict)
			return
		}
		http.Error(w, `{"error":"internal_error","message":"`+err.Error()+`"}`,
			http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusCreated, map[string]any{
		"status":           "published",
		"listing_id":       result.ListingID,
		"title":            result.Title,
		"artifact_version": result.Version,
		"idempotency_reused": idemRec != nil,
	})
}

// installMarketplaceItem creates a template subscription for the calling tenant.
func (h *RBACHandlers) installMarketplaceItem(w http.ResponseWriter, r *http.Request) {
	actorID, _, ok := middleware.MarketplaceAuthFromRequest(r)
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	listingIDStr := chi.URLParam(r, "listingId")
	listingID, err := uuid.Parse(listingIDStr)
	if err != nil {
		http.Error(w, `{"error":"bad_request","message":"invalid listing_id"}`,
			http.StatusBadRequest)
		return
	}

	actorUUID, _ := uuid.Parse(actorID)
	tenantCtx, _, _ := middleware.MarketplaceAuthFromRequest(r)
	tenantUUID, err := uuid.Parse(tenantCtx)
	if err != nil {
		http.Error(w, `{"error":"unauthorized","message":"invalid tenant in token"}`,
			http.StatusUnauthorized)
		return
	}

	raw, _ := marketplace.ReadAllBounds(r.Body, marketplace.MaxBodyBytes)

	svc := marketplace.NewService(h.db)
	result, idemRec, err := svc.Install(r.Context(), marketplace.InstallRequest{
		ListingID:      listingID,
		ActorID:       actorUUID,
		TenantID:      tenantUUID,
		IdempotencyKey: r.Header.Get("Idempotency-Key"),
		RequestHash:   sha256Sum(raw),
	})
	if err != nil {
		if idemRec != nil {
			http.Error(w, `{"error":"conflict","message":"Idempotency-Key reused with a different request body"}`,
				http.StatusConflict)
			return
		}
		http.Error(w, `{"error":"internal_error","message":"`+err.Error()+`"}`,
			http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"status":            "installed",
		"listing_id":        result.ListingID,
		"artifact_type":     result.ArtifactType,
		"artifact_key":     result.ArtifactKey,
		"artifact_version": result.Version,
		"is_new":           result.IsNew,
		"idempotency_reused": idemRec != nil,
	})
}

// getProductEvolution returns AI-clustered customization recommendations.
func (h *RBACHandlers) getProductEvolution(w http.ResponseWriter, r *http.Request) {
	_, _, ok := middleware.MarketplaceAuthFromRequest(r)
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	minConf := 0.5
	if v := r.URL.Query().Get("min_confidence"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			minConf = f
		}
	}

	svc := marketplace.NewService(h.db)
	recs, err := svc.ProductEvolution(r.Context(), minConf)
	if err != nil {
		http.Error(w, `{"error":"internal_error","message":"`+err.Error()+`"}`,
			http.StatusInternalServerError)
		return
	}
	if recs == nil {
		recs = []marketplace.EvolutionRecommendation{}
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"status":         "evaluated",
		"total_clusters": len(recs),
		"recommendations": recs,
	})
}

// getInstallations returns all marketplace artifacts installed by the calling tenant.
func (h *RBACHandlers) getInstallations(w http.ResponseWriter, r *http.Request) {
	_, tenantCtx, ok := middleware.MarketplaceAuthFromRequest(r)
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	tenantUUID, err := uuid.Parse(tenantCtx)
	if err != nil {
		http.Error(w, `{"error":"unauthorized","message":"invalid tenant"}`,
			http.StatusUnauthorized)
		return
	}

	svc := marketplace.NewService(h.db)
	installations, err := svc.GetInstallations(r.Context(), tenantUUID)
	if err != nil {
		http.Error(w, `{"error":"internal_error","message":"`+err.Error()+`"}`,
			http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"installations": installations,
	})
}

// hasMarketplacePublishScope checks whether the request's AuthInfo grants publish scope.
func (h *RBACHandlers) hasMarketplacePublishScope(r *http.Request) bool {
	return middleware.MarketplaceScopeSatisfied(r.Context(),
		middleware.ScopeMarketplacePublish)
}

func sha256Sum(data []byte) []byte {
	h := sha256.Sum256(data)
	return []byte(hex.EncodeToString(h[:]))
}

// RegisterMarketplaceEcosystemRoutes wires the marketplace HTTP handlers.
// Route group /api/marketplace/* is secured by RequireMarketplaceScope middleware
// applied in registerWorkflowRoutes (api.go).  Publish requires an additional
// per-handler scope check inside publishMarketplaceItem.
func RegisterMarketplaceEcosystemRoutes(r chi.Router, h *RBACHandlers) {
	r.Get("/browse", h.browseMarketplace)
	r.Post("/publish", h.publishMarketplaceItem)
	r.Post("/{listingId}/install", h.installMarketplaceItem)
	r.Get("/product-evolution", h.getProductEvolution)
	r.Get("/installations", h.getInstallations)
}
