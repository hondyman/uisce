import { useCallback, useState } from 'react';

const useSidebarWidth = (initialDefault = 450) => {
  const [isSidebarCollapsed, setIsSidebarCollapsed] = useState(false);

  const [sidebarWidth, setSidebarWidth] = useState<number>(() => {
    try {
      const s = localStorage.getItem('catalogSidebarWidth');
      return s ? Number(s) : initialDefault;
    } catch (e) {
      return initialDefault;
    }
  });

  const onResize = useCallback((_e: any, data: { size: { width: number } }) => {
    const w = Math.max(200, Math.min(600, Math.round(data.size.width)));
    setSidebarWidth(w);
    if (isSidebarCollapsed) setIsSidebarCollapsed(false);
    try { localStorage.setItem('catalogSidebarWidth', String(w)); } catch (e) { /* ignore */ }
  }, [isSidebarCollapsed]);

  return { sidebarWidth, setSidebarWidth, onResize, isSidebarCollapsed, setIsSidebarCollapsed } as const;
};

export default useSidebarWidth;
