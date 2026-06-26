import React, { createContext, useContext, useState, ReactNode } from 'react';

export interface DashboardTenant {
  id: string;
  name: string;
}

export interface DashboardContextType {
  selectedTenant: DashboardTenant | null;
  valuationDate: string;
  selectTenant: (tenant: DashboardTenant) => void;
  setValuationDate: (date: string) => void;
  isLoading: boolean;
}

const DashboardContext = createContext<DashboardContextType | null>(null);

export function DashboardProvider({ children }: { children: ReactNode }) {
  const [selectedTenant, setSelectedTenantState] = useState<DashboardTenant | null>(() => {
    const stored = localStorage.getItem('dashboard_selected_tenant');
    return stored ? JSON.parse(stored) : null;
  });

  const [valuationDate, setValuationDateState] = useState(() => 
    localStorage.getItem('dashboard_valuation_date') || new Date().toISOString().split('T')[0]
  );

  const [isLoading, setIsLoading] = useState(false);

  const selectTenant = (tenant: DashboardTenant) => {
    localStorage.setItem('dashboard_selected_tenant', JSON.stringify(tenant));
    setSelectedTenantState(tenant);
  };

  const setValuationDate = (date: string) => {
    localStorage.setItem('dashboard_valuation_date', date);
    setValuationDateState(date);
  };

  const value: DashboardContextType = {
    selectedTenant,
    valuationDate,
    selectTenant,
    setValuationDate,
    isLoading,
  };

  return (
    <DashboardContext.Provider value={value}>
      {children}
    </DashboardContext.Provider>
  );
}

export function useDashboardContext(): DashboardContextType {
  const context = useContext(DashboardContext);
  if (!context) {
    throw new Error('useDashboardContext must be used within DashboardProvider');
  }
  return context;
}
