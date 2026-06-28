package main

import (
	"log"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"github.com/semlayer/bp-backend/pkg/workflow"
)

func main() {
	// Create the client object just like in the server
	c, err := client.Dial(client.Options{})
	if err != nil {
		log.Fatalln("unable to create Temporal client", err)
	}
	defer c.Close()

	// This is the task queue the server starts workflows on.
	taskQueue := "bp-task-queue"

	// Create a new worker
	w := worker.New(c, taskQueue, worker.Options{})

	// Register the workflow
	w.RegisterWorkflow(workflow.InterpreterWorkflow)

	// Register activities
	w.RegisterActivity(workflow.SendEmailActivity)
	w.RegisterActivity(workflow.ChargeCreditCardActivity)
	w.RegisterActivity(workflow.CreateUserActivity)

	// Start the worker.
	log.Println("Starting worker...")
	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalln("unable to start worker", err)
	}
}
