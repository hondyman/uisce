import React, { useState, useEffect } from 'react';
import './SemanticTermTags.css';

// ============================================================================
// SEMANTIC TERM TAGS UI COMPONENTS
// ============================================================================

interface Tag {
  id: string;
  tagKey: string;
  tagLabel: string;
  tagCategory: string;
  description?: string;
  colorCode?: string;
  iconName?: string;
}

interface TagSuggestion extends Tag {
  suggestionReason: string;
  confidenceScore: number;
}

interface SemanticTermTagsProps {
  termId: string;
  currentTags: string[];
  onTagsChange: (tags: string[]) => void;
  readOnly?: boolean;
}

interface _TagSuggestionWidgetProps {
  _termId: string;
  suggestedTags: TagSuggestion[];
  onAcceptSuggestion: (tagKey: string) => void;
  onRejectSuggestion: (tagKey: string) => void;
}

// ============================================================================
// MAIN TAG DISPLAY & EDITOR COMPONENT
// ============================================================================

export const SemanticTermTagsEditor: React.FC<SemanticTermTagsProps> = ({
  termId,
  currentTags,
  onTagsChange,
  readOnly = false,
}) => {
  const [availableTags, setAvailableTags] = useState<Tag[]>([]);
  const [selectedTags, setSelectedTags] = useState<string[]>(currentTags);
  const [searchTerm, setSearchTerm] = useState('');
  const [showDropdown, setShowDropdown] = useState(false);
  const [tagsByCategory, setTagsByCategory] = useState<Record<string, Tag[]>>({});

  // Fetch available tags on mount
  useEffect(() => {
    fetchAvailableTags();
  }, []);

  const fetchAvailableTags = async () => {
    try {
      const response = await fetch('/api/graphql', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          query: `
            query {
              semanticTags {
                id
                tagKey
                tagLabel
                tagCategory
                description
                colorCode
                iconName
              }
            }
          `,
        }),
      });
      const data = await response.json();
      const tags = data.data?.semanticTags || [];
      setAvailableTags(tags);

      // Group by category
      const grouped: Record<string, Tag[]> = {};
      tags.forEach((tag: Tag) => {
        if (!grouped[tag.tagCategory]) {
          grouped[tag.tagCategory] = [];
        }
        grouped[tag.tagCategory].push(tag);
      });
      setTagsByCategory(grouped);
    } catch (error) {
      console.error('Failed to fetch tags:', error);
    }
  };

  const handleTagSelect = (tagKey: string) => {
    if (!selectedTags.includes(tagKey)) {
      const newTags = [...selectedTags, tagKey];
      setSelectedTags(newTags);
      onTagsChange(newTags);
    }
    setSearchTerm('');
  };

  const handleTagRemove = (tagKey: string) => {
    const newTags = selectedTags.filter(t => t !== tagKey);
    setSelectedTags(newTags);
    onTagsChange(newTags);
  };

  const filteredTags = availableTags.filter(tag =>
    tag.tagLabel.toLowerCase().includes(searchTerm.toLowerCase()) ||
    tag.tagKey.toLowerCase().includes(searchTerm.toLowerCase())
  );

  const selectedTagObjects = availableTags.filter(tag =>
    selectedTags.includes(tag.tagKey)
  );

  return (
    <div className="semantic-term-tags-editor">
      <div className="tags-header">
        <h3>Tags</h3>
        {!readOnly && <span className="tag-count">{selectedTags.length} selected</span>}
      </div>

      {/* Selected Tags Display */}
      <div className="selected-tags">
        {selectedTagObjects.map(tag => (
          <div key={tag.tagKey} className="tag-pill" style={{ borderColor: tag.colorCode }}>
            {tag.iconName && <span className="tag-icon">{tag.iconName}</span>}
            <span className="tag-label">{tag.tagLabel}</span>
            {!readOnly && (
              <button
                className="tag-remove"
                onClick={() => handleTagRemove(tag.tagKey)}
                title="Remove tag"
              >
                ×
              </button>
            )}
          </div>
        ))}
        {selectedTags.length === 0 && (
          <p className="no-tags-message">No tags assigned. Add tags to classify this term.</p>
        )}
      </div>

      {/* Tag Search & Selection */}
      {!readOnly && (
        <div className="tag-input-wrapper">
          <input
            type="text"
            placeholder="Search and add tags..."
            value={searchTerm}
            onChange={(e) => {
              setSearchTerm(e.target.value);
              setShowDropdown(true);
            }}
            onFocus={() => setShowDropdown(true)}
            className="tag-search-input"
          />

          {showDropdown && (
            <div className="tag-dropdown">
              {filteredTags.length > 0 ? (
                <div className="tag-dropdown-content">
                  {Object.entries(tagsByCategory).map(([category, tags]) => {
                    const filteredCategoryTags = tags.filter(t =>
                      filteredTags.some(ft => ft.tagKey === t.tagKey)
                    );

                    if (filteredCategoryTags.length === 0) return null;

                    return (
                      <div key={category} className="tag-category-group">
                        <div className="category-label">{category}</div>
                        {filteredCategoryTags.map(tag => (
                          <div
                            key={tag.tagKey}
                            className="tag-option"
                            onClick={() => handleTagSelect(tag.tagKey)}
                            style={{ borderLeftColor: tag.colorCode }}
                          >
                            <span className="tag-name">{tag.tagLabel}</span>
                            {tag.description && (
                              <span className="tag-description">{tag.description}</span>
                            )}
                          </div>
                        ))}
                      </div>
                    );
                  })}
                </div>
              ) : (
                <div className="tag-dropdown-empty">No tags match your search</div>
              )}
            </div>
          )}
        </div>
      )}

      {/* Tag Info Tooltip */}
      <div className="tags-info">
        <p className="info-text">
          Tags help classify and organize semantic terms for better discoverability and governance.
        </p>
      </div>
    </div>
  );
};

// ============================================================================
// TAG SUGGESTION WIZARD COMPONENT
// ============================================================================

interface TagWizardProps {
  termName: string;
  displayName?: string;
  description?: string;
  dataType?: string;
  domain?: string;
  expression?: string;
  existingTags?: string[];
  onApplySuggestions: (tags: string[]) => void;
  onCancel: () => void;
}

export const TagSuggestionWizard: React.FC<TagWizardProps> = ({
  termName,
  displayName,
  description,
  dataType,
  domain,
  expression,
  existingTags = [],
  onApplySuggestions,
  onCancel,
}) => {
  const [suggestions, setSuggestions] = useState<TagSuggestion[]>([]);
  const [selectedSuggestions, setSelectedSuggestions] = useState<Set<string>>(new Set());
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetchTagSuggestions();
  }, []);

  const fetchTagSuggestions = async () => {
    setLoading(true);
    setError(null);

    try {
      const response = await fetch('/api/graphql', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          query: `
            query SuggestTags($input: TagSuggestionInput!) {
              suggestSemanticTermTags(input: $input) {
                suggestions {
                  tagKey
                  tagLabel
                  tagCategory
                  suggestionReason
                  confidenceScore
                  colorCode
                  iconName
                }
                reasons
              }
            }
          `,
          variables: {
            input: {
              nodeName: termName,
              displayName,
              description,
              dataType,
              domain,
              expression,
              existingTags,
            },
          },
        }),
      });

      const data = await response.json();

      if (data.errors) {
        setError(data.errors[0]?.message || 'Failed to fetch suggestions');
      } else {
        const suggestionData = data.data?.suggestSemanticTermTags?.suggestions || [];
        setSuggestions(suggestionData);
        // Pre-select high-confidence suggestions (>0.8)
        const preSelected = new Set<string>(
          suggestionData
            .filter((s: TagSuggestion) => s.confidenceScore > 0.8)
            .map((s: TagSuggestion) => s.tagKey)
        );
        setSelectedSuggestions(preSelected);
      }
    } catch (err) {
      setError('Error fetching tag suggestions');
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const handleToggleSuggestion = (tagKey: string) => {
    const newSelected = new Set(selectedSuggestions);
    if (newSelected.has(tagKey)) {
      newSelected.delete(tagKey);
    } else {
      newSelected.add(tagKey);
    }
    setSelectedSuggestions(newSelected);
  };

  const handleApply = () => {
    onApplySuggestions(Array.from(selectedSuggestions));
  };

  const confidenceBarColor = (score: number) => {
    if (score > 0.8) return '#4CAF50'; // Green
    if (score > 0.6) return '#FFC107'; // Yellow
    return '#FF9800'; // Orange
  };

  return (
    <div className="tag-suggestion-wizard">
      <div className="wizard-header">
        <h2>Tag Suggestions</h2>
        <p>Smart recommendations for organizing your semantic term</p>
      </div>

      {loading && <div className="wizard-loading">Analyzing term characteristics...</div>}

      {error && <div className="wizard-error">{error}</div>}

      {!loading && suggestions.length === 0 && (
        <div className="wizard-empty">No tag suggestions available</div>
      )}

      {!loading && suggestions.length > 0 && (
        <div className="suggestions-list">
          {suggestions.map(suggestion => (
            <div
              key={suggestion.tagKey}
              className={`suggestion-item ${selectedSuggestions.has(suggestion.tagKey) ? 'selected' : ''}`}
            >
              <input
                type="checkbox"
                checked={selectedSuggestions.has(suggestion.tagKey)}
                onChange={() => handleToggleSuggestion(suggestion.tagKey)}
                className="suggestion-checkbox"
              />

              <div className="suggestion-content">
                <div className="suggestion-header">
                  <span
                    className="suggestion-tag"
                    style={{ backgroundColor: suggestion.colorCode }}
                  >
                    {suggestion.tagLabel}
                  </span>
                  <span className="suggestion-category">{suggestion.tagCategory}</span>
                </div>

                <div className="suggestion-confidence">
                  <span className="confidence-label">Confidence:</span>
                  <div className="confidence-bar">
                    <div
                      className="confidence-fill"
                      style={{
                        width: `${suggestion.confidenceScore * 100}%`,
                        backgroundColor: confidenceBarColor(suggestion.confidenceScore),
                      }}
                    />
                  </div>
                  <span className="confidence-value">
                    {(suggestion.confidenceScore * 100).toFixed(0)}%
                  </span>
                </div>

                <p className="suggestion-reason">
                  <strong>Reason:</strong> {suggestion.suggestionReason}
                </p>
              </div>
            </div>
          ))}
        </div>
      )}

      <div className="wizard-footer">
        <button className="btn btn-secondary" onClick={onCancel}>
          Cancel
        </button>
        <button
          className="btn btn-primary"
          onClick={handleApply}
          disabled={selectedSuggestions.size === 0}
        >
          Apply Selected Tags ({selectedSuggestions.size})
        </button>
      </div>
    </div>
  );
};

// ============================================================================
// TAG STATISTICS COMPONENT
// ============================================================================

interface TagStatisticsProps {
  termId: string;
}

export const TagStatistics: React.FC<TagStatisticsProps> = ({ termId }) => {
  const [stats, setStats] = useState<any>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchTagStatistics();
  }, [termId]);

  const fetchTagStatistics = async () => {
    try {
      const response = await fetch(`/api/semantic-terms/${termId}/tag-stats`);
      const data = await response.json();
      setStats(data);
    } catch (error) {
      console.error('Failed to fetch tag statistics:', error);
    } finally {
      setLoading(false);
    }
  };

  if (loading) return <div className="tag-stats-loading">Loading...</div>;
  if (!stats) return <div className="tag-stats-empty">No statistics available</div>;

  return (
    <div className="tag-statistics">
      <div className="stat-item">
        <span className="stat-label">Total Tags</span>
        <span className="stat-value">{stats.totalTags}</span>
      </div>
      <div className="stat-item">
        <span className="stat-label">Most Used Category</span>
        <span className="stat-value">{stats.mostUsedCategory}</span>
      </div>
      <div className="stat-item">
        <span className="stat-label">Suggested Tags</span>
        <span className="stat-value">{stats.suggestedCount}</span>
      </div>
    </div>
  );
};

// ============================================================================
// BATCH TAG MANAGER (for multiple terms)
// ============================================================================

interface BatchTagManagerProps {
  termIds: string[];
  onApply: (termIds: string[], tags: string[]) => void;
  onCancel: () => void;
}

export const BatchTagManager: React.FC<BatchTagManagerProps> = ({
  termIds,
  onApply,
  onCancel,
}) => {
  const [selectedTags, setSelectedTags] = useState<Set<string>>(new Set());
  const [availableTags, setAvailableTags] = useState<Tag[]>([]);

  useEffect(() => {
    fetchAvailableTags();
  }, []);

  const fetchAvailableTags = async () => {
    try {
      const response = await fetch('/api/graphql', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          query: `query { semanticTags { id tagKey tagLabel tagCategory colorCode } }`,
        }),
      });
      const data = await response.json();
      setAvailableTags(data.data?.semanticTags || []);
    } catch (error) {
      console.error('Failed to fetch tags:', error);
    }
  };

  const handleApply = () => {
    onApply(termIds, Array.from(selectedTags));
  };

  return (
    <div className="batch-tag-manager">
      <h3>Apply Tags to {termIds.length} Terms</h3>
      <div className="tag-selection">
        {availableTags.map(tag => (
          <label key={tag.tagKey} className="tag-checkbox">
            <input
              type="checkbox"
              checked={selectedTags.has(tag.tagKey)}
              onChange={(e) => {
                const newSelected = new Set(selectedTags);
                if (e.target.checked) {
                  newSelected.add(tag.tagKey);
                } else {
                  newSelected.delete(tag.tagKey);
                }
                setSelectedTags(newSelected);
              }}
            />
            <span className="tag-label">{tag.tagLabel}</span>
          </label>
        ))}
      </div>
      <div className="batch-actions">
        <button onClick={onCancel}>Cancel</button>
        <button onClick={handleApply} disabled={selectedTags.size === 0}>
          Apply to {termIds.length} Terms
        </button>
      </div>
    </div>
  );
};
