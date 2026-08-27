import { useState, useCallback } from 'react';
import {
  SemanticField,
  ExplorerQueryDefinition,
  ChartViewMode,
  ChatMessage,
  AIInsightSummary,
} from '../types/explorerTypes';
import { apiFetch } from '../../../lib/apiClient';

interface UseExplorerAIProps {
  catalog: SemanticField[];
  currentQuery: ExplorerQueryDefinition;
  onQueryChange: (query: ExplorerQueryDefinition) => void;
  onViewModeChange: (mode: ChartViewMode) => void;
  onExecuteQuery: (query: ExplorerQueryDefinition) => void;
  userContext?: { accountId?: string; tenantId?: string };
  /**
   * Optional hook fired whenever a chat message is appended.
   * Used by DataExplorerPage to mirror the thread into the
   * conversation store (history / saved / shared / folders).
   */
  onMessage?: (message: ChatMessage) => void;
  /**
   * Optional messages override — when provided, the hook uses
   * external state and onMessage becomes the source of truth.
   */
  messages?: ChatMessage[];
}

export function useExplorerAI({
  catalog,
  currentQuery,
  onQueryChange,
  onViewModeChange,
  onExecuteQuery,
  userContext,
  onMessage,
  messages: messagesOverride,
}: UseExplorerAIProps) {
  const [internalMessages, setMessages] = useState<ChatMessage[]>([]);
  const messages = messagesOverride ?? internalMessages;
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [insight, setInsight] = useState<AIInsightSummary | null>(null);
  const [isCacheHit, setIsCacheHit] = useState<boolean>(false);
  const [suggestedFollowUps, setSuggestedFollowUps] = useState<string[]>([
    'Break down by Region',
    'Filter for Active status only',
    'Show monthly trend over 2026',
  ]);
  const [ambiguities, setAmbiguities] = useState<string[]>([]);

  /**
   * Deterministic client-side heuristic engine for when backend LLM is unreachable or in mock mode
   */
  const localSynthesize = (prompt: string, prevQuery: ExplorerQueryDefinition): ExplorerQueryDefinition => {
    const p = prompt.toLowerCase();
    const dims = new Set<string>(prevQuery.dimensions);
    const measures = [...prevQuery.measures];
    const timeDims = [...prevQuery.timeDimensions];
    const filters = [...prevQuery.filters];

    // If asking to break down / group by
    for (const f of catalog) {
      const match = p.includes(f.name.toLowerCase()) || p.includes(f.label.toLowerCase());
      if (match) {
        if (f.category === 'dimension') {
          dims.add(f.id);
        } else if (f.category === 'time') {
          if (!timeDims.some((t) => t.fieldId === f.id)) {
            timeDims.push({ fieldId: f.id, granularity: 'month' });
          }
        } else if (f.category === 'measure') {
          if (!measures.some((m) => m.fieldId === f.id)) {
            let agg: any = f.aggregation || 'SUM';
            if (p.includes('avg') || p.includes('average')) agg = 'AVG';
            if (p.includes('count')) agg = 'COUNT';
            measures.push({ fieldId: f.id, agg });
          }
        }
      }
    }

    // Status filter
    if (p.includes('active')) {
      if (!filters.some((flt) => flt.fieldId === 'status')) {
        filters.push({
          id: `f-${Date.now()}`,
          fieldId: 'status',
          operator: '=',
          value: 'Active',
        });
      }
    }

    // Determine visual mode
    let suggestedChart: ChartViewMode = 'table';
    if (timeDims.length > 0 && measures.length > 0) {
      suggestedChart = 'line';
    } else if (dims.size === 0 && measures.length === 1) {
      suggestedChart = 'kpi';
    } else if (dims.size === 1 && measures.length >= 1) {
      suggestedChart = 'bar';
    }

    return {
      ...prevQuery,
      title: prompt,
      dimensions: Array.from(dims),
      measures: measures.length > 0 ? measures : prevQuery.measures,
      timeDimensions: timeDims,
      filters,
      suggestedChart,
    };
  };

  const submitPrompt = useCallback(
    async (promptText: string) => {
      if (!promptText.trim()) return;
      setError(null);
      setIsLoading(true);

      const userMsg: ChatMessage = {
        id: `msg-${Date.now()}`,
        role: 'user',
        content: promptText,
        timestamp: new Date().toISOString(),
      };

      const updatedHistory = [...messages, userMsg];
      setMessages(updatedHistory);
    onMessage?.(userMsg);

      try {
        const payload = {
          messages: updatedHistory.map((m) => ({ role: m.role, content: m.content })),
          currentQuery,
          catalog,
          userAccountId: userContext?.accountId,
          userTenantId: userContext?.tenantId,
        };

        const res = await apiFetch('/api/v1/ai/query-completion', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload),
        });

        let nextQuery: ExplorerQueryDefinition;
        let assistantContent = '';
        let followUps: string[] = [];
        let data: any = {};

        if (res.ok) {
          data = await res.json();
          nextQuery = data.generatedQuery;
          assistantContent = data.assistantMessage;
          followUps = data.suggestedFollowUps || [];
          setAmbiguities(data.ambiguityQuestions || []);
        } else {
          // Fallback to local heuristic synthesis
          nextQuery = localSynthesize(promptText, currentQuery);
          assistantContent = `Synthesized query for "${promptText}".`;
          followUps = [
            `Compare across ${catalog.find((c) => c.category === 'dimension' && !nextQuery.dimensions.includes(c.id))?.label || 'Account Type'}`,
            'Export as PDF Report',
            'Filter Top 10 by Valuation',
          ];
        }

        // Add assistant turn
        const assistantMsg: ChatMessage = {
          id: `msg-${Date.now() + 1}`,
          role: 'assistant',
          content: assistantContent,
          timestamp: new Date().toISOString(),
          querySnapshot: nextQuery,
          mutationIntent: (data.mutation_intent as any) || 'unknown',
        };
        setMessages((prev) => [...prev, assistantMsg]);
        onMessage?.(assistantMsg);

        // Update UI state
        setIsCacheHit(Boolean(data.isCacheHit));
        onQueryChange(nextQuery);
        if (nextQuery.suggestedChart) {
          onViewModeChange(nextQuery.suggestedChart);
        }
        setSuggestedFollowUps(followUps);

        // Generate synthetic executive insight badge
        const primaryDim = catalog.find((c) => c.id === nextQuery.dimensions[0]);
        const primaryMeas = catalog.find((c) => c.id === nextQuery.measures[0]?.fieldId);
        setInsight({
          summaryText: data.insightSummary || `Query synthesized for ${primaryMeas?.label || 'Valuation'} aggregated across ${primaryDim?.label || 'all entities'}. Dataset reflects current bitemporal isolation constraints.`,
          topDriver: data.topDriver || primaryDim?.label || undefined,
          anomalies: data.anomalies || [],
        });

        // Trigger live query run
        onExecuteQuery(nextQuery);
      } catch (err: any) {
        // Safe fallback
        setIsCacheHit(false);
        const nextQuery = localSynthesize(promptText, currentQuery);
        onQueryChange(nextQuery);
        if (nextQuery.suggestedChart) {
          onViewModeChange(nextQuery.suggestedChart);
        }
        onExecuteQuery(nextQuery);
      } finally {
        setIsLoading(false);
      }
    },
    [messages, currentQuery, catalog, userContext, onMessage, onQueryChange, onViewModeChange, onExecuteQuery]
  );

  const clearConversation = useCallback(() => {
    setMessages([]);
    setError(null);
    setInsight(null);
    setIsCacheHit(false);
    setAmbiguities([]);
  }, []);

  return {
    messages,
    isLoading,
    error,
    insight,
    isCacheHit,
    suggestedFollowUps,
    ambiguities,
    submitPrompt,
    clearConversation,
  };
}

export default useExplorerAI;
