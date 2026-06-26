import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { devWarn, devError } from '../../utils/devLogger';

import { PrivateMarketsBundle as PMBundle } from '../../types/bundles';

export interface User {
  id: string;
  name: string;
  role: 'lp' | 'gp' | 'fof' | 'steward';
  organization: string;
  permissions: string[];
}

export type Bundle = PMBundle;

interface ExplorerContextType {
  user: User | null;
  bundle: Bundle | null;
  setUser: (user: User | null) => void;
  setBundle: (bundle: Bundle | null) => void;
  isLoading: boolean;
  error: string | null;
  excelResults: Record<string, Record<string, any>> | null;
  selectedEntities: string[];
  setSelectedEntities: (entities: string[]) => void;
}

const ExplorerContext = createContext<ExplorerContextType | undefined>(undefined);

interface ExplorerProviderProps {
  children: ReactNode;
}

export const ExplorerProvider: React.FC<ExplorerProviderProps> = ({ children }) => {
  const [user, setUser] = useState<User | null>(null);
  const [bundle, setBundle] = useState<Bundle | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [excelResults, setExcelResults] = useState<Record<string, Record<string, any>> | null>(null);
  const [selectedEntities, setSelectedEntities] = useState<string[]>([]);

  // Initialize user and bundle based on URL params or local storage
  useEffect(() => {
    const initializeContext = async () => {
      setIsLoading(true);
      try {
        // Get user role from URL params or local storage
        const urlParams = new URLSearchParams(window.location.search);
        const role = (urlParams.get('role') as 'lp' | 'gp' | 'fof' | 'steward') || 'lp'; // default to lp

        // Mock user data - in real app, this would come from auth service
        const mockUser: User = {
          id: 'user-1',
          name: 'John Doe',
          role,
          organization: 'Sample Organization',
          permissions: ['read', 'write', 'admin']
        };

        setUser(mockUser);

        // Load appropriate bundle based on role
        try {
          const bundleModule = await import(`./bundles/${role}_private_markets_bundle.json`);
          setBundle(bundleModule.default || bundleModule);
        } catch (e) {
          // If bundle import fails, continue with null bundle
          devWarn('Bundle import failed:', e);
          setBundle(null);
        }

        setError(null);
      } catch (err) {
        setError('Failed to initialize explorer context');
        devError('Context initialization error:', err);
      } finally {
        setIsLoading(false);
      }
    };

    initializeContext();
  }, []);

  // Vectorized Excel calculation logic
  useEffect(() => {
    if (!bundle || selectedEntities.length === 0) {
      setExcelResults(null);
      return;
    }

    const excelMetrics = bundle.metrics?.filter(m => m.financial_calc?.type === 'excel_formula') || [];
    if (excelMetrics.length === 0) {
      setExcelResults(null);
      return;
    }

    // Request vectorized calculations for Excel metrics
    const requestVectorizedCalculations = async () => {
      try {
        const response = await fetch('/api/calc/vectorized', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            metrics: excelMetrics.map(m => m.node_id),
            entities: selectedEntities
          })
        });

        if (!response.ok) {
          throw new Error(`HTTP error! status: ${response.status}`);
        }

        const data = await response.json();
        setExcelResults(data.results || null);
      } catch (err) {
        devError('Failed to fetch vectorized calculations:', err);
        setError('Failed to load Excel calculations');
      }
    };

    requestVectorizedCalculations();
  }, [bundle, selectedEntities]);

  const value: ExplorerContextType = {
    user,
    bundle,
    setUser,
    setBundle,
    isLoading,
    error,
    excelResults,
    selectedEntities,
    setSelectedEntities,
  };

  return (
    <ExplorerContext.Provider value={value}>
      {children}
    </ExplorerContext.Provider>
  );
};

export const useExplorer = (): ExplorerContextType => {
  const context = useContext(ExplorerContext);
  if (context === undefined) {
    throw new Error('useExplorer must be used within an ExplorerProvider');
  }
  return context;
};
