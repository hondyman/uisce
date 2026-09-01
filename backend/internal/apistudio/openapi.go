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

// openAPISchema is a minimal OpenAPI 3.0 Schema Object.
type openAPISchema struct {
	Type       string                    `json:"type,omitempty"`
	Items      *openAPISchema            `json:"items,omitempty"`
	Properties map[string]*openAPISchema `json:"properties,omitempty"`
}

type openAPIParameter struct {
	Name     string         `json:"name"`
	In       string         `json:"in"` // "query"
	Required bool           `json:"required"`
	Schema   *openAPISchema `json:"schema"`
}

type openAPIMediaType struct {
	Schema *openAPISchema `json:"schema"`
}

type openAPIResponse struct {
	Description string                      `json:"description"`
	Content     map[string]openAPIMediaType `json:"content,omitempty"`
}

type openAPIOperation struct {
	OperationID string                     `json:"operationId"`
	Summary     string                     `json:"summary"`
	Description string                     `json:"description"`
	Deprecated  bool                       `json:"deprecated,omitempty"`
	Tags        []string                   `json:"tags,omitempty"`
	Parameters  []openAPIParameter         `json:"parameters,omitempty"`
	Responses   map[string]openAPIResponse `json:"responses"`
	Security    []map[string][]string      `json:"security,omitempty"`
}

type openAPIPathItem struct {
	Get  *openAPIOperation `json:"get,omitempty"`
	Post *openAPIOperation `json:"post,omitempty"`
}

type openAPIServer struct {
	Description string `json:"description,omitempty"`
	URL         string `json:"url"`
}

type openAPISecurityScheme struct {
	Type         string `json:"type"`
	Scheme       string `json:"scheme"`
	BearerFormat string `json:"bearerFormat,omitempty"`
}

type openAPIComponents struct {
	SecuritySchemes map[string]openAPISecurityScheme `json:"securitySchemes"`
}

type openAPIInfo struct {
	Title       string `json:"title"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
}

type openAPISpec struct {
	OpenAPI    string                     `json:"openapi"`
	Info       openAPIInfo                `json:"info"`
	Servers    []openAPIServer            `json:"servers,omitempty"`
	Paths      map[string]openAPIPathItem `json:"paths"`
	Components openAPIComponents          `json:"components"`
}

// GenerateOpenAPI produces a schema-complete OpenAPI 3.0 specification for a
// set of endpoints — request query parameters (from Filters), response
// object schemas (from Fields), the bearer-auth security requirement, and
// deprecation status — so it can drive a real client generator
// (openapi-generator, oapi-codegen) rather than being informational-only.
//
// Field/filter *types* are not modeled: APIEndpoint stores field and filter
// names as opaque string lists (see types.go), not typed column metadata, so
// every property is emitted as "string". Response objects are therefore
// permissive rather than precisely typed — good enough to generate a client
// method with the right shape, not to catch a caller passing the wrong
// primitive type.
func GenerateOpenAPI(env, tenantID string, endpoints []APIEndpoint) (string, error) {
	spec := openAPISpec{
		OpenAPI: "3.0.0",
		Info: openAPIInfo{
			Title:       fmt.Sprintf("Semantic API - %s", tenantID),
			Version:     "1.0.0",
			Description: fmt.Sprintf("Governed REST API over Business Objects, env=%s", env),
		},
		Servers: []openAPIServer{
			{URL: APIRuntimeMountPrefixPlaceholder, Description: "API Studio dynamic runtime"},
		},
		Paths: make(map[string]openAPIPathItem),
		Components: openAPIComponents{
			SecuritySchemes: map[string]openAPISecurityScheme{
				"bearerAuth": {Type: "http", Scheme: "bearer", BearerFormat: "JWT"},
			},
		},
	}

	security := []map[string][]string{{"bearerAuth": {}}}

	for _, ep := range endpoints {
		if ep.Type != "rest" {
			continue
		}

		fields := decodeStringList(ep.Fields)
		filterNames := decodeFilterNames(ep.Filters)

		props := make(map[string]*openAPISchema, len(fields))
		for _, f := range fields {
			props[f] = &openAPISchema{Type: "string"}
		}

		op := &openAPIOperation{
			OperationID: operationID(ep),
			Summary:     ep.Name,
			Description: fmt.Sprintf("Governed API over Business Object: %s (semantic_version=%s)", ep.BOName, ep.SemanticVersion),
			Deprecated:  ep.Status == "deprecated" || ep.Status == "retired",
			Tags:        []string{ep.BOName},
			Security:    security,
			Responses: map[string]openAPIResponse{
				"200": {
					Description: "Successful operation",
					Content: map[string]openAPIMediaType{
						"application/json": {
							Schema: &openAPISchema{
								Type:  "array",
								Items: &openAPISchema{Type: "object", Properties: props},
							},
						},
					},
				},
				"403": {Description: "Caller does not satisfy the Business Object's entitlement policy"},
				"429": {Description: "Rate limit exceeded"},
			},
		}

		if strings.ToUpper(ep.Method) == "GET" {
			for _, fname := range filterNames {
				op.Parameters = append(op.Parameters, openAPIParameter{
					Name:   fname,
					In:     "query",
					Schema: &openAPISchema{Type: "string"},
				})
			}
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

// APIRuntimeMountPrefixPlaceholder mirrors api.APIRuntimeMountPrefix
// ("/api/runtime") without importing the api package (apistudio is a
// dependency of api, not the reverse — importing it back would cycle).
// Keep in sync with internal/api/apistudio_handler.go's APIRuntimeMountPrefix.
const APIRuntimeMountPrefixPlaceholder = "/api/runtime"

func operationID(ep APIEndpoint) string {
	name := strings.ReplaceAll(ep.Name, " ", "")
	return strings.ToLower(ep.Method) + name
}
