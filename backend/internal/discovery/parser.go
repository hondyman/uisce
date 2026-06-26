package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/hondyman/semlayer/backend/internal/models"
)

// LogParser extracts structured fields from log lines and events
type LogParser struct {
	logger *log.Logger
}

// ParsedLogField represents a field extracted from logs
type ParsedLogField struct {
	FieldName  string
	FieldType  string // inferred: string, number, boolean, timestamp, json
	SampleVal  interface{}
	Frequency  int
	SourceType string // "json_field", "regex_pattern", "kv_pair"
	Pattern    *regexp.Regexp
	Confidence float64 // 0.0-1.0
}

// LogSource represents where logs come from
type LogSource struct {
	Type      string // "kubernetes", "application", "database", "syslog"
	Namespace string
	Service   string
	LogLevel  string
}

// NewLogParser creates a new log parser
func NewLogParser(logger *log.Logger) *LogParser {
	return &LogParser{
		logger: logger,
	}
}

// ParseStructuredLogs extracts fields from JSON-structured logs
func (lp *LogParser) ParseStructuredLogs(ctx context.Context, logLines []string) ([]ParsedLogField, error) {
	fieldMap := make(map[string]*ParsedLogField)

	for _, line := range logLines {
		var logEntry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &logEntry); err != nil {
			// Try unstructured parsing if JSON fails
			lp.parseUnstructuredLog(ctx, line, fieldMap)
			continue
		}

		// Recursively extract all fields from nested JSON
		lp.extractJSONFields(logEntry, "", fieldMap)
	}

	// Convert map to slice
	result := make([]ParsedLogField, 0, len(fieldMap))
	for _, field := range fieldMap {
		if field.Frequency >= 3 { // Require field to appear at least 3 times
			result = append(result, *field)
		}
	}

	return result, nil
}

// extractJSONFields recursively extracts fields from JSON objects
func (lp *LogParser) extractJSONFields(obj interface{}, prefix string, fieldMap map[string]*ParsedLogField) {
	switch v := obj.(type) {
	case map[string]interface{}:
		for key, val := range v {
			newKey := key
			if prefix != "" {
				newKey = prefix + "_" + key
			}

			// Skip system fields
			if strings.HasPrefix(strings.ToLower(key), "_") || shouldSkipField(key, "") {
				continue
			}

			switch innerVal := val.(type) {
			case string:
				lp.updateFieldMap(fieldMap, newKey, "string", innerVal, "json_field", 1.0)
			case float64:
				lp.updateFieldMap(fieldMap, newKey, "number", innerVal, "json_field", 1.0)
			case bool:
				lp.updateFieldMap(fieldMap, newKey, "boolean", innerVal, "json_field", 1.0)
			case map[string]interface{}:
				// Recurse into nested objects
				lp.extractJSONFields(innerVal, newKey, fieldMap)
			case []interface{}:
				// Sample array elements
				if len(innerVal) > 0 {
					lp.extractJSONFields(innerVal[0], newKey, fieldMap)
				}
			}
		}

	case []interface{}:
		for i, item := range v {
			newKey := fmt.Sprintf("%s_%d", prefix, i)
			lp.extractJSONFields(item, newKey, fieldMap)
		}
	}
}

// parseUnstructuredLog extracts fields using regex patterns
func (lp *LogParser) parseUnstructuredLog(ctx context.Context, line string, fieldMap map[string]*ParsedLogField) {
	// Common log patterns
	patterns := map[string]string{
		"timestamp":     `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`,
		"http_code":     `status[=:](\d{3})`,
		"http_method":   `(GET|POST|PUT|DELETE|PATCH|HEAD)\s`,
		"duration_ms":   `duration[=:]\s*(\d+)\s*ms`,
		"response_time": `response_time[=:]\s*(\d+(?:\.\d+)?)\s*ms`,
		"error_message": `error[=:]?\s*"([^"]*)"`,
		"user_id":       `user_id[=:](\w+)`,
		"request_id":    `request_id[=:]([a-f0-9\-]+)`,
		"service":       `service[=:](\w+)`,
		"component":     `component[=:](\w+)`,
	}

	for fieldName, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(line); len(matches) > 0 {
			val := matches[1] // First capture group
			if val == "" && len(matches) > 0 {
				val = matches[0]
			}

			ftype := lp.inferFieldType(fieldName, val)
			lp.updateFieldMap(fieldMap, fieldName, ftype, val, "regex_pattern", 0.7)
		}
	}

	// Key-value pair extraction: key=value or key:"value"
	kvPattern := regexp.MustCompile(`(\w+)[=:]\s*([^\s,\]]+)`)
	for _, match := range kvPattern.FindAllStringSubmatch(line, -1) {
		key := match[1]
		val := match[2]

		// Clean up quotes
		val = strings.Trim(val, `"'`)

		if !shouldSkipField(key, "") {
			ftype := lp.inferFieldType(key, val)
			lp.updateFieldMap(fieldMap, key, ftype, val, "kv_pair", 0.6)
		}
	}
}

// updateFieldMap updates field map with new observation
func (lp *LogParser) updateFieldMap(fieldMap map[string]*ParsedLogField, key string, ftype string, val interface{}, sourceType string, confidence float64) {
	if field, exists := fieldMap[key]; exists {
		field.Frequency++
		// Update confidence on repeated observation
		field.Confidence = (field.Confidence + confidence) / 2.0
	} else {
		fieldMap[key] = &ParsedLogField{
			FieldName:  key,
			FieldType:  ftype,
			SampleVal:  val,
			Frequency:  1,
			SourceType: sourceType,
			Confidence: confidence,
		}
	}
}

// inferFieldType infers data type from field name and value
func (lp *LogParser) inferFieldType(fieldName string, value string) string {
	lower := strings.ToLower(fieldName)

	// Type inference from field name
	if strings.Contains(lower, "time") || strings.Contains(lower, "date") {
		return "timestamp"
	}
	if strings.Contains(lower, "count") || strings.Contains(lower, "total") || strings.Contains(lower, "number") {
		return "number"
	}
	if strings.Contains(lower, "enabled") || strings.Contains(lower, "active") || strings.Contains(lower, "flag") {
		return "boolean"
	}
	if strings.Contains(lower, "status") || strings.Contains(lower, "code") {
		return "categorical"
	}

	// Type inference from value
	if value == "true" || value == "false" {
		return "boolean"
	}
	if regexp.MustCompile(`^\d+$`).MatchString(value) {
		return "number"
	}
	if regexp.MustCompile(`^\d+\.\d+$`).MatchString(value) {
		return "number"
	}
	if regexp.MustCompile(`^\d{4}-\d{2}-\d{2}`).MatchString(value) {
		return "timestamp"
	}

	return "string"
}

// ExtractLogMetrics analyzes log patterns for key metrics
func (lp *LogParser) ExtractLogMetrics(ctx context.Context, logLines []string) (map[string]interface{}, error) {
	metrics := make(map[string]interface{})

	httpCodeCounts := make(map[string]int)
	httpMethodCounts := make(map[string]int)
	var responseTimes []float64
	errorCount := 0

	codePattern := regexp.MustCompile(`status[=:](\d{3})`)
	methodPattern := regexp.MustCompile(`(GET|POST|PUT|DELETE|PATCH|HEAD)\s`)
	rtPattern := regexp.MustCompile(`response_time[=:]\s*(\d+(?:\.\d+)?)\s*ms`)
	errorPattern := regexp.MustCompile(`(?i)error|exception|failed`)

	for _, line := range logLines {
		// HTTP codes
		if matches := codePattern.FindStringSubmatch(line); len(matches) > 0 {
			httpCodeCounts[matches[1]]++
		}

		// HTTP methods
		if matches := methodPattern.FindStringSubmatch(line); len(matches) > 0 {
			httpMethodCounts[matches[1]]++
		}

		// Response times
		if matches := rtPattern.FindStringSubmatch(line); len(matches) > 0 {
			var rt float64
			fmt.Sscanf(matches[1], "%f", &rt)
			responseTimes = append(responseTimes, rt)
		}

		// Error detection
		if errorPattern.MatchString(line) {
			errorCount++
		}
	}

	metrics["http_code_distribution"] = httpCodeCounts
	metrics["http_method_distribution"] = httpMethodCounts
	metrics["error_rate"] = float64(errorCount) / float64(len(logLines))

	if len(responseTimes) > 0 {
		avg := 0.0
		for _, rt := range responseTimes {
			avg += rt
		}
		avg /= float64(len(responseTimes))
		metrics["avg_response_time_ms"] = avg
	}

	return metrics, nil
}

// ConvertToFeatureCandidates converts parsed log fields to feature candidates
func (lp *LogParser) ConvertToFeatureCandidates(fields []ParsedLogField) []models.FeatureCandidate {
	candidates := make([]models.FeatureCandidate, len(fields))

	for i, field := range fields {
		candidates[i] = models.FeatureCandidate{
			Name:           "log_" + field.FieldName,
			SourceDatabase: "logs",
			SourceField:    field.FieldName,
			DataType:       field.FieldType,
			Completeness:   0.8, // Assume logs are relatively complete
			Cardinality:    -1,  // Unknown
			BusinessValue:  0,   // To be scored
			TechnicalScore: field.Confidence,
			DiscoveredAt:   time.Now(),
			Status:         "candidate",
		}
	}

	return candidates
}
