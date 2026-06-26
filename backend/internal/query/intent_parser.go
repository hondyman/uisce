package query

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// IntentParser parses natural language into structured intent
type IntentParser struct {
	// Predefined patterns for common query types
	patterns map[string]*QueryPattern
}

// QueryPattern represents a pattern for parsing queries
type QueryPattern struct {
	Regex       *regexp.Regexp
	IntentType  string
	ExtractFunc func(matches []string) *ParsedIntent
}

// NewIntentParser creates a new intent parser
func NewIntentParser() *IntentParser {
	parser := &IntentParser{
		patterns: make(map[string]*QueryPattern),
	}
	parser.initializePatterns()
	return parser
}

// ParseIntent parses natural language text into structured intent
func (ip *IntentParser) ParseIntent(text string) (*ParsedIntent, error) {
	text = strings.ToLower(strings.TrimSpace(text))

	intent := &ParsedIntent{
		Metrics:     []string{},
		Dimensions:  []string{},
		Filters:     []IntentFilter{},
		Confidence:  0.0,
		RawEntities: make(map[string]string),
	}

	// Try to match against known patterns
	for _, pattern := range ip.patterns {
		matches := pattern.Regex.FindStringSubmatch(text)
		if matches != nil {
			extractedIntent := pattern.ExtractFunc(matches)
			if extractedIntent != nil {
				// Merge extracted intent with current intent
				intent.Metrics = append(intent.Metrics, extractedIntent.Metrics...)
				intent.Dimensions = append(intent.Dimensions, extractedIntent.Dimensions...)
				intent.Filters = append(intent.Filters, extractedIntent.Filters...)
				if extractedIntent.TimeRange != nil {
					intent.TimeRange = extractedIntent.TimeRange
				}
				intent.Confidence = 0.8 // High confidence for pattern matches
				break
			}
		}
	}

	// If no pattern matched, try basic entity extraction
	if intent.Confidence == 0.0 {
		intent = ip.extractBasicEntities(text)
	}

	// Remove duplicates
	intent.Metrics = removeDuplicates(intent.Metrics)
	intent.Dimensions = removeDuplicates(intent.Dimensions)

	return intent, nil
}

// initializePatterns sets up common query patterns
func (ip *IntentParser) initializePatterns() {
	// Pattern: "Show me [metric] by [dimension] for [time_range]"
	ip.patterns["metric_by_dimension_time"] = &QueryPattern{
		Regex: regexp.MustCompile(`show me (.*?) by (.*?) (?:for|in) (.*)`),
		ExtractFunc: func(matches []string) *ParsedIntent {
			metric := strings.TrimSpace(matches[1])
			dimension := strings.TrimSpace(matches[2])
			timeStr := strings.TrimSpace(matches[3])

			intent := &ParsedIntent{
				Metrics:    []string{metric},
				Dimensions: []string{dimension},
			}

			// Parse time range
			intent.TimeRange = ip.parseTimeRange(timeStr)

			return intent
		},
	}

	// Pattern: "What is the [metric] [time_range]"
	ip.patterns["what_is_metric_time"] = &QueryPattern{
		Regex: regexp.MustCompile(`what is the (.*?) (.*)`),
		ExtractFunc: func(matches []string) *ParsedIntent {
			metric := strings.TrimSpace(matches[1])
			timeStr := strings.TrimSpace(matches[2])

			return &ParsedIntent{
				Metrics:   []string{metric},
				TimeRange: ip.parseTimeRange(timeStr),
			}
		},
	}

	// Pattern: "[metric] for [dimension] [time_range]"
	ip.patterns["metric_for_dimension_time"] = &QueryPattern{
		Regex: regexp.MustCompile(`(.*?) for (.*?) (.*)`),
		ExtractFunc: func(matches []string) *ParsedIntent {
			metric := strings.TrimSpace(matches[1])
			dimension := strings.TrimSpace(matches[2])
			timeStr := strings.TrimSpace(matches[3])

			return &ParsedIntent{
				Metrics:    []string{metric},
				Dimensions: []string{dimension},
				TimeRange:  ip.parseTimeRange(timeStr),
			}
		},
	}
}

// extractBasicEntities performs basic entity extraction when patterns don't match
func (ip *IntentParser) extractBasicEntities(text string) *ParsedIntent {
	intent := &ParsedIntent{
		Metrics:     []string{},
		Dimensions:  []string{},
		Filters:     []IntentFilter{},
		Confidence:  0.5,
		RawEntities: make(map[string]string),
	}

	// Simple keyword-based extraction
	words := strings.Fields(text)

	// Look for common metric keywords
	metricKeywords := []string{"average", "avg", "sum", "total", "count", "min", "max", "margin", "value", "revenue", "profit"}
	for _, word := range words {
		for _, keyword := range metricKeywords {
			if strings.Contains(word, keyword) {
				intent.Metrics = append(intent.Metrics, word)
				break
			}
		}
	}

	// Look for common dimension keywords
	dimensionKeywords := []string{"region", "country", "customer", "product", "category", "date", "time", "quarter", "month", "year"}
	for _, word := range words {
		for _, keyword := range dimensionKeywords {
			if strings.Contains(word, keyword) {
				intent.Dimensions = append(intent.Dimensions, word)
				break
			}
		}
	}

	// Look for time-related terms
	timeKeywords := []string{"last", "this", "previous", "current", "quarter", "month", "year", "week"}
	for _, word := range words {
		for _, keyword := range timeKeywords {
			if strings.Contains(word, keyword) {
				if intent.TimeRange == nil {
					intent.TimeRange = &TimeRange{Label: word + " " + getNextWord(words, getWordIndex(words, word))}
				}
				break
			}
		}
	}

	return intent
}

// parseTimeRange parses time range expressions
func (ip *IntentParser) parseTimeRange(timeStr string) *TimeRange {
	timeStr = strings.ToLower(strings.TrimSpace(timeStr))

	// Common time range patterns
	if strings.Contains(timeStr, "last quarter") {
		now := time.Now()
		quarter := (int(now.Month())-1)/3 + 1
		year := now.Year()
		if quarter == 1 {
			quarter = 4
			year--
		} else {
			quarter--
		}

		startMonth := (quarter-1)*3 + 1
		return &TimeRange{
			Start: fmt.Sprintf("%d-%02d-01", year, startMonth),
			End:   fmt.Sprintf("%d-%02d-31", year, startMonth+2),
			Label: "last quarter",
		}
	}

	if strings.Contains(timeStr, "this quarter") {
		now := time.Now()
		quarter := (int(now.Month())-1)/3 + 1
		year := now.Year()

		startMonth := (quarter-1)*3 + 1
		return &TimeRange{
			Start: fmt.Sprintf("%d-%02d-01", year, startMonth),
			End:   fmt.Sprintf("%d-%02d-31", year, startMonth+2),
			Label: "this quarter",
		}
	}

	if strings.Contains(timeStr, "last month") {
		now := time.Now()
		lastMonth := now.AddDate(0, -1, 0)
		year, month, _ := lastMonth.Date()

		return &TimeRange{
			Start: fmt.Sprintf("%d-%02d-01", year, month),
			End:   fmt.Sprintf("%d-%02d-31", year, month),
			Label: "last month",
		}
	}

	// Default: return the string as label
	return &TimeRange{
		Label: timeStr,
	}
}

// Helper functions
func removeDuplicates(slice []string) []string {
	keys := make(map[string]bool)
	var result []string
	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}
	return result
}

func getWordIndex(words []string, target string) int {
	for i, word := range words {
		if word == target {
			return i
		}
	}
	return -1
}

func getNextWord(words []string, index int) string {
	if index >= 0 && index+1 < len(words) {
		return words[index+1]
	}
	return ""
}
