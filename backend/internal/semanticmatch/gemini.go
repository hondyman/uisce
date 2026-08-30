package semanticmatch

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type GeminiClient struct {
	APIKey   string
	Model    string // e.g. "gemini-2.5-flash"
	BaseURL  string
	HTTP     *http.Client
	Limiter  *rate.Limiter
	CacheDir string // optional persistent cache of responses

	mu    sync.Mutex
	cache map[string]string
}

func NewGeminiClient(apiKey, model string, rpm int, cacheDir string) *GeminiClient {
	if rpm <= 0 {
		rpm = 60
	}
	return &GeminiClient{
		APIKey:   apiKey,
		Model:    model,
		BaseURL:  "https://generativelanguage.googleapis.com",
		HTTP:     &http.Client{Timeout: 120 * time.Second},
		Limiter:  rate.NewLimiter(rate.Every(time.Minute/time.Duration(rpm)), 2),
		CacheDir: cacheDir,
		cache:    map[string]string{},
	}
}

type genContent struct {
	Parts []genPart `json:"parts"`
}
type genPart struct {
	Text string `json:"text"`
}

type generateRequest struct {
	SystemInstruction *genContent    `json:"systemInstruction,omitempty"`
	Contents          []genContent   `json:"contents"`
	GenerationConfig  generateConfig `json:"generationConfig"`
}
type generateConfig struct {
	Temperature      float32 `json:"temperature"`
	MaxOutputTokens  int     `json:"maxOutputTokens,omitempty"`
	ResponseMIMEType string  `json:"responseMimeType"`
	ResponseSchema   any     `json:"responseSchema,omitempty"`
}

type generateResponse struct {
	Candidates []struct {
		Content      genContent `json:"content"`
		FinishReason string     `json:"finishReason"`
	} `json:"candidates"`
	PromptFeedback *struct {
		BlockReason string `json:"blockReason"`
	} `json:"promptFeedback"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// GenerateJSON calls generateContent with JSON-schema-constrained output.
// Identical requests are served from cache (in-memory + optional disk).
func (g *GeminiClient) GenerateJSON(ctx context.Context, system, user string, schema any) (string, error) {
	key := g.cacheKey(system, user, schema)
	if v, ok := g.cacheGet(key); ok {
		return v, nil
	}
	req := generateRequest{
		SystemInstruction: &genContent{Parts: []genPart{{Text: system}}},
		Contents:          []genContent{{Parts: []genPart{{Text: user}}}},
		GenerationConfig: generateConfig{
			Temperature:      0,
			MaxOutputTokens:  8192,
			ResponseMIMEType: "application/json",
			ResponseSchema:   schema,
		},
	}
	body, _ := json.Marshal(req)
	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent", g.BaseURL, g.Model)

	var lastErr error
	for attempt := 0; attempt < 4; attempt++ {
		if err := g.Limiter.Wait(ctx); err != nil {
			return "", err
		}
		if attempt > 0 {
			d := time.Duration(1<<uint(attempt)) * 700 * time.Millisecond
			d += time.Duration(rand.Int63n(int64(400 * time.Millisecond)))
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(d):
			}
		}
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return "", err
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("x-goog-api-key", g.APIKey)

		resp, err := g.HTTP.Do(httpReq)
		if err != nil {
			lastErr = err
			continue
		}
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var gr generateResponse
		_ = json.Unmarshal(raw, &gr)
		if resp.StatusCode == 200 && len(gr.Candidates) > 0 {
			var text string
			for _, p := range gr.Candidates[0].Content.Parts {
				text += p.Text
			}
			if text == "" {
				lastErr = errors.New("gemini: empty response")
				continue
			}
			g.cachePut(key, text)
			return text, nil
		}
		if gr.Error != nil {
			lastErr = fmt.Errorf("gemini %d: %s", gr.Error.Code, gr.Error.Message)
		} else {
			lastErr = fmt.Errorf("gemini: http %d: %s", resp.StatusCode, truncate(string(raw), 300))
		}
		if resp.StatusCode != http.StatusTooManyRequests && resp.StatusCode/100 != 5 {
			return "", lastErr // non-retryable (bad request, auth, ...)
		}
	}
	return "", lastErr
}

// ---- Embeddings ----

type batchEmbedRequest struct {
	Requests []embedRequest `json:"requests"`
}
type embedRequest struct {
	Model    string     `json:"model"`
	Content  genContent `json:"content"`
	TaskType string     `json:"taskType,omitempty"`
}
type batchEmbedResponse struct {
	Embeddings []struct {
		Values []float32 `json:"values"`
	} `json:"embeddings"`
}

// EmbedBatch embeds texts via models/{model}:batchEmbedContents (100 per call).
func (g *GeminiClient) EmbedBatch(ctx context.Context, model, taskType string, texts []string) ([][]float32, error) {
	out := make([][]float32, 0, len(texts))
	for start := 0; start < len(texts); start += 100 {
		end := start + 100
		if end > len(texts) {
			end = len(texts)
		}
		chunk := texts[start:end]

		req := batchEmbedRequest{}
		for _, t := range chunk {
			req.Requests = append(req.Requests, embedRequest{
				Model:    "models/" + model,
				Content:  genContent{Parts: []genPart{{Text: t}}},
				TaskType: taskType, // RETRIEVAL_DOCUMENT for terms, RETRIEVAL_QUERY for columns
			})
		}
		body, _ := json.Marshal(req)
		url := fmt.Sprintf("%s/v1beta/models/%s:batchEmbedContents", g.BaseURL, model)

		var respRaw []byte
		var err error
		for attempt := 0; attempt < 4; attempt++ {
			if err = g.Limiter.Wait(ctx); err != nil {
				return nil, err
			}
			if attempt > 0 {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(time.Duration(attempt) * 1500 * time.Millisecond):
				}
			}
			httpReq, rerr := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
			if rerr != nil {
				return nil, rerr
			}
			httpReq.Header.Set("Content-Type", "application/json")
			httpReq.Header.Set("x-goog-api-key", g.APIKey)
			resp, derr := g.HTTP.Do(httpReq)
			if derr != nil {
				err = derr
				continue
			}
			respRaw, _ = io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode == 200 {
				err = nil
				break
			}
			err = fmt.Errorf("embed: http %d", resp.StatusCode)
			if resp.StatusCode != http.StatusTooManyRequests && resp.StatusCode/100 != 5 {
				return nil, err
			}
		}
		if err != nil {
			return nil, err
		}
		var ber batchEmbedResponse
		if err := json.Unmarshal(respRaw, &ber); err != nil {
			return nil, err
		}
		if len(ber.Embeddings) != len(chunk) {
			return nil, fmt.Errorf("embed: got %d of %d vectors", len(ber.Embeddings), len(chunk))
		}
		for _, e := range ber.Embeddings {
			out = append(out, e.Values)
		}
	}
	return out, nil
}

// ---- cache ----

func (g *GeminiClient) cacheKey(system, user string, schema any) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s\x00%s\x00%s\x00", g.Model, system, user)
	if schema != nil {
		b, _ := json.Marshal(schema)
		h.Write(b)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func (g *GeminiClient) cacheGet(k string) (string, bool) {
	g.mu.Lock()
	v, ok := g.cache[k]
	g.mu.Unlock()
	if ok {
		return v, true
	}
	if g.CacheDir != "" {
		if b, err := os.ReadFile(filepath.Join(g.CacheDir, k+".json")); err == nil {
			g.mu.Lock()
			g.cache[k] = string(b)
			g.mu.Unlock()
			return string(b), true
		}
	}
	return "", false
}

func (g *GeminiClient) cachePut(k, v string) {
	g.mu.Lock()
	g.cache[k] = v
	g.mu.Unlock()
	if g.CacheDir != "" {
		_ = os.MkdirAll(g.CacheDir, 0o755)
		_ = os.WriteFile(filepath.Join(g.CacheDir, k+".json"), []byte(v), 0o644)
	}
}
