package document

import (
	"context"
	"fmt"

	"google.golang.org/genai"
)

// ExtractionWorker manages the lifecycle of document processing tasks.
type ExtractionWorker struct {
	client    *genai.Client
	modelName string
	projectID string
	location  string
}

// NewExtractionWorker initializes the Gemini client with Vertex AI backend.
// It requires GOOGLE_CLOUD_PROJECT and GOOGLE_CLOUD_LOCATION to be set in the environment
// or passed explicitly to ensure correct routing of requests.
func NewExtractionWorker(ctx context.Context, projectID, location, modelName string) (*ExtractionWorker, error) {
	// Initialize the client with Vertex AI backend configuration.
	// We explicitly set the Backend to BackendVertexAI to use IAM auth instead of API Keys.
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:  projectID,
		Location: location,
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create vertex ai genai client: %w", err)
	}

	return &ExtractionWorker{
		client:    client,
		modelName: modelName, // e.g., "gemini-1.5-pro-001"
		projectID: projectID,
		location:  location,
	}, nil
}

// ExtractFinancialData parses a PDF and returns the raw JSON string.
func (w *ExtractionWorker) ExtractFinancialData(ctx context.Context, pdfBytes []byte, schemaDef string) (string, error) {
	// Define the prompt that guides the model's focus.
	// We inject the schema definition into the prompt context to ground the model.
	userPrompt := fmt.Sprintf(`
		Analyze the attached financial statement. 
		Extract all relevant fields according to the following JSON schema definition.
		Ensure that numerical values are extracted as raw numbers, not strings.
		Dates must be formatted as ISO 8601 (YYYY-MM-DD).
		
		Schema Definition:
		%s
	`, schemaDef)

	// Construct the multimodal content parts.
	// Part 1: The textual instructions.
	// Part 2: The PDF file passed as inline binary data.
	parts := []*genai.Part{
		{Text: userPrompt},
		{InlineData: &genai.Blob{
			Data:     pdfBytes,
			MIMEType: "application/pdf",
		}},
	}

	// Configure the generation parameters.
	// Setting ResponseMIMEType to "application/json" is crucial for controlled generation.
	// Temperature is set to 0.0 to maximize determinism and reduce hallucinations.
	config := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		Temperature:      genai.Ptr(float32(0.0)),
		CandidateCount:   1,
	}

	// Execute the request.
	// The Content structure wraps the parts and assigns the "user" role.
	resp, err := w.client.Models.GenerateContent(ctx, w.modelName, []*genai.Content{
		{
			Parts: parts,
			Role:  "user",
		},
	}, config)

	if err != nil {
		return "", fmt.Errorf("gemini generation error: %w", err)
	}

	// Extract the text result from the first candidate.
	return resp.Text(), nil
}
