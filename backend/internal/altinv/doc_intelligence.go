package altinv

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"github.com/google/uuid"
	"google.golang.org/api/option"
)

// DocumentIntelligenceService processes alternative investment documents using AI
type DocumentIntelligenceService interface {
	ProcessQuarterlyStatement(ctx context.Context, documentID uuid.UUID, pdfText string) (*ExtractedQuarterlyData, error)
	ProcessK1Document(ctx context.Context, documentID uuid.UUID, pdfText string) (map[string]interface{}, error)
}

type documentIntelligenceService struct {
	geminiClient *genai.Client
	altInvSvc    Service
}

// NewDocumentIntelligenceService creates a new document intelligence service
func NewDocumentIntelligenceService(apiKey string, altInvSvc Service) (DocumentIntelligenceService, error) {
	client, err := genai.NewClient(context.Background(), option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	return &documentIntelligenceService{
		geminiClient: client,
		altInvSvc:    altInvSvc,
	}, nil
}

// ProcessQuarterlyStatement extracts structured data from a GP quarterly statement
func (s *documentIntelligenceService) ProcessQuarterlyStatement(ctx context.Context, documentID uuid.UUID, pdfText string) (*ExtractedQuarterlyData, error) {
	model := s.geminiClient.GenerativeModel("gemini-1.5-flash")
	model.SetTemperature(0.1) // Low temperature for consistent extraction

	prompt := fmt.Sprintf(`
Extract the following information from this private equity/alternative investment quarterly statement.
Return ONLY a valid JSON object with these exact keys:

{
  "nav": <net asset value as a number or null>,
  "nav_date": "<date in YYYY-MM-DD format or null>",
  "capital_called": <capital called this quarter as a number or null>,
  "distributions": <distributions paid this quarter as a number or null>,
  "irr": <IRR since inception as a percentage number (e.g., 15.5 for 15.5%%) or null>,
  "tvpi": <TVPI (Total Value / Paid-In) as a decimal (e.g., 1.45) or null>,
  "unfunded_commitment": <unfunded commitment remaining as a number or null>
}

If a value is not found in the document, use null. Do not include any explanatory text, ONLY the JSON.

Document text:
%s
`, pdfText)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("Gemini API error: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no response from Gemini")
	}

	// Extract text from response
	responseText := ""
	for _, part := range resp.Candidates[0].Content.Parts {
		if txt, ok := part.(genai.Text); ok {
			responseText += string(txt)
		}
	}

	// Clean up response (remove markdown code blocks if present)
	responseText = strings.TrimSpace(responseText)
	responseText = strings.TrimPrefix(responseText, "```json")
	responseText = strings.TrimPrefix(responseText, "```")
	responseText = strings.TrimSuffix(responseText, "```")
	responseText = strings.TrimSpace(responseText)

	// Parse JSON response
	var extracted ExtractedQuarterlyData
	if err := json.Unmarshal([]byte(responseText), &extracted); err != nil {
		return nil, fmt.Errorf("failed to parse Gemini response as JSON: %w\nResponse: %s", err, responseText)
	}

	// Calculate confidence based on how many fields were extracted
	fieldsExtracted := 0
	totalFields := 7

	if extracted.NAV != nil {
		fieldsExtracted++
	}
	if extracted.NAVDate != nil {
		fieldsExtracted++
	}
	if extracted.CapitalCalled != nil {
		fieldsExtracted++
	}
	if extracted.Distributions != nil {
		fieldsExtracted++
	}
	if extracted.IRR != nil {
		fieldsExtracted++
	}
	if extracted.TVPI != nil {
		fieldsExtracted++
	}
	if extracted.UnfundedCommitment != nil {
		fieldsExtracted++
	}

	confidence := float64(fieldsExtracted) / float64(totalFields)

	// Update document with extracted data
	extractedJSON, _ := json.Marshal(extracted)
	status := ExtractCompleted
	if confidence < 0.5 {
		status = ExtractManualReviewRequired
	}

	err = s.altInvSvc.UpdateDocumentExtraction(ctx, documentID, extractedJSON, confidence, status)
	if err != nil {
		return nil, fmt.Errorf("failed to update document extraction: %w", err)
	}

	return &extracted, nil
}

// ProcessK1Document extracts data from a K-1 tax document
func (s *documentIntelligenceService) ProcessK1Document(ctx context.Context, documentID uuid.UUID, pdfText string) (map[string]interface{}, error) {
	model := s.geminiClient.GenerativeModel("gemini-1.5-flash")
	model.SetTemperature(0.1)

	prompt := fmt.Sprintf(`
Extract the following information from this K-1 tax document for a partnership investment.
Return ONLY a valid JSON object with these keys:

{
  "partnership_name": "<name or null>",
  "tax_year": <year as number or null>,
  "ordinary_income": <amount or null>,
  "capital_gains": <amount or null>,
  "distributions": <amount or null>,
  "partner_share_percentage": <percentage as decimal (e.g., 0.05 for 5%%) or null>
}

Document text:
%s
`, pdfText)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("Gemini API error: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no response from Gemini")
	}

	responseText := ""
	for _, part := range resp.Candidates[0].Content.Parts {
		if txt, ok := part.(genai.Text); ok {
			responseText += string(txt)
		}
	}

	responseText = strings.TrimSpace(responseText)
	responseText = strings.TrimPrefix(responseText, "```json")
	responseText = strings.TrimPrefix(responseText, "```")
	responseText = strings.TrimSuffix(responseText, "```")
	responseText = strings.TrimSpace(responseText)

	var extracted map[string]interface{}
	if err := json.Unmarshal([]byte(responseText), &extracted); err != nil {
		return nil, fmt.Errorf("failed to parse Gemini response: %w", err)
	}

	// Update document
	extractedJSON, _ := json.Marshal(extracted)
	confidence := 0.8 // K-1s are more standardized
	status := ExtractCompleted

	err = s.altInvSvc.UpdateDocumentExtraction(ctx, documentID, extractedJSON, confidence, status)
	if err != nil {
		return nil, fmt.Errorf("failed to update document extraction: %w", err)
	}

	return extracted, nil
}
