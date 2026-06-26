import React, { useState } from 'react';
import { LineageService } from '../../../services/lineageService';
import './BusinessTermSearch.css';

interface SemanticAsset {
  id: string;
  node_name: string;
  description: string;
  parent_id?: string;
  properties: Record<string, unknown>;
}

interface BusinessTermSearchProps {
  onSearchResults: (results: SemanticAsset[]) => void;
  onClearSearch: () => void;
}

const BusinessTermSearch: React.FC<BusinessTermSearchProps> = ({
  onSearchResults,
  onClearSearch
}) => {
  const [query, setQuery] = useState('');
  const [category, setCategory] = useState('');
  const [status, setStatus] = useState('');
  const [tags, setTags] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const lineageService = new LineageService();

  const handleSearch = async () => {
    if (!query.trim() && !category && !status && !tags.trim()) {
      onClearSearch();
      return;
    }

    setIsLoading(true);
    setError(null);

    try {
      const tagArray = tags.trim() ? tags.split(',').map(tag => tag.trim()) : undefined;

      const results = await lineageService.searchBusinessTerms({
        query: query.trim() || undefined,
        category: category || undefined,
        status: status || undefined,
        tags: tagArray,
        limit: 50
      });

      // Transform the results to match the expected format
      const transformedResults: SemanticAsset[] = results.business_terms.map(term => ({
        id: term.id,
        node_name: term.name,
        description: term.description,
        properties: {
          business_term_id: term.id,
          name: term.name,
          description: term.description,
          category: term.category,
          sub_category: term.sub_category,
          owner: term.owner,
          steward: term.steward,
          status: term.status,
          version: term.version,
          tags: term.tags,
          parent_id: term.parent_id
        }
      }));

      onSearchResults(transformedResults);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Search failed');
      onClearSearch();
    } finally {
      setIsLoading(false);
    }
  };

  const handleClear = () => {
    setQuery('');
    setCategory('');
    setStatus('');
    setTags('');
    setError(null);
    onClearSearch();
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleSearch();
    }
  };

  return (
    <div className="business-term-search">
      <div className="search-inputs">
        <div className="search-field">
          <label htmlFor="query-input" className="search-label">Search Query</label>
          <input
            id="query-input"
            type="text"
            placeholder="Search business terms..."
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyPress={handleKeyPress}
            className="search-input"
          />
        </div>

        <div className="search-field">
          <label htmlFor="category-select" className="search-label">Category</label>
          <select
            id="category-select"
            value={category}
            onChange={(e) => setCategory(e.target.value)}
            className="search-select"
          >
            <option value="">All Categories</option>
            <option value="Business Entity">Business Entity</option>
            <option value="Financial">Financial</option>
            <option value="Operational">Operational</option>
            <option value="Technical">Technical</option>
          </select>
        </div>

        <div className="search-field">
          <label htmlFor="status-select" className="search-label">Status</label>
          <select
            id="status-select"
            value={status}
            onChange={(e) => setStatus(e.target.value)}
            className="search-select"
          >
            <option value="">All Statuses</option>
            <option value="draft">Draft</option>
            <option value="approved">Approved</option>
            <option value="deprecated">Deprecated</option>
            <option value="archived">Archived</option>
          </select>
        </div>

        <div className="search-field">
          <label htmlFor="tags-input" className="search-label">Tags</label>
          <input
            id="tags-input"
            type="text"
            placeholder="Tags (comma-separated)"
            value={tags}
            onChange={(e) => setTags(e.target.value)}
            onKeyPress={handleKeyPress}
            className="search-input"
          />
        </div>

        <div className="search-buttons">
          <button
            onClick={handleSearch}
            disabled={isLoading}
            className="search-button primary"
          >
            {isLoading ? '🔍 Searching...' : '🔍 Search'}
          </button>
          <button
            onClick={handleClear}
            className="search-button secondary"
          >
            Clear
          </button>
        </div>
      </div>

      {error && (
        <div className="search-error">
          <span className="error-icon">⚠️</span>
          {error}
        </div>
      )}
    </div>
  );
};

export default BusinessTermSearch;
