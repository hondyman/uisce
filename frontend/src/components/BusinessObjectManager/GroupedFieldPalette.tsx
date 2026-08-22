import React, { useState, useEffect } from 'react';
import { 
  Box, Typography, Chip, Accordion, AccordionSummary, AccordionDetails, 
  IconButton, Tooltip, TextField, InputAdornment, Button, Stack, CircularProgress
} from '@mui/material';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import SearchIcon from '@mui/icons-material/Search';
import ShieldIcon from '@mui/icons-material/Shield';
import TrendingUpIcon from '@mui/icons-material/TrendingUp';
import LandmarkIcon from '@mui/icons-material/AccountBalance';
import ZapIcon from '@mui/icons-material/Bolt';
import FunctionsIcon from '@mui/icons-material/Functions';
import FilterListIcon from '@mui/icons-material/FilterList';
import CalendarTodayIcon from '@mui/icons-material/CalendarToday';
import AddIcon from '@mui/icons-material/Add';
import { fetchAPI } from '../../api';

export interface GroupedField {
  fieldId: string;
  key: string;
  displayName: string;
  role: 'IDENTIFIER' | 'DIMENSION' | 'MEASURE' | 'CALCULATION' | string;
  dataType: string;
  subtypeScope: string;
  isRequired: boolean;
  isGoverned: boolean;
  formula?: string;
}

export interface FieldGroup {
  groupKey: string;
  groupName: string;
  sequence: number;
  subtypeCode?: string;
  fields: GroupedField[];
}

export interface BOLayoutResponse {
  boId: string;
  boKey: string;
  discriminatorField?: string;
  subtypes?: Record<string, { displayName: string; color: string; icon: string; description?: string }>;
  activeSubtype: string;
  groups: FieldGroup[];
  totalFields: number;
}

interface Props {
  boId: string;
  tenantId?: string;
  subtypesConfig?: Record<string, { displayName: string; color: string; icon: string }>;
  onSelectField?: (field: GroupedField) => void;
  onAddDimension?: (field: GroupedField) => void;
  onAddMeasure?: (field: GroupedField) => void;
  onAddTimeDimension?: (field: GroupedField) => void;
  onAddFilter?: (field: GroupedField) => void;
  isFieldSelected?: (fieldId: string, role?: string) => boolean;
}

export const GroupedFieldPalette: React.FC<Props> = ({ 
  boId, 
  tenantId, 
  subtypesConfig: propSubtypesConfig, 
  onSelectField,
  onAddDimension,
  onAddMeasure,
  onAddTimeDimension,
  onAddFilter,
  isFieldSelected
}) => {
  const [selectedSubtype, setSelectedSubtype] = useState<string>('ALL');
  const [searchQuery, setSearchQuery] = useState<string>('');
  const [groups, setGroups] = useState<FieldGroup[]>([]);
  const [layoutSubtypes, setLayoutSubtypes] = useState<Record<string, { displayName: string; color: string; icon: string }> | undefined>(propSubtypesConfig);
  const [loading, setLoading] = useState<boolean>(true);

  useEffect(() => {
    if (!boId) return;
    let isMounted = true;
    const fetchLayout = async () => {
      setLoading(true);
      try {
        const data = await fetchAPI<BOLayoutResponse>(`/business-objects/${encodeURIComponent(boId)}/layout?subtype=${encodeURIComponent(selectedSubtype)}`);
        if (isMounted && data) {
          setGroups(data.groups || []);
          if (data.subtypes && Object.keys(data.subtypes).length > 0) {
            setLayoutSubtypes(data.subtypes);
          }
        }
      } catch (err) {
        console.error('Failed to load BO grouped layout', err);
      } finally {
        if (isMounted) setLoading(false);
      }
    };

    fetchLayout();
    return () => { isMounted = false; };
  }, [boId, selectedSubtype]);

  const effectiveSubtypes = propSubtypesConfig || layoutSubtypes;

  const getSubtypeIcon = (iconName?: string) => {
    switch (iconName?.toLowerCase()) {
      case 'trending-up': return <TrendingUpIcon sx={{ fontSize: 14 }} />;
      case 'landmark': return <LandmarkIcon sx={{ fontSize: 14 }} />;
      case 'zap': return <ZapIcon sx={{ fontSize: 14 }} />;
      default: return <ShieldIcon sx={{ fontSize: 14 }} />;
    }
  };

  const isDateType = (dt: string) => {
    const l = (dt || '').toLowerCase();
    return l.includes('date') || l.includes('time');
  };

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%', gap: 1.25, minHeight: 0 }}>
      {/* 1. Subtype Selector Pills (Polymorphic Filter) */}
      {effectiveSubtypes && Object.keys(effectiveSubtypes).length > 0 && (
        <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5, pb: 1, borderBottom: '1px solid', borderColor: 'divider' }}>
          <Chip
            label="All Fields"
            size="small"
            clickable
            variant={selectedSubtype === 'ALL' ? 'filled' : 'outlined'}
            color={selectedSubtype === 'ALL' ? 'primary' : 'default'}
            onClick={() => setSelectedSubtype('ALL')}
            sx={{ fontSize: '0.72rem', fontWeight: 700, height: 24 }}
          />
          {Object.entries(effectiveSubtypes).map(([key, cfg]) => {
            const isSelected = selectedSubtype === key;
            const chipColor = cfg.color || '#6366f1';
            return (
              <Chip
                key={key}
                icon={getSubtypeIcon(cfg.icon)}
                label={cfg.displayName}
                size="small"
                clickable
                variant={isSelected ? 'filled' : 'outlined'}
                onClick={() => setSelectedSubtype(key)}
                sx={{
                  fontSize: '0.72rem',
                  fontWeight: 700,
                  height: 24,
                  borderColor: chipColor,
                  color: isSelected ? '#fff' : chipColor,
                  backgroundColor: isSelected ? chipColor : 'transparent',
                  '&:hover': { backgroundColor: chipColor, color: '#fff' }
                }}
              />
            );
          })}
        </Box>
      )}

      {/* 2. Fast Field Search */}
      <TextField
        placeholder="Search fields, terms, metrics..."
        size="small"
        fullWidth
        value={searchQuery}
        onChange={(e) => setSearchQuery(e.target.value)}
        InputProps={{
          startAdornment: (
            <InputAdornment position="start">
              <SearchIcon fontSize="small" sx={{ color: 'text.secondary' }} />
            </InputAdornment>
          ),
          sx: { fontSize: '0.78rem' }
        }}
      />

      {/* 3. Taxonomy-Grouped Accordions */}
      <Box sx={{ flex: 1, overflowY: 'auto', pr: 0.5, minHeight: 0 }}>
        {loading ? (
          <Box sx={{ display: 'flex', justifyContent: 'center', py: 4 }}>
            <CircularProgress size={24} />
          </Box>
        ) : groups.length === 0 ? (
          <Typography variant="caption" color="text.secondary" sx={{ display: 'block', textAlign: 'center', py: 3, fontStyle: 'italic' }}>
            No taxonomy fields found for this facet.
          </Typography>
        ) : (
          groups.map((group) => {
            const filteredFields = group.fields.filter(f => 
              (f.displayName || '').toLowerCase().includes(searchQuery.toLowerCase()) ||
              (f.key || '').toLowerCase().includes(searchQuery.toLowerCase()) ||
              (f.role || '').toLowerCase().includes(searchQuery.toLowerCase())
            );

            if (filteredFields.length === 0) return null;

            return (
              <Accordion 
                key={group.groupKey} 
                defaultExpanded 
                disableGutters
                sx={{ 
                  mb: 1, 
                  border: '1px solid',
                  borderColor: 'divider',
                  borderRadius: '6px !important',
                  '&:before': { display: 'none' },
                  bgcolor: 'background.paper',
                }}
              >
                <AccordionSummary 
                  expandIcon={<ExpandMoreIcon sx={{ fontSize: 18 }} />}
                  sx={{ minHeight: 34, py: 0, '& .MuiAccordionSummary-content': { my: 0.5, alignItems: 'center', gap: 1 } }}
                >
                  <Typography sx={{ fontSize: '0.78rem', fontWeight: 700, color: 'text.primary', flex: 1 }} noWrap>
                    {group.groupName}
                  </Typography>
                  <Chip 
                    label={filteredFields.length} 
                    size="small" 
                    sx={{ height: 18, fontSize: '0.65rem', fontWeight: 700 }} 
                  />
                </AccordionSummary>

                <AccordionDetails sx={{ p: 0.75, pt: 0 }}>
                  <Stack spacing={0.5}>
                    {filteredFields.map((field) => {
                      const isDate = isDateType(field.dataType);
                      const isMeas = field.role === 'MEASURE' || ['number', 'numeric', 'float', 'decimal', 'integer'].includes((field.dataType || '').toLowerCase());
                      const isDimSelected = isFieldSelected ? isFieldSelected(field.fieldId, 'DIM') : false;
                      const isMeasSelected = isFieldSelected ? isFieldSelected(field.fieldId, 'MEAS') : false;

                      return (
                        <Box
                          key={field.fieldId}
                          sx={{
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'space-between',
                            p: '4px 6px',
                            borderRadius: '4px',
                            bgcolor: 'background.default',
                            border: '1px solid',
                            borderColor: 'divider',
                            '&:hover': { bgcolor: 'action.hover' }
                          }}
                        >
                          <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75, minWidth: 0, flex: 1 }}>
                            {field.isGoverned ? (
                              <Tooltip title={`Governed Formula: ${field.formula}`}>
                                <FunctionsIcon sx={{ fontSize: 16, color: 'secondary.main' }} />
                              </Tooltip>
                            ) : isMeas ? (
                              <Box sx={{ width: 7, height: 7, borderRadius: '50%', bgcolor: 'success.main', flexShrink: 0 }} />
                            ) : isDate ? (
                              <Box sx={{ width: 7, height: 7, borderRadius: '50%', bgcolor: 'warning.main', flexShrink: 0 }} />
                            ) : (
                              <Box sx={{ width: 7, height: 7, borderRadius: '50%', bgcolor: 'primary.main', flexShrink: 0 }} />
                            )}

                            <Typography 
                              noWrap 
                              sx={{ 
                                fontSize: '0.74rem', 
                                fontWeight: field.isRequired ? 700 : 500,
                                color: 'text.primary',
                              }}
                              title={`${field.displayName} (${field.key})`}
                            >
                              {field.displayName}
                            </Typography>
                          </Box>

                          {/* Quick Projection Triggers */}
                          <Stack direction="row" spacing={0.5} alignItems="center">
                            {onAddDimension && (
                              <Tooltip title="Add Dimension">
                                <Button
                                  size="small"
                                  variant={isDimSelected ? 'contained' : 'outlined'}
                                  color="primary"
                                  onClick={() => onAddDimension(field)}
                                  sx={{ minWidth: 26, px: 0.6, py: 0.1, fontSize: '0.62rem', height: 20 }}
                                >
                                  Dim
                                </Button>
                              </Tooltip>
                            )}

                            {onAddMeasure && (
                              <Tooltip title="Add Measure">
                                <Button
                                  size="small"
                                  variant={isMeasSelected ? 'contained' : 'outlined'}
                                  color="success"
                                  onClick={() => onAddMeasure(field)}
                                  sx={{ minWidth: 26, px: 0.6, py: 0.1, fontSize: '0.62rem', height: 20 }}
                                >
                                  Agg
                                </Button>
                              </Tooltip>
                            )}

                            {isDate && onAddTimeDimension && (
                              <Tooltip title="Add Time Dimension">
                                <Button
                                  size="small"
                                  variant="outlined"
                                  color="warning"
                                  onClick={() => onAddTimeDimension(field)}
                                  sx={{ minWidth: 26, px: 0.6, py: 0.1, fontSize: '0.62rem', height: 20 }}
                                >
                                  Time
                                </Button>
                              </Tooltip>
                            )}

                            {onAddFilter && (
                              <Tooltip title="Add Filter">
                                <IconButton size="small" onClick={() => onAddFilter(field)} sx={{ p: 0.25 }}>
                                  <FilterListIcon sx={{ fontSize: 14 }} />
                                </IconButton>
                              </Tooltip>
                            )}
                          </Stack>
                        </Box>
                      );
                    })}
                  </Stack>
                </AccordionDetails>
              </Accordion>
            );
          })
        )}
      </Box>
    </Box>
  );
};
