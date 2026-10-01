import React, { useState, useEffect } from 'react';
import { lakehouseApi, type IcebergTableDef } from '../api';

export const LakekeeperCatalogManager: React.FC = () => {
  const [tables, setTables] = useState<IcebergTableDef[]>([]);
  const [selectedTable, setSelectedTable] = useState<IcebergTableDef | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isProvisionModalOpen, setIsProvisionModalOpen] = useState(false);
  const [newNamespace, setNewNamespace] = useState('raw_market_data');
  const [newTableName, setNewTableName] = useState('');
  const [newPartitions, setNewPartitions] = useState('tenant_id, as_of_date');
  const [provisioning, setProvisioning] = useState(false);

  const fetchTables = async () => {
    try {
      setLoading(true);
      setError(null);
      const res = await lakehouseApi.listTables();
      setTables(res);
      if (res.length > 0 && !selectedTable) {
        setSelectedTable(res[0]);
      }
    } catch (err: any) {
      setError(err.message || 'Failed to fetch Iceberg tables');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchTables();
  }, []);

  const handleProvision = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      setProvisioning(true);
      const parts = newPartitions.split(',').map((p) => p.trim()).filter(Boolean);
      await lakehouseApi.provisionTable({
        namespace: newNamespace,
        table_name: newTableName,
        partition_fields: parts,
      });
      setIsProvisionModalOpen(false);
      setNewTableName('');
      await fetchTables();
    } catch (err: any) {
      alert(`Provisioning failed: ${err.message}`);
    } finally {
      setProvisioning(false);
    }
  };

  return (
    <div style={{ padding: '16px', display: 'flex', flexDirection: 'column', gap: '20px' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div>
          <h3 style={{ margin: '0 0 4px 0', fontSize: '18px', fontWeight: 600, color: 'var(--text-primary, #1e293b)' }}>
            Lakekeeper REST Catalog & Cold Tier Management
          </h3>
          <p style={{ margin: 0, fontSize: '13px', color: 'var(--text-secondary, #64748b)' }}>
            Apache Iceberg tables, tenant-partitioned parquet storage on MinIO S3, and StarRocks external catalog query federation
          </p>
        </div>
        <button
          onClick={() => setIsProvisionModalOpen(true)}
          style={{
            padding: '8px 16px',
            backgroundColor: '#0284c7',
            color: '#fff',
            border: 'none',
            borderRadius: '6px',
            fontSize: '13px',
            cursor: 'pointer',
            fontWeight: 600,
          }}
        >
          + Provision Iceberg Table
        </button>
      </div>

      {error && (
        <div style={{ padding: '12px 16px', backgroundColor: '#fef2f2', border: '1px solid #fecaca', borderRadius: '8px', color: '#dc2626', fontSize: '13px' }}>
          {error}
        </div>
      )}

      {/* Main split view */}
      <div style={{ display: 'grid', gridTemplateColumns: '320px 1fr', gap: '20px' }}>
        {/* Table List Sidebar */}
        <div style={{ backgroundColor: '#fff', border: '1px solid #e2e8f0', borderRadius: '8px', overflow: 'hidden' }}>
          <div style={{ padding: '12px 16px', borderBottom: '1px solid #e2e8f0', backgroundColor: '#f8fafc', fontWeight: 600, fontSize: '13px', color: '#475569' }}>
            Registered Iceberg Tables ({tables.length})
          </div>
          <div style={{ display: 'flex', flexDirection: 'column' }}>
            {tables.map((t) => {
              const isSelected = selectedTable?.namespace === t.namespace && selectedTable?.table_name === t.table_name;
              return (
                <div
                  key={`${t.namespace}.${t.table_name}`}
                  onClick={() => setSelectedTable(t)}
                  style={{
                    padding: '12px 16px',
                    borderBottom: '1px solid #f1f5f9',
                    cursor: 'pointer',
                    backgroundColor: isSelected ? '#eff6ff' : 'transparent',
                    borderLeft: isSelected ? '3px solid #0284c7' : '3px solid transparent',
                  }}
                >
                  <div style={{ fontSize: '13px', fontWeight: 600, color: isSelected ? '#0284c7' : '#1e293b' }}>
                    {t.namespace}.{t.table_name}
                  </div>
                  <div style={{ fontSize: '12px', color: '#64748b', marginTop: '4px', display: 'flex', justifyContent: 'space-between' }}>
                    <span>{t.storage_tier}</span>
                    <span>{t.row_count?.toLocaleString()} rows</span>
                  </div>
                </div>
              );
            })}
          </div>
        </div>

        {/* Table Details & Schema */}
        {selectedTable ? (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
            {/* Metadata Card */}
            <div style={{ backgroundColor: '#fff', border: '1px solid #e2e8f0', borderRadius: '8px', padding: '16px' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '12px' }}>
                <div>
                  <h4 style={{ margin: '0 0 4px 0', fontSize: '16px', fontWeight: 600, color: '#0f172a' }}>
                    {selectedTable.namespace}.{selectedTable.table_name}
                  </h4>
                  <div style={{ fontSize: '12px', fontFamily: 'monospace', color: '#64748b' }}>
                    {selectedTable.location}
                  </div>
                </div>
                <span style={{
                  padding: '4px 10px',
                  borderRadius: '4px',
                  fontSize: '11px',
                  fontWeight: 600,
                  backgroundColor: '#e0f2fe',
                  color: '#0369a1',
                }}>
                  {selectedTable.file_format}
                </span>
              </div>

              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: '12px', marginTop: '12px', paddingTop: '12px', borderTop: '1px solid #f1f5f9' }}>
                <div>
                  <div style={{ fontSize: '11px', color: '#64748b', textTransform: 'uppercase' }}>Partition Strategy</div>
                  <div style={{ fontSize: '13px', fontWeight: 600, color: '#0f172a', marginTop: '2px' }}>
                    ({selectedTable.partition_fields.join(', ')})
                  </div>
                </div>
                <div>
                  <div style={{ fontSize: '11px', color: '#64748b', textTransform: 'uppercase' }}>Total Size</div>
                  <div style={{ fontSize: '13px', fontWeight: 600, color: '#0f172a', marginTop: '2px' }}>
                    {(selectedTable.size_bytes / (1024 * 1024)).toFixed(2)} MB
                  </div>
                </div>
                <div>
                  <div style={{ fontSize: '11px', color: '#64748b', textTransform: 'uppercase' }}>Last Compacted</div>
                  <div style={{ fontSize: '13px', fontWeight: 600, color: '#0f172a', marginTop: '2px' }}>
                    {new Date(selectedTable.last_updated).toLocaleTimeString()}
                  </div>
                </div>
              </div>
            </div>

            {/* Schema Table */}
            <div style={{ backgroundColor: '#fff', border: '1px solid #e2e8f0', borderRadius: '8px', overflow: 'hidden' }}>
              <div style={{ padding: '12px 16px', borderBottom: '1px solid #e2e8f0', backgroundColor: '#f8fafc', fontWeight: 600, fontSize: '13px' }}>
                Schema Definition & Partition Columns
              </div>
              <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px' }}>
                <thead>
                  <tr style={{ backgroundColor: '#f1f5f9', color: '#475569', textAlign: 'left' }}>
                    <th style={{ padding: '10px 14px', borderBottom: '1px solid #cbd5e1' }}>Column Name</th>
                    <th style={{ padding: '10px 14px', borderBottom: '1px solid #cbd5e1' }}>Data Type</th>
                    <th style={{ padding: '10px 14px', borderBottom: '1px solid #cbd5e1' }}>Partition Role</th>
                    <th style={{ padding: '10px 14px', borderBottom: '1px solid #cbd5e1' }}>Required</th>
                    <th style={{ padding: '10px 14px', borderBottom: '1px solid #cbd5e1' }}>Doc / Description</th>
                  </tr>
                </thead>
                <tbody>
                  {selectedTable.columns.map((c, idx) => {
                    const isPartition = selectedTable.partition_fields.includes(c.name);
                    return (
                      <tr key={idx} style={{ borderBottom: '1px solid #f1f5f9' }}>
                        <td style={{ padding: '10px 14px', fontFamily: 'monospace', fontWeight: 600, color: '#334155' }}>
                          {c.name}
                        </td>
                        <td style={{ padding: '10px 14px', fontFamily: 'monospace', color: '#0284c7' }}>
                          {c.type}
                        </td>
                        <td style={{ padding: '10px 14px' }}>
                          {isPartition ? (
                            <span style={{ padding: '2px 8px', borderRadius: '4px', fontSize: '11px', fontWeight: 600, backgroundColor: '#fef3c7', color: '#92400e' }}>
                              PARTITION KEY
                            </span>
                          ) : (
                            <span style={{ color: '#94a3b8', fontSize: '12px' }}>—</span>
                          )}
                        </td>
                        <td style={{ padding: '10px 14px', color: c.required ? '#0f172a' : '#64748b' }}>
                          {c.required ? 'YES' : 'NO'}
                        </td>
                        <td style={{ padding: '10px 14px', color: '#64748b' }}>
                          {c.doc || '—'}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </div>
        ) : (
          <div style={{ padding: '40px', textAlign: 'center', color: '#64748b', backgroundColor: '#fff', border: '1px solid #e2e8f0', borderRadius: '8px' }}>
            Select an Iceberg table to view its schema and partition configuration.
          </div>
        )}
      </div>

      {/* Provision Modal */}
      {isProvisionModalOpen && (
        <div style={{
          position: 'fixed',
          top: 0,
          left: 0,
          right: 0,
          bottom: 0,
          backgroundColor: 'rgba(0,0,0,0.5)',
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'center',
          zIndex: 1000,
        }}>
          <div style={{ backgroundColor: '#fff', borderRadius: '8px', padding: '24px', width: '480px', maxWidth: '90%' }}>
            <h3 style={{ margin: '0 0 16px 0', fontSize: '18px', fontWeight: 600 }}>Provision Iceberg Table</h3>
            <form onSubmit={handleProvision} style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
              <div>
                <label style={{ display: 'block', fontSize: '12px', fontWeight: 600, marginBottom: '4px' }}>Namespace</label>
                <input
                  type="text"
                  value={newNamespace}
                  onChange={(e) => setNewNamespace(e.target.value)}
                  style={{ width: '100%', padding: '8px 12px', border: '1px solid #cbd5e1', borderRadius: '6px', fontSize: '13px' }}
                  required
                />
              </div>
              <div>
                <label style={{ display: 'block', fontSize: '12px', fontWeight: 600, marginBottom: '4px' }}>Table Name</label>
                <input
                  type="text"
                  value={newTableName}
                  onChange={(e) => setNewTableName(e.target.value)}
                  placeholder="e.g. source_attribute_value"
                  style={{ width: '100%', padding: '8px 12px', border: '1px solid #cbd5e1', borderRadius: '6px', fontSize: '13px' }}
                  required
                />
              </div>
              <div>
                <label style={{ display: 'block', fontSize: '12px', fontWeight: 600, marginBottom: '4px' }}>Partition Fields (Comma separated)</label>
                <input
                  type="text"
                  value={newPartitions}
                  onChange={(e) => setNewPartitions(e.target.value)}
                  placeholder="tenant_id, as_of_date"
                  style={{ width: '100%', padding: '8px 12px', border: '1px solid #cbd5e1', borderRadius: '6px', fontSize: '13px' }}
                  required
                />
                <span style={{ fontSize: '11px', color: '#64748b', marginTop: '2px', display: 'block' }}>
                  tenant_id is always enforced as the first partition key for defense-in-depth tenant isolation.
                </span>
              </div>
              <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '10px', marginTop: '10px' }}>
                <button
                  type="button"
                  onClick={() => setIsProvisionModalOpen(false)}
                  style={{ padding: '8px 16px', backgroundColor: '#f1f5f9', border: '1px solid #cbd5e1', borderRadius: '6px', fontSize: '13px', cursor: 'pointer' }}
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={provisioning}
                  style={{ padding: '8px 16px', backgroundColor: '#0284c7', color: '#fff', border: 'none', borderRadius: '6px', fontSize: '13px', cursor: 'pointer', fontWeight: 600 }}
                >
                  {provisioning ? 'Provisioning...' : 'Confirm & Provision'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};
