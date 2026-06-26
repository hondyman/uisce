/**
 * Marketplace.tsx
 *
 * Complete marketplace for browsing and adding rules and calculations
 * to a tenant's platform. Items are persisted in PostgreSQL.
 *
 * Features:
 * - Browse marketplace catalog (rules & calculations)
 * - Search and filter by category, severity, status
 * - View detailed item information
 * - Add items to tenant (persisted to DB)
 * - Manage added items
 * - Rate and provide feedback
 * - Track usage analytics
 */

// Debug: marketplace module load log removed

import React, { useState, useEffect, useMemo, useCallback } from 'react';
import { useTenant } from '../../contexts/TenantContext';
import { useConfirm } from '../../components/ConfirmProvider';
import { useNotification } from '../../hooks/useNotification';
import styles from './Marketplace.module.css';
import { MarketplaceValidationRulesBrowser } from '../../components/MarketplaceValidationRulesBrowser';

export interface MarketplaceItem {
  id: string;
  name: string;
  description: string;
  item_type: 'rule' | 'calculation';
  version: string;
  category: string;
  subcategories: string[];
  severity?: 'BLOCK' | 'WARNING' | 'INFO';
  icon_emoji: string;
  color_hex: string;
  summary: string;
  long_description: string;
  implementation_json: Record<string, any>;
  scope: string;
  rule_type: string;
  frequency: string;
  evaluation_order: number;
  is_public: boolean;
  is_official: boolean;
  is_core: boolean;
  status: 'active' | 'beta' | 'deprecated' | 'archived';
  external_api_providers: string[];
  requires_credentials: boolean;
  usage_count: number;
  rating?: number;
  downloads_count: number;
  created_at: string;
  updated_at: string;
  published_at?: string;
}

export interface TenantMarketplaceItem {
  id: string;
  tenant_id: string;
  marketplace_item_id: string;
  custom_name?: string;
  custom_parameters: Record<string, any>;
  enabled_for_tenant: boolean;
  added_at: string;
  last_used_at?: string;
  usage_count: number;
  marketplace_version_at_time_of_add: string;
  local_version: string;
  has_local_modifications: boolean;
  tenant_rating?: number;
  tenant_feedback?: string;
}

interface TabType {
  value: 'browse' | 'validation-rules' | 'my-items' | 'analytics';
  label: string;
}

const Marketplace: React.FC = () => {
  // Removed verbose render-time debug logs; keep runtime error logging only
  try {
    const { tenant, datasource } = useTenant();
    const tenantId = tenant?.id?.trim() ?? '';
    const datasourceId = (datasource?.id ?? datasource?.alpha_tenant_instance_id ?? '').trim();

  // State
  const [activeTab, setActiveTab] = useState<'browse' | 'validation-rules' | 'my-items' | 'analytics'>('validation-rules');
  const [marketplaceItems, setMarketplaceItems] = useState<MarketplaceItem[]>([]);
  const [tenantItems, setTenantItems] = useState<TenantMarketplaceItem[]>([]);
  const [selectedItem, setSelectedItem] = useState<MarketplaceItem | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Filter state
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedItemType, setSelectedItemType] = useState<'rule' | 'calculation' | ''>('');
  const [selectedCategories, setSelectedCategories] = useState<string[]>([]);
  const [selectedSeverities, setSelectedSeverities] = useState<string[]>([]);
  const [showOnlyOfficial, setShowOnlyOfficial] = useState(false);
  const [showOnlyCore, setShowOnlyCore] = useState(false);
  const [sortBy, setSortBy] = useState<'relevance' | 'popular' | 'rating' | 'newest'>('relevance');

  // View mode
  const [viewMode, setViewMode] = useState<'grid' | 'list'>('grid');

  // ========================================================================
  // Fetch Functions
  // ========================================================================

  const fetchMarketplaceItems = useCallback(async () => {
    // Marketplace items are public — tenant/datasource are not required for this call.
    // Previously there was a guard that prevented fetching when tenant/datasource
    // were not set which blocked public marketplace access. Remove that guard
    // so the public catalog can be fetched regardless of tenant scope.
    setLoading(true);
    setError(null);

    try {
      const params = new URLSearchParams({
        search: searchTerm,
        item_type: selectedItemType,
        sort_by: sortBy,
      });

      selectedCategories.forEach(cat => params.append('category', cat));
      selectedSeverities.forEach(sev => params.append('severity', sev));

      if (showOnlyOfficial) params.append('only_official', 'true');
      if (showOnlyCore) params.append('only_core', 'true');

      const response = await fetch(`/api/marketplace/items?${params.toString()}`, {
        headers: {
          // Marketplace items are public, no tenant scope required
        },
      });

      // If the backend returns a non-200, surface a helpful error
      if (!response.ok) {
        const serverText = await response.text().catch(() => '');
        throw new Error(`Failed to fetch marketplace items: ${response.status} ${response.statusText} ${serverText}`);
      }

      // Defensive parsing: some server errors return plain text (SQL/scan errors), not JSON.
      let data: any = null;
      try {
        data = await response.json();
      } catch (parseErr) {
        const raw = await response.text().catch(() => '<unreadable response>');
        console.error('[Marketplace] Non-JSON response from /api/marketplace/items:', raw);
        throw new Error(`Invalid JSON response from server: ${raw}`);
      }

      setMarketplaceItems((data && data.items) || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      console.error('Error fetching marketplace items:', err);
    } finally {
      setLoading(false);
    }
  }, [tenantId, datasourceId, searchTerm, selectedItemType, selectedCategories, selectedSeverities, showOnlyOfficial, showOnlyCore, sortBy]);

  const fetchTenantItems = useCallback(async () => {
    if (!tenantId || !datasourceId) return;

    try {
      const response = await fetch(
        `/api/marketplace/tenant-items?tenant_id=${tenantId}&tenant_instance_id=${datasourceId}`,
        {
          headers: {
            'X-Tenant-ID': tenantId,
            'X-Tenant-Datasource-ID': datasourceId,
          },
        }
      );

      if (!response.ok) throw new Error('Failed to fetch tenant items');

      const data = await response.json();
      setTenantItems(data || []);
    } catch (err) {
      console.error('Error fetching tenant items:', err);
    }
  }, [tenantId, datasourceId]);

  // Initial load
  useEffect(() => {
    if (activeTab === 'browse') {
      fetchMarketplaceItems();
    } else if (activeTab === 'my-items') {
      fetchTenantItems();
    }
  }, [activeTab, fetchMarketplaceItems, fetchTenantItems]);

  // ========================================================================
  // Add Item to Tenant
  // ========================================================================

  const confirm = useConfirm();
  const notification = useNotification();

  const handleAddItemToTenant = async (itemId: string) => {
    if (!tenantId || !datasourceId) return;

    try {
      const response = await fetch(
        `/api/marketplace/items/add-to-tenant?tenant_id=${tenantId}&tenant_instance_id=${datasourceId}`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'X-Tenant-ID': tenantId,
            'X-Tenant-Datasource-ID': datasourceId,
          },
          body: JSON.stringify({
            marketplace_item_id: itemId,
          }),
        }
      );

      if (!response.ok) throw new Error('Failed to add item');

      // Refresh both lists
      await fetchMarketplaceItems();
      await fetchTenantItems();
      notification.success('Item added to your platform!');
    } catch (err) {
      notification.error(`Error adding item: ${err instanceof Error ? err.message : 'Unknown error'}`);
    }
  };

  const handleRemoveItemFromTenant = async (tenantItemId: string) => {
    if (!tenantId || !datasourceId) return;

    const confirm = useConfirm();
    if (!(await confirm({ title: 'Remove item', description: 'Are you sure you want to remove this item?' }))) return;

    try {
      const response = await fetch(
        `/api/marketplace/tenant-items/${tenantItemId}?tenant_id=${tenantId}&tenant_instance_id=${datasourceId}`,
        {
          method: 'DELETE',
          headers: {
            'X-Tenant-ID': tenantId,
            'X-Tenant-Datasource-ID': datasourceId,
          },
        }
      );

      if (!response.ok) throw new Error('Failed to remove item');

      await fetchTenantItems();
      notification.success('Item removed successfully');
    } catch (err) {
      notification.error(`Error removing item: ${err instanceof Error ? err.message : 'Unknown error'}`);
    }
  };

  // ========================================================================
  // Rate Item
  // ========================================================================

  const handleRateItem = async (itemId: string, rating: number, feedback?: string) => {
    if (!tenantId || !datasourceId) return;

    try {
      const response = await fetch(
        `/api/marketplace/items/${itemId}/feedback?tenant_id=${tenantId}&tenant_instance_id=${datasourceId}`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'X-Tenant-ID': tenantId,
            'X-Tenant-Datasource-ID': datasourceId,
          },
          body: JSON.stringify({
            rating,
            feedback,
          }),
        }
      );

      if (!response.ok) throw new Error('Failed to submit feedback');

      notification.success('Thank you for your feedback!');
    } catch (err) {
      notification.error(`Error submitting feedback: ${err instanceof Error ? err.message : 'Unknown error'}`);
    }
  };

  // ========================================================================
  // Render
  // ========================================================================

  const isItemAdded = (itemId: string) => {
    return tenantItems.some(t => t.marketplace_item_id === itemId);
  };

  return (
    <div className={styles.container}>
      {/* Header */}
      <div className={styles.header}>
        <h1>📦 Marketplace</h1>
        <p>Discover and add rules and calculations to your platform</p>
      </div>

      {/* Tabs */}
      <div className={styles.tabs}>
        <button
          className={`${styles.tab} ${activeTab === 'browse' ? styles.active : ''}`}
          onClick={() => setActiveTab('browse')}
        >
          📚 Browse Catalog
        </button>
        <button
          className={`${styles.tab} ${activeTab === 'validation-rules' ? styles.active : ''}`}
          onClick={() => setActiveTab('validation-rules')}
        >
          ✅ Validation Rules
        </button>
        <button
          className={`${styles.tab} ${activeTab === 'my-items' ? styles.active : ''}`}
          onClick={() => setActiveTab('my-items')}
        >
          ⭐ My Items ({tenantItems.length})
        </button>
        <button
          className={`${styles.tab} ${activeTab === 'analytics' ? styles.active : ''}`}
          onClick={() => setActiveTab('analytics')}
        >
          📊 Analytics
        </button>
      </div>

      {/* Browse Catalog Tab */}
      {activeTab === 'browse' && (
        <div className={styles.browseTab}>
          {/* Filters */}
          <aside className={styles.sidebar}>
            <h3>Search & Filter</h3>

            <div className={styles.searchBox}>
              <input
                type="text"
                placeholder="Search items..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className={styles.searchInput}
              />
            </div>

            <div className={styles.filterSection}>
              <label htmlFor="item-type-select">Item Type</label>
              <select
                id="item-type-select"
                value={selectedItemType}
                onChange={(e) => setSelectedItemType(e.target.value as any)}
              >
                <option value="">All Types</option>
                <option value="rule">Rules</option>
                <option value="calculation">Calculations</option>
              </select>
            </div>

            <div className={styles.filterSection}>
              <label>
                <input
                  type="checkbox"
                  checked={showOnlyOfficial}
                  onChange={(e) => setShowOnlyOfficial(e.target.checked)}
                />
                Only Official
              </label>
            </div>

            <div className={styles.filterSection}>
              <label>
                <input
                  type="checkbox"
                  checked={showOnlyCore}
                  onChange={(e) => setShowOnlyCore(e.target.checked)}
                />
                Only Core
              </label>
            </div>

            <div className={styles.filterSection}>
              <label htmlFor="sort-by-select">Sort By</label>
              <select
                id="sort-by-select"
                title="Sort By"
                aria-label="Sort By"
                value={sortBy}
                onChange={(e) => setSortBy(e.target.value as any)}
              >
                <option value="relevance">Relevance</option>
                <option value="popular">Most Popular</option>
                <option value="rating">Highest Rated</option>
                <option value="newest">Newest</option>
              </select>
            </div>

            <button
              className={styles.clearButton}
              onClick={() => {
                setSearchTerm('');
                setSelectedItemType('');
                setSelectedCategories([]);
                setSelectedSeverities([]);
                setShowOnlyOfficial(false);
                setShowOnlyCore(false);
              }}
            >
              Clear Filters
            </button>
          </aside>

          {/* Content */}
          <main className={styles.content}>
            {loading ? (
              <div className={styles.loading}>Loading marketplace items...</div>
            ) : error ? (
              <div className={styles.error}>{error}</div>
            ) : marketplaceItems.length === 0 ? (
              <div className={styles.emptyState}>
                <p>No items found. Try adjusting your filters.</p>
              </div>
            ) : (
              <>
                <div className={styles.resultsHeader}>
                  <span>{marketplaceItems.length} items found</span>
                  <div className={styles.viewButtons}>
                    <button
                      className={`${styles.viewButton} ${viewMode === 'grid' ? styles.active : ''}`}
                      onClick={() => setViewMode('grid')}
                      title="Grid view"
                    >
                      ⊞
                    </button>
                    <button
                      className={`${styles.viewButton} ${viewMode === 'list' ? styles.active : ''}`}
                      onClick={() => setViewMode('list')}
                      title="List view"
                    >
                      ☰
                    </button>
                  </div>
                </div>

                {viewMode === 'grid' && (
                  <div className={styles.grid}>
                    {marketplaceItems.map((item) => (
                      <div
                        key={item.id}
                        className={styles.card}
                        onClick={() => setSelectedItem(item)}
                      >
                        <div className={styles.cardHeader}>
                          <span className={styles.icon}>{item.icon_emoji}</span>
                          <div className={styles.titleSection}>
                            <h4>{item.name}</h4>
                            {item.is_official && <span className={styles.badge}>OFFICIAL</span>}
                            {item.is_core && <span className={styles.badge}>CORE</span>}
                          </div>
                        </div>

                        <p className={styles.summary}>{item.summary}</p>

                        <div className={styles.meta}>
                          <span className={styles.category}>{item.category}</span>
                          {item.severity && (
                            <span className={styles.severity}>{item.severity}</span>
                          )}
                          {item.rating && (
                            <span className={styles.rating}>⭐ {item.rating.toFixed(1)}</span>
                          )}
                        </div>

                        <div className={styles.actions}>
                          {isItemAdded(item.id) ? (
                            <button className={styles.addedButton} disabled>
                              ✓ Added
                            </button>
                          ) : (
                            <button
                              className={styles.addButton}
                              onClick={(e) => {
                                e.stopPropagation();
                                handleAddItemToTenant(item.id);
                              }}
                            >
                              + Add to Platform
                            </button>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                )}

                {viewMode === 'list' && (
                  <div className={styles.list}>
                    {marketplaceItems.map((item) => (
                      <div
                        key={item.id}
                        className={styles.listItem}
                        onClick={() => setSelectedItem(item)}
                      >
                        <div className={styles.listContent}>
                          <div className={styles.listHeader}>
                            <span className={styles.icon}>{item.icon_emoji}</span>
                            <h4>{item.name}</h4>
                            {item.is_official && <span className={styles.badge}>OFFICIAL</span>}
                          </div>
                          <p className={styles.description}>{item.summary}</p>
                          <div className={styles.listMeta}>
                            <span>{item.category}</span>
                            {item.severity && <span>{item.severity}</span>}
                            <span>{item.usage_count} uses</span>
                          </div>
                        </div>

                        <button
                          className={`${styles.listAction} ${
                            isItemAdded(item.id) ? styles.added : ''
                          }`}
                          onClick={(e) => {
                            e.stopPropagation();
                            if (!isItemAdded(item.id)) {
                              handleAddItemToTenant(item.id);
                            }
                          }}
                        >
                          {isItemAdded(item.id) ? '✓ Added' : '+ Add'}
                        </button>
                      </div>
                    ))}
                  </div>
                )}
              </>
            )}
          </main>
        </div>
      )}

      {/* Validation Rules Tab */}
      {activeTab === 'validation-rules' && (
        <div className={styles.validationRulesTab}>
          <MarketplaceValidationRulesBrowser
            tenantId={tenantId}
            datasourceId={datasourceId}
          />
        </div>
      )}

      {/* My Items Tab */}
      {activeTab === 'my-items' && (
        <div className={styles.myItemsTab}>
          {tenantItems.length === 0 ? (
            <div className={styles.emptyState}>
              <p>You haven't added any items yet. Browse the catalog to get started!</p>
              <button
                className={styles.primaryButton}
                onClick={() => setActiveTab('browse')}
              >
                Browse Marketplace →
              </button>
            </div>
          ) : (
            <div className={styles.itemsList}>
              {tenantItems.map((tenantItem) => {
                const marketplaceItem = marketplaceItems.find(
                  (m) => m.id === tenantItem.marketplace_item_id
                );

                if (!marketplaceItem) return null;

                return (
                  <div key={tenantItem.id} className={styles.myItemCard}>
                    <div className={styles.myItemHeader}>
                      <div>
                        <h4>{tenantItem.custom_name || marketplaceItem.name}</h4>
                        <p>{marketplaceItem.summary}</p>
                      </div>
                      <div className={styles.myItemStatus}>
                        {tenantItem.enabled_for_tenant ? (
                          <span className={styles.statusActive}>● Active</span>
                        ) : (
                          <span className={styles.statusInactive}>● Disabled</span>
                        )}
                      </div>
                    </div>

                    <div className={styles.myItemDetails}>
                      <span>Added: {new Date(tenantItem.added_at).toLocaleDateString()}</span>
                      <span>Version: {tenantItem.local_version}</span>
                      <span>Used: {tenantItem.usage_count} times</span>
                    </div>

                    <div className={styles.myItemActions}>
                      <button className={styles.infoButton}>ℹ️ Details</button>
                      <button className={styles.editButton}>✏️ Configure</button>
                      <button
                        className={styles.removeButton}
                        onClick={() => handleRemoveItemFromTenant(tenantItem.id)}
                      >
                        🗑️ Remove
                      </button>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      )}

      {/* Analytics Tab */}
      {activeTab === 'analytics' && (
        <div className={styles.analyticsTab}>
          <div className={styles.analyticsContent}>
            <h2>Marketplace Analytics</h2>
            <p>Coming soon: Track usage, performance, and feedback on your added items.</p>
            <div className={styles.analyticsCards}>
              <div className={styles.analyticsCard}>
                <h4>Total Items Added</h4>
                <p className={styles.analyticsNumber}>{tenantItems.length}</p>
              </div>
              <div className={styles.analyticsCard}>
                <h4>Total Uses</h4>
                <p className={styles.analyticsNumber}>
                  {tenantItems.reduce((sum, item) => sum + item.usage_count, 0)}
                </p>
              </div>
              <div className={styles.analyticsCard}>
                <h4>Active Items</h4>
                <p className={styles.analyticsNumber}>
                  {tenantItems.filter((item) => item.enabled_for_tenant).length}
                </p>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Detail Modal */}
      {selectedItem && (
        <div className={styles.modal} onClick={() => setSelectedItem(null)}>
          <div className={styles.modalContent} onClick={(e) => e.stopPropagation()}>
            <button
              className={styles.closeButton}
              onClick={() => setSelectedItem(null)}
            >
              ✕
            </button>

            <div className={styles.modalHeader}>
              <div className={styles.headerContent}>
                <h2>
                  <span className={styles.largeIcon}>{selectedItem.icon_emoji}</span>
                  {selectedItem.name}
                </h2>
                <p className={styles.version}>v{selectedItem.version}</p>
              </div>
              <div className={styles.headerMeta}>
                {selectedItem.is_official && <span className={styles.officialBadge}>OFFICIAL</span>}
                {selectedItem.is_core && <span className={styles.coreBadge}>CORE</span>}
              </div>
            </div>

            <div className={styles.modalBody}>
              <p>{selectedItem.long_description}</p>

              <div className={styles.detailsGrid}>
                <div className={styles.detailItem}>
                  <strong>Category:</strong>
                  <span>{selectedItem.category}</span>
                </div>
                <div className={styles.detailItem}>
                  <strong>Type:</strong>
                  <span>{selectedItem.item_type === 'rule' ? 'Validation Rule' : 'Calculation'}</span>
                </div>
                {selectedItem.severity && (
                  <div className={styles.detailItem}>
                    <strong>Severity:</strong>
                    <span>{selectedItem.severity}</span>
                  </div>
                )}
                <div className={styles.detailItem}>
                  <strong>Frequency:</strong>
                  <span>{selectedItem.frequency}</span>
                </div>
                <div className={styles.detailItem}>
                  <strong>Uses:</strong>
                  <span>{selectedItem.usage_count}</span>
                </div>
                {selectedItem.rating && (
                  <div className={styles.detailItem}>
                    <strong>Rating:</strong>
                    <span>⭐ {selectedItem.rating.toFixed(1)}</span>
                  </div>
                )}
              </div>

              {selectedItem.external_api_providers && selectedItem.external_api_providers.length > 0 && (
                <div className={styles.providersSection}>
                  <strong>External APIs:</strong>
                  <div className={styles.providers}>
                    {selectedItem.external_api_providers.map((provider) => (
                      <span key={provider} className={styles.provider}>
                        {provider}
                      </span>
                    ))}
                  </div>
                </div>
              )}
            </div>

            <div className={styles.modalFooter}>
              {isItemAdded(selectedItem.id) ? (
                <span className={styles.addedText}>✓ Already added to your platform</span>
              ) : (
                <button
                  className={styles.primaryButton}
                  onClick={() => {
                    handleAddItemToTenant(selectedItem.id);
                    setSelectedItem(null);
                  }}
                >
                  + Add to Platform
                </button>
              )}
            </div>
          </div>
        </div>
      )}

    </div>
  );
  } catch (error) {
    console.error('[Marketplace] Error rendering component:', error);
    return (
      <div style={{ padding: '20px', color: 'red' }}>
        <h2>Error loading Marketplace</h2>
        <p>{error instanceof Error ? error.message : String(error)}</p>
      </div>
    );
  }
};

export default Marketplace;
