import React, { useState, useRef, useEffect } from 'react';
import * as TablerIcons from '@tabler/icons-react';
import './ProfessionalSearchInput.css';
import PayloadSearch from '../ProfessionalSearchInput';
import type { SearchResult } from '../../types/search';

export interface SearchSuggestion {
  id: string;
  title: string;
  subtitle?: string;
  type?: string;
  description?: string;
}

// Props for the legacy suggestion-based API
interface SuggestionModeProps {
  value: string;
  onChange: (value: string) => void;
  onClear?: () => void;
  placeholder?: string;
  suggestions?: SearchSuggestion[];
  onSuggestionSelect?: (suggestion: SearchSuggestion) => void;
  showSuggestions?: boolean;
  mode?: 'suggestions' | 'filter';
  onFocus?: () => void;
  onBlur?: () => void;
  onKeyDown?: (e: React.KeyboardEvent) => void;
  highlightedIndex?: number;
  onHighlightChange?: (index: number) => void;
  navigationEnabled?: boolean;
  currentMatch?: number;
  totalMatches?: number;
  onNavigateMatch?: (direction: 1 | -1) => void;
  disabled?: boolean;
  size?: 'sm' | 'md' | 'lg';
  variant?: 'default' | 'compact' | 'enhanced';
  className?: string;
  loading?: boolean;
  inputRef?: React.RefObject<HTMLInputElement>;
}

// Props for the payload/data-based API (new)
interface PayloadModeProps<T = any> {
  // payload style
  data: SearchResult<T>[];
  onSelect: (payload: T | undefined) => void;
  onSearch?: (query: string) => void;
  placeholder?: string;
  debounceMs?: number;
  initialSelected?: SearchResult<T> | null;
  className?: string;
}

// Union of both
type ProfessionalSearchInputProps = SuggestionModeProps | PayloadModeProps<any>;

export const ProfessionalSearchInput: React.FC<ProfessionalSearchInputProps> = (props) => {
  // If props contains 'data' and 'onSelect', treat it as payload mode and delegate to the new search component
  if ((props as any).data && (props as any).onSelect) {
    const p = props as PayloadModeProps<any>;
    return (
      <PayloadSearch
        placeholder={p.placeholder}
        data={p.data}
        onSelect={p.onSelect}
        onSearch={p.onSearch}
        className={p.className}
        debounceMs={p.debounceMs}
        initialSelected={p.initialSelected}
      />
    );
  }

  // Otherwise provide the legacy suggestion-based UI (preserve original behavior)
  const {
    value,
    onChange,
    onClear,
    placeholder = 'Search...',
    suggestions = [],
    onSuggestionSelect,
    showSuggestions = false,
    mode = 'suggestions',
    onFocus,
    onBlur,
    onKeyDown,
    highlightedIndex = -1,
    onHighlightChange,
    navigationEnabled = false,
    currentMatch,
    totalMatches,
    onNavigateMatch,
    disabled = false,
    size = 'md',
    variant = 'default',
    className = '',
    loading = false,
    inputRef,
  } = props as SuggestionModeProps;

  const [isFocused, setIsFocused] = useState(false);
  const internalInputRef = useRef<HTMLInputElement>(null);
  const actualInputRef = inputRef || internalInputRef;
  const suggestionsRef = useRef<HTMLDivElement>(null);
  const isClickingSuggestionRef = useRef(false);

  // Handle keyboard navigation
  const handleKeyDown = (e: React.KeyboardEvent) => {
    // Call external onKeyDown handler first
    onKeyDown?.(e as any);

    if (!showSuggestions || mode !== 'suggestions' || suggestions.length === 0) {
      // Handle match navigation when no suggestions are shown
      if (navigationEnabled && totalMatches && totalMatches > 0) {
        if (e.key === 'ArrowLeft' || (e.key === 'ArrowUp' && e.shiftKey)) {
          e.preventDefault();
          onNavigateMatch?.(-1);
        } else if (e.key === 'ArrowRight' || (e.key === 'ArrowDown' && e.shiftKey)) {
          e.preventDefault();
          onNavigateMatch?.(1);
        }
      }
      return;
    }

    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault();
        const nextIndex = highlightedIndex < suggestions.length - 1 ? highlightedIndex + 1 : 0;
        onHighlightChange?.(nextIndex);
        break;
      case 'ArrowUp':
        e.preventDefault();
        const prevIndex = highlightedIndex > 0 ? highlightedIndex - 1 : suggestions.length - 1;
        onHighlightChange?.(prevIndex);
        break;
      case 'Enter':
        e.preventDefault();
        if (highlightedIndex >= 0 && highlightedIndex < suggestions.length) {
          onSuggestionSelect?.(suggestions[highlightedIndex]);
        }
        break;
      case 'Escape':
        e.preventDefault();
        actualInputRef.current?.blur();
        break;
    }
  };

  const handleFocus = () => {
    setIsFocused(true);
    onFocus?.();
  };

  const handleBlur = () => {
    // Don't blur if we're clicking on a suggestion
    if (isClickingSuggestionRef.current) {
      return;
    }
    setIsFocused(false);
    onBlur?.();
  };

  const handleClear = () => {
    onChange('');
    onClear?.();
    actualInputRef.current?.focus();
  };

  // Scroll highlighted suggestion into view
  useEffect(() => {
    if (highlightedIndex >= 0 && suggestionsRef.current) {
      const suggestionElement = suggestionsRef.current.children[highlightedIndex] as HTMLElement;
      if (suggestionElement) {
        suggestionElement.scrollIntoView({
          block: 'nearest',
          behavior: 'smooth'
        });
      }
    }
  }, [highlightedIndex, suggestionsRef, suggestions]);

  const inputClasses = [
    'professional-search-input',
    `size-${size}`,
    `variant-${variant}`,
    isFocused ? 'focused' : '',
    disabled ? 'disabled' : '',
    className
  ].filter(Boolean).join(' ');

  const containerClasses = [
    'professional-search-container',
    (showSuggestions && mode === 'suggestions' && suggestions.length > 0) ? 'has-suggestions' : '',
    value ? 'has-value' : ''
  ].filter(Boolean).join(' ');

  return (
    <div className={containerClasses}>
      <div className={`search-input-wrapper ${isFocused ? 'focused' : ''} ${value ? 'has-value' : ''}`}>
        <TablerIcons.IconSearch size={16} className="search-icon" />

        <input
          ref={actualInputRef}
          type="text"
          className={inputClasses}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          onKeyDown={handleKeyDown}
          onFocus={handleFocus}
          onBlur={handleBlur}
          placeholder={placeholder}
          disabled={disabled}
        />

        {/* Navigation Controls */}
        {navigationEnabled && totalMatches && totalMatches > 0 && (
          <div className="search-navigation">
            <span className="match-counter">
              {currentMatch}/{totalMatches}
            </span>
            <button
              type="button"
              className="nav-btn"
              onClick={() => onNavigateMatch?.(-1)}
              disabled={totalMatches <= 1}
              title="Previous match"
            >
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <path d="m18 15-6-6-6 6"/>
              </svg>
            </button>
            <button
              type="button"
              className="nav-btn"
              onClick={() => onNavigateMatch?.(1)}
              disabled={totalMatches <= 1}
              title="Next match"
            >
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <path d="m6 9 6 6 6-6"/>
              </svg>
            </button>
          </div>
        )}

        {/* Clear Button: show when clear handler is provided */}
        {!disabled && onClear && (
          <button
            type="button"
            className="clear-btn"
            onClick={handleClear}
            title="Clear search"
          >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="m18 6-6 6 6 6"/>
              <path d="m6 6 6 6-6 6"/>
            </svg>
          </button>
        )}
      </div>

      {/* Suggestions Dropdown */}
      {showSuggestions && mode === 'suggestions' && suggestions.length > 0 && (
        <div ref={suggestionsRef} className="suggestions-dropdown">
          {suggestions.map((suggestion, index) => (
            <div
              key={suggestion.id}
              className={`suggestion-item ${index === highlightedIndex ? 'highlighted' : ''}`}
              onMouseDown={() => { isClickingSuggestionRef.current = true; }}
              onClick={() => {
                onSuggestionSelect?.(suggestion);
                isClickingSuggestionRef.current = false;
              }}
            >
              <div className="suggestion-content">
                <div className="suggestion-title">{suggestion.title}</div>
                {suggestion.subtitle && (
                  <div className="suggestion-subtitle">{suggestion.subtitle}</div>
                )}
                {suggestion.description && (
                  <div className="suggestion-description">{suggestion.description}</div>
                )}
              </div>
              {/* Score chip on the right if subtitle doesn't already include it */}
              {(suggestion as any).result && (typeof (suggestion as any).result.score === 'number') && (
                <div className="suggestion-score-chip">{Math.round(((suggestion as any).result.score || 0) * 100)}%</div>
              )}
              {suggestion.type && (
                <div className={`suggestion-type-chip ${suggestion.type}`}>
                  {suggestion.type}
                </div>
              )}
            </div>
          ))}
        </div>
      )}
      {/* Loading indicator */}
      {loading && (
        <div className="search-loading">
          <svg width="16" height="16" viewBox="0 0 50 50" className="spinner">
            <circle cx="25" cy="25" r="20" fill="none" strokeWidth="4" stroke="#666" strokeLinecap="round" strokeDasharray="31.4 31.4"/>
          </svg>
        </div>
      )}

      {/* No Results */}
      {showSuggestions && mode === 'suggestions' && suggestions.length === 0 && value.trim() && (
        <div className="suggestions-dropdown">
          <div className="no-results">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="no-results-icon">
              <circle cx="11" cy="11" r="8"/>
              <path d="M21 21l-4.35-4.35"/>
            </svg>
            <div className="no-results-text">No results found</div>
            <div className="no-results-hint">Try adjusting your search terms</div>
          </div>
        </div>
      )}
    </div>
  );
};

export default ProfessionalSearchInput;
