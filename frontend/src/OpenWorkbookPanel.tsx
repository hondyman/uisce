import { useState, useEffect, useCallback } from 'react';
import { useNotification } from './hooks/useNotification';
import { devLog, devError } from './utils/devLogger';
import { listWorkbooks, getWorkbook } from './api'; // Assuming these API functions exist
import type { Workbook, FullWorkbook } from './types';

interface OpenWorkbookPanelProps {
  onOpen: (workbook: FullWorkbook) => void;
}

export default function OpenWorkbookPanel({ onOpen }: OpenWorkbookPanelProps) {
  const [workbooks, setWorkbooks] = useState<Workbook[]>([]);

  const fetchWorkbooks = useCallback(() => {
    // Placeholder for listWorkbooks API call
  listWorkbooks({ scope: 'mine' }).then(setWorkbooks).catch((e) => { devError(e); });
  devLog('Fetching workbooks...');
  }, []);

  useEffect(fetchWorkbooks, [fetchWorkbooks]);

  const handleOpen = async (id: string) => {
    try {
      const fullWorkbook = await getWorkbook(id);
      onOpen(fullWorkbook);
    } catch (error) {
      const notification = useNotification();
      notification.error(`Failed to open workbook: ${(error as Error).message}`);
    }
  };

  return (
    <div className="open-workbook-panel">
      <h4>Workbooks</h4>
      <button onClick={fetchWorkbooks}>Refresh</button>
      {workbooks.length === 0 && <p className="text-placeholder">No workbooks found.</p>}
      <ul>
        {workbooks.map((_workbook) => (
          <li key={_workbook.id}>
            <div className="workbook-info" onClick={() => handleOpen(_workbook.id)}>
              <strong>{_workbook.name}</strong>
              <small>{_workbook.description}</small>
            </div>
            {/* Add clone/delete buttons here */}
          </li>
        ))}
      </ul>
    </div>
  );
}