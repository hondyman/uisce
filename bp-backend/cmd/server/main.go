package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/pgxpool"
	"go.temporal.io/sdk/client"

	"github.com/semlayer/bp-backend/pkg/workflow"
)

func main() {
	// Temporal client
	temporalClient, err := client.Dial(client.Options{})
	if err != nil {
		log.Fatalln("Unable to create Temporal client", err)
	}
	defer temporalClient.Close()

	// Postgres connection
	dbpool, err := pgxpool.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalln("Unable to connect to database", err)
	}
	defer dbpool.Close()

	r := gin.Default()

	// API endpoints
	api := r.Group("/api/v1")
	{
		api.POST("/workflow_versions", func(c *gin.Context) {
			var dsl workflow.WorkflowDefinition
			if err := c.ShouldBindJSON(&dsl); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			dslJSON, err := json.Marshal(dsl)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			_, err = dbpool.Exec(context.Background(), "INSERT INTO workflow_versions (definition_snapshot) VALUES ($1)", dslJSON)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		api.POST("/workflow_versions/:id/run", func(c *gin.Context) {
			id := c.Param("id")

			var dslJSON json.RawMessage
			err := dbpool.QueryRow(context.Background(), "SELECT definition_snapshot FROM workflow_versions WHERE id = $1", id).Scan(&dslJSON)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "workflow not found"})
				return
			}

			var dsl workflow.WorkflowDefinition
			if err := json.Unmarshal(dslJSON, &dsl); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			workflowOptions := client.StartWorkflowOptions{
				ID:        "interpreter-workflow-" + id,
				TaskQueue: "bp-task-queue",
			}

			we, err := temporalClient.ExecuteWorkflow(context.Background(), workflowOptions, workflow.InterpreterWorkflow, dsl)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"workflow_id": we.GetID(), "run_id": we.GetRunID()})
		})
	}

	// Serve frontend
	r.StaticFS("/", http.Dir("../business-process-designer/dist"))

	r.Run(":8080")
}