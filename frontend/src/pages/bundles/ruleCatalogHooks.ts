/**
 * Rules Catalog Custom Hooks
 *
 * Custom hooks for managing Rules Catalog state
 */

import { useState, useCallback } from 'react';
import { FilterOptions, ViewMode } from './ruleCatalogConstants';

export const useFilters = (initialFilters?: Partial<FilterOptions>) => {
  const [filters, setFilters] = useState<FilterOptions>({
    search: '',
    categories: [],
    severities: [],
    frequencies: [],
    ruleTypes: [],
    sortBy: 'evaluationOrder',
    ...initialFilters
  });

  const updateSearch = useCallback((search: string) => {
    setFilters(prev => ({ ...prev, search }));
  }, []);

  const toggleCategory = useCallback((categoryId: string) => {
    setFilters(prev => ({
      ...prev,
      categories: prev.categories.includes(categoryId)
        ? prev.categories.filter(c => c !== categoryId)
        : [...prev.categories, categoryId]
    }));
  }, []);

  const toggleSeverity = useCallback((severity: string) => {
    setFilters(prev => ({
      ...prev,
      severities: prev.severities.includes(severity)
        ? prev.severities.filter(s => s !== severity)
        : [...prev.severities, severity]
    }));
  }, []);

  const toggleFrequency = useCallback((frequency: string) => {
    setFilters(prev => ({
      ...prev,
      frequencies: prev.frequencies.includes(frequency)
        ? prev.frequencies.filter(f => f !== frequency)
        : [...prev.frequencies, frequency]
    }));
  }, []);

  const toggleRuleType = useCallback((ruleType: string) => {
    setFilters(prev => ({
      ...prev,
      ruleTypes: prev.ruleTypes.includes(ruleType)
        ? prev.ruleTypes.filter(t => t !== ruleType)
        : [...prev.ruleTypes, ruleType]
    }));
  }, []);

  const setSortBy = useCallback((sortBy: FilterOptions['sortBy']) => {
    setFilters(prev => ({ ...prev, sortBy }));
  }, []);

  const clearFilters = useCallback(() => {
    setFilters({
      search: '',
      categories: [],
      severities: [],
      frequencies: [],
      ruleTypes: [],
      sortBy: 'evaluationOrder'
    });
  }, []);

  return {
    filters,
    updateSearch,
    toggleCategory,
    toggleSeverity,
    toggleFrequency,
    toggleRuleType,
    setSortBy,
    clearFilters,
    setFilters
  };
};

export const useSelectedRules = () => {
  const [selectedRules, setSelectedRules] = useState<string[]>([]);

  const toggleRuleSelection = useCallback((ruleId: string) => {
    setSelectedRules(prev =>
      prev.includes(ruleId)
        ? prev.filter(r => r !== ruleId)
        : [...prev, ruleId]
    );
  }, []);

  const clearSelection = useCallback(() => {
    setSelectedRules([]);
  }, []);

  const selectAll = useCallback((ruleIds: string[]) => {
    setSelectedRules(ruleIds);
  }, []);

  return {
    selectedRules,
    toggleRuleSelection,
    clearSelection,
    selectAll
  };
};

export const useSavedRules = () => {
  const [savedRules, setSavedRules] = useState<string[]>([]);

  const toggleSaved = useCallback((ruleId: string) => {
    setSavedRules(prev =>
      prev.includes(ruleId)
        ? prev.filter(r => r !== ruleId)
        : [...prev, ruleId]
    );
  }, []);

  return {
    savedRules,
    toggleSaved
  };
};

export const useViewMode = (initialMode: ViewMode = 'grid') => {
  const [viewMode, setViewMode] = useState<ViewMode>(initialMode);

  return {
    viewMode,
    setViewMode
  };
};