import React, { createContext, useContext, useState, useCallback, useRef } from 'react';

export interface EventAction {
  targetChannel: string;
  sourcePropertyKey: string;
  actionType: 'SET_PARAMETER' | 'CLEAR_PARAMETER' | 'NAVIGATE' | 'LAUNCH_MODAL_FORM' | 'TRIGGER_REFETCH';
  targetPageKey?: string;
}

export interface WidgetEventConfig {
  onRowSelect?: EventAction[];
  onRowDoubleClick?: EventAction[];
  onChartSelect?: EventAction[];
}

interface PageEventBusContextType {
  parameters: Record<string, any>;
  setParameter: (channel: string, value: any) => void;
  setParametersBatch: (params: Record<string, any>) => void;
  clearParameter: (channel: string) => void;
  subscribeToParameter: (channel: string) => any;
}

const PageEventBusContext = createContext<PageEventBusContextType | undefined>(undefined);

export const PageEventBusProvider: React.FC<{
  initialParams?: Record<string, any>;
  children: React.ReactNode;
}> = ({ initialParams = {}, children }) => {
  const [parameters, setParameters] = useState<Record<string, any>>(initialParams);
  const paramsRef = useRef(parameters);
  paramsRef.current = parameters;

  const setParameter = useCallback((channel: string, value: any) => {
    // Guard against redundant state updates & infinite loops
    if (paramsRef.current[channel] === value) return;
    setParameters((prev) => ({ ...prev, [channel]: value }));
  }, []);

  const setParametersBatch = useCallback((newParams: Record<string, any>) => {
    setParameters((prev) => {
      let hasChange = false;
      const updated = { ...prev };
      for (const [k, v] of Object.entries(newParams)) {
        if (updated[k] !== v) {
          updated[k] = v;
          hasChange = true;
        }
      }
      return hasChange ? updated : prev;
    });
  }, []);

  const clearParameter = useCallback((channel: string) => {
    setParameters((prev) => {
      if (!(channel in prev)) return prev;
      const next = { ...prev };
      delete next[channel];
      return next;
    });
  }, []);

  const subscribeToParameter = useCallback(
    (channel: string) => parameters[channel],
    [parameters]
  );

  return (
    <PageEventBusContext.Provider
      value={{
        parameters,
        setParameter,
        setParametersBatch,
        clearParameter,
        subscribeToParameter,
      }}
    >
      {children}
    </PageEventBusContext.Provider>
  );
};

export const usePageEventBus = () => {
  const ctx = useContext(PageEventBusContext);
  if (!ctx) throw new Error('usePageEventBus must be used within PageEventBusProvider');
  return ctx;
};
