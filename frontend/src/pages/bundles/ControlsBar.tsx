/**
 * Controls Bar Component
 *
 * Top controls for search, view mode, and sorting
 */

import type React from 'react';
import { ViewMode, FilterOptions } from './ruleCatalogConstants';
import styles from './RulesCatalog.module.css';

interface ControlsBarProps {
  search: string;
  viewMode: ViewMode;
  sortBy: FilterOptions['sortBy'];
  onSearchChange: (search: string) => void;
  onViewModeChange: (mode: ViewMode) => void;
  onSortChange: (sortBy: FilterOptions['sortBy']) => void;
  disableCompare?: boolean;
}

export const ControlsBar: React.FC<ControlsBarProps> = ({
  search,
  viewMode,
  sortBy,
  onSearchChange,
  onViewModeChange,
  onSortChange,
  disableCompare = false
}) => {
  return (
    <div className={styles.controlsBar}>
      <div className={styles.searchBox}>
        <label htmlFor="search-input" className={styles.searchLabel}>Search rules</label>
        <input
          id="search-input"
          type="text"
          placeholder="Search rules by name, description, or category..."
          value={search}
          onChange={(e) => onSearchChange(e.target.value)}
          className={styles.searchInput}
        />
        <span className={styles.searchIcon}>🔍</span>
      </div>

      <div className={styles.viewButtons}>
        <button
          className={`${styles.viewButton} ${viewMode === 'grid' ? styles.active : ''}`}
          onClick={() => onViewModeChange('grid')}
          title="Grid view"
        >
          ⊞
        </button>
        <button
          className={`${styles.viewButton} ${viewMode === 'list' ? styles.active : ''}`}
          onClick={() => onViewModeChange('list')}
          title="List view"
        >
          ☰
        </button>
        <button
          className={`${styles.viewButton} ${viewMode === 'compare' ? styles.active : ''}`}
          onClick={() => onViewModeChange('compare')}
          title="Compare view"
          disabled={disableCompare}
        >
          ⇄
        </button>
      </div>

      <div className={styles.sortDropdown}>
        <label htmlFor="sort-select">Sort by:</label>
        <select
          id="sort-select"
          value={sortBy}
          onChange={(e) => onSortChange(e.target.value as FilterOptions['sortBy'])}
        >
          <option value="evaluationOrder">Evaluation Order</option>
          <option value="name">Name (A-Z)</option>
          <option value="severity">Severity</option>
        </select>
      </div>
    </div>
  );
};