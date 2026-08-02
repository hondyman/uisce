package handlers

import (
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func HandleCreateQuery(c *gin.Context) {
	var req struct {
		Name        string                 `json:"name" binding:"required"`
		Description string                 `json:"description"`
		Type        string                 `json:"type" binding:"required"`
		Config      map[string]interface{} `json:"config" binding:"required"`
		Tags        []string               `json:"tags"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	query := gin.H{
		"id":          generateID(),
		"name":        req.Name,
		"description": req.Description,
		"type":        req.Type,
		"config":      req.Config,
		"tags":        req.Tags,
		"created_by":  "current_user",
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"is_core":     false,
	}

	c.JSON(201, query)
}

func HandleGetQueries(c *gin.Context) {
	queries := []gin.H{
		{
			"id":          "1",
			"name":        "Monthly Sales Report",
			"description": "Sales performance by month",
			"type":        "public",
			"created_by":  "john.doe",
			"created_at":  "2024-01-15T10:00:00Z",
			"updated_at":  "2024-01-15T10:00:00Z",
			"is_core":     true,
			"tags":        []string{"sales", "monthly"},
		},
	}

	c.JSON(200, gin.H{"queries": queries})
}

func HandleGetQuery(c *gin.Context) {
	id := c.Param("id")
	query := gin.H{
		"id":          id,
		"name":        "Sample Query",
		"description": "Sample query description",
		"type":        "public",
		"config":      gin.H{"dataSource": "orders", "measures": []string{"total_amount"}},
		"created_by":  "john.doe",
		"created_at":  "2024-01-15T10:00:00Z",
		"updated_at":  "2024-01-15T10:00:00Z",
		"is_core":     false,
	}

	c.JSON(200, query)
}

func HandleUpdateQuery(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Name        string                 `json:"name"`
		Description string                 `json:"description"`
		Type        string                 `json:"type"`
		Config      map[string]interface{} `json:"config"`
		Tags        []string               `json:"tags"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	query := gin.H{
		"id":          id,
		"name":        req.Name,
		"description": req.Description,
		"type":        req.Type,
		"config":      req.Config,
		"tags":        req.Tags,
		"updated_at":  time.Now(),
	}

	c.JSON(200, query)
}

func HandleDeleteQuery(c *gin.Context) {
	id := c.Param("id")
	log.Printf("Deleting query with id: %s", id)
	c.JSON(204, gin.H{})
}

func HandleCloneQuery(c *gin.Context) {
	id := c.Param("id")
	log.Printf("Cloning query with id: %s", id)
	clonedQuery := gin.H{
		"id":          generateID(),
		"name":        "Cloned Query",
		"description": "Cloned from original query",
		"type":        "private",
		"created_by":  "current_user",
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"is_core":     false,
	}

	c.JSON(201, clonedQuery)
}

func HandleShareQuery(c *gin.Context) {
	id := c.Param("id")
	log.Printf("Sharing query with id: %s", id)
	var req struct {
		Users []string `json:"users"`
		Teams []string `json:"teams"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"id":     id,
		"shared": true,
		"users":  req.Users,
		"teams":  req.Teams,
	})
}
