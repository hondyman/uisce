package activities

import (
	"context"
	"fmt"

	"github.com/hondyman/semlayer/backend/internal/intelligence/document"
)

// Re-export ValidationResult for workflow usage
type ValidationResult = document.ValidationResult

// GeminiExtractActivity invokes the ExtractionWorker.
func GeminiExtractActivity(ctx context.Context, input struct {
	DocumentID string
	StorageURI string
	SchemaDef  string
}) (string, error) {
	// In a real implementation, we would fetch the PDF bytes from StorageURI here.
	// For now, we'll assume a placeholder or that the bytes are passed (though passing bytes to activity is bad practice for large files).
	// Let's assume we have a global or injected worker instance.
	// This is a stub for the activity.

	// TODO: Retrieve PDF bytes from StorageURI (GCS/S3)
	// pdfBytes := []byte("dummy pdf content")

	// TODO: Use the actual worker instance
	// worker := GetGlobalExtractionWorker()
	// return worker.ExtractFinancialData(ctx, pdfBytes, input.SchemaDef)

	return "{}", nil
}

// ValidateSchemaActivity invokes the validator.
func ValidateSchemaActivity(ctx context.Context, rawJSON string, schemaDef string) (*ValidationResult, error) {
	return document.ValidateAgainstODS(rawJSON, schemaDef)
}

// PersistDataActivity saves the result to the database.
func PersistDataActivity(ctx context.Context, jsonResult string) error {
	fmt.Printf("Persisting data: %s\n", jsonResult)
	return nil
}

// NotifyHumanReviewActivity sends a notification for review.
func NotifyHumanReviewActivity(ctx context.Context, docID string, errors []string) error {
	fmt.Printf("Review required for doc %s. Errors: %v\n", docID, errors)
	return nil
}

// MarkDocumentRejectedActivity marks the document as rejected.
func MarkDocumentRejectedActivity(ctx context.Context, docID string) error {
	fmt.Printf("Document %s rejected.\n", docID)
	return nil
}
