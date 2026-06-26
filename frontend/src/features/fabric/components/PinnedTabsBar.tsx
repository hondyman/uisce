import type { FC } from 'react';
import { Box, Tabs, Tab, IconButton, Tooltip } from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import CompareArrowsIcon from '@mui/icons-material/CompareArrows';

export interface TabState {
  id: string;
  label: string;
  context: string;
  filters: any;
  data?: any[];
  isLoading: boolean;
}

interface PinnedTabsBarProps {
  tabs: TabState[];
  activeTabId: string;
  onSelectTab: (id: string) => void;
  onCloseTab: (id: string) => void;
  onStartDiff: () => void;
}

const PinnedTabsBar: FC<PinnedTabsBarProps> = ({ tabs, activeTabId, onSelectTab, onCloseTab, onStartDiff }) => {
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', borderBottom: 1, borderColor: 'divider' }}>
      <Tabs
        value={activeTabId}
        onChange={(_e, newValue) => onSelectTab(newValue)}
        variant="scrollable"
        scrollButtons="auto"
        sx={{ flexGrow: 1 }}
      >
        {tabs.map((tab) => (
          <Tab
            key={tab.id}
            value={tab.id}
            label={tab.label}
            icon={tab.id !== 'base' ? <IconButton size="small" onClick={(e) => { e.stopPropagation(); onCloseTab(tab.id); }}><CloseIcon fontSize="inherit" /></IconButton> : undefined}
            iconPosition="end"
          />
        ))}
      </Tabs>
      <Tooltip title="Compare two tabs">
        <IconButton onClick={onStartDiff} disabled={tabs.length < 2}><CompareArrowsIcon /></IconButton>
      </Tooltip>
    </Box>
  );
};

export default PinnedTabsBar;