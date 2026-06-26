/* eslint-disable jsx-a11y/aria-proptypes */
import React, { useState, useEffect, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import './ProfessionalSearchInput.css';
import type { SearchResult as SearchResultType } from '../types/search';

interface ProfessionalSearchInputProps<T = any> {
  placeholder?: string;
  data: SearchResultType<T>[];
  // onSelect returns the payload (full object) if provided, else undefined
  onSelect: (payload: T | undefined) => void;
  onSearch?: (query: string) => void;
  className?: string;
  debounceMs?: number;
  // optional initial selected item (useful for edit mode)
  initialSelected?: SearchResultType<T> | null;
  // If provided, the component will call this when scroll hits bottom (for infinite load)
  onLoadMore?: () => void;
}

export const ProfessionalSearchInput = <T,>({
  placeholder = 'Type to search...',
  data,
  onSelect,
  onSearch,
  className = '',
  debounceMs = 300,
  initialSelected = null,
  onLoadMore,
}: ProfessionalSearchInputProps<T>) => {
  const [query, setQuery] = useState(initialSelected?.text || '');
  const [isOpen, setIsOpen] = useState(false);
  const [filteredResults, setFilteredResults] = useState<SearchResultType<T>[]>([]);
  const [highlightedIndex, setHighlightedIndex] = useState(-1);
  const searchRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const { t } = useTranslation();

  // Update query when initialSelected changes (for edit mode)
  useEffect(() => {
    if (initialSelected?.text) {
      setQuery(initialSelected.text);
    }
  }, [initialSelected?.id]);

  // Debounced search effect
  useEffect(() => {
    const timer = setTimeout(() => {
      if (query.trim()) {
        // Filter results based on query
        const q = query.toLowerCase();
        const filtered = data.filter((item) =>
          item.text.toLowerCase().includes(q) ||
          (item.subtext && item.subtext.toLowerCase().includes(q))
        );
        setFilteredResults(filtered);
        setIsOpen(filtered.length > 0);
        onSearch?.(query);
      } else {
        setFilteredResults([]);
        setIsOpen(false);
      }
      setHighlightedIndex(-1);
    }, debounceMs);

    return () => clearTimeout(timer);
  }, [query, data, onSearch, debounceMs]);

  // Handle keyboard navigation
  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (!isOpen) return;

    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault();
        setHighlightedIndex((prev) => (prev < filteredResults.length - 1 ? prev + 1 : prev));
        break;
      case 'ArrowUp':
        e.preventDefault();
        setHighlightedIndex((prev) => (prev > 0 ? prev - 1 : -1));
        break;
      case 'Enter':
        e.preventDefault();
        if (highlightedIndex >= 0 && filteredResults[highlightedIndex]) {
          handleSelect(filteredResults[highlightedIndex]);
        } else if (filteredResults.length === 1) {
          // If only one result, allow Enter to select it
          handleSelect(filteredResults[0]);
        }
        break;
      case 'Escape':
        setIsOpen(false);
        setHighlightedIndex(-1);
        inputRef.current?.blur();
        break;
    }
  };

  const handleSelect = (item: SearchResultType<T>) => {
    setQuery(item.text);
    setIsOpen(false);
    setHighlightedIndex(-1);
    // Return only the payload (the full object) to simplify consumers. If no payload, return undefined.
    onSelect(item.payload ?? undefined);
    inputRef.current?.blur();
  };

  // Close dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (searchRef.current && !searchRef.current.contains(event.target as Node)) {
        setIsOpen(false);
        setHighlightedIndex(-1);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const highlightMatch = (text: string, query: string) => {
    if (!query.trim()) return text;
    
    const regex = new RegExp(`(${query})`, 'gi');
    const parts = text.split(regex);
    
    return parts.map((part, index) => 
      regex.test(part) ? (
        <mark key={index} className="search-highlight">{part}</mark>
      ) : (
        part
      )
    );
  };

  const handleClear = () => {
    setQuery('');
    setIsOpen(false);
    setHighlightedIndex(-1);
    onSearch?.('');
    inputRef.current?.focus();
  };

  return (
    <div ref={searchRef} className={`professional-search ${className}`}>
      <div className="search-input-container">
        <input
          ref={inputRef}
          type="text"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder={placeholder ?? t('search.placeholder', 'Type to search...')}
          className="search-input"
          autoComplete="off"
        />
        <div className="search-icon">
          🔍
        </div>
        {query && (
          <button
            type="button"
            onClick={handleClear}
            className="clear-button"
            title={t('search.clear', 'Clear search')}
            aria-label={t('search.clear', 'Clear search')}
          >
            ✕
          </button>
        )}
      </div>

      {isOpen && filteredResults.length > 0 && (
        <div className="search-dropdown">
          <div className="search-results" role="listbox" aria-label="Search results" aria-activedescendant={highlightedIndex >= 0 && filteredResults[highlightedIndex] ? `search-option-${filteredResults[highlightedIndex].id}` : undefined} onScroll={(e) => {
            const el = e.currentTarget as HTMLElement;
            if (onLoadMore && el.scrollTop + el.clientHeight >= el.scrollHeight - 10) {
              // trigger load more when near bottom
              onLoadMore();
            }
          }}>
            {filteredResults.map((item, index) => {
              const isSelected = index === highlightedIndex;
              const ariaSelected = isSelected ? 'true' : undefined;
              return (
              <div
                key={item.id}
                id={`search-option-${item.id}`}
                className={`search-result-item ${index === highlightedIndex ? 'highlighted' : ''}`}
                onClick={() => handleSelect(item)}
                onMouseEnter={() => setHighlightedIndex(index)}
                role="option"
                
                tabIndex={-1}
              >
                <div className="result-text">{highlightMatch(item.text, query)}</div>
                {item.subtext && <div className="result-subtext">{highlightMatch(item.subtext, query)}</div>}
              </div>
              );
            })}
          </div>
          
          {filteredResults.length > 0 && (
            <div className="search-footer">
              <span className="results-count">
                {t('search.results_count', { count: filteredResults.length })}
              </span>
                <span className="keyboard-hint">
                {t('search.keyboard_hint', '↑↓ to navigate • Enter to select • Esc to close')}
              </span>
            </div>
          )}
        </div>
      )}

      {isOpen && query.trim() && filteredResults.length === 0 && (
        <div className="search-dropdown">
          <div className="no-results">
            <div className="no-results-icon">🔍</div>
            <div className="no-results-text">{t('search.no_results', 'No results found')}</div>
            <div className="no-results-subtext">
              {t('search.no_results_subtext', 'Try adjusting your search terms')}
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default ProfessionalSearchInput;