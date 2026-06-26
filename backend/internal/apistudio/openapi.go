package apistudio

import (
	"encoding/json"
	"fmt"
	"strings"
)

// OpenAPIResult represents a generated OpenAPI 3.0 spec snippet
type OpenAPIResult struct {
	Spec string `json:"spec"`
}

// GenerateOpenAPI produces an OpenAPI 3.0 JSON specification for a set of endpoints
func GenerateOpenAPI(env, tenantID string, endpoints []APIEndpoint) (string, error) {
	// Simple OpenAPI structure
	type Info struct {
		Title   string `json:"title"`
		Version string `json:"version"`
	}
	type Response struct {
		Description string `json:"description"`
	}
	type Operation struct {
		Summary     string              `json:"summary"`
		Description string              `json:"description"`
		Responses   map[string]Response `json:"responses"`
		Parameters  []interface{}       `json:"parameters,omitempty"`
	}
	type PathItem struct {
		Get  *Operation `json:"get,omitempty"`
		Post *Operation `json:"post,omitempty"`
	}
	type Spec struct {
		OpenAPI string              `json:"openapi"`
		Info    Info                `json:"info"`
		Paths   map[string]PathItem `json:"paths"`
	}

	spec := Spec{
		OpenAPI: "3.0.0",
		Info: Info{
			Title:   fmt.Sprintf("Semantic API - %s", tenantID),
			Version: "1.0.0",
		},
		Paths: make(map[string]PathItem),
	}

	for _, ep := range endpoints {
		op := &Operation{
			Summary:     ep.Name,
			Description: fmt.Sprintf("Governed API over Business Object: %s", ep.BOName),
			Responses: map[string]Response{
				"200": {Description: "Successful operation"},
			},
		}

		item := spec.Paths[ep.Path]
		switch strings.ToUpper(ep.Method) {
		case "GET":
			item.Get = op
		case "POST":
			item.Post = op
		}
		spec.Paths[ep.Path] = item
	}

	b, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
