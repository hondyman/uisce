package forms

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// Mock BusinessObject struct as we don't have the full semlayer/bo package imported here effectively
// In a real app, this would import from internal/semantic/bo
type BOField struct {
	Name string `json:"name"`
	Type string `json:"type"` // string, number, reference
	Ref  string `json:"ref,omitempty"`
	Min  int    `json:"min,omitempty"`
}

type BusinessObject struct {
	ID     uuid.UUID `json:"id"`
	Name   string    `json:"name"`
	Fields []BOField `json:"fields"`
}

type FormConfig struct {
	Title     string                   `json:"title"`
	Fields    []map[string]interface{} `json:"fields"`
	SubmitAPI string                   `json:"submit_api"`
}

type FormGenerator struct{}

func NewFormGenerator() *FormGenerator {
	return &FormGenerator{}
}

func (g *FormGenerator) Generate(ctx context.Context, bo BusinessObject) (*FormConfig, error) {
	config := &FormConfig{
		Title:     fmt.Sprintf("Create %s", bo.Name),
		Fields:    make([]map[string]interface{}, 0),
		SubmitAPI: fmt.Sprintf("/api/bo/%s", bo.Name),
	}

	for _, field := range bo.Fields {
		formField := map[string]interface{}{
			"name":  field.Name,
			"label": field.Name, // Capitalize in real impl
		}

		switch field.Type {
		case "string":
			formField["component"] = "TextInput"
		case "number":
			formField["component"] = "NumberInput"
			if field.Min > 0 {
				formField["min"] = field.Min
			}
		case "reference":
			formField["component"] = "Select"
			formField["dataSource"] = fmt.Sprintf("/api/bo/%s", field.Ref)
		default:
			formField["component"] = "TextInput"
		}

		config.Fields = append(config.Fields, formField)
	}

	return config, nil
}
