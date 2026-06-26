// React default import removed — component uses only JSX and no React namespace
import BuilderTabs from './BuilderTabs';

interface Props {
  activeWorkspaceTab: 'canvas' | 'custom' | 'calculations';
  setActiveWorkspaceTab: (t: 'canvas' | 'custom' | 'calculations') => void;
}

const WorkspaceTabs: React.FC<Props> = ({ activeWorkspaceTab, setActiveWorkspaceTab }) => {
  const tabs = [
    { id: 'canvas', label: 'Canvas' },
    { id: 'custom', label: 'Code' },
    { id: 'calculations', label: 'Calculations' },
  ];
  return (
    <BuilderTabs
      activeTab={activeWorkspaceTab}
      setActiveTab={(tab) => setActiveWorkspaceTab(tab as 'canvas' | 'custom' | 'calculations')}
      tabs={tabs}
    />
  );
};

export default WorkspaceTabs;
