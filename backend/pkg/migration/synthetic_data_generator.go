package migration

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"
)

// SyntheticDataGenerator creates artificial training data for fine-tuning migration models
type SyntheticDataGenerator struct {
	DB *sql.DB
}

func NewSyntheticDataGenerator(db *sql.DB) *SyntheticDataGenerator {
	return &SyntheticDataGenerator{DB: db}
}

// SyntheticExample represents a generated training example
type SyntheticExample struct {
	ID             string                 `json:"id"`
	SourceCode     string                 `json:"sourceCode"`
	SourceLanguage string                 `json:"sourceLanguage"`
	BusinessIntent map[string]interface{} `json:"businessIntent"`
	TitanDAG       map[string]interface{} `json:"titanDag"`
	OPARego        string                 `json:"opaRego,omitempty"`
}

// BusinessRuleTemplate defines a parameterized business rule pattern
type BusinessRuleTemplate struct {
	Name         string
	Description  string
	CodeTemplate string
	DAGTemplate  map[string]interface{}
	RegoTemplate string
	Parameters   []string
}

// Pre-defined business rule templates
var businessRuleTemplates = []BusinessRuleTemplate{
	{
		Name:        "ThresholdApproval",
		Description: "Require approval for values exceeding a threshold",
		Parameters:  []string{"field", "threshold", "approver"},
		CodeTemplate: `
public void process{{.Field}}(Transaction tx) {
    if (tx.get{{.Field}}() > {{.Threshold}}) {
        // Requires {{.Approver}} approval
        requestApproval(tx, "{{.Approver}}");
    } else {
        autoApprove(tx);
    }
}`,
		DAGTemplate: map[string]interface{}{
			"nodes": map[string]interface{}{
				"start": map[string]interface{}{
					"id":   "start",
					"type": "BRANCH",
					"config": map[string]interface{}{
						"conditionField": "{{.field}}",
						"operator":       "gt",
						"value":          "{{.threshold}}",
						"trueNext":       "approval",
						"falseNext":      "auto_approve",
					},
				},
				"approval": map[string]interface{}{
					"id":   "approval",
					"type": "ACTIVITY",
					"config": map[string]interface{}{
						"activityName":       "ActivityUserInteraction",
						"viewDefinitionName": "{{.approver}}_approval_form",
					},
					"next": "end",
				},
				"auto_approve": map[string]interface{}{
					"id":   "auto_approve",
					"type": "ACTIVITY",
					"config": map[string]interface{}{
						"activityName": "SendEmail",
						"subject":      "Auto-Approved",
					},
					"next": "end",
				},
			},
			"startNodeId": "start",
		},
	},
	{
		Name:        "ComplianceCheck",
		Description: "Validate against compliance rules before processing",
		Parameters:  []string{"checkType", "jurisdiction"},
		CodeTemplate: `
public boolean validateCompliance(Trade trade) {
    ComplianceResult result = complianceService.check{{.CheckType}}(trade, "{{.Jurisdiction}}");
    if (!result.isCompliant()) {
        throw new ComplianceViolationException(result.getViolations());
    }
    return true;
}`,
		DAGTemplate: map[string]interface{}{
			"nodes": map[string]interface{}{
				"compliance_check": map[string]interface{}{
					"id":   "compliance_check",
					"type": "ACTIVITY",
					"config": map[string]interface{}{
						"activityName": "ActivityCheckCompliance",
						"tradeType":    "{{.checkType}}",
						"jurisdiction": "{{.jurisdiction}}",
					},
					"next": "process",
				},
				"process": map[string]interface{}{
					"id":   "process",
					"type": "ACTIVITY",
					"config": map[string]interface{}{
						"activityName": "ProcessTrade",
					},
				},
			},
			"startNodeId": "compliance_check",
		},
		RegoTemplate: `
package titan.compliance.{{.jurisdiction}}

default allow = false

allow {
    input.checkType == "{{.checkType}}"
    not violation
}

violation {
    input.amount > 1000000
}`,
	},
	{
		Name:        "NotificationOnEvent",
		Description: "Send notification when specific condition is met",
		Parameters:  []string{"event", "recipient", "channel"},
		CodeTemplate: `
public void on{{.Event}}(Event event) {
    NotificationService.send(
        "{{.Channel}}",
        "{{.Recipient}}",
        "Event: " + event.getDescription()
    );
}`,
		DAGTemplate: map[string]interface{}{
			"nodes": map[string]interface{}{
				"notify": map[string]interface{}{
					"id":   "notify",
					"type": "ACTIVITY",
					"config": map[string]interface{}{
						"activityName": "SendNotification",
						"channel":      "{{.channel}}",
						"to":           "{{.recipient}}",
						"template":     "{{.event}}_notification",
					},
				},
			},
			"startNodeId": "notify",
		},
	},
	{
		Name:        "SequentialApprovals",
		Description: "Multiple approval stages in sequence",
		Parameters:  []string{"stage1", "stage2", "stage3"},
		CodeTemplate: `
public void processHighValue(Order order) {
    // Stage 1: {{.Stage1}}
    if (!approve(order, "{{.Stage1}}")) return;
    
    // Stage 2: {{.Stage2}}
    if (!approve(order, "{{.Stage2}}")) return;
    
    // Stage 3: {{.Stage3}}
    if (!approve(order, "{{.Stage3}}")) return;
    
    finalizeOrder(order);
}`,
		DAGTemplate: map[string]interface{}{
			"nodes": map[string]interface{}{
				"stage1": map[string]interface{}{
					"id":   "stage1",
					"type": "ACTIVITY",
					"config": map[string]interface{}{
						"activityName":       "ActivityUserInteraction",
						"viewDefinitionName": "{{.stage1}}_approval",
						"title":              "{{.stage1}} Approval",
					},
					"next": "stage2",
				},
				"stage2": map[string]interface{}{
					"id":   "stage2",
					"type": "ACTIVITY",
					"config": map[string]interface{}{
						"activityName":       "ActivityUserInteraction",
						"viewDefinitionName": "{{.stage2}}_approval",
						"title":              "{{.stage2}} Approval",
					},
					"next": "stage3",
				},
				"stage3": map[string]interface{}{
					"id":   "stage3",
					"type": "ACTIVITY",
					"config": map[string]interface{}{
						"activityName":       "ActivityUserInteraction",
						"viewDefinitionName": "{{.stage3}}_approval",
						"title":              "{{.stage3}} Approval",
					},
					"next": "finalize",
				},
				"finalize": map[string]interface{}{
					"id":   "finalize",
					"type": "ACTIVITY",
					"config": map[string]interface{}{
						"activityName": "FinalizeOrder",
					},
				},
			},
			"startNodeId": "stage1",
		},
	},
}

// Parameter value pools for random generation
var parameterPools = map[string][]string{
	"field":        {"Amount", "Value", "Quantity", "Price", "Total", "Balance"},
	"threshold":    {"1000", "5000", "10000", "50000", "100000", "1000000"},
	"approver":     {"manager", "director", "vp", "compliance_officer", "risk_manager"},
	"checkType":    {"equity", "fixed_income", "derivatives", "forex", "commodities"},
	"jurisdiction": {"US", "EU", "UK", "APAC", "GLOBAL"},
	"event":        {"TradeExecuted", "OrderFilled", "SettlementComplete", "RiskAlert"},
	"recipient":    {"operations@company.com", "{{context.submitter}}", "alerts@company.com"},
	"channel":      {"email", "slack", "sms", "webhook"},
	"stage1":       {"Compliance", "Legal", "Operations"},
	"stage2":       {"Risk", "Finance", "Treasury"},
	"stage3":       {"Executive", "Board", "Audit"},
}

// GenerateBatch creates a batch of synthetic training examples
func (g *SyntheticDataGenerator) GenerateBatch(ctx context.Context, count int, language string) ([]SyntheticExample, error) {
	rand.Seed(time.Now().UnixNano())

	examples := make([]SyntheticExample, 0, count)

	for i := 0; i < count; i++ {
		// Pick random template
		template := businessRuleTemplates[rand.Intn(len(businessRuleTemplates))]

		// Generate random parameter values
		params := make(map[string]string)
		for _, param := range template.Parameters {
			pool := parameterPools[param]
			if len(pool) > 0 {
				params[param] = pool[rand.Intn(len(pool))]
			}
		}

		// Instantiate code
		code := template.CodeTemplate
		for k, v := range params {
			code = replaceAll(code, "{{."+capitalize(k)+"}}", v)
		}

		// Instantiate DAG
		dagBytes, _ := json.Marshal(template.DAGTemplate)
		dagStr := string(dagBytes)
		for k, v := range params {
			dagStr = replaceAll(dagStr, "{{."+k+"}}", v)
		}
		var dag map[string]interface{}
		_ = json.Unmarshal([]byte(dagStr), &dag)

		// Instantiate Rego
		rego := template.RegoTemplate
		for k, v := range params {
			rego = replaceAll(rego, "{{."+k+"}}", v)
		}

		// Build intent
		intent := map[string]interface{}{
			"summary":       fmt.Sprintf("%s: %s", template.Name, template.Description),
			"ruleType":      template.Name,
			"preconditions": []interface{}{},
			"actions":       []interface{}{},
		}
		for k, v := range params {
			intent[k] = v
		}

		example := SyntheticExample{
			ID:             fmt.Sprintf("synth_%d_%d", time.Now().Unix(), i),
			SourceCode:     code,
			SourceLanguage: language,
			BusinessIntent: intent,
			TitanDAG:       dag,
			OPARego:        rego,
		}

		examples = append(examples, example)
	}

	return examples, nil
}

// SaveToKnowledgeBase stores generated examples in the Knowledge Base for RAG
func (g *SyntheticDataGenerator) SaveToKnowledgeBase(ctx context.Context, examples []SyntheticExample) error {
	query := `
		INSERT INTO titan_knowledge_base (category, name, description, content, tags)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT DO NOTHING
	`

	for _, ex := range examples {
		content := map[string]interface{}{
			"sourceCode": ex.SourceCode,
			"dag":        ex.TitanDAG,
			"rego":       ex.OPARego,
		}
		contentJSON, _ := json.Marshal(content)

		tags := []string{"synthetic", ex.SourceLanguage}
		if ruleType, ok := ex.BusinessIntent["ruleType"].(string); ok {
			tags = append(tags, ruleType)
		}

		_, err := g.DB.ExecContext(ctx, query,
			"example",
			fmt.Sprintf("Synthetic: %s", ex.BusinessIntent["summary"]),
			ex.BusinessIntent["summary"],
			contentJSON,
			tags,
		)
		if err != nil {
			return fmt.Errorf("failed to save synthetic example: %w", err)
		}
	}

	return nil
}

// ExportForFineTuning exports examples in JSONL format for model fine-tuning
func (g *SyntheticDataGenerator) ExportForFineTuning(examples []SyntheticExample) (string, error) {
	var result string
	for _, ex := range examples {
		line := map[string]interface{}{
			"input": map[string]interface{}{
				"code":     ex.SourceCode,
				"language": ex.SourceLanguage,
			},
			"output": map[string]interface{}{
				"intent": ex.BusinessIntent,
				"dag":    ex.TitanDAG,
				"rego":   ex.OPARego,
			},
		}
		lineBytes, _ := json.Marshal(line)
		result += string(lineBytes) + "\n"
	}
	return result, nil
}

// Helper functions
func replaceAll(s, old, new string) string {
	for {
		replaced := stringReplace(s, old, new)
		if replaced == s {
			return s
		}
		s = replaced
	}
}

func stringReplace(s, old, new string) string {
	for i := 0; i < len(s)-len(old)+1; i++ {
		if s[i:i+len(old)] == old {
			return s[:i] + new + s[i+len(old):]
		}
	}
	return s
}

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	if s[0] >= 'a' && s[0] <= 'z' {
		return string(s[0]-32) + s[1:]
	}
	return s
}
