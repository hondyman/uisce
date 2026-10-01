import React, { createContext, useContext, useEffect, useState } from 'react';
import type { RuntimeFilter } from '../services/savedQueryApi';

export interface ActiveCrossFilter {
  termNodeId: string;
  value: unknown;
  sourceWidgetId: string;
  operator?: 'eq' | 'in';
}

export interface CrossFilterConfig {
  enabled?: boolean;
  mode?: 'emit' | 'receive' | 'both';
}

export type CrossFilterScope = 'page' | 'tab';

export class CrossFilterBus {
  private filters: Map<string, ActiveCrossFilter> = new Map();
  private listeners: Set<() => void> = new Set();

  constructor(initialFilters?: ActiveCrossFilter[]) {
    if (initialFilters) {
      initialFilters.forEach((f) => this.filters.set(f.termNodeId, f));
    }
  }

  public subscribe(listener: () => void): () => void {
    this.listeners.add(listener);
    return () => {
      this.listeners.delete(listener);
    };
  }

  private notify(): void {
    this.listeners.forEach((l) => l());
  }

  /**
   * Emits a filter on a semantic term.
   * If value is an array, uses operator 'in'. If array is empty, removes filter.
   * If the same scalar value is already set by this widget on this term, toggles it off.
   */
  public emit(
    sourceWidgetId: string,
    termNodeId: string,
    value: unknown,
    operator?: 'eq' | 'in'
  ): void {
    if (value === undefined || value === null) {
      this.filters.delete(termNodeId);
      this.notify();
      return;
    }

    if (Array.isArray(value)) {
      if (value.length === 0) {
        this.filters.delete(termNodeId);
      } else {
        this.filters.set(termNodeId, {
          termNodeId,
          value,
          sourceWidgetId,
          operator: operator || 'in',
        });
      }
      this.notify();
      return;
    }

    const existing = this.filters.get(termNodeId);
    if (existing && existing.sourceWidgetId === sourceWidgetId && existing.value === value) {
      this.filters.delete(termNodeId);
    } else {
      this.filters.set(termNodeId, {
        termNodeId,
        value,
        sourceWidgetId,
        ...(operator ? { operator } : {}),
      });
    }
    this.notify();
  }

  public remove(termNodeId: string): void {
    if (this.filters.delete(termNodeId)) {
      this.notify();
    }
  }

  public clear(): void {
    if (this.filters.size > 0) {
      this.filters.clear();
      this.notify();
    }
  }

  public getAll(): ActiveCrossFilter[] {
    return Array.from(this.filters.values());
  }

  public get count(): number {
    return this.filters.size;
  }

  /**
   * Computes runtime filters for a target widget:
   * 1. If widget disabled or mode == 'emit', returns []
   * 2. Self-exclusion: filters emitted by widgetId are excluded
   * 3. Term matching: only filters whose termNodeId is in queryTerms are included
   */
  public getFiltersForWidget(
    widgetId: string,
    queryTerms: string[],
    config?: CrossFilterConfig
  ): RuntimeFilter[] {
    if (config && config.enabled === false) return [];
    if (config?.mode === 'emit') return [];

    const termSet = new Set(queryTerms);
    const applicable: RuntimeFilter[] = [];

    for (const filter of this.filters.values()) {
      if (filter.sourceWidgetId === widgetId) {
        continue;
      }
      if (termSet.has(filter.termNodeId)) {
        applicable.push({
          termNodeId: filter.termNodeId,
          operator: filter.operator || (Array.isArray(filter.value) ? 'in' : 'eq'),
          value: filter.value,
        });
      }
    }

    return applicable;
  }
}

const defaultGlobalBus = new CrossFilterBus();

const CrossFilterContext = createContext<CrossFilterBus>(defaultGlobalBus);

export const CrossFilterProvider: React.FC<{
  bus?: CrossFilterBus;
  children: React.ReactNode;
}> = ({ bus: propBus, children }) => {
  const [bus] = useState(() => propBus || new CrossFilterBus());
  return React.createElement(CrossFilterContext.Provider, { value: bus }, children);
};

export function useCrossFilterBus(): CrossFilterBus {
  return useContext(CrossFilterContext);
}

export function useActiveCrossFilters(bus?: CrossFilterBus): ActiveCrossFilter[] {
  const contextBus = useCrossFilterBus();
  const activeBus = bus || contextBus;
  const [filters, setFilters] = useState<ActiveCrossFilter[]>(() => activeBus.getAll());

  useEffect(() => {
    return activeBus.subscribe(() => {
      setFilters(activeBus.getAll());
    });
  }, [activeBus]);

  return filters;
}
