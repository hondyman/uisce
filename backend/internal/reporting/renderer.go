package reporting

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Renderer handles report rendering to various output formats
type Renderer struct {
	cubeClient *CubeClient
}

// NewRenderer creates a new renderer
func NewRenderer(cubeClient *CubeClient) *Renderer {
	return &Renderer{cubeClient: cubeClient}
}

// RenderResult contains the rendered report output
type RenderResult struct {
	Data     []byte          `json:"-"`
	URL      string          `json:"url,omitempty"`
	Metadata json.RawMessage `json:"metadata"`
}

// Render generates a report in the specified format
func (r *Renderer) Render(ctx context.Context, tenantID, datasourceID uuid.UUID, layout *ReportLayout, parameters json.RawMessage, format string) (*RenderResult, error) {
	// Parse parameters
	var params map[string]interface{}
	if len(parameters) > 0 {
		if err := json.Unmarshal(parameters, &params); err != nil {
			return nil, fmt.Errorf("failed to parse parameters: %w", err)
		}
	}

	// Fetch data for all data bindings
	dataResults := make(map[string]*CubeResult)
	for name, binding := range layout.DataBindings {
		// Check conditional binding
		if binding.Conditional != nil {
			paramVal, ok := params[binding.Conditional.Parameter]
			if !ok || paramVal != binding.Conditional.Equals {
				continue // Skip this binding
			}
		}

		// Build and execute query
		query, err := BuildQueryFromBinding(&binding, params)
		if err != nil {
			return nil, fmt.Errorf("failed to build query for %s: %w", name, err)
		}

		result, err := r.cubeClient.ExecuteQuery(ctx, query, tenantID, datasourceID)
		if err != nil {
			return nil, fmt.Errorf("failed to execute query for %s: %w", name, err)
		}

		dataResults[name] = result
	}

	// Render based on format
	var data []byte
	var err error

	switch format {
	case "pdf":
		data, err = r.renderPDF(ctx, layout, dataResults, params)
	case "html":
		data, err = r.renderHTML(ctx, layout, dataResults, params)
	case "excel":
		data, err = r.renderExcel(ctx, layout, dataResults, params)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to render %s: %w", format, err)
	}

	// Build metadata
	metadata := map[string]interface{}{
		"format":      format,
		"rendered_at": time.Now().Format(time.RFC3339),
		"page_count":  1, // Would be calculated properly
		"data_rows":   countTotalRows(dataResults),
	}
	metadataJSON, _ := json.Marshal(metadata)

	return &RenderResult{
		Data:     data,
		Metadata: metadataJSON,
	}, nil
}

// renderPDF generates a PDF report
func (r *Renderer) renderPDF(ctx context.Context, layout *ReportLayout, data map[string]*CubeResult, params map[string]interface{}) ([]byte, error) {
	// This would use a PDF library like gofpdf or unidoc
	// For now, returning a placeholder

	var buf strings.Builder

	// Build PDF content structure
	buf.WriteString("%PDF-1.4\n")

	// Header
	if layout.Layout.Header != nil {
		for _, elem := range layout.Layout.Header.Elements {
			if elem.Type == "text" {
				content := resolveExpression(elem.Content, data, params)
				buf.WriteString(fmt.Sprintf("Header: %s\n", content))
			}
		}
	}

	// Body sections
	for _, section := range layout.Layout.Body.Sections {
		buf.WriteString(fmt.Sprintf("\nSection: %s\n", section.Title))

		// Get data for this section
		var sectionData *CubeResult
		if section.DataBinding != "" {
			sectionData = data[section.DataBinding]
		}

		switch section.Type {
		case "summary":
			renderPDFSummary(&buf, section, sectionData, params)
		case "table":
			renderPDFTable(&buf, section, sectionData, params)
		case "chart":
			renderPDFChart(&buf, section, sectionData, params)
		}
	}

	// Footer
	if layout.Layout.Footer != nil {
		for _, elem := range layout.Layout.Footer.Elements {
			if elem.Type == "text" {
				content := resolveExpression(elem.Content, data, params)
				buf.WriteString(fmt.Sprintf("Footer: %s\n", content))
			}
		}
	}

	return []byte(buf.String()), nil
}

// renderHTML generates an HTML report
func (r *Renderer) renderHTML(ctx context.Context, layout *ReportLayout, data map[string]*CubeResult, params map[string]interface{}) ([]byte, error) {
	var buf strings.Builder

	buf.WriteString("<!DOCTYPE html>\n<html>\n<head>\n")
	buf.WriteString(fmt.Sprintf("<title>%s</title>\n", layout.Metadata.DisplayName))
	buf.WriteString("<style>\n")
	buf.WriteString(`
		body { font-family: Arial, sans-serif; margin: 40px; }
		.header { border-bottom: 2px solid #333; padding-bottom: 20px; margin-bottom: 30px; }
		.section { margin-bottom: 30px; }
		.section-title { font-size: 18px; font-weight: bold; margin-bottom: 15px; color: #333; }
		table { width: 100%; border-collapse: collapse; }
		th, td { padding: 8px 12px; border: 1px solid #ddd; text-align: left; }
		th { background-color: #f5f5f5; font-weight: bold; }
		.kpi-card { display: inline-block; padding: 20px; margin: 10px; background: #f8f9fa; border-radius: 8px; min-width: 150px; }
		.kpi-title { font-size: 12px; color: #666; }
		.kpi-value { font-size: 24px; font-weight: bold; color: #333; }
		.kpi-change.positive { color: #16a34a; }
		.kpi-change.negative { color: #dc2626; }
		.footer { margin-top: 40px; padding-top: 20px; border-top: 1px solid #ddd; color: #666; font-size: 12px; }
	`)
	buf.WriteString("</style>\n</head>\n<body>\n")

	// Header
	if layout.Layout.Header != nil {
		buf.WriteString("<div class=\"header\">\n")
		for _, elem := range layout.Layout.Header.Elements {
			renderHTMLElement(&buf, elem, data, params)
		}
		buf.WriteString("</div>\n")
	}

	// Body sections
	for _, section := range layout.Layout.Body.Sections {
		buf.WriteString("<div class=\"section\">\n")

		if section.Title != "" {
			buf.WriteString(fmt.Sprintf("<div class=\"section-title\">%s</div>\n", section.Title))
		}

		var sectionData *CubeResult
		if section.DataBinding != "" {
			sectionData = data[section.DataBinding]
		}

		switch section.Type {
		case "summary":
			renderHTMLSummary(&buf, section, sectionData, params)
		case "table":
			renderHTMLTable(&buf, section, sectionData, params, layout.ConditionalStyles)
		case "chart":
			renderHTMLChart(&buf, section, sectionData, params)
		case "text":
			for _, elem := range section.Elements {
				renderHTMLElement(&buf, elem, data, params)
			}
		}

		buf.WriteString("</div>\n")
	}

	// Footer
	if layout.Layout.Footer != nil {
		buf.WriteString("<div class=\"footer\">\n")
		for _, elem := range layout.Layout.Footer.Elements {
			renderHTMLElement(&buf, elem, data, params)
		}
		buf.WriteString("</div>\n")
	}

	buf.WriteString("</body>\n</html>")

	return []byte(buf.String()), nil
}

// renderExcel generates an Excel report
func (r *Renderer) renderExcel(ctx context.Context, layout *ReportLayout, data map[string]*CubeResult, params map[string]interface{}) ([]byte, error) {
	// This would use excelize library
	// For now, returning a simple CSV-like format

	var buf strings.Builder

	// Write each section
	for _, section := range layout.Layout.Body.Sections {
		buf.WriteString(fmt.Sprintf("## %s\n", section.Title))

		var sectionData *CubeResult
		if section.DataBinding != "" {
			sectionData = data[section.DataBinding]
		}

		if section.Type == "table" && len(section.Columns) > 0 && sectionData != nil {
			// Header row
			var headers []string
			for _, col := range section.Columns {
				headers = append(headers, col.Label)
			}
			buf.WriteString(strings.Join(headers, "\t") + "\n")

			// Data rows
			for _, row := range sectionData.Data {
				var values []string
				for _, col := range section.Columns {
					key := col.Dimension
					if key == "" {
						key = col.Measure
					}
					val := fmt.Sprintf("%v", row[key])
					values = append(values, val)
				}
				buf.WriteString(strings.Join(values, "\t") + "\n")
			}
		}

		buf.WriteString("\n")
	}

	return []byte(buf.String()), nil
}

// ============================================================================
// HELPER RENDER FUNCTIONS
// ============================================================================

func renderPDFSummary(buf *strings.Builder, section ReportSection, data *CubeResult, params map[string]interface{}) {
	for _, elem := range section.Elements {
		if elem.Type == "kpiCard" {
			title := resolveExpression(elem.Title, nil, params)
			value := resolveDataExpression(elem.Value, data, params)
			buf.WriteString(fmt.Sprintf("  KPI: %s = %s\n", title, value))
		}
	}
}

func renderPDFTable(buf *strings.Builder, section ReportSection, data *CubeResult, params map[string]interface{}) {
	if data == nil || len(data.Data) == 0 {
		buf.WriteString("  (No data)\n")
		return
	}

	// Headers
	var headers []string
	for _, col := range section.Columns {
		headers = append(headers, col.Label)
	}
	buf.WriteString(fmt.Sprintf("  | %s |\n", strings.Join(headers, " | ")))

	// Data rows
	for _, row := range data.Data {
		var values []string
		for _, col := range section.Columns {
			key := col.Dimension
			if key == "" {
				key = col.Measure
			}
			val := formatValue(row[key], col.Format)
			values = append(values, val)
		}
		buf.WriteString(fmt.Sprintf("  | %s |\n", strings.Join(values, " | ")))
	}
}

func renderPDFChart(buf *strings.Builder, section ReportSection, data *CubeResult, params map[string]interface{}) {
	buf.WriteString(fmt.Sprintf("  [Chart: %s - %s]\n", section.ChartConfig.Type, section.Title))
}

func renderHTMLElement(buf *strings.Builder, elem Element, data map[string]*CubeResult, params map[string]interface{}) {
	switch elem.Type {
	case "text":
		content := resolveExpression(elem.Content, data, params)
		style := ""
		if len(elem.Style) > 0 {
			var styleMap map[string]interface{}
			json.Unmarshal(elem.Style, &styleMap)
			for k, v := range styleMap {
				style += fmt.Sprintf("%s: %v; ", k, v)
			}
		}
		buf.WriteString(fmt.Sprintf("<div style=\"%s\">%s</div>\n", style, content))
	case "image":
		src := resolveExpression(elem.Src, data, params)
		buf.WriteString(fmt.Sprintf("<img src=\"%s\" style=\"max-height: 60px;\" />\n", src))
	case "pageNumber":
		// Page numbers handled in PDF, just show placeholder in HTML
		buf.WriteString("<span class=\"page-number\">Page 1</span>\n")
	}
}

func renderHTMLSummary(buf *strings.Builder, section ReportSection, data *CubeResult, params map[string]interface{}) {
	buf.WriteString("<div class=\"kpi-container\">\n")

	for _, elem := range section.Elements {
		if elem.Type == "kpiCard" {
			title := resolveExpression(elem.Title, nil, params)
			value := resolveDataExpression(elem.Value, data, params)

			changeClass := "neutral"
			if elem.ChangeType != "" {
				changeType := resolveDataExpression(elem.ChangeType, data, params)
				if changeType == "positive" {
					changeClass = "positive"
				} else if changeType == "negative" {
					changeClass = "negative"
				}
			}

			buf.WriteString("<div class=\"kpi-card\">\n")
			buf.WriteString(fmt.Sprintf("  <div class=\"kpi-title\">%s</div>\n", title))
			buf.WriteString(fmt.Sprintf("  <div class=\"kpi-value\">%s</div>\n", value))

			if elem.Change != "" {
				change := resolveDataExpression(elem.Change, data, params)
				buf.WriteString(fmt.Sprintf("  <div class=\"kpi-change %s\">%s</div>\n", changeClass, change))
			}

			if elem.Benchmark != "" {
				benchmark := resolveDataExpression(elem.Benchmark, data, params)
				buf.WriteString(fmt.Sprintf("  <div class=\"kpi-benchmark\">%s: %s</div>\n", elem.BenchmarkLabel, benchmark))
			}

			buf.WriteString("</div>\n")
		}
	}

	buf.WriteString("</div>\n")
}

func renderHTMLTable(buf *strings.Builder, section ReportSection, data *CubeResult, params map[string]interface{}, conditionalStyles map[string]ConditionalStyle) {
	buf.WriteString("<table>\n<thead>\n<tr>\n")

	// Header row
	for _, col := range section.Columns {
		alignment := col.Alignment
		if alignment == "" {
			alignment = "left"
		}
		buf.WriteString(fmt.Sprintf("<th style=\"text-align: %s;\">%s</th>\n", alignment, col.Label))
	}
	buf.WriteString("</tr>\n</thead>\n<tbody>\n")

	// Data rows
	if data != nil {
		for _, row := range data.Data {
			buf.WriteString("<tr>\n")
			for _, col := range section.Columns {
				key := col.Dimension
				if key == "" {
					key = col.Measure
				}

				value := row[key]
				formattedValue := formatValue(value, col.Format)

				// Apply conditional styling
				style := ""
				if col.ConditionalStyle != "" {
					if cs, ok := conditionalStyles[col.ConditionalStyle]; ok {
						if numVal, ok := toFloat64(value); ok {
							if numVal > 0 {
								for k, v := range cs.Positive {
									style += fmt.Sprintf("%s: %s; ", k, v)
								}
							} else if numVal < 0 {
								for k, v := range cs.Negative {
									style += fmt.Sprintf("%s: %s; ", k, v)
								}
							}
						}
					}
				}

				alignment := col.Alignment
				if alignment == "" {
					alignment = "left"
				}
				style += fmt.Sprintf("text-align: %s;", alignment)

				buf.WriteString(fmt.Sprintf("<td style=\"%s\">%s</td>\n", style, formattedValue))
			}
			buf.WriteString("</tr>\n")
		}
	}

	buf.WriteString("</tbody>\n</table>\n")
}

func renderHTMLChart(buf *strings.Builder, section ReportSection, data *CubeResult, params map[string]interface{}) {
	// In a real implementation, this would use Chart.js or similar
	buf.WriteString(fmt.Sprintf("<div class=\"chart-placeholder\" style=\"height: 300px; background: #f5f5f5; display: flex; align-items: center; justify-content: center;\">\n"))
	buf.WriteString(fmt.Sprintf("  <span>📊 %s Chart: %s</span>\n", section.ChartConfig.Type, section.Title))
	buf.WriteString("</div>\n")
}

// ============================================================================
// EXPRESSION RESOLUTION
// ============================================================================

// resolveExpression handles template expressions like {{tenant.name}}
func resolveExpression(expr string, data map[string]*CubeResult, params map[string]interface{}) string {
	result := expr

	// Handle parameter references: {{parameters.name}}
	for key, value := range params {
		placeholder := fmt.Sprintf("{{parameters.%s}}", key)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
	}

	// Handle date formatting: {{value | date:'format'}}
	// Simplified handling for now
	if strings.Contains(result, "| date:") {
		// Extract and format date
		result = strings.ReplaceAll(result, "| date:'MMMM d, yyyy'", "")
		result = strings.TrimSpace(result)
		result = strings.Trim(result, "{}")
		if t, err := time.Parse(time.RFC3339, result); err == nil {
			result = t.Format("January 2, 2006")
		}
	}

	return result
}

// resolveDataExpression handles data expressions like {{data.total_value | currency}}
func resolveDataExpression(expr string, data *CubeResult, params map[string]interface{}) string {
	if data == nil || len(data.Data) == 0 {
		return expr
	}

	result := expr
	row := data.Data[0] // Use first row for summary values

	// Extract field reference
	// Format: {{data.field_name | format}} or just {{data.field_name}}
	if strings.HasPrefix(expr, "{{data.") && strings.HasSuffix(expr, "}}") {
		inner := strings.TrimPrefix(expr, "{{data.")
		inner = strings.TrimSuffix(inner, "}}")

		// Check for format
		parts := strings.Split(inner, " | ")
		fieldName := parts[0]
		format := ""
		if len(parts) > 1 {
			format = strings.TrimSpace(parts[1])
		}

		if value, ok := row[fieldName]; ok {
			result = formatValue(value, format)
		}
	}

	return result
}

// formatValue formats a value according to the specified format
func formatValue(value interface{}, format string) string {
	if value == nil {
		return ""
	}

	switch format {
	case "currency":
		if f, ok := toFloat64(value); ok {
			if f >= 0 {
				return fmt.Sprintf("$%.2f", f)
			}
			return fmt.Sprintf("-$%.2f", -f)
		}
	case "percent":
		if f, ok := toFloat64(value); ok {
			return fmt.Sprintf("%.2f%%", f*100)
		}
	case "number":
		if f, ok := toFloat64(value); ok {
			return fmt.Sprintf("%.2f", f)
		}
	case "date":
		if s, ok := value.(string); ok {
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				return t.Format("2006-01-02")
			}
		}
	}

	return fmt.Sprintf("%v", value)
}

// toFloat64 converts a value to float64
func toFloat64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case json.Number:
		f, err := v.Float64()
		return f, err == nil
	}
	return 0, false
}

// countTotalRows counts total data rows across all results
func countTotalRows(data map[string]*CubeResult) int {
	count := 0
	for _, result := range data {
		if result != nil {
			count += len(result.Data)
		}
	}
	return count
}
