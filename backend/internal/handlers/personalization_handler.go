package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/hondyman/uisce/backend/internal/ai"
	"github.com/hondyman/uisce/backend/internal/middleware"
	"github.com/hondyman/uisce/backend/internal/personalization"
	"github.com/hondyman/uisce/backend/internal/security"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
)

type PersonalizationHandler struct {
	svc            *personalization.Service
	pageCopilotSvc *ai.PageCopilotService
}

func NewPersonalizationHandler(svc *personalization.Service, pageCopilotSvc *ai.PageCopilotService) *PersonalizationHandler {
	return &PersonalizationHandler{svc: svc, pageCopilotSvc: pageCopilotSvc}
}

func (h *PersonalizationHandler) RegisterRoutes(r chi.Router) {
	r.Route("/personalization", func(r chi.Router) {
		r.Get("/profile", h.GetProfile)
		r.Put("/profile", h.PutProfile)
		r.Post("/pin/{boKey}", h.PinBO)
		r.Delete("/pin/{boKey}", h.UnpinBO)
		r.Post("/bump/{boKey}", h.BumpBO)
		r.Get("/context", h.GetCanonicalContext)
	})
	r.Route("/ai", func(r chi.Router) {
		r.Post("/page-copilot/layout", h.GenerateLayout)
	})
}

func (h *PersonalizationHandler) RegisterMuxRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/personalization/profile", h.GetProfile).Methods("GET", "PUT")
	r.HandleFunc("/api/v1/personalization/pin/{boKey}", h.pinBOMux).Methods("POST")
	r.HandleFunc("/api/v1/personalization/pin/{boKey}", h.unpinBOMux).Methods("DELETE")
	r.HandleFunc("/api/v1/personalization/bump/{boKey}", h.bumpBOMux).Methods("POST")
	r.HandleFunc("/api/v1/personalization/context", h.GetCanonicalContext).Methods("GET")
	r.HandleFunc("/api/v1/ai/page-copilot/layout", h.GenerateLayout).Methods("POST")
}

func (h *PersonalizationHandler) pinBOMux(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, userID, _ := h.resolveIdentityFromRequest(r)
	boKey := mux.Vars(r)["boKey"]
	if boKey == "" {
		http.Error(w, "boKey is required", http.StatusBadRequest)
		return
	}
	if err := h.svc.PinBO(ctx, tenantID, userID, boKey); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *PersonalizationHandler) unpinBOMux(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, userID, _ := h.resolveIdentityFromRequest(r)
	boKey := mux.Vars(r)["boKey"]
	if err := h.svc.UnpinBO(ctx, tenantID, userID, boKey); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *PersonalizationHandler) bumpBOMux(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, userID, _ := h.resolveIdentityFromRequest(r)
	boKey := mux.Vars(r)["boKey"]
	if err := h.svc.BumpBOFrequency(ctx, tenantID, userID, boKey); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *PersonalizationHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, userID, err := h.resolveIdentityFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	profile, err := h.svc.GetProfile(ctx, tenantID, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if profile == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

func (h *PersonalizationHandler) PutProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, userID, err := h.resolveIdentityFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var p personalization.Profile
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	p.TenantID, _ = uuid.Parse(tenantID)
	p.UserHash = personalization.ComputeUserHash(userID, tenantID)
	if p.ProfileID == uuid.Nil {
		p.ProfileID = uuid.New()
	}

	if err := h.svc.UpsertProfile(ctx, &p); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}

func (h *PersonalizationHandler) PinBO(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, userID, err := h.resolveIdentityFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	boKey := chi.URLParam(r, "boKey")
	if boKey == "" {
		http.Error(w, "boKey is required", http.StatusBadRequest)
		return
	}

	if err := h.svc.PinBO(ctx, tenantID, userID, boKey); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *PersonalizationHandler) UnpinBO(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, userID, err := h.resolveIdentityFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	boKey := chi.URLParam(r, "boKey")
	if err := h.svc.UnpinBO(ctx, tenantID, userID, boKey); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *PersonalizationHandler) BumpBO(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, userID, err := h.resolveIdentityFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	boKey := chi.URLParam(r, "boKey")
	if err := h.svc.BumpBOFrequency(ctx, tenantID, userID, boKey); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *PersonalizationHandler) GetCanonicalContext(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, userID, err := h.resolveIdentityFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	identity := middleware.ResolveIdentityContext(ctx)

	cctx, err := h.svc.ResolveCanonicalContext(
		ctx, tenantID, userID,
		identity.FunctionalRole, identity.ClearanceLevel,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cctx)
}

func (h *PersonalizationHandler) GenerateLayout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, _, err := h.resolveIdentityFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var req ai.PageCopilotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	req.TenantID = tenantID

	var uc *ai.UserLayoutContext
	if authInfo, ok := security.AuthInfoFromContext(ctx); ok {
		profile, _ := h.svc.GetProfile(ctx, tenantID, authInfo.UserID)
		pinnedBOKeys := []string{}
		frequentBOKeys := []string{}
		if profile != nil {
			pinnedBOKeys = profile.PinnedBOKeys
			frequentBOKeys = profile.FrequentBOKeys
		}
		uc = &ai.UserLayoutContext{
			FunctionalRole:   authInfo.FunctionalRole,
			ClearanceLevel:  authInfo.ClearanceLevel,
			FrequentBOKeys:  frequentBOKeys,
			PinnedBOKeys:    pinnedBOKeys,
			PreferredDomain:  req.Domain,
		}
	}

	result, err := h.pageCopilotSvc.GeneratePersonalizedLayoutSpec(ctx, req, uc)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *PersonalizationHandler) resolveIdentityFromRequest(r *http.Request) (tenantID, userID string, err error) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		return "", "", nil
	}
	userID = claims.UserID
	tenantID = claims.TenantID
	if userID == "" || tenantID == "" {
		return "", "", nil
	}
	return tenantID, userID, nil
}
