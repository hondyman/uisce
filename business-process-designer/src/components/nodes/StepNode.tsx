import { Handle, Position, NodeProps } from 'reactflow';
import { Card, Text, Badge, Group } from '@mantine/core';
import {
  IconPlayerPlay,
  IconCheck,
  IconShield,
  IconUserCheck,
  IconFileText,
  IconCircleCheck,
  IconMail,
  IconSettings
} from '@tabler/icons-react';

interface StepNodeData {
  label: string;
  stepType: string;
  rules?: any[];
  eventId?: string;
  status?: 'draft' | 'active' | 'error';
}

export default function StepNode({ data, selected }: NodeProps<StepNodeData>) {
  const getStepIcon = (stepType: string) => {
    switch (stepType) {
      case 'initiate': return <IconPlayerPlay size={16} />;
      case 'validate': return <IconCheck size={16} />;
      case 'aml': return <IconShield size={16} />;
      case 'approve': return <IconUserCheck size={16} />;
      case 'generate': return <IconFileText size={16} />;
      case 'complete': return <IconCircleCheck size={16} />;
      case 'notify': return <IconMail size={16} />;
      default: return <IconSettings size={16} />;
    }
  };

  const getStatusColor = (status?: string) => {
    switch (status) {
      case 'active': return 'green';
      case 'error': return 'red';
      default: return 'gray';
    }
  };

  const rulesCount = data.rules?.length || 0;

  return (
    <Card
      shadow={selected ? 'lg' : 'sm'}
      padding="sm"
      radius="md"
      withBorder
      style={{
        minWidth: 200,
        border: selected ? '2px solid var(--mantine-color-blue-6)' : undefined,
      }}
    >
      {/* Input Handle */}
      <Handle
        type="target"
        position={Position.Top}
        style={{
          background: 'var(--mantine-color-blue-6)',
          width: 8,
          height: 8,
        }}
      />

      {/* Node Header */}
      <Group justify="space-between" mb="xs">
        <Group gap="xs">
          {getStepIcon(data.stepType)}
          <Text size="sm" fw={600}>
            {data.label}
          </Text>
        </Group>
        <Badge size="xs" color={getStatusColor(data.status)}>
          {data.status || 'draft'}
        </Badge>
      </Group>

      {/* Node Content */}
      <div className="step-node-status">
        {data.stepType === 'validate' && (
          <div>• {rulesCount} Rules active</div>
        )}
        {data.eventId && (
          <div>• Event: {data.eventId}</div>
        )}
        <div>• Status: {data.status || 'draft'}</div>
      </div>

      {/* Output Handle */}
      <Handle
        type="source"
        position={Position.Bottom}
        style={{
          background: 'var(--mantine-color-blue-6)',
          width: 8,
          height: 8,
        }}
      />
    </Card>
  );
}