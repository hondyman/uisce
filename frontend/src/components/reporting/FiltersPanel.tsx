import React, { useState, useEffect, useCallback } from 'react';
import {
  Box, Typography, Button, Paper, Chip, IconButton,
  Tooltip, Divider, TextField, Menu, MenuItem,
  ListItemIcon, ListItemText,
} from '@mui/material';
import { useDroppable } from '@dnd-kit/core';
import FilterAltIcon from '@mui/icons-material/FilterAlt';
import AddIcon from '@mui/icons-material/Add';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import DragHandleIcon from '@mui/icons-material/DragHandle';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import PlaylistAddIcon from '@mui/icons-material/PlaylistAdd';
import FilterListIcon from '@mui/icons-material/FilterList';
import DragIndicatorIcon from '@mui/icons-material/DragIndicator';
import {
  Filter, FilterGroup, FilterModel, TenantDefaults, TenantCalendar,
  ReportParameter,
} from './filterTypes';
import { FilterEditModal } from './FilterEditModal';
import {
  createFilter, createGroup, loadTenantDefaults, listTenantCalendars,
  loadFilterModel, saveFilterModel, emptyFilterModel,
} from './filterApi';
import { getOperatorById } from './operatorMetadata';

interface FiltersPanelProps {
  selectedBO: any;
  reportId: string;
  parameters: ReportParameter[];
  onOpenParameters: () => void;
  pendingFieldDrop?: { field: any; meta?: { _scope?: string; _subtypeKey?: string } } | null;
  onPendingFieldDropConsumed?: () => void;
}

const FiltersPanel: React.FC<FiltersPanelProps> = ({
  selectedBO, reportId, parameters, onOpenParameters, pendingFieldDrop, onPendingFieldDropConsumed,
}) => {
  const { setNodeRef: setFilterDropRef, isOver } = useDroppable({
    id: 'filters-drop-zone',
  });
  const [filterModel, setFilterModel] = useState<FilterModel>(emptyFilterModel());
  const [editingFilter, setEditingFilter] = useState<Partial<Filter> | null>(null);
  const [editingGroupIdx, setEditingGroupIdx] = useState<number | null>(null);
  const [editModalOpen, setEditModalOpen] = useState(false);
  const [tenantDefaults, setTenantDefaults] = useState<TenantDefaults>({
    defaultCalendarCode: 'US', defaultFiscalYear: new Date().getFullYear(), defaultRegion: 'us-east-1',
  });
  const [calendars, setCalendars] = useState<TenantCalendar[]>([]);
  const [groupAddAnchor, setGroupAddAnchor] = useState<null | HTMLElement>(null);

  // Load tenant defaults + calendars on mount
  useEffect(() => {
    loadTenantDefaults().then(setTenantDefaults).catch(() => {});
    listTenantCalendars().then(setCalendars).catch(() => {});
  }, []);

  // Load saved filter model when reportId changes
  useEffect(() => {
    if (!reportId) return;
    loadFilterModel(reportId)
      .then(setFilterModel)
      .catch(() => setFilterModel(emptyFilterModel()));
  }, [reportId]);

  // Handle pending field drop from BOFieldsPalette
  useEffect(() => {
    if (pendingFieldDrop) {
      const { field, meta } = pendingFieldDrop;
      openEditModal(
        filterModel.groups.length > 0 ? filterModel.groups.length - 1 : undefined,
        {
          field: field.name || field.technicalName,
          fieldScope: meta?._scope,
          fieldSubtypeKey: meta?._subtypeKey,
          operator: 'equals',
          valueSource: { kind: 'constant', value: '' },
          values: [],
          enabled: true,
        }
      );
      onPendingFieldDropConsumed?.();
    }
  }, [pendingFieldDrop]);

  const saveFilters = useCallback(async (model: FilterModel) => {
    if (!reportId) return;
    try {
      await saveFilterModel(reportId, model, parameters, tenantDefaults);
    } catch (e) {
      console.warn('Could not persist filters:', e);
    }
  }, [reportId, parameters, tenantDefaults]);

  // Gather all fields from selectedBO (including subtypes)
  const allFields = React.useMemo(() => {
    if (!selectedBO) return [];
    const fields: Array<{ name: string; label: string; dataType: string; _scope?: string; _subtypeKey?: string }> = [];
    const seen = new Set<string>();

    const addField = (f: any, scope: 'root' | 'subtype' = 'root', subtypeKey?: string) => {
      const key = f.key || f.technicalName || f.name;
      if (!key || seen.has(key)) return;
      seen.add(key);
      fields.push({
        name: key,
        label: f.label || f.displayName || f.name || key,
        dataType: f.dataType || f.type || 'string',
        _scope: scope,
        _subtypeKey: subtypeKey,
      });
    };

    (selectedBO.coreFields || []).forEach((f: any) => addField(f, 'root'));
    (selectedBO.customFields || []).forEach((f: any) => addField(f, 'root'));
    (selectedBO.fields || []).forEach((f: any) => addField(f, 'root'));
    (selectedBO.config?.fields || []).forEach((f: any) => addField(f, 'root'));
    if (selectedBO.subtypes && typeof selectedBO.subtypes === 'object') {
      Object.entries(selectedBO.subtypes).forEach(([stKey, subtype]: [string, any]) => {
        (subtype.subtypeFields || []).forEach((f: any) => addField(f, 'subtype', stKey));
      });
    }
    return fields;
  }, [selectedBO]);

  const totalFilterCount = filterModel.groups.reduce((n, g) => n + g.filters.filter(f => f.enabled).length, 0);

  const openEditModal = (groupIdx: number, filter?: Partial<Filter>) => {
    setEditingGroupIdx(groupIdx);
    setEditingFilter(filter || null);
    setEditModalOpen(true);
  };

  const handleSaveFilter = (filterData: Partial<Filter>) => {
    const newFilter: Filter = {
      id: filterData.id || `f_${Date.now()}`,
      field: filterData.field || '',
      operator: filterData.operator || 'equals',
      valueSource: filterData.valueSource || { kind: 'constant', value: '' },
      values: filterData.values || [],
      enabled: filterData.enabled !== false,
      fieldScope: filterData.fieldScope,
      fieldSubtypeKey: filterData.fieldSubtypeKey,
    };

    setFilterModel(prev => {
      const groups = [...prev.groups];
      if (editingGroupIdx === null || editingGroupIdx === undefined) {
        // Add new group for this filter
        const newGroup = createGroup({ filters: [newFilter] });
        groups.push(newGroup);
      } else {
        const existing = groups[editingGroupIdx];
        if (filterData.id && existing.filters.some(f => f.id === filterData.id)) {
          // Update existing filter
          groups[editingGroupIdx] = {
            ...existing,
            filters: existing.filters.map(f => f.id === filterData.id ? newFilter : f),
          };
        } else {
          // Add new filter to existing group
          groups[editingGroupIdx] = {
            ...existing,
            filters: [...existing.filters, newFilter],
          };
        }
      }
      const updated = { ...prev, groups };
      saveFilters(updated);
      return updated;
    });
  };

  const handleDeleteFilter = (groupIdx: number, filterId: string) => {
    setFilterModel(prev => {
      const groups = [...prev.groups];
      groups[groupIdx] = {
        ...groups[groupIdx],
        filters: groups[groupIdx].filters.filter(f => f.id !== filterId),
      };
      // Remove empty groups
      const filtered = groups.filter(g => g.filters.length > 0);
      const updated = { ...prev, groups: filtered };
      saveFilters(updated);
      return updated;
    });
  };

  const handleDeleteGroup = (groupIdx: number) => {
    setFilterModel(prev => {
      const groups = prev.groups.filter((_, i) => i !== groupIdx);
      const updated = { ...prev, groups };
      saveFilters(updated);
      return updated;
    });
  };

  const handleAddGroup = (combinator: 'AND' | 'OR' = 'AND') => {
    setFilterModel(prev => {
      const updated = { ...prev, groups: [...prev.groups, createGroup({ combinator })] };
      saveFilters(updated);
      return updated;
    });
    setGroupAddAnchor(null);
  };

  const handleGroupCombinatorToggle = (groupIdx: number) => {
    setFilterModel(prev => {
      const groups = [...prev.groups];
      groups[groupIdx] = {
        ...groups[groupIdx],
        combinator: groups[groupIdx].combinator === 'AND' ? 'OR' : 'AND',
      };
      const updated = { ...prev, groups };
      saveFilters(updated);
      return updated;
    });
  };

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
      {/* Sticky header */}
      <Box sx={{
        position: 'sticky', top: 0, zIndex: 2,
        bgcolor: 'background.paper',
        borderBottom: '1px solid', borderColor: 'divider',
        p: 2,
      }}>
        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1.5 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <FilterAltIcon color="primary" sx={{ fontSize: 20 }} />
            <Typography variant="subtitle1" fontWeight={700}>Filters</Typography>
            {totalFilterCount > 0 && (
              <Chip label={totalFilterCount} size="small" color="primary" sx={{ height: 18, fontSize: '0.7rem' }} />
            )}
          </Box>
          <Box sx={{ display: 'flex', gap: 0.5 }}>
            <Button
              size="small"
              startIcon={<PlaylistAddIcon sx={{ fontSize: 14 }} />}
              onClick={(e) => setGroupAddAnchor(e.currentTarget)}
              sx={{ textTransform: 'none', fontSize: '0.72rem', py: 0.4 }}
            >
              Add Group
            </Button>
            <Menu
              anchorEl={groupAddAnchor}
              open={Boolean(groupAddAnchor)}
              onClose={() => setGroupAddAnchor(null)}
            >
              <MenuItem onClick={() => handleAddGroup('AND')}>
                <ListItemText primary="Add Group (AND)" />
              </MenuItem>
              <MenuItem onClick={() => handleAddGroup('OR')}>
                <ListItemText primary="Add Group (OR)" />
              </MenuItem>
            </Menu>
          </Box>
        </Box>

        {/* Drop Zone */}
        <Box
          ref={setFilterDropRef}
          sx={{
            border: '2px dashed',
            borderColor: isOver ? 'primary.main' : 'divider',
            bgcolor: isOver ? 'action.hover' : 'transparent',
            borderRadius: 1.5,
            p: 1.5,
            textAlign: 'center',
            cursor: 'default',
            transition: 'all 0.15s',
            '&:hover': { borderColor: 'primary.main', bgcolor: 'action.hover' },
          }}
        >
          <Typography variant="caption" color="text.secondary">
            Drag a field from the BO Fields palette here to create a filter
          </Typography>
        </Box>

        {/* Manual Add */}
        <Box sx={{ mt: 1, display: 'flex', gap: 1, alignItems: 'center' }}>
          <Button
            fullWidth
            variant="outlined"
            size="small"
            startIcon={<AddIcon sx={{ fontSize: 14 }} />}
            onClick={() => openEditModal(filterModel.groups.length > 0 ? 0 : undefined)}
            disabled={allFields.length === 0}
            sx={{ textTransform: 'none', fontSize: '0.72rem', py: 0.5 }}
          >
            Add Filter
          </Button>
          {parameters.length === 0 && (
            <Tooltip title="No parameters defined. Click to add parameters.">
              <Button
                size="small"
                variant="text"
                onClick={onOpenParameters}
                sx={{ textTransform: 'none', fontSize: '0.7rem', py: 0.5, minWidth: 0 }}
              >
                + Parameter
              </Button>
            </Tooltip>
          )}
        </Box>
      </Box>

      {/* Filter groups list */}
      <Box sx={{ flex: 1, overflowY: 'auto', p: 2, display: 'flex', flexDirection: 'column', gap: 1.5 }}>
        {filterModel.groups.length === 0 ? (
          <Box sx={{ textAlign: 'center', py: 4 }}>
            <FilterListIcon sx={{ fontSize: 40, color: 'text.disabled', mb: 1 }} />
            <Typography variant="body2" color="text.secondary">
              No filters yet.
            </Typography>
            <Typography variant="caption" color="text.disabled">
              Drag a field here or click "Add Filter"
            </Typography>
          </Box>
        ) : (
          filterModel.groups.map((group, gIdx) => (
            <Paper
              key={group.id}
              elevation={1}
              sx={{ overflow: 'hidden', border: '1px solid', borderColor: 'divider' }}
            >
              {/* Group header */}
              <Box sx={{
                display: 'flex', alignItems: 'center', justifyContent: 'space-between',
                px: 1.5, py: 1, bgcolor: 'action.hover', borderBottom: '1px solid', borderColor: 'divider',
              }}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                  <DragHandleIcon sx={{ fontSize: 16, color: 'text.disabled', cursor: 'grab' }} />
                  <Chip
                    label={group.combinator}
                    size="small"
                    onClick={() => handleGroupCombinatorToggle(gIdx)}
                    sx={{
                      height: 20, fontSize: '0.65rem', fontWeight: 700, cursor: 'pointer',
                      bgcolor: group.combinator === 'AND' ? 'primary.main' : 'secondary.main',
                      color: 'white',
                      '&:hover': { opacity: 0.85 },
                    }}
                  />
                  <Typography variant="caption" color="text.secondary">
                    {group.filters.filter(f => f.enabled).length} filter{group.filters.filter(f => f.enabled).length !== 1 ? 's' : ''}
                  </Typography>
                </Box>
                <Box sx={{ display: 'flex', gap: 0.5 }}>
                  <Tooltip title="Add filter to this group">
                    <IconButton size="small" onClick={() => openEditModal(gIdx)} sx={{ p: 0.25 }}>
                      <AddIcon sx={{ fontSize: 14 }} />
                    </IconButton>
                  </Tooltip>
                  <Tooltip title="Delete group">
                    <IconButton size="small" onClick={() => handleDeleteGroup(gIdx)} sx={{ p: 0.25 }}>
                      <DeleteOutlineIcon sx={{ fontSize: 14 }} />
                    </IconButton>
                  </Tooltip>
                </Box>
              </Box>

              {/* Filters */}
              <Box sx={{ p: 1, display: 'flex', flexDirection: 'column', gap: 0.5 }}>
                {group.filters.map((filter, fIdx) => (
                  <FilterRow
                    key={filter.id}
                    filter={filter}
                    fields={allFields}
                    onEdit={() => openEditModal(gIdx, filter)}
                    onDelete={() => handleDeleteFilter(gIdx, filter.id)}
                  />
                ))}
                {group.filters.length === 0 && (
                  <Typography variant="caption" color="text.disabled" sx={{ textAlign: 'center', py: 0.5 }}>
                    No filters in this group
                  </Typography>
                )}
              </Box>
            </Paper>
          ))
        )}
      </Box>

      {/* Edit Modal */}
      <FilterEditModal
        open={editModalOpen}
        onClose={() => setEditModalOpen(false)}
        onSave={handleSaveFilter}
        initialFilter={editingFilter || undefined}
        fields={allFields}
        parameters={parameters}
        tenantDefaults={tenantDefaults}
        calendars={calendars}
      />
    </Box>
  );
};

interface FilterRowProps {
  filter: Filter;
  fields: Array<{ name: string; label: string; dataType: string; _scope?: string; _subtypeKey?: string }>;
  onEdit: () => void;
  onDelete: () => void;
}

const FilterRow: React.FC<FilterRowProps> = ({ filter, fields, onEdit, onDelete }) => {
  const fieldDef = fields.find(f => f.name === filter.field);
  const opDef = React.useMemo(() => {
    return getOperatorById(filter.operator);
  }, [filter.operator]);

  const sourceLabel = React.useMemo(() => {
    const vs = filter.valueSource;
    if (!vs) return '';
    switch (vs.kind) {
      case 'constant': return vs.value || '(empty)';
      case 'parameter': return vs.parameterName || vs.parameterId || '@param';
      case 'function': return `ƒ ${vs.expression?.slice(0, 20)}...`;
      case 'tenant_default': return vs.defaultKey || 'tenant default';
      case 'calendar': return vs.calendarCode || 'calendar';
      default: return '';
    }
  }, [filter.valueSource]);

  return (
    <Box
      onClick={onEdit}
      sx={{
        display: 'flex', alignItems: 'center', gap: 1, p: 1,
        borderRadius: 1, border: '1px solid', borderColor: 'divider',
        bgcolor: filter.enabled ? 'transparent' : 'action.disabledBackground',
        cursor: 'pointer',
        opacity: filter.enabled ? 1 : 0.5,
        '&:hover': { bgcolor: 'action.hover', borderColor: 'primary.main' },
      }}
    >
      <DragIndicatorIcon sx={{ fontSize: 14, color: 'text.disabled', flexShrink: 0 }} />
      <Box sx={{ flex: 1, minWidth: 0 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, flexWrap: 'nowrap', overflow: 'hidden' }}>
          <Typography variant="body2" sx={{ fontFamily: 'monospace', fontSize: '0.75rem', fontWeight: 600, noWrap: true }}>
            {fieldDef?.label || filter.field}
          </Typography>
          <Typography variant="caption" color="text.secondary" sx={{ noWrap: true, overflow: 'hidden', textOverflow: 'ellipsis', maxWidth: 80 }}>
            {filter.field}
          </Typography>
        </Box>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, mt: 0.25 }}>
          <Chip
            label={opDef?.label || filter.operator}
            size="small"
            sx={{ height: 16, fontSize: '0.6rem', bgcolor: 'action.selected' }}
          />
          <Typography variant="caption" color="text.secondary" sx={{ fontSize: '0.65rem', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', maxWidth: 120 }}>
            {sourceLabel}
          </Typography>
        </Box>
      </Box>
      <IconButton size="small" onClick={(e) => { e.stopPropagation(); onDelete(); }} sx={{ p: 0.25, flexShrink: 0 }}>
        <DeleteOutlineIcon sx={{ fontSize: 14 }} />
      </IconButton>
    </Box>
  );
};

export default FiltersPanel;
