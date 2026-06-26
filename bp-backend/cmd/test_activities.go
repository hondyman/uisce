package main

import (
	"context"
	"log"
	"time"

	bpworkflow "github.com/semlayer/bp-backend/pkg/workflow"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	tworkflow "go.temporal.io/sdk/workflow"
)

// TestActivitiesWorkflow is a simple workflow that executes the new activities.
func TestActivitiesWorkflow(ctx tworkflow.Context) (string, error) {
	ao := tworkflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
	}
	ctx = tworkflow.WithActivityOptions(ctx, ao)

	var result string
	err := tworkflow.ExecuteActivity(ctx, bpworkflow.SendEmailActivity, "test@example.com", "Test Email", "Hello, World!").Get(ctx, &result)
	if err != nil {
		return "", err
	}
	log.Println("SendEmailActivity result:", result)

	err = tworkflow.ExecuteActivity(ctx, bpworkflow.ChargeCreditCardActivity, 123.45, "1234567812345678").Get(ctx, &result)
	if err != nil {
		return "", err
	}
	log.Println("ChargeCreditCardActivity result:", result)

	err = tworkflow.ExecuteActivity(ctx, bpworkflow.CreateUserActivity, "testuser", "testuser@example.com").Get(ctx, &result)
	if err != nil {
		return "", err
	}
	log.Println("CreateUserActivity result:", result)

	return "All activities executed successfully", nil
}

func main() {
	c, err := client.Dial(client.Options{})
	if err != nil {
		log.Fatalln("Unable to create client", err)
	}
	defer c.Close()

	taskQueue := "test-activities-queue"

	// Start a worker
	w := worker.New(c, taskQueue, worker.Options{})
	w.RegisterWorkflow(TestActivitiesWorkflow)
	w.RegisterActivity(bpworkflow.SendEmailActivity)
	w.RegisterActivity(bpworkflow.ChargeCreditCardActivity)
	w.RegisterActivity(bpworkflow.CreateUserActivity)

	// The worker is started in a separate goroutine
	go func() {
		log.Println("Starting test worker...")
		if err := w.Run(worker.InterruptCh()); err != nil {
			log.Fatalln("Unable to start worker", err)
		}
	}()

	// Start the workflow
	workflowOptions := client.StartWorkflowOptions{
		ID:        "test-activities-workflow",
		TaskQueue: taskQueue,
	}

	log.Println("Executing TestActivitiesWorkflow...")
	we, err := c.ExecuteWorkflow(context.Background(), workflowOptions, TestActivitiesWorkflow)
	if err != nil {
		log.Fatalln("Unable to execute workflow", err)
	}

	// Get the results
	var result string
	err = we.Get(context.Background(), &result)
	if err != nil {
		log.Fatalln("Unable get workflow result", err)
	}

	log.Printf("Workflow finished. Result: %s\n", result)
}
