import React, { useState, useRef, useEffect } from 'react';
import { Search, Zap, Database, Target } from 'lucide-react';
import { 
  EnhancedSuggestion, 
  generateSemanticSuggestions,
  formatConfidence,
  handleEnhancedSuggestionSelect,
  testAbbreviationExpansion
} from '../utils/enhancedSemanticMatching';

interface EnhancedSearchInputProps {
  value: string;
  onChange: (value: string) => void;
  onSuggestionSelect?: (suggestion: any) => void;
  placeholder?: string;
  className?: string;
  disabled?: boolean;
}

export const EnhancedSearchInput: React.FC<EnhancedSearchInputProps> = ({
  value,
  onChange,
  onSuggestionSelect,
  placeholder = "Search with enhanced semantic matching...",
  className = "",
  disabled = false
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const [suggestions, setSuggestions] = useState<EnhancedSuggestion[]>([]);
  const [highlightedIndex, setHighlightedIndex] = useState(-1);
  const inputRef = useRef<HTMLInputElement>(null);
  const suggestionTimeoutRef = useRef<NodeJS.Timeout>();
  const containerRef = useRef<HTMLDivElement>(null);

  // Generate suggestions when search term changes
  useEffect(() => {
    if (suggestionTimeoutRef.current) {
      clearTimeout(suggestionTimeoutRef.current);
    }
    
    suggestionTimeoutRef.current = setTimeout(() => {
      if (value && value.length >= 2) {
        const newSuggestions = generateSemanticSuggestions(value);
        setSuggestions(newSuggestions);
        setIsOpen(newSuggestions.length > 0);
        setHighlightedIndex(-1);
      } else {
        setSuggestions([]);
        setIsOpen(false);
      }
    }, 150);

    return () => {
      if (suggestionTimeoutRef.current) {
        clearTimeout(suggestionTimeoutRef.current);
      }
    };
  }, [value]);

  // Handle keyboard navigation
  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (!isOpen || suggestions.length === 0) return;

    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault();
        setHighlightedIndex(prev => 
          prev < suggestions.length - 1 ? prev + 1 : 0
        );
        break;
      case 'ArrowUp':
        e.preventDefault();
        setHighlightedIndex(prev => 
          prev > 0 ? prev - 1 : suggestions.length - 1
        );
        break;
      case 'Enter':
        e.preventDefault();
        if (highlightedIndex >= 0) {
          handleSuggestionClick(suggestions[highlightedIndex]);
        }
        break;
      case 'Escape':
        setIsOpen(false);
        setHighlightedIndex(-1);
        break;
    }
  };

  const handleSuggestionClick = (suggestion: EnhancedSuggestion) => {
    if (onSuggestionSelect) {
      handleEnhancedSuggestionSelect(suggestion, onSuggestionSelect);
    }
    onChange(suggestion.title);
    setIsOpen(false);
    setHighlightedIndex(-1);
  };

  const handleInputFocus = () => {
    if (suggestions.length > 0) {
      setIsOpen(true);
    }
  };

  const handleInputBlur = () => {
    // Delay closing to allow suggestion clicks
    setTimeout(() => setIsOpen(false), 200);
  };

  // Test abbreviation expansion on double-click
  const handleDoubleClick = () => {
    if ((typeof process !== 'undefined' && process.env?.NODE_ENV === 'development') || import.meta.env.DEV) {
      testAbbreviationExpansion();
    }
  };

  return (
    <div className={`relative ${className}`} ref={containerRef}>
      {/* Input with enhanced styling */}
      <div className="relative group">
        <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
          <Search className="h-4 w-4 text-gray-400 group-focus-within:text-blue-500 transition-colors" />
        </div>
        
        <input
          ref={inputRef}
          type="text"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          onKeyDown={handleKeyDown}
          onFocus={handleInputFocus}
          onBlur={handleInputBlur}
          onDoubleClick={handleDoubleClick}
          placeholder={placeholder}
          disabled={disabled}
          className={`
            w-full pl-10 pr-4 py-2.5
            border border-gray-200 rounded-lg
            bg-white/50 backdrop-blur-sm
            text-gray-900 placeholder-gray-400
            transition-all duration-200
            focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500
            focus:bg-white focus:shadow-lg focus:shadow-blue-500/10
            hover:border-gray-300 hover:bg-white/80
            disabled:opacity-50 disabled:cursor-not-allowed
            ${isOpen ? 'ring-2 ring-blue-500/20 border-blue-500 bg-white shadow-lg' : ''}
          `}
        />
        
        {/* Enhanced indicator */}
        <div className="absolute inset-y-0 right-0 pr-3 flex items-center">
          <div className="flex items-center space-x-1">
            {value && suggestions.length > 0 && (
              <>
                <Zap className="h-3 w-3 text-yellow-500" />
                <span className="text-xs text-gray-500 bg-gray-100 px-1.5 py-0.5 rounded">
                  {suggestions.length}
                </span>
              </>
            )}
          </div>
        </div>
      </div>

      {/* Enhanced suggestions dropdown */}
      {isOpen && suggestions.length > 0 && (
        <div className="absolute top-full left-0 right-0 mt-1 z-50">
          <div className="bg-white/95 backdrop-blur-md rounded-xl border border-gray-200/80 shadow-2xl shadow-blue-500/10 overflow-hidden">
            {/* Header with enhancement info */}
            <div className="px-4 py-2 bg-gradient-to-r from-blue-50 to-purple-50 border-b border-gray-200/50">
              <div className="flex items-center justify-between text-xs text-gray-600">
                <div className="flex items-center space-x-2">
                  <Target className="h-3 w-3" />
                  <span>Enhanced Semantic Matching</span>
                </div>
                <div className="flex items-center space-x-1">
                  <Database className="h-3 w-3" />
                  <span>Profile-Aware</span>
                </div>
              </div>
            </div>

            {/* Suggestions list */}
            <div className="max-h-80 overflow-y-auto">
              {suggestions.map((suggestion, index) => (
                <div
                  key={suggestion.id}
                  onClick={() => handleSuggestionClick(suggestion)}
                  className={`
                    px-4 py-3 cursor-pointer transition-all duration-150
                    border-b border-gray-100/50 last:border-b-0
                    ${index === highlightedIndex 
                      ? 'bg-gradient-to-r from-blue-50 to-purple-50 border-blue-200' 
                      : 'hover:bg-gray-50/80'
                    }
                  `}
                >
                  <div className="flex items-start justify-between">
                    <div className="flex-1 min-w-0">
                      {/* Title with abbreviation indicator */}
                      <div className="flex items-center space-x-2">
                        <span className="font-medium text-gray-900 truncate">
                          {suggestion.title}
                        </span>
                        {suggestion.abbreviationExpanded && (
                          <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-yellow-100 text-yellow-800">
                            <Zap className="w-3 h-3 mr-1" />
                            Expanded
                          </span>
                        )}
                      </div>
                      
                      {/* Subtitle */}
                      <p className="text-sm text-gray-600 mt-1 truncate">
                        {suggestion.subtitle}
                      </p>
                      
                      {/* Match reason */}
                      {suggestion.matchReason && (
                        <p className="text-xs text-gray-500 mt-1">
                          {suggestion.matchReason}
                        </p>
                      )}
                      
                      {/* Profile data indicators */}
                      {suggestion.profileData && (
                        <div className="flex items-center space-x-3 mt-2 text-xs">
                          {suggestion.profileData.valueOverlap && suggestion.profileData.valueOverlap > 0 && (
                            <span className="text-green-600">
                              📊 {Math.round(suggestion.profileData.valueOverlap * 100)}% value overlap
                            </span>
                          )}
                          {suggestion.profileData.cardinalityMatch && suggestion.profileData.cardinalityMatch > 0.8 && (
                            <span className="text-blue-600">
                              🔢 Cardinality match
                            </span>
                          )}
                          {suggestion.profileData.dataTypeMatch && (
                            <span className="text-purple-600">
                              ⚡ Type compatible
                            </span>
                          )}
                        </div>
                      )}
                    </div>
                    
                    {/* Confidence indicators */}
                    <div className="flex flex-col items-end space-y-1 ml-4">
                      {suggestion.confidence && (
                        <div className="flex items-center space-x-2">
                          <span 
                            className={`w-3 h-3 rounded-full ${
                              suggestion.confidence >= 0.9 ? 'bg-green-500' :
                              suggestion.confidence >= 0.8 ? 'bg-blue-500' :
                              suggestion.confidence >= 0.7 ? 'bg-yellow-500' :
                              suggestion.confidence >= 0.6 ? 'bg-orange-500' : 'bg-red-500'
                            }`}
                          ></span>
                          <span 
                            className={`text-sm font-medium ${
                              suggestion.confidence >= 0.9 ? 'text-green-500' :
                              suggestion.confidence >= 0.8 ? 'text-blue-500' :
                              suggestion.confidence >= 0.7 ? 'text-yellow-600' :
                              suggestion.confidence >= 0.6 ? 'text-orange-500' : 'text-red-500'
                            }`}
                          >
                            {Math.round(suggestion.confidence * 100)}%
                          </span>
                        </div>
                      )}
                      
                      <div className="text-xs text-gray-500">
                        {formatConfidence(suggestion.confidence || 0)}
                      </div>
                      
                      {/* Detailed confidence breakdown */}
                      {suggestion.nameConfidence !== undefined && (
                        <div className="text-xs text-gray-400 space-y-0.5">
                          <div>Name: {Math.round(suggestion.nameConfidence * 100)}%</div>
                          {suggestion.profileConfidence !== undefined && suggestion.profileConfidence > 0 && (
                            <div>Profile: {Math.round(suggestion.profileConfidence * 100)}%</div>
                          )}
                          {suggestion.typeConfidence !== undefined && (
                            <div>Type: {Math.round(suggestion.typeConfidence * 100)}%</div>
                          )}
                        </div>
                      )}
                    </div>
                  </div>
                </div>
              ))}
            </div>
            
            {/* Footer */}
            <div className="px-4 py-2 bg-gray-50/80 border-t border-gray-200/50">
              <p className="text-xs text-gray-500 text-center">
                Enhanced matching includes abbreviation expansion and profile analysis
              </p>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};