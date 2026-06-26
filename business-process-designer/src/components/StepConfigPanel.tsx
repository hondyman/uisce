import { useState } from 'react';
import { Stack, Title, Text, Select, Button, TextInput, Radio, Group, Table, ActionIcon } from '@mantine/core';
import { IconPlus, IconTrash } from '@tabler/icons-react';
import { Node } from 'reactflow';
import RuleBuilderModal from './RuleBuilderModal';

export interface ValidationRule {
  id: string;
  field: string;
  operator: string;
  value: string;
  message: string;
}

interface StepNodeData {
  label: string;
  stepType: string;
  rules?: ValidationRule[];
  eventId?: string;
  status?: 'draft' | 'active' | 'error';
}

interface StepConfigPanelProps {
  selectedNode: Node<StepNodeData> | null;
  onUpdateNode?: (nodeId: string, updates: Partial<StepNodeData>) => void;
}

export default function StepConfigPanel({ selectedNode, onUpdateNode }: StepConfigPanelProps) {
  const [showRuleModal, setShowRuleModal] = useState(false);
  const [eventId, setEventId] = useState(selectedNode?.data.eventId || '');
  const [stepName, setStepName] = useState(selectedNode?.data.label || '');
  const [onFailure, setOnFailure] = useState<'reject' | 'route' | 'escalate'>('reject');
  const [escalationRole, setEscalationRole] = useState('');

  // Mock events data - in real app this would come from API
  const events = [
    { value: 'client-app-submitted', label: 'Client Application Submitted' },
    { value: 'client-data-updated', label: 'Client Data Updated' },
    { value: 'account-created', label: 'Account Created' },
    { value: 'trade-requested', label: 'Trade Requested' },
  ];

  const handleSave = () => {
    if (selectedNode && onUpdateNode) {
      onUpdateNode(selectedNode.id, {
        label: stepName,
        eventId,
      });
    }
  };

  const handleAddRule = (rule: ValidationRule) => {
    if (selectedNode && onUpdateNode) {
      const currentRules = selectedNode.data.rules || [];
      onUpdateNode(selectedNode.id, {
        rules: [...currentRules, rule],
      });
    }
    setShowRuleModal(false);
  };

  const handleDeleteRule = (ruleId: string) => {
    if (selectedNode && onUpdateNode) {
      const currentRules = selectedNode.data.rules || [];
      onUpdateNode(selectedNode.id, {
        rules: currentRules.filter(r => r.id !== ruleId),
      });
    }
  };

  if (!selectedNode) {
    return (
      <Stack gap="md" p="md">
        <Title order={4}>Step Configuration</Title>
        <Text c="dimmed" size="sm">
          Select a step on the canvas to configure it
        </Text>
      </Stack>
    );
  }

  const isValidationStep = selectedNode.data.stepType === 'validate';
  const rules = selectedNode.data.rules || [];

  return (
    <Stack gap="md" p="md">
      <Title order={4}>Step Configuration</Title>

      <TextInput
        label="Step Name"
        value={stepName}
        onChange={(e) => setStepName(e.target.value)}
        placeholder="Enter step name"
      />

      <Select
        label="Trigger Event"
        placeholder="Select an event"
        data={events}
        value={eventId}
        onChange={(value) => setEventId(value || '')}
        clearable
      />

      {isValidationStep && (
        <>
          <Title order={5}>Validation Rules</Title>

          <Button
            leftSection={<IconPlus size={16} />}
            variant="light"
            onClick={() => setShowRuleModal(true)}
          >
            Add Validation Rule
          </Button>

          {rules.length > 0 && (
            <Table>
              <Table.Thead>
                <Table.Tr>
                  <Table.Th>Field</Table.Th>
                  <Table.Th>Operator</Table.Th>
                  <Table.Th>Value</Table.Th>
                  <Table.Th>Message</Table.Th>
                  <Table.Th>Actions</Table.Th>
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {rules.map((rule) => (
                  <Table.Tr key={rule.id}>
                    <Table.Td>{rule.field}</Table.Td>
                    <Table.Td>{rule.operator}</Table.Td>
                    <Table.Td>{rule.value}</Table.Td>
                    <Table.Td>{rule.message}</Table.Td>
                    <Table.Td>
                      <ActionIcon
                        color="red"
                        onClick={() => handleDeleteRule(rule.id)}
                      >
                        <IconTrash size={16} />
                      </ActionIcon>
                    </Table.Td>
                  </Table.Tr>
                ))}
              </Table.Tbody>
            </Table>
          )}

          <div>
            <Text size="sm" fw={500} mb="xs">On Failure</Text>
            <Radio.Group value={onFailure} onChange={(value) => setOnFailure(value as any)}>
              <Group mt="xs">
                <Radio value="reject" label="Reject" />
                <Radio value="route" label="Route" />
                <Radio value="escalate" label="Escalate" />
              </Group>
            </Radio.Group>

            {onFailure === 'escalate' && (
              <TextInput
                label="Escalation Role"
                placeholder="e.g., Compliance Officer"
                value={escalationRole}
                onChange={(e) => setEscalationRole(e.target.value)}
                mt="sm"
              />
            )}
          </div>
        </>
      )}

      <Group mt="md">
        <Button onClick={handleSave}>Save Step</Button>
      </Group>

      {showRuleModal && (
        <RuleBuilderModal
          onSave={handleAddRule}
          onCancel={() => setShowRuleModal(false)}
        />
      )}
    </Stack>
  );
}