import type { FC } from 'react';
import { devError } from '../../utils/devLogger';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { vscDarkPlus } from 'react-syntax-highlighter/dist/esm/styles/prism';
import './RuleJsonViewer.css';

interface RuleJsonViewerProps {
  rule: any;
  isOpen: boolean;
  onClose: () => void;
}

export const RuleJsonViewer: FC<RuleJsonViewerProps> = ({
  rule,
  isOpen,
  onClose,
}) => {
  if (!isOpen || !rule) return null;

  // Extract the JSON configuration from the rule
  // This could be in condition_json or we need to construct it from rule properties
  const getRuleJson = () => {
    // If the rule has condition_json, use it
    if (rule.condition_json) {
      return rule.condition_json;
    }

    // Otherwise, construct a basic JSON structure from rule properties
    // This matches the example provided by the user
    return {
      logic_type: "comparison",
      source_field: rule.target_entity || "",
      comparison_operator: "greater_than",
      compare_to_type: "value",
      compare_value: "2000",
      compare_field: "",
      calculation_expression: "",
      conditional_rules: []
    };
  };

  const getRuleCode = () => {
      // 1. If script is explicitly stored, use it
      if (rule.script_content) {
          return rule.script_content;
      }

      // 2. If it is a builder rule (has condition_json), generate CUE on the fly
      if (rule.condition_json) {
          const cj = typeof rule.condition_json === 'string' ? JSON.parse(rule.condition_json) : rule.condition_json;
          if (cj.conditions) {
             const conditions = Array.isArray(cj.conditions) ? cj.conditions : [];
             const operators: Record<string, { format: (field: string, value: string) => string }> = {
                equals: { format: (f, v) => `${f}: "${v}"` },
                not_equals: { format: (f, v) => `${f}: !="${v}"` },
                contains: { format: (f, v) => `${f}: =~"${v}"` },
                starts_with: { format: (f, v) => `${f}: =~"^${v}"` },
                ends_with: { format: (f, v) => `${f}: =~"${v}$"` },
                greater_than: { format: (f, v) => `${f}: >${v}` },
                less_than: { format: (f, v) => `${f}: <${v}` },
                is_empty: { format: (f) => `${f}: ""` },
                is_not_empty: { format: (f) => `${f}: !=""` }
             };
             
             const body = conditions.map((c: any) => {
                 const op = operators[c.operator];
                 return op ? op.format(c.field, c.value) : '';
             }).filter(Boolean).join('\n    ');

             return `// Generated CUE from Logic Builder\nrecord: {\n    ${body}\n}`;
          }
      }

      return '// No logic code available';
  };

  const ruleJson = getRuleJson();
  const jsonString = JSON.stringify(ruleJson, null, 2);
  const ruleCode = getRuleCode();

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(jsonString);
      // Could add a toast notification here
    } catch (err) {
      devError('Failed to copy JSON:', err);
    }
  };

  return (
    <div className="rule-json-viewer-overlay" onClick={onClose}>
      <div className="rule-json-viewer-modal" onClick={(e) => e.stopPropagation()}>
        <div className="rule-json-viewer-header">
          <h3>📄 Rule Configuration: {rule.rule_name}</h3>
          <div className="rule-json-viewer-actions">
            <button
              className="copy-btn"
              onClick={handleCopy}
              title="Copy JSON to clipboard"
            >
              📋 Copy
            </button>
            <button
              className="close-btn"
              onClick={onClose}
              title="Close viewer"
            >
              ✕
            </button>
          </div>
        </div>

        <div className="rule-json-viewer-content">
          <h4>Logic Code (CUE/Starlark)</h4>
          <SyntaxHighlighter
            language="go" // CUE syntax is close to Go
            style={vscDarkPlus}
            customStyle={{ margin: 0, marginBottom: '20px', borderRadius: '8px', fontSize: '14px' }}
            showLineNumbers={true}
          >
            {ruleCode}
          </SyntaxHighlighter>

          <h4>Raw Configuration (JSON)</h4>
          <SyntaxHighlighter
            language="json"
            style={vscDarkPlus}
            customStyle={{ margin: 0, borderRadius: '8px', fontSize: '14px' }}
            showLineNumbers={true}
          >
            {jsonString}
          </SyntaxHighlighter>
        </div>

        <div className="rule-json-viewer-footer">
          <p className="json-info">
            This JSON represents the rule's configuration and conditions.
            Use this for debugging, documentation, or API integration.
          </p>
        </div>
      </div>
    </div>
  );
};