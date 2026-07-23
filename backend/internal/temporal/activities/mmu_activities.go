package activities

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// MMUActivities contains activities for Multi-Modal Understanding.
type MMUActivities struct {
	db         *sqlx.DB
	httpClient *http.Client
	config     MMUConfig
}

// MMUConfig holds configuration for MMU processing.
type MMUConfig struct {
	OpenAIAPIKey   string
	OpenAIModel    string // e.g., "gpt-4o"
	EmbeddingModel string // e.g., "text-embedding-3-small"
	ChunkSize      int    // target chunk size in characters
	ChunkOverlap   int    // overlap between chunks
	MaxPagesPerDoc int    // max pages to process per document
}

// NewMMUActivities creates a new MMU activities instance.
func NewMMUActivities(db *sqlx.DB, config MMUConfig) *MMUActivities {
	if config.ChunkSize == 0 {
		config.ChunkSize = 1000
	}
	if config.ChunkOverlap == 0 {
		config.ChunkOverlap = 200
	}
	if config.MaxPagesPerDoc == 0 {
		config.MaxPagesPerDoc = 100
	}
	if config.OpenAIModel == "" {
		config.OpenAIModel = "gpt-4o"
	}
	if config.EmbeddingModel == "" {
		config.EmbeddingModel = "text-embedding-3-small"
	}

	return &MMUActivities{
		db: db,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		config: config,
	}
}

// DocumentInput represents a document to be processed.
type DocumentInput struct {
	TenantID     string `json:"tenant_id"`
	DocumentID   string `json:"document_id"`
	SourceURL    string `json:"source_url,omitempty"`
	Content      []byte `json:"content,omitempty"`
	ContentType  string `json:"content_type"`
	Filename     string `json:"filename"`
	EntityID     string `json:"entity_id,omitempty"`
	DocumentType string `json:"document_type"`
}

// ProcessedDocument represents the result of document processing.
type ProcessedDocument struct {
	DocumentID    string                 `json:"document_id"`
	ChunkCount    int                    `json:"chunk_count"`
	PageCount     int                    `json:"page_count"`
	ExtractedData map[string]interface{} `json:"extracted_data"`
	Metadata      map[string]interface{} `json:"metadata"`
	ProcessedAt   time.Time              `json:"processed_at"`
}

// ExtractTextFromDocument extracts text from various document formats.
func (a *MMUActivities) ExtractTextFromDocument(
	ctx context.Context,
	input DocumentInput,
) (string, error) {
	var content []byte
	var err error

	// Fetch content if URL provided
	if input.SourceURL != "" {
		content, err = a.fetchDocumentContent(ctx, input.SourceURL)
		if err != nil {
			return "", fmt.Errorf("fetch document: %w", err)
		}
	} else {
		content = input.Content
	}

	// Determine content type and extract text
	switch {
	case strings.HasSuffix(input.Filename, ".pdf") || input.ContentType == "application/pdf":
		return a.extractTextFromPDF(ctx, content)
	case strings.HasSuffix(input.Filename, ".txt") || input.ContentType == "text/plain":
		return string(content), nil
	case strings.HasSuffix(input.Filename, ".html") || input.ContentType == "text/html":
		return a.extractTextFromHTML(content)
	case strings.HasSuffix(input.Filename, ".json") || input.ContentType == "application/json":
		return string(content), nil
	default:
		return string(content), nil
	}
}

func (a *MMUActivities) fetchDocumentContent(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (a *MMUActivities) extractTextFromPDF(ctx context.Context, content []byte) (string, error) {
	// Note: In production, this would use go-fitz (MuPDF bindings) or similar
	// For now, we'll use a placeholder that could be replaced with actual PDF extraction
	// Example with go-fitz:
	// doc, err := fitz.New(content)
	// if err != nil { return "", err }
	// defer doc.Close()
	// var text strings.Builder
	// for n := 0; n < doc.NumPage() && n < a.config.MaxPagesPerDoc; n++ {
	//     pageText, _ := doc.Text(n)
	//     text.WriteString(pageText)
	// }
	// return text.String(), nil

	// Placeholder - would use actual PDF library
	return fmt.Sprintf("[PDF content - %d bytes - requires go-fitz integration]", len(content)), nil
}

func (a *MMUActivities) extractTextFromHTML(content []byte) (string, error) {
	// Simple HTML text extraction (strip tags)
	// In production, would use a proper HTML parser
	text := string(content)
	// Basic tag removal - production would use golang.org/x/net/html
	for {
		start := strings.Index(text, "<")
		if start == -1 {
			break
		}
		end := strings.Index(text[start:], ">")
		if end == -1 {
			break
		}
		text = text[:start] + " " + text[start+end+1:]
	}
	return strings.TrimSpace(text), nil
}

// ChunkDocument splits document text into overlapping chunks.
func (a *MMUActivities) ChunkDocument(
	ctx context.Context,
	text string,
	documentID string,
) ([]DocumentChunk, error) {
	chunks := []DocumentChunk{}

	if len(text) == 0 {
		return chunks, nil
	}

	// Split by paragraphs first, then combine into chunks
	paragraphs := strings.Split(text, "\n\n")

	var currentChunk strings.Builder
	chunkIndex := 0

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		// If adding this paragraph exceeds chunk size, save current chunk
		if currentChunk.Len()+len(para) > a.config.ChunkSize && currentChunk.Len() > 0 {
			chunk := DocumentChunk{
				ChunkID:    uuid.New().String(),
				DocumentID: documentID,
				ChunkIndex: chunkIndex,
				Content:    currentChunk.String(),
			}
			chunks = append(chunks, chunk)

			// Start new chunk with overlap
			overlapText := getOverlapText(currentChunk.String(), a.config.ChunkOverlap)
			currentChunk.Reset()
			currentChunk.WriteString(overlapText)
			chunkIndex++
		}

		if currentChunk.Len() > 0 {
			currentChunk.WriteString("\n\n")
		}
		currentChunk.WriteString(para)
	}

	// Don't forget the last chunk
	if currentChunk.Len() > 0 {
		chunk := DocumentChunk{
			ChunkID:    uuid.New().String(),
			DocumentID: documentID,
			ChunkIndex: chunkIndex,
			Content:    currentChunk.String(),
		}
		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

// DocumentChunk represents a chunk of document text.
type DocumentChunk struct {
	ChunkID    string    `json:"chunk_id"`
	DocumentID string    `json:"document_id"`
	ChunkIndex int       `json:"chunk_index"`
	Content    string    `json:"content"`
	Embedding  []float32 `json:"embedding,omitempty"`
}

func getOverlapText(text string, overlapSize int) string {
	if len(text) <= overlapSize {
		return text
	}
	// Try to break at word boundary
	start := len(text) - overlapSize
	for i := start; i < len(text); i++ {
		if text[i] == ' ' {
			return text[i+1:]
		}
	}
	return text[start:]
}

// GenerateEmbeddings generates embeddings for document chunks.
func (a *MMUActivities) GenerateEmbeddings(
	ctx context.Context,
	chunks []DocumentChunk,
) ([]DocumentChunk, error) {
	if len(chunks) == 0 {
		return chunks, nil
	}

	// Batch embeddings in groups of 20
	batchSize := 20
	for i := 0; i < len(chunks); i += batchSize {
		end := i + batchSize
		if end > len(chunks) {
			end = len(chunks)
		}

		batch := chunks[i:end]
		texts := make([]string, len(batch))
		for j, chunk := range batch {
			texts[j] = chunk.Content
		}

		embeddings, err := a.callOpenAIEmbeddings(ctx, texts)
		if err != nil {
			return nil, fmt.Errorf("generate embeddings batch %d: %w", i/batchSize, err)
		}

		for j := range batch {
			chunks[i+j].Embedding = embeddings[j]
		}
	}

	return chunks, nil
}

func (a *MMUActivities) callOpenAIEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	reqBody := map[string]interface{}{
		"input": texts,
		"model": a.config.EmbeddingModel,
	}

	jsonBody, _ := json.Marshal(reqBody)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/embeddings", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+a.config.OpenAIAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API error: %d - %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	embeddings := make([][]float32, len(result.Data))
	for i, d := range result.Data {
		embeddings[i] = d.Embedding
	}

	return embeddings, nil
}

// StoreDocumentChunks stores document chunks with embeddings in the database.
func (a *MMUActivities) StoreDocumentChunks(
	ctx context.Context,
	tenantID string,
	documentID string,
	entityID string,
	chunks []DocumentChunk,
) error {
	tx, err := a.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, chunk := range chunks {
		// Convert embedding to pgvector format
		embeddingStr := formatEmbedding(chunk.Embedding)

		_, err := tx.ExecContext(ctx, `
			INSERT INTO document_chunks (
				chunk_id, tenant_id, document_id, entity_id, 
				chunk_index, content, content_hash, embedding, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8::vector, NOW())
			ON CONFLICT (chunk_id) DO UPDATE SET
				content = EXCLUDED.content,
				embedding = EXCLUDED.embedding
		`, chunk.ChunkID, tenantID, documentID, entityID,
			chunk.ChunkIndex, chunk.Content, hashContent(chunk.Content), embeddingStr)

		if err != nil {
			return fmt.Errorf("store chunk %s: %w", chunk.ChunkID, err)
		}
	}

	return tx.Commit()
}

func formatEmbedding(embedding []float32) string {
	if len(embedding) == 0 {
		return ""
	}
	parts := make([]string, len(embedding))
	for i, v := range embedding {
		parts[i] = fmt.Sprintf("%f", v)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func hashContent(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:])
}

// ExtractStructuredData extracts structured data from document text using OpenAI.
func (a *MMUActivities) ExtractStructuredData(
	ctx context.Context,
	text string,
	documentType string,
	schema map[string]interface{},
) (map[string]interface{}, error) {
	// Build prompt based on document type
	systemPrompt := buildExtractionPrompt(documentType, schema)

	reqBody := map[string]interface{}{
		"model": a.config.OpenAIModel,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": text},
		},
		"response_format": map[string]interface{}{
			"type": "json_object",
		},
		"temperature": 0.1,
	}

	jsonBody, _ := json.Marshal(reqBody)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+a.config.OpenAIAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API error: %d - %s", resp.StatusCode, string(body))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenAI")
	}

	var extracted map[string]interface{}
	if err := json.Unmarshal([]byte(result.Choices[0].Message.Content), &extracted); err != nil {
		return nil, fmt.Errorf("parse extraction result: %w", err)
	}

	return extracted, nil
}

func buildExtractionPrompt(documentType string, schema map[string]interface{}) string {
	schemaJSON, _ := json.MarshalIndent(schema, "", "  ")

	basePrompt := `You are a financial document analysis expert. Extract structured data from the provided document.
Output ONLY valid JSON matching the provided schema. If a field cannot be extracted, use null.

Document Type: %s

Expected Schema:
%s

Extract the following information from the document and return as JSON:`

	return fmt.Sprintf(basePrompt, documentType, string(schemaJSON))
}

// GetExtractionSchema returns the extraction schema for a document type.
func GetExtractionSchema(documentType string) map[string]interface{} {
	schemas := map[string]map[string]interface{}{
		"10K": {
			"company_name":       "string",
			"cik":                "string",
			"fiscal_year_end":    "string",
			"total_revenue":      "number",
			"net_income":         "number",
			"total_assets":       "number",
			"total_liabilities":  "number",
			"risk_factors":       "array of strings",
			"executive_officers": "array of objects with name and title",
			"material_contracts": "array of strings",
		},
		"10Q": {
			"company_name":       "string",
			"quarter":            "string",
			"revenue":            "number",
			"net_income":         "number",
			"significant_events": "array of strings",
		},
		"8K": {
			"company_name":      "string",
			"event_date":        "string",
			"event_types":       "array of strings",
			"event_description": "string",
			"financial_impact":  "string or null",
		},
		"proxy": {
			"company_name":           "string",
			"meeting_date":           "string",
			"executive_compensation": "array of objects",
			"board_members":          "array of objects with name and committees",
			"proposals":              "array of objects with description and recommendation",
		},
		"prospectus": {
			"issuer_name":     "string",
			"offering_type":   "string",
			"shares_offered":  "number",
			"price_range":     "string",
			"use_of_proceeds": "string",
			"risk_factors":    "array of strings",
		},
		"contract": {
			"parties":          "array of strings",
			"effective_date":   "string",
			"termination_date": "string or null",
			"key_terms":        "array of strings",
			"obligations":      "array of objects",
			"governing_law":    "string",
		},
	}

	if schema, ok := schemas[documentType]; ok {
		return schema
	}

	// Default schema for unknown document types
	return map[string]interface{}{
		"title":           "string",
		"date":            "string",
		"parties":         "array of strings",
		"key_information": "array of strings",
		"amounts":         "array of objects with description and value",
	}
}

// ProcessDocumentPipeline runs the full document processing pipeline.
func (a *MMUActivities) ProcessDocumentPipeline(
	ctx context.Context,
	input DocumentInput,
) (*ProcessedDocument, error) {
	// 1. Extract text
	text, err := a.ExtractTextFromDocument(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("extract text: %w", err)
	}

	// 2. Chunk document
	chunks, err := a.ChunkDocument(ctx, text, input.DocumentID)
	if err != nil {
		return nil, fmt.Errorf("chunk document: %w", err)
	}

	// 3. Generate embeddings
	chunks, err = a.GenerateEmbeddings(ctx, chunks)
	if err != nil {
		return nil, fmt.Errorf("generate embeddings: %w", err)
	}

	// 4. Store chunks
	if err := a.StoreDocumentChunks(ctx, input.TenantID, input.DocumentID, input.EntityID, chunks); err != nil {
		return nil, fmt.Errorf("store chunks: %w", err)
	}

	// 5. Extract structured data
	schema := GetExtractionSchema(input.DocumentType)
	extractedData, err := a.ExtractStructuredData(ctx, text, input.DocumentType, schema)
	if err != nil {
		// Don't fail the whole pipeline if extraction fails
		extractedData = map[string]interface{}{"extraction_error": err.Error()}
	}

	return &ProcessedDocument{
		DocumentID:    input.DocumentID,
		ChunkCount:    len(chunks),
		PageCount:     estimatePageCount(text),
		ExtractedData: extractedData,
		Metadata: map[string]interface{}{
			"filename":      input.Filename,
			"content_type":  input.ContentType,
			"document_type": input.DocumentType,
			"text_length":   len(text),
		},
		ProcessedAt: time.Now(),
	}, nil
}

func estimatePageCount(text string) int {
	// Rough estimate: ~3000 characters per page
	return (len(text) / 3000) + 1
}
