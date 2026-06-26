import React, { useRef } from 'react';
import './BuilderTabs.css';

interface Tab {
  id: string;
  label: string;
}

interface BuilderTabsProps {
  activeTab: string;
  setActiveTab: (tab: string) => void;
  tabs: Tab[];
}

const BuilderTabs: React.FC<BuilderTabsProps> = ({ activeTab, setActiveTab, tabs }) => {
  const containerRef = useRef<HTMLDivElement | null>(null);

  const onKeyDown = (e: React.KeyboardEvent) => {
    const currentIndex = tabs.findIndex(t => t.id === activeTab);
    if (e.key === 'ArrowRight') {
      const next = tabs[(currentIndex + 1) % tabs.length];
      setActiveTab(next.id);
      // move focus
      const btn = containerRef.current?.querySelector<HTMLButtonElement>(`button[data-tab="${next.id}"]`);
      btn?.focus();
      e.preventDefault();
    } else if (e.key === 'ArrowLeft') {
      const prev = tabs[(currentIndex - 1 + tabs.length) % tabs.length];
      setActiveTab(prev.id);
      const btn = containerRef.current?.querySelector<HTMLButtonElement>(`button[data-tab="${prev.id}"]`);
      btn?.focus();
      e.preventDefault();
    }
  };

  return (
    <div className="workspace-tab-header" role="tablist" aria-label="Workspace tabs" ref={containerRef} onKeyDown={onKeyDown}>
      {tabs.map(tab => (
        <button
          key={tab.id}
          data-tab={tab.id}
          role="tab"
          {...{ 'aria-selected': activeTab === tab.id }}
          tabIndex={activeTab === tab.id ? 0 : -1}
          className={`btn btn-sm workspace-tab-button ${activeTab === tab.id ? 'btn-primary' : 'btn-outline'}`}
          onClick={() => setActiveTab(tab.id)}
          title={tab.label}
          aria-label={tab.label}
        >
          {tab.label}
        </button>
      ))}
    </div>
  );
};

export default BuilderTabs;
