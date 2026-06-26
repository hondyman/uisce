package rag

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

// Tokenizer defines the interface for text tokenization
type Tokenizer interface {
	Encode(text string) []string
	Decode(tokens []string) string
}

// WhitespaceTokenizer is a simple tokenizer that splits on whitespace
// In production, replace this with a proper BPE tokenizer (e.g., tiktoken)
type WhitespaceTokenizer struct{}

func (WhitespaceTokenizer) Encode(text string) []string {
	return strings.Fields(text)
}

func (WhitespaceTokenizer) Decode(tokens []string) string {
	return strings.Join(tokens, " ")
}

var normalizeRe = regexp.MustCompile(`\s+`)

// normalizeWhitespace standardizes text for canonical ID generation
func normalizeWhitespace(s string) string {
	s = strings.ToLower(s)
	s = normalizeRe.ReplaceAllString(s, " ")
	s = strings.TrimSpace(s)
	return s
}

// sha256sum generates a SHA256 hash of the input string
func sha256sum(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// ChunkDocument splits a document into deterministic chunks
func ChunkDocument(documentID, text string, tokenizer Tokenizer, window, overlap int) []Chunk {
	tokens := tokenizer.Encode(text)
	
	// Default values if invalid
	if window <= 0 {
		window = 200
	}
	if overlap < 0 || overlap >= window {
		overlap = window / 4
	}

	var chunks []Chunk
	step := window - overlap
	if step <= 0 {
		step = window
	}

	idx := 0
	for start := 0; start < len(tokens); start += step {
		end := start + window
		if end > len(tokens) {
			end = len(tokens)
		}

		tokSlice := tokens[start:end]
		chunkText := tokenizer.Decode(tokSlice)
		
		// Generate canonical ID
		// ID = SHA256(docID + ":" + index + ":" + normalizedText)
		// This ensures that if the same document is processed again, we get the same IDs
		canonical := normalizeWhitespace(chunkText)
		id := sha256sum(fmt.Sprintf("%s:%d:%s", documentID, idx, canonical))

		chunks = append(chunks, Chunk{
			ChunkID:    id,
			DocumentID: documentID,
			Index:      idx,
			Text:       chunkText,
			TokenCount: len(tokSlice),
			Metadata:   make(map[string]any),
		})

		idx++
		if end == len(tokens) {
			break
		}
	}

	return chunks
}
