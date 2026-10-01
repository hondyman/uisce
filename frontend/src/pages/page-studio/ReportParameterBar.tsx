import React, { useEffect, useState } from 'react';
import {
  Box,
  Paper,
  Stack,
  Typography,
  TextField,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Button,
  Chip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Table,
  TableHead,
  TableBody,
  TableRow,
  TableCell,
  IconButton,
} from '@mui/material';
import TuneIcon from '@mui/icons-material/Tune';
import LinkIcon from '@mui/icons-material/Link';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';
import CloseIcon from '@mui/icons-material/Close';
import type { ComponentDefinition, ReportParameterBarConfig, ReportParameterItem } from '../../types/pageStudio';

export interface ReportParameterBarProps {
  config: ReportParameterBarConfig;
  components?: Record<string, ComponentDefinition>;
  variables: Record<string, any>;
  onUpdateVariable: (name: string, value: any) => void;
  onBulkBindTiles?: (paramVarName: string, updatedComponents: Record<string, ComponentDefinition>) => void;
  mode?: 'design' | 'preview';
  searchParams?: URLSearchParams;
  onUpdateSearchParams?: (params: Record<string, string>) => void;
}

export const ReportParameterBar: React.FC<ReportParameterBarProps> = ({
  config,
  components = {},
  variables,
  onUpdateVariable,
  onBulkBindTiles,
  mode = 'preview',
  searchParams,
  onUpdateSearchParams,
}) => {
  const [bindModalParam, setBindModalParam] = useState<ReportParameterItem | null>(null);

  // Hydrate parameters on mount from URL search params or defaults
  useEffect(() => {
    if (!config?.parameters) return;
    const urlUpdates: Record<string, string> = {};

    for (const p of config.parameters) {
      const fromUrl = searchParams?.get(p.varName);
      if (fromUrl !== null && fromUrl !== undefined) {
        onUpdateVariable(p.varName, fromUrl);
      } else if (variables[p.varName] === undefined && p.default !== undefined) {
        onUpdateVariable(p.varName, p.default);
        urlUpdates[p.varName] = String(p.default);
      }
    }

    if (Object.keys(urlUpdates).length > 0 && onUpdateSearchParams) {
      onUpdateSearchParams(urlUpdates);
    }
  }, [config?.parameters]);

  const handleChange = (param: ReportParameterItem, value: any) => {
    onUpdateVariable(param.varName, value);
    if (onUpdateSearchParams) {
      onUpdateSearchParams({ [param.varName]: String(value ?? '') });
    }
  };

  // Inspect matching status for bulk bind modal
  const tileMatches = (param: ReportParameterItem) => {
    return Object.values(components).map((comp) => {
      const savedQueryParams = comp.props?.savedQueryParams as Record<string, any> | undefined;
      const currentBinding = savedQueryParams?.[param.varName];
      const isBound = currentBinding?.mode === 'pageVar' && currentBinding?.varName === param.varName;

      return {
        comp,
        isBound,
        title: comp.label || comp.id,
      };
    });
  };

  const handleApplyBulkBind = (param: ReportParameterItem) => {
    if (!onBulkBindTiles) return;
    const updated = { ...components };

    for (const [id, comp] of Object.entries(updated)) {
      const prevParams = (comp.props?.savedQueryParams as Record<string, any>) || {};
      updated[id] = {
        ...comp,
        props: {
          ...comp.props,
          savedQueryParams: {
            ...prevParams,
            [param.varName]: { mode: 'pageVar', varName: param.varName },
          },
        },
      };
    }

    onBulkBindTiles(param.varName, updated);
    setBindModalParam(null);
  };

  if (!config?.enabled || !config.parameters?.length) {
    return null;
  }

  return (
    <Paper
      variant="outlined"
      sx={{
        p: 1.5,
        mb: 2,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        flexWrap: 'wrap',
        gap: 2,
        bgcolor: 'background.paper',
      }}
    >
      <Stack direction="row" alignItems="center" spacing={2} flexWrap="wrap">
        <Stack direction="row" alignItems="center" spacing={0.5}>
          <TuneIcon fontSize="small" color="primary" />
          <Typography variant="subtitle2" fontWeight={700}>
            Report Parameters
          </Typography>
        </Stack>

        {config.parameters.map((p) => {
          const val = variables[p.varName] ?? p.default ?? '';

          return (
            <Stack key={p.varName} direction="row" alignItems="center" spacing={0.5}>
              {p.control === 'date' && (
                <TextField
                  size="small"
                  type="date"
                  label={p.label}
                  InputLabelProps={{ shrink: true }}
                  value={val}
                  onChange={(e) => handleChange(p, e.target.value)}
                  sx={{ minWidth: 150 }}
                />
              )}

              {p.control === 'text' && (
                <TextField
                  size="small"
                  label={p.label}
                  value={val}
                  onChange={(e) => handleChange(p, e.target.value)}
                  sx={{ minWidth: 140 }}
                />
              )}

              {p.control === 'number' && (
                <TextField
                  size="small"
                  type="number"
                  label={p.label}
                  value={val}
                  onChange={(e) => handleChange(p, Number(e.target.value))}
                  sx={{ minWidth: 120 }}
                />
              )}

              {p.control === 'select' && (
                <FormControl size="small" sx={{ minWidth: 150 }}>
                  <InputLabel>{p.label}</InputLabel>
                  <Select
                    value={val}
                    label={p.label}
                    onChange={(e) => handleChange(p, e.target.value)}
                  >
                    {p.selectOptions && p.selectOptions.source === 'static' &&
                      p.selectOptions.options.map((opt) => (
                        <MenuItem key={opt.value} value={opt.value}>
                          {opt.label}
                        </MenuItem>
                      ))}
                    {(!p.selectOptions || p.selectOptions.source !== 'static') && (
                      <MenuItem value={val || 'All'}>{val || 'All'}</MenuItem>
                    )}
                  </Select>
                </FormControl>
              )}

              {mode === 'design' && (
                <IconButton
                  size="small"
                  title={`Bind "${p.varName}" to dashboard tiles`}
                  onClick={() => setBindModalParam(p)}
                  sx={{ p: 0.5 }}
                >
                  <LinkIcon fontSize="small" color="action" />
                </IconButton>
              )}
            </Stack>
          );
        })}
      </Stack>

      {/* Bulk Bind Modal in Design Mode */}
      {bindModalParam && (
        <Dialog open onClose={() => setBindModalParam(null)} maxWidth="sm" fullWidth>
          <DialogTitle sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <span>Bind Parameter: {bindModalParam.label} ({bindModalParam.varName})</span>
            <IconButton size="small" onClick={() => setBindModalParam(null)}>
              <CloseIcon fontSize="small" />
            </IconButton>
          </DialogTitle>
          <DialogContent dividers>
            <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
              Bulk bind this report parameter to matching query tiles on the dashboard.
            </Typography>
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell>Tile / Component</TableCell>
                  <TableCell>Binding Status</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {tileMatches(bindModalParam).map(({ comp, isBound, title }) => (
                  <TableRow key={comp.id}>
                    <TableCell>{title}</TableCell>
                    <TableCell>
                      {isBound ? (
                        <Chip
                          size="small"
                          color="success"
                          icon={<CheckCircleIcon fontSize="small" />}
                          label="Bound to PageVar"
                        />
                      ) : (
                        <Chip
                          size="small"
                          color="default"
                          label="Unbound"
                        />
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setBindModalParam(null)}>Cancel</Button>
            <Button
              variant="contained"
              onClick={() => handleApplyBulkBind(bindModalParam)}
            >
              Bulk Bind All Tiles
            </Button>
          </DialogActions>
        </Dialog>
      )}
    </Paper>
  );
};

export default ReportParameterBar;
