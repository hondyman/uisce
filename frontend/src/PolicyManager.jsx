import { useState, useEffect, useCallback } from 'react';
import { useConfirm } from './components/ConfirmProvider';
import { useNotification } from './hooks/useNotification';
import { devError } from './utils/devLogger';
import PolicyForm from './PolicyForm';

const API_URL = (import.meta.env.VITE_API_BASE_URL || 'http://localhost:8000').replace(/\/$/, '') + '/api/v1';

const PolicyManager = () => {
  const [policies, setPolicies] = useState([]);
  const [editingPolicy, setEditingPolicy] = useState(null);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState(null);

  const fetchPolicies = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const response = await fetch(`${API_URL}/policies`);
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const data = await response.json();
      setPolicies(data || []);
    } catch (e) {
      setError('Failed to fetch policies. Please ensure the backend is running.');
      devError(e);
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchPolicies();
  }, [fetchPolicies]);

  const handleOpenModal = (policy = null) => {
    setEditingPolicy(policy);
    setIsModalOpen(true);
  };

  const handleCloseModal = () => {
    setIsModalOpen(false);
    setEditingPolicy(null);
  };

  const handleSave = async (policyToSave) => {
    const isNew = !policyToSave.id;
    const url = isNew ? `${API_URL}/policies` : `${API_URL}/policies/${policyToSave.id}`;
    const method = isNew ? 'POST' : 'PUT';

    try {
      const response = await fetch(url, {
        method: method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(policyToSave),
      });

      if (!response.ok) {
        const errData = await response.json();
        throw new Error(errData.error || 'Failed to save policy');
      }

      fetchPolicies(); // Refresh the list
      handleCloseModal();
    } catch (e) {
      setError(e.message);
      devError(e);
    }
  };

  const handleDelete = async (policyId) => {
    const confirm = useConfirm();
    const notification = useNotification();
    if (!(await confirm({ title: 'Delete policy', description: 'Are you sure you want to delete this policy?' }))) return;
      try {
        const response = await fetch(`${API_URL}/policies/${policyId}`, {
          method: 'DELETE',
        });
        if (!response.ok) {
          throw new Error('Failed to delete policy');
        }
        fetchPolicies(); // Refresh the list
        notification.success('Policy deleted');
      } catch (e) {
        setError(e.message);
        devError(e);
      }
    }
  };

  if (isLoading) return <div>Loading policies...</div>;

  return (
    <div className="policy-manager">
      <h2>ABAC Policy Management</h2>
      <button onClick={() => handleOpenModal()} className="create-btn">
        Create New Policy
      </button>

      {error && <div className="error-message">{error}</div>}

      <table>
        <thead>
          <tr>
            <th>Name</th>
            <th>Priority</th>
            <th>Status</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {policies.length > 0 ? (
            policies.map((policy) => (
              <tr key={policy.id != null && typeof policy.id !== 'object' ? String(policy.id) : ''}>
                <td>{policy.name}</td>
                <td>{policy.priority}</td>
                <td>
                  <span className={`status ${policy.active ? 'active' : 'inactive'}`}>
                    {policy.active ? 'Active' : 'Inactive'}
                  </span>
                </td>
                <td>
                  <button onClick={() => handleOpenModal(policy)}>Edit</button>
                  <button onClick={() => handleDelete(policy.id)} className="delete-btn">
                    Delete
                  </button>
                </td>
              </tr>
            ))
          ) : (
            <tr>
              <td colSpan="4">No policies found.</td>
            </tr>
          )}
        </tbody>
      </table>

      {isModalOpen && (
        <PolicyForm
          policy={editingPolicy}
          onSave={handleSave}
          onClose={handleCloseModal}
        />
      )}
    </div>
  );
};

export default PolicyManager;