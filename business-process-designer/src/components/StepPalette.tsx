import { Stack, Title, Card, Text, Group, Badge } from '@mantine/core';
import {
  IconPlayerPlay,
  IconCheck,
  IconShield,
  IconUserCheck,
  IconFileText,
  IconCircleCheck,
  IconMail
} from '@tabler/icons-react';

export interface StepType {
  id: string;
  label: string;
  icon: React.ReactNode;
  description: string;
  category: 'flow' | 'action' | 'decision';
}

const STEP_TYPES: StepType[] = [
  {
    id: 'initiate',
    label: 'Initiate Request',
    icon: <IconPlayerPlay size={20} />,
    description: 'Start the business process',
    category: 'flow'
  },
  {
    id: 'validate',
    label: 'Validate Data',
    icon: <IconCheck size={20} />,
    description: 'Apply validation rules',
    category: 'action'
  },
  {
    id: 'aml',
    label: 'AML Screening',
    icon: <IconShield size={20} />,
    description: 'Anti-money laundering checks',
    category: 'action'
  },
  {
    id: 'approve',
    label: 'Route for Approval',
    icon: <IconUserCheck size={20} />,
    description: 'Send to approval workflow',
    category: 'decision'
  },
  {
    id: 'generate',
    label: 'Generate Docs',
    icon: <IconFileText size={20} />,
    description: 'Create documents automatically',
    category: 'action'
  },
  {
    id: 'complete',
    label: 'Complete Onboarding',
    icon: <IconCircleCheck size={20} />,
    description: 'Mark process as complete',
    category: 'flow'
  },
  {
    id: 'notify',
    label: 'Notify Client',
    icon: <IconMail size={20} />,
    description: 'Send notifications',
    category: 'action'
  },
];

interface StepPaletteProps {
  onDragStart?: (stepType: StepType) => void;
}

export default function StepPalette({ onDragStart }: StepPaletteProps) {
  const handleDragStart = (event: React.DragEvent, stepType: StepType) => {
    event.dataTransfer.setData('application/json', JSON.stringify(stepType));
    onDragStart?.(stepType);
  };

  const getCategoryColor = (category: string) => {
    switch (category) {
      case 'flow': return 'blue';
      case 'action': return 'green';
      case 'decision': return 'orange';
      default: return 'gray';
    }
  };

  return (
    <Stack gap="md">
      <Title order={4}>Step Palette</Title>

      <Stack gap="xs">
        {STEP_TYPES.map((stepType) => (
          <Card
            key={stepType.id}
            shadow="sm"
            padding="sm"
            radius="md"
            withBorder
            draggable
            onDragStart={(e) => handleDragStart(e, stepType)}
            style={{
              cursor: 'grab',
              transition: 'transform 0.2s',
            }}
          >
            <Group>
              <div style={{ color: 'var(--mantine-color-blue-6)' }}>
                {stepType.icon}
              </div>
              <div style={{ flex: 1 }}>
                <Text size="sm" fw={500}>
                  {stepType.label}
                </Text>
                <Text size="xs" c="dimmed">
                  {stepType.description}
                </Text>
              </div>
              <Badge size="xs" color={getCategoryColor(stepType.category)}>
                {stepType.category}
              </Badge>
            </Group>
          </Card>
        ))}
      </Stack>
    </Stack>
  );
}