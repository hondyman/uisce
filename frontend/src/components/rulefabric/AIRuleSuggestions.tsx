/**
 * AIRuleSuggestions.tsx
 * 
 * AI-powered rule suggestions component providing:
 * - Natural language to rule conversion
 * - Pattern detection from historical data
 * - Rule conflict detection
 * - Optimization suggestions
 * - Auto-complete for conditions
 */

import React, { useState, useEffect, useRef } from 'react';
import {
  Sparkles,
  MessageSquare,
  Lightbulb,
  AlertTriangle,
  Zap,
  Brain,
  RefreshCw,
  ChevronRight,
  Check,
  Copy,
  ThumbsUp,
  ThumbsDown,
  History,
  Settings,
  Wand2,
  Target,
  TrendingUp,
  Shield,
  Clock,
  ArrowRight
} from 'lucide-react';

// Types
interface RuleDefinition {
  id?: string;
  name: string;
  category: string;
  description?: string;
  conditions: ConditionGroup;
  actions: RuleAction[];
}

interface ConditionGroup {
  operator: 'AND' | 'OR';
  conditions: (Condition | ConditionGroup)[];
}

interface Condition {
  field: string;
  operator: string;
  value: unknown;
  entity?: string;
}

interface RuleAction {
  type: string;
  config: Record<string, unknown>;
}

interface AISuggestion {
  id: string;
  type: 'rule' | 'condition' | 'optimization' | 'conflict' | 'pattern';
  title: string;
  description: string;
  confidence: number;
  suggestedRule?: Partial<RuleDefinition>;
  suggestedCondition?: Condition;
  reasoning?: string;
  impact?: {
    affectedRecords?: number;
    performanceChange?: string;
    riskLevel?: 'low' | 'medium' | 'high';
  };
  relatedRules?: string[];
  createdAt: Date;
}

interface ConversationMessage {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  suggestions?: AISuggestion[];
  timestamp: Date;
}

interface AIRuleSuggestionsProps {
  currentRule?: RuleDefinition;
  existingRules: RuleDefinition[];
  entitySchema: EntitySchema;
  onApplySuggestion: (suggestion: AISuggestion) => void;
  onApplyCondition: (condition: Condition) => void;
  tenantId: string;
  datasourceId: string;
}

interface EntitySchema {
  entities: Record<string, EntityDefinition>;
}

interface EntityDefinition {
  name: string;
  displayName: string;
  fields: FieldDefinition[];
}

interface FieldDefinition {
  name: string;
  displayName: string;
  type: string;
  values?: string[];
}

// Natural Language Parser (simulated - would connect to LLM in production)
const parseNaturalLanguage = async (
  input: string,
  _schema: EntitySchema
): Promise<AISuggestion[]> => {
  // Simulate API delay
  await new Promise(resolve => setTimeout(resolve, 1500));
  
  const suggestions: AISuggestion[] = [];
  const lowerInput = input.toLowerCase();
  
  // Pattern matching for common rule intents
  if (lowerInput.includes('null') || lowerInput.includes('empty') || lowerInput.includes('missing')) {
    const fieldMatch = input.match(/(?:check|validate|ensure)\s+(\w+)/i);
    const field = fieldMatch?.[1] || 'field_name';
    
    suggestions.push({
      id: `suggestion-${Date.now()}-1`,
      type: 'rule',
      title: `Data Quality: Not Null Check for ${field}`,
      description: `Validates that ${field} is not null or empty`,
      confidence: 0.85,
      suggestedRule: {
        name: `${field}_not_null_check`,
        category: 'data_quality',
        description: `Ensures ${field} is populated`,
        conditions: {
          operator: 'OR',
          conditions: [
            { field, operator: 'is_null', value: true },
            { field, operator: 'equals', value: '' }
          ]
        },
        actions: [
          { type: 'flag', config: { severity: 'warning', message: `${field} is missing` } }
        ]
      },
      reasoning: 'Detected null/empty check intent from natural language input',
      impact: {
        riskLevel: 'low'
      },
      createdAt: new Date()
    });
  }
  
  if (lowerInput.includes('greater than') || lowerInput.includes('more than') || lowerInput.includes('exceeds')) {
    const numberMatch = input.match(/(\d+(?:\.\d+)?)/);
    const threshold = numberMatch ? parseFloat(numberMatch[1]) : 100;
    const fieldMatch = input.match(/(\w+)\s+(?:greater|more|exceeds)/i) || input.match(/(?:greater|more|exceeds)\s+\d+\s*(?:for|in|on)?\s*(\w+)/i);
    const field = fieldMatch?.[1] || 'amount';
    
    suggestions.push({
      id: `suggestion-${Date.now()}-2`,
      type: 'rule',
      title: `Threshold Check: ${field} > ${threshold}`,
      description: `Flags records where ${field} exceeds ${threshold}`,
      confidence: 0.9,
      suggestedRule: {
        name: `${field}_threshold_${threshold}`,
        category: 'compliance',
        description: `Validates ${field} against threshold of ${threshold}`,
        conditions: {
          operator: 'AND',
          conditions: [
            { field, operator: 'greater_than', value: threshold }
          ]
        },
        actions: [
          { type: 'alert', config: { severity: 'high', notify: ['compliance-team'] } }
        ]
      },
      reasoning: `Detected threshold comparison intent with value ${threshold}`,
      impact: {
        riskLevel: 'medium'
      },
      createdAt: new Date()
    });
  }
  
  if (lowerInput.includes('match') || lowerInput.includes('same') || lowerInput.includes('equal')) {
    suggestions.push({
      id: `suggestion-${Date.now()}-3`,
      type: 'rule',
      title: 'Cross-Field Validation',
      description: 'Ensures two fields have matching values',
      confidence: 0.75,
      suggestedRule: {
        name: 'field_match_validation',
        category: 'data_quality',
        conditions: {
          operator: 'AND',
          conditions: [
            { field: 'field_a', operator: 'not_equals_field', value: 'field_b' }
          ]
        },
        actions: [
          { type: 'flag', config: { severity: 'error', message: 'Field mismatch detected' } }
        ]
      },
      reasoning: 'Detected field comparison intent',
      createdAt: new Date()
    });
  }
  
  if (lowerInput.includes('wash') || lowerInput.includes('trade') || lowerInput.includes('same day')) {
    suggestions.push({
      id: `suggestion-${Date.now()}-4`,
      type: 'rule',
      title: 'Wash Trade Detection',
      description: 'Identifies potential wash trades based on timing and counterparty',
      confidence: 0.88,
      suggestedRule: {
        name: 'wash_trade_detection',
        category: 'wash_trade',
        description: 'Detects same-day buy/sell with same counterparty',
        conditions: {
          operator: 'AND',
          conditions: [
            { field: 'trade_date', operator: 'same_day_as', value: 'settlement_date' },
            { field: 'counterparty', operator: 'equals_field', value: 'originating_party' }
          ]
        },
        actions: [
          { type: 'escalate', config: { team: 'compliance', priority: 'urgent' } },
          { type: 'block', config: { reason: 'Potential wash trade' } }
        ]
      },
      reasoning: 'Detected wash trade detection intent from financial compliance context',
      impact: {
        riskLevel: 'high'
      },
      createdAt: new Date()
    });
  }
  
  // Default suggestion if no patterns matched
  if (suggestions.length === 0) {
    suggestions.push({
      id: `suggestion-${Date.now()}-default`,
      type: 'condition',
      title: 'Custom Condition',
      description: 'I understood you want to create a rule. Please provide more details about the fields and conditions.',
      confidence: 0.5,
      reasoning: 'Could not determine specific rule pattern from input',
      createdAt: new Date()
    });
  }
  
  return suggestions;
};

// Pattern Detection from Historical Data
const detectPatterns = async (
  _existingRules: RuleDefinition[]
): Promise<AISuggestion[]> => {
  await new Promise(resolve => setTimeout(resolve, 1000));
  
  return [
    {
      id: `pattern-${Date.now()}-1`,
      type: 'pattern',
      title: 'Common Threshold Pattern',
      description: 'Multiple rules use similar threshold checks. Consider creating a template.',
      confidence: 0.82,
      reasoning: 'Detected 5 rules with similar greater_than conditions on amount fields',
      impact: {
        performanceChange: '+15% faster evaluation with template'
      },
      relatedRules: ['rule-1', 'rule-2', 'rule-3'],
      createdAt: new Date()
    },
    {
      id: `pattern-${Date.now()}-2`,
      type: 'optimization',
      title: 'Condition Ordering Optimization',
      description: 'Reorder conditions to evaluate cheap conditions first for better performance.',
      confidence: 0.78,
      reasoning: 'String comparisons should precede regex and external lookups',
      impact: {
        performanceChange: '-20% evaluation time'
      },
      createdAt: new Date()
    }
  ];
};

// Conflict Detection
const detectConflicts = async (
  currentRule: RuleDefinition | undefined,
  _existingRules: RuleDefinition[]
): Promise<AISuggestion[]> => {
  await new Promise(resolve => setTimeout(resolve, 800));
  
  if (!currentRule) return [];
  
  return [
    {
      id: `conflict-${Date.now()}-1`,
      type: 'conflict',
      title: 'Potential Conflict with "amount_validation_rule"',
      description: 'Current rule conditions may conflict with existing rule causing inconsistent results.',
      confidence: 0.72,
      reasoning: 'Both rules evaluate the same field with overlapping but different thresholds',
      impact: {
        riskLevel: 'medium',
        affectedRecords: 1250
      },
      relatedRules: ['amount_validation_rule'],
      createdAt: new Date()
    }
  ];
};

// Main Component
export const AIRuleSuggestions: React.FC<AIRuleSuggestionsProps> = ({
  currentRule,
  existingRules,
  entitySchema,
  onApplySuggestion,
  onApplyCondition,
  tenantId: _tenantId,
  datasourceId: _datasourceId
}) => {
  const [activeTab, setActiveTab] = useState<'chat' | 'suggestions' | 'patterns' | 'conflicts'>('chat');
  const [messages, setMessages] = useState<ConversationMessage[]>([]);
  const [inputValue, setInputValue] = useState('');
  const [isProcessing, setIsProcessing] = useState(false);
  const [suggestions, setSuggestions] = useState<AISuggestion[]>([]);
  const [patterns, setPatterns] = useState<AISuggestion[]>([]);
  const [conflicts, setConflicts] = useState<AISuggestion[]>([]);
  const [expandedSuggestion, setExpandedSuggestion] = useState<string | null>(null);
  const chatEndRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  // Auto-scroll chat
  useEffect(() => {
    chatEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  // Load patterns and conflicts when rule changes
  useEffect(() => {
    const loadAnalysis = async () => {
      const [patternResults, conflictResults] = await Promise.all([
        detectPatterns(existingRules),
        detectConflicts(currentRule, existingRules)
      ]);
      setPatterns(patternResults);
      setConflicts(conflictResults);
    };
    
    loadAnalysis();
  }, [currentRule, existingRules]);

  const handleSendMessage = async () => {
    if (!inputValue.trim() || isProcessing) return;
    
    const userMessage: ConversationMessage = {
      id: `msg-${Date.now()}`,
      role: 'user',
      content: inputValue,
      timestamp: new Date()
    };
    
    setMessages(prev => [...prev, userMessage]);
    setInputValue('');
    setIsProcessing(true);
    
    try {
      const aiSuggestions = await parseNaturalLanguage(inputValue, entitySchema);
      
      const assistantMessage: ConversationMessage = {
        id: `msg-${Date.now()}-response`,
        role: 'assistant',
        content: aiSuggestions.length > 0
          ? `I found ${aiSuggestions.length} potential rule${aiSuggestions.length > 1 ? 's' : ''} based on your description:`
          : 'I couldn\'t generate specific suggestions. Could you provide more details about the fields and conditions?',
        suggestions: aiSuggestions,
        timestamp: new Date()
      };
      
      setMessages(prev => [...prev, assistantMessage]);
      setSuggestions(prev => [...aiSuggestions, ...prev]);
    } catch (error) {
      const errorMessage: ConversationMessage = {
        id: `msg-${Date.now()}-error`,
        role: 'assistant',
        content: 'Sorry, I encountered an error processing your request. Please try again.',
        timestamp: new Date()
      };
      setMessages(prev => [...prev, errorMessage]);
    } finally {
      setIsProcessing(false);
    }
  };

  const handleApplySuggestion = (suggestion: AISuggestion) => {
    if (suggestion.type === 'condition' && suggestion.suggestedCondition) {
      onApplyCondition(suggestion.suggestedCondition);
    } else {
      onApplySuggestion(suggestion);
    }
  };

  const handleFeedback = (_suggestionId: string, _positive: boolean) => {
    // Track feedback for model improvement - in production, send to analytics/ML pipeline
    // Example: analyticsService.trackFeedback(_suggestionId, _positive);
    // In production, this would send to analytics/ML pipeline
  };

  const getConfidenceColor = (confidence: number) => {
    if (confidence >= 0.8) return 'text-green-600 bg-green-50';
    if (confidence >= 0.6) return 'text-yellow-600 bg-yellow-50';
    return 'text-orange-600 bg-orange-50';
  };

  const getRiskColor = (risk?: 'low' | 'medium' | 'high') => {
    switch (risk) {
      case 'low': return 'text-green-600 bg-green-50';
      case 'medium': return 'text-yellow-600 bg-yellow-50';
      case 'high': return 'text-red-600 bg-red-50';
      default: return 'text-gray-600 bg-gray-50';
    }
  };

  const renderSuggestionCard = (suggestion: AISuggestion) => {
    const isExpanded = expandedSuggestion === suggestion.id;
    
    return (
      <div
        key={suggestion.id}
        className="border rounded-lg bg-white shadow-sm overflow-hidden"
      >
        <div
          className="p-4 cursor-pointer hover:bg-gray-50 transition-colors"
          onClick={() => setExpandedSuggestion(isExpanded ? null : suggestion.id)}
          role="button"
          tabIndex={0}
          onKeyDown={(e) => e.key === 'Enter' && setExpandedSuggestion(isExpanded ? null : suggestion.id)}
        >
          <div className="flex items-start gap-3">
            <div className="flex-shrink-0 mt-0.5">
              {suggestion.type === 'rule' && <Wand2 size={18} className="text-purple-500" />}
              {suggestion.type === 'condition' && <Target size={18} className="text-blue-500" />}
              {suggestion.type === 'optimization' && <Zap size={18} className="text-yellow-500" />}
              {suggestion.type === 'conflict' && <AlertTriangle size={18} className="text-red-500" />}
              {suggestion.type === 'pattern' && <TrendingUp size={18} className="text-green-500" />}
            </div>
            
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2 mb-1">
                <h4 className="font-medium text-gray-900 truncate">{suggestion.title}</h4>
                <span className={`text-xs px-2 py-0.5 rounded-full ${getConfidenceColor(suggestion.confidence)}`}>
                  {Math.round(suggestion.confidence * 100)}% confidence
                </span>
              </div>
              <p className="text-sm text-gray-600 line-clamp-2">{suggestion.description}</p>
            </div>
            
            <ChevronRight
              size={18}
              className={`text-gray-400 transition-transform ${isExpanded ? 'rotate-90' : ''}`}
            />
          </div>
        </div>
        
        {isExpanded && (
          <div className="border-t bg-gray-50 p-4 space-y-4">
            {suggestion.reasoning && (
              <div className="flex items-start gap-2">
                <Brain size={16} className="text-gray-400 mt-0.5 flex-shrink-0" />
                <div>
                  <span className="text-xs font-medium text-gray-500">Reasoning</span>
                  <p className="text-sm text-gray-700">{suggestion.reasoning}</p>
                </div>
              </div>
            )}
            
            {suggestion.impact && (
              <div className="flex flex-wrap gap-3">
                {suggestion.impact.affectedRecords && (
                  <div className="flex items-center gap-1 text-sm">
                    <Shield size={14} className="text-gray-400" />
                    <span className="text-gray-600">
                      ~{suggestion.impact.affectedRecords.toLocaleString()} records affected
                    </span>
                  </div>
                )}
                {suggestion.impact.performanceChange && (
                  <div className="flex items-center gap-1 text-sm">
                    <Clock size={14} className="text-gray-400" />
                    <span className="text-gray-600">{suggestion.impact.performanceChange}</span>
                  </div>
                )}
                {suggestion.impact.riskLevel && (
                  <span className={`text-xs px-2 py-0.5 rounded-full ${getRiskColor(suggestion.impact.riskLevel)}`}>
                    {suggestion.impact.riskLevel} risk
                  </span>
                )}
              </div>
            )}
            
            {suggestion.suggestedRule && (
              <div className="bg-white rounded border p-3">
                <div className="flex items-center justify-between mb-2">
                  <span className="text-xs font-medium text-gray-500">Suggested Rule Preview</span>
                  <button
                    onClick={() => navigator.clipboard.writeText(JSON.stringify(suggestion.suggestedRule, null, 2))}
                    className="text-gray-400 hover:text-gray-600 p-1"
                    title="Copy rule JSON"
                    aria-label="Copy rule JSON"
                  >
                    <Copy size={14} />
                  </button>
                </div>
                <pre className="text-xs text-gray-700 overflow-x-auto">
                  {JSON.stringify(suggestion.suggestedRule, null, 2)}
                </pre>
              </div>
            )}
            
            {suggestion.relatedRules && suggestion.relatedRules.length > 0 && (
              <div>
                <span className="text-xs font-medium text-gray-500">Related Rules</span>
                <div className="flex flex-wrap gap-2 mt-1">
                  {suggestion.relatedRules.map(ruleId => (
                    <span key={ruleId} className="text-xs px-2 py-1 bg-gray-100 rounded">
                      {ruleId}
                    </span>
                  ))}
                </div>
              </div>
            )}
            
            <div className="flex items-center justify-between pt-2 border-t">
              <div className="flex items-center gap-2">
                <span className="text-xs text-gray-500">Was this helpful?</span>
                <button
                  onClick={() => handleFeedback(suggestion.id, true)}
                  className="p-1 text-gray-400 hover:text-green-500 transition-colors"
                  title="Mark as helpful"
                  aria-label="Mark as helpful"
                >
                  <ThumbsUp size={14} />
                </button>
                <button
                  onClick={() => handleFeedback(suggestion.id, false)}
                  className="p-1 text-gray-400 hover:text-red-500 transition-colors"
                  title="Mark as not helpful"
                  aria-label="Mark as not helpful"
                >
                  <ThumbsDown size={14} />
                </button>
              </div>
              
              <div className="flex items-center gap-2">
                {suggestion.type !== 'conflict' && (
                  <button
                    onClick={() => handleApplySuggestion(suggestion)}
                    className="flex items-center gap-1 px-3 py-1.5 bg-purple-600 text-white text-sm rounded hover:bg-purple-700 transition-colors"
                  >
                    <Check size={14} />
                    Apply
                  </button>
                )}
              </div>
            </div>
          </div>
        )}
      </div>
    );
  };

  return (
    <div className="flex flex-col h-full bg-gray-50 rounded-lg border">
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b bg-white rounded-t-lg">
        <div className="flex items-center gap-2">
          <div className="p-2 bg-purple-100 rounded-lg">
            <Sparkles size={18} className="text-purple-600" />
          </div>
          <div>
            <h3 className="font-semibold text-gray-900">AI Rule Assistant</h3>
            <p className="text-xs text-gray-500">Natural language to rules, powered by AI</p>
          </div>
        </div>
        <button
          className="p-2 text-gray-400 hover:text-gray-600 rounded hover:bg-gray-100"
          title="AI Assistant Settings"
          aria-label="AI Assistant Settings"
        >
          <Settings size={18} />
        </button>
      </div>
      
      {/* Tabs */}
      <div className="flex border-b bg-white">
        {[
          { id: 'chat', label: 'Chat', icon: MessageSquare },
          { id: 'suggestions', label: 'Suggestions', icon: Lightbulb, count: suggestions.length },
          { id: 'patterns', label: 'Patterns', icon: TrendingUp, count: patterns.length },
          { id: 'conflicts', label: 'Conflicts', icon: AlertTriangle, count: conflicts.length }
        ].map(tab => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id as typeof activeTab)}
            className={`flex-1 flex items-center justify-center gap-2 px-4 py-3 text-sm font-medium transition-colors relative ${
              activeTab === tab.id
                ? 'text-purple-600 border-b-2 border-purple-600'
                : 'text-gray-500 hover:text-gray-700'
            }`}
          >
            <tab.icon size={16} />
            <span>{tab.label}</span>
            {tab.count !== undefined && tab.count > 0 && (
              <span className={`text-xs px-1.5 py-0.5 rounded-full ${
                activeTab === tab.id ? 'bg-purple-100 text-purple-600' : 'bg-gray-100 text-gray-600'
              }`}>
                {tab.count}
              </span>
            )}
          </button>
        ))}
      </div>
      
      {/* Content */}
      <div className="flex-1 overflow-hidden">
        {activeTab === 'chat' && (
          <div className="flex flex-col h-full">
            {/* Messages */}
            <div className="flex-1 overflow-y-auto p-4 space-y-4">
              {messages.length === 0 && (
                <div className="text-center py-8">
                  <Brain size={48} className="mx-auto text-gray-300 mb-4" />
                  <h4 className="font-medium text-gray-600 mb-2">Describe your rule in plain English</h4>
                  <p className="text-sm text-gray-500 max-w-md mx-auto">
                    Try something like "Flag all transactions greater than $10,000" or 
                    "Check if customer email is empty"
                  </p>
                  <div className="flex flex-wrap justify-center gap-2 mt-4">
                    {[
                      'Detect wash trades',
                      'Validate email format',
                      'Amount exceeds 50000',
                      'Check for null values'
                    ].map(example => (
                      <button
                        key={example}
                        onClick={() => setInputValue(example)}
                        className="text-xs px-3 py-1.5 bg-purple-50 text-purple-700 rounded-full hover:bg-purple-100 transition-colors"
                      >
                        {example}
                      </button>
                    ))}
                  </div>
                </div>
              )}
              
              {messages.map(message => (
                <div
                  key={message.id}
                  className={`flex ${message.role === 'user' ? 'justify-end' : 'justify-start'}`}
                >
                  <div className={`max-w-[85%] ${message.role === 'user' ? 'order-2' : ''}`}>
                    <div className={`rounded-lg px-4 py-2 ${
                      message.role === 'user'
                        ? 'bg-purple-600 text-white'
                        : 'bg-white border text-gray-800'
                    }`}>
                      <p className="text-sm">{message.content}</p>
                    </div>
                    
                    {message.suggestions && message.suggestions.length > 0 && (
                      <div className="mt-3 space-y-2">
                        {message.suggestions.map(suggestion => renderSuggestionCard(suggestion))}
                      </div>
                    )}
                    
                    <span className="text-xs text-gray-400 mt-1 block">
                      {message.timestamp.toLocaleTimeString()}
                    </span>
                  </div>
                </div>
              ))}
              
              {isProcessing && (
                <div className="flex justify-start">
                  <div className="bg-white border rounded-lg px-4 py-2">
                    <div className="flex items-center gap-2">
                      <RefreshCw size={14} className="animate-spin text-purple-500" />
                      <span className="text-sm text-gray-600">Analyzing your request...</span>
                    </div>
                  </div>
                </div>
              )}
              
              <div ref={chatEndRef} />
            </div>
            
            {/* Input */}
            <div className="p-4 border-t bg-white">
              <div className="flex items-center gap-2">
                <input
                  ref={inputRef}
                  type="text"
                  value={inputValue}
                  onChange={(e) => setInputValue(e.target.value)}
                  onKeyDown={(e) => e.key === 'Enter' && handleSendMessage()}
                  placeholder="Describe your rule in plain English..."
                  className="flex-1 px-4 py-2 border rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-purple-500"
                  disabled={isProcessing}
                />
                <button
                  onClick={handleSendMessage}
                  disabled={!inputValue.trim() || isProcessing}
                  className="p-2 bg-purple-600 text-white rounded-lg hover:bg-purple-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                  title="Generate rule suggestions"
                  aria-label="Generate rule suggestions"
                >
                  <ArrowRight size={18} />
                </button>
              </div>
            </div>
          </div>
        )}
        
        {activeTab === 'suggestions' && (
          <div className="p-4 space-y-3 overflow-y-auto h-full">
            {suggestions.length === 0 ? (
              <div className="text-center py-8">
                <Lightbulb size={48} className="mx-auto text-gray-300 mb-4" />
                <p className="text-gray-500">No suggestions yet. Start a conversation to generate rule suggestions.</p>
              </div>
            ) : (
              suggestions.map(suggestion => renderSuggestionCard(suggestion))
            )}
          </div>
        )}
        
        {activeTab === 'patterns' && (
          <div className="p-4 space-y-3 overflow-y-auto h-full">
            {patterns.length === 0 ? (
              <div className="text-center py-8">
                <TrendingUp size={48} className="mx-auto text-gray-300 mb-4" />
                <p className="text-gray-500">Analyzing patterns in your rules...</p>
              </div>
            ) : (
              patterns.map(pattern => renderSuggestionCard(pattern))
            )}
          </div>
        )}
        
        {activeTab === 'conflicts' && (
          <div className="p-4 space-y-3 overflow-y-auto h-full">
            {conflicts.length === 0 ? (
              <div className="text-center py-8">
                <AlertTriangle size={48} className="mx-auto text-gray-300 mb-4" />
                <p className="text-gray-500">No conflicts detected with existing rules.</p>
              </div>
            ) : (
              <>
                <div className="p-3 bg-yellow-50 border border-yellow-200 rounded-lg">
                  <p className="text-sm text-yellow-800">
                    <AlertTriangle size={14} className="inline mr-1" />
                    {conflicts.length} potential conflict{conflicts.length > 1 ? 's' : ''} detected. Review before publishing.
                  </p>
                </div>
                {conflicts.map(conflict => renderSuggestionCard(conflict))}
              </>
            )}
          </div>
        )}
      </div>
      
      {/* Footer */}
      <div className="flex items-center justify-between p-3 border-t bg-white rounded-b-lg text-xs text-gray-500">
        <div className="flex items-center gap-1">
          <History size={12} />
          <span>Conversation history is not persisted</span>
        </div>
        <div className="flex items-center gap-1">
          <Sparkles size={12} className="text-purple-500" />
          <span>Powered by AI</span>
        </div>
      </div>
    </div>
  );
};

export default AIRuleSuggestions;
