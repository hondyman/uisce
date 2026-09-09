import React, { useState } from 'react';
import { Box, Container, Typography, Paper, Button, Tabs, Tab, TextField } from '@mui/material';
import AdvancedConditionBuilder, { ConditionGroup, ConditionNode, EntityDefinition } from '../components/ExpressionBuilder/AdvancedConditionBuilder';
import { evaluateRuleWasm } from '../rules/wasmRuntime';

// Converts the editor's ConditionNode shape into the wire format
// internal/rules/vm.RuleNode.UnmarshalJSON expects (flat "type" +
// sibling fields, not nested under a "Condition"/"Group" key - see
// backend/internal/rules/vm/ast.go). Structural discrimination
// ("conditions" in node) rather than trusting node.type, since Condition
// nodes from the builder don't always set an explicit type.
function toRuleNode(node: ConditionNode): unknown {
  if ('conditions' in node) {
    return {
      type: 'group',
      id: node.id,
      operator: node.operator,
      conditions: node.conditions.map(toRuleNode),
    };
  }
  return {
    type: 'condition',
    id: node.id,
    field: node.fieldPath || node.field,
    operator: node.operator,
    value: node.value,
  };
}

// Mock Data
const MOCK_ENTITIES: EntityDefinition[] = [
  {
    name: 'order',
    label: 'Order',
    fields: [
      { name: 'id', label: 'Order ID', type: 'string' },
      { name: 'total', label: 'Total Amount', type: 'number' },
      { name: 'status', label: 'Status', type: 'enum', enumValues: ['pending', 'shipped', 'delivered', 'cancelled'] },
      { name: 'created_at', label: 'Created At', type: 'date' },
      { name: 'is_gift', label: 'Is Gift', type: 'boolean' },
    ],
    relationships: [
      { name: 'customer', targetEntity: 'customer', type: 'many-to-one', label: 'Customer' },
      { name: 'line_items', targetEntity: 'line_item', type: 'one-to-many', label: 'Line Items' },
    ]
  },
  {
    name: 'customer',
    label: 'Customer',
    fields: [
      { name: 'id', label: 'Customer ID', type: 'string' },
      { name: 'name', label: 'Name', type: 'string' },
      { name: 'email', label: 'Email', type: 'string' },
      { name: 'vip_status', label: 'VIP Status', type: 'boolean' },
      { name: 'signup_date', label: 'Signup Date', type: 'date' },
    ],
    relationships: [
      { name: 'orders', targetEntity: 'order', type: 'one-to-many', label: 'Orders' },
    ]
  },
  {
    name: 'line_item',
    label: 'Line Item',
    fields: [
      { name: 'id', label: 'ID', type: 'string' },
      { name: 'product_name', label: 'Product Name', type: 'string' },
      { name: 'quantity', label: 'Quantity', type: 'number' },
      { name: 'price', label: 'Price', type: 'number' },
    ],
    relationships: [
      { name: 'order', targetEntity: 'order', type: 'many-to-one', label: 'Order' },
    ]
  }
];

const INITIAL_RULE: ConditionGroup = {
  id: 'root',
  type: 'group',
  operator: 'AND',
  conditions: [
    {
      id: 'c1',
      type: 'condition',
      field: 'total',
      operator: 'greater_than',
      value: 100
    }
  ]
};

const SAMPLE_CONTEXT = { total: 150, status: 'pending', is_gift: false };

const AdvancedRuleBuilderPage: React.FC = () => {
  const [rule, setRule] = useState<ConditionGroup>(INITIAL_RULE);
  const [tabIndex, setTabIndex] = useState(0);
  const [contextJson, setContextJson] = useState(JSON.stringify(SAMPLE_CONTEXT, null, 2));
  const [evalResult, setEvalResult] = useState<{ result?: boolean; error?: string } | null>(null);
  const [evaluating, setEvaluating] = useState(false);

  const runBackendPreview = async () => {
    setEvaluating(true);
    setEvalResult(null);
    try {
      const ctx = JSON.parse(contextJson);
      const ruleNode = toRuleNode(rule);
      const result = await evaluateRuleWasm(ruleNode, ctx);
      setEvalResult({ result });
    } catch (err) {
      setEvalResult({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      setEvaluating(false);
    }
  };

  return (
    <Container maxWidth="lg" sx={{ py: 4 }}>
      <Typography variant="h4" gutterBottom>
        Advanced Rule Builder
      </Typography>
      <Typography variant="body1" color="textSecondary" paragraph>
        Build complex validation rules with nested conditions, cross-entity traversal, and type-aware operators.
      </Typography>

      <Paper sx={{ p: 3, mb: 4 }}>
        <AdvancedConditionBuilder
          value={rule}
          onChange={setRule}
          entities={MOCK_ENTITIES}
          primaryEntity="order"
          enableCrossEntity={true}
          enableDragDrop={true}
          showValidation={true}
        />
      </Paper>

      <Paper sx={{ p: 2 }}>
        <Tabs value={tabIndex} onChange={(_, v) => setTabIndex(v)} sx={{ mb: 2 }}>
          <Tab label="JSON Output" />
          <Tab label="Backend Preview" />
        </Tabs>

        {tabIndex === 0 && (
          <Box sx={{ bgcolor: '#f5f5f5', p: 2, borderRadius: 1, overflow: 'auto' }}>
            <pre style={{ margin: 0 }}>{JSON.stringify(rule, null, 2)}</pre>
          </Box>
        )}

        {tabIndex === 1 && (
          <Box sx={{ p: 2 }}>
            <Typography variant="body2" color="textSecondary" paragraph>
              Runs this rule against internal/rules/vm.AdvancedEvaluator via the same
              rule_engine.wasm build the browser live-preview panel uses (see
              frontend/src/rules/wasmRuntime.ts) - the real evaluator, not a claim about it.
            </Typography>
            <TextField
              label="Sample data (JSON)"
              value={contextJson}
              onChange={(e) => setContextJson(e.target.value)}
              multiline
              minRows={4}
              fullWidth
              sx={{ mb: 2, fontFamily: 'monospace' }}
            />
            <Button variant="contained" onClick={runBackendPreview} disabled={evaluating}>
              {evaluating ? 'Evaluating...' : 'Evaluate against sample data'}
            </Button>
            {evalResult && (
              <Box sx={{ mt: 2 }}>
                {evalResult.error ? (
                  <Typography color="error">Error: {evalResult.error}</Typography>
                ) : (
                  <Typography sx={{ fontWeight: 'bold' }} color={evalResult.result ? 'success.main' : 'text.secondary'}>
                    Result: {evalResult.result ? 'PASS' : 'FAIL'}
                  </Typography>
                )}
              </Box>
            )}
          </Box>
        )}
      </Paper>
    </Container>
  );
};

export default AdvancedRuleBuilderPage;
