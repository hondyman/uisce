package llm

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// LLMConfig represents configurable LLM provider settings persisted for dev/admin UI.
type LLMConfig struct {
	Provider       string         `json:"provider"`
	Model          string         `json:"model"`
	EmbeddingModel string         `json:"embedding_model"`
	Params         map[string]any `json:"params"`
	APIKey         string         `json:"api_key,omitempty"`
}

// LLMConfigService provides simple file-backed config storage for admin UI.
type LLMConfigService struct {
	path string
	mu   sync.RWMutex
}

// NewLLMConfigService creates a new service storing config at the provided path.
func NewLLMConfigService(path string) *LLMConfigService {
	return &LLMConfigService{path: path}
}

// Get loads the config from disk. If file does not exist, returns a sensible default.
func (s *LLMConfigService) Get() (*LLMConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	f, err := os.Open(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			// default config
			return &LLMConfig{Provider: "gemini", Model: "gemini-2.0-flash-exp", EmbeddingModel: "text-embedding-004", Params: map[string]any{"temperature": 0.2}}, nil
		}
		return nil, err
	}
	defer f.Close()
	var cfg LLMConfig
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Set writes the config to disk.
func (s *LLMConfigService) Set(cfg *LLMConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := os.Create(s.path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(cfg)
}

// Test uses the configured provider to run a quick prompt to validate connectivity.
func (s *LLMConfigService) Test(ctx context.Context, cfg *LLMConfig, prompt string) (string, error) {
	// For now support Gemini via existing llm package. API key may be passed in cfg.APIKey or read from env by provider.
	provider := NewGeminiProvider(cfg.APIKey, cfg.Model)
	return provider.GenerateResponse(ctx, prompt)
}

// Helper to ensure config path exists (directory)
func (s *LLMConfigService) EnsurePathDir() error {
	parent := filepath.Dir(s.path)
	if parent == "" || parent == "." {
		return nil
	}
	return os.MkdirAll(parent, 0o755)
}
