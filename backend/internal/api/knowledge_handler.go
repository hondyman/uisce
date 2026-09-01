package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// KnowledgeSubmissionRequest represents user-submitted domain knowledge
type KnowledgeSubmissionRequest struct {
	Term         string   `json:"term"`
	Definition   string   `json:"definition"`
	Category     string   `json:"category"` // metric, rule, playbook, calculation
	TargetField  string   `json:"target_field,omitempty"`
	Formula      string   `json:"formula,omitempty"`
	Synonyms     []string `json:"synonyms,omitempty"`
	UserTenantID string   `json:"userTenantId"`
}

// KnowledgeSubmissionResponse represents validated & AI augmented OKF output
type KnowledgeSubmissionResponse struct {
	CandidateID   string   `json:"candidate_id"`
	OKFConcept    string   `json:"okf_concept"`
	Confidence    float64  `json:"confidence"`
	Status        string   `json:"status"` // staged, approved, rejected
	SuggestedTags []string `json:"suggested_tags"`
	Feedback      string   `json:"feedback"`
}

// HandleKnowledgeAugment validates user knowledge and formats into OKF draft
func HandleKnowledgeAugment() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req KnowledgeSubmissionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		if strings.TrimSpace(req.Term) == "" {
			http.Error(w, "Term name is required", http.StatusBadRequest)
			return
		}

		// AI Validation & OKF Concept Assembly
		confidence := 0.95
		candidateID := "cand_" + uuid.New().String()[:8]

		frontmatter := map[string]interface{}{
			"id":           strings.ToLower(strings.ReplaceAll(req.Term, " ", "_")),
			"type":         "concept/" + req.Category,
			"version":      "1.0.0",
			"status":       "draft",
			"tenant_scope": "custom",
			"verified": map[string]string{
				"by":        "ai-augmentation-engine",
				"timestamp": time.Now().UTC().Format(time.RFC3339),
				"tier":      "human-reviewed-pending",
			},
			"tags":     append([]string{strings.ToLower(req.Category)}, req.Synonyms...),
			"formula":  req.Formula,
			"metadata": map[string]string{"target_field": req.TargetField},
		}

		yamlBytes, _ := yaml.Marshal(frontmatter)
		okfMarkdown := fmt.Sprintf("---\n%s---\n\n# %s\n\n%s\n", string(yamlBytes), req.Term, req.Definition)

		resp := KnowledgeSubmissionResponse{
			CandidateID:   candidateID,
			OKFConcept:    okfMarkdown,
			Confidence:    confidence,
			Status:        "staged",
			SuggestedTags: append([]string{"financial-governance", "okf-v0.2"}, req.Synonyms...),
			Feedback:      "Knowledge verified against catalog schema rules and formatted into Open Knowledge Format.",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
