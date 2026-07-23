package workflowpages

import (
	"context"

	"github.com/google/uuid"
)

type WorkflowStep struct {
	StepID       string   `json:"step_id"`
	StepName     string   `json:"step_name"`
	RequiredData []string `json:"required_data"`
	Actions      []string `json:"actions"`
	Approvers    []string `json:"approvers,omitempty"`
}

type WorkflowDefinition struct {
	WorkflowID   string         `json:"workflow_id"`
	WorkflowName string         `json:"workflow_name"`
	Steps        []WorkflowStep `json:"steps"`
	SLOs         map[string]int `json:"slos"` // step -> max_time_ms
}

type GeneratedPage struct {
	PageID          uuid.UUID                    `json:"page_id"`
	StepID          string                       `json:"step_id"`
	Layout          string                       `json:"layout"`
	Components      []string                     `json:"components"`
	DataBindings    map[string]string            `json:"data_bindings"`
	ValidationRules []string                     `json:"validation_rules"`
	Navigation      map[string]string            `json:"navigation"`
	Accessibility   map[string]string            `json:"accessibility"`
	MultiLingual    map[string]map[string]string `json:"multi_lingual"` // locale -> field -> text
	Tests           []string                     `json:"tests"`
}

type WorkflowPageGenerator struct{}

func NewWorkflowPageGenerator() *WorkflowPageGenerator {
	return &WorkflowPageGenerator{}
}

func (g *WorkflowPageGenerator) Generate(ctx context.Context, workflow *WorkflowDefinition) ([]GeneratedPage, error) {
	pages := make([]GeneratedPage, 0)

	// Mock: Generate pages for each workflow step
	// Real: Analyze workflow, generate complete pages with all metadata

	for _, step := range workflow.Steps {
		page := GeneratedPage{
			PageID:       uuid.New(),
			StepID:       step.StepID,
			Layout:       "form_layout",
			Components:   []string{"form", "submit_button", "cancel_button"},
			DataBindings: make(map[string]string),
			ValidationRules: []string{
				"required_fields",
				"format_validation",
			},
			Navigation: map[string]string{
				"next": "next_step",
				"back": "previous_step",
			},
			Accessibility: map[string]string{
				"form_aria_label":   step.StepName,
				"submit_aria_label": "Submit " + step.StepName,
			},
			MultiLingual: map[string]map[string]string{
				"en": {
					"title":  step.StepName,
					"submit": "Continue",
					"cancel": "Cancel",
				},
				"es": {
					"title":  step.StepName + " (ES)",
					"submit": "Continuar",
					"cancel": "Cancelar",
				},
			},
			Tests: []string{
				"test_form_submission",
				"test_validation",
				"test_navigation",
				"test_accessibility",
			},
		}

		// Add data bindings for required data
		for _, data := range step.RequiredData {
			page.DataBindings[data] = "workflow." + step.StepID + "." + data
		}

		pages = append(pages, page)
	}

	return pages, nil
}
