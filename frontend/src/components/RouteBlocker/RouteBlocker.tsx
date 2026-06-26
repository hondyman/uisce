import React, { createContext, useRef } from 'react';

type BlockHandler = (tx: any) => Promise<boolean> | boolean;

type RouteBlockerContextShape = {
  register: (h: BlockHandler) => () => void;
  run: (tx: any) => Promise<boolean>;
};

const RouteBlockerContext = createContext<RouteBlockerContextShape | null>(null);

export const RouteBlockerProvider: React.FC<{ children?: React.ReactNode }> = ({ children }) => {
  const handlersRef = useRef<BlockHandler[]>([]);
  const runHandlers = async (tx: any) => {
    const handlers = handlersRef.current;
    if (!handlers || handlers.length === 0) return true;
    // call handlers from last to first, stop at first that returns boolean
    for (let i = handlers.length - 1; i >= 0; i--) {
      try {
        const res = handlers[i](tx);
        const allowed = await Promise.resolve(res);
        if (!allowed) return false;
      } catch (e) {
        // If handler throws, treat as allowed and continue
        continue;
      }
    }
    return true;
  };
  const register = (h: BlockHandler) => {
    handlersRef.current.push(h);
    return () => { handlersRef.current = handlersRef.current.filter(x => x !== h); };
  };

  const run = async (tx: any) => {
    try {
      return await runHandlers(tx);
    } catch (e) {
      return true;
    }
  };

  return (
    <RouteBlockerContext.Provider value={{ register, run }}>
      {children}
    </RouteBlockerContext.Provider>
  );
};

export const useRouteBlocker = () => {
  const ctx = React.useContext(RouteBlockerContext);
  if (!ctx) {
    // Return a safe no-op implementation so components can be rendered in tests
    return {
      register: (_: BlockHandler) => () => {},
      run: async (_tx: any) => true,
    } as RouteBlockerContextShape;
  }
  return ctx;
};

export default RouteBlockerProvider;
