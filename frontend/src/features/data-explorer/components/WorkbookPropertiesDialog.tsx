import React, { useState } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  TextField,
  Typography,
  Stack,
  Box,
  MenuItem,
  Select,
  FormControl,
  InputLabel,
  Chip,
  Switch,
  FormControlLabel,
  Divider,
} from '@mui/material';
import {
  MenuBook as BookIcon,
  Info as InfoIcon,
  Settings as SettingsIcon,
} from '@mui/icons-material';
import { useExplorerTheme } from '../hooks/useExplorerTheme';
import type { QueryWorkbook } from '../types/dataExplorerTypes';

export interface WorkbookPropertiesDialogProps {
  open: boolean;
  onClose: () => void;
  workbook: Partial<QueryWorkbook>;
  onSave: (props: {
    name: string;
    description: string;
    category?: string;
    tags?: string[];
    refreshIntervalMinutes?: number;
    autoExecuteOnLoad?: boolean;
  }) => void;
}

export const WorkbookPropertiesDialog: React.FC<WorkbookPropertiesDialogProps> = ({
  open,
  onClose,
  workbook,
  onSave,
}) => {
  const theme = useExplorerTheme();
  const [name, setName] = useState(workbook.name || '');
  const [description, setDescription] = useState(workbook.description || '');
  const [category, setCategory] = useState('Portfolio Management');
  const [tagInput, setTagInput] = useState('');
  const [tags, setTags] = useState<string[]>(['Analytics', 'SQL', 'Uuisce']);
  const [refreshInterval, setRefreshInterval] = useState<number>(0);
  const [autoExecute, setAutoExecute] = useState(true);

  const handleAddTag = () => {
    if (tagInput.trim() && !tags.includes(tagInput.trim())) {
      setTags([...tags, tagInput.trim()]);
      setTagInput('');
    }
  };

  const handleRemoveTag = (tagToRemove: string) => {
    setTags(tags.filter((t) => t !== tagToRemove));
  };

  const handleSubmit = () => {
    onSave({
      name: name.trim() || 'Untitled Workbook',
      description: description.trim(),
      category,
      tags,
      refreshIntervalMinutes: refreshInterval,
      autoExecuteOnLoad: autoExecute,
    });
    onClose();
  };

  return (
    <Dialog
      open={open}
      onClose={onClose}
      maxWidth="sm"
      fullWidth
      PaperProps={{
        sx: {
          bgcolor: theme.backgroundElevated,
          border: `1px solid ${theme.border}`,
          borderRadius: 3,
        },
      }}
    >
      <DialogTitle sx={{ borderBottom: `1px solid ${theme.border}`, px: 3, py: 2 }}>
        <Stack direction="row" spacing={1.5} alignItems="center">
          <BookIcon sx={{ color: theme.accent }} />
          <Box>
            <Typography variant="subtitle1" fontWeight={700} sx={{ color: theme.text }}>
              Workbook Properties & Metadata
            </Typography>
            <Typography variant="caption" sx={{ color: theme.textMuted }}>
              Manage title, description, governance tags, and auto-refresh intervals.
            </Typography>
          </Box>
        </Stack>
      </DialogTitle>

      <DialogContent sx={{ p: 3 }}>
        <Stack spacing={2.5} sx={{ mt: 1 }}>
          <TextField
            label="Workbook Title"
            size="small"
            fullWidth
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="e.g. Executive Risk & PnL Ledger"
            autoFocus
          />

          <TextField
            label="Workbook Description & Purpose"
            size="small"
            fullWidth
            multiline
            rows={3}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Describe the business objectives, data sources, audit assumptions, and target audience..."
          />

          <FormControl size="small" fullWidth>
            <InputLabel>Business Category / Domain</InputLabel>
            <Select
              value={category}
              label="Business Category / Domain"
              onChange={(e) => setCategory(e.target.value)}
            >
              <MenuItem value="Portfolio Management">Portfolio Management (OMS)</MenuItem>
              <MenuItem value="Risk & Compliance">Risk & Compliance</MenuItem>
              <MenuItem value="Alternative Investments">Alternative Investments (PE/VC/Real Estate)</MenuItem>
              <MenuItem value="Cash Flow & Settlement">Cash Flow & Settlement</MenuItem>
              <MenuItem value="Master Directory">Master Directory & Accounts</MenuItem>
              <MenuItem value="Executive BI">Executive BI & Board Reporting</MenuItem>
            </Select>
          </FormControl>

          {/* Tag Manager */}
          <Box>
            <Typography variant="caption" fontWeight={700} sx={{ color: theme.textMuted, mb: 1, display: 'block' }}>
              Governance Tags:
            </Typography>
            <Stack direction="row" spacing={1} sx={{ mb: 1.5 }}>
              <TextField
                size="small"
                placeholder="Add tag (e.g. SOX, Monthly, Production)..."
                value={tagInput}
                onChange={(e) => setTagInput(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    e.preventDefault();
                    handleAddTag();
                  }
                }}
                sx={{ flex: 1 }}
              />
              <Button
                variant="outlined"
                size="small"
                onClick={handleAddTag}
                sx={{ textTransform: 'none', borderRadius: 2 }}
              >
                Add Tag
              </Button>
            </Stack>

            <Stack direction="row" spacing={0.8} flexWrap="wrap" useFlexGap>
              {tags.map((t) => (
                <Chip
                  key={t}
                  label={t}
                  size="small"
                  onDelete={() => handleRemoveTag(t)}
                  sx={{
                    bgcolor: 'rgba(0, 201, 200, 0.1)',
                    color: theme.accent,
                    border: `1px solid ${theme.accent}40`,
                    fontWeight: 600,
                  }}
                />
              ))}
            </Stack>
          </Box>

          <Divider sx={{ borderColor: theme.border }} />

          {/* Execution Settings */}
          <Stack spacing={1.5}>
            <FormControlLabel
              control={
                <Switch
                  checked={autoExecute}
                  onChange={(e) => setAutoExecute(e.target.checked)}
                />
              }
              label={
                <Box>
                  <Typography variant="body2" fontWeight={600} sx={{ color: theme.text }}>
                    Auto-Run Queries on Load
                  </Typography>
                  <Typography variant="caption" sx={{ color: theme.textMuted }}>
                    Automatically fetch data when opening this workbook.
                  </Typography>
                </Box>
              }
            />

            <FormControl size="small" fullWidth>
              <InputLabel>Background Auto-Refresh</InputLabel>
              <Select
                value={refreshInterval}
                label="Background Auto-Refresh"
                onChange={(e) => setRefreshInterval(Number(e.target.value))}
              >
                <MenuItem value={0}>Manual Refresh Only (Disabled)</MenuItem>
                <MenuItem value={1}>Every 1 Minute (Realtime)</MenuItem>
                <MenuItem value={5}>Every 5 Minutes</MenuItem>
                <MenuItem value={15}>Every 15 Minutes</MenuItem>
                <MenuItem value={60}>Every 1 Hour</MenuItem>
              </Select>
            </FormControl>
          </Stack>
        </Stack>
      </DialogContent>

      <DialogActions sx={{ px: 3, py: 2, borderTop: `1px solid ${theme.border}` }}>
        <Button onClick={onClose} sx={{ textTransform: 'none', color: theme.textMuted }}>
          Cancel
        </Button>
        <Button
          variant="contained"
          onClick={handleSubmit}
          sx={{
            textTransform: 'none',
            fontWeight: 700,
            borderRadius: 2,
            bgcolor: theme.accent,
            color: theme.text,
            '&:hover': { bgcolor: theme.accentDark },
          }}
        >
          Save Properties
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default WorkbookPropertiesDialog;
