package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
)

var temporalClient client.Client

func main() {
	var err error
	temporalClient, err = client.Dial(client.Options{})
	if err != nil {
		log.Fatalln("Unable to create Temporal client", err)
	}
	defer temporalClient.Close()

	http.HandleFunc("/start_rebalance", startRebalance)
	http.HandleFunc("/signal/approve", signalApprove)

	log.Println("Orchestration API listening on :8081")
	// wrap with JWT middleware
	jwtMw := jwtmiddleware.NewJWTMiddleware("/health")
	handler := jwtMw.Handler(http.DefaultServeMux)
	log.Fatal(http.ListenAndServe(":8081", handler))
}

type RebalanceInput struct {
	PortfolioID string `json:"portfolio_id"`
}

func startRebalance(w http.ResponseWriter, r *http.Request) {
	tclaims := jwtmiddleware.GetClaimsFromContext(r)
	if tclaims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	tenantID := tclaims.TenantID

	var input RebalanceInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	workflowID := fmt.Sprintf("%s:%s:%s", tenantID, input.PortfolioID, uuid.NewString())
	workflowOptions := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: "rebalance",
	}

	// Assuming PortfolioRebalanceWorkflow is registered in the worker
	// For this API stub, we use the string name
	we, err := temporalClient.ExecuteWorkflow(context.Background(), workflowOptions, "PortfolioRebalanceWorkflow", input)
	if err != nil {
		http.Error(w, "Failed to start workflow", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"workflow_id": we.GetID(), "run_id": we.GetRunID()})
}

type SignalInput struct {
	WorkflowID string `json:"workflow_id"`
	Signal     string `json:"signal"`
}

func signalApprove(w http.ResponseWriter, r *http.Request) {
	var sig SignalInput
	if err := json.NewDecoder(r.Body).Decode(&sig); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := temporalClient.SignalWorkflow(context.Background(), sig.WorkflowID, "", sig.Signal, true)
	if err != nil {
		http.Error(w, "Failed to signal workflow", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "signaled"})
}
