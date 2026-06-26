/**
 * Marketplace Context - Centralized State Management
 * Manages search, filters, installations, and component selection across the marketplace
 */

import React, { createContext, useContext, useState, useCallback, useMemo } from 'react';
import { Component } from '../data/marketplaceComponents';

interface MarketplaceContextType {
  // State
  searchQuery: string;
  selectedCategory: string;
  selectedComponent: Component | null;
  installedComponents: Set<string>;
  sortBy: 'downloads' | 'rating' | 'name';
  priceFilter: {
    free: boolean;
    paid: boolean;
  };

  // Actions
  setSearchQuery: (query: string) => void;
  setSelectedCategory: (category: string) => void;
  setSelectedComponent: (component: Component | null) => void;
  setSortBy: (sort: 'downloads' | 'rating' | 'name') => void;
  setPriceFilter: (filter: { free: boolean; paid: boolean }) => void;
  handleInstall: (componentId: string) => void;
  handleUninstall: (componentId: string) => void;
  isInstalled: (componentId: string) => boolean;
}

const MarketplaceContext = createContext<MarketplaceContextType | undefined>(undefined);

interface MarketplaceProviderProps {
  children: React.ReactNode;
  initialInstalledComponents?: string[];
}

/**
 * Provider component for marketplace state management
 */
export const MarketplaceProvider: React.FC<MarketplaceProviderProps> = ({
  children,
  initialInstalledComponents = ['MetricCardGrid', 'DataTable']
}) => {
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedCategory, setSelectedCategory] = useState('all');
  const [selectedComponent, setSelectedComponent] = useState<Component | null>(null);
  const [installedComponents, setInstalledComponents] = useState<Set<string>>(
    new Set(initialInstalledComponents)
  );
  const [sortBy, setSortBy] = useState<'downloads' | 'rating' | 'name'>('downloads');
  const [priceFilter, setPriceFilter] = useState({ free: true, paid: true });

  const handleInstall = useCallback((componentId: string) => {
    setInstalledComponents((prev) => new Set([...prev, componentId]));
  }, []);

  const handleUninstall = useCallback((componentId: string) => {
    setInstalledComponents((prev) => {
      const newSet = new Set(prev);
      newSet.delete(componentId);
      return newSet;
    });
  }, []);

  const isInstalled = useCallback(
    (componentId: string) => installedComponents.has(componentId),
    [installedComponents]
  );

  const value: MarketplaceContextType = useMemo(
    () => ({
      searchQuery,
      selectedCategory,
      selectedComponent,
      installedComponents,
      sortBy,
      priceFilter,
      setSearchQuery,
      setSelectedCategory,
      setSelectedComponent,
      setSortBy,
      setPriceFilter,
      handleInstall,
      handleUninstall,
      isInstalled
    }),
    [
      searchQuery,
      selectedCategory,
      selectedComponent,
      installedComponents,
      sortBy,
      priceFilter,
      handleInstall,
      handleUninstall,
      isInstalled
    ]
  );

  return (
    <MarketplaceContext.Provider value={value}>{children}</MarketplaceContext.Provider>
  );
};

/**
 * Hook to access marketplace context
 * @throws Error if used outside MarketplaceProvider
 */
export const useMarketplace = (): MarketplaceContextType => {
  const context = useContext(MarketplaceContext);
  if (!context) {
    throw new Error('useMarketplace must be used within a MarketplaceProvider');
  }
  return context;
};
