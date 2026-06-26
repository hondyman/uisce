// React default import removed — using automatic JSX runtime
import { Modal, Button, Text, Badge } from '@mantine/core';
import './ScanResultsModal.css';

export interface ScanResultItem {
  tenant_instance_id: string;
  name?: string;
  success: boolean;
  error?: string;
}

interface Props {
  opened: boolean;
  onClose: () => void;
  results: ScanResultItem[];
  onRetry: (datasourceId: string) => Promise<void> | void;
}

const ScanResultsModal: React.FC<Props> = ({ opened, onClose, results, onRetry }) => {
  const successes = results.filter((r) => r.success).length;
  const failures = results.length - successes;

  return (
    <Modal opened={opened} onClose={onClose} title={`Scan Results (${successes} ok, ${failures} failed)`} size="lg">
      <div className="scan-results-list">
        {results.map((r) => (
          <div key={r.tenant_instance_id} className="scan-results-item">
            <div className="scan-results-item-main">
              <Text fw={500}>{r.name || r.tenant_instance_id}</Text>
              {!r.success && (
                <Text color="red" fz="sm">
                  {r.error || 'Unknown error'}
                </Text>
              )}
            </div>
            <div className="scan-results-actions">
              <Badge color={r.success ? 'green' : 'red'}>{r.success ? 'Success' : 'Failed'}</Badge>
              {!r.success && (
                <Button size="xs" onClick={() => onRetry(r.tenant_instance_id)}>
                  Retry
                </Button>
              )}
            </div>
          </div>
        ))}

        <div className="scan-results-footer">
          <Button variant="outline" onClick={onClose}>
            Close
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default ScanResultsModal;
