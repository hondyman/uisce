package handlers

import (
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

func HandleCreateAPI(c *gin.Context) {
	var req struct {
		Name        string                   `json:"name" binding:"required"`
		Description string                   `json:"description"`
		Type        string                   `json:"type" binding:"required"`
		Config      map[string]interface{}   `json:"config" binding:"required"`
		Endpoints   []map[string]interface{} `json:"endpoints"`
		Tags        []string                 `json:"tags"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	api := gin.H{
		"id":          generateID(),
		"name":        req.Name,
		"description": req.Description,
		"type":        req.Type,
		"config":      req.Config,
		"endpoints":   req.Endpoints,
		"tags":        req.Tags,
		"created_by":  "current_user",
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"is_core":     false,
	}

	c.JSON(201, api)
}

func HandleGetAPI(c *gin.Context) {
	id := c.Param("id")
	api := gin.H{
		"id":          id,
		"name":        "Sample API",
		"description": "Sample API description",
		"type":        "public",
		"config":      gin.H{"basePath": "/api", "authentication": "jwt"},
		"endpoints":   []gin.H{},
		"created_by":  "john.doe",
		"created_at":  "2024-01-15T10:00:00Z",
		"updated_at":  "2024-01-15T10:00:00Z",
		"is_core":     false,
	}

	c.JSON(200, api)
}

func HandleUpdateAPI(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Name        string                   `json:"name"`
		Description string                   `json:"description"`
		Type        string                   `json:"type"`
		Config      map[string]interface{}   `json:"config"`
		Endpoints   []map[string]interface{} `json:"endpoints"`
		Tags        []string                 `json:"tags"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	api := gin.H{
		"id":          id,
		"name":        req.Name,
		"description": req.Description,
		"type":        req.Type,
		"config":      req.Config,
		"endpoints":   req.Endpoints,
		"tags":        req.Tags,
		"updated_at":  time.Now(),
	}

	c.JSON(200, api)
}

func HandleDeleteAPI(c *gin.Context) {
	id := c.Param("id")
	log.Printf("Deleting API with id: %s", id)
	c.JSON(204, gin.H{})
}

func HandleCloneAPI(c *gin.Context) {
	id := c.Param("id")
	log.Printf("Cloning API with id: %s", id)
	clonedAPI := gin.H{
		"id":          generateID(),
		"name":        "Cloned API",
		"description": "Cloned from original API",
		"type":        "private",
		"created_by":  "current_user",
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"is_core":     false,
	}

	c.JSON(201, clonedAPI)
}

func HandleShareAPI(c *gin.Context) {
	id := c.Param("id")
	log.Printf("Sharing API with id: %s", id)
	var req struct {
		Users []string `json:"users"`
		Teams []string `json:"teams"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "API shared successfully"})
}

func HandleExecuteAPI(c *gin.Context) {
	apiId := c.Param("apiId")
	path := c.Param("path")
	method := c.Request.Method

	log.Printf("Executing API: %s %s", method, path)
	c.JSON(200, gin.H{
		"api_id":    apiId,
		"path":      path,
		"method":    method,
		"executed":  true,
		"result":    fmt.Sprintf("API %s executed successfully", apiId),
	})
}

func HandleGetAPIs(c *gin.Context) {
	apis := []gin.H{
		{
			"id":          "1",
			"name":        "Customer API",
			"description": "Customer management API",
			"type":        "public",
			"created_by":  "john.doe",
			"created_at":  "2024-01-15T10:00:00Z",
			"updated_at":  "2024-01-15T10:00:00Z",
			"is_core":     true,
			"tags":        []string{"customer", "api"},
		},
	}

	c.JSON(200, gin.H{"apis": apis})
}
