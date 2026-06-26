import React, { Suspense, ReactNode } from 'react';

// Dynamic import wrapper for reactflow to keep it out of the main bundle.
const ReactFlowAsync = React.lazy(async () => {
  const mod = await import('reactflow');
  // also import styles (will be included in the chunk)
  await import('reactflow/dist/style.css');
  return { default: mod.default || mod.ReactFlow || mod } as any;
});

export const LazyReactFlow: React.FC<any> = (props) => {
  return (
    <Suspense fallback={<div className="rf-loading">Loading diagram...</div>}>
      <ReactFlowAsync {...props} />
    </Suspense>
  );
};

// Helper to lazy-load subcomponents (Background, Controls, MiniMap) when needed
export const LazyReactFlowSubcomponents = React.lazy(async () => {
  const mod = await import('reactflow');
  return {
    default: {
      Background: mod.Background,
      Controls: mod.Controls,
      MiniMap: mod.MiniMap,
      ReactFlowProvider: mod.ReactFlowProvider,
    }
  } as any;
});

export const ReactFlowFallback: React.FC<{ children?: ReactNode }> = ({ children }) => (
  <Suspense fallback={<div className="rf-loading">Loading diagram controls...</div>}>
    {children}
  </Suspense>
);

export default LazyReactFlow;
