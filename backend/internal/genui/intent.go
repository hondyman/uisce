package genui

import (
	"context"
	"strings"
)

func containsWord(query string, word string) bool {
	tokens := strings.FieldsFunc(query, func(r rune) bool {
		return !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9')
	})
	for _, token := range tokens {
		if token == word {
			return true
		}
	}
	return false
}

// Intent represents a parsed user query
type Intent struct {
	Type       string         `json:"type"`       // dashboard, chart, grid, form, etc.
	Objects    []string       `json:"objects"`    // business objects involved
	Metrics    []string       `json:"metrics"`    // metrics to display
	TimeRange  *TimeRange     `json:"time_range"` // optional time filter
	Filters    map[string]any `json:"filters"`    // additional filters
	Confidence float64        `json:"confidence"` // 0.0 - 1.0
}

// TimeRange represents a time filter
type TimeRange struct {
	Start string `json:"start"`
	End   string `json:"end"`
	Unit  string `json:"unit"` // day, week, month, year
}

// IntentClassifier parses natural language queries into structured intents
type IntentClassifier struct {
	// In production, this would use an LLM or trained model
	// For now, we'll use simple pattern matching
}

func NewIntentClassifier() *IntentClassifier {
	return &IntentClassifier{}
}

// Classify parses a natural language query into a structured intent
func (ic *IntentClassifier) Classify(ctx context.Context, query string) (*Intent, error) {
	query = strings.ToLower(strings.TrimSpace(query))

	intent := &Intent{
		Objects:    []string{},
		Metrics:    []string{},
		Filters:    make(map[string]any),
		Confidence: 0.8, // Default confidence
	}

	// Determine intent type
	// Check for approval inbox queries first
	if strings.Contains(query, "approval") || strings.Contains(query, "pending") || strings.Contains(query, "inbox") {
		intent.Type = "approval_inbox"
		intent.Objects = append(intent.Objects, "Workflow", "Approval")
	} else if strings.Contains(query, "show") || strings.Contains(query, "display") {
		if containsWord(query, "form") {
			intent.Type = "form"
		} else if containsWord(query, "list") || containsWord(query, "table") {
			intent.Type = "grid"
		} else {
			intent.Type = "chart"
		}
	} else if strings.Contains(query, "compare") {
		intent.Type = "chart"
	} else {
		intent.Type = "dashboard"
	}

	// Extract business objects
	objects := []struct {
		keyword string
		object  string
	}{
		{"portfolio", "Portfolio"},
		{"account", "Account"},
		{"position", "Position"},
		{"holding", "Position"},
		{"holdings", "Position"},
		{"trade", "Trade"},
		{"client", "Client"},
		{"advisor", "Advisor"},
		{"goal", "Goal"},
	}

	for _, obj := range objects {
		if strings.Contains(query, obj.keyword) {
			intent.Objects = append(intent.Objects, obj.object)
		}
	}

	// Extract metrics
	metrics := []struct {
		keyword string
		metric  string
	}{
		{"performance", "nav"},
		{"return", "return_pct"},
		{"drift", "drift_pct"},
		{"value", "market_value"},
		{"allocation", "allocation_pct"},
		{"nav", "nav"},
	}

	for _, m := range metrics {
		if strings.Contains(query, m.keyword) {
			intent.Metrics = append(intent.Metrics, m.metric)
		}
	}

	// Extract time range
	if strings.Contains(query, "ytd") || strings.Contains(query, "year to date") {
		intent.TimeRange = &TimeRange{Start: "ytd", Unit: "day"}
	} else if strings.Contains(query, "last month") {
		intent.TimeRange = &TimeRange{Start: "1m", Unit: "day"}
	} else if strings.Contains(query, "last year") {
		intent.TimeRange = &TimeRange{Start: "1y", Unit: "month"}
	}

	return intent, nil
}

// Example implementation with LLM integration (commented out)
/*
func (ic *IntentClassifier) ClassifyWithLLM(ctx context.Context, query string) (*Intent, error) {
	// Call OpenAI/Anthropic with structured output
	prompt := fmt.Sprintf(`Parse this wealth management query into structured intent:
Query: %s

Extract:
- Type: dashboard|chart|grid|form
- Objects: list of business objects (Portfolio, Account, etc.)
- Metrics: list of metrics (nav, return_pct, etc.)
- TimeRange: {start, end, unit}
- Filters: any additional filters

Return JSON.`, query)

	// response := llm.Call(ctx, prompt)
	// var intent Intent
	// json.Unmarshal(response, &intent)
	// return &intent, nil
}
*/
