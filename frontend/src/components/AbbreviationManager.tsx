import React, { useState, useMemo, useCallback } from 'react';
import { 
  useAbbreviations, 
  useAbbreviationExpansion, 
  useSemanticTermValidation
} from '../utils/abbreviationApi';
import ProfessionalSearchInput from './ProfessionalSearchInput';
import { devDebug, devError } from '../utils/devLogger';
import { useConfirm } from './ConfirmProvider';
import { useNotification } from '../hooks/useNotification';
import { useTenant } from '../contexts/TenantContext';
import { Button } from '@mui/material'; // Adding Button import if not present, checking usage
import './AbbreviationManager.css';

interface AbbreviationManagerProps {
  className?: string;
  tenantId?: string; // Optional prop if we want to force a specific tenant context
}

export const AbbreviationManager: React.FC<AbbreviationManagerProps> = ({
  className = '',
  tenantId: propsTenantId
}) => {
  const { tenant } = useTenant();
  const tenantId = propsTenantId || tenant?.id;

  const { 
    abbreviations, 
    loading, 
    error, 
    loaded, 
    fetchAbbreviations, 
    addAbbreviation, 
    updateAbbreviation, 
    deleteAbbreviation,
    totalCount,
    hasMore,
    loadMore,
    searchAbbreviations
  } = useAbbreviations(tenantId);
  const { expandColumn } = useAbbreviationExpansion();
  const { validateTerms } = useSemanticTermValidation();
  
  const [newAbbreviation, setNewAbbreviation] = useState('');
  const [newFullWord, setNewFullWord] = useState('');
  const [newNotes, setNewNotes] = useState('');
  const [searchQuery, setSearchQuery] = useState('');
  const [testColumn, setTestColumn] = useState('');
  const [testTerms, setTestTerms] = useState('');
  const [expansionResult, setExpansionResult] = useState<any>(null);
  const [validationResult, setValidationResult] = useState<any>(null);
  const [activeTab, setActiveTab] = useState<'list' | 'add' | 'test' | 'validate'>('list');
  
  // Edit state
  const [editingAbbreviation, setEditingAbbreviation] = useState<any>(null);
  const [showEditModal, setShowEditModal] = useState(false);

  // Debounced search
  React.useEffect(() => {
    const timer = setTimeout(() => {
      // Only search if we are in the list tab, or generally if query changes
      searchAbbreviations(searchQuery);
    }, 400);
    return () => clearTimeout(timer);
  }, [searchQuery, searchAbbreviations]);

  // Handle tab changes with lazy loading
  const handleTabChange = useCallback((tabId: 'list' | 'add' | 'test' | 'validate') => {
    setActiveTab(tabId);
    
    // Lazy load abbreviations when switching to the list tab
    if (tabId === 'list' && !loaded) {
      fetchAbbreviations();
    }
  }, [loaded, fetchAbbreviations]);

  // Lazy load abbreviations when component mounts and list tab is active
  React.useEffect(() => {
    if (activeTab === 'list' && !loaded) {
      fetchAbbreviations();
    }
  }, [activeTab, loaded, fetchAbbreviations]);

  // Fetch abbreviations on component mount if not already loaded is handled by useEffect above
  // but if we want to ensure data on mount:
  React.useEffect(() => {
     if (!loaded) fetchAbbreviations();
  }, []);

  // Prepare search data for the search component (Autocomplete)
  // Note: Only shows currently loaded items
  const searchData = useMemo(() => 
    (abbreviations || []).map(abbrev => ({
      id: abbrev.id,
      text: abbrev.abbreviation,
      subtext: abbrev.full_word
    }))
  , [abbreviations]);

  // Handle adding new abbreviation
  const handleAddAbbreviation = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newAbbreviation.trim() || !newFullWord.trim()) return;

    const success = await addAbbreviation(
      newAbbreviation.toUpperCase().trim(),
      newFullWord.toUpperCase().trim(),
      newNotes.trim()
    );

    if (success) {
      setNewAbbreviation('');
      setNewFullWord('');
      setNewNotes('');
    }
  };

  // ... (Test Expansion and Validation handlers remain same) ...
  const handleTestExpansion = async () => {
    if (!testColumn.trim()) return;
    try {
      const result = await expandColumn(testColumn.trim());
      setExpansionResult(result);
    } catch (err) {
      devError('Failed to test expansion:', err);
    }
  };

  const handleValidateTerms = async () => {
    if (!testTerms.trim()) return;
    const terms = testTerms.split(',').map(t => t.trim()).filter(t => t);
    if (terms.length === 0) return;
    try {
      const result = await validateTerms(terms);
      setValidationResult(result);
    } catch (err) {
      devError('Failed to validate terms:', err);
    }
  };

  const handleEditAbbreviation = (abbrev: any) => {
    setEditingAbbreviation(abbrev);
    setNewAbbreviation(abbrev.abbreviation);
    setNewFullWord(abbrev.full_word);
    setNewNotes(abbrev.notes || '');
    setShowEditModal(true);
  };

  const handleUpdateAbbreviation = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!editingAbbreviation || !newAbbreviation.trim() || !newFullWord.trim()) return;
    const success = await updateAbbreviation(
      editingAbbreviation.id, 
      newAbbreviation.trim(), 
      newFullWord.trim(), 
      newNotes.trim()
    );
    if (success) {
      setShowEditModal(false);
      setEditingAbbreviation(null);
      setNewAbbreviation('');
      setNewFullWord('');
      setNewNotes('');
    }
  };

  const confirm = useConfirm();
  const notification = useNotification();

  const handleDeleteAbbreviation = async (id: number) => {
    if (!(await confirm({ title: 'Delete abbreviation', description: 'Are you sure you want to delete this abbreviation?' }))) return;
    try {
      await deleteAbbreviation(id);
      notification.success('Abbreviation deleted');
    } catch (err) {
      notification.error('Failed to delete abbreviation');
    }
  };

  const handleCloseEditModal = () => {
    setShowEditModal(false);
    setEditingAbbreviation(null);
    setNewAbbreviation('');
    setNewFullWord('');
    setNewNotes('');
  };

  if (error) {
    return (
      <div className={`p-4 bg-red-50 border border-red-200 rounded-lg ${className}`}>
        <p className="text-red-600">Error: {error}</p>
      </div>
    );
  }

  return (
    <div className={`abbreviation-manager ${className}`}>
      {/* Tab Navigation */}
      <div className="tab-navigation">
        <nav className="tab-nav">
          {[
            { id: 'list', label: loaded ? `Abbreviations (${totalCount})` : 'Abbreviations' },
            { id: 'add', label: 'Add New' },
            { id: 'test', label: 'Test Expansion' },
            { id: 'validate', label: 'Validate Terms' }
          ].map((tab) => (
            <button
              key={tab.id}
              onClick={() => handleTabChange(tab.id as any)}
              className={`tab-button ${activeTab === tab.id ? 'active' : ''}`}
            >
              {tab.label}
              {(tab as any).count !== undefined && tab.id !== 'list' && (
                <span className="ml-2 bg-gray-100 text-gray-600 py-1 px-2 rounded-full text-xs">
                  {(tab as any).count}
                </span>
              )}
            </button>
          ))}
        </nav>
      </div>

      <div className="tab-content">
        {/* Abbreviations List Tab */}
        {activeTab === 'list' && (
          <div>
            <div className="mb-6">
              <h3 className="text-lg font-medium">Abbreviation Dictionary</h3>
            </div>

            {/* Search Component */}
            <div className="abbreviation-search" style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
              <div style={{ flex: 1 }}>
                <ProfessionalSearchInput
                  placeholder="Search abbreviations..."
                  data={searchData}
                  onSelect={(item: any) => {
                    devDebug('[AbbreviationManager] Selected:', item);
                    setSearchQuery(item.text); // Trigger search
                  }}
                  onSearch={(query) => setSearchQuery(query)}
                />
              </div>
              <button
                onClick={() => handleTabChange('add')}
                className="form-button-primary"
                style={{
                  padding: '8px 16px',
                  display: 'flex',
                  alignItems: 'center',
                  gap: '6px',
                  whiteSpace: 'nowrap',
                  height: '42px', // Match search input height
                }}
              >
                <span style={{ fontSize: '18px', lineHeight: 1 }}>+</span> Add New
              </button>
              <button
                onClick={() => {
                  const format = window.confirm('Export as CSV? (Cancel for JSON)') ? 'csv' : 'json';
                  const url = `/api/abbreviations/export?format=${format}`;
                  window.location.href = url;
                }}
                title="Export abbreviations"
                style={{
                  padding: '10px 20px',
                  backgroundColor: '#4CAF50',
                  border: '1px solid #45a049',
                  borderRadius: '6px',
                  cursor: 'pointer',
                  fontSize: '14px',
                  fontWeight: '500',
                  color: 'white',
                  whiteSpace: 'nowrap',
                  transition: 'all 0.2s',
                  display: 'flex',
                  alignItems: 'center',
                  gap: '6px',
                }}
                onMouseEnter={(e) => {
                  e.currentTarget.style.backgroundColor = '#45a049';
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.backgroundColor = '#4CAF50';
                }}
              >
                <span style={{ fontSize: '16px' }}>📥</span> Export
              </button>
              {searchQuery && (
                <button
                  onClick={() => setSearchQuery('')}
                  title="Clear search"
                  style={{
                    padding: '10px 20px',
                    backgroundColor: '#f5f5f5',
                    border: '1px solid #ddd',
                    borderRadius: '6px',
                    cursor: 'pointer',
                    fontSize: '14px',
                    fontWeight: '500',
                    color: '#666',
                    whiteSpace: 'nowrap',
                    transition: 'all 0.2s',
                  }}
                >
                  ✕ Reset
                </button>
              )}
            </div>

            {searchQuery && (
               <div className="search-results-info">
                 Found {totalCount} matching abbreviations
               </div>
            )}
            
            {loading && !abbreviations.length ? (
              <div className="text-center py-8 text-gray-500">
                Loading abbreviations...
              </div>
            ) : abbreviations.length === 0 ? (
              <div className="empty-state">
                <div className="empty-icon">📚</div>
                <div className="empty-title">
                  {searchQuery ? 'No matching abbreviations' : 'No abbreviations found'}
                </div>
                <div className="empty-description">
                  {searchQuery ? 'Try adjusting your search terms' : 'Add some abbreviations to get started'}
                </div>
              </div>
            ) : (
              <div className="abbreviations-list">
                {abbreviations.map((abbrev) => (
                  <div key={abbrev.id} className="abbreviation-card">
                    <div className="abbreviation-header">
                      <div className="abbreviation-main">
                        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                          <div className="abbreviation-term">
                            {abbrev.abbreviation}
                          </div>
                          {abbrev.is_core ? (
                            <span 
                              style={{
                                padding: '2px 8px',
                                backgroundColor: '#e3f2fd',
                                color: '#1976d2',
                                borderRadius: '4px',
                                fontSize: '11px',
                                fontWeight: '600',
                                textTransform: 'uppercase',
                                letterSpacing: '0.5px',
                              }}
                              title="Core abbreviation - managed by Uisce, available to all tenants (read-only)"
                            >
                              CORE
                            </span>
                          ) : (
                            <span 
                              style={{
                                padding: '2px 8px',
                                backgroundColor: '#f3e5f5',
                                color: '#7b1fa2',
                                borderRadius: '4px',
                                fontSize: '11px',
                                fontWeight: '600',
                                textTransform: 'uppercase',
                                letterSpacing: '0.5px',
                              }}
                              title="Custom abbreviation - specific to your tenant"
                            >
                              CUSTOM
                            </span>
                          )}
                        </div>
                        <div className="abbreviation-expansion">
                          {abbrev.full_word}
                        </div>
                        {abbrev.notes && (
                          <div className="abbreviation-notes">
                            {abbrev.notes}
                          </div>
                        )}
                      </div>
                      <div className="abbreviation-actions">
                        {(!abbrev.is_core || abbrev.tenant_id === tenantId) && (
                          <>
                            <button 
                              className="abbreviation-action-btn"
                              onClick={() => handleEditAbbreviation(abbrev)}
                              title="Edit abbreviation"
                            >
                              ✏️
                            </button>
                            <button 
                              className="abbreviation-action-btn"
                              onClick={() => handleDeleteAbbreviation(abbrev.id)}
                              title="Delete abbreviation"
                            >
                              🗑️
                            </button>
                          </>
                        )}
                        {abbrev.is_core && abbrev.tenant_id !== tenantId && (
                          <span 
                            style={{ 
                              fontSize: '12px', 
                              color: '#999',
                              fontStyle: 'italic',
                              padding: '4px 8px',
                            }}
                            title="Core abbreviations cannot be edited or deleted"
                          >
                            Read-only
                          </span>
                        )}
                      </div>
                    </div>
                  </div>
                ))}
                
                {/* Load More Button */}
                {hasMore && (
                  <div className="load-more-container text-center py-4">
                    <button 
                      onClick={loadMore}
                      disabled={loading}
                      style={{
                        padding: '10px 20px',
                        backgroundColor: '#f0f0f0',
                        border: '1px solid #ddd',
                        borderRadius: '6px',
                        cursor: loading ? 'not-allowed' : 'pointer',
                        color: '#444'
                      }}
                    >
                      {loading ? 'Loading...' : 'Load More'}
                    </button>
                  </div>
                )}
              </div>
            )}
          </div>
        )}

        {/* Add New Tab */}
        {activeTab === 'add' && (
          <div>
            <h3 className="text-lg font-medium mb-4">Add New Abbreviation</h3>
            <form onSubmit={handleAddAbbreviation} className="abbreviation-form">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div className="form-group">
                  <label className="form-label">
                    Abbreviation *
                  </label>
                  <input
                    type="text"
                    value={newAbbreviation}
                    onChange={(e) => setNewAbbreviation(e.target.value.toUpperCase())}
                    placeholder="e.g., TXN, ACCT, CUST"
                    className="form-input"
                    required
                  />
                </div>
                <div className="form-group">
                  <label className="form-label">
                    Full Word *
                  </label>
                  <input
                    type="text"
                    value={newFullWord}
                    onChange={(e) => setNewFullWord(e.target.value.toUpperCase())}
                    placeholder="e.g., TRANSACTION, ACCOUNT, CUSTOMER"
                    className="form-input"
                    required
                  />
                </div>
              </div>
              <div className="form-group">
                <label className="form-label">
                  Notes (Optional)
                </label>
                <textarea
                  value={newNotes}
                  onChange={(e) => setNewNotes(e.target.value)}
                  placeholder="Additional context or usage notes..."
                  rows={3}
                  className="form-textarea"
                />
              </div>
              <button
                type="submit"
                className="form-button-primary"
              >
                Add Abbreviation
              </button>
            </form>
          </div>
        )}

        {/* Test Expansion Tab */}
        {activeTab === 'test' && (
          <div className="test-section">
            <div className="test-header">
              <h3 className="test-title">Test Abbreviation Expansion</h3>
            </div>
            
            <div className="test-input-group">
              <input
                type="text"
                value={testColumn}
                onChange={(e) => setTestColumn(e.target.value)}
                placeholder="e.g., CUST_ID, TXN_AMT, ACCT_BAL"
                className="test-input"
              />
              <button
                onClick={handleTestExpansion}
                className="test-button"
              >
                Test
              </button>
            </div>
            
            {expansionResult && (
              <div className="test-results">
                <h4 className="result-label">Expansion Results:</h4>
                <div className="space-y-2">
                  <div className="result-item">
                    <div className="result-label">Original:</div>
                    <div className="result-value">{expansionResult.column_name}</div>
                  </div>
                  <div className="result-item">
                    <div className="result-label">Variations:</div>
                    <ul className="ml-4 mt-1">
                      {expansionResult.variations.map((variation: string, idx: number) => (
                        <li key={idx} className="result-value">{variation}</li>
                      ))}
                    </ul>
                  </div>
                  {expansionResult.expansions && (
                    <div className="result-item">
                      <div className="result-label">Expansions:</div>
                      <div className="result-value">{expansionResult.expansions}</div>
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>
        )}

        {/* Validate Terms Tab */}
        {activeTab === 'validate' && (
          <div className="test-section">
            <div className="test-header">
              <h3 className="test-title">Validate Semantic Terms</h3>
            </div>
            
            <div className="form-group">
              <label className="form-label">
                Semantic Term Names (comma-separated)
              </label>
              <textarea
                value={testTerms}
                onChange={(e) => setTestTerms(e.target.value)}
                placeholder="e.g., CUSTOMER_ID, TXN_AMT, ACCT_BALANCE, CUST_NAME"
                rows={3}
                className="form-textarea"
              />
              <button
                onClick={handleValidateTerms}
                className="test-button"
              >
                Validate Terms
              </button>
            </div>
            
            {validationResult && (
              <div className="test-results">
                <h4 className="result-label">Validation Results:</h4>
                <div className="space-y-3">
                  <div className="grid grid-cols-3 gap-4 text-center">
                    <div className="result-item bg-green-50 text-green-800">
                      <div className="result-label">Valid Terms</div>
                      <div className="text-lg font-bold">{validationResult.valid_terms}</div>
                    </div>
                    <div className="result-item bg-red-50 text-red-800">
                      <div className="result-label">With Abbreviations</div>
                      <div className="text-lg font-bold">
                        {Object.keys(validationResult.violations).length}
                      </div>
                    </div>
                    <div className="result-item bg-blue-50 text-blue-800">
                      <div className="result-label">Total Terms</div>
                      <div className="text-lg font-bold">{validationResult.total_terms}</div>
                    </div>
                  </div>
                  
                  {Object.keys(validationResult.violations).length > 0 && (
                    <div>
                      <h5 className="result-label text-red-800 mb-2">Terms with Abbreviations:</h5>
                      <div className="space-y-2">
                        {Object.entries(validationResult.violations).map(([term, violations]: [string, any]) => (
                          <div key={term} className="result-item bg-red-50 border-l-4 border-red-400">
                            <div className="result-value text-red-900">{term}</div>
                            <div className="text-red-700 text-xs mt-1">
                              Found abbreviations: {Array.isArray(violations) ? violations.join(', ') : violations}
                            </div>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>
        )}
      </div>

      {/* Edit Modal */}
      {showEditModal && editingAbbreviation && (
        <div className="modal-overlay">
          <div className="modal-content">
            <div className="modal-header">
              <h3 className="modal-title">Edit Abbreviation</h3>
            </div>
            <div className="modal-body">
              <form onSubmit={handleUpdateAbbreviation} className="abbreviation-form">
                <div className="form-group">
                  <label className="form-label">
                    Abbreviation *
                  </label>
                  <input
                    type="text"
                    value={newAbbreviation}
                    onChange={(e) => setNewAbbreviation(e.target.value.toUpperCase())}
                    placeholder="e.g., TXN, ACCT, CUST"
                    className="form-input"
                    required
                  />
                </div>
                <div className="form-group">
                  <label className="form-label">
                    Full Word *
                  </label>
                  <input
                    type="text"
                    value={newFullWord}
                    onChange={(e) => setNewFullWord(e.target.value.toUpperCase())}
                    placeholder="e.g., TRANSACTION, ACCOUNT, CUSTOMER"
                    className="form-input"
                    required
                  />
                </div>
                <div className="form-group">
                  <label className="form-label">
                    Notes (Optional)
                  </label>
                  <textarea
                    value={newNotes}
                    onChange={(e) => setNewNotes(e.target.value)}
                    placeholder="Additional context or usage notes..."
                    className="form-textarea"
                    rows={3}
                  />
                </div>
              </form>
            </div>
            <div className="modal-footer">
              <button
                type="button"
                onClick={handleCloseEditModal}
                className="form-button-secondary"
              >
                Cancel
              </button>
              <button
                type="submit"
                onClick={handleUpdateAbbreviation}
                className="form-button-primary"
              >
                Update Abbreviation
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default AbbreviationManager;