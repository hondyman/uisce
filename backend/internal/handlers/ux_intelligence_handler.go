package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/ux_intelligence/accessibility"
	"github.com/hondyman/semlayer/backend/internal/ux_intelligence/copywriting"
	"github.com/hondyman/semlayer/backend/internal/ux_intelligence/disability"
	"github.com/hondyman/semlayer/backend/internal/ux_intelligence/i18n"
	"github.com/hondyman/semlayer/backend/internal/ux_intelligence/themes"
	"github.com/hondyman/semlayer/backend/internal/ux_intelligence/variants"
	workflowpages "github.com/hondyman/semlayer/backend/internal/ux_intelligence/workflow_pages"
)

type UXIntelligenceHandler struct {
	copyGenerator      *copywriting.CopyGenerator
	accessibilityFixer *accessibility.AccessibilityFixer
	themeGenerator     *themes.ThemeGenerator
	variantSuggester   *variants.VariantSuggester
	workflowPageGen    *workflowpages.WorkflowPageGenerator
	translationMemory  *i18n.TranslationMemory
	localeManager      *i18n.LocaleManager
	disabilityManager  *disability.DisabilityManager
}

func NewUXIntelligenceHandler(
	copy *copywriting.CopyGenerator,
	access *accessibility.AccessibilityFixer,
	theme *themes.ThemeGenerator,
	variant *variants.VariantSuggester,
	workflow *workflowpages.WorkflowPageGenerator,
	trans *i18n.TranslationMemory,
	locale *i18n.LocaleManager,
	disab *disability.DisabilityManager,
) *UXIntelligenceHandler {
	return &UXIntelligenceHandler{
		copyGenerator:      copy,
		accessibilityFixer: access,
		themeGenerator:     theme,
		variantSuggester:   variant,
		workflowPageGen:    workflow,
		translationMemory:  trans,
		localeManager:      locale,
		disabilityManager:  disab,
	}
}

func (h *UXIntelligenceHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Copywriting
	r.Post("/copywriting/generate", h.GenerateCopy)

	// Accessibility
	r.Post("/accessibility/analyze/{pageId}", h.AnalyzeAccessibility)
	r.Post("/accessibility/fix", h.GenerateAccessibilityFixes)

	// Themes
	r.Post("/themes/generate", h.GenerateTheme)

	// Variants
	r.Post("/variants/suggest", h.SuggestVariants)

	// Workflow Pages
	r.Post("/workflow-pages/generate", h.GenerateWorkflowPages)

	// I18N
	r.Get("/i18n/translations/{locale}", h.GetTranslations)
	r.Post("/i18n/translations", h.AddTranslation)

	// Disability
	r.Get("/disability/profile/{componentId}", h.GetAccessibilityProfile)
	r.Get("/disability/slo/{pageId}", h.GetAccessibilitySLO)

	return r
}

func (h *UXIntelligenceHandler) GenerateCopy(w http.ResponseWriter, r *http.Request) {
	var req copywriting.CopyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	copy, _ := h.copyGenerator.Generate(r.Context(), &req)
	json.NewEncoder(w).Encode(copy)
}

func (h *UXIntelligenceHandler) AnalyzeAccessibility(w http.ResponseWriter, r *http.Request) {
	pageID, _ := uuid.Parse(chi.URLParam(r, "pageId"))
	report, _ := h.accessibilityFixer.Analyze(r.Context(), pageID)
	json.NewEncoder(w).Encode(report)
}

func (h *UXIntelligenceHandler) GenerateAccessibilityFixes(w http.ResponseWriter, r *http.Request) {
	var violations []accessibility.AccessibilityViolation
	if err := json.NewDecoder(r.Body).Decode(&violations); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	fixes, _ := h.accessibilityFixer.GenerateFixes(r.Context(), violations)
	json.NewEncoder(w).Encode(fixes)
}

func (h *UXIntelligenceHandler) GenerateTheme(w http.ResponseWriter, r *http.Request) {
	var req themes.ThemeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	theme, _ := h.themeGenerator.Generate(r.Context(), &req)
	json.NewEncoder(w).Encode(theme)
}

func (h *UXIntelligenceHandler) SuggestVariants(w http.ResponseWriter, r *http.Request) {
	var req variants.VariantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	suggestions, _ := h.variantSuggester.Suggest(r.Context(), &req)
	json.NewEncoder(w).Encode(suggestions)
}

func (h *UXIntelligenceHandler) GenerateWorkflowPages(w http.ResponseWriter, r *http.Request) {
	var workflow workflowpages.WorkflowDefinition
	if err := json.NewDecoder(r.Body).Decode(&workflow); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	pages, _ := h.workflowPageGen.Generate(r.Context(), &workflow)
	json.NewEncoder(w).Encode(pages)
}

func (h *UXIntelligenceHandler) GetTranslations(w http.ResponseWriter, r *http.Request) {
	locale := chi.URLParam(r, "locale")
	translations, _ := h.translationMemory.GetTranslations(r.Context(), locale)
	json.NewEncoder(w).Encode(translations)
}

func (h *UXIntelligenceHandler) AddTranslation(w http.ResponseWriter, r *http.Request) {
	var translation i18n.Translation
	if err := json.NewDecoder(r.Body).Decode(&translation); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	h.translationMemory.AddTranslation(r.Context(), &translation)
	w.WriteHeader(http.StatusCreated)
}

func (h *UXIntelligenceHandler) GetAccessibilityProfile(w http.ResponseWriter, r *http.Request) {
	componentID, _ := uuid.Parse(chi.URLParam(r, "componentId"))
	profile, _ := h.disabilityManager.GetProfile(r.Context(), componentID)
	json.NewEncoder(w).Encode(profile)
}

func (h *UXIntelligenceHandler) GetAccessibilitySLO(w http.ResponseWriter, r *http.Request) {
	pageID, _ := uuid.Parse(chi.URLParam(r, "pageId"))
	slo, _ := h.disabilityManager.GetSLO(r.Context(), pageID)
	json.NewEncoder(w).Encode(slo)
}
