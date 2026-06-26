import { useState, useEffect } from 'react';
import {
  Modal,
  Stack,
  Title,
  Select,
  TextInput,
  Textarea,
  Button,
  Group,
  Text,
  Card,
  Code,
  Grid
} from '@mantine/core';
import { v4 as uuidv4 } from 'uuid';
import { ValidationRule } from './StepConfigPanel';

interface RuleBuilderModalProps {
  onSave: (rule: ValidationRule) => void;
  onCancel: () => void;
}

// Mock API function - replace with real API call
const fetchBusinessObjectSchemas = async () => {
  // Simulate API call to get business object schemas from bundles
  await new Promise(resolve => setTimeout(resolve, 500));
  return {
    client: [
      { value: 'client.name', label: 'Name' },
      { value: 'client.net_worth', label: 'Net Worth' },
      { value: 'client.country', label: 'Country' },
      { value: 'client.age', label: 'Age' },
      { value: 'client.email', label: 'Email' },
    ],
    account: [
      { value: 'account.balance', label: 'Balance' },
      { value: 'account.type', label: 'Account Type' },
      { value: 'account.status', label: 'Status' },
    ],
    trade: [
      { value: 'trade.symbol', label: 'Symbol' },
      { value: 'trade.quantity', label: 'Quantity' },
      { value: 'trade.price', label: 'Price' },
    ],
  };
};

const OPERATORS = [
  { value: 'equals', label: 'Equals' },
  { value: 'not_equals', label: 'Not Equals' },
  { value: 'greater_than', label: 'Greater Than' },
  { value: 'less_than', label: 'Less Than' },
  { value: 'contains', label: 'Contains' },
  { value: 'starts_with', label: 'Starts With' },
  { value: 'ends_with', label: 'Ends With' },
  { value: 'is_empty', label: 'Is Empty' },
  { value: 'is_not_empty', label: 'Is Not Empty' },
  { value: 'in_list', label: 'In List' },
];

export default function RuleBuilderModal({ onSave, onCancel }: RuleBuilderModalProps) {
  const [field, setField] = useState('');
  const [operator, setOperator] = useState('');
  const [value, setValue] = useState('');
  const [message, setMessage] = useState('');
  const [showScriptRule, setShowScriptRule] = useState(false);
  const [script, setScript] = useState('');
  const [businessObjects, setBusinessObjects] = useState<Record<string, Array<{value: string, label: string}>>>({});

  // Load business object schemas and validation rules on mount
  useEffect(() => {
    const loadData = async () => {
      try {
        const schemas = await fetchBusinessObjectSchemas();
        setBusinessObjects(schemas);
      } catch (error) {
        console.error('Failed to load rule builder data:', error);
        // Fallback to empty data
        setBusinessObjects({});
      }
    };

    loadData();
  }, []);

  // Flatten all fields for the dropdown
  const allFields = Object.values(businessObjects).flat();

  const generatePreview = () => {
    if (!field || !operator) return null;

    const rule = {
      field,
      op: operator,
      value: value || undefined,
      message: message || `Validation failed for ${field}`,
    };

    return rule;
  };

  const handleSave = () => {
    if (!field || !operator || !message) {
      return; // Basic validation
    }

    const rule: ValidationRule = {
      id: uuidv4(),
      field,
      operator,
      value,
      message,
    };

    onSave(rule);
  };

  const preview = generatePreview();

  return (
    <Modal
      opened={true}
      onClose={onCancel}
      title="Add Validation Rule"
      size="lg"
    >
      <Stack gap="md">
        <Grid>
          <Grid.Col span={6}>
            <Select
              label="Field"
              placeholder="Select a field"
              data={allFields as Array<{value: string, label: string}>}
              value={field}
              onChange={(value) => setField(value || '')}
              required
            />
          </Grid.Col>
          <Grid.Col span={6}>
            <Select
              label="Operator"
              placeholder="Select operator"
              data={OPERATORS}
              value={operator}
              onChange={(value) => setOperator(value || '')}
              required
            />
          </Grid.Col>
        </Grid>

        <TextInput
          label="Value"
          placeholder="Enter comparison value"
          value={value}
          onChange={(e) => setValue(e.target.value)}
        />

        <TextInput
          label="Error Message"
          placeholder="Message shown when validation fails"
          value={message}
          onChange={(e) => setMessage(e.target.value)}
          required
        />

        <Button
          variant="subtle"
          onClick={() => setShowScriptRule(!showScriptRule)}
        >
          {showScriptRule ? 'Hide' : 'Show'} Advanced Script Rule
        </Button>

        {showScriptRule && (
          <div>
            <Text size="sm" fw={500} mb="xs">Custom JavaScript Rule</Text>
            <Text size="xs" c="dimmed" mb="xs">
              For complex validation logic. Return {'{valid: true}'} or {'{valid: false, message: "error"}'}
            </Text>
            <Textarea
              placeholder="return client.net_worth > 100000 ? {valid: true} : {valid: false, message: 'Net worth too low'};"
              value={script}
              onChange={(e) => setScript(e.target.value)}
              minRows={3}
            />
          </div>
        )}

        {preview && (
          <Card withBorder>
            <Title order={6} mb="xs">Rule Preview</Title>
            <Code block>
              {JSON.stringify(preview, null, 2)}
            </Code>
          </Card>
        )}

        <Group justify="flex-end" mt="md">
          <Button variant="light" onClick={onCancel}>
            Cancel
          </Button>
          <Button onClick={handleSave} disabled={!field || !operator || !message}>
            Add Rule
          </Button>
        </Group>
      </Stack>
    </Modal>
  );
}