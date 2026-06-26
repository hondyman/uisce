import { Group, Title, Button, TextInput } from '@mantine/core';
import { IconDeviceFloppy } from '@tabler/icons-react';

interface ProcessHeaderProps {
  processName?: string;
  onSave?: () => void;
  saving?: boolean;
}

export default function ProcessHeader({
  processName = 'New Business Process',
  onSave,
  saving = false,
}: ProcessHeaderProps) {
  return (
    <Group justify="space-between" style={{ width: '100%' }}>
      <Group>
        <Title order={3}>Business Process Designer</Title>
        <TextInput
          placeholder="Process Name"
          defaultValue={processName}
          size="sm"
          style={{ width: 250 }}
        />
      </Group>

      <Group>
        <Button
          leftSection={<IconDeviceFloppy size={16} />}
          variant="light"
          onClick={onSave}
          loading={saving}
          disabled={saving}
        >
          Save
        </Button>
      </Group>
    </Group>
  );
}