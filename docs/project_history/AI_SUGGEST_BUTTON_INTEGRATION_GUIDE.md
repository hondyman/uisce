# AI Suggest Button Integration Guide

**Date:** October 20, 2025  
**Status:** Implementation Ready  
**Target Component:** ValidationRuleEditor + AdvancedRuleConfiguration  

---

## Executive Summary

The AI Suggest Button feature provides intelligent automation for rule creation and optimization within the Fabric Builder validation system. This guide details optimal button placement, implementation strategies, and integration patterns for the existing backend rule engine.

---

## Table of Contents

1. [Button Placement Strategy](#button-placement-strategy)
2. [Component Architecture](#component-architecture)
3. [Integration Points](#integration-points)
4. [Implementation Examples](#implementation-examples)
5. [API Integration](#api-integration)
6. [State Management](#state-management)
7. [User Experience Flows](#user-experience-flows)

---

## Button Placement Strategy

### Recommended Locations (Priority Order)

#### **1. ValidationRuleEditor Header (PRIMARY)**
```
Location: /frontend/src/pages/bundles/ValidationRuleEditor.tsx
Position: Top-right of the editor panel, next to "Create Rule" button
Priority: HIGHEST
Rationale: Users see it immediately when creating/editing rules
```

**Visual Layout:**
```
┌─────────────────────────────────────────────────┐
│ Validation Rules                [AI ✨] [+ New] │ ← Button placement
├─────────────────────────────────────────────────┤
│                                                 │
│  Rule List / Editor                             │
│                                                 │
└─────────────────────────────────────────────────┘
```

#### **2. AdvancedRuleConfiguration Tabs (SECONDARY)**
```
Location: /frontend/src/components/validation/AdvancedRuleConfiguration.tsx
Position: Floating button in top-right corner of rule editor
Priority: HIGH
Rationale: Contextual suggestions while building rules
```

**Visual Layout:**
```
┌─────────────────────────────────────────────────┐
│ ◄ Rule Dependencies │ Cross-Entity [✨ AI Ideas] │ ← Button placement
├─────────────────────────────────────────────────┤
│                                                 │
│  Tab Content Area                               │
│                                                 │
└─────────────────────────────────────────────────┘
```

#### **3. AdvancedConditionBuilder (TERTIARY)**
```
Location: /frontend/src/components/validation/AdvancedConditionBuilder.tsx
Position: Inside condition group header, next to AND/OR selector
Priority: MEDIUM
Rationale: Help users build complex nested conditions
```

**Visual Layout:**
```
┌─────────────────────┐
│ Condition Group     │
│ [AND ▼] [✨ AI Help]│ ← Button placement
├─────────────────────┤
│ ○ Condition 1       │
│ ○ Condition 2       │
└─────────────────────┘
```

#### **4. RuleDependencyChain (OPTIONAL)**
```
Location: /frontend/src/components/validation/RuleDependencyChain.tsx
Position: Floating action button in bottom-right of component
Priority: LOW
Rationale: Suggest dependency patterns and detect conflicts
```

---

## Component Architecture

### AISuggestButton Component

```typescript
// frontend/src/components/validation/AISuggestButton.tsx

interface AISuggestButtonProps {
  context: 'rule_editor' | 'condition_builder' | 'dependency_chain' | 'cross_entity';
  entity?: string;
  existingRules?: ValidationRule[];
  onSuggestionApplied?: (suggestion: AISuggestion) => void;
  disabled?: boolean;
  variant?: 'icon' | 'button' | 'floating';
  tenantId?: string;
  datasourceId?: string;
}

interface AISuggestion {
  id: string;
  type: 'rule' | 'optimization' | 'conflict' | 'pattern' | 'dependency';
  title: string;
  description: string;
  confidence: number;
  reasoning: string;
  suggestedRule?: Partial<ValidationRule>;
  suggestedCondition?: ConditionGroup;
  impact?: string;
  action: () => void | Promise<void>;
}

interface AISuggestState {
  isLoading: boolean;
  isOpen: boolean;
  suggestions: AISuggestion[];
  error?: string;
  activeTab?: 'suggestions' | 'patterns' | 'insights';
}
```

### Button Variants

#### Variant 1: Icon Button (Compact)
```typescript
// For inline use in headers
<button
  className="p-2 hover:bg-purple-100 rounded-lg transition-colors"
  title="Get AI suggestions"
  onClick={handleOpenSuggestions}
>
  <Sparkles className="text-purple-600" size={20} />
</button>
```

#### Variant 2: Full Button (Prominent)
```typescript
// For rule editor headers
<button
  className="flex items-center gap-2 px-4 py-2 bg-gradient-to-r from-purple-600 to-blue-600 text-white rounded-lg hover:shadow-lg transition-all"
  onClick={handleOpenSuggestions}
>
  <Sparkles size={18} />
  <span>AI Ideas</span>
</button>
```

#### Variant 3: Floating Button (Non-intrusive)
```typescript
// For floating use in component corners
<button
  className="fixed bottom-6 right-6 w-14 h-14 bg-gradient-to-r from-purple-600 to-blue-600 text-white rounded-full shadow-lg hover:shadow-xl transition-shadow flex items-center justify-center"
  onClick={handleOpenSuggestions}
>
  <Sparkles size={24} />
</button>
```

---

## Integration Points

### 1. ValidationRuleEditor Integration

**File:** `/frontend/src/pages/bundles/ValidationRuleEditor.tsx`

```typescript
// Add to component state
const [aiSuggestionsOpen, setAiSuggestionsOpen] = useState(false);
const [selectedEntity, setSelectedEntity] = useState<string>(entity);

// Add button to header
<div className="flex items-center justify-between mb-4">
  <h2 className="text-xl font-semibold">Validation Rules</h2>
  <div className="flex gap-2">
    <AISuggestButton
      context="rule_editor"
      entity={selectedEntity}
      existingRules={rules}
      onSuggestionApplied={handleRuleGenerated}
      tenantId={tenantId}
      datasourceId={datasourceId}
      variant="button"
    />
    <button
      onClick={handleCreateRule}
      className="px-4 py-2 bg-blue-600 text-white rounded-lg"
    >
      + Create Rule
    </button>
  </div>
</div>
```

### 2. AdvancedRuleConfiguration Integration

**File:** `/frontend/src/components/validation/AdvancedRuleConfiguration.tsx`

```typescript
// Add to tab header
const TabHeader: React.FC = () => (
  <div className="flex items-center justify-between border-b border-gray-200 mb-4">
    <div className="flex gap-4">
      <button
        onClick={() => setActiveTab('dependencies')}
        className={activeTab === 'dependencies' ? 'border-b-2 border-purple-600' : ''}
      >
        Rule Dependencies
      </button>
      <button
        onClick={() => setActiveTab('cross-entity')}
        className={activeTab === 'cross-entity' ? 'border-b-2 border-purple-600' : ''}
      >
        Cross-Entity Validation
      </button>
    </div>
    <AISuggestButton
      context={activeTab === 'dependencies' ? 'dependency_chain' : 'cross_entity'}
      existingRules={rules}
      onSuggestionApplied={handleSuggestionApplied}
      variant="icon"
      tenantId={tenantId}
      datasourceId={datasourceId}
    />
  </div>
);
```

### 3. AdvancedConditionBuilder Integration

**File:** `/frontend/src/components/validation/AdvancedConditionBuilder.tsx`

```typescript
// Add within ConditionGroup header
const ConditionGroupHeader: React.FC<{ groupId: string }> = ({ groupId }) => (
  <div className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
    <div className="flex items-center gap-2">
      <select
        value={getGroupOperator(groupId)}
        onChange={(e) => updateGroupOperator(groupId, e.target.value)}
        className="px-2 py-1 border rounded"
      >
        <option>AND</option>
        <option>OR</option>
      </select>
    </div>
    <AISuggestButton
      context="condition_builder"
      onSuggestionApplied={(suggestion) => {
        if (suggestion.suggestedCondition) {
          addConditionToGroup(groupId, suggestion.suggestedCondition);
        }
      }}
      variant="icon"
      disabled={isMaxDepthReached(groupId)}
    />
  </div>
);
```

---

## Implementation Examples

### Complete AISuggestButton Component

```typescript
// frontend/src/components/validation/AISuggestButton.tsx

import React, { useState, useRef, useEffect } from 'react';
import { Sparkles, X, Loader } from 'lucide-react';
import { useQuery, useMutation } from '@apollo/client';
import { GET_AI_SUGGESTIONS, GENERATE_AI_RULE } from './graphql';

export const AISuggestButton: React.FC<AISuggestButtonProps> = ({
  context,
  entity,
  existingRules = [],
  onSuggestionApplied,
  disabled = false,
  variant = 'icon',
  tenantId,
  datasourceId
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const panelRef = useRef<HTMLDivElement>(null);
  const [suggestions, setSuggestions] = useState<AISuggestion[]>([]);
  const [loading, setLoading] = useState(false);
  const [activeTab, setActiveTab] = useState<'suggestions' | 'patterns' | 'insights'>('suggestions');

  // Query for AI suggestions
  const { data: suggestionsData } = useQuery(GET_AI_SUGGESTIONS, {
    variables: {
      tenantId,
      datasourceId,
      entity,
      context,
      existingRuleIds: existingRules.map(r => r.id)
    },
    skip: !isOpen || !entity,
    onCompleted: (data) => {
      setSuggestions(data.getAISuggestions.suggestions);
    }
  });

  // Mutation for generating rule from suggestion
  const [generateRule] = useMutation(GENERATE_AI_RULE, {
    onCompleted: (data) => {
      if (onSuggestionApplied) {
        onSuggestionApplied(data.generateAIRule);
      }
      setIsOpen(false);
    }
  });

  // Close panel when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (panelRef.current && !panelRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    };

    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
      return () => document.removeEventListener('mousedown', handleClickOutside);
    }
  }, [isOpen]);

  const handleAcceptSuggestion = async (suggestion: AISuggestion) => {
    setLoading(true);
    try {
      await generateRule({
        variables: {
          suggestionId: suggestion.id,
          tenantId,
          datasourceId
        }
      });
    } finally {
      setLoading(false);
    }
  };

  // Button rendering
  const renderButton = () => {
    if (variant === 'icon') {
      return (
        <button
          onClick={() => setIsOpen(!isOpen)}
          disabled={disabled}
          className="p-2 hover:bg-purple-100 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          title="Get AI suggestions"
        >
          <Sparkles className="text-purple-600" size={20} />
        </button>
      );
    }

    if (variant === 'button') {
      return (
        <button
          onClick={() => setIsOpen(!isOpen)}
          disabled={disabled}
          className="flex items-center gap-2 px-4 py-2 bg-gradient-to-r from-purple-600 to-blue-600 text-white rounded-lg hover:shadow-lg transition-all disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <Sparkles size={18} />
          <span>AI Ideas</span>
        </button>
      );
    }

    if (variant === 'floating') {
      return (
        <button
          onClick={() => setIsOpen(!isOpen)}
          disabled={disabled}
          className="fixed bottom-6 right-6 w-14 h-14 bg-gradient-to-r from-purple-600 to-blue-600 text-white rounded-full shadow-lg hover:shadow-xl transition-shadow flex items-center justify-center disabled:opacity-50"
        >
          <Sparkles size={24} />
        </button>
      );
    }
  };

  return (
    <div className="relative">
      {renderButton()}

      {/* Suggestions Panel */}
      {isOpen && (
        <div
          ref={panelRef}
          className="absolute right-0 top-full mt-2 w-96 bg-white rounded-lg shadow-xl border border-gray-200 z-50 max-h-96 overflow-y-auto"
        >
          {/* Panel Header */}
          <div className="sticky top-0 bg-gradient-to-r from-purple-600 to-blue-600 text-white p-4 flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Sparkles size={20} />
              <span className="font-semibold">AI Assistant</span>
            </div>
            <button
              onClick={() => setIsOpen(false)}
              className="p-1 hover:bg-white hover:bg-opacity-20 rounded"
            >
              <X size={18} />
            </button>
          </div>

          {/* Tabs */}
          <div className="flex border-b border-gray-200">
            {['suggestions', 'patterns', 'insights'].map((tab) => (
              <button
                key={tab}
                onClick={() => setActiveTab(tab as any)}
                className={`flex-1 px-4 py-3 text-sm font-medium border-b-2 transition-colors ${
                  activeTab === tab
                    ? 'border-purple-600 text-purple-600'
                    : 'border-transparent text-gray-600 hover:text-gray-900'
                }`}
              >
                {tab.charAt(0).toUpperCase() + tab.slice(1)}
              </button>
            ))}
          </div>

          {/* Loading State */}
          {suggestionsData?.getAISuggestions.loading && (
            <div className="p-8 flex flex-col items-center justify-center">
              <Loader className="animate-spin text-purple-600 mb-2" size={24} />
              <p className="text-sm text-gray-600">Analyzing your rules...</p>
            </div>
          )}

          {/* Suggestions Content */}
          {!suggestionsData?.getAISuggestions.loading && suggestions.length > 0 && (
            <div className="p-4 space-y-3">
              {suggestions.map((suggestion) => (
                <SuggestionCard
                  key={suggestion.id}
                  suggestion={suggestion}
                  onAccept={() => handleAcceptSuggestion(suggestion)}
                  loading={loading}
                />
              ))}
            </div>
          )}

          {/* Empty State */}
          {!suggestionsData?.getAISuggestions.loading && suggestions.length === 0 && (
            <div className="p-8 text-center">
              <p className="text-gray-600 text-sm">No suggestions at this time</p>
              <p className="text-xs text-gray-400 mt-2">
                Suggestions will appear as you build your rules
              </p>
            </div>
          )}
        </div>
      )}
    </div>
  );
};

// Suggestion Card Component
const SuggestionCard: React.FC<{
  suggestion: AISuggestion;
  onAccept: () => void;
  loading: boolean;
}> = ({ suggestion, onAccept, loading }) => {
  return (
    <div className="border border-gray-200 rounded-lg p-3 hover:border-purple-300 transition-colors">
      <div className="flex items-start gap-2 mb-2">
        <Sparkles className="text-purple-600 flex-shrink-0" size={16} />
        <div className="flex-1">
          <h4 className="font-semibold text-sm text-gray-900">{suggestion.title}</h4>
          <p className="text-xs text-gray-600 mt-1">{suggestion.description}</p>
        </div>
        <span className="px-2 py-0.5 bg-purple-100 text-purple-700 text-xs font-medium rounded">
          {Math.round(suggestion.confidence * 100)}%
        </span>
      </div>

      {suggestion.reasoning && (
        <details className="text-xs text-gray-600 mb-2">
          <summary className="cursor-pointer font-medium hover:text-gray-900">
            Why?
          </summary>
          <p className="mt-1 pl-2 border-l border-purple-300 text-gray-700">
            {suggestion.reasoning}
          </p>
        </details>
      )}

      <button
        onClick={onAccept}
        disabled={loading}
        className="w-full py-2 bg-purple-600 text-white text-xs font-semibold rounded hover:bg-purple-700 disabled:opacity-50 transition-colors"
      >
        {loading ? 'Applying...' : 'Apply'}
      </button>
    </div>
  );
};
```

---

## API Integration

### GraphQL Queries & Mutations

```graphql
# Backend GraphQL Schema

type AISuggestion {
  id: ID!
  type: SuggestionType!
  title: String!
  description: String!
  confidence: Float!
  reasoning: String!
  suggestedRule: ValidationRule
  suggestedCondition: ConditionGroup
  impact: String
}

enum SuggestionType {
  RULE
  OPTIMIZATION
  CONFLICT
  PATTERN
  DEPENDENCY
}

type AISuggestionsResponse {
  suggestions: [AISuggestion!]!
  loading: Boolean!
  timestamp: DateTime!
}

type Query {
  # Get AI suggestions for a context
  getAISuggestions(
    tenantId: ID!
    datasourceId: ID!
    entity: String!
    context: String!
    existingRuleIds: [ID!]
  ): AISuggestionsResponse!

  # Detect patterns in data
  detectDataPatterns(
    tenantId: ID!
    datasourceId: ID!
    entity: String!
    sampleSize: Int
  ): [DataPattern!]!

  # Detect conflicts in rules
  detectRuleConflicts(
    tenantId: ID!
    datasourceId: ID!
    ruleIds: [ID!]!
  ): [RuleConflict!]!

  # Generate insights from validation history
  generateValidationInsights(
    tenantId: ID!
    datasourceId: ID!
    entity: String!
    lookbackDays: Int
  ): [ValidationInsight!]!
}

type Mutation {
  # Generate rule from suggestion
  generateAIRule(
    suggestionId: ID!
    tenantId: ID!
    datasourceId: ID!
  ): ValidationRule!

  # Parse natural language to rule
  parseNaturalLanguageRule(
    description: String!
    entity: String!
    tenantId: ID!
    datasourceId: ID!
  ): ValidationRule!

  # Apply optimization suggestion
  applyOptimizationSuggestion(
    suggestionId: ID!
    affectedRuleIds: [ID!]!
    tenantId: ID!
    datasourceId: ID!
  ): [ValidationRule!]!
}
```

### Backend Implementation (Go)

```go
// internal/api/ai_suggestions.go

package api

import (
    "context"
    "github.com/graphql-go/graphql"
)

type AISuggestionService struct {
    db *sql.DB
    ml *MLEngine
}

func (s *AISuggestionService) GetAISuggestions(
    ctx context.Context,
    tenantID, datasourceID, entity, contextType string,
    existingRuleIDs []string,
) ([]AISuggestion, error) {
    suggestions := []AISuggestion{}

    // Get context data
    rules, err := s.db.GetValidationRules(ctx, tenantID, datasourceID, entity)
    if err != nil {
        return nil, err
    }

    // Generate suggestions based on context
    switch contextType {
    case "rule_editor":
        suggestions = append(suggestions, s.suggestMissingRules(ctx, entity, rules)...)
        suggestions = append(suggestions, s.suggestRuleOptimizations(ctx, rules)...)
        suggestions = append(suggestions, s.detectRuleConflicts(ctx, rules)...)

    case "condition_builder":
        suggestions = append(suggestions, s.suggestConditionPatterns(ctx, entity)...)

    case "dependency_chain":
        suggestions = append(suggestions, s.suggestDependencyPatterns(ctx, rules)...)
        suggestions = append(suggestions, s.validateDependencies(ctx, rules)...)

    case "cross_entity":
        suggestions = append(suggestions, s.suggestCrossEntityValidations(ctx, entity)...)
    }

    return suggestions, nil
}

func (s *AISuggestionService) DetectDataPatterns(
    ctx context.Context,
    tenantID, datasourceID, entity string,
    sampleSize int,
) ([]DataPattern, error) {
    // Query sample data
    records, err := s.db.GetSampleRecords(ctx, tenantID, datasourceID, entity, sampleSize)
    if err != nil {
        return nil, err
    }

    // Use ML model to detect patterns
    patterns, err := s.ml.DetectPatterns(records, entity)
    if err != nil {
        return nil, err
    }

    return patterns, nil
}

func (s *AISuggestionService) DetectRuleConflicts(
    ctx context.Context,
    tenantID, datasourceID string,
    ruleIDs []string,
) ([]RuleConflict, error) {
    // Fetch all rules
    rules := make([]*ValidationRule, len(ruleIDs))
    for i, ruleID := range ruleIDs {
        rule, err := s.db.GetValidationRule(ctx, ruleID)
        if err != nil {
            return nil, err
        }
        rules[i] = rule
    }

    // Detect conflicts
    conflicts := []RuleConflict{}
    for i := 0; i < len(rules); i++ {
        for j := i + 1; j < len(rules); j++ {
            if s.hasConflict(rules[i], rules[j]) {
                conflicts = append(conflicts, RuleConflict{
                    Rule1:     rules[i],
                    Rule2:     rules[j],
                    Severity:  "high",
                    Explanation: "These rules have contradictory conditions",
                })
            }
        }
    }

    return conflicts, nil
}
```

---

## State Management

### Redux Integration (Optional)

```typescript
// frontend/src/store/aiSuggestions.ts

import { createSlice, createAsyncThunk } from '@reduxjs/toolkit';

export const fetchAISuggestions = createAsyncThunk(
  'aiSuggestions/fetchSuggestions',
  async (
    { tenantId, datasourceId, entity, context },
    { rejectWithValue }
  ) => {
    try {
      const response = await fetch(
        `/api/ai/suggestions?tenant_id=${tenantId}&datasource_id=${datasourceId}&entity=${entity}&context=${context}`,
        {
          headers: {
            'X-Tenant-ID': tenantId,
            'X-Tenant-Datasource-ID': datasourceId
          }
        }
      );
      return await response.json();
    } catch (error) {
      return rejectWithValue(error.message);
    }
  }
);

const aiSuggestionsSlice = createSlice({
  name: 'aiSuggestions',
  initialState: {
    suggestions: [],
    loading: false,
    error: null,
    dismissedIds: []
  },
  extraReducers: (builder) => {
    builder
      .addCase(fetchAISuggestions.pending, (state) => {
        state.loading = true;
      })
      .addCase(fetchAISuggestions.fulfilled, (state, action) => {
        state.loading = false;
        state.suggestions = action.payload.suggestions;
      })
      .addCase(fetchAISuggestions.rejected, (state, action) => {
        state.loading = false;
        state.error = action.payload;
      });
  }
});

export default aiSuggestionsSlice.reducer;
```

---

## User Experience Flows

### Flow 1: Rule Creator Flow

```
User opens ValidationRuleEditor
        ↓
Sees "AI Ideas" button in header
        ↓
Clicks button → Panel opens with suggestions
        ↓
Options appear:
  • "Missing Email Validation" (92% confidence)
  • "Simplify Redundant Rules" (85% confidence)
  • "Detect Salary Anomalies" (78% confidence)
        ↓
User clicks "Apply" on "Missing Email Validation"
        ↓
Rule is generated and added to editor
        ↓
User reviews and fine-tunes the rule
        ↓
User saves rule
```

### Flow 2: Condition Builder Flow

```
User is building complex nested conditions
        ↓
Clicks AI help icon within condition group
        ↓
Panel shows:
  • "Common Email Validation Pattern"
  • "Suggested Operator: AND"
  • "Pre-built condition snippet"
        ↓
User accepts suggestion
        ↓
Condition is inserted into builder
```

### Flow 3: Conflict Detection Flow

```
User adds rule that conflicts with existing rule
        ↓
AI detects circular dependency or contradiction
        ↓
Shows conflict warning with "Review" button
        ↓
User clicks "Review"
        ↓
Panel shows:
  • Which rules conflict
  • Why they conflict
  • Suggested resolution
        ↓
User applies fix or dismisses warning
```

---

## Performance Considerations

### Caching Strategy
```typescript
// Cache suggestions for 5 minutes
const SUGGESTION_CACHE_TTL = 5 * 60 * 1000;

const suggestionCache = new Map<string, {
  data: AISuggestion[];
  timestamp: number;
}>();

function getCacheKey(tenantId: string, datasourceId: string, entity: string, context: string): string {
  return `${tenantId}:${datasourceId}:${entity}:${context}`;
}

async function getCachedSuggestions(
  tenantId: string,
  datasourceId: string,
  entity: string,
  context: string
): Promise<AISuggestion[] | null> {
  const key = getCacheKey(tenantId, datasourceId, entity, context);
  const cached = suggestionCache.get(key);

  if (cached && Date.now() - cached.timestamp < SUGGESTION_CACHE_TTL) {
    return cached.data;
  }

  return null;
}
```

### Lazy Loading
```typescript
// Only fetch suggestions when panel is opened
useEffect(() => {
  if (isOpen && !suggestions.length && !loading) {
    fetchSuggestions();
  }
}, [isOpen]);
```

---

## Accessibility

### WCAG 2.1 AA Compliance
```typescript
<button
  onClick={() => setIsOpen(!isOpen)}
  disabled={disabled}
  aria-label="Get AI suggestions for validation rules"
  aria-expanded={isOpen}
  aria-controls="ai-suggestions-panel"
  className="..."
>
  <Sparkles size={20} aria-hidden="true" />
  <span className="sr-only">AI Ideas</span>
</button>

<div
  id="ai-suggestions-panel"
  role="region"
  aria-label="AI suggestions"
  aria-live="polite"
  aria-busy={loading}
>
  {/* Panel content */}
</div>
```

---

## Security & Tenant Isolation

### Validation Rules
```typescript
// Ensure tenant isolation
const validateTenantAccess = (
  tenantId: string,
  datasourceId: string,
  userId: string,
  context: GraphQLResolverContext
): boolean => {
  // Verify user has access to tenant
  if (!context.userTenants.includes(tenantId)) {
    throw new Error('Unauthorized');
  }

  // Verify datasource belongs to tenant
  const datasource = context.db.getDatasource(datasourceId);
  if (datasource.tenant_id !== tenantId) {
    throw new Error('Datasource mismatch');
  }

  return true;
};

// Add to all AI suggestion queries
query.before(async (resolve, root, args, context) => {
  validateTenantAccess(args.tenantId, args.datasourceId, context.userId, context);
  return resolve();
});
```

---

## Testing Strategy

### Unit Tests

```typescript
// tests/AISuggestButton.test.tsx

describe('AISuggestButton', () => {
  it('renders icon button by default', () => {
    render(<AISuggestButton context="rule_editor" />);
    expect(screen.getByRole('button')).toBeInTheDocument();
  });

  it('opens panel when clicked', async () => {
    render(<AISuggestButton context="rule_editor" isOpen={false} />);
    fireEvent.click(screen.getByRole('button'));
    await waitFor(() => {
      expect(screen.getByRole('region')).toBeInTheDocument();
    });
  });

  it('fetches suggestions when panel opens', async () => {
    const { rerender } = render(
      <MockedProvider mocks={mocks}>
        <AISuggestButton context="rule_editor" tenantId="tenant1" />
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
          onSuggestionApplied={onApply}
        />
      </MockedProvider>
    );

    fireEvent.click(screen.getByRole('button'));
    
    await waitFor(() => {
      fireEvent.click(screen.getByText('Apply'));
    });

    expect(onApply).toHaveBeenCalled();
  });

  it('respects disabled state', () => {
    render(<AISuggestButton context="rule_editor" disabled={true} />);
    expect(screen.getByRole('button')).toBeDisabled();
  });
});
```

---

## Rollout Plan

### Phase 1: Beta (Week 1)
- Deploy to staging environment
- Test with internal team
- Gather feedback on placement and UX

### Phase 2: Limited Release (Week 2)
- Deploy to 10% of users
- Monitor performance and usage metrics
- Fix critical issues

### Phase 3: Full Release (Week 3)
- Deploy to all users
- Document in help center
- Train support team

---

## Monitoring & Analytics

### Event Tracking
```typescript
// Track user interactions
import { analytics } from '../services/analytics';

const trackAIEvent = (eventType: string, properties: Record<string, any>) => {
  analytics.track('ai_suggest_event', {
    eventType,
    timestamp: new Date(),
    ...properties
  });
};

// Usage examples
trackAIEvent('button_clicked', { context, variant });
trackAIEvent('suggestion_applied', { suggestionId, type });
trackAIEvent('suggestion_dismissed', { suggestionId });
trackAIEvent('suggestion_generated', { count, context });
```

---

## Next Steps

1. **Create AISuggestButton component** - Use provided implementation
2. **Integrate with ValidationRuleEditor** - Add button to header
3. **Create GraphQL resolvers** - Implement backend suggestions
4. **Add backend ML service** - Pattern detection and analysis
5. **Write tests** - Unit and integration tests
6. **Gather feedback** - Beta testing with team
7. **Deploy** - Gradual rollout strategy

---

**Status:** Ready for Implementation ✅  
**Last Updated:** October 20, 2025  
**Owner:** Fabric Builder Team
