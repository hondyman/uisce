import React, { useState, useMemo } from 'react';
import * as TablerIcons from '@tabler/icons-react';
import './CreateCustomModelModal.css';
import { getTableIdFromVal } from '../../utils/tableHelpers';

interface CreateCustomModelModalProps {
  open: boolean;
  onClose: () => void;
  onCreate: (formData: CreateCustomModelFormData) => void;
  nodes?: any[]; // Available tables from datasource
}

export interface CreateCustomModelFormData {
  cubeName: string;
  sourceTable: string;
  description: string;
}

const CreateCustomModelModal: React.FC<CreateCustomModelModalProps> = ({
  open,
  onClose,
  onCreate,
  nodes = []
}) => {
  const [formData, setFormData] = useState<CreateCustomModelFormData>({
    cubeName: '',
    sourceTable: '',
    description: ''
  });
  const [searchTerm, setSearchTerm] = useState('');
  const [showDropdown, setShowDropdown] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Extract table options from nodes
  const tableOptions = useMemo(() => {
    return nodes
      .filter((node: any) => node.data && (node.data.label || node.data.tableName))
      .map((node: any) => {
        const tableName = node.data.label || node.data.tableName || node.id;
        const schema = node.data.schema || '';
        const fullName = schema ? `${schema}.${tableName}` : tableName;
        return {
          value: fullName,
          label: fullName,
          columns: node.data.columns || []
        };
      })
      .sort((a, b) => a.label.localeCompare(b.label));
  }, [nodes]);

  // Filter tables based on search term
  const filteredTables = useMemo(() => {
    if (!searchTerm.trim()) return tableOptions;
    const term = searchTerm.toLowerCase();
    return tableOptions.filter(table => 
      table.label.toLowerCase().includes(term)
    );
  }, [tableOptions, searchTerm]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    // Validation
    if (!formData.cubeName.trim()) {
      setError('Cube name is required');
      return;
    }
    if (!formData.sourceTable.trim()) {
      setError('Source table is required');
      return;
    }
    if (!formData.description.trim()) {
      setError('Description is required');
      return;
    }

    // Validate cube name (alphanumeric + underscore)
    if (!/^[a-zA-Z][a-zA-Z0-9_]*$/.test(formData.cubeName)) {
      setError('Cube name must start with a letter and contain only letters, numbers, and underscores');
      return;
    }

    // Ensure sourceTable is a string id
    const payload = { ...formData, sourceTable: getTableIdFromVal(formData.sourceTable) };
    onCreate(payload);
    handleClose();
  };

  const handleClose = () => {
    setFormData({
      cubeName: '',
      sourceTable: '',
      description: ''
    });
    setSearchTerm('');
    setShowDropdown(false);
    setError(null);
    onClose();
  };

  const handleTableSelect = (table: string) => {
    setFormData(prev => ({ ...prev, sourceTable: table }));
    setSearchTerm(table);
    setShowDropdown(false);
  };

  const handleTableInputChange = (value: string) => {
    setSearchTerm(value);
    setFormData(prev => ({ ...prev, sourceTable: value }));
    setShowDropdown(true);
  };

  const handleFieldChange = (field: keyof CreateCustomModelFormData, value: string) => {
    setFormData(prev => ({ ...prev, [field]: value }));
    if (error) setError(null);
  };

  if (!open) return null;

  return (
    <div className="create-custom-modal-overlay">
      <div className="create-custom-modal">
        <div className="modal-header">
          <div className="modal-header-left">
            <h3>Create Custom Model</h3>
            <p>Define a new custom semantic model</p>
          </div>
          <button className="close-btn" onClick={handleClose} title="Close">
            <TablerIcons.IconX size={18} />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="modal-form">
          <div className="form-section">
            <h4>Basic Information</h4>
            
            <div className="form-group">
              <label htmlFor="cubeName">Cube Name *</label>
              <input
                type="text"
                id="cubeName"
                value={formData.cubeName}
                onChange={(e) => handleFieldChange('cubeName', e.target.value)}
                placeholder="e.g., customer_analytics, sales_metrics"
                className="form-input"
                autoFocus
              />
              <div className="field-hint">
                Must start with a letter and contain only letters, numbers, and underscores
              </div>
            </div>

            <div className="form-group">
              <label htmlFor="sourceTable">Source Table *</label>
              <div className="table-selector">
                <input
                  type="text"
                  id="sourceTable"
                  value={searchTerm}
                  onChange={(e) => handleTableInputChange(e.target.value)}
                  onFocus={() => setShowDropdown(true)}
                  placeholder="Search and select a table..."
                  className="form-input"
                />
                <TablerIcons.IconSearch size={16} className="search-icon" />
                
                {showDropdown && filteredTables.length > 0 && (
                  <div className="table-dropdown">
                    {filteredTables.slice(0, 10).map((table) => (
                      <div
                        key={table.value}
                        className="table-option"
                        onClick={() => handleTableSelect(table.value)}
                      >
                        <div className="table-name">
                          <TablerIcons.IconDatabase size={14} />
                          {table.label}
                        </div>
                        <div className="table-meta">
                          {table.columns.length} columns
                        </div>
                      </div>
                    ))}
                    {filteredTables.length > 10 && (
                      <div className="table-option-note">
                        {filteredTables.length - 10} more tables...
                      </div>
                    )}
                  </div>
                )}
              </div>
              <div className="field-hint">
                Select the primary table this model will be based on
              </div>
            </div>

            <div className="form-group">
              <label htmlFor="description">Description *</label>
              <textarea
                id="description"
                value={formData.description}
                onChange={(e) => handleFieldChange('description', e.target.value)}
                placeholder="Describe what this model represents and its purpose..."
                className="form-textarea"
                rows={3}
              />
              <div className="field-hint">
                Provide a clear description of what this model represents
              </div>
            </div>
          </div>

          {error && (
            <div className="error-message">
              <TablerIcons.IconAlertTriangle size={16} />
              {error}
            </div>
          )}

          <div className="form-actions">
            <button type="button" onClick={handleClose} className="btn btn-secondary">
              Cancel
            </button>
            <button type="submit" className="btn btn-primary">
              <TablerIcons.IconPlus size={16} />
              Create Model
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default CreateCustomModelModal;
