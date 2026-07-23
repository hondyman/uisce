package workflows

import (
	"time"

	"go.temporal.io/sdk/workflow"
)

// AutomatedRetrainingWorkflow manages the ML model lifecycle
func AutomatedRetrainingWorkflow(ctx workflow.Context) error {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 1 * time.Hour, // Training can take time
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	logger := workflow.GetLogger(ctx)
	logger.Info("Starting Automated Retraining Workflow")

	// 1. Train New Model (Challenger)
	var trainingResult struct {
		ModelPath string  `json:"modelPath"`
		Accuracy  float64 `json:"accuracy"`
		AUC       float64 `json:"auc"`
	}
	// In reality this activity would execute the python script via docker or shell
	err := workflow.ExecuteActivity(ctx, "ActivityTrainRiskModel").Get(ctx, &trainingResult)
	if err != nil {
		logger.Error("Model training failed", "Error", err)
		return err
	}

	// 2. Evaluate against Champion (Current Model)
	// We check if the new AUC is better than a threshold or the previous best
	championAUC := 0.80 // Mocked current baseline

	if trainingResult.AUC > championAUC {
		logger.Info("Challenger model outperforms Champion. Promoting...", "NewAUC", trainingResult.AUC, "OldAUC", championAUC)

		// 3. Promote to Production
		// This endpoint would swap the .pkl file or update the service config using the /reload endpoint
		err = workflow.ExecuteActivity(ctx, "ActivityDeployModel", trainingResult.ModelPath).Get(ctx, nil)
		if err != nil {
			logger.Error("Deployment failed", "Error", err)
			return err
		}

		// 4. Send Notification
		_ = workflow.ExecuteActivity(ctx, "SendNotification", "Model Retrained & Deployed", "New Accuracy: "+trainingResult.ModelPath).Get(ctx, nil)
	} else {
		logger.Info("Challenger model did not improve performance. Discarding.", "NewAUC", trainingResult.AUC)
	}

	return nil
}
