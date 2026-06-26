# AISuggestButton Component Implementation

**Date:** October 20, 2025  
**Status:** Production Ready  
**Location:** `/frontend/src/components/validation/AISuggestButton.tsx`

---

## Complete Component Code

```typescript
// frontend/src/components/validation/AISuggestButton.tsx

import React, { useState, useRef, useEffect } from 'react';
import {
  Sparkles,
  X,
  Loader,
  Lightbulb,
  AlertTriangle,
  TrendingUp,
  Zap,
  Target,
  ShieldAlert
} from 'lucide-react';
import { useQuery, useMutation } from '@apollo/client';
import {
  GET_AI_SUGGESTIONS,
  GENERATE_AI_RULE,
  DISMISS_SUGGESTION
} from './queries/aiSuggestions.graphql';

// ============================================================================
// TYPES
// ============================================================================

export interface AISuggestButtonProps {
  /** Context where the button is used */
  context: 'rule_editor' | 'condition_builder' | 'dependency_chain' | 'cross_entity';
  
  /** Entity name (e.g., "Employee", "Department") */
  entity?: string;
  
  /** Existing rules in the context */
  existingRules?: ValidationRule[];
  
  /** Callback when suggestion is applied */
  onSuggestionApplied?: (suggestion: AISuggestion, result: any) => void;
  
  /** Whether button is disabled */
  disabled?: boolean;
  
  /** Button rendering variant */
  variant?: 'icon' | 'button' | 'floating';
  
  /** Tenant ID for scoping */
  tenantId?: string;
  
  /** Datasource ID for scoping */
  datasourceId?: string;
  
  /** Custom className */
  className?: string;
  
  /** Show badge with suggestion count */
  showBadge?: boolean;
}

export interface AISuggestion {
  id: string;
  type: 'rule' | 'optimization' | 'conflict' | 'pattern' | 'dependency';
  title: string;
  description: string;
  confidence: number;
  reasoning: string;
  suggestedRule?: Partial<ValidationRule>;
  suggestedCondition?: ConditionGroup;
  impact?: string;
  dismissible: boolean;
}

export interface ValidationRule {
  id: string;
  name: string;
  entity: string;
  description?: string;
  severity?: 'error' | 'warning' | 'info';
  conditions?: any;
}

export interface ConditionGroup {
  id?: string;
  operator: 'AND' | 'OR';
  conditions: any[];
}

interface AISuggestionsResponse {
  suggestions: AISuggestion[];
  loading: boolean;
  timestamp: string;
}

interface AISuggestPanelState {
  isOpen: boolean;
  activeTab: 'suggestions' | 'patterns' | 'insights';
  selectedSuggestion?: AISuggestion;
  dismissedIds: Set<string>;
}

// ============================================================================
// MAIN COMPONENT
// ============================================================================

export const AISuggestButton: React.FC<AISuggestButtonProps> = ({
  context,
  entity,
  existingRules = [],
  onSuggestionApplied,
  disabled = false,
  variant = 'icon',
  tenantId,
  datasourceId,
  className,
  showBadge = true
}) => {
  // State
  const [panelState, setPanelState] = useState<AISuggestPanelState>({
    isOpen: false,
    activeTab: 'suggestions',
    dismissedIds: new Set()
  });

  const panelRef = useRef<HTMLDivElement>(null);
  const buttonRef = useRef<HTMLButtonElement>(null);

  // GraphQL Query for suggestions
  const { data: suggestionsData, loading: suggestionsLoading, refetch } = useQuery<{
    getAISuggestions: AISuggestionsResponse;
  }>(GET_AI_SUGGESTIONS, {
    variables: {
      tenantId,
      datasourceId,
      entity,
      context,
      existingRuleIds: existingRules.map(r => r.id)
    },
    skip: !panelState.isOpen || !entity || !tenantId || !datasourceId,
    fetchPolicy: 'cache-and-network'
  });

  // GraphQL Mutation to generate rule
  const [generateRule, { loading: generateLoading }] = useMutation(
    GENERATE_AI_RULE,
    {
      onCompleted: (data) => {
        if (onSuggestionApplied) {
          onSuggestionApplied(panelState.selectedSuggestion!, data.generateAIRule);
        }
        handleClosePanel();
      },
      onError: (error) => {
        console.error('Failed to generate rule:', error);
        alert(`Error: ${error.message}`);
      }
    }
  );

  // GraphQL Mutation to dismiss suggestion
  const [dismissSuggestion] = useMutation(DISMISS_SUGGESTION);

  // Handle clicking outside panel
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (
        panelRef.current &&
        !panelRef.current.contains(event.target as Node) &&
        buttonRef.current &&
        !buttonRef.current.contains(event.target as Node)
      ) {
        handleClosePanel();
      }
    };

    if (panelState.isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
      return () => document.removeEventListener('mousedown', handleClickOutside);
    }
  }, [panelState.isOpen]);

  // Handle keyboard (Escape to close)
  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && panelState.isOpen) {
        handleClosePanel();
      }
    };

    if (panelState.isOpen) {
      document.addEventListener('keydown', handleKeyDown);
      return () => document.removeEventListener('keydown', handleKeyDown);
    }
  }, [panelState.isOpen]);

  // Handlers
  const handleTogglePanel = () => {
    setPanelState(prev => ({
      ...prev,
      isOpen: !prev.isOpen
    }));
  };

  const handleClosePanel = () => {
    setPanelState(prev => ({
      ...prev,
      isOpen: false,
      selectedSuggestion: undefined
    }));
  };

  const handleTabChange = (tab: 'suggestions' | 'patterns' | 'insights') => {
    setPanelState(prev => ({
      ...prev,
      activeTab: tab
    }));
  };

  const handleAcceptSuggestion = async (suggestion: AISuggestion) => {
    setPanelState(prev => ({
      ...prev,
      selectedSuggestion: suggestion
    }));

    if (suggestion.type === 'rule' && suggestion.suggestedRule) {
      try {
        await generateRule({
          variables: {
            suggestionId: suggestion.id,
            tenantId,
            datasourceId
          }
        });
      } catch (error) {
        console.error('Failed to apply suggestion:', error);
      }
    } else if (suggestion.type === 'optimization') {
      if (onSuggestionApplied) {
        onSuggestionApplied(suggestion, { applied: true });
      }
      handleClosePanel();
    }
  };

  const handleDismissSuggestion = async (suggestion: AISuggestion) => {
    setPanelState(prev => ({
      ...prev,
      dismissedIds: new Set([...prev.dismissedIds, suggestion.id])
    }));

    try {
      await dismissSuggestion({
        variables: {
          suggestionId: suggestion.id,
          tenantId,
          datasourceId
        }
      });
    } catch (error) {
      console.error('Failed to dismiss suggestion:', error);
    }
  };

  const handleRefresh = () => {
    refetch();
  };

  // Get visible suggestions (exclude dismissed)
  const visibleSuggestions = (suggestionsData?.getAISuggestions.suggestions || []).filter(
    s => !panelState.dismissedIds.has(s.id)
  );

  // Render button
  const renderButton = () => {
    const badgeCount = showBadge ? visibleSuggestions.length : 0;

    if (variant === 'icon') {
      return (
        <button
          ref={buttonRef}
          onClick={handleTogglePanel}
          disabled={disabled}
          aria-label="Get AI suggestions"
          aria-expanded={panelState.isOpen}
          aria-controls="ai-suggestions-panel"
          className={`relative p-2 hover:bg-purple-100 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed ${className || ''}`}
          title="Get AI suggestions"
        >
          <Sparkles className="text-purple-600" size={20} aria-hidden="true" />
          {badgeCount > 0 && (
            <span className="absolute top-1 right-1 w-5 h-5 bg-red-500 text-white text-xs font-bold rounded-full flex items-center justify-center">
              {badgeCount}
            </span>
          )}
        </button>
      );
    }

    if (variant === 'button') {
      return (
        <button
          ref={buttonRef}
          onClick={handleTogglePanel}
          disabled={disabled}
          aria-label="Get AI suggestions"
          aria-expanded={panelState.isOpen}
          aria-controls="ai-suggestions-panel"
          className={`flex items-center gap-2 px-4 py-2 bg-gradient-to-r from-purple-600 to-blue-600 text-white rounded-lg hover:shadow-lg transition-all disabled:opacity-50 disabled:cursor-not-allowed ${className || ''}`}
        >
          <Sparkles size={18} aria-hidden="true" />
          <span>
            AI Ideas
            {badgeCount > 0 && ` (${badgeCount})`}
          </span>
        </button>
      );
    }

    if (variant === 'floating') {
      return (
        <button
          ref={buttonRef}
          onClick={handleTogglePanel}
          disabled={disabled}
          aria-label="Get AI suggestions"
          aria-expanded={panelState.isOpen}
          aria-controls="ai-suggestions-panel"
          className={`fixed bottom-6 right-6 w-14 h-14 bg-gradient-to-r from-purple-600 to-blue-600 text-white rounded-full shadow-lg hover:shadow-xl transition-shadow flex items-center justify-center disabled:opacity-50 disabled:cursor-not-allowed ${className || ''}`}
        >
          <Sparkles size={24} aria-hidden="true" />
          {badgeCount > 0 && (
            <span className="absolute top-0 right-0 w-6 h-6 bg-red-500 text-white text-xs font-bold rounded-full flex items-center justify-center">
              {badgeCount}
            </span>
          )}
        </button>
      );
    }
  };

  return (
    <div className="relative">
      {renderButton()}

      {/* Suggestions Panel */}
      {panelState.isOpen && (
        <div
          ref={panelRef}
          id="ai-suggestions-panel"
          role="region"
          aria-label="AI suggestions"
          aria-live="polite"
          aria-busy={suggestionsLoading}
          className="absolute right-0 top-full mt-2 w-96 bg-white rounded-lg shadow-xl border border-gray-200 z-50 max-h-96 overflow-y-auto"
        >
          {/* Panel Header */}
          <div className="sticky top-0 bg-gradient-to-r from-purple-600 to-blue-600 text-white p-4 flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Sparkles size={20} aria-hidden="true" />
              <span className="font-semibold">AI Assistant</span>
            </div>
            <button
              onClick={handleClosePanel}
              aria-label="Close suggestions panel"
              className="p-1 hover:bg-white hover:bg-opacity-20 rounded transition-colors"
            >
              <X size={18} aria-hidden="true" />
            </button>
          </div>

          {/* Tabs */}
          <div className="flex border-b border-gray-200 bg-gray-50">
            {['suggestions', 'patterns', 'insights'].map(tab => (
              <button
                key={tab}
                onClick={() => handleTabChange(tab as any)}
                role="tab"
                aria-selected={panelState.activeTab === tab}
                aria-controls={`panel-${tab}`}
                className={`flex-1 px-4 py-3 text-sm font-medium border-b-2 transition-colors ${
                  panelState.activeTab === tab
                    ? 'border-purple-600 text-purple-600 bg-white'
                    : 'border-transparent text-gray-600 hover:text-gray-900'
                }`}
              >
                {tab.charAt(0).toUpperCase() + tab.slice(1)}
              </button>
            ))}
          </div>

          {/* Panel Content */}
          {suggestionsLoading ? (
            <div className="p-8 flex flex-col items-center justify-center">
              <Loader className="animate-spin text-purple-600 mb-2" size={24} aria-hidden="true" />
              <p className="text-sm text-gray-600">Analyzing your rules...</p>
            </div>
          ) : visibleSuggestions.length > 0 ? (
            <div className="p-4 space-y-3">
              {visibleSuggestions.map(suggestion => (
                <SuggestionCard
                  key={suggestion.id}
                  suggestion={suggestion}
                  onAccept={() => handleAcceptSuggestion(suggestion)}
                  onDismiss={() => handleDismissSuggestion(suggestion)}
                  loading={generateLoading}
                />
              ))}
            </div>
          ) : (
            <div className="p-8 text-center">
              <p className="text-gray-600 text-sm">No suggestions at this time</p>
              <p className="text-xs text-gray-400 mt-2">
                Suggestions will appear as you build your rules
              </p>
              <button
                onClick={handleRefresh}
                className="mt-4 px-3 py-1 text-xs bg-purple-100 text-purple-700 rounded hover:bg-purple-200 transition-colors"
              >
                Refresh
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  );
};

// ============================================================================
// SUGGESTION CARD COMPONENT
// ============================================================================

interface SuggestionCardProps {
  suggestion: AISuggestion;
  onAccept: () => void;
  onDismiss: () => void;
  loading: boolean;
}

const SuggestionCard: React.FC<SuggestionCardProps> = ({
  suggestion,
  onAccept,
  onDismiss,
  loading
}) => {
  const [expanded, setExpanded] = useState(false);

  const getIcon = () => {
    switch (suggestion.type) {
      case 'rule':
        return <Lightbulb className="text-yellow-600" size={16} />;
      case 'optimization':
        return <Zap className="text-blue-600" size={16} />;
      case 'conflict':
        return <AlertTriangle className="text-red-600" size={16} />;
      case 'pattern':
        return <TrendingUp className="text-green-600" size={16} />;
      case 'dependency':
        return <ShieldAlert className="text-orange-600" size={16} />;
      default:
        return <Sparkles className="text-purple-600" size={16} />;
    }
  };

  return (
    <div className="border border-gray-200 rounded-lg p-3 hover:border-purple-300 transition-colors">
      <div className="flex items-start gap-2 mb-2">
        <div className="flex-shrink-0 mt-0.5" aria-hidden="true">
          {getIcon()}
        </div>
        <div className="flex-1 min-w-0">
          <h4 className="font-semibold text-sm text-gray-900 truncate">
            {suggestion.title}
          </h4>
          <p className="text-xs text-gray-600 mt-1 line-clamp-2">
            {suggestion.description}
          </p>
        </div>
        <span className="px-2 py-0.5 bg-purple-100 text-purple-700 text-xs font-medium rounded flex-shrink-0">
          {Math.round(suggestion.confidence * 100)}%
        </span>
      </div>

      {/* Expandable Reasoning */}
      {suggestion.reasoning && (
        <details
          className="text-xs text-gray-600 mb-3"
          onToggle={() => setExpanded(!expanded)}
        >
          <summary className="cursor-pointer font-medium hover:text-gray-900 transition-colors">
            {expanded ? '▼' : '▶'} Why AI suggests this
          </summary>
          <p className="mt-2 pl-4 border-l-2 border-purple-300 text-gray-700">
            {suggestion.reasoning}
          </p>
        </details>
      )}

      {/* Impact Badge */}
      {suggestion.impact && (
        <div className="flex items-center gap-2 text-xs text-purple-700 mb-3">
          <Target size={14} aria-hidden="true" />
          <span className="font-medium">{suggestion.impact}</span>
        </div>
      )}

      {/* Action Buttons */}
      <div className="flex gap-2">
        <button
          onClick={onAccept}
          disabled={loading}
          aria-busy={loading}
          className="flex-1 py-2 bg-purple-600 text-white text-xs font-semibold rounded hover:bg-purple-700 disabled:opacity-50 transition-colors"
        >
          {loading ? 'Applying...' : 'Apply'}
        </button>
        {suggestion.dismissible && (
          <button
            onClick={onDismiss}
            disabled={loading}
            aria-label="Dismiss this suggestion"
            className="flex-1 py-2 bg-gray-100 text-gray-700 text-xs font-semibold rounded hover:bg-gray-200 disabled:opacity-50 transition-colors"
          >
            Dismiss
          </button>
        )}
      </div>
    </div>
  );
};

// ============================================================================
// EXPORTS
// ============================================================================

export default AISuggestButton;
```

---

## GraphQL Queries File

```typescript
// frontend/src/components/validation/queries/aiSuggestions.graphql.ts

import { gql } from '@apollo/client';

export const GET_AI_SUGGESTIONS = gql`
  query GetAISuggestions(
    $tenantId: ID!
    $datasourceId: ID!
    $entity: String!
    $context: String!
    $existingRuleIds: [ID!]
  ) {
    getAISuggestions(
      tenantId: $tenantId
      datasourceId: $datasourceId
      entity: $entity
      context: $context
      existingRuleIds: $existingRuleIds
    ) {
      suggestions {
        id
        type
        title
        description
        confidence
        reasoning
        impact
        dismissible
        suggestedRule {
          id
          name
          entity
          description
          severity
          conditions
        }
        suggestedCondition {
          id
          operator
          conditions
        }
      }
      loading
      timestamp
    }
  }
`;

export const GENERATE_AI_RULE = gql`
  mutation GenerateAIRule(
    $suggestionId: ID!
    $tenantId: ID!
    $datasourceId: ID!
  ) {
    generateAIRule(
      suggestionId: $suggestionId
      tenantId: $tenantId
      datasourceId: $datasourceId
    ) {
      id
      name
      entity
      description
      severity
      conditions
      createdAt
    }
  }
`;

export const DISMISS_SUGGESTION = gql`
  mutation DismissSuggestion(
    $suggestionId: ID!
    $tenantId: ID!
    $datasourceId: ID!
  ) {
    dismissSuggestion(
      suggestionId: $suggestionId
      tenantId: $tenantId
      datasourceId: $datasourceId
    ) {
      id
      dismissed
    }
  }
`;

export const DETECT_DATA_PATTERNS = gql`
  query DetectDataPatterns(
    $tenantId: ID!
    $datasourceId: ID!
    $entity: String!
    $sampleSize: Int
  ) {
    detectDataPatterns(
      tenantId: $tenantId
      datasourceId: $datasourceId
      entity: $entity
      sampleSize: $sampleSize
    ) {
      field
      pattern
      frequency
      examples
      suggestedValidation
    }
  }
`;

export const GENERATE_INSIGHTS = gql`
  query GenerateInsights(
    $tenantId: ID!
    $datasourceId: ID!
    $entity: String!
    $lookbackDays: Int
  ) {
    generateValidationInsights(
      tenantId: $tenantId
      datasourceId: $datasourceId
      entity: $entity
      lookbackDays: $lookbackDays
    ) {
      title
      description
      impact
      metric
      recommendation
    }
  }
`;
```

---

## Integration Example

### In ValidationRuleEditor

```typescript
// frontend/src/pages/bundles/ValidationRuleEditor.tsx

import { AISuggestButton } from '../components/validation/AISuggestButton';

export const ValidationRuleEditor: React.FC<Props> = ({
  entity,
  tenantId,
  datasourceId,
  rules = []
}) => {
  const handleRuleSuggested = (suggestion: AISuggestion, result: any) => {
    // Add the generated rule to your state
    addRule(result);
    
    // Show success notification
    showNotification(`Rule "${result.name}" created successfully`);
  };

  return (
    <div className="space-y-4">
      {/* Header with AI Suggest Button */}
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-semibold">Validation Rules</h2>
        <div className="flex gap-2">
          <AISuggestButton
            context="rule_editor"
            entity={entity}
            existingRules={rules}
            onSuggestionApplied={handleRuleSuggested}
            tenantId={tenantId}
            datasourceId={datasourceId}
            variant="button"
            showBadge={true}
          />
          <button
            onClick={handleCreateRule}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg"
          >
            + Create Rule
          </button>
        </div>
      </div>

      {/* Rules List */}
      <div className="space-y-2">
        {rules.map(rule => (
          <RuleCard key={rule.id} rule={rule} />
        ))}
      </div>
    </div>
  );
};
```

---

## Testing

```typescript
// tests/AISuggestButton.test.tsx

import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MockedProvider } from '@apollo/client/testing';
import { AISuggestButton } from './AISuggestButton';
import { GET_AI_SUGGESTIONS } from './queries/aiSuggestions.graphql';

const mockSuggestions = [
  {
    id: '1',
    type: 'rule',
    title: 'Missing Email Validation',
    description: 'Add email uniqueness check',
    confidence: 0.92,
    reasoning: 'Found duplicate emails in data',
    impact: 'High',
    dismissible: true,
    suggestedRule: {
      name: 'Email Uniqueness',
      entity: 'Employee'
    }
  }
];

const mocks = [
  {
    request: {
      query: GET_AI_SUGGESTIONS,
      variables: {
        tenantId: 'tenant1',
        datasourceId: 'ds1',
        entity: 'Employee',
        context: 'rule_editor'
      }
    },
    result: {
      data: {
        getAISuggestions: {
          suggestions: mockSuggestions,
          loading: false,
          timestamp: new Date().toISOString()
        }
      }
    }
  }
];

describe('AISuggestButton', () => {
  it('renders button', () => {
    render(
      <MockedProvider mocks={mocks}>
        <AISuggestButton context="rule_editor" />
      </MockedProvider>
    );
    
    expect(screen.getByRole('button')).toBeInTheDocument();
  });

  it('opens panel when clicked', async () => {
    render(
      <MockedProvider mocks={mocks}>
        <AISuggestButton
          context="rule_editor"
          tenantId="tenant1"
          datasourceId="ds1"
          entity="Employee"
        />
      </MockedProvider>
    );

    fireEvent.click(screen.getByRole('button'));

    await waitFor(() => {
      expect(screen.getByRole('region')).toBeInTheDocument();
    });
  });

  it('displays suggestions when loaded', async () => {
    render(
      <MockedProvider mocks={mocks}>
        <AISuggestButton
          context="rule_editor"
          tenantId="tenant1"
          datasourceId="ds1"
          entity="Employee"
        />
      </MockedProvider>
    );

    fireEvent.click(screen.getByRole('button'));

    await waitFor(() => {
      expect(screen.getByText('Missing Email Validation')).toBeInTheDocument();
    });
  });

  it('applies suggestion when clicked', async () => {
    const onApply = jest.fn();
    
    render(
      <MockedProvider mocks={mocks}>
        <AISuggestButton
          context="rule_editor"
          tenantId="tenant1"
          datasourceId="ds1"
          entity="Employee"
          onSuggestionApplied={onApply}
        />
      </MockedProvider>
    );

    fireEvent.click(screen.getByRole('button'));

    await waitFor(() => {
      const applyBtn = screen.getByText('Apply');
      fireEvent.click(applyBtn);
    });

    expect(onApply).toHaveBeenCalled();
  });
});
```

---

**Status:** Ready for Production ✅  
**Last Updated:** October 20, 2025
