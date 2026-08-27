import React, { useState } from 'react';
import {
  Box,
  Paper,
  InputBase,
  Chip,
  Stack,
  IconButton,
  Tooltip,
  Divider,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import InsightsIcon from '@mui/icons-material/Insights';
import VerifiedIcon from '@mui/icons-material/Verified';
import ArrowForwardIcon from '@mui/icons-material/ArrowForward';

export interface SemanticChip {
  id: string;
  label: string;
  type: 'DIMENSION' | 'MEASURE' | 'FILTER' | 'DATE_BUCKET' | 'SORT';
}

interface ThoughtSpotOmniboxProps {
  tenantId?: string;
  boKey?: string;
  onExecuteSearch?: (chips: SemanticChip[]) => void;
  onRunSpotIQ?: () => void;
}

export const ThoughtSpotOmnibox: React.FC<ThoughtSpotOmniboxProps> = ({
  onExecuteSearch,
  onRunSpotIQ,
}) => {
  const [inputVal, setInputVal] = useState('');
  const [chips, setChips] = useState<SemanticChip[]>([
    { id: '1', label: 'Tech Sector', type: 'FILTER' },
    { id: '2', label: 'SUM(Market Value)', type: 'MEASURE' },
    { id: '3', label: 'by Country', type: 'DIMENSION' },
  ]);

  const [isGoldenCertified] = useState(true);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && inputVal.trim()) {
      const newChip: SemanticChip = {
        id: String(Date.now()),
        label: inputVal.trim(),
        type: inputVal.toLowerCase().includes('sum') ? 'MEASURE' : 'DIMENSION',
      };
      const updated = [...chips, newChip];
      setChips(updated);
      setInputVal('');
      if (onExecuteSearch) onExecuteSearch(updated);
    }
  };

  const removeChip = (id: string) => {
    const updated = chips.filter((c) => c.id !== id);
    setChips(updated);
    if (onExecuteSearch) onExecuteSearch(updated);
  };

  const getChipColor = (type: SemanticChip['type']) => {
    switch (type) {
      case 'MEASURE':
        return { bg: 'rgba(16, 185, 129, 0.15)', text: '#34D399', border: '#059669' };
      case 'DIMENSION':
        return { bg: 'rgba(59, 130, 246, 0.15)', text: '#60A5FA', border: '#2563EB' };
      case 'FILTER':
        return { bg: 'rgba(245, 158, 11, 0.15)', text: '#FBBF24', border: '#D97706' };
      default:
        return { bg: 'rgba(148, 163, 184, 0.15)', text: '#CBD5E1', border: '#475569' };
    }
  };

  return (
    <Box sx={{ width: '100%', mb: 3, fontFamily: 'sans-serif' }}>
      <Paper
        sx={{
          p: '6px 14px',
          display: 'flex',
          alignItems: 'center',
          bgcolor: '#071526',
          border: '1px solid #1E293B',
          borderRadius: 2.5,
          boxShadow: '0 4px 20px rgba(0,0,0,0.35)',
        }}
      >
        <SearchIcon sx={{ color: '#00D4FF', mr: 1.5, fontSize: 22 }} />

        {/* Grounded Semantic Chips */}
        <Stack direction="row" spacing={1} sx={{ mr: 1, flexWrap: 'nowrap' }}>
          {chips.map((chip) => {
            const style = getChipColor(chip.type);
            return (
              <Chip
                key={chip.id}
                label={chip.label}
                onDelete={() => removeChip(chip.id)}
                size="small"
                sx={{
                  bgcolor: style.bg,
                  color: style.text,
                  border: `1px solid ${style.border}`,
                  fontWeight: 600,
                  fontSize: '11px',
                  fontFamily: 'monospace',
                }}
              />
            );
          })}
        </Stack>

        {/* Active Input */}
        <InputBase
          sx={{
            flex: 1,
            color: '#F8FAFC',
            fontSize: '13px',
            fontFamily: 'sans-serif',
          }}
          placeholder="Ask a question, add metrics, dimensions, or filters (e.g., 'top 10 by yield')..."
          value={inputVal}
          onChange={(e) => setInputVal(e.target.value)}
          onKeyDown={handleKeyDown}
        />

        {/* Golden Certification Seal */}
        {isGoldenCertified && (
          <Tooltip title="Verified Golden Asset: Grounded in audited catalog bindings">
            <Chip
              icon={<VerifiedIcon sx={{ fontSize: '14px !important', color: '#F59E0B' }} />}
              label="Golden Answer"
              size="small"
              sx={{
                bgcolor: 'rgba(245, 158, 11, 0.12)',
                color: '#FBBF24',
                border: '1px solid rgba(245, 158, 11, 0.3)',
                fontSize: '10px',
                fontWeight: 700,
                mr: 1,
              }}
            />
          </Tooltip>
        )}

        <Divider sx={{ height: 28, mx: 1, borderColor: '#1E293B' }} orientation="vertical" />

        {/* Autonomous SpotIQ Trigger */}
        <Tooltip title="Run SpotIQ: Autonomous Variance & Outlier Decomposition">
          <IconButton
            onClick={onRunSpotIQ}
            sx={{
              bgcolor: 'rgba(0, 212, 255, 0.1)',
              color: '#00D4FF',
              '&:hover': { bgcolor: 'rgba(0, 212, 255, 0.2)' },
              p: 0.8,
              mr: 0.5,
            }}
          >
            <InsightsIcon sx={{ fontSize: 18 }} />
          </IconButton>
        </Tooltip>

        <IconButton
          onClick={() => onExecuteSearch && onExecuteSearch(chips)}
          sx={{
            bgcolor: '#00D4FF',
            color: '#030914',
            '&:hover': { bgcolor: '#38BDF8' },
            p: 0.8,
          }}
        >
          <ArrowForwardIcon sx={{ fontSize: 18 }} />
        </IconButton>
      </Paper>
    </Box>
  );
};

export default ThoughtSpotOmnibox;
