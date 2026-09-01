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
  Radio,
  Checkbox,
  Tooltip,
} from '@mui/material';
import {
  Search as SearchIcon,
  Dataset as DatasetIcon,
  Storage as StorageIcon,
  Close as CloseIcon,
  Lock as LockIcon,
  CheckCircle as CheckCircleIcon,
} from '@mui/icons-material';
import { fetchBusinessObjects, fetchJSON } from '../../features/data-explorer/services/dataExplorerApi';
import type { BusinessObjectSummary, SavedExplorerQuery } from '../../features/data-explorer/types/dataExplorerTypes';
import { useExplorerTheme } from '../../features/data-explorer/hooks/useExplorerTheme';
import { SavedQueriesGrid } from '../../features/data-explorer/components/SavedQueriesGrid';

export interface UnifiedBOPickerModalProps {
  open: boolean;
  onClose: () => void;
  title?: string;
  subtitle?: string;
  context?: 'report' | 'query' | 'page';
  onPick?: (bo: BusinessObjectSummary, bindingId?: string, selectedRelatedBOs?: string[], bindingDetails?: any, selectedSubtypeKey?: string | null, selectedSubtypeKeys?: string[]) => void;
  onSelect?: (bo: BusinessObjectSummary, bindingId?: string, selectedRelatedBOs?: string[], bindingDetails?: any, selectedSubtypeKey?: string | null, selectedSubtypeKeys?: string[]) => void;
  businessObjects?: BusinessObjectSummary[];
  selectedBoId?: string;
  savedQueries?: SavedExplorerQuery[];
  savedLoading?: boolean;
  onOpenSaved?: (saved: SavedExplorerQuery) => void;
  onDeleteSaved?: (id: string) => void;
}

export const UnifiedBOPickerModal: React.FC<UnifiedBOPickerModalProps> = ({
  open,
  onClose,
  title,
  subtitle,
  context = 'query',
  onPick,
  onSelect,
  businessObjects: initialBusinessObjects,
  savedQueries = [],
  savedLoading = false,
  onOpenSaved = () => {},
  onDeleteSaved = () => {},
}) => {
  const theme = useExplorerTheme();
  const allowMultiSubtype = context === 'page';
  const [search, setSearch] = useState('');
  const [businessObjects, setBusinessObjects] = useState<BusinessObjectSummary[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Selected Business Object & Bindings
  // selectedBO is typed as any to capture the full BO from with_bindings, which includes subtypes
  const [selectedBO, setSelectedBO] = useState<any | null>(null);
  const [boDetailsLoading, setBoDetailsLoading] = useState(false);
  const [availableBindings, setAvailableBindings] = useState<any[]>([]);
  const [selectedBindingId, setSelectedBindingId] = useState<string>('');
  const [availableRelatedBOs, setAvailableRelatedBOs] = useState<any[]>([]);
  const [selectedRelatedBOs, setSelectedRelatedBOs] = useState<string[]>([]);
  const [selectedSubtypeKey, setSelectedSubtypeKey] = useState<string | null>(null);
  const [selectedSubtypeKeys, setSelectedSubtypeKeys] = useState<string[]>([]);

  useEffect(() => {
    if (!open) {
      setSelectedBO(null);
      setAvailableBindings([]);
      setSelectedBindingId('');
      setAvailableRelatedBOs([]);
      setSelectedRelatedBOs([]);
      setSelectedSubtypeKey(null);
      setSelectedSubtypeKeys([]);
      return;
    }

    let mounted = true;
    setLoading(true);
    setError(null);
    fetchBusinessObjects()
      .then((list) => {
        if (!mounted) return;
        setBusinessObjects(list);
        if (list.length > 0 && !selectedBO) {
          handleSelectBO(list[0]);
        }
      })
      .catch((err) => {
        if (!mounted) return;
        setError(err instanceof Error ? err.message : 'Failed to load accessible Business Objects.');
      })
      .finally(() => {
        if (mounted) setLoading(false);
      });
    return () => {
      mounted = false;
    };
  }, [open]);

  // Load details & bindings when a BO is clicked in the left pane
  const handleSelectBO = async (bo: BusinessObjectSummary) => {
    setSelectedBO(bo);
    setSelectedSubtypeKey(null);
    setSelectedSubtypeKeys([]);
    setBoDetailsLoading(true);
    try {
      const res = await fetchJSON<any>(`/api/business-objects/${encodeURIComponent(bo.id)}/with_bindings`).catch(() => null);
      let bindings = res?.bindings || [];
      if (!Array.isArray(bindings) || bindings.length === 0) {
        bindings = [
          {
            id: bo.defaultBindingId || 'default-postgres-binding',
            name: 'Primary Datasource (PostgreSQL)',
            driverTable: `${bo.name || bo.displayName || 'entity'}_table`,
            isDefault: true,
          }
        ];
      }
      const rawBO = res?.businessObject || res?.bo || res || bo;
      const subtypes = rawBO?.subtypes || bo?.subtypes || {};
      const rels = res?.related_bos || res?.relatedBOs || rawBO?.relatedBOs || bo?.relatedBOs || [];
      const fields = res?.fields || rawBO?.fields || bo?.fields || [];
      const coreFields = rawBO?.coreFields || rawBO?.core_fields || bo?.coreFields || bo?.core_fields || (res?.fields ? res.fields : []);

      setSelectedBO({
        ...bo,
        ...rawBO,
        fields,
        coreFields,
        subtypes,
        relatedBOs: rels,
      });

      setAvailableBindings(bindings);
      const defBinding = bindings.find((b: any) => b.isDefault || b.is_default) || bindings[0];
      setSelectedBindingId(defBinding?.bindingId || defBinding?.id || 'default-postgres-binding');
      setAvailableRelatedBOs(rels);
      setSelectedRelatedBOs([]);
    } catch {
      setAvailableBindings([
        {
          id: 'default-postgres-binding',
          name: 'Primary Datasource (PostgreSQL)',
          driverTable: 'catalog_binding_primary',
          isDefault: true,
        }
      ]);
      setSelectedBindingId('default-postgres-binding');
      setAvailableRelatedBOs([]);
    } finally {
      setBoDetailsLoading(false);
    }
  };

  const hasMultipleBindings = availableBindings.length > 1;

  const handleConfirm = () => {
    if (!selectedBO) return;
    const chosenBinding = availableBindings.find(
      (b) => (b.bindingId || b.id) === selectedBindingId
    ) || availableBindings[0];

    const keysArg = allowMultiSubtype ? selectedSubtypeKeys : (selectedSubtypeKey ? [selectedSubtypeKey] : []);
    if (onPick) onPick(selectedBO, selectedBindingId, selectedRelatedBOs, chosenBinding, selectedSubtypeKey, keysArg);
    if (onSelect) onSelect(selectedBO, selectedBindingId, selectedRelatedBOs, chosenBinding, selectedSubtypeKey, keysArg);
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

  const modalTitle = title || (
    context === 'page'
      ? 'New Application Page • Select Root Business Object & Binding'
      : context === 'report'
      ? 'New Report • Select Business Object & Binding'
      : 'New Query • Select Business Object & Binding'
  );
  const modalSubtitle = subtitle || `Choose a Business Object and its Datasource Binding. Once confirmed, this selection is immutable for this ${context === 'page' ? 'application page' : context === 'report' ? 'report' : 'query tab'}.`;

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
        p: 3,
      }}
    >
      <Paper
        elevation={6}
        sx={{
          width: '100%',
          maxWidth: 980,
          maxHeight: '90vh',
          display: 'flex',
          flexDirection: 'column',
          borderRadius: 3,
          border: `1px solid ${theme.border}`,
          bgcolor: theme.backgroundElevated,
          overflow: 'hidden',
        }}
      >
        {/* Modal Header */}
        <Box
          sx={{
            px: 3,
            py: 2,
            borderBottom: `1px solid ${theme.border}`,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            bgcolor: theme.background,
          }}
        >
          <Stack direction="row" spacing={1.5} alignItems="center">
            <Avatar
              sx={{
                width: 38,
                height: 38,
                bgcolor: theme.accent,
                color: theme.isDark ? theme.background : '#FFFFFF',
                borderRadius: 2,
              }}
            >
              <StorageIcon sx={{ color: 'inherit', fontSize: 20 }} />
            </Avatar>
            <Box>
              <Typography variant="subtitle1" fontWeight={700} sx={{ color: theme.text, lineHeight: 1.2 }}>
                {modalTitle}
              </Typography>
              <Typography variant="caption" sx={{ color: theme.textMuted }}>
                {modalSubtitle}
              </Typography>
            </Box>
          </Stack>
          <IconButton size="small" onClick={onClose} sx={{ color: theme.textMuted }}>
            <CloseIcon />
          </IconButton>
        </Box>

        {/* Modal Unified 2-Column Body */}
        <Box sx={{ flex: 1, display: 'flex', overflow: 'hidden', minHeight: 460 }}>
          {/* Left Column: Accessible Business Objects List */}
          <Box
            sx={{
              width: 360,
              borderRight: `1px solid ${theme.border}`,
              display: 'flex',
              flexDirection: 'column',
              bgcolor: theme.background,
            }}
          >
            <Box sx={{ p: 2, borderBottom: `1px solid ${theme.border}` }}>
              <TextField
                size="small"
                fullWidth
                placeholder="Filter accessible objects..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                InputProps={{
                  startAdornment: (
                    <InputAdornment position="start">
                      <SearchIcon sx={{ fontSize: 18, color: theme.textMuted }} />
                    </InputAdornment>
                  ),
                }}
                sx={{
                  '& .MuiOutlinedInput-root': {
                    borderRadius: 2,
                    bgcolor: theme.backgroundElevated,
                    fontSize: '0.85rem',
                  },
                }}
              />
            </Box>

            <Box sx={{ flex: 1, overflowY: 'auto', p: 1.5, display: 'flex', flexDirection: 'column', gap: 1 }}>
              {loading && (
                <Box sx={{ display: 'flex', justifyContent: 'center', py: 4 }}>
                  <CircularProgress size={24} sx={{ color: theme.accent }} />
                </Box>
              )}

              {error && (
                <Alert severity="warning" sx={{ m: 1, fontSize: '0.75rem' }}>
                  {error}
                </Alert>
              )}

              {!loading && filtered.length === 0 && (
                <Typography variant="body2" sx={{ color: theme.textMuted, textAlign: 'center', py: 4 }}>
                  No accessible Business Objects found.
                </Typography>
              )}

              {!loading &&
                filtered.map((bo) => {
                  const isSelected = selectedBO?.id === bo.id;
                  return (
                    <Paper
                      key={bo.id}
                      onClick={() => handleSelectBO(bo)}
                      sx={{
                        p: 1.5,
                        cursor: 'pointer',
                        borderRadius: 2,
                        border: `1.5px solid ${isSelected ? theme.accent : theme.border}`,
                        bgcolor: isSelected
                          ? (theme.isDark ? 'rgba(59,130,246,0.12)' : 'rgba(59,130,246,0.06)')
                          : theme.backgroundElevated,
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'space-between',
                        gap: 1.5,
                        transition: 'all 0.15s',
                        '&:hover': {
                          borderColor: theme.accent,
                        },
                      }}
                    >
                      <Box sx={{ minWidth: 0, flex: 1 }}>
                        <Typography variant="body2" fontWeight={700} sx={{ color: theme.text }} noWrap>
                          {bo.displayName}
                        </Typography>
                        <Typography variant="caption" sx={{ color: theme.textMuted }} noWrap display="block">
                          {bo.description || bo.name}
                        </Typography>
                      </Box>
                      {typeof bo.fieldCount === 'number' && (
                        <Chip
                          label={`${bo.fieldCount} fields`}
                          size="small"
                          sx={{ height: 18, fontSize: 10, bgcolor: theme.background }}
                        />
                      )}
                    </Paper>
                  );
                })}
            </Box>
          </Box>

          {/* Right Column: Binding & Options Details for Selected BO */}
          <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', p: 3, overflowY: 'auto', bgcolor: theme.backgroundElevated }}>
            {selectedBO ? (
              <Stack spacing={2.5}>
                {/* Chosen BO Summary Card */}
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
                      {selectedBO.description || selectedBO.name} • {selectedBO.id}
                    </Typography>
                  </Box>
                  <Chip
                    icon={<LockIcon sx={{ fontSize: '11px !important' }} />}
                    label="Immutable Target"
                    size="small"
                    color="primary"
                    variant="outlined"
                    sx={{ height: 22, fontSize: 10, fontWeight: 700 }}
                  />
                </Paper>

                {/* Subtype Scope Section */}
                {selectedBO?.subtypes && Object.keys(selectedBO.subtypes).length > 0 && (
                  <Box>
                    <Stack direction="row" alignItems="center" justifyContent="space-between" sx={{ mb: 1 }}>
                      <Typography variant="subtitle2" fontWeight={700} sx={{ color: theme.text }}>
                        Subtype Scope
                      </Typography>
                      <Chip
                        label={allowMultiSubtype ? 'Select One or More' : 'STI Partial-Index Routing'}
                        size="small"
                        color="info"
                        variant="outlined"
                        sx={{ height: 20, fontSize: 10 }}
                      />
                    </Stack>
                    <Stack spacing={0.5}>
                      {!allowMultiSubtype && (
                        <Paper
                          onClick={() => setSelectedSubtypeKey(null)}
                          sx={{
                            p: 1.25,
                            cursor: 'pointer',
                            borderRadius: 2,
                            border: `2px solid ${selectedSubtypeKey === null ? theme.accent : theme.border}`,
                            bgcolor:
                              selectedSubtypeKey === null
                                ? theme.isDark
                                  ? 'rgba(59,130,246,0.1)'
                                  : 'rgba(59,130,246,0.05)'
                                : theme.background,
                            transition: 'all 0.15s',
                            '&:hover': { borderColor: theme.accent },
                          }}
                        >
                          <Stack direction="row" alignItems="center" gap={1}>
                            <Radio
                              size="small"
                              checked={selectedSubtypeKey === null}
                              sx={{ p: 0, color: theme.textMuted, '&.Mui-checked': { color: theme.accent } }}
                            />
                            <Box>
                              <Typography variant="body2" fontWeight={700} sx={{ color: theme.text }}>
                                All Subtypes (Core Only)
                              </Typography>
                              <Typography variant="caption" sx={{ color: theme.textMuted }}>
                                Baseline fields only · Full table scan · No subtype-specific joins
                              </Typography>
                            </Box>
                          </Stack>
                        </Paper>
                      )}
                      {Object.entries(selectedBO.subtypes).map(([subKey, subDef]: [string, any]) => {
                        const isSelected = allowMultiSubtype
                          ? selectedSubtypeKeys.includes(subKey)
                          : selectedSubtypeKey === subKey;
                        const toggle = () => {
                          if (allowMultiSubtype) {
                            setSelectedSubtypeKeys((prev) =>
                              prev.includes(subKey) ? prev.filter((k) => k !== subKey) : [...prev, subKey]
                            );
                          } else {
                            setSelectedSubtypeKey(subKey);
                          }
                        };
                        return (
                          <Paper
                            key={subKey}
                            onClick={toggle}
                            sx={{
                              p: 1.25,
                              cursor: 'pointer',
                              borderRadius: 2,
                              border: `2px solid ${isSelected ? theme.accent : theme.border}`,
                              bgcolor: isSelected
                                ? theme.isDark
                                  ? 'rgba(59,130,246,0.1)'
                                  : 'rgba(59,130,246,0.05)'
                                : theme.background,
                              transition: 'all 0.15s',
                              '&:hover': { borderColor: theme.accent },
                            }}
                          >
                            <Stack direction="row" alignItems="center" gap={1}>
                              {allowMultiSubtype ? (
                                <Checkbox
                                  size="small"
                                  checked={isSelected}
                                  sx={{ p: 0, color: theme.textMuted, '&.Mui-checked': { color: theme.accent } }}
                                />
                              ) : (
                                <Radio
                                  size="small"
                                  checked={isSelected}
                                  sx={{ p: 0, color: theme.textMuted, '&.Mui-checked': { color: theme.accent } }}
                                />
                              )}
                              <Box sx={{ flex: 1 }}>
                                <Typography variant="body2" fontWeight={700} sx={{ color: theme.text }}>
                                  {subDef?.displayName || subDef?.name || subKey}
                                </Typography>
                                {subDef?.description && (
                                  <Typography variant="caption" sx={{ color: theme.textMuted }} noWrap>
                                    {subDef.description}
                                  </Typography>
                                )}
                              </Box>
                              <Chip
                                label={`+${(subDef?.subtypeFields?.length ?? 0)} fields`}
                                size="small"
                                sx={{ height: 16, fontSize: 9, bgcolor: theme.border }}
                              />
                            </Stack>
                          </Paper>
                        );
                      })}
                    </Stack>
                    {!allowMultiSubtype && selectedSubtypeKey !== null && (
                      <Alert severity="info" sx={{ borderRadius: 2, fontSize: '0.72rem', mt: 0.5 }}>
                        <strong>STI pushdown active:</strong> Query will route through the{' '}
                        <code>WHERE t0.subtype_code = '{selectedSubtypeKey}'</code> partial index.
                      </Alert>
                    )}
                    {allowMultiSubtype && selectedSubtypeKeys.length > 0 && (
                      <Alert severity="info" sx={{ borderRadius: 2, fontSize: '0.72rem', mt: 0.5 }}>
                        <strong>{selectedSubtypeKeys.length} subtype{selectedSubtypeKeys.length > 1 ? 's' : ''} selected:</strong>{' '}
                        {selectedSubtypeKeys.join(', ')}
                      </Alert>
                    )}
                  </Box>
                )}

                {/* Binding Section */}
                <Box>
                  <Stack direction="row" alignItems="center" justifyContent="space-between" sx={{ mb: 1 }}>
                    <Typography variant="subtitle2" fontWeight={700} sx={{ color: theme.text }}>
                      Datasource Binding
                    </Typography>
                    {!hasMultipleBindings ? (
                      <Chip
                        icon={<LockIcon sx={{ fontSize: 12 }} />}
                        label="Single Binding (Read-Only)"
                        size="small"
                        sx={{ height: 20, fontSize: 10, bgcolor: theme.background, color: theme.textMuted }}
                      />
                    ) : (
                      <Chip
                        label="Multiple Bindings Available"
                        size="small"
                        color="info"
                        variant="outlined"
                        sx={{ height: 20, fontSize: 10 }}
                      />
                    )}
                  </Stack>

                  {boDetailsLoading ? (
                    <Box sx={{ display: 'flex', justifyContent: 'center', py: 3 }}>
                      <CircularProgress size={20} sx={{ color: theme.accent }} />
                    </Box>
                  ) : (
                    <Stack spacing={1}>
                      {availableBindings.map((b) => {
                        const bId = b.bindingId || b.id || 'default-postgres-binding';
                        const isSelected = selectedBindingId === bId;
                        const isDefault = Boolean(b.isDefault || b.is_default);

                        return (
                          <Paper
                            key={bId}
                            onClick={() => {
                              if (hasMultipleBindings) {
                                setSelectedBindingId(bId);
                              }
                            }}
                            sx={{
                              p: 1.5,
                              cursor: hasMultipleBindings ? 'pointer' : 'default',
                              display: 'flex',
                              alignItems: 'center',
                              justifyContent: 'space-between',
                              borderRadius: 2,
                              border: `2px solid ${isSelected ? theme.accent : theme.border}`,
                              bgcolor: isSelected
                                ? (theme.isDark ? 'rgba(59,130,246,0.1)' : 'rgba(59,130,246,0.05)')
                                : theme.background,
                              transition: 'all 0.15s',
                              opacity: !hasMultipleBindings && !isSelected ? 0.6 : 1,
                              '&:hover': hasMultipleBindings
                                ? { borderColor: theme.accent, transform: 'translateY(-1px)' }
                                : {},
                            }}
                          >
                            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
                              {hasMultipleBindings ? (
                                <Radio
                                  size="small"
                                  checked={isSelected}
                                  sx={{ p: 0, color: theme.textMuted, '&.Mui-checked': { color: theme.accent } }}
                                />
                              ) : (
                                <LockIcon sx={{ color: theme.textMuted, fontSize: 16 }} />
                              )}
                              <Box>
                                <Stack direction="row" spacing={1} alignItems="center">
                                  <Typography variant="body2" fontWeight={700} sx={{ color: theme.text }}>
                                    {b.name || b.displayName || b.bindingName || 'Primary Datasource (PostgreSQL)'}
                                  </Typography>
                                  {isDefault && (
                                    <Chip label="Default" size="small" sx={{ height: 16, fontSize: 9, bgcolor: theme.border }} />
                                  )}
                                </Stack>
                                <Typography variant="caption" sx={{ color: theme.textMuted }}>
                                  {b.database || b.driverTable || b.datasourceId || 'Semantic catalog binding table'}
                                </Typography>
                              </Box>
                            </Box>

                            {isSelected && (
                              <Chip
                                icon={<CheckCircleIcon sx={{ fontSize: 13 }} />}
                                size="small"
                                label="Selected"
                                color="primary"
                                sx={{ height: 20, fontSize: 10, fontWeight: 700 }}
                              />
                            )}
                          </Paper>
                        );
                      })}
                    </Stack>
                  )}
                </Box>

                {/* Related Objects Section */}
                {availableRelatedBOs.length > 0 && (
                  <Box>
                    <Stack direction="row" alignItems="center" justifyContent="space-between" sx={{ mb: 1 }}>
                      <Typography variant="subtitle2" fontWeight={700} sx={{ color: theme.text }}>
                        Related Business Objects (Multi-BO Joins)
                      </Typography>
                      <Chip
                        label={`${selectedRelatedBOs.length} Joined`}
                        size="small"
                        color={selectedRelatedBOs.length > 0 ? 'primary' : 'default'}
                        sx={{ height: 20, fontSize: 10, fontWeight: 700 }}
                      />
                    </Stack>
                    <Stack spacing={1}>
                      {availableRelatedBOs.map((rel) => {
                        const relName = rel.boName || rel.targetBO || rel.name;
                        const isIncluded = selectedRelatedBOs.includes(relName);
                        return (
                          <Paper
                            key={relName}
                            onClick={() => {
                              setSelectedRelatedBOs((prev) =>
                                prev.includes(relName) ? prev.filter((x) => x !== relName) : [...prev, relName]
                              );
                            }}
                            sx={{
                              p: 1.25,
                              cursor: 'pointer',
                              borderRadius: 2,
                              border: `1.5px solid ${isIncluded ? theme.accent : theme.border}`,
                              bgcolor: isIncluded
                                ? (theme.isDark ? 'rgba(59,130,246,0.1)' : 'rgba(59,130,246,0.05)')
                                : theme.background,
                              display: 'flex',
                              alignItems: 'center',
                              justifyContent: 'space-between',
                              gap: 1,
                              transition: 'all 0.15s',
                              '&:hover': { borderColor: theme.accent },
                            }}
                          >
                            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                              <Radio
                                size="small"
                                checked={isIncluded}
                                sx={{ p: 0, color: theme.textMuted, '&.Mui-checked': { color: theme.accent } }}
                              />
                              <Box>
                                <Typography variant="body2" fontWeight={700} sx={{ color: theme.text }}>
                                  {relName}
                                </Typography>
                                <Typography variant="caption" sx={{ color: theme.textMuted }}>
                                  Relationship: <code>{rel.edge || 'RELATED'}</code> {rel.description ? `• ${rel.description}` : ''}
                                </Typography>
                              </Box>
                            </Box>
                            <Chip
                              label={isIncluded ? 'Included in Query' : 'Optional Join'}
                              size="small"
                              color={isIncluded ? 'primary' : 'default'}
                              variant={isIncluded ? 'filled' : 'outlined'}
                              sx={{ height: 20, fontSize: 10, fontWeight: 600 }}
                            />
                          </Paper>
                        );
                      })}
                    </Stack>
                  </Box>
                )}

                {/* Immutability Notice */}
                <Alert severity="info" sx={{ borderRadius: 2, fontSize: '0.78rem' }}>
                  <strong>Immutability Guarantee:</strong> Once created, this {context === 'report' ? 'report' : 'query tab'} cannot be switched to another Business Object or binding.
                </Alert>
              </Stack>
            ) : (
              <Box sx={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                <Typography variant="body2" sx={{ color: theme.textMuted }}>
                  Select a Business Object on the left to configure binding.
                </Typography>
              </Box>
            )}
          </Box>
        </Box>

        {/* Modal Footer */}
        <Box
          sx={{
            px: 3,
            py: 1.8,
            borderTop: `1px solid ${theme.border}`,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'flex-end',
            gap: 1.5,
            bgcolor: theme.background,
          }}
        >
          <Button onClick={onClose} sx={{ textTransform: 'none', color: theme.textMuted }}>
            Cancel
          </Button>
          <Button
            variant="contained"
            color="primary"
            onClick={handleConfirm}
            disabled={!selectedBO || (allowMultiSubtype && Boolean(selectedBO?.subtypes) && Object.keys(selectedBO.subtypes).length > 0 && selectedSubtypeKeys.length === 0)}
            sx={{ textTransform: 'none', fontWeight: 700, px: 3, borderRadius: 2 }}
          >
            {context === 'page' ? 'Confirm & Create Page' : context === 'report' ? 'Confirm & Create Report' : 'Confirm & Create Query Tab'}
          </Button>
        </Box>
      </Paper>
    </Box>
  );
};

export default UnifiedBOPickerModal;
