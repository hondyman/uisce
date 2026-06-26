import { useState, useEffect } from 'react';
import { Card, Text, Badge, ScrollArea, Loader, Alert } from '@mantine/core';
import { IconHistory, IconAlertCircle } from '@tabler/icons-react';
import { businessProcessService, AuditEntry } from '../services/businessProcessService';

interface AuditTrailPanelProps {
  processId: string;
  isVisible?: boolean;
}

export default function AuditTrailPanel({ processId, isVisible = true }: AuditTrailPanelProps) {
  const [auditEntries, setAuditEntries] = useState<AuditEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (processId && isVisible) {
      loadAuditTrail();
    }
  }, [processId, isVisible]);

  const loadAuditTrail = async () => {
    if (!processId) return;

    setLoading(true);
    setError(null);

    try {
      const response = await businessProcessService.getBusinessProcessAuditTrail(processId);
      setAuditEntries(response.entries);
    } catch (err: any) {
      setError(err.message || 'Failed to load audit trail');
    } finally {
      setLoading(false);
    }
  };

  const getActionColor = (actionType: string) => {
    switch (actionType) {
      case 'created': return 'green';
      case 'execution_started': return 'blue';
      case 'step_approved': return 'orange';
      case 'updated': return 'yellow';
      case 'deleted': return 'red';
      default: return 'gray';
    }
  };

  const formatTimestamp = (timestamp: string) => {
    return new Date(timestamp).toLocaleString();
  };

  if (!isVisible) return null;

  return (
    <Card shadow="sm" p="md" withBorder>
      <Card.Section withBorder inheritPadding py="xs">
        <Text fw={500} size="lg" style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          <IconHistory size={20} />
          Audit Trail
        </Text>
      </Card.Section>

      <Card.Section inheritPadding py="md">
        {loading && (
          <div style={{ display: 'flex', justifyContent: 'center', padding: '20px' }}>
            <Loader size="sm" />
            <Text ml="sm">Loading audit trail...</Text>
          </div>
        )}

        {error && (
          <Alert icon={<IconAlertCircle size={16} />} title="Error" color="red" variant="light">
            {error}
          </Alert>
        )}

        {!loading && !error && auditEntries.length === 0 && (
          <Text c="dimmed" ta="center" py="xl">
            No audit entries found for this process.
          </Text>
        )}

        {!loading && !error && auditEntries.length > 0 && (
          <ScrollArea h={300}>
            {auditEntries.map((entry) => (
              <div key={entry.id} style={{
                padding: '12px',
                borderBottom: '1px solid var(--mantine-color-gray-2)',
                marginBottom: '8px'
              }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '8px' }}>
                  <Badge color={getActionColor(entry.actionType)} variant="light">
                    {entry.actionType.replace('_', ' ')}
                  </Badge>
                  <Text size="sm" c="dimmed">
                    {formatTimestamp(entry.timestamp)}
                  </Text>
                </div>

                <div style={{ marginBottom: '4px' }}>
                  <Text size="sm" fw={500}>{entry.actorEmail}</Text>
                  {entry.actorRole && (
                    <Text size="xs" c="dimmed">Role: {entry.actorRole}</Text>
                  )}
                </div>

                {entry.actionDetails && (
                  <div style={{ marginTop: '8px', padding: '8px', backgroundColor: 'var(--mantine-color-gray-0)', borderRadius: '4px' }}>
                    {Object.entries(entry.actionDetails).map(([key, value]) => (
                      <div key={key} style={{ display: 'flex', gap: '8px' }}>
                        <Text size="xs" fw={500} style={{ minWidth: '80px' }}>{key}:</Text>
                        <Text size="xs">{String(value)}</Text>
                      </div>
                    ))}
                  </div>
                )}

                {entry.ipAddress && (
                  <Text size="xs" c="dimmed" mt="4px">
                    IP: {entry.ipAddress}
                  </Text>
                )}
              </div>
            ))}
          </ScrollArea>
        )}
      </Card.Section>
    </Card>
  );
}