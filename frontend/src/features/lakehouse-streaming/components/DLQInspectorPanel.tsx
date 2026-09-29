import React, { useState, useEffect } from 'react';
import { lakehouseApi, type DLQViolationRecord } from '../api';

export const DLQInspectorPanel: React.FC = () => {
  const [records, setRecords] = useState<DLQViolationRecord[]>([]);
  const [selectedRecord, setSelectedRecord] = useState<DLQViolationRecord | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [replaying, setReplaying] = useState(false);

  const fetchDLQ = async () => {
    try {
      setLoading(true);
      setError(null);
      const res = await lakehouseApi.listDLQ(100);
      setRecords(res);
      if (res.length > 0 && !selectedRecord) {
        setSelectedRecord(res[0]);
      }
    } catch (err: any) {
      setError(err.message || 'Failed to fetch DLQ records');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchDLQ();
  }, []);

  const handleReplay = async (action: 'replay' | 'dismiss') => {
    if (!selectedRecord) return;
    try {
      setReplaying(true);
      await lakehouseApi.replayDLQ([selectedRecord.id], action);
      await fetchDLQ();
    } catch (err: any) {
      alert(`Action failed: ${err.message}`);
    } finally {
      setReplaying(false);
    }
  };

  return (
    <div style={{ padding: '16px', display: 'flex', flexDirection: 'column', gap: '20px' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div>
          <h3 style={{ margin: '0 0 4px 0', fontSize: '18px', fontWeight: 600, color: 'var(--text-primary, #1e293b)' }}>
            Dead-Letter Queue (DLQ) & Blocked Stream Inspector
          </h3>
          <p style={{ margin: 0, fontSize: '13px', color: 'var(--text-secondary, #64748b)' }}>
            Inspection, tenant mismatch diagnostics, and in-line rule rejection telemetry
          </p>
        </div>
        <button
          onClick={fetchDLQ}
          style={{
            padding: '6px 14px',
            backgroundColor: '#f1f5f9',
            border: '1px solid #cbd5e1',
            borderRadius: '6px',
            fontSize: '13px',
            cursor: 'pointer',
            fontWeight: 500,
          }}
        >
          Refresh DLQ
        </button>
      </div>

      {error && (
        <div style={{ padding: '12px 16px', backgroundColor: '#fef2f2', border: '1px solid #fecaca', borderRadius: '8px', color: '#dc2626', fontSize: '13px' }}>
          {error}
        </div>
      )}

      {/* Main split */}
      <div style={{ display: 'grid', gridTemplateColumns: '400px 1fr', gap: '20px' }}>
        {/* DLQ Records list */}
        <div style={{ backgroundColor: '#fff', border: '1px solid #e2e8f0', borderRadius: '8px', overflow: 'hidden' }}>
          <div style={{ padding: '12px 16px', borderBottom: '1px solid #e2e8f0', backgroundColor: '#f8fafc', fontWeight: 600, fontSize: '13px', color: '#475569' }}>
            Dead-Letter Events ({records.length})
          </div>
          <div style={{ maxHeight: '600px', overflowY: 'auto' }}>
            {records.length === 0 ? (
              <div style={{ padding: '30px', textAlign: 'center', color: '#64748b', fontSize: '13px' }}>
                No active DLQ violations or rejected stream events.
              </div>
            ) : (
              records.map((r) => {
                const isSelected = selectedRecord?.id === r.id;
                return (
                  <div
                    key={r.id}
                    onClick={() => setSelectedRecord(r)}
                    style={{
                      padding: '12px 16px',
                      borderBottom: '1px solid #f1f5f9',
                      cursor: 'pointer',
                      backgroundColor: isSelected ? '#fef2f2' : 'transparent',
                      borderLeft: isSelected ? '3px solid #dc2626' : '3px solid transparent',
                    }}
                  >
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                      <span style={{ fontSize: '13px', fontWeight: 600, color: '#1e293b' }}>
                        {r.business_object || 'Record'} · {r.rule_key || 'Tenant Guard'}
                      </span>
                      <span style={{
                        padding: '2px 6px',
                        borderRadius: '4px',
                        fontSize: '10px',
                        fontWeight: 700,
                        backgroundColor: '#fee2e2',
                        color: '#991b1b',
                      }}>
                        BLOCK
                      </span>
                    </div>
                    <div style={{ fontSize: '12px', color: '#dc2626', marginTop: '4px', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                      {r.error_message}
                    </div>
                    <div style={{ fontSize: '11px', color: '#94a3b8', marginTop: '4px' }}>
                      {new Date(r.created_at).toLocaleTimeString()} · Tenant: {r.tenant_id?.slice(0, 8)}...
                    </div>
                  </div>
                );
              })
            )}
          </div>
        </div>

        {/* Selected Record Detail */}
        {selectedRecord ? (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
            <div style={{ backgroundColor: '#fff', border: '1px solid #e2e8f0', borderRadius: '8px', padding: '16px' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '12px' }}>
                <div>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                    <span style={{
                      padding: '3px 8px',
                      borderRadius: '4px',
                      fontSize: '11px',
                      fontWeight: 700,
                      backgroundColor: '#fee2e2',
                      color: '#b91c1c',
                    }}>
                      WRITE BLOCKED
                    </span>
                    <h4 style={{ margin: 0, fontSize: '16px', fontWeight: 600, color: '#0f172a' }}>
                      {selectedRecord.rule_name || selectedRecord.rule_key || 'Tenant Gatekeeper Rejection'}
                    </h4>
                  </div>
                  <div style={{ fontSize: '13px', color: '#dc2626', marginTop: '6px', fontWeight: 500 }}>
                    {selectedRecord.error_message}
                  </div>
                </div>
                <div style={{ display: 'flex', gap: '8px' }}>
                  <button
                    onClick={() => handleReplay('dismiss')}
                    disabled={replaying}
                    style={{
                      padding: '6px 12px',
                      backgroundColor: '#f1f5f9',
                      border: '1px solid #cbd5e1',
                      borderRadius: '6px',
                      fontSize: '12px',
                      cursor: 'pointer',
                    }}
                  >
                    Dismiss
                  </button>
                  <button
                    onClick={() => handleReplay('replay')}
                    disabled={replaying}
                    style={{
                      padding: '6px 12px',
                      backgroundColor: '#0284c7',
                      color: '#fff',
                      border: 'none',
                      borderRadius: '6px',
                      fontSize: '12px',
                      cursor: 'pointer',
                      fontWeight: 600,
                    }}
                  >
                    Replay Stream Event
                  </button>
                </div>
              </div>

              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: '12px', marginTop: '14px', paddingTop: '12px', borderTop: '1px solid #f1f5f9' }}>
                <div>
                  <div style={{ fontSize: '11px', color: '#64748b', textTransform: 'uppercase' }}>Subsystem</div>
                  <div style={{ fontSize: '13px', fontWeight: 600, color: '#0f172a' }}>{selectedRecord.source_subsystem}</div>
                </div>
                <div>
                  <div style={{ fontSize: '11px', color: '#64748b', textTransform: 'uppercase' }}>Tenant ID</div>
                  <div style={{ fontSize: '12px', fontFamily: 'monospace', color: '#0f172a' }}>{selectedRecord.tenant_id}</div>
                </div>
                <div>
                  <div style={{ fontSize: '11px', color: '#64748b', textTransform: 'uppercase' }}>Event Timestamp</div>
                  <div style={{ fontSize: '12px', color: '#0f172a' }}>{new Date(selectedRecord.created_at).toLocaleString()}</div>
                </div>
              </div>
            </div>

            {/* Payload / Context Viewer */}
            <div style={{ backgroundColor: '#fff', border: '1px solid #e2e8f0', borderRadius: '8px', overflow: 'hidden' }}>
              <div style={{ padding: '12px 16px', borderBottom: '1px solid #e2e8f0', backgroundColor: '#f8fafc', fontWeight: 600, fontSize: '13px' }}>
                Event Payload & Execution Context
              </div>
              <pre style={{
                margin: 0,
                padding: '16px',
                backgroundColor: '#0f172a',
                color: '#e2e8f0',
                fontSize: '12px',
                fontFamily: 'monospace',
                overflowX: 'auto',
                maxHeight: '350px',
              }}>
                {JSON.stringify(selectedRecord.details || selectedRecord, null, 2)}
              </pre>
            </div>
          </div>
        ) : (
          <div style={{ padding: '40px', textAlign: 'center', color: '#64748b', backgroundColor: '#fff', border: '1px solid #e2e8f0', borderRadius: '8px' }}>
            Select a DLQ event to inspect violation reason, tenant provenance, and full JSON payload.
          </div>
        )}
      </div>
    </div>
  );
};
