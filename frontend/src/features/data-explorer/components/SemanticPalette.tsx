import React, { useMemo, useState } from 'react';
import {
  Box,
  Typography,
  Paper,
  TextField,
  InputAdornment,
  Collapse,
  IconButton,
  Tooltip,
  Stack,
  Chip,
  Accordion,
  AccordionSummary,
  AccordionDetails,
} from '@mui/material';
import {
  Search as SearchIcon,
  ExpandMore as ExpandMoreIcon,
  Numbers as NumberIcon,
  CalendarToday as DateIcon,
  Abc as StringIcon,
  ToggleOn as BoolIcon,
  FilterAlt as FilterIcon,
  Add as AddIcon,
  Functions as FunctionsIcon,
  Layers as LayersIcon,
  Category as CategoryIcon,
  AccountTree as RelatedIcon,
} from '@mui/icons-material';
import { CoreIcon, CustomIcon } from '../../../components/common/CoreCustomIcons';
import type {
  ExplorerField,
  ExplorerSource,
  ExplorerQueryState,
  FieldCategory,
  FilterSelection,
} from '../types/dataExplorerTypes';
import { useExplorerTheme } from '../hooks/useExplorerTheme';

const CATEGORY_LABELS: Record<FieldCategory, string> = {
  dimension: 'Dimensions',
  measure: 'Measures',
  time: 'Time',
};

const CATEGORY_ORDER: FieldCategory[] = ['dimension', 'measure', 'time'];

interface SemanticPaletteProps {
  source: ExplorerSource;
  state: ExplorerQueryState;
  onToggleDimension: (fieldId: string) => void;
  onToggleMeasure: (fieldId: string, agg: ExplorerField['defaultAggregation'] | undefined) => void;
  onAddTimeDimension: (fieldId: string) => void;
  onOpenFilterModal: (fieldId?: string) => void;
  onOpenCalculationModal?: () => void;
  onRemoveCalculation?: (calcId: string) => void;
  onSelectRelatedBO?: (boName: string) => void;
}

function isSelectedAs(
  fieldId: string,
  category: FieldCategory,
  state: ExplorerQueryState
): boolean {
  if (category === 'dimension') {
    return (
      state.dimensions.some((d) => d.fieldId === fieldId) ||
      state.timeDimensions.some((t) => t.fieldId === fieldId)
    );
  }
  if (category === 'measure') {
    return state.measures.some((m) => m.fieldId === fieldId);
  }
  return state.timeDimensions.some((t) => t.fieldId === fieldId);
}

function FieldTypeIcon({ type, theme }: { type: ExplorerField['type']; theme: ReturnType<typeof useExplorerTheme> }) {
  const typeColors: Record<string, string> = {
    string: theme.info,
    number: theme.warning,
    date: '#a855f7',
    boolean: theme.success,
    unknown: theme.textMuted,
  };
  const color = typeColors[type] ?? theme.textMuted;

  const icons: Record<string, React.ElementType> = {
    string: StringIcon,
    number: NumberIcon,
    date: DateIcon,
    boolean: BoolIcon,
    unknown: StringIcon,
  };
  const Icon = icons[type] ?? StringIcon;

  return <Icon sx={{ fontSize: 18, color }} />;
}

export const SemanticPalette: React.FC<SemanticPaletteProps> = ({
  source,
  state,
  onToggleDimension,
  onToggleMeasure,
  onAddTimeDimension,
  onOpenFilterModal,
  onOpenCalculationModal,
  onRemoveCalculation,
  onSelectRelatedBO,
}) => {
  const theme = useExplorerTheme();
  const [search, setSearch] = useState('');
  const [expandedAccordion, setExpandedAccordion] = useState<string | false>('base');

  const handleAccordionChange = (panel: string) => (_: React.SyntheticEvent, isExpanded: boolean) => {
    setExpandedAccordion(isExpanded ? panel : false);
  };

  const subtypes = useMemo(() => source.subtypes || {}, [source]);
  const subtypeKeys = useMemo(() => Object.keys(subtypes), [subtypes]);
  const hasSubtypes = subtypeKeys.length > 0;
  const relatedBOs = useMemo(() => source.relatedBOs || [], [source]);

  // Group fields into base and subtype groups
  const groupedFields = useMemo(() => {
    const q = search.trim().toLowerCase();
    const filterFn = (f: ExplorerField) =>
      !q ||
      f.displayName.toLowerCase().includes(q) ||
      f.name.toLowerCase().includes(q) ||
      (f.technicalName && f.technicalName.toLowerCase().includes(q));

    const baseFields = source.fields.filter((f) => (f._scope || 'root') === 'root');
    const filteredBase = baseFields.filter(filterFn);

    const subtypeGroups: Record<string, ExplorerField[]> = {};
    if (hasSubtypes) {
      subtypeKeys.forEach((key) => {
        const stDef = subtypes[key];
        const fields = (stDef?.fields && stDef.fields.length > 0)
          ? stDef.fields
          : source.fields.filter((f) => f._scope === 'subtype' && f._subtypeKey === key);
        subtypeGroups[key] = fields.filter(filterFn);
      });
    }

    return {
      base: filteredBase,
      subtypeGroups,
    };
  }, [source.fields, subtypes, subtypeKeys, hasSubtypes, search]);

  const filterCountByField = useMemo(() => {
    const map = new Map<string, number>();
    state.filters.forEach((f: FilterSelection) => {
      map.set(f.fieldId, (map.get(f.fieldId) ?? 0) + 1);
    });
    return map;
  }, [state.filters]);

  const renderFieldItem = (field: ExplorerField) => {
    const selected = isSelectedAs(field.id, field.category, state);
    const filterCount = filterCountByField.get(field.id) ?? 0;

    return (
      <Box
        key={field.id}
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: 1.5,
          p: 1,
          borderRadius: 2,
          cursor: 'pointer',
          bgcolor: selected ? theme.accentMuted : 'transparent',
          border: selected
            ? `1px solid ${theme.accent}`
            : '1px solid transparent',
          transition: 'background 0.2s, border 0.2s',
          '&:hover': {
            bgcolor: selected
              ? theme.accentMuted
              : theme.backgroundElevated,
          },
        }}
        onClick={() => {
          if (field.category === 'measure') onToggleMeasure(field.id, field.defaultAggregation);
          else if (field.category === 'time') onAddTimeDimension(field.id);
          else onToggleDimension(field.id);
        }}
      >
        <FieldTypeIcon type={field.type} theme={theme} />
        <Typography
          variant="body2"
          fontWeight={500}
          sx={{ flex: 1, color: theme.text, minWidth: 0 }}
          noWrap
          title={field.displayName}
        >
          {field.displayName}
        </Typography>

        {field.category === 'measure' && (
          <Chip
            size="small"
            label={field.defaultAggregation || 'SUM'}
            sx={{
              height: 16,
              fontSize: 8.5,
              fontWeight: 700,
              bgcolor: 'rgba(245, 158, 11, 0.15)',
              color: theme.warning,
            }}
          />
        )}

        {/* Right-justified core/custom icon (managed centrally in CoreCustomIcons.tsx) */}
        <Box sx={{ display: 'inline-flex', alignItems: 'center', flexShrink: 0, ml: 0.5 }}>
          {field.isCustom ? (
            <CustomIcon fontSize="small" sx={{ fontSize: 16 }} />
          ) : (
            <CoreIcon fontSize="small" sx={{ fontSize: 16 }} />
          )}
        </Box>

        {filterCount > 0 && (
          <Chip
            size="small"
            label={filterCount}
            sx={{
              height: 18,
              fontSize: 10,
              fontWeight: 700,
              bgcolor: theme.accent,
              color: theme.isDark ? theme.background : '#FFFFFF',
            }}
          />
        )}

        <Stack
          direction="row"
          spacing={0.5}
          sx={{ opacity: 0, transition: 'opacity 0.2s', '.MuiBox-root:hover &': { opacity: 1 } }}
        >
          <Tooltip title="Add as filter">
            <IconButton
              size="small"
              onClick={(e) => {
                e.stopPropagation();
                onOpenFilterModal(field.id);
              }}
              sx={{ p: 0.5 }}
            >
              <FilterIcon sx={{ fontSize: 16, color: theme.textMuted }} />
            </IconButton>
          </Tooltip>
          <Tooltip title="Add to query">
            <IconButton
              size="small"
              onClick={(e) => {
                e.stopPropagation();
                if (field.category === 'measure') onToggleMeasure(field.id, field.defaultAggregation);
                else if (field.category === 'time') onAddTimeDimension(field.id);
                else onToggleDimension(field.id);
              }}
              sx={{ p: 0.5 }}
            >
              <AddIcon sx={{ fontSize: 16, color: theme.textMuted }} />
            </IconButton>
          </Tooltip>
        </Stack>
      </Box>
    );
  };

  return (
    <Paper
      elevation={0}
      sx={{
        width: 320,
        borderRight: `1px solid ${theme.border}`,
        bgcolor: theme.background,
        display: 'flex',
        flexDirection: 'column',
        overflow: 'hidden',
        flexShrink: 0,
      }}
    >
      <Box sx={{ p: 2, borderBottom: `1px solid ${theme.border}` }}>
        <Typography
          variant="caption"
          sx={{
            color: theme.textMuted,
            fontWeight: 700,
            letterSpacing: 1,
            textTransform: 'uppercase',
            display: 'block',
            mb: 1,
          }}
        >
          {source.displayName}
        </Typography>
        <TextField
          placeholder="Search fields..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          size="small"
          fullWidth
          sx={{
            '& .MuiOutlinedInput-root': {
              borderRadius: 3,
              bgcolor: theme.backgroundElevated,
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
      </Box>

      <Box sx={{ flex: 1, overflow: 'auto', p: 1.5 }}>
        {/* Base Fields Accordion */}
        <Accordion
          expanded={expandedAccordion === 'base'}
          onChange={handleAccordionChange('base')}
          defaultExpanded
          disableGutters
          sx={{
            '&::before': { display: 'none' },
            bgcolor: 'transparent',
            boxShadow: 'none',
            mb: 1,
          }}
        >
          <AccordionSummary
            expandIcon={<ExpandMoreIcon sx={{ fontSize: 18, color: theme.textMuted }} />}
            sx={{
              minHeight: 36,
              px: 1,
              '& .MuiAccordionSummary-content': { my: 0.5, alignItems: 'center', gap: 0.75 },
            }}
          >
            <LayersIcon sx={{ fontSize: 18, color: theme.accent }} />
            <Typography sx={{ fontSize: '0.85rem', fontWeight: 700, color: theme.text }}>
              Base Fields
            </Typography>
            <Box
              component="span"
              sx={{
                ml: 'auto',
                mr: 1,
                minWidth: 20,
                height: 20,
                borderRadius: '10px',
                bgcolor: theme.accentMuted,
                color: theme.accent,
                border: `1px solid ${theme.border}`,
                display: 'inline-flex',
                alignItems: 'center',
                justifyContent: 'center',
                fontSize: '0.65rem',
                fontWeight: 700,
                px: 0.75,
              }}
            >
              {groupedFields.base.length}
            </Box>
          </AccordionSummary>
          <AccordionDetails sx={{ pt: 0, pb: 1, px: 0.5 }}>
            {groupedFields.base.length === 0 ? (
              <Typography variant="caption" sx={{ color: theme.textMuted, pl: 1 }}>
                No base fields
              </Typography>
            ) : (
              <Stack spacing={0.5}>
                {groupedFields.base.map((field) => renderFieldItem(field))}
              </Stack>
            )}
          </AccordionDetails>
        </Accordion>

        {/* Subtype Accordions */}
        {subtypeKeys.map((key, idx) => {
          const st = subtypes[key];
          const fields = groupedFields.subtypeGroups[key] || [];

          return (
            <Accordion
              key={key}
              expanded={expandedAccordion === key}
              onChange={handleAccordionChange(key)}
              defaultExpanded={idx === 0}
              disableGutters
              sx={{
                '&::before': { display: 'none' },
                bgcolor: 'transparent',
                boxShadow: 'none',
                mb: 1,
              }}
            >
              <AccordionSummary
                expandIcon={<ExpandMoreIcon sx={{ fontSize: 18, color: theme.textMuted }} />}
                sx={{
                  minHeight: 36,
                  px: 1,
                  '& .MuiAccordionSummary-content': { my: 0.5, alignItems: 'center', gap: 0.75 },
                }}
              >
                <CategoryIcon sx={{ fontSize: 18, color: theme.info }} />
                <Typography sx={{ fontSize: '0.85rem', fontWeight: 700, color: theme.text }}>
                  {st.displayName || key}
                </Typography>
                <Box
                  component="span"
                  sx={{
                    ml: 'auto',
                    mr: 1,
                    minWidth: 20,
                    height: 20,
                    borderRadius: '10px',
                    bgcolor: 'rgba(59, 130, 246, 0.15)',
                    color: theme.info,
                    border: '1px solid rgba(59, 130, 246, 0.3)',
                    display: 'inline-flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    fontSize: '0.65rem',
                    fontWeight: 700,
                    px: 0.75,
                  }}
                >
                  {fields.length}
                </Box>
              </AccordionSummary>
              <AccordionDetails sx={{ pt: 0, pb: 1, px: 0.5 }}>
                {fields.length === 0 ? (
                  <Typography variant="caption" sx={{ color: theme.textMuted, pl: 1 }}>
                    No fields declared on this subtype
                  </Typography>
                ) : (
                  <Stack spacing={0.5}>
                    {fields.map((field) => renderFieldItem(field))}
                  </Stack>
                )}
              </AccordionDetails>
            </Accordion>
          );
        })}

        {/* Related Business Objects Accordion */}
        {relatedBOs.length > 0 && (
          <Accordion
            expanded={expandedAccordion === 'related'}
            onChange={handleAccordionChange('related')}
            disableGutters
            sx={{
              '&::before': { display: 'none' },
              bgcolor: 'transparent',
              boxShadow: 'none',
              mb: 1,
            }}
          >
            <AccordionSummary
              expandIcon={<ExpandMoreIcon sx={{ fontSize: 18, color: theme.textMuted }} />}
              sx={{
                minHeight: 36,
                px: 1,
                '& .MuiAccordionSummary-content': { my: 0.5, alignItems: 'center', gap: 0.75 },
              }}
            >
              <RelatedIcon sx={{ fontSize: 18, color: '#a855f7' }} />
              <Typography sx={{ fontSize: '0.85rem', fontWeight: 700, color: theme.text }}>
                Related Objects
              </Typography>
              <Box
                component="span"
                sx={{
                  ml: 'auto',
                  mr: 1,
                  minWidth: 20,
                  height: 20,
                  borderRadius: '10px',
                  bgcolor: 'rgba(168, 85, 247, 0.15)',
                  color: '#a855f7',
                  border: '1px solid rgba(168, 85, 247, 0.3)',
                  display: 'inline-flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontSize: '0.65rem',
                  fontWeight: 700,
                  px: 0.75,
                }}
              >
                {relatedBOs.length}
              </Box>
            </AccordionSummary>
            <AccordionDetails sx={{ pt: 0, pb: 1, px: 0.5 }}>
              <Stack spacing={0.5}>
                {relatedBOs.map((r, idx) => (
                  <Box
                    key={`${r.boName}-${idx}`}
                    onClick={() => onSelectRelatedBO?.(r.boName)}
                    sx={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: 1.5,
                      p: 1,
                      borderRadius: 2,
                      cursor: onSelectRelatedBO ? 'pointer' : 'default',
                      bgcolor: 'transparent',
                      border: `1px solid ${theme.border}`,
                      '&:hover': onSelectRelatedBO ? { bgcolor: theme.backgroundElevated } : {},
                    }}
                  >
                    <RelatedIcon sx={{ fontSize: 16, color: '#a855f7' }} />
                    <Box sx={{ flex: 1, minWidth: 0 }}>
                      <Typography variant="body2" fontWeight={600} sx={{ color: theme.text }} noWrap>
                        {r.boName}
                      </Typography>
                      {r.edge && (
                        <Typography variant="caption" sx={{ color: theme.textMuted }}>
                          {r.edge}
                        </Typography>
                      )}
                    </Box>
                  </Box>
                ))}
              </Stack>
            </AccordionDetails>
          </Accordion>
        )}

        {/* Calculations Section */}
        <Box sx={{ mb: 2, mt: 1.5, px: 0.5 }}>
          <Box
            sx={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              px: 0.5,
              mb: 1,
            }}
          >
            <Typography
              variant="caption"
              fontWeight={700}
              sx={{
                textTransform: 'uppercase',
                letterSpacing: 1,
                color: theme.accent,
                display: 'flex',
                alignItems: 'center',
                gap: 0.5,
              }}
            >
              <FunctionsIcon sx={{ fontSize: 14 }} /> Calculations ({state.calculations.length})
            </Typography>
            {onOpenCalculationModal && (
              <IconButton
                size="small"
                onClick={onOpenCalculationModal}
                sx={{
                  p: 0.3,
                  bgcolor: theme.backgroundElevated,
                  border: `1px solid ${theme.border}`,
                  color: theme.accent,
                }}
              >
                <AddIcon sx={{ fontSize: 13 }} />
              </IconButton>
            )}
          </Box>
          <Stack spacing={0.5}>
            {state.calculations.map((calc) => (
              <Box
                key={calc.id}
                sx={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 1.5,
                  p: 1,
                  borderRadius: 2,
                  bgcolor: theme.accentMuted,
                  border: `1px solid ${theme.border}`,
                }}
              >
                <FunctionsIcon sx={{ fontSize: 16, color: theme.accent }} />
                <Typography
                  variant="body2"
                  fontWeight={600}
                  sx={{ flex: 1, color: theme.text, minWidth: 0 }}
                  noWrap
                  title={calc.displayName}
                >
                  {calc.displayName}
                </Typography>
                {onRemoveCalculation && (
                  <IconButton
                    size="small"
                    onClick={() => onRemoveCalculation(calc.id)}
                    sx={{ p: 0.2, color: theme.error }}
                  >
                    <AddIcon sx={{ fontSize: 14, transform: 'rotate(45deg)' }} />
                  </IconButton>
                )}
              </Box>
            ))}
            {state.calculations.length === 0 && (
              <Box
                onClick={onOpenCalculationModal}
                sx={{
                  p: 1,
                  textAlign: 'center',
                  border: `1px dashed ${theme.border}`,
                  borderRadius: 2,
                  cursor: 'pointer',
                  '&:hover': { bgcolor: theme.backgroundElevated },
                }}
              >
                <Typography variant="caption" sx={{ color: theme.textMuted, fontWeight: 600 }}>
                  + Create custom formula
                </Typography>
              </Box>
            )}
          </Stack>
        </Box>
      </Box>
    </Paper>
  );
};

export default SemanticPalette;
