import { useState, useEffect } from 'react';
import { useNotification } from './hooks/useNotification';

const PolicyForm = ({ policy, onSave, onClose }) => {
  const [formData, setFormData] = useState({
    name: '',
    rules: '{}',
    priority: 0,
    active: true,
  });

  useEffect(() => {
    if (policy) {
      setFormData({
        id: policy.id,
        name: policy.name || '',
        // Pretty-print JSON for better readability in the textarea
        rules: JSON.stringify(policy.rules, null, 2) || '{}',
        priority: policy.priority || 0,
        active: policy.active !== undefined ? policy.active : true,
      });
    }
  }, [policy]);

  const notification = useNotification();

  const handleChange = (e) => {
    const { name, value, type, checked } = e.target;
    setFormData((prev) => ({
      ...prev,
      [name]: type === 'checkbox' ? checked : value,
    }));
  };

  const handleSubmit = (e) => {
    e.preventDefault();
    try {
      // Parse the rules from JSON string back to an object before saving
      const rulesObject = JSON.parse(formData.rules);
      onSave({ ...formData, rules: rulesObject });
    } catch (error) {
      const notification = useNotification();
      notification.error('Invalid JSON in rules field.');
    }
  };

  return (
    <div className="modal-overlay">
      <div className="modal-content">
        <h3>{policy ? 'Edit Policy' : 'Create New Policy'}</h3>
        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label htmlFor="name">Policy Name</label>
            <input
              type="text"
              id="name"
              name="name"
              value={formData.name}
              onChange={handleChange}
              required
            />
          </div>
          <div className="form-group">
            <label htmlFor="rules">Rules (JSON)</label>
            <textarea
              id="rules"
              name="rules"
              value={formData.rules}
              onChange={handleChange}
              rows="10"
              required
            />
          </div>
          <div className="form-group">
            <label htmlFor="priority">Priority</label>
            <input
              type="number"
              id="priority"
              name="priority"
              value={formData.priority}
              onChange={handleChange}
            />
          </div>
          <div className="form-group checkbox-group">
            <label htmlFor="active">Active</label>
            <input type="checkbox" id="active" name="active" checked={formData.active} onChange={handleChange} />
          </div>
          <div className="form-actions">
            <button type="submit">Save Policy</button>
            <button type="button" onClick={onClose}>Cancel</button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default PolicyForm;