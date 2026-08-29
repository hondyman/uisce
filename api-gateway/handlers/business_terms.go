package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

var backendURL string

func SetBackendURL(url string) {
	backendURL = url
}

func HandleBusinessTermSearch(c *gin.Context) {
	log.Printf("HANDLER CALLED: HandleBusinessTermSearch")

	rawBody, _ := c.GetRawData()
	log.Printf("RAW REQUEST BODY: %s", string(rawBody))
	c.Request.Body = io.NopCloser(bytes.NewBuffer(rawBody))

	var req BusinessTermSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("JSON BINDING ERROR: %v", err)
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	log.Printf("PARSED REQUEST: %+v", req)

	if req.Limit == 0 {
		req.Limit = 20
	}

	// SECURITY: tenant scope must come only from the caller's verified JWT
	// (set by JWTMiddleware into gin context + the X-Tenant-ID header it
	// re-signs after stripping any client-supplied value) — never from the
	// request body, and never a "default" fallback tenant. A request-body
	// tenant_id is fully attacker-controlled.
	tenantID, _ := c.Get("semlayer_tenant_id")
	tenantIDStr, _ := tenantID.(string)
	if tenantIDStr == "" {
		c.JSON(401, gin.H{"error": "missing or invalid tenant context"})
		return
	}

	datasourceID := c.GetHeader("X-Tenant-Datasource-ID")
	if datasourceID == "" {
		datasourceID = "default"
	}

	backendReq := map[string]interface{}{
		"query": req.Query,
		"limit": req.Limit,
	}

	bodyJSON, err := json.Marshal(backendReq)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to marshal request: " + err.Error()})
		return
	}

	backendURLStr := backendURL
	if backendURLStr == "" {
		backendURLStr = "http://localhost:8080"
	}
	httpReq, err := http.NewRequest("POST", backendURLStr+"/business-terms/search", bytes.NewBuffer(bodyJSON))
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to create backend request: " + err.Error()})
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Tenant-ID", tenantIDStr)
	httpReq.Header.Set("X-Tenant-Datasource-ID", datasourceID)
	// Forward the caller's bearer token so the backend independently
	// re-validates auth and re-derives tenant from its own JWT check —
	// this proxied request previously carried no Authorization at all.
	if auth := c.GetHeader("Authorization"); auth != "" {
		httpReq.Header.Set("Authorization", auth)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to reach backend service: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to read backend response: " + err.Error()})
		return
	}

	c.Data(resp.StatusCode, "application/json", body)
}

func HandleBusinessTermValidation(c *gin.Context) {
	var req BusinessTermValidationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"valid":    true,
		"errors":   []string{},
		"warnings": []string{},
	})
}

func HandleGetBusinessTerms(c *gin.Context) {
	businessTerms := []gin.H{
		{
			"id":           "customer_id",
			"name":         "Customer ID",
			"description":  "Unique identifier for customers",
			"category":     "Customer Data",
			"owner":        "Data Team",
			"status":       "approved",
			"related_apis": []string{"1", "2"},
		},
	}

	c.JSON(200, gin.H{"business_terms": businessTerms})
}

func HandleCreateBusinessTerm(c *gin.Context) {
	var termData map[string]interface{}
	if err := c.ShouldBindJSON(&termData); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Creating business term: %+v", termData)

	term := gin.H{
		"id":          generateID(),
		"name":        termData["name"],
		"description": termData["description"],
		"category":    termData["category"],
		"owner":       termData["owner"],
		"status":      "pending",
		"created_at":  time.Now(),
	}

	c.JSON(201, term)
}
