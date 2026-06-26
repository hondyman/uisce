import { Card, Text, Switch } from '@mantine/core';

interface MiniMapProps {
  visible?: boolean;
  onToggle?: (visible: boolean) => void;
}

export default function MiniMap({ visible = true, onToggle }: MiniMapProps) {
  return (
    <Card
      shadow="sm"
      padding="xs"
      radius="md"
      withBorder
      className={`minimap-card ${visible ? 'minimap-visible' : 'minimap-hidden'}`}
    >
      <div className="minimap-header">
        <Text size="xs" fw={500}>Mini Map</Text>
        <Switch
          size="xs"
          checked={visible}
          onChange={(event) => onToggle?.(event.currentTarget.checked)}
        />
      </div>

      <div className="minimap-content">
        <Text size="xs" c="dimmed">Process Overview</Text>
      </div>
    </Card>
  );
}