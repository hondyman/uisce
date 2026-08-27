import React from 'react';
import { Box, Typography, FormControlLabel, Switch, TextField, Button, Divider } from '@mui/material';
import { PaginationConfig } from '../tableColumnModel';

interface PaginationEditorProps {
  pagination: PaginationConfig;
  onChange: (pagination: PaginationConfig) => void;
}

export const PaginationEditor: React.FC<PaginationEditorProps> = ({
  pagination,
  onChange,
}) => {
  const update = (partial: Partial<PaginationConfig>) => {
    onChange({ ...pagination, ...partial });
  };

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
        <Typography variant="caption" sx={{ fontSize: '0.75rem', fontWeight: 600, mb: 0.5 }}>
          Preview Mode
        </Typography>
        <Box sx={{ display: 'flex', gap: 0.5 }}>
          <Button
            size="small"
            variant={pagination.mode === 'expand' ? 'contained' : 'outlined'}
            onClick={() => update({ mode: 'expand' })}
            sx={{ flex: 1, textTransform: 'none', fontSize: '0.7rem', py: 0.4 }}
          >
            Show All Rows
          </Button>
          <Button
            size="small"
            variant={pagination.mode === 'paginate' ? 'contained' : 'outlined'}
            onClick={() => update({ mode: 'paginate' })}
            sx={{ flex: 1, textTransform: 'none', fontSize: '0.7rem', py: 0.4 }}
          >
            Paginate
          </Button>
        </Box>
      </Box>

      {pagination.mode === 'paginate' && (
        <>
          <TextField
            size="small"
            label="Rows per page"
            type="number"
            value={pagination.rowsPerPage}
            onChange={e => update({ rowsPerPage: Math.max(1, parseInt(e.target.value, 10) || 20) })}
            fullWidth
            inputProps={{ min: 1 }}
            sx={{ '& .MuiInputBase-input': { fontSize: '0.75rem' } }}
          />

          <Divider sx={{ borderColor: 'rgba(255,255,255,0.06)' }} />

          <FormControlLabel
            control={
              <Switch
                size="small"
                checked={pagination.repeatHeadersOnEachPage}
                onChange={(_, v) => update({ repeatHeadersOnEachPage: v })}
              />
            }
            label={
              <Typography variant="caption" sx={{ fontSize: '0.75rem', fontWeight: 600 }}>
                Repeat headers on each page
              </Typography>
            }
          />

          <Divider sx={{ borderColor: 'rgba(255,255,255,0.06)' }} />

          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
            <FormControlLabel
              control={
                <Switch
                  size="small"
                  checked={pagination.pageTotalEnabled}
                  onChange={(_, v) => update({ pageTotalEnabled: v })}
                />
              }
              label={
                <Typography variant="caption" sx={{ fontSize: '0.75rem', fontWeight: 600 }}>
                  Page total row
                </Typography>
              }
            />

            {pagination.pageTotalEnabled && (
              <Box sx={{ pl: 2, display: 'flex', flexDirection: 'column', gap: 1 }}>
                <TextField
                  size="small"
                  label="Label"
                  value={pagination.pageTotalLabel}
                  onChange={e => update({ pageTotalLabel: e.target.value })}
                  fullWidth
                  sx={{ '& .MuiInputBase-input': { fontSize: '0.75rem' } }}
                />
                <Box sx={{ display: 'flex', gap: 0.5 }}>
                  <Button
                    size="small"
                    variant={pagination.pageTotalPosition === 'top' ? 'contained' : 'outlined'}
                    onClick={() => update({ pageTotalPosition: 'top' })}
                    sx={{ flex: 1, textTransform: 'none', fontSize: '0.7rem', py: 0.4 }}
                  >
                    Top
                  </Button>
                  <Button
                    size="small"
                    variant={pagination.pageTotalPosition === 'bottom' ? 'contained' : 'outlined'}
                    onClick={() => update({ pageTotalPosition: 'bottom' })}
                    sx={{ flex: 1, textTransform: 'none', fontSize: '0.7rem', py: 0.4 }}
                  >
                    Bottom
                  </Button>
                </Box>
              </Box>
            )}
          </Box>
        </>
      )}
    </Box>
  );
};

export default PaginationEditor;
