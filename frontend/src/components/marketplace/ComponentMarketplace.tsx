/**
 * Component Marketplace - Main Component
 * A feature-rich marketplace for discovering and installing dashboard components
 */

import React, { useMemo } from 'react';
import { Package, Code } from 'lucide-react';
import SearchBar from './SearchBar';
import CategoryFilter from './CategoryFilter';
import ComponentCard from './ComponentCard';
import ComponentModal from './ComponentModal';
import FeaturedComponents from './FeaturedComponents';
import { useMarketplace } from '../../contexts/MarketplaceContext';
import { categories, components } from '../../data/marketplaceComponents';

const ComponentMarketplace: React.FC = () => {
  const {
    searchQuery,
    setSearchQuery,
    selectedCategory,
    setSelectedCategory,
    selectedComponent,
    setSelectedComponent,
    installedComponents,
    handleInstall,
    handleUninstall,
    isInstalled,
    sortBy,
    setSortBy,
    priceFilter,
    setPriceFilter
  } = useMarketplace();

  // Filter and sort components
  const filteredAndSortedComponents = useMemo(() => {
    const filtered = components.filter((comp) => {
      // Search filter
      const matchesSearch =
        comp.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
        comp.description.toLowerCase().includes(searchQuery.toLowerCase()) ||
        comp.tags.some((tag) => tag.toLowerCase().includes(searchQuery.toLowerCase()));

      // Category filter
      const matchesCategory = selectedCategory === 'all' || comp.category === selectedCategory;

      // Price filter
      const isPaid = comp.price !== 'free';
      const matchesPrice = (priceFilter.free && !isPaid) || (priceFilter.paid && isPaid);

      return matchesSearch && matchesCategory && matchesPrice;
    });

    // Sort
    return filtered.sort((a, b) => {
      if (sortBy === 'downloads') return b.downloads - a.downloads;
      if (sortBy === 'rating') return b.rating - a.rating;
      return a.name.localeCompare(b.name);
    });
  }, [searchQuery, selectedCategory, sortBy, priceFilter]);

  const featuredComponents = useMemo(() => {
    return components.filter((c) => c.featured);
  }, []);

  const showFeatured = selectedCategory === 'all' && searchQuery === '';

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-900 via-slate-800 to-slate-900 text-white">
      {/* Header */}
      <div className="bg-slate-800/50 backdrop-blur border-b border-slate-700">
        <div className="max-w-7xl mx-auto px-6 py-6">
          <div className="flex items-center justify-between mb-6">
            <div>
              <h1 className="text-4xl font-bold bg-gradient-to-r from-blue-400 to-purple-500 bg-clip-text text-transparent mb-2">
                Component Marketplace
              </h1>
              <p className="text-slate-400">
                Discover and install components for your dashboards
              </p>
            </div>
            <div className="flex items-center gap-3">
              <button
                className="flex items-center gap-2 px-4 py-2 bg-slate-700 hover:bg-slate-600 rounded-lg transition"
                title="View installed components"
              >
                <Package className="w-4 h-4" />
                My Components ({installedComponents.size})
              </button>
              <button
                className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg transition"
                title="Publish new component"
              >
                <Code className="w-4 h-4" />
                Publish Component
              </button>
            </div>
          </div>

          {/* Search Bar */}
          <SearchBar value={searchQuery} onChange={setSearchQuery} />
        </div>
      </div>

      <div className="max-w-7xl mx-auto px-6 py-8">
        <div className="grid grid-cols-1 lg:grid-cols-4 gap-8">
          {/* Sidebar Filters */}
          <aside className="space-y-6">
            <CategoryFilter
              categories={categories}
              selectedCategory={selectedCategory}
              onSelectCategory={setSelectedCategory}
            />

            {/* Price Filter */}
            <div className="bg-slate-800 rounded-xl border border-slate-700 p-4">
              <h3 className="font-bold mb-3">Price</h3>
              <div className="space-y-2">
                <label className="flex items-center gap-2 text-sm cursor-pointer">
                  <input
                    type="checkbox"
                    checked={priceFilter.free}
                    onChange={(e) =>
                      setPriceFilter({ ...priceFilter, free: e.target.checked })
                    }
                    className="rounded"
                  />
                  <span>Free</span>
                </label>
                <label className="flex items-center gap-2 text-sm cursor-pointer">
                  <input
                    type="checkbox"
                    checked={priceFilter.paid}
                    onChange={(e) =>
                      setPriceFilter({ ...priceFilter, paid: e.target.checked })
                    }
                    className="rounded"
                  />
                  <span>Paid</span>
                </label>
              </div>
            </div>

            {/* Sort Options */}
            <div className="bg-slate-800 rounded-xl border border-slate-700 p-4">
              <h3 className="font-bold mb-3" id="sort-label">Sort By</h3>
              <select
                value={sortBy}
                onChange={(e) =>
                  setSortBy(e.target.value as 'downloads' | 'rating' | 'name')
                }
                aria-labelledby="sort-label"
                className="w-full bg-slate-700 text-white px-3 py-2 rounded-lg border border-slate-600 focus:border-blue-500 focus:outline-none"
              >
                <option value="downloads">Most Downloaded</option>
                <option value="rating">Highest Rated</option>
                <option value="name">Alphabetical</option>
              </select>
            </div>

            {/* Pro Tip */}
            <div className="bg-gradient-to-br from-purple-900/30 to-blue-900/30 rounded-xl border border-purple-500/30 p-4">
              <div className="text-sm font-bold mb-2">💡 Pro Tip</div>
              <div className="text-xs text-slate-300">
                Install components to your library and use them in any dashboard with
                drag-and-drop.
              </div>
            </div>
          </aside>

          {/* Main Content */}
          <main className="col-span-1 lg:col-span-3 space-y-6">
            {/* Featured Section */}
            {showFeatured && (
              <FeaturedComponents
                components={featuredComponents}
                onSelectComponent={setSelectedComponent}
              />
            )}

            {/* Components Grid */}
            <div>
              <h2 className="text-xl font-bold mb-4">
                {filteredAndSortedComponents.length} Component
                {filteredAndSortedComponents.length !== 1 ? 's' : ''}
              </h2>

              {filteredAndSortedComponents.length === 0 ? (
                <div className="text-center py-12 text-slate-400">
                  <p className="mb-2">No components found</p>
                  <p className="text-sm">Try adjusting your search or filters</p>
                </div>
              ) : (
                <div className="space-y-4">
                  {filteredAndSortedComponents.map((comp) => (
                    <ComponentCard
                      key={comp.id}
                      component={comp}
                      isInstalled={isInstalled(comp.id)}
                      onInstall={handleInstall}
                      onUninstall={handleUninstall}
                      onSelect={setSelectedComponent}
                    />
                  ))}
                </div>
              )}
            </div>
          </main>
        </div>
      </div>

      {/* Component Detail Modal */}
      <ComponentModal
        component={selectedComponent}
        isInstalled={selectedComponent ? isInstalled(selectedComponent.id) : false}
        onClose={() => setSelectedComponent(null)}
        onInstall={handleInstall}
        onUninstall={handleUninstall}
      />
    </div>
  );
};

export default ComponentMarketplace;
