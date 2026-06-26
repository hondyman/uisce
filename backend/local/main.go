package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/machinebox/graphql"
	"github.com/qri-io/jsonschema"
)

var hasuraURL string
var hasuraAdminSecret string
var jwtSecret string
var compiledSchemas map[string]*jsonschema.Schema

func loadSchemas() error {
	compiledSchemas = map[string]*jsonschema.Schema{}
	files := map[string]string{
		"client_investors": "backend/local/schemas/client_investors.json",
		"portfolios":       "backend/local/schemas/portfolios.json",
	}
	missing := []string{}
	for name, path := range files {
		b, err := os.ReadFile(path)
		if err != nil {
			missing = append(missing, fmt.Sprintf("%s (path: %s): %v", name, path, err))
			continue
		}
		rs := &jsonschema.Schema{}
		if err := json.Unmarshal(b, rs); err != nil {
			missing = append(missing, fmt.Sprintf("%s (parse error): %v", name, err))
			continue
		}
		compiledSchemas[name] = rs
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing or invalid schemas: %v", missing)
	}
	return nil
}

// validateAgainstSchemaStructured validates payload and returns a map of field->message when validation fails.
func validateAgainstSchemaStructured(name string, payload interface{}) (map[string]string, error) {
	if compiledSchemas == nil {
		return nil, nil
	}
	rs, ok := compiledSchemas[name]
	if !ok {
		// no schema configured for this entity
		return nil, nil
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	verrs, err := rs.ValidateBytes(ctx, b)
	if err != nil {
		// Execution error during validation (e.g. schema processing), return it directly.
		return nil, err
	}
	if len(verrs) > 0 {
		// Try to parse error strings into field-level messages.
		// Each KeyError has its own message; join them into lines and attempt to extract a path and message.
		res := map[string]string{}
		lines := []string{}
		for _, ke := range verrs {
			s := strings.TrimSpace(ke.Error())
			if s == "" {
				continue
			}
			if strings.Contains(s, "\n") {
				for _, l := range strings.Split(s, "\n") {
					l = strings.TrimSpace(l)
					if l != "" {
						lines = append(lines, l)
					}
				}
			} else {
				lines = append(lines, s)
			}
		}
		full := strings.Join(lines, "\n")
		// if no lines were produced, fall back to a generic message
		if len(lines) == 0 {
			full = "validation error"
			lines = []string{full}
		}
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			// try split around ':' to separate path from message
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				path := strings.TrimSpace(parts[0])
				msg := strings.TrimSpace(parts[1])
				// cleanup path to simple field name
				// remove leading data. or / or (root)
				path = strings.TrimPrefix(path, "data.")
				path = strings.TrimPrefix(path, "$(root)")
				path = strings.Trim(path, " /\\")
				// take last segment after dot or slash
				if idx := strings.LastIndexAny(path, "./"); idx != -1 {
					path = path[idx+1:]
				}
				if path == "" {
					path = "_error"
				}
				if _, exists := res[path]; !exists {
					res[path] = msg
				}
			} else {
				// fallback to full message
				if _, exists := res["_error"]; !exists {
					res["_error"] = line
				}
			}
		}
		return res, fmt.Errorf("%s", full)
	}
	return nil, nil
}

type Claims struct {
	TenantID string `json:"tenant_id"`
	jwt.RegisteredClaims
}

func tenantAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Allow unauthenticated in dev if header not provided
		tokenStr := c.GetHeader("Authorization")
		if tokenStr == "" {
			c.Next()
			return
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}
		c.Set("tenant_id", claims.TenantID)
		c.Next()
	}
}

func proxyGraphQL(c *gin.Context) {
	client := graphql.NewClient(hasuraURL)
	var body map[string]interface{}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	queryStr, _ := body["query"].(string)
	req := graphql.NewRequest(queryStr)
	if hasuraAdminSecret != "" {
		req.Header.Set("X-Hasura-Admin-Secret", hasuraAdminSecret)
	}
	// propagate tenant header
	if t := c.GetString("tenant_id"); t != "" {
		req.Header.Set("X-Hasura-Tenant-Id", t)
	} else if h := c.GetHeader("X-Hasura-Tenant-Id"); h != "" {
		req.Header.Set("X-Hasura-Tenant-Id", h)
	}
	var resp interface{}
	if err := client.Run(context.Background(), req, &resp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// DynamicRequest for CRUD operations
type DynamicRequest struct {
	EntityType string                 `json:"entity_type"`
	Data       map[string]interface{} `json:"data"`
	IDs        []string               `json:"ids"`
}

// buildMutationVars builds a GraphQL mutation string that uses variables and returns variables map
func buildMutationVars(entityType string, data map[string]interface{}, ids []string, method string) (string, map[string]interface{}, error) {
	vars := make(map[string]interface{})
	switch method {
	case "POST":
		// insert_<entity>_one(object: $object)
		query := fmt.Sprintf(`mutation InsertEntity($object: %s_insert_input!) { insert_%s_one(object: $object) { id } }`, entityType, entityType)
		vars["object"] = data
		return query, vars, nil
	case "PUT":
		// update_<entity>_by_pk(pk_columns: {id: $id}, _set: $changes)
		if data == nil || data["id"] == nil {
			return "", nil, fmt.Errorf("missing id for update")
		}
		query := fmt.Sprintf(`mutation UpdateEntity($id: uuid!, $changes: %s_set_input!) { update_%s_by_pk(pk_columns: {id: $id}, _set: $changes) { id } }`, entityType, entityType)
		vars["id"] = data["id"]
		vars["changes"] = data
		return query, vars, nil
	case "DELETE":
		query := fmt.Sprintf(`mutation DeleteEntities($ids: [uuid!]!) { delete_%s(where: {id: {_in: $ids}}) { affected_rows } }`, entityType)
		vars["ids"] = ids
		return query, vars, nil
	default:
		return "", nil, fmt.Errorf("unsupported method")
	}
}

// sendHasuraRequest executes a GraphQL request with variables against Hasura
func sendHasuraRequest(query string, variables map[string]interface{}, extraHeaders map[string]string) (interface{}, error) {
	client := graphql.NewClient(hasuraURL)
	req := graphql.NewRequest(query)
	if hasuraAdminSecret != "" {
		req.Header.Set("X-Hasura-Admin-Secret", hasuraAdminSecret)
	}
	for k, v := range variables {
		req.Var(k, v)
	}
	for hk, hv := range extraHeaders {
		if hk != "" && hv != "" {
			req.Header.Set(hk, hv)
		}
	}
	var resp interface{}
	if err := client.Run(context.Background(), req, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// dynamicCRUD handles REST CRUD by building variable-based GraphQL mutations
func dynamicCRUD(c *gin.Context) {
	var req DynamicRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID := c.GetString("tenant_id")
	if tenantID != "" {
		if req.Data == nil {
			req.Data = map[string]interface{}{}
		}
		req.Data["tenant_id"] = tenantID
	}

	// Validate data against JSON schema if one exists
	if req.EntityType != "" && req.Data != nil {
		if fieldErrors, err := validateAgainstSchemaStructured(req.EntityType, req.Data); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"validation_errors": fieldErrors, "message": "validation failed"})
			return
		} else if fieldErrors != nil {
			c.JSON(http.StatusBadRequest, gin.H{"validation_errors": fieldErrors, "message": "validation failed"})
			return
		}
	}

	query, vars, err := buildMutationVars(req.EntityType, req.Data, req.IDs, c.Request.Method)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	hdrs := map[string]string{}
	if tenant := c.GetString("tenant_id"); tenant != "" {
		hdrs["X-Hasura-Tenant-Id"] = tenant
	} else if h := c.GetHeader("X-Hasura-Tenant-Id"); h != "" {
		hdrs["X-Hasura-Tenant-Id"] = h
	}
	resp, err := sendHasuraRequest(query, vars, hdrs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// Hasura action payload
type HasuraActionPayload struct {
	Input map[string]interface{} `json:"input"`
}

// actionDynamicInsert handles Hasura action for dynamic_insert
func actionDynamicInsert(c *gin.Context) {
	var payload HasuraActionPayload
	if err := c.BindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	entityType, _ := payload.Input["entity_type"].(string)
	object, _ := payload.Input["object"].(map[string]interface{})
	// attach tenant if present
	tenantID := c.GetHeader("X-Hasura-Tenant-Id")
	if tenantID != "" {
		object["tenant_id"] = tenantID
	}
	// Validate
	if entityType != "" && object != nil {
		if fieldErrors, err := validateAgainstSchemaStructured(entityType, object); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"validation_errors": fieldErrors, "message": "validation failed"})
			return
		} else if fieldErrors != nil {
			c.JSON(http.StatusBadRequest, gin.H{"validation_errors": fieldErrors, "message": "validation failed"})
			return
		}
	}
	query, vars, err := buildMutationVars(entityType, object, nil, "POST")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	hdrs := map[string]string{}
	if t := c.GetHeader("X-Hasura-Tenant-Id"); t != "" {
		hdrs["X-Hasura-Tenant-Id"] = t
	}
	resp, err := sendHasuraRequest(query, vars, hdrs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// actionDynamicUpdate handles Hasura action for dynamic_update
func actionDynamicUpdate(c *gin.Context) {
	var payload HasuraActionPayload
	if err := c.BindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	entityType, _ := payload.Input["entity_type"].(string)
	object, _ := payload.Input["changes"].(map[string]interface{})
	tenantID := c.GetHeader("X-Hasura-Tenant-Id")
	if tenantID != "" {
		object["tenant_id"] = tenantID
	}
	if entityType != "" && object != nil {
		if fieldErrors, err := validateAgainstSchemaStructured(entityType, object); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"validation_errors": fieldErrors, "message": "validation failed"})
			return
		} else if fieldErrors != nil {
			c.JSON(http.StatusBadRequest, gin.H{"validation_errors": fieldErrors, "message": "validation failed"})
			return
		}
	}
	query, vars, err := buildMutationVars(entityType, object, nil, "PUT")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	hdrs := map[string]string{}
	if t := c.GetHeader("X-Hasura-Tenant-Id"); t != "" {
		hdrs["X-Hasura-Tenant-Id"] = t
	}
	resp, err := sendHasuraRequest(query, vars, hdrs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// actionDynamicDelete handles Hasura action for dynamic_delete
func actionDynamicDelete(c *gin.Context) {
	var payload HasuraActionPayload
	if err := c.BindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	entityType, _ := payload.Input["entity_type"].(string)
	idsIface, _ := payload.Input["ids"].([]interface{})
	ids := []string{}
	for _, v := range idsIface {
		if s, ok := v.(string); ok {
			ids = append(ids, s)
		}
	}
	query, vars, err := buildMutationVars(entityType, nil, ids, "DELETE")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	hdrs := map[string]string{}
	if t := c.GetHeader("X-Hasura-Tenant-Id"); t != "" {
		hdrs["X-Hasura-Tenant-Id"] = t
	}
	resp, err := sendHasuraRequest(query, vars, hdrs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// func main() removed to resolve redeclaration error.
// Please ensure only one main function exists in your project.
