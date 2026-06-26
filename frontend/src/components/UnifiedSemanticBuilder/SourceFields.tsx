// React default import removed — using automatic JSX runtime
import { useEffect } from 'react';
import { devDebug, devError } from '../../utils/devLogger';

import { useTenant } from '../../contexts/TenantContext';
import SimpleTableAutocomplete from '../common/SimpleTableAutocomplete';
import SimpleColumnAutocomplete from '../common/SimpleColumnAutocomplete';
import ModelTableAutocomplete from '../common/ModelTableAutocomplete';

interface Props {
  formData: any;
  setFormData: (v: any) => void;
  disabledSourceTable?: string | { id: string; qualified_path: string } | null;
  semanticModel?: any; // When provided, limit table search to model tables
}

const SourceFields: React.FC<Props> = ({ formData, setFormData, disabledSourceTable, semanticModel }) => {
  // tenant datasource scope for typeahead
  const tenant = (() => {
    try { return useTenant(); } catch { return { tenant: null, product: null, datasource: null } as any; }
  })();

  // Pre-populate source table if disabledSourceTable is provided and not already set
  useEffect(() => {
    if (disabledSourceTable && !formData.sourceTable) {
      setFormData({ ...formData, sourceTable: disabledSourceTable });
    }
  }, [disabledSourceTable, formData.sourceTable, setFormData]);

  // Fetch business term for title when sourceColumn changes
  useEffect(() => {
    if (formData.sourceColumn && typeof formData.sourceColumn === 'object' && formData.sourceColumn.id) {
      (async () => {
        const url = `/api/business-term?column_id=${formData.sourceColumn.id}&edge_type_id=97d82101-2b84-47a6-9ec0-f930fe389c3c&target_node_id=820b942a-9c9e-4abc-acdc-84616db33098`;
          try {
          devDebug('[SourceFields] fetch business term url:', url);
          const res = await fetch(url);
          const text = await res.text();
          devDebug('[SourceFields] business-term response status:', res.status, 'body:', text.slice(0, 1000));
          if (!res.ok) {
            devError('[SourceFields] business-term request failed, status:', res.status);
            return;
          }
          let data: any = null;
          try {
            data = JSON.parse(text);
          } catch (err) {
            devError('[SourceFields] Error parsing business-term JSON:', err, 'raw:', text);
            return;
          }
          if (data && data.business_term) {
            setFormData((prev: any) => ({ ...prev, title: data.business_term }));
          }
        } catch (err) {
          devError('[SourceFields] Error fetching business term:', err);
        }
      })();
    }
  }, [formData.sourceColumn, setFormData]);

  return (
    <div className="source-fields-vertical">
      {semanticModel ? (
        <ModelTableAutocomplete
          datasourceId={tenant?.datasource?.id}
          semanticModel={semanticModel}
          value={formData.sourceTable || null}
          onChange={(val) => {
            if (val === null) {
              setFormData({ ...formData, sourceTable: null, sourceColumn: null, name: '', type: '', sql: '', format: '' });
            } else {
              setFormData({ ...formData, sourceTable: val });
            }
          }}
          className="wide-input source-table-autocomplete"
          disabled={!!disabledSourceTable}
        />
      ) : (
        <SimpleTableAutocomplete
          datasourceId={tenant?.datasource?.id}
          value={formData.sourceTable || null}
          onChange={(val) => {
            if (val === null) {
              setFormData({ ...formData, sourceTable: null, sourceColumn: null, name: '', type: '', sql: '', format: '' });
            } else {
              setFormData({ ...formData, sourceTable: val });
            }
          }}
          className="wide-input source-table-autocomplete"
          disabled={!!disabledSourceTable}
        />
      )}
      <SimpleColumnAutocomplete
        datasourceId={tenant?.datasource?.id}
        parentId={disabledSourceTable || (typeof formData.sourceTable === 'object' ? formData.sourceTable?.id : formData.sourceTable)}
        value={formData.sourceColumn || null}
        onChange={(val) => {
          if (val === null) {
            setFormData({ ...formData, sourceColumn: null, name: '', type: '', sql: '', format: '' });
          } else {
            const newData = { ...formData, sourceColumn: val };
            if (val && typeof val === 'object') {
              // Set name to qualified path. If the column doesn't include a qualified_path,
              // synthesize one from the selected table's qualified_path and the column name.
              if (val.qualified_path) {
                newData.name = val.qualified_path;
              } else {
                const tbl = formData.sourceTable && typeof formData.sourceTable === 'object'
                  ? (formData.sourceTable.qualified_path || formData.sourceTable.node_name || '')
                  : (formData.sourceTable || '');
                const tblDot = tbl.replace(/\//g, '.');
                newData.name = tblDot ? `${tblDot}.${val.node_name}` : (val.node_name || '');
              }
              // Set type to data_type from properties
              if (val.properties && val.properties.data_type) {
                newData.type = val.properties.data_type;
              }
              // Set sql to schema.table.column
              const tableName = formData.sourceTable && typeof formData.sourceTable === 'object' ? (formData.sourceTable.qualified_path || formData.sourceTable.node_name || '').replace(/\//g, '.') : (formData.sourceTable || '').replace(/\//g, '.');
              const tableParts = tableName.split('.');
              const schemaTable = tableParts.length >= 2 ? tableParts.slice(-2).join('.') : tableName;
              if (schemaTable && val.node_name) {
                newData.sql = `${schemaTable}.${val.node_name}`;
              }
            }
            setFormData(newData);
          }
        }}
        className="wide-input source-column-autocomplete"
        minChars={0}
        showAllOnFocus={true}
      />
    </div>
  );
};

export default SourceFields;
