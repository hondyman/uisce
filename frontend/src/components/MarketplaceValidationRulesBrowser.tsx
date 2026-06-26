/**
 * Marketplace Validation Rules Browser & Importer Component
 *
 * Allows users to browse, filter, and import validation rules from the marketplace
 */

import React, { useState, useEffect, useMemo } from 'react';
import {
  MARKETPLACE_VALIDATION_RULES,
} from '../data/marketplaceValidationRules';

interface MarketplaceValidationRule {
  id: string;
  name: string;
  description: string;
  category?: string;
  rule_type: string;
  scope: string | string[];
  severity: string;
  frequency: string;
  evaluationOrder: number;
  parameters: Record<string, any>;
  isActive?: boolean;
  effectiveFrom?: string;
  tags?: string[];
  downloadsCount?: number;
  usageCount?: number;
  rating?: number;
}
import {
  importMarketplaceValidationRules,
  getMarketplaceRulesSummary,
} from '../utils/marketplaceImportHandler';
import { useConfirm } from './ConfirmProvider';
import { useNotification } from '../hooks/useNotification';
import styles from './MarketplaceValidationRulesBrowser.module.css';

export interface MarketplaceRulesBrowserProps {
  tenantId: string;
  datasourceId: string;
  onImportComplete?: (result: any) => void;
  onClose?: () => void;
}

export const MarketplaceValidationRulesBrowser: React.FC<
  MarketplaceRulesBrowserProps
> = ({ tenantId, datasourceId, onImportComplete, onClose }) => {
  const [selectedRuleIds, setSelectedRuleIds] = useState<Set<string>>(new Set());
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedCategory, setSelectedCategory] = useState<string | null>(null);
  const [viewMode, setViewMode] = useState<'browse' | 'summary'>('browse');
  const [importing, setImporting] = useState(false);
  const [importMessage, setImportMessage] = useState<string | null>(null);
  const [marketplaceRules, setMarketplaceRules] = useState<MarketplaceValidationRule[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const confirm = useConfirm();
  const notification = useNotification();

  // Fetch marketplace validation rules from API
  useEffect(() => {
    const fetchMarketplaceRules = async () => {
      try {
        setLoading(true);
        setError(null);

  // Fetch validation rules (public endpoint)
        const response = await fetch(
          `/api/marketplace/validation-rules`,
          {
            headers: {
              // Marketplace rules are public, no tenant scope required
            },
          }
        );

        if (!response.ok) {
          throw new Error(`Failed to fetch validation rules: ${response.status}`);
        }

        const data = await response.json();
  // Data received; populate rules
        setMarketplaceRules(data.rules || []);
      } catch (err) {
        console.error('[MarketplaceValidationRulesBrowser] Failed to fetch marketplace validation rules:', err);
        setError(err instanceof Error ? err.message : 'Failed to load marketplace validation rules');
        // Fallback to static data
        setMarketplaceRules(MARKETPLACE_VALIDATION_RULES);
      } finally {
        setLoading(false);
      }
    };

    fetchMarketplaceRules();
  }, [tenantId, datasourceId]);

  const categories = useMemo(() => {
    const uniqueCategories = new Set(marketplaceRules.map(rule => rule.category).filter(Boolean));
    return Array.from(uniqueCategories).sort() as string[];
  }, [marketplaceRules]);
  const summary = useMemo(() => {
    const bySeverity = marketplaceRules.reduce((acc, rule) => {
      const severity = rule.severity.toLowerCase();
      acc[severity] = (acc[severity] || 0) + 1;
      return acc;
    }, {} as Record<string, number>);

    const byCategory = categories.map(category => {
      const categoryRules = marketplaceRules.filter(rule => rule.category === category);
      const byRisk = categoryRules.reduce((acc, rule) => {
        const risk = rule.severity.toLowerCase();
        acc[risk] = (acc[risk] || 0) + 1;
        return acc;
      }, {} as Record<string, number>);
      return {
        category,
        count: categoryRules.length,
        byRisk
      };
    });

    return {
      totalRules: marketplaceRules.length,
      totalCategories: categories.length,
      bySeverity,
      byCategory
    };
  }, [marketplaceRules, categories]);

  // Filter rules based on search and category
  const filteredRules = useMemo(() => {
    let rules = marketplaceRules;

    if (searchQuery) {
      rules = rules.filter((r: MarketplaceValidationRule) =>
        r.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
        r.description.toLowerCase().includes(searchQuery.toLowerCase()) ||
        (r.category && r.category.toLowerCase().includes(searchQuery.toLowerCase()))
      );
    }

    if (selectedCategory) {
      rules = rules.filter((r: MarketplaceValidationRule) => r.category === selectedCategory);
    }

    return rules;
  }, [marketplaceRules, searchQuery, selectedCategory]);

  // Handle individual rule selection
  const toggleRuleSelection = (ruleId: string) => {
    const newSelected = new Set(selectedRuleIds);
    if (newSelected.has(ruleId)) {
      newSelected.delete(ruleId);
    } else {
      newSelected.add(ruleId);
    }
    setSelectedRuleIds(newSelected);
  };

  // Handle select/deselect all in current view
  const toggleAllInView = () => {
    const currentRuleIds = new Set(
      filteredRules.map((r) => r.id)
    );

    const newSelected = new Set(selectedRuleIds);
    let allSelected = true;

    for (const id of currentRuleIds) {
      if (!newSelected.has(id)) {
        allSelected = false;
        break;
      }
    }

    if (allSelected) {
      // Deselect all in view
      for (const id of currentRuleIds) {
        newSelected.delete(id);
      }
    } else {
      // Select all in view
      for (const id of currentRuleIds) {
        newSelected.add(id);
      }
    }

    setSelectedRuleIds(newSelected);
  };

  // Handle import
  const handleImport = async () => {
    if (selectedRuleIds.size === 0) {
      notification.error('Please select at least one rule to import');
      return;
    }

    if (!(await confirm({ title: 'Import rules', description: `Import ${selectedRuleIds.size} validation rule(s) into the current tenant/datasource?` }))) {
      return;
    }

    setImporting(true);
    setImportMessage('Importing rules...');

    try {
      const result = await importMarketplaceValidationRules(
        tenantId,
        datasourceId,
        Array.from(selectedRuleIds)
      );

      let message = `✓ Created: ${result.created}\n`;
      if (result.updated > 0) message += `✓ Updated: ${result.updated}\n`;
      if (result.skipped > 0) message += `⚠ Skipped: ${result.skipped}\n`;

      if (result.failed.length > 0) {
        message += `\n❌ Failed rules:\n`;
        result.failed.forEach((f) => {
          message += `  - ${f.ruleName}: ${f.error}\n`;
        });
      }

      setImportMessage(message);
      notification.success('Marketplace rules imported successfully');

      if (onImportComplete) {
        onImportComplete(result);
      }

      // Clear selection after successful import
      if (result.failed.length === 0) {
        setSelectedRuleIds(new Set());
        setTimeout(() => {
          if (onClose) onClose();
        }, 2000);
      }
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : String(error);
      setImportMessage(`❌ Import failed: ${errorMsg}`);
    } finally {
      setImporting(false);
    }
  };

  // Render browse mode with left-side facets
  const renderBrowseMode = () => (
    <div className={styles.browseContainer}>
      <div className={styles.sidebar}>
        {/* Search Bar */}
        <div className={styles.searchBar}>
          <input
            type="text"
            placeholder="Search rules by name, description, or tags..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className={styles.searchInput}
          />
        </div>

        {/* Category Filter (facets) */}
        <div className={styles.categoryFilter}>
          <button
            className={`${styles.categoryBtn} ${
              selectedCategory === null ? styles.active : ''
            }`}
            onClick={() => setSelectedCategory(null)}
          >
            All ({marketplaceRules.length})
          </button>
          {categories.map((cat: string) => {
            const count = marketplaceRules.filter((r: MarketplaceValidationRule) => r.category === cat).length;
            return (
              <button
                key={cat}
                className={`${styles.categoryBtn} ${
                  selectedCategory === cat ? styles.active : ''
                }`}
                onClick={() => setSelectedCategory(cat)}
              >
                {cat} ({count})
              </button>
            );
          })}
        </div>

        {/* Select All */}
        <div className={styles.selectAllContainer}>
          <label className={styles.selectAllLabel}>
            <input
              type="checkbox"
              checked={selectedRuleIds.size > 0 && filteredRules.every((r) => selectedRuleIds.has(r.id))}
              onChange={toggleAllInView}
            />
            Select all in view ({filteredRules.length})
          </label>
        </div>
      </div>

      {/* Main content: rules list */}
      <div className={styles.mainContent}>
        <div className={styles.rulesList}>
          {filteredRules.length === 0 ? (
            <div className={styles.noRules}>No rules found</div>
          ) : (
            filteredRules.map((rule) => (
              <div
                key={rule.id}
                className={`${styles.ruleCard} ${
                  selectedRuleIds.has(rule.id) ? styles.selected : ''
                }`}
                onClick={() => toggleRuleSelection(rule.id)}
              >
                <div className={styles.ruleHeader}>
                  <input
                    type="checkbox"
                    checked={selectedRuleIds.has(rule.id)}
                    onChange={(e) => {
                      e.stopPropagation();
                      toggleRuleSelection(rule.id);
                    }}
                    title="Select this rule for import"
                    aria-label={`Select ${rule.name}`}
                  />
                  <h3 className={styles.ruleName}>{rule.name}</h3>
                  <span className={`${styles.severity} ${styles[`severity-${rule.severity.toLowerCase()}`]}`}>
                    {rule.severity}
                  </span>
                </div>

                <p className={styles.ruleDescription}>{rule.description}</p>

                <div className={styles.ruleMetadata}>
                  <div className={styles.metaItem}>
                    <span className={styles.label}>Type:</span>
                    <span className={styles.value}>{rule.rule_type}</span>
                  </div>
                  <div className={styles.metaItem}>
                    <span className={styles.label}>Frequency:</span>
                    <span className={styles.value}>{rule.frequency}</span>
                  </div>
                  <div className={styles.metaItem}>
                    <span className={styles.label}>Category:</span>
                    <span className={styles.value}>{rule.category}</span>
                  </div>
                </div>

                {rule.tags && rule.tags.length > 0 && (
                  <div className={styles.tags}>
                    {rule.tags.map((tag: string) => (
                      <span key={tag} className={styles.tag}>
                        {tag}
                      </span>
                    ))}
                  </div>
                )}
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  );

  // Render summary mode
  const renderSummaryMode = () => (
    <div className={styles.summaryContainer}>
      <div className={styles.summaryStats}>
        <div className={styles.statCard}>
          <div className={styles.statNumber}>{summary.totalRules}</div>
          <div className={styles.statLabel}>Total Rules</div>
        </div>
        <div className={styles.statCard}>
          <div className={styles.statNumber}>{summary.totalCategories}</div>
          <div className={styles.statLabel}>Categories</div>
        </div>
        <div className={`${styles.statCard} ${styles.block}`}>
          <div className={styles.statNumber}>{summary.bySeverity.block}</div>
          <div className={styles.statLabel}>BLOCK Rules</div>
        </div>
        <div className={`${styles.statCard} ${styles.warning}`}>
          <div className={styles.statNumber}>{summary.bySeverity.warning}</div>
          <div className={styles.statLabel}>WARNING Rules</div>
        </div>
        <div className={`${styles.statCard} ${styles.info}`}>
          <div className={styles.statNumber}>{summary.bySeverity.info}</div>
          <div className={styles.statLabel}>INFO Rules</div>
        </div>
      </div>

      <div className={styles.categorySummary}>
        <h3>Rules by Category</h3>
        {summary.byCategory.map((cat) => (
          <div key={cat.category} className={styles.categorySummaryItem}>
            <div className={styles.categoryName}>{cat.category}</div>
            <div className={styles.categoryBreakdown}>
              <span className={`${styles.risk} ${styles.block}`}>
                BLOCK: {cat.byRisk.block}
              </span>
              <span className={`${styles.risk} ${styles.warning}`}>
                WARNING: {cat.byRisk.warning}
              </span>
              <span className={`${styles.risk} ${styles.info}`}>
                INFO: {cat.byRisk.info}
              </span>
              <span className={styles.total}>Total: {cat.count}</span>
            </div>
          </div>
        ))}
      </div>
    </div>
  );

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <h2>📦 Marketplace Validation Rules</h2>
        <div className={styles.viewToggle}>
          <button
            className={`${styles.toggleBtn} ${viewMode === 'browse' ? styles.active : ''}`}
            onClick={() => setViewMode('browse')}
          >
            Browse
          </button>
          <button
            className={`${styles.toggleBtn} ${viewMode === 'summary' ? styles.active : ''}`}
            onClick={() => setViewMode('summary')}
          >
            Summary
          </button>
        </div>
      </div>

      {/* Loading State */}
      {loading && (
        <div className={styles.loadingState}>
          <div className={styles.spinner}></div>
          <p>Loading marketplace validation rules...</p>
        </div>
      )}

      {/* Error State */}
      {error && (
        <div className={styles.errorState}>
          <p>❌ {error}</p>
          <button
            onClick={() => window.location.reload()}
            className={styles.retryBtn}
          >
            Retry
          </button>
        </div>
      )}

      {/* Content */}
      {!loading && !error && (
        <>
          {viewMode === 'browse' ? renderBrowseMode() : renderSummaryMode()}
        </>
      )}

      {/* Import Section */}
      <div className={styles.footer}>
        {importMessage && (
          <div className={`${styles.message} ${styles[importing ? 'loading' : 'done']}`}>
            <pre>{importMessage}</pre>
          </div>
        )}

        <div className={styles.actions}>
          <button
            className={`${styles.btn} ${styles.secondary}`}
            onClick={onClose}
            disabled={importing}
          >
            Close
          </button>
          {selectedRuleIds.size > 0 && (
            <button
              className={`${styles.btn} ${styles.primary}`}
              onClick={handleImport}
              disabled={importing}
            >
              {importing ? 'Importing...' : `Import ${selectedRuleIds.size} Rule(s)`}
            </button>
          )}
        </div>
      </div>
    </div>
  );
};

export default MarketplaceValidationRulesBrowser;
