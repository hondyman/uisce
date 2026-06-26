/**
 * RulesCatalog.tsx
 *
 * Rules Catalog page - Browse, search, filter, and add rules to the rules builder
 * Similar to Calculations Catalog but optimized for validation rules
 *
 * Features:
 * - Search by rule name, description
 * - Filter by category (ESG, Private Capital, Mutual Funds, Funds Accounting, Risk, Compliance, etc.)
 * - Filter by severity (BLOCK, WARNING, INFO)
 * - Filter by frequency (ON_TRADE, DAILY, MONTHLY, etc.)
 * - View rule details and parameters
 * - Add rules to active rules builder
 * - Compare multiple rules side-by-side
 * - Sort by evaluation order, rule type, severity
 * - Save favorite rules for quick access
 */

import React, { useMemo, useCallback } from 'react';
import WEALTH_VALIDATION_RULES from '@/data/wealthValidationRules';
import { devLog } from '../../utils/devLogger';
import styles from './RulesCatalog.module.css';

// Import our new modular components and utilities
import { RuleCard } from './RuleCard';
import { RuleListItem } from './RuleListItem';
import { FilterSidebar } from './FilterSidebar';
import { ControlsBar } from './ControlsBar';
import { CompareView } from './CompareView';
import { useFilters, useSelectedRules, useSavedRules, useViewMode } from './ruleCatalogHooks';
import { filterAndSortRules, getUniqueValues } from './ruleCatalogFilters';

const RulesCatalog: React.FC = () => {
  // Use our custom hooks for state management
  const {
    filters,
    updateSearch,
    toggleCategory,
    toggleSeverity,
    toggleFrequency,
    toggleRuleType,
    setSortBy,
    clearFilters
  } = useFilters();

  const { selectedRules, toggleRuleSelection } = useSelectedRules();
  const { savedRules, toggleSaved } = useSavedRules();
  const { viewMode, setViewMode } = useViewMode();

  // Get unique values for filters
  const uniqueValues = useMemo(() => getUniqueValues(), []);

  // Filter and sort rules using our utility function
  const filteredRules = useMemo(() => filterAndSortRules(filters), [filters]);

  // Get selected items for comparison
  const selectedItems = useMemo(() =>
    filteredRules.filter(item => selectedRules.includes(item.rule.id)),
    [filteredRules, selectedRules]
  );

  const handleAddSelectedToBuilder = useCallback(() => {
    // Emit event or callback to parent to add selected rules to builder
    devLog('Adding selected rules to builder:', selectedRules);
    // TODO: Implement callback to parent component
  }, [selectedRules]);

  return (
    <div className={styles.container}>
      {/* Header */}
      <div className={styles.header}>
        <h1>Rules Catalog</h1>
        <p>Browse, search, and add validation rules to your rules builder</p>
      </div>

      {/* Controls Bar */}
      <ControlsBar
        search={filters.search}
        viewMode={viewMode}
        sortBy={filters.sortBy}
        onSearchChange={updateSearch}
        onViewModeChange={setViewMode}
        onSortChange={setSortBy}
        disableCompare={selectedRules.length < 2}
      />

      <div className={styles.mainContent}>
        {/* Sidebar Filters */}
        <FilterSidebar
          filters={{
            categories: filters.categories,
            severities: filters.severities,
            frequencies: filters.frequencies,
            ruleTypes: filters.ruleTypes
          }}
          uniqueValues={uniqueValues}
          onToggleCategory={toggleCategory}
          onToggleSeverity={toggleSeverity}
          onToggleFrequency={toggleFrequency}
          onToggleRuleType={toggleRuleType}
          onClearFilters={clearFilters}
        />

        {/* Main Content */}
        <main className={styles.content}>
          {/* Results Summary */}
          <div className={styles.resultsSummary}>
            <span>
              Showing <strong>{filteredRules.length}</strong> of <strong>{WEALTH_VALIDATION_RULES.length}</strong> rules
              {selectedRules.length > 0 && (
                <span>
                  {' '} • <strong>{selectedRules.length}</strong> selected
                </span>
              )}
            </span>
            {selectedRules.length > 0 && (
              <button
                className={styles.addButton}
                onClick={handleAddSelectedToBuilder}
              >
                Add {selectedRules.length} to Builder →
              </button>
            )}
          </div>

          {/* Grid View */}
          {viewMode === 'grid' && (
            <div className={styles.grid}>
              {filteredRules.map(item => (
                <RuleCard
                  key={item.rule.id}
                  item={item}
                  isSelected={selectedRules.includes(item.rule.id)}
                  isSaved={savedRules.includes(item.rule.id)}
                  onSelect={toggleRuleSelection}
                  onToggleSaved={toggleSaved}
                />
              ))}
            </div>
          )}

          {/* List View */}
          {viewMode === 'list' && (
            <div className={styles.list}>
              {filteredRules.map(item => (
                <RuleListItem
                  key={item.rule.id}
                  item={item}
                  isSelected={selectedRules.includes(item.rule.id)}
                  isSaved={savedRules.includes(item.rule.id)}
                  onSelect={toggleRuleSelection}
                  onToggleSaved={toggleSaved}
                />
              ))}
            </div>
          )}

          {/* Compare View */}
          {viewMode === 'compare' && (
            <CompareView selectedItems={selectedItems} />
          )}

          {filteredRules.length === 0 && (
            <div className={styles.emptyState}>
              <p>No rules found matching your criteria</p>
              <button onClick={clearFilters}>
                Clear Filters
              </button>
            </div>
          )}
        </main>
      </div>
    </div>
  );
};

export default RulesCatalog;
