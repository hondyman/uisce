import React, { createContext, useState, useContext, ReactNode } from 'react';
import UnifiedHeatmapDrillDown from '../features/fabric/components/UnifiedHeatmapDrillDown';

export interface FilterState {
  severity?: string[];
  dateRange?: { from: string; to: string };
  policyId?: string;
  versionA?: number;
  versionB?: number;
  bucket?: string;
  changeType?: string;
  runId?: string;
  migrationSQL?: string;
  environment?: string;
  // UI-specific context
  policyName?: string;
  bucketSize?: string;
}

interface DrillDownContextType {
  showDrillDown: (context: string, payload: any) => void;
  hideDrillDown: () => void;
  context: string | null;
  filters: FilterState | null;
  setFilters: React.Dispatch<React.SetStateAction<FilterState | null>>;
}

const DrillDownContext = createContext<DrillDownContextType | undefined>(undefined);

function getDefaultsForContext(context: string, payload: any): FilterState {
  switch (context) {
    case 'historical':
      return {
        policyId: payload.policyId,
        policyName: payload.policyName,
        bucket: payload.bucket,
        dateRange: { from: payload.fromDate, to: payload.toDate },
        bucketSize: payload.bucketSize,
      };
    case 'policy_compare':
      return {
        policyId: payload.policyId,
        versionA: payload.versionA,
        versionB: payload.versionB,
        bucket: payload.bucket,
        changeType: payload.changeType,
        dateRange: { from: payload.fromDate, to: payload.toDate },
        bucketSize: payload.bucketSize,
      };
    // Add other contexts like 'single_run' or 'forecast_detail' here
    default:
      return {};
  }
}

export const DrillDownProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
  const [isOpen, setIsOpen] = useState(false);
  const [context, setContext] = useState<string | null>(null);
  const [filters, setFilters] = useState<FilterState | null>(null);

  const showDrillDown = (ctx: string, payload: any) => {
    const defaultFilters = getDefaultsForContext(ctx, payload);
    setContext(ctx);
    setFilters(defaultFilters);
    setIsOpen(true);
  };

  const hideDrillDown = () => {
    setIsOpen(false);
    // Reset context and filters after a short delay to allow the drawer to close smoothly
    setTimeout(() => {
      setContext(null);
      setFilters(null);
    }, 300);
  };

  const value = { showDrillDown, hideDrillDown, context, filters, setFilters };

  return (
    <DrillDownContext.Provider value={value}>
      {children}
      <UnifiedHeatmapDrillDown open={isOpen} onClose={hideDrillDown} />
    </DrillDownContext.Provider>
  );
};

export const useDrillDown = (): DrillDownContextType => {
  const context = useContext(DrillDownContext);
  if (context === undefined) {
    throw new Error('useDrillDown must be used within a DrillDownProvider');
  }
  return context;
};