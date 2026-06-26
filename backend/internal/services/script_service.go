package services

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/platform"
)

// ScriptService defines the interface for managing scripts.
type ScriptService interface {
	CreateScript(user models.User, name, description, scope string) (*models.ScriptDetail, error)
	GetScript(user models.User, id string) (*models.ScriptDetail, error)
	ListScripts(user models.User, query, state, scope, tag, steward string) ([]*models.ScriptSummary, error)
	AddScriptVersion(user models.User, scriptID, content string) (*models.ScriptVersion, error)
	PublishScript(user models.User, scriptID string) error
	GetImpactReport(user models.User, scriptID, version string) (*models.ImpactReport, error)
}

// NewScriptService creates a new instance of the script service.
func NewScriptService(policyService platform.PolicyService) ScriptService {
	store := &inMemoryScriptStore{
		scripts: make(map[string]*models.ScriptDetail),
	}

	// Initialize with some dummy data
	scriptID := uuid.New().String()
	store.scripts[scriptID] = &models.ScriptDetail{
		ScriptSummary: models.ScriptSummary{
			ID:            scriptID,
			Name:          "calc_effective_duration",
			Description:   "Calculates the effective duration for a bond.",
			DomainTags:    []string{"Fixed Income", "Risk"},
			Scope:         "semantic",
			State:         models.ScriptStatePublished,
			LatestVersion: "1.0.0",
			Steward:       "risk_team",
			UpdatedAt:     time.Now(),
		},
		Versions: []models.ScriptVersion{
			{
				Version:   "1.0.0",
				CreatedAt: time.Now(),
				CreatedBy: "admin",
				Content:   "function calculateDuration(params) { return 5; }",
				Hash:      "abc123def456",
			},
		},
		Lineage: models.ScriptLineage{
			AttachedTo: []string{"measure:duration"},
		},
	}

	return &scriptServiceImpl{
		store:         store,
		policyService: policyService,
	}
}

// inMemoryScriptStore simulates a database for scripts.
type inMemoryScriptStore struct {
	mu      sync.RWMutex
	scripts map[string]*models.ScriptDetail
}

type scriptServiceImpl struct {
	store         *inMemoryScriptStore
	policyService platform.PolicyService
}

func (s *scriptServiceImpl) CreateScript(user models.User, name, description, scope string) (*models.ScriptDetail, error) {
	s.store.mu.Lock()
	defer s.store.mu.Unlock()

	newScript := &models.ScriptDetail{
		ScriptSummary: models.ScriptSummary{
			ID:          uuid.New().String(),
			Name:        name,
			Description: description,
			Scope:       scope,
			State:       models.ScriptStateDraft,
			Steward:     user.ID,
			UpdatedAt:   time.Now(),
		},
	}

	s.store.scripts[newScript.ID] = newScript
	return newScript, nil
}

func (s *scriptServiceImpl) GetScript(user models.User, id string) (*models.ScriptDetail, error) {
	s.store.mu.RLock()
	defer s.store.mu.RUnlock()

	script, exists := s.store.scripts[id]
	if !exists {
		return nil, fmt.Errorf("script with id %s not found", id)
	}
	return script, nil
}

func (s *scriptServiceImpl) ListScripts(user models.User, query, state, scope, tag, steward string) ([]*models.ScriptSummary, error) {
	s.store.mu.RLock()
	defer s.store.mu.RUnlock()

	var summaries []*models.ScriptSummary
	for _, script := range s.store.scripts {
		// Basic filtering (a real implementation would be more robust)
		if query != "" && !strings.Contains(script.Name, query) {
			continue
		}
		if state != "" && string(script.State) != state {
			continue
		}
		if scope != "" && script.Scope != scope {
			continue
		}
		summary := script.ScriptSummary
		summaries = append(summaries, &summary)
	}
	return summaries, nil
}

func (s *scriptServiceImpl) AddScriptVersion(user models.User, scriptID, content string) (*models.ScriptVersion, error) {
	// ... implementation for adding a new version to a script ...
	return nil, fmt.Errorf("not implemented")
}

func (s *scriptServiceImpl) PublishScript(user models.User, scriptID string) error {
	// ... implementation for changing script state to published ...
	return fmt.Errorf("not implemented")
}

func (s *scriptServiceImpl) GetImpactReport(user models.User, scriptID, version string) (*models.ImpactReport, error) {
	// In a real app, this would traverse a dependency graph.
	// For now, return mock data.
	return &models.ImpactReport{
		ScriptID:      scriptID,
		ScriptVersion: version,
		ImpactedBundles: []models.ImpactedBundle{
			{ID: "bundle-1", Name: "Front Office Performance", Version: "1.4.0", State: "Published"},
		},
		ImpactedViews: []models.ImpactedView{
			{Name: "pm_performance_view", BundleID: "bundle-1", BundleName: "Front Office Performance", State: "Published"},
		},
		ImpactedObjects: []models.ImpactedObject{
			{Type: "measure", ID: "duration", BundleID: "bundle-1"},
		},
	}, nil
}
