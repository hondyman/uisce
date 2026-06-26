package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/altinvest"
	"github.com/hondyman/semlayer/backend/internal/billing"
	"github.com/hondyman/semlayer/backend/internal/household"
	"github.com/hondyman/semlayer/backend/internal/succession"
)

func registerFinancialRoutes(r chi.Router, srv *Server) {
	// Household Complexity Routes
	r.Route("/households", func(r chi.Router) {
		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				Name string `json:"name"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			h, err := srv.HouseholdService.CreateHousehold(r.Context(), req.Name)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(h)
		})
		r.Get("/{id}/entities", func(w http.ResponseWriter, r *http.Request) {
			idStr := chi.URLParam(r, "id")
			id, err := uuid.Parse(idStr)
			if err != nil {
				http.Error(w, "invalid household id", http.StatusBadRequest)
				return
			}
			hierarchy, err := srv.HouseholdService.GetHouseholdHierarchy(r.Context(), id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(hierarchy)
		})
		r.Post("/{id}/entities", func(w http.ResponseWriter, r *http.Request) {
			idStr := chi.URLParam(r, "id")
			id, err := uuid.Parse(idStr)
			if err != nil {
				http.Error(w, "invalid household id", http.StatusBadRequest)
				return
			}
			var entity household.Entity
			if err := json.NewDecoder(r.Body).Decode(&entity); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			entity.HouseholdID = id
			if err := srv.HouseholdService.CreateEntity(r.Context(), &entity); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusCreated)
		})
	})

	// Alternative Investment Routes
	r.Route("/altinvest", func(r chi.Router) {
		r.Get("/client/{clientId}", func(w http.ResponseWriter, r *http.Request) {
			idStr := chi.URLParam(r, "clientId")
			id, err := uuid.Parse(idStr)
			if err != nil {
				http.Error(w, "invalid client id", http.StatusBadRequest)
				return
			}
			investments, err := srv.AltInvestService.GetClientInvestments(r.Context(), id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(investments)
		})
		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			var inv altinvest.AlternativeInvestment
			if err := json.NewDecoder(r.Body).Decode(&inv); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := srv.AltInvestService.CreateInvestment(r.Context(), &inv); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusCreated)
		})
		r.Post("/capital-call", func(w http.ResponseWriter, r *http.Request) {
			var call altinvest.CapitalCall
			if err := json.NewDecoder(r.Body).Decode(&call); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := srv.AltInvestService.RecordCapitalCall(r.Context(), &call); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusCreated)
		})
	})

	// Billing Routes
	r.Route("/billing", func(r chi.Router) {
		r.Post("/schedules", func(w http.ResponseWriter, r *http.Request) {
			var sched billing.FeeSchedule
			if err := json.NewDecoder(r.Body).Decode(&sched); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := srv.BillingService.CreateFeeSchedule(r.Context(), &sched); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusCreated)
		})
		r.Post("/calculate", func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				ClientID uuid.UUID `json:"client_id"`
				Start    time.Time `json:"start"`
				End      time.Time `json:"end"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			calc, err := srv.BillingService.CalculateClientFee(r.Context(), req.ClientID, req.Start, req.End)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(calc)
		})
	})

	// Tax Planning Routes
	r.Route("/taxplan", func(r chi.Router) {
		r.Get("/opportunities/{clientId}", func(w http.ResponseWriter, r *http.Request) {
			idStr := chi.URLParam(r, "clientId")
			id, err := uuid.Parse(idStr)
			if err != nil {
				http.Error(w, "invalid client id", http.StatusBadRequest)
				return
			}
			opps, err := srv.TaxPlanService.GetClientOpportunities(r.Context(), id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(opps)
		})
		r.Post("/detect/{clientId}", func(w http.ResponseWriter, r *http.Request) {
			idStr := chi.URLParam(r, "clientId")
			id, err := uuid.Parse(idStr)
			if err != nil {
				http.Error(w, "invalid client id", http.StatusBadRequest)
				return
			}
			newOpps, err := srv.TaxPlanService.DetectOpportunities(r.Context(), id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(newOpps)
		})
	})

	// Succession Planning Routes
	r.Route("/succession", func(r chi.Router) {
		r.Get("/metrics/{advisorId}", func(w http.ResponseWriter, r *http.Request) {
			idStr := chi.URLParam(r, "advisorId")
			id, err := uuid.Parse(idStr)
			if err != nil {
				http.Error(w, "invalid advisor id", http.StatusBadRequest)
				return
			}
			metrics, err := srv.SuccessionService.CalculatePracticeMetrics(r.Context(), id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(metrics)
		})
		r.Get("/recommend/{advisorId}", func(w http.ResponseWriter, r *http.Request) {
			idStr := chi.URLParam(r, "advisorId")
			id, err := uuid.Parse(idStr)
			if err != nil {
				http.Error(w, "invalid advisor id", http.StatusBadRequest)
				return
			}
			recommendations, err := srv.SuccessionService.RecommendSuccessor(r.Context(), id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(recommendations)
		})
		r.Post("/plans", func(w http.ResponseWriter, r *http.Request) {
			var plan succession.SuccessionPlan
			if err := json.NewDecoder(r.Body).Decode(&plan); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := srv.SuccessionService.CreateSuccessionPlan(r.Context(), &plan); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusCreated)
		})
	})
}
