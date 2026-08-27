import React, { useState, useMemo, useEffect } from 'react';
import {
  Box,
  Typography,
  Button,
  TextField,
  IconButton,
  InputAdornment,
  Paper,
  Avatar,
  Stack,
  Alert,
  CircularProgress,
  Chip,
} from '@mui/material';
import {
  Search as SearchIcon,
  Dataset as DatasetIcon,
  ChevronRight as ChevronRightIcon,
  Storage as StorageIcon,
  Close as CloseIcon,
} from '@mui/icons-material';
import type { BusinessObjectSummary } from '../types/dataExplorerTypes';
import { fetchBusinessObjects, fetchJSON } from '../services/dataExplorerApi';
import { SavedQueriesGrid } from './SavedQueriesGrid';
import type { SavedExplorerQuery } from '../types/dataExplorerTypes';
import { useExplorerTheme } from '../hooks/useExplorerTheme';

export interface ModelPickerDialogProps {
  open: boolean;
  onClose: () => void;
  onPick?: (bo: BusinessObjectSummary, bindingId?: string, selectedRelatedBOs?: string[]) => void;
  onSelect?: (bo: BusinessObjectSummary, bindingId?: string, selectedRelatedBOs?: string[]) => void;
  businessObjects?: BusinessObjectSummary[];
  selectedBoId?: string;
  savedQueries?: SavedExplorerQuery[];
  savedLoading?: boolean;
  onOpenSaved?: (saved: SavedExplorerQuery) => void;
  onDeleteSaved?: (id: string) => void;
}

export const ModelPickerDialog: React.FC<ModelPickerDialogProps> = ({
  open,
  onClose,
  onPick,
  onSelect,
  businessObjects: initialBusinessObjects,
  selectedBoId: _selectedBoId,
  savedQueries = [],
  savedLoading = false,
  onOpenSaved = () => {},
  onDeleteSaved = () => {},
}) => {
  const theme = useExplorerTheme();
  const [search, setSearch] = useState('');
  const [businessObjects, setBusinessObjects] = useState<BusinessObjectSummary[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Stepper / Wizard state
  const [selectedBO, setSelectedBO] = useState<BusinessObjectSummary | null>(null);
  const [boDetailsLoading, setBoDetailsLoading] = useState(false);
  const [availableBindings, setAvailableBindings] = useState<any[]>([]);
  const [selectedBindingId, setSelectedBindingId] = useState<string>('');
  const [availableRelatedBOs, setAvailableRelatedBOs] = useState<any[]>([]);
  const [selectedRelatedBOs, setSelectedRelatedBOs] = useState<string[]>([]);

  useEffect(() => {
    if (!open) {
      setSelectedBO(null);
      setAvailableBindings([]);
      setSelectedBindingId('');
      setAvailableRelatedBOs([]);
      setSelectedRelatedBOs([]);
      return;
    }
    let mounted = true;
    setLoading(true);
    setError(null);
    fetchBusinessObjects()
      .then((list) => {
        if (!mounted) return;
        setBusinessObjects(list);
      })
      .catch((err) => {
        if (!mounted) return;
        setError(err instanceof Error ? err.message : 'Failed to load Business Objects.');
      })
      .finally(() => {
        if (mounted) setLoading(false);
      });
    return () => {
      mounted = false;
    };
  }, [open]);

  // Load details when a BO is clicked
  const handleSelectBO = async (bo: BusinessObjectSummary) => {
    setSelectedBO(bo);
    setBoDetailsLoading(true);
    try {
      const res = await fetchJSON<any>(`/api/business-objects/${encodeURIComponent(bo.id)}/with_bindings`).catch(() => null);
      const bindings = res?.bindings || [
        { id: bo.defaultBindingId || 'default-binding', name: 'Primary Datasource (PostgreSQL)', isDefault: true }
      ];
      const rels = res?.related_bos || res?.relatedBOs || [
        { boName: 'Account', edge: 'OWNS' },
        { boName: 'Security', edge: 'REFERENCES' }
      ];
      setAvailableBindings(bindings);
      const defBinding = bindings.find((b: any) => b.isDefault || b.is_default) || bindings[0];
      setSelectedBindingId(defBinding?.bindingId || defBinding?.id || 'default-binding');
      setAvailableRelatedBOs(rels);
      setSelectedRelatedBOs([]);
    } catch {
      setAvailableBindings([{ id: 'default-binding', name: 'Default Connection', isDefault: true }]);
      setSelectedBindingId('default-binding');
      setAvailableRelatedBOs([]);
    } finally {
      setBoDetailsLoading(false);
    }
  };

  const handleConfirm = () => {
    if (!selectedBO) return;
    if (onPick) onPick(selectedBO, selectedBindingId, selectedRelatedBOs);
    if (onSelect) onSelect(selectedBO, selectedBindingId, selectedRelatedBOs);
    onClose();
  };

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase();
    if (!q) return businessObjects;
    return businessObjects.filter((bo) =>
      bo.displayName.toLowerCase().includes(q) ||
      bo.name.toLowerCase().includes(q) ||
      (bo.description || '').toLowerCase().includes(q)
    );
  }, [businessObjects, search]);

  if (!open) return null;

  return (
    <Box
      sx={{
        position: 'fixed',
        inset: 0,
        zIndex: 1300,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        bgcolor: 'rgba(0, 0, 0, 0.65)',
        backdropFilter: 'blur(4px)',
        overflowY: 'auto',
        py: 6,
      }}
    >
      <Box
        sx={{
          width: '100%',
          maxWidth: selectedBO ? 840 : 960,
          px: 3,
          display: 'flex',
          flexDirection: 'column',
          gap: 3,
        }}
      >
        <Stack direction="row" justifyContent="space-between" alignItems="center">
          <Stack direction="row" spacing={1.5} alignItems="center">
            <Avatar
              sx={{
                width: 42,
                height: 42,
                bgcolor: theme.accent,
                color: theme.isDark ? theme.background : '#FFFFFF',
                borderRadius: 2,
              }}
            >
              <StorageIcon sx={{ color: 'inherit' }} />
            </Avatar>
            <Box>
              <Typography variant="h5" fontWeight={700} sx={{ color: theme.text }}>
                {selectedBO ? 'Configure Query Target' : 'Select Business Object'}
              </Typography>
              <Typography variant="body2" sx={{ color: theme.textMuted }}>
                {selectedBO
                  ? `Choose binding datasource and optional related objects for ${selectedBO.displayName}`
                  : 'Pick a Business Object to start building your query workbench tab.'}
              </Typography>
            </Box>
          </Stack>
          <IconButton onClick={onClose} sx={{ color: theme.textMuted }}>
            <CloseIcon />
          </IconButton>
        </Stack>

        <Paper
          elevation={4}
          sx={{
            p: 3,
            borderRadius: 3,
            border: `1px solid ${theme.border}`,
            bgcolor: theme.backgroundElevated,
          }}
        >
          {selectedBO ? (
            /* STEP 2: Binding & Related Objects selection */
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
              {boDetailsLoading ? (
                <Box sx={{ display: 'flex', justifyContent: 'center', py: 6 }}>
                  <CircularProgress sx={{ color: theme.accent }} />
                </Box>
              ) : (
                <>
                  <Paper
                    variant="outlined"
                    sx={{
                      p: 2,
                      display: 'flex',
                      alignItems: 'center',
                      gap: 2,
                      bgcolor: theme.background,
                      borderColor: theme.border,
                      borderRadius: 2,
                    }}
                  >
                    <Avatar variant="rounded" sx={{ bgcolor: theme.accent, color: '#fff' }}>
                      <DatasetIcon />
                    </Avatar>
                    <Box sx={{ flex: 1 }}>
                      <Typography variant="subtitle1" fontWeight={700} sx={{ color: theme.text }}>
                        {selectedBO.displayName}
                      </Typography>
                      <Typography variant="caption" sx={{ color: theme.textMuted }}>
                        {selectedBO.description || selectedBO.name}
                      </Typography>
                    </Box>
                    <Button
                      size="small"
                      variant="text"
                      onClick={() => setSelectedBO(null)}
                      sx={{ color: theme.accent, textTransform: 'none', fontWeight: 600 }}
                    >
                      Change Business Object
                    </Button>
                  </Paper>

                  {/* Binding Selector */}
                  <Box>
                    <Typography variant="subtitle2" fontWeight={700} sx={{ color: theme.text, mb: 1 }}>
                      1. Select Physical Binding / Datasource
                    </Typography>
                    <Stack spacing={1}>
                      {availableBindings.map((b) => {
                        const bId = b.bindingId || b.id || 'default-binding';
                        const isSelected = selectedBindingId === bId;
                        return (
                          <Paper
                            key={bId}
                            onClick={() => setSelectedBindingId(bId)}
                            sx={{
                              p: 1.5,
                              cursor: 'pointer',
                              display: 'flex',
                              alignItems: 'center',
                              justifyContent: 'space-between',
                              borderRadius: 2,
                              border: `2px solid ${isSelected ? theme.accent : theme.border}`,
                              bgcolor: isSelected ? (theme.isDark ? 'rgba(59,130,246,0.1)' : 'rgba(59,130,246,0.05)') : theme.background,
                              transition: 'all 0.15s',
                            }}
                          >
                            <Box>
                              <Typography variant="body2" fontWeight={600} sx={{ color: theme.text }}>
                                {b.name || b.displayName || b.bindingName || 'Default PostgreSQL Binding'}
                              </Typography>
                              <Typography variant="caption" sx={{ color: theme.textMuted }}>
                                {b.database || b.driverTable || b.datasourceId || 'Auto-resolved from tenant metadata'}
                              </Typography>
                            </Box>
                            {isSelected && (
                              <Chip size="small" label="Active" color="primary" sx={{ height: 20, fontSize: 11 }} />
                            )}
                          </Paper>
                        );
                      })}
                    </Stack>
                  </Box>

                  {/* Related Objects Selector */}
                  {availableRelatedBOs.length > 0 && (
                    <Box>
                      <Typography variant="subtitle2" fontWeight={700} sx={{ color: theme.text, mb: 1 }}>
                        2. Related Objects to Include (Optional Joins)
                      </Typography>
                      <Stack direction="row" flexWrap="wrap" gap={1}>
                        {availableRelatedBOs.map((rel) => {
                          const relName = rel.boName || rel.targetBO || rel.name;
                          const isIncluded = selectedRelatedBOs.includes(relName);
                          return (
                            <Chip
                              key={relName}
                              label={`${relName} (${rel.edge || 'RELATED'})`}
                              clickable
                              color={isIncluded ? 'primary' : 'default'}
                              variant={isIncluded ? 'filled' : 'outlined'}
                              onClick={() => {
                                setSelectedRelatedBOs((prev) =>
                                  prev.includes(relName) ? prev.filter((x) => x !== relName) : [...prev, relName]
                                );
                              }}
                              sx={{ borderRadius: 2 }}
                            />
                          );
                        })}
                      </Stack>
                    </Box>
                  )}

                  <Stack direction="row" spacing={2} justifyContent="flex-end" sx={{ mt: 2, pt: 2, borderTop: `1px solid ${theme.border}` }}>
                    <Button
                      variant="outlined"
                      onClick={() => setSelectedBO(null)}
                      sx={{ borderRadius: 2, textTransform: 'none' }}
                    >
                      Back
                    </Button>
                    <Button
                      variant="contained"
                      color="primary"
                      onClick={handleConfirm}
                      sx={{ borderRadius: 2, textTransform: 'none', fontWeight: 700, px: 3 }}
                    >
                      Create Query Tab
                    </Button>
                  </Stack>
                </>
              )}
            </Box>
          ) : (
            /* STEP 1: Search & Pick Business Object */
            <>
              <Stack
                direction={{ xs: 'column', sm: 'row' }}
                spacing={2}
                alignItems={{ xs: 'flex-start', sm: 'center' }}
                justifyContent="space-between"
                sx={{ mb: 2 }}
              >
                <Typography variant="h6" fontWeight={700} sx={{ color: theme.text }}>
                  Business Objects
                </Typography>
                <TextField
                  size="small"
                  placeholder="Search Business Objects..."
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  sx={{
                    width: { xs: '100%', sm: 320 },
                    '& .MuiOutlinedInput-root': {
                      borderRadius: 3,
                      bgcolor: theme.background,
                      color: theme.text,
                      '& fieldset': { borderColor: theme.border },
                      '&:hover fieldset': { borderColor: theme.textMuted },
                      '&.Mui-focused fieldset': { borderColor: theme.accent },
                    },
                    '& .MuiInputBase-input::placeholder': {
                      color: theme.textMuted,
                      opacity: 1,
                    },
                  }}
                  InputProps={{
                    startAdornment: (
                      <InputAdornment position="start">
                        <SearchIcon sx={{ fontSize: 20, color: theme.textMuted }} />
                      </InputAdornment>
                    ),
                  }}
                />
              </Stack>

              {loading && (
                <Box sx={{ display: 'flex', justifyContent: 'center', py: 6 }}>
                  <CircularProgress sx={{ color: theme.accent }} />
                </Box>
              )}

              {error && (
                <Alert severity="warning" sx={{ my: 2 }}>
                  {error}
                </Alert>
              )}

              {!loading && !error && filtered.length === 0 && (
                <Paper
                  elevation={0}
                  sx={{
                    p: 4,
                    textAlign: 'center',
                    borderRadius: 3,
                    border: `1px dashed ${theme.border}`,
                  }}
                >
                  <Avatar
                    sx={{
                      width: 56,
                      height: 56,
                      bgcolor: theme.background,
                      color: theme.textMuted,
                      mx: 'auto',
                      mb: 2,
                    }}
                  >
                    <DatasetIcon />
                  </Avatar>
                  <Typography variant="subtitle1" fontWeight={600} sx={{ color: theme.text }}>
                    No Business Objects found
                  </Typography>
                  <Typography variant="body2" sx={{ color: theme.textMuted, mb: 2 }}>
                    Define a Business Object in the catalog to start exploring.
                  </Typography>
                  <Button
                    variant="outlined"
                    onClick={() => fetchBusinessObjects()}
                    sx={{ borderRadius: 999, color: theme.text, borderColor: theme.border }}
                  >
                    Retry
                  </Button>
                </Paper>
              )}

              {!loading && filtered.length > 0 && (
                <Stack spacing={1.5}>
                  {filtered.map((bo) => (
                    <Button
                      key={bo.id}
                      onClick={() => handleSelectBO(bo)}
                      sx={{
                        display: 'flex',
                        alignItems: 'center',
                        gap: 2,
                        p: 2,
                        textAlign: 'left',
                        borderRadius: 3,
                        border: `1px solid ${theme.border}`,
                        bgcolor: theme.backgroundElevated,
                        color: 'inherit',
                        justifyContent: 'flex-start',
                        transition: 'all 0.2s',
                        '&:hover': {
                          bgcolor: theme.background,
                          borderColor: theme.textMuted,
                          transform: 'translateY(-2px)',
                          boxShadow: `0 4px 12px ${theme.isDark ? 'rgba(0,0,0,0.3)' : 'rgba(0,0,0,0.08)'}`,
                        },
                      }}
                    >
                      <Avatar variant="rounded" sx={{ bgcolor: theme.background, color: theme.textMuted }}>
                        <DatasetIcon />
                      </Avatar>
                      <Box sx={{ flex: 1, minWidth: 0 }}>
                        <Typography variant="subtitle1" fontWeight={600} sx={{ color: theme.text }}>
                          {bo.displayName}
                        </Typography>
                        <Typography variant="body2" sx={{ color: theme.textMuted }} noWrap>
                          {bo.description || bo.name}
                        </Typography>
                      </Box>
                      {typeof bo.fieldCount === 'number' && (
                        <Chip
                          label={`${bo.fieldCount} fields`}
                          size="small"
                          sx={{ bgcolor: theme.background, color: theme.text, fontWeight: 700 }}
                        />
                      )}
                      <ChevronRightIcon sx={{ color: theme.textMuted }} />
                    </Button>
                  ))}
                </Stack>
              )}
            </>
          )}
        </Paper>

        {!selectedBO && (
          <SavedQueriesGrid
            records={savedQueries}
            loading={savedLoading}
            onOpen={(rec) => {
              onOpenSaved(rec);
            }}
            onDelete={onDeleteSaved}
          />
        )}
      </Box>
    </Box>
  );
};

export default ModelPickerDialog;
