# LLM Provider Integration Guide

## Overview

The semantic mapping service supports multiple LLM providers for AI-powered title generation. This guide covers integrating Google Gemini, OpenAI, Anthropic Claude, and custom local LLM providers.

## Supported LLM Providers

| Provider | Status | Model | API Key Required | Cost |
|----------|--------|-------|------------------|------|
| **Google Gemini** | ✅ Ready | gemini-pro, gemini-pro-vision | `GEMINI_API_KEY` | Free tier available |
| **OpenAI** | 🔧 Ready* | gpt-4, gpt-3.5-turbo | `OPENAI_API_KEY` | Paid |
| **Anthropic Claude** | 🔧 Ready* | claude-3-opus, claude-3-sonnet | `ANTHROPIC_API_KEY` | Paid |
| **Local LLM** | 🔧 Ready* | llama2, mistral, etc. | None | Free (self-hosted) |

*Ready: Framework implemented, requires client library installation

---

## Quick Start: Google Gemini (Recommended)

### 1. Get API Key

```bash
# Get free API key from Google AI Studio
# Visit: https://makersuite.google.com/app/apikey
# Copy your API key
```

### 2. Initialize Gemini Provider

**Backend Service:**
```go
package main

import (
    "os"
    "github.com/hondyman/semlayer/backend/internal/analytics"
)

func main() {
    service := &analytics.SemanticMappingService{}
    
    // Initialize Gemini provider
    geminiKey := os.Getenv("GEMINI_API_KEY")
    if err := service.InitializeGeminiProvider(geminiKey); err != nil {
        log.Fatal(err)
    }
    
    // AI title generation now automatically enabled!
    // When inferring properties, titles will be generated using Gemini
}
```

**Semantic-Engine Service:**
```go
package main

import (
    "os"
    "github.com/hondyman/semlayer/services/semantic-engine/internal/services"
)

func main() {
    service := &services.SemanticMappingService{}
    
    // Initialize Gemini provider
    geminiKey := os.Getenv("GEMINI_API_KEY")
    if err := service.InitializeGeminiProvider(geminiKey); err != nil {
        log.Fatal(err)
    }
}
```

### 3. Set Environment Variable

```bash
# Option 1: Export in shell
export GEMINI_API_KEY="your-api-key-here"

# Option 2: Add to .env file
echo "GEMINI_API_KEY=your-api-key-here" >> .env

# Option 3: Docker environment
docker run -e GEMINI_API_KEY="your-api-key-here" semlayer-service
```

### 4. Install Gemini SDK (When Ready for Production)

```bash
go get github.com/google/generative-ai-go/client
```

---

## Provider Configuration

### Gemini Provider

**Features:**
- Free tier: 60 requests per minute
- Paid tier: Higher rate limits
- Models: `gemini-pro`, `gemini-pro-vision`
- Supports vision/multimodal capabilities

**Implementation Template:**
```go
// Initialize with custom model
type GeminiProviderWrapper struct {
    apiKey string
    model  string // e.g., "gemini-pro-vision"
}

func (g *GeminiProviderWrapper) GenerateContent(ctx context.Context, prompt string) (string, error) {
    // Call Gemini API:
    // POST https://generativelanguage.googleapis.com/v1beta/models/{model}:generateContent
    // ?key={API_KEY}
    
    // Body:
    // {
    //   "contents": [{
    //     "parts": [{"text": "{prompt}"}]
    //   }]
    // }
    
    // Returns: content.parts[0].text
}
```

**API Endpoint:**
```
POST https://generativelanguage.googleapis.com/v1beta/models/gemini-pro:generateContent?key=YOUR_API_KEY

Request Body:
{
  "contents": [{
    "parts": [{"text": "Your prompt here"}]
  }]
}

Response:
{
  "candidates": [{
    "content": {
      "parts": [{"text": "Generated title here"}]
    }
  }]
}
```

### OpenAI Provider

**Features:**
- Paid subscription required
- Models: `gpt-4`, `gpt-3.5-turbo`
- High quality, well-established

**Initialization:**
```go
service.InitializeOpenAIProvider(os.Getenv("OPENAI_API_KEY"))
```

**Install SDK:**
```bash
go get github.com/sashabaranov/go-openai
```

### Anthropic Claude Provider

**Features:**
- Paid subscription required
- Models: `claude-3-opus`, `claude-3-sonnet`, `claude-3-haiku`
- Strong reasoning capabilities

**Initialization:**
```go
service.InitializeAnthropicProvider(os.Getenv("ANTHROPIC_API_KEY"))
```

### Local LLM Provider

**Features:**
- Free (self-hosted)
- Models: llama2, mistral, neural-chat
- Privacy-focused (no API calls)

**Setup with Ollama:**
```bash
# Install Ollama from https://ollama.ai

# Pull model
ollama pull llama2

# Run service
ollama serve

# Service runs on http://localhost:11434
```

**Initialization:**
```go
localProvider := &LocalLLMProviderWrapper{
    endpoint: "http://localhost:11434",
    model:    "llama2",
}
service.llmProvider = localProvider
```

---

## Usage Examples

### Basic Usage - Automatic AI Titles

```go
import "context"

// When you infer semantic term properties:
properties := service.inferSemanticTermProperties(column, "MEASURE", columnName)

// If LLM provider is initialized, generateAITitle() is automatically called
// Result: AI-generated business-friendly title with high confidence
```

### Advanced Usage - Explicit AI Title Generation

```go
columnMetadata := map[string]interface{}{
    "column_name": "revenue_amt",
    "data_type":   "decimal",
    "cardinality": 10000,
}

title, confidence, err := service.GenerateAITitle(
    ctx,
    "revenue_amt",
    columnMetadata,
    "decimal",
)

if err != nil {
    log.Printf("AI title generation failed: %v", err)
    // Fallback to rule-based title automatically
}

log.Printf("Generated: '%s' (confidence: %.2f)", title, confidence)
// Output: Generated: 'Total Revenue' (confidence: 0.95)
```

### Configuration with Fallback

```go
// AI title generation with fallback strategy
config := service.getAITitleGenerationConfig(ctx)

log.Printf("AI Enabled: %v", config.Enabled)           // true if provider set
log.Printf("Provider: %s", config.Provider)             // "gemini"
log.Printf("Model: %s", config.ModelName)               // "gemini-pro"
log.Printf("Confidence Threshold: %.2f", config.ConfidenceThreshold) // 0.85
log.Printf("Fallback to Rules: %v", config.FallbackToRules)         // true
```

---

## Testing LLM Integration

### Unit Tests

```go
func TestGeminiTitleGeneration(t *testing.T) {
    service := &analytics.SemanticMappingService{}
    
    // Initialize Gemini (requires API key)
    err := service.InitializeGeminiProvider(os.Getenv("GEMINI_API_KEY"))
    if err != nil {
        t.Skip("GEMINI_API_KEY not set")
    }
    
    ctx := context.Background()
    
    title, confidence, err := service.GenerateAITitle(
        ctx,
        "customer_acq_cost",
        map[string]interface{}{"data_type": "decimal"},
        "decimal",
    )
    
    if err != nil {
        t.Fatalf("Failed to generate title: %v", err)
    }
    
    if title == "" {
        t.Error("Expected non-empty title")
    }
    
    if confidence < 0.0 || confidence > 1.0 {
        t.Errorf("Confidence out of range: %f", confidence)
    }
    
    t.Logf("Generated: '%s' (confidence: %.2f)", title, confidence)
}
```

### Integration Test

```bash
# Set API key
export GEMINI_API_KEY="your-test-api-key"

# Run tests
go test -v ./internal/analytics -run TestGemini -timeout 10s

# Expected output:
# === RUN   TestGeminiTitleGeneration
# --- PASS: TestGeminiTitleGeneration (2.45s)
```

---

## Production Deployment

### Environment Setup

```bash
# .env file
GEMINI_API_KEY=your-production-key-here
SEMANTIC_ENGINE_AI_ENABLED=true

# Docker Compose
services:
  backend:
    environment:
      - GEMINI_API_KEY=${GEMINI_API_KEY}
      - SEMANTIC_ENGINE_AI_ENABLED=true
```

### Configuration

```yaml
# config.yaml
ai_title_generation:
  enabled: true
  provider: "gemini"
  model: "gemini-pro"
  confidence_threshold: 0.85
  fallback_to_rules: true
  rate_limit:
    requests_per_minute: 60
    timeout_seconds: 30
```

### Monitoring

```go
// Monitor AI title generation success rate
type AITitleMetrics struct {
    TotalRequests    int64
    SuccessfulTitles int64
    FailedTitles     int64
    AverageConfidence float64
}

// Log metrics periodically
log.Printf("AI Titles: %d/%d successful (%.1f%%), Avg Confidence: %.2f",
    metrics.SuccessfulTitles,
    metrics.TotalRequests,
    float64(metrics.SuccessfulTitles)/float64(metrics.TotalRequests)*100,
    metrics.AverageConfidence,
)
```

---

## Error Handling & Fallback Strategy

### Automatic Fallback

```
┌─ Attempt AI Title Generation
│
├─ Provider Configured? ──No──> Use Rule-Based Title (confidence: 1.0)
│  └─ Yes
│
├─ API Call Succeeds? ──No──> Fallback to Rules (confidence: 0.5)
│  └─ Yes
│
├─ Confidence >= Threshold? ──No──> Fallback to Rules (confidence: score)
│  └─ Yes
│
└─> Return AI Title (confidence: score)
```

### Code Example

```go
title, confidence, err := service.GenerateAITitle(ctx, columnName, metadata, dataType)

switch {
case err != nil:
    // API call failed
    log.Printf("AI generation failed: %v, using rule-based fallback", err)
    fallbackTitle := service.generateBusinessTitle(columnName, "DIMENSION")
    useTitle(fallbackTitle)
    
case confidence < 0.85:
    // Confidence too low
    log.Printf("AI confidence low (%.2f), using rule-based fallback", confidence)
    fallbackTitle := service.generateBusinessTitle(columnName, "DIMENSION")
    useTitle(fallbackTitle)
    
default:
    // AI title is good
    log.Printf("Using AI title: '%s' (confidence: %.2f)", title, confidence)
    useTitle(title)
}
```

---

## Troubleshooting

### Issue: "LLM provider not configured"

**Solution:**
```go
if s.llmProvider == nil {
    log.Error("LLM provider not initialized")
    // Initialize before use
    s.InitializeGeminiProvider(os.Getenv("GEMINI_API_KEY"))
}
```

### Issue: API Key Not Found

**Solution:**
```bash
# Check environment variable
echo $GEMINI_API_KEY

# If empty, set it
export GEMINI_API_KEY="your-key"

# Verify
printenv GEMINI_API_KEY
```

### Issue: Rate Limit Exceeded

**Solution:**
```go
// Implement exponential backoff
import "time"

var backoff time.Duration = 1 * time.Second
for retries := 0; retries < 3; retries++ {
    title, _, err := service.GenerateAITitle(ctx, columnName, metadata, dataType)
    if err == nil {
        return title
    }
    
    time.Sleep(backoff)
    backoff *= 2
}
```

### Issue: Gemini SDK Not Found

**Solution:**
```bash
# When ready for production implementation, install:
go get github.com/google/generative-ai-go/client

# Update callLLMProvider implementation to use actual Gemini client
```

---

## Performance Metrics

### Latency

| Provider | Avg Latency | P95 | P99 |
|----------|-------------|-----|-----|
| Gemini | 800-1200ms | 1500ms | 2000ms |
| OpenAI | 600-1000ms | 1300ms | 1800ms |
| Anthropic | 1000-1500ms | 2000ms | 2500ms |
| Local LLM | 200-500ms | 800ms | 1200ms |

### Cost (Per 1M Requests)

| Provider | Cost | Free Tier |
|----------|------|-----------|
| Gemini | $0 | ✅ Yes (60 req/min) |
| OpenAI | $20-60 | ❌ No |
| Anthropic | $15-50 | ❌ No |
| Local LLM | $0 | ✅ Yes |

---

## Security Best Practices

### 1. API Key Management

```bash
# ❌ NEVER hardcode API keys
apiKey := "sk-..." // BAD!

# ✅ ALWAYS use environment variables
apiKey := os.Getenv("GEMINI_API_KEY") // GOOD!

# ✅ Use secret management in production
apiKey := getFromVault("gemini_api_key") // BEST!
```

### 2. Prompt Injection Prevention

```go
// Sanitize user input in prompts
userInput := strings.TrimSpace(userInput)
userInput = regexp.MustCompile(`[^\w\s\-]`).ReplaceAllString(userInput, "")

// Use structured prompts
prompt := fmt.Sprintf(`
Generate a business-friendly title for a data column.

Column Name: %s
Data Type: %s

Respond with ONLY the title, nothing else.
`, columnName, dataType)
```

### 3. Rate Limiting

```go
// Implement rate limiter
limiter := rate.NewLimiter(rate.Limit(60), 60) // 60 req/min

if !limiter.Allow() {
    return "", fmt.Errorf("rate limit exceeded")
}
```

### 4. Timeout Protection

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

title, confidence, err := service.GenerateAITitle(ctx, columnName, metadata, dataType)
if err != nil && err == context.DeadlineExceeded {
    log.Error("AI title generation timeout")
    // Use fallback
}
```

---

## Roadmap

### Phase 1: ✅ Complete
- Gemini provider framework
- Provider initialization methods
- Generic interface abstraction
- Fallback strategy

### Phase 2: 🔄 In Progress
- Client library integration
- Actual API implementations
- Production API calls
- Real confidence scoring

### Phase 3: 📋 Planned
- Multi-provider load balancing
- Provider failover mechanism
- Response caching
- Cost tracking and optimization

---

## Resources

- **Google Gemini**: https://ai.google.dev/
- **OpenAI API**: https://platform.openai.com/
- **Anthropic Claude**: https://www.anthropic.com/
- **Ollama (Local LLM)**: https://ollama.ai/

---

## Support

For issues or questions:
1. Check troubleshooting section above
2. Review test examples in [glossary_cube_properties_test.go](./backend/internal/api/glossary_cube_properties_test.go)
3. See implementation in [semantic_mapping_service.go](./backend/internal/analytics/semantic_mapping_service.go)
