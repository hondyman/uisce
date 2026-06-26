import React, { useState } from 'react';
import { Box, Container, Typography, Paper, Button, Tabs, Tab } from '@mui/material';
import AdvancedConditionBuilder, { ConditionGroup, EntityDefinition } from '../components/ExpressionBuilder/AdvancedConditionBuilder';

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

const AdvancedRuleBuilderPage: React.FC = () => {
  const [rule, setRule] = useState<ConditionGroup>(INITIAL_RULE);
  const [tabIndex, setTabIndex] = useState(0);

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
            <Typography variant="body2" color="textSecondary">
              This JSON structure is directly compatible with the backend <code>AdvancedEvaluator</code>.
            </Typography>
          </Box>
        )}
      </Paper>
    </Container>
  );
};

export default AdvancedRuleBuilderPage;
