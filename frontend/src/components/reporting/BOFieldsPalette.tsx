import React, { useState, useMemo } from 'react';
import {
  Box,
  Typography,
  TextField,
  InputAdornment,
  Button,
  Tooltip,
  IconButton,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Chip,
} from '@mui/material';
import { useDraggable } from '@dnd-kit/core';
import SearchIcon from '@mui/icons-material/Search';
import TableChartIcon from '@mui/icons-material/TableChart';
import AddIcon from '@mui/icons-material/Add';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import LayersIcon from '@mui/icons-material/Layers';
import CategoryIcon from '@mui/icons-material/Category';
import AbcIcon from '@mui/icons-material/Abc';
import NumbersIcon from '@mui/icons-material/Numbers';
import CalendarTodayIcon from '@mui/icons-material/CalendarToday';
import ToggleOnIcon from '@mui/icons-material/ToggleOn';
import FingerprintIcon from '@mui/icons-material/Fingerprint';
import DataObjectIcon from '@mui/icons-material/DataObject';
import { CoreIcon, CustomIcon } from '../common/CoreCustomIcons';

export interface BOField {
  name: string;
  technicalName?: string;
  label?: string;
  dataType?: string;
  type?: string;
  description?: string;
  isCore?: boolean;
  _scope?: 'root' | 'subtype';
  _subtypeKey?: string;
}

export const getTypeIconConfig = (type: string = 'string') => {
  const t = (type || '').toLowerCase();
  if (['number', 'int', 'float', 'double', 'decimal', 'numeric', 'currency', 'money'].some((k) => t.includes(k))) {
    return { Icon: NumbersIcon, color: 'success.main', label: 'Numeric' };
  }
  if (['date', 'time', 'timestamp', 'datetime'].some((k) => t.includes(k))) {
    return { Icon: CalendarTodayIcon, color: 'warning.main', label: 'Date/Time' };
  }
  if (['uuid', 'guid', 'key'].some((k) => t.includes(k))) {
    return { Icon: FingerprintIcon, color: 'info.main', label: 'UUID' };
  }
  if (['bool', 'boolean'].some((k) => t.includes(k))) {
    return { Icon: ToggleOnIcon, color: 'secondary.main', label: 'Boolean' };
  }
  if (['json', 'object', 'jsonb'].some((k) => t.includes(k))) {
    return { Icon: DataObjectIcon, color: 'primary.main', label: 'JSON' };
  }
  return { Icon: AbcIcon, color: 'text.secondary', label: 'String' };
};

export const extractAllBOFields = (
  bo: any,
  activeSubtypeFilter: string = 'all',
  extraFields?: any[]
): BOField[] => {
  if (!bo && (!extraFields || extraFields.length === 0)) return [];
  const fields: BOField[] = [];
  const seen = new Set<string>();

  const addField = (
    f: any,
    isCoreDefault = false,
    scope: 'root' | 'subtype' = 'root',
    subtypeKey?: string
  ) => {
    if (!f) return;
    const key = f.key || f.technicalName || f.technical_name || f.name;
    if (!key || typeof key !== 'string') return;

    if (activeSubtypeFilter === 'root' && scope !== 'root') return;
    if (
      activeSubtypeFilter !== 'all' &&
      activeSubtypeFilter !== 'root' &&
      scope === 'subtype' &&
      subtypeKey !== activeSubtypeFilter
    )
      return;

    const scopedKey = `${scope}:${subtypeKey || 'root'}:${key}`;
    if (seen.has(scopedKey)) return;
    seen.add(scopedKey);

    const isCore =
      typeof f === 'object'
        ? f.isCore ?? f.is_core ?? isCoreDefault
        : isCoreDefault;

    fields.push({
      name: key,
      technicalName: key,
      label:
        typeof f === 'object'
          ? f.displayName || f.display_name || f.businessName || f.label || f.name || key
          : key,
      dataType:
        typeof f === 'object'
          ? f.dataType || f.data_type || f.type || f.fieldType || f.field_type || 'string'
          : 'string',
      type:
        typeof f === 'object'
          ? f.dataType || f.data_type || f.type || f.fieldType || f.field_type || 'string'
          : 'string',
      description: typeof f === 'object' ? f.description || '' : '',
      isCore: !!isCore,
      _scope: scope,
      _subtypeKey: subtypeKey,
    });
  };

  (bo?.coreFields || bo?.core_fields || []).forEach((f: any) => addField(f, true, 'root'));
  (bo?.customFields || bo?.custom_fields || []).forEach((f: any) => addField(f, false, 'root'));
  (bo?.fields || []).forEach((f: any) => addField(f, false, 'root'));
  (bo?.entity_fields || []).forEach((f: any) => addField(f, false, 'root'));
  (bo?.config?.entity_fields || []).forEach((f: any) => addField(f, false, 'root'));
  (bo?.config?.fields || []).forEach((f: any) => addField(f, false, 'root'));
  (bo?.config?.inheritedFields || []).forEach((f: any) => addField(f, true, 'root'));
  (bo?.config?.customFields || []).forEach((f: any) => addField(f, false, 'root'));
  (extraFields || []).forEach((f: any) => addField(f, false, 'root'));

  if (bo?.subtypes && typeof bo.subtypes === 'object') {
    Object.entries(bo.subtypes).forEach(([stKey, subtype]: [string, any]) => {
      const stFields = subtype?.subtypeFields || subtype?.subtype_fields || subtype?.fields || [];
      (Array.isArray(stFields) ? stFields : []).forEach((f: any) =>
        addField(f, f.isCore ?? f.is_core ?? false, 'subtype', stKey)
      );
    });
  }

  return fields;
};

interface DraggableBOFieldItemProps {
  field: BOField;
  isSelected?: boolean;
  onAdd: (field: BOField) => void;
  onSelect: (fieldKey: string, isShift: boolean, isCtrl: boolean) => void;
  selectedFields?: BOField[];
  mode?: 'design' | 'form';
}

const DraggableBOFieldItem: React.FC<DraggableBOFieldItemProps> = ({
  field,
  isSelected = false,
  onAdd,
  onSelect,
  selectedFields = [],
  mode = 'design',
}) => {
  const fieldKey = field.technicalName || field.name;
  const { attributes, listeners, setNodeRef, isDragging } = useDraggable({
    id: `bofield_${fieldKey}_${field._scope || 'root'}_${field._subtypeKey || 'root'}`,
    data: {
      isBOField: true,
      field,
      selectedFields: selectedFields.length > 0 && selectedFields.some(f => (f.technicalName || f.name) === fieldKey)
        ? selectedFields
        : [field],
    },
  });

  const { Icon, color, label } = getTypeIconConfig(field.dataType || field.type);

  const bundle = selectedFields.length > 0 && selectedFields.some(f => (f.technicalName || f.name) === fieldKey)
    ? selectedFields
    : [field];

  return (
    <Box
      ref={setNodeRef}
      elevation={0}
      draggable
      onDragStart={(e) => {
        const payload =
          mode === 'form'
            ? {
                type: 'BO_FIELD',
                key: fieldKey,
                name: field.displayName || field.name,
                dataType: field.dataType || field.type,
              }
            : {
                type: bundle.length > 1 ? 'bofield_batch' : 'bofield',
                field: field,
                fields: bundle,
                fieldKey: fieldKey,
              };
        e.dataTransfer.setData('application/json', JSON.stringify(payload));
        e.dataTransfer.setData('bo-field-bundle', JSON.stringify(bundle));
        e.dataTransfer.setData('text/plain', fieldKey);
        e.dataTransfer.effectAllowed = 'copy';
      }}
      onClick={(e) => {
        onSelect(fieldKey, e.shiftKey, e.metaKey || e.ctrlKey);
      }}
      sx={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        p: 0.75,
        mb: 0.5,
        bgcolor: isSelected ? 'rgba(99, 102, 241, 0.15)' : 'rgba(255, 255, 255, 0.03)',
        border: '1px solid',
        borderColor: isSelected ? 'primary.main' : 'rgba(255, 255, 255, 0.08)',
        borderRadius: 1,
        cursor: 'grab',
        userSelect: 'none',
        opacity: isDragging ? 0.4 : 1,
        '&:hover': {
          bgcolor: isSelected ? 'rgba(99, 102, 241, 0.25)' : 'rgba(99, 102, 241, 0.08)',
          borderColor: 'primary.main',
          '& .add-btn': { opacity: 1 },
        },
      }}
      {...attributes}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, minWidth: 0, flex: 1 }}>
        {/* Checkbox / Drag-handle */}
        <Box
          component="span"
          {...listeners}
          sx={{ display: 'flex', alignItems: 'center', cursor: 'grab', flexShrink: 0 }}
          onClick={(e) => {
            e.stopPropagation();
            onSelect(fieldKey, e.shiftKey, e.metaKey || e.ctrlKey);
          }}
        >
          <Box
            sx={{
              width: 14,
              height: 14,
              borderRadius: '2px',
              border: '1px solid',
              borderColor: isSelected ? 'primary.main' : 'text.disabled',
              bgcolor: isSelected ? 'primary.main' : 'transparent',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              flexShrink: 0,
            }}
          >
            {isSelected && <Box sx={{ width: 6, height: 6, bgcolor: 'white', borderRadius: '1px' }} />}
          </Box>
        </Box>

        <Tooltip title={label} placement="top">
          <Box
            sx={{
              display: 'inline-flex',
              p: 0.25,
              borderRadius: 0.5,
              bgcolor: 'action.selected',
              flexShrink: 0,
            }}
          >
            <Icon sx={{ fontSize: 15, color }} />
          </Box>
        </Tooltip>
        <Box sx={{ minWidth: 0, flex: 1 }}>
          <Typography
            variant="body2"
            sx={{ fontSize: '0.75rem', fontWeight: isSelected ? 700 : 600, noWrap: true, color: isSelected ? 'primary.light' : 'inherit' }}
          >
            {field.label || field.name}
          </Typography>
          <Typography
            variant="caption"
            color="text.secondary"
            sx={{ display: 'block', fontSize: '0.65rem', noWrap: true }}
          >
            {field.technicalName || field.name}
          </Typography>
        </Box>

        {/* Right-justified core/custom icon (managed centrally in components/common/CoreCustomIcons.tsx) */}
        <Box sx={{ display: 'inline-flex', alignItems: 'center', flexShrink: 0, ml: 0.5 }}>
          {field.isCore ? (
            <CoreIcon fontSize="small" sx={{ fontSize: 16 }} />
          ) : (
            <CustomIcon fontSize="small" sx={{ fontSize: 16 }} />
          )}
        </Box>
      </Box>

      <Tooltip title="Add to Canvas">
        <IconButton
          className="add-btn"
          size="small"
          onClick={(e) => {
            e.stopPropagation();
            onAdd(field);
          }}
          sx={{ opacity: isSelected ? 1 : 0.6, p: 0.25, ml: 0.25, '&:hover': { color: 'primary.main' } }}
        >
          <AddIcon sx={{ fontSize: 14 }} />
        </IconButton>
      </Tooltip>
    </Box>
  );
};


interface BOFieldsPaletteProps {
  selectedBO?: any;
  relatedBOs?: any[];
  selectedSubtypeKey?: string | null;
  onAddFieldToCanvas: (field: BOField) => void;
  onAddAllAsTable: (fields: BOField[]) => void;
  width?: number;
  onResize?: (e: React.MouseEvent) => void;
  mode?: 'design' | 'form';
}

const BOFieldsPalette: React.FC<BOFieldsPaletteProps> = ({
  selectedBO,
  relatedBOs,
  selectedSubtypeKey: propSubtypeKey,
  onAddFieldToCanvas,
  onAddAllAsTable,
  onResize,
  mode = 'design',
}) => {
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedFieldKeys, setSelectedFieldKeys] = useState<string[]>([]);
  const [lastClickedIndex, setLastClickedIndex] = useState<number | null>(null);
  const [expandedAccordion, setExpandedAccordion] = useState<string | false>('base');

  const handleAccordionChange = (panel: string) => (_: React.SyntheticEvent, isExpanded: boolean) => {
    setExpandedAccordion(isExpanded ? panel : false);
  };

  const activeSubtype = propSubtypeKey ?? selectedBO?.selectedSubtypeKey ?? null;

  const subtypes = useMemo(() => {
    const rawSubtypes = selectedBO?.subtypes || {};
    if (!activeSubtype) return rawSubtypes;
    // If a subtype was chosen, only include that subtype
    if (rawSubtypes[activeSubtype]) {
      return { [activeSubtype]: rawSubtypes[activeSubtype] };
    }
    return rawSubtypes;
  }, [selectedBO, activeSubtype]);

  const hasSubtypes = Object.keys(subtypes).length > 0;

  const allFields = useMemo(
    () => extractAllBOFields(selectedBO, activeSubtype ? activeSubtype : 'all'),
    [selectedBO, activeSubtype]
  );

  const groupedFields = useMemo(() => {
    const base = allFields.filter((f) => f._scope === 'root');
    const subtypeGroups: Record<string, BOField[]> = {};
    if (hasSubtypes) {
      Object.keys(subtypes).forEach((key) => {
        subtypeGroups[key] = allFields.filter(
          (f) => f._scope === 'subtype' && f._subtypeKey === key
        );
      });
    }
    return { base, subtypeGroups };
  }, [allFields, subtypes, hasSubtypes]);

  const filteredGroup = useMemo(() => {
    const term = searchTerm.toLowerCase().trim();
    const filterFn = (f: BOField) =>
      !term ||
      (f.label || f.name).toLowerCase().includes(term) ||
      (f.technicalName && f.technicalName.toLowerCase().includes(term)) ||
      f.name.toLowerCase().includes(term);

    const base = groupedFields.base.filter(filterFn);
    const subtypeGroups: Record<string, BOField[]> = {};
    Object.entries(groupedFields.subtypeGroups).forEach(([k, arr]) => {
      subtypeGroups[k] = arr.filter(filterFn);
    });

    const totalFiltered = base.length + Object.values(subtypeGroups).reduce((sum, arr) => sum + arr.length, 0);

    return { base, subtypeGroups, totalFiltered };
  }, [groupedFields, searchTerm]);

  const selectedFields = useMemo(() => {
    return allFields.filter(f => selectedFieldKeys.includes(f.technicalName || f.name));
  }, [allFields, selectedFieldKeys]);

  if (!selectedBO) {
    return (
      <Box sx={{ p: 2, textAlign: 'center', color: 'text.secondary' }}>
        <Typography variant="caption">
          Select a Business Object in the top bar to view its palette fields.
        </Typography>
      </Box>
    );
  }

  const handleToggleFieldSelection = (fieldKey: string, isShift: boolean, isCtrl: boolean) => {
    const flatList = [...filteredGroup.base, ...Object.values(filteredGroup.subtypeGroups).flat()];
    const clickedIdx = flatList.findIndex(f => (f.technicalName || f.name) === fieldKey);

    if (isShift && lastClickedIndex !== null && clickedIdx !== -1) {
      const start = Math.min(lastClickedIndex, clickedIdx);
      const end = Math.max(lastClickedIndex, clickedIdx);
      const rangeKeys = flatList.slice(start, end + 1).map(f => f.technicalName || f.name);
      const newKeys = Array.from(new Set([...selectedFieldKeys, ...rangeKeys]));
      setSelectedFieldKeys(newKeys);
    } else if (isCtrl) {
      if (selectedFieldKeys.includes(fieldKey)) {
        setSelectedFieldKeys(selectedFieldKeys.filter(k => k !== fieldKey));
      } else {
        setSelectedFieldKeys([...selectedFieldKeys, fieldKey]);
      }
      setLastClickedIndex(clickedIdx);
    } else {
      if (selectedFieldKeys.length === 1 && selectedFieldKeys[0] === fieldKey) {
        setSelectedFieldKeys([]);
        setLastClickedIndex(null);
      } else {
        setSelectedFieldKeys([fieldKey]);
        setLastClickedIndex(clickedIdx);
      }
    }
  };

  const subtypeKeys = Object.keys(subtypes);

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%', position: 'relative' }}>
      {/* Sticky header — search + add-all */}
      <Box
        sx={{
          position: 'sticky',
          top: 0,
          zIndex: 2,
          bgcolor: 'background.paper',
          borderBottom: '1px solid',
          borderColor: 'divider',
          p: 1,
        }}
      >
        {mode === 'form' && (
          <Typography variant="caption" sx={{ color: '#00D4FF', fontSize: '0.7rem', mb: 0.5, display: 'block' }}>
            Drag fields onto a Form Section
          </Typography>
        )}
        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 0.75 }}>
          <Typography variant="caption" sx={{ fontWeight: 700, letterSpacing: '0.04em', color: 'primary.main', textTransform: 'uppercase', fontSize: '0.68rem' }}>
            Semantic Terms ({allFields.length})
          </Typography>
          {selectedFieldKeys.length > 0 && (
            <Chip
              size="small"
              label={`${selectedFieldKeys.length} selected`}
              color="primary"
              variant="outlined"
              onDelete={() => setSelectedFieldKeys([])}
              sx={{ height: 20, fontSize: '0.65rem', fontWeight: 700 }}
            />
          )}
        </Box>

        <TextField
          size="small"
          placeholder="Filter fields... (Shift/Cmd click to multi-select)"
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          fullWidth
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <SearchIcon sx={{ fontSize: 16, color: 'text.secondary' }} />
              </InputAdornment>
            ),
          }}
          sx={{ '& .MuiInputBase-input': { fontSize: '0.75rem', py: 0.5 }, mb: 1 }}
        />
        <Button
          fullWidth
          variant="outlined"
          size="small"
          startIcon={<TableChartIcon sx={{ fontSize: 16 }} />}
          onClick={() => {
            const payload = selectedFields.length > 0
              ? selectedFields
              : [...filteredGroup.base, ...Object.values(filteredGroup.subtypeGroups).flat()];
            onAddAllAsTable(payload);
          }}
          sx={{ textTransform: 'none', fontSize: '0.72rem', py: 0.5 }}
        >
          {selectedFields.length > 0
            ? `Add Selected (${selectedFields.length}) as Table`
            : `Add All as Table Grid (${filteredGroup.totalFiltered})`}
        </Button>
      </Box>

      {/* Scrollable accordion list */}
      <Box sx={{ flex: 1, overflowY: 'auto' }}>
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
          }}
        >
          <AccordionSummary
            expandIcon={<ExpandMoreIcon />}
            sx={{
              minHeight: 36,
              '& .MuiAccordionSummary-content': { my: 0.5, alignItems: 'center', gap: 0.5 },
            }}
          >
            <LayersIcon sx={{ fontSize: 18, color: 'primary.main' }} />
            <Typography sx={{ fontSize: '0.8rem', fontWeight: 600, ml: 0.5 }}>
              Base Fields
            </Typography>
            <Box
              component="span"
              sx={{
                ml: 'auto',
                mr: 1,
                minWidth: 20, height: 20, borderRadius: '10px',
                bgcolor: 'primary.main', color: 'white',
                display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
                fontSize: '0.65rem', fontWeight: 700,
                px: 0.75, lineHeight: 1,
              }}
            >
              {filteredGroup.base.length}
            </Box>
          </AccordionSummary>
          <AccordionDetails sx={{ pt: 0, pb: 1 }}>
            {filteredGroup.base.length === 0 ? (
              <Typography variant="caption" color="text.secondary" sx={{ pl: 1 }}>
                No base fields
              </Typography>
            ) : (
              filteredGroup.base.map((field) => {
                const k = field.technicalName || field.name;
                return (
                  <DraggableBOFieldItem
                    key={`base-${k}`}
                    field={field}
                    isSelected={selectedFieldKeys.includes(k)}
                    onAdd={onAddFieldToCanvas}
                    onSelect={handleToggleFieldSelection}
                    selectedFields={selectedFields}
                    mode={mode}
                  />
                );
              })
            )}
          </AccordionDetails>
        </Accordion>

        {/* One Accordion per subtype */}
        {subtypeKeys.map((key, idx) => {
          const st = subtypes[key];
          const fields = filteredGroup.subtypeGroups[key] || [];

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
              }}
            >
              <AccordionSummary
                expandIcon={<ExpandMoreIcon />}
                sx={{
                  minHeight: 36,
                  '& .MuiAccordionSummary-content': { my: 0.5, alignItems: 'center', gap: 0.5 },
                }}
              >
                <CategoryIcon sx={{ fontSize: 18, color: 'info.main' }} />
                <Typography sx={{ fontSize: '0.8rem', fontWeight: 600, ml: 0.5 }}>
                  {st.displayName || key}
                </Typography>
                <Box
                  component="span"
                  sx={{
                    ml: 'auto',
                    mr: 1,
                    minWidth: 20, height: 20, borderRadius: '10px',
                    bgcolor: 'info.main', color: 'white',
                    display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
                    fontSize: '0.65rem', fontWeight: 700,
                    px: 0.75, lineHeight: 1,
                  }}
                >
                  {fields.length}
                </Box>
              </AccordionSummary>
              <AccordionDetails sx={{ pt: 0, pb: 1 }}>
                {fields.length === 0 ? (
                  <Typography variant="caption" color="text.secondary" sx={{ pl: 1 }}>
                    No fields declared on this subtype
                  </Typography>
                ) : (
                  fields.map((field) => {
                    const k = field.technicalName || field.name;
                    return (
                      <DraggableBOFieldItem
                        key={`${key}-${k}`}
                        field={field}
                        isSelected={selectedFieldKeys.includes(k)}
                        onAdd={onAddFieldToCanvas}
                        onSelect={handleToggleFieldSelection}
                        selectedFields={selectedFields}
                        mode={mode}
                      />
                    );
                  })
                )}
              </AccordionDetails>
            </Accordion>
          );
        })}

        {/* Related Business Objects Accordion */}
        {relatedBOs && relatedBOs.length > 0 && (
          <Accordion
            expanded={expandedAccordion === 'related'}
            onChange={handleAccordionChange('related')}
            disableGutters
            sx={{
              '&::before': { display: 'none' },
              bgcolor: 'transparent',
              boxShadow: 'none',
            }}
          >
            <AccordionSummary
              expandIcon={<ExpandMoreIcon />}
              sx={{
                minHeight: 36,
                '& .MuiAccordionSummary-content': { my: 0.5, alignItems: 'center', gap: 0.5 },
              }}
            >
              <LayersIcon sx={{ fontSize: 18, color: 'secondary.main' }} />
              <Typography sx={{ fontSize: '0.8rem', fontWeight: 600, ml: 0.5 }}>
                Related Objects
              </Typography>
              <Box
                component="span"
                sx={{
                  ml: 'auto',
                  mr: 1,
                  minWidth: 20, height: 20, borderRadius: '10px',
                  bgcolor: 'secondary.main', color: 'white',
                  display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
                  fontSize: '0.65rem', fontWeight: 700,
                  px: 0.75, lineHeight: 1,
                }}
              >
                {relatedBOs.length}
              </Box>
            </AccordionSummary>
            <AccordionDetails sx={{ pt: 0, pb: 1 }}>
              <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
                {relatedBOs.map((r: any, idx: number) => {
                  const rName = r.boName || r.bo_name || r.name || r.id || 'Related Object';
                  const rEdge = r.edge || r.relationship || r.type || '';
                  return (
                    <Box
                      key={`${rName}-${idx}`}
                      sx={{
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'space-between',
                        p: 0.75,
                        borderRadius: 1,
                        bgcolor: 'rgba(255, 255, 255, 0.03)',
                        border: '1px solid rgba(255, 255, 255, 0.08)',
                      }}
                    >
                      <Box sx={{ minWidth: 0 }}>
                        <Typography variant="body2" sx={{ fontSize: '0.75rem', fontWeight: 600 }} noWrap>
                          {rName}
                        </Typography>
                        {rEdge && (
                          <Typography variant="caption" color="text.secondary" sx={{ display: 'block', fontSize: '0.65rem' }}>
                            {rEdge}
                          </Typography>
                        )}
                      </Box>
                    </Box>
                  );
                })}
              </Box>
            </AccordionDetails>
          </Accordion>
        )}
      </Box>

      {/* Resize handle */}
      {onResize && (
        <Box
          onMouseDown={onResize}
          sx={{
            position: 'absolute',
            right: 0,
            top: 0,
            bottom: 0,
            width: '6px',
            cursor: 'ew-resize',
            bgcolor: 'transparent',
            transition: 'background-color 0.15s',
            zIndex: 3,
            '&:hover': { bgcolor: 'primary.main', opacity: 0.6 },
          }}
        />
      )}
    </Box>
  );
};

export default BOFieldsPalette;
