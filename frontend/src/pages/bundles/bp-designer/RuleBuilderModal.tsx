/**
 * RuleBuilderModal.tsx
 * Modal for building validation rules with field, operator, value, and message
 */

import React, { useState } from 'react';
import { useNotification } from '../../hooks/useNotification';
import { BusinessObject, BusinessObjectField, ValidationOperator, ValidationRule } from './types';
import styles from './BPDesigner.module.css';

interface RuleBuilderModalProps {
  objects: BusinessObject[];
  operators: ValidationOperator[];
  onSave: (rule: ValidationRule) => void;
  onCancel: () => void;
}

export const RuleBuilderModal: React.FC<RuleBuilderModalProps> = ({
  objects,
  operators,
  onSave,
  onCancel,
}) => {
  const [selectedObject, setSelectedObject] = useState<BusinessObject | null>(objects[0] || null);
  const [selectedField, setSelectedField] = useState<BusinessObjectField | null>(
    selectedObject?.fields[0] || null
  );
  const [selectedOperator, setSelectedOperator] = useState<ValidationOperator | null>(
    operators[0] || null
  );
  const [value, setValue] = useState('');
  const [message, setMessage] = useState('');
  const [useScript, setUseScript] = useState(false);
  const [scriptCode, setScriptCode] = useState('');

  const handleObjectChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const obj = objects.find((o) => o.id === e.target.value);
    setSelectedObject(obj || null);
    setSelectedField(obj?.fields[0] || null);
  };

  const handleFieldChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const field = selectedObject?.fields.find((f) => f.name === e.target.value);
    setSelectedField(field || null);
  };

  const handleOperatorChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const op = operators.find((o) => o.id === e.target.value);
    setSelectedOperator(op || null);
  };

  const handleSave = () => {
    const notification = useNotification();
    if (!selectedObject || !selectedField || !selectedOperator) {
      notification.error('Please select object, field, and operator');
      return;
    }

    if (useScript && !scriptCode.trim()) {
      notification.error('Please enter script code');
      return;
    }

    if (!useScript && !value) {
      notification.error('Please enter a value');
      return;
    }

    const rule: ValidationRule = {
      field: `${selectedObject.name}.${selectedField.name}`,
      field_label: selectedField.label,
      op: selectedOperator.key,
      op_label: selectedOperator.label,
      value: useScript ? `script:${scriptCode}` : value,
      message: message || `Validation failed for ${selectedField.label}`,
    };

    onSave(rule);
  };

  const renderValueInput = () => {
    if (useScript) {
      return (
        <textarea
          className={styles.scriptInput}
          value={scriptCode}
          onChange={(e) => setScriptCode(e.target.value)}
          placeholder="// JavaScript code&#10;return client.net_worth > 0 ? {valid: true} : {valid: false, message: 'Value too low'};"
          rows={6}
        />
      );
    }

    switch (selectedOperator?.value_type) {
      case 'number':
        return (
          <input
            type="number"
            className={styles.input}
            value={value}
            onChange={(e) => setValue(e.target.value)}
            placeholder="Enter number"
          />
        );
      case 'list':
        return (
          <textarea
            className={styles.textarea}
            value={value}
            onChange={(e) => setValue(e.target.value)}
            placeholder="Enter comma-separated values"
            rows={3}
          />
        );
      case 'date':
        return (
          <input
            type="date"
            className={styles.input}
            value={value}
            onChange={(e) => setValue(e.target.value)}
            placeholder="YYYY-MM-DD"
            title="Enter a date (YYYY-MM-DD)"
          />
        );
      case 'currency':
        return (
          <input
            type="number"
            className={styles.input}
            value={value}
            onChange={(e) => setValue(e.target.value)}
            placeholder="Enter amount"
            step="0.01"
          />
        );
      default:
        return (
          <input
            type="text"
            className={styles.input}
            value={value}
            onChange={(e) => setValue(e.target.value)}
            placeholder="Enter value"
          />
        );
    }
  };

  return (
    <div className={styles.modalOverlay} onClick={onCancel}>
      <div className={styles.modal} onClick={(e) => e.stopPropagation()}>
        <div className={styles.modalHeader}>
          <h3>Add Validation Rule</h3>
          <button className={styles.closeButton} onClick={onCancel}>
            ✕
          </button>
        </div>

        <div className={styles.modalBody}>
          <div className={styles.twoColumn}>
            <div className={styles.column}>
              <div className={styles.formGroup}>
                <label htmlFor="business-object-select">Business Object</label>
                <select
                  id="business-object-select"
                  className={styles.select}
                  value={selectedObject?.id || ''}
                  onChange={handleObjectChange}
                >
                  {objects.map((obj) => (
                    <option key={obj.id} value={obj.id}>
                      {obj.display_name}
                    </option>
                  ))}
                </select>
              </div>

              <div className={styles.formGroup}>
                <label htmlFor="field-select">Field</label>
                <select
                  id="field-select"
                  className={styles.select}
                  value={selectedField?.name || ''}
                  onChange={handleFieldChange}
                >
                  {selectedObject?.fields.map((field) => (
                    <option key={field.name} value={field.name}>
                      {field.label}
                    </option>
                  ))}
                </select>
              </div>

              <div className={styles.formGroup}>
                <label htmlFor="operator-select">Operator</label>
                <select
                  id="operator-select"
                  className={styles.select}
                  value={selectedOperator?.id || ''}
                  onChange={handleOperatorChange}
                >
                  {operators.map((op) => (
                    <option key={op.id} value={op.id}>
                      {op.label}
                    </option>
                  ))}
                </select>
              </div>

              <div className={styles.formGroup}>
                <label>Value</label>
                {renderValueInput()}
              </div>
            </div>

            <div className={styles.column}>
              <div className={styles.previewBox}>
                <h4>Preview</h4>
                <code className={styles.previewCode}>
                  {`{
  "field": "${selectedObject?.name}.${selectedField?.name}",
  "op": "${selectedOperator?.key}",
  "value": "${value}",
  "message": "${message}"
}`}
                </code>
              </div>
            </div>
          </div>

          <div className={styles.formGroup}>
            <label>Error Message</label>
            <input
              type="text"
              className={styles.input}
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              placeholder="e.g., Net worth must be greater than $0"
            />
          </div>

          <div className={styles.formGroup}>
            <label>
              <input
                    type="checkbox"
                    checked={useScript}
                    onChange={(e) => setUseScript(e.target.checked)}
                    aria-label="Use Script Rule"
                  />
              Use Script Rule (JavaScript)
            </label>
            {useScript && (
              <p className={styles.scriptHint}>
                Write JavaScript that returns {'{'} valid: boolean, message?: string {'}'} or a boolean.
              </p>
            )}
          </div>
        </div>

        <div className={styles.modalFooter}>
          <button className={styles.buttonSecondary} onClick={onCancel}>
            Cancel
          </button>
          <button className={styles.buttonPrimary} onClick={handleSave}>
            Add Rule
          </button>
        </div>
      </div>
    </div>
  );
};
