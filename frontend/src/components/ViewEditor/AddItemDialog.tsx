import React, { useEffect, useState, useMemo } from 'react';
import { devError } from '../../utils/devLogger';
import {
  Dialog,
  DialogContent,
  DialogActions,
  Button,
  Box,
  Autocomplete,
  TextField,
  Checkbox,
  Typography,
  Chip,
} from '@mui/material';
import ModalHeader from '../../components/ModalHeader';
import renderCoreCustomChips from '../../components/common/semanticChips';

interface AddItemDialogProps {
  open: boolean;
  type: 'cube' | 'dimension' | 'measure';
  onClose: () => void;
  onAdd: (type: 'cube' | 'dimension' | 'measure', data: any) => void;
  tenantId?: string;
  datasourceId?: string;
  viewData?: any;
  onUpgradeCubeRef?: (index: number, cubeObj: any) => void;
}

export const AddItemDialog: React.FC<AddItemDialogProps> = ({ open, type, onClose, onAdd, tenantId, datasourceId, viewData, onUpgradeCubeRef }) => {
  const isMultiSelect = type === 'dimension' || type === 'measure';
  const [selectedItems, setSelectedItems] = useState<any[]>([]);
  const [inputValue, setInputValue] = useState('');
  const [loading, setLoading] = useState(false);
  const [allowedDimensionSources, setAllowedDimensionSources] = useState<any[]>([]);
  const [filterType, setFilterType] = useState<'all' | 'core' | 'custom'>('all');

  useEffect(() => {
    if (open && tenantId && datasourceId) {
      setLoading(true);

      if (type === 'cube') {
        const fetchCubes = async () => {
          try {
            const dsQuery = datasourceId ? `?tenant_instance_id=${encodeURIComponent(datasourceId)}` : '';
            const response = await fetch(`/api/fabric/models${dsQuery}`);
            if (response.ok) {
              const data = await response.json();
              const cubes = data?.models || [];
              const mapped = cubes
                .filter((cube: any) => cube.published_at || cube.is_core)
                .map((cube: any) => ({
                  id: cube.id || cube.model_key,
                  name: cube.display_name || cube.title || cube.model_key,
                  path: cube.model_key || cube.modelKey || cube.path,
                  type: 'cube',
                  source: cube.source || cube.source_config?.source || 'core',
                  description: cube.description || cube.title || cube.display_name,
                  published_at: cube.published_at || cube.publishedAt,
                  is_core: Boolean(cube.is_core),
                  is_custom: Boolean(cube.is_custom),
                }));
              mapped.sort((a: any, b: any) => {
                const pa = (a.path || a.name || '').toLowerCase();
                const pb = (b.path || b.name || '').toLowerCase();
                if (pa < pb) return -1;
                if (pa > pb) return 1;
                return 0;
              });
              setAllowedDimensionSources(mapped);
            }
          } catch (error) {
            devError('Failed to fetch cubes:', error);
          } finally {
            setLoading(false);
          }
        };
        fetchCubes();
      } else if ((type === 'dimension' || type === 'measure') && viewData?.cubes) {
        const fetchDimensionsAndMeasures = async () => {
          try {
            const dsQuery = datasourceId ? `?tenant_instance_id=${encodeURIComponent(datasourceId)}` : '';
            const response = await fetch(`/api/fabric/models${dsQuery}`);
            if (response.ok) {
              const data = await response.json();
              const allCubes = data?.models || [];

              let selectedCubeRef = null;
              if (viewData.selectedCube) {
                selectedCubeRef = viewData.selectedCube;
              } else if (Array.isArray(viewData.cubes) && viewData.cubes.length > 0) {
                selectedCubeRef = viewData.cubes[viewData.cubes.length - 1];
              }
              if (!selectedCubeRef) {
                setAllowedDimensionSources([]);
                setLoading(false);
                return;
              }

              let cubeId = typeof selectedCubeRef === 'object' ? selectedCubeRef.id : null;
              let cubeModel = null;
              if (cubeId) {
                cubeModel = allCubes.find((c: any) => c.id === cubeId);
              } else {
                let legacyValue = typeof selectedCubeRef === 'string' ? selectedCubeRef : (selectedCubeRef.join_path || selectedCubeRef.name);
                if (typeof legacyValue === 'string') {
                  legacyValue = legacyValue.replace(/\s*\(Custom\)\s*/gi, '').replace(/\s*\(Core\)\s*/gi, '').trim();
                }
                cubeModel = allCubes.find((c: any) => {
                  const displayName = (c.display_name || c.displayName || '').toLowerCase().trim();
                  const modelKey = (c.model_key || c.modelKey || c.path || '').toLowerCase();
                  const searchValue = legacyValue.toLowerCase();
                  if (displayName === searchValue || displayName === `${searchValue} (custom)`) return true;
                  const modelKeyStripped = modelKey.replace(/^\/public\//, '').replace(/^\//, '');
                  return modelKey === searchValue || modelKey === `/${searchValue}` || modelKey === `/public/${searchValue}` || modelKey === `/public/${searchValue}_custom` || modelKeyStripped === searchValue || modelKeyStripped === `${searchValue}_custom`;
                });
                if (cubeModel) {
                  cubeId = cubeModel.id;
                  const cubeIndex = (viewData.cubes || []).length - 1;
                  if (onUpgradeCubeRef && cubeIndex >= 0) {
                    const upgradedCube: any = {
                      id: cubeModel.id,
                      join_path: cubeModel.model_key || cubeModel.modelKey,
                      includes: typeof selectedCubeRef === 'object' ? (selectedCubeRef.includes || '*') : '*',
                    };
                    if (typeof selectedCubeRef === 'object') {
                      if (selectedCubeRef.alias) upgradedCube.alias = selectedCubeRef.alias;
                      if (selectedCubeRef.prefix) upgradedCube.prefix = selectedCubeRef.prefix;
                      if (selectedCubeRef.excludes) upgradedCube.excludes = selectedCubeRef.excludes;
                    }
                    onUpgradeCubeRef(cubeIndex, upgradedCube);
                  }
                }
              }

              const alias = typeof selectedCubeRef === 'object' ? selectedCubeRef.alias : null;
              let joinPath = typeof selectedCubeRef === 'string' ? selectedCubeRef : (selectedCubeRef.join_path || selectedCubeRef.name || cubeModel?.model_key);
              if (typeof joinPath === 'string') {
                joinPath = joinPath.replace(/\s*\(Custom\)\s*/gi, '').replace(/\s*\(Core\)\s*/gi, '').trim();
              }

              if (!cubeModel) {
                setAllowedDimensionSources([]);
                setLoading(false);
                return;
              }

              const rawDimensions = cubeModel.resolved_config?.cubes?.[0]?.dimensions;
              const rawMeasures = cubeModel.resolved_config?.cubes?.[0]?.measures;

              const dimensionEntries: [string, any][] = Array.isArray(rawDimensions)
                ? rawDimensions.map((dim: any, index: number): [string, any] => {
                    const fallbackName = dim?.name || dim?.sql || `dimension_${index}`;
                    return [fallbackName, dim];
                  })
                : (Object.entries(rawDimensions || {}) as [string, any][]);

              const measureEntries: [string, any][] = Array.isArray(rawMeasures)
                ? rawMeasures.map((meas: any, index: number): [string, any] => {
                    const fallbackName = meas?.name || meas?.title || `measure_${index}`;
                    return [fallbackName, meas];
                  })
                : (Object.entries(rawMeasures || {}) as [string, any][]);

              const items: any[] = [];
              if (type === 'dimension') {
                dimensionEntries.forEach(([dimKey, dimValue]: [string, any]) => {
                  const dimensionName = (dimValue && dimValue.name) || dimKey;
                  const qualifiedName = alias ? `${alias}.${dimensionName}` : `${joinPath}.${dimensionName}`;
                  items.push({
                    id: `${joinPath}.${dimensionName}`,
                    name: dimValue?.title || dimensionName,
                    path: qualifiedName,
                    type: 'dimension',
                    source: joinPath,
                    description: dimValue?.description,
                    dimensionData: {
                      ...dimValue,
                      id: `${joinPath}.${dimensionName}`,
                      name: dimensionName,
                      join_path: joinPath,
                      qualified_name: qualifiedName,
                      source: joinPath,
                      alias,
                    }
                  });
                });
              }
              if (type === 'measure') {
                measureEntries.forEach(([measureKey, measureValue]: [string, any]) => {
                  const measureName = (measureValue && measureValue.name) || measureKey;
                  const qualifiedName = alias ? `${alias}.${measureName}` : `${joinPath}.${measureName}`;
                  items.push({
                    id: `${joinPath}.${measureName}`,
                    name: measureValue?.title || measureName,
                    path: qualifiedName,
                    type: 'measure',
                    source: joinPath,
                    description: measureValue?.description,
                    measureData: {
                      ...measureValue,
                      id: `${joinPath}.${measureName}`,
                      name: measureName,
                      join_path: joinPath,
                      qualified_name: qualifiedName,
                      source: joinPath,
                      alias,
                    }
                  });
                });
              }

              items.sort((a: any, b: any) => {
                const pa = a.path.toLowerCase();
                const pb = b.path.toLowerCase();
                if (pa < pb) return -1;
                if (pa > pb) return 1;
                return 0;
              });

              setAllowedDimensionSources(items);
            }
          } catch (error) {
            devError('Failed to fetch dimensions/measures:', error);
          } finally {
            setLoading(false);
          }
        };
        fetchDimensionsAndMeasures();
      } else {
        setAllowedDimensionSources([]);
        setLoading(false);
      }
    }
  }, [open, tenantId, datasourceId, type, viewData]);

  const filteredOptions = useMemo(() => {
    if (type !== 'cube') return allowedDimensionSources;
    return allowedDimensionSources.filter(option => {
      if (filterType === 'all') return true;
      if (filterType === 'core') return option.is_core;
      if (filterType === 'custom') return option.is_custom;
      return true;
    });
  }, [allowedDimensionSources, filterType, type]);

  useEffect(() => {
    if (!open) {
      setSelectedItems([]);
      setInputValue('');
    }
  }, [open]);

  useEffect(() => {
    setFilterType('all');
    setSelectedItems([]);
    setInputValue('');
  }, [type]);

  const dialogTitle = type === 'cube' ? 'Cube' : type === 'measure' ? 'Add Measure' : 'Add Dimension';
  const searchLabel = type === 'cube' ? 'Search cubes' : type === 'measure' ? 'Search measures' : 'Search dimensions';
  const searchPlaceholder = type === 'cube' ? 'Type to search cubes...' : type === 'measure' ? 'Type to search measures...' : 'Type to search dimensions...';

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <ModalHeader title={<>{dialogTitle}</>} onClose={onClose} />
      <DialogContent>
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, mt: 1 }}>
          {type === 'cube' && (
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, p: 2, bgcolor: 'grey.50', borderRadius: 1 }}>
              <Typography variant="body2" sx={{ fontWeight: 500, minWidth: 'fit-content' }}>
                Filter:
              </Typography>
              <Box sx={{ display: 'flex', gap: 1 }}>
                <Button size="small" variant={filterType === 'all' ? 'contained' : 'outlined'} onClick={() => setFilterType('all')} sx={{ minWidth: '60px' }}>All</Button>
                <Button size="small" variant={filterType === 'core' ? 'contained' : 'outlined'} onClick={() => setFilterType('core')} sx={{ minWidth: '60px' }}>Core</Button>
                <Button size="small" variant={filterType === 'custom' ? 'contained' : 'outlined'} onClick={() => setFilterType('custom')} sx={{ minWidth: '70px' }}>Custom</Button>
              </Box>
              {filterType !== 'all' && (<Button size="small" variant="text" onClick={() => setFilterType('all')} sx={{ minWidth: 'fit-content', ml: 'auto' }}>Clear</Button>)}
            </Box>
          )}

          <Autocomplete
            multiple={isMultiSelect}
            disableCloseOnSelect={isMultiSelect}
            options={filteredOptions}
            getOptionLabel={(opt: any) => opt.path || opt.name}
            value={isMultiSelect ? selectedItems : (selectedItems[0] ?? null)}
            onChange={(_, v) => {
              if (isMultiSelect) {
                setSelectedItems(Array.isArray(v) ? v : []);
              } else {
                setSelectedItems(v ? [v] : []);
              }
            }}
            inputValue={inputValue}
            onInputChange={(_, v) => setInputValue(v)}
            loading={loading}
            filterOptions={(opts) => opts}
            isOptionEqualToValue={(option, value) => option.id === value.id}
            renderOption={(props, option: any, { selected }) => (
              <li {...props} key={option.id}>
                <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', width: '100%' }}>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, flexGrow: 1 }}>
                    {isMultiSelect && (<Checkbox checked={selected} tabIndex={-1} disableRipple size="small" />)}
                    <Box sx={{ flexGrow: 1 }}>
                      <Typography variant="body2" sx={{ fontWeight: 500 }}>{option.name}</Typography>
                      <Typography variant="caption" color="text.secondary">{option.path}</Typography>
                    </Box>
                  </Box>
                  <Box sx={{ display: 'flex', gap: 0.5, ml: 1 }}>
                    {option.type === 'cube' && renderCoreCustomChips(option)}
                    {(option.type === 'dimension' || option.type === 'measure') && (<Chip label={option.source} size="small" color="info" variant="outlined" sx={{ fontSize: '0.7rem', height: '18px' }} />)}
                  </Box>
                </Box>
              </li>
            )}
            renderInput={(params) => (<TextField {...params} label={searchLabel} placeholder={searchPlaceholder} />)}
            sx={{ mb: 2 }}
          />
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button onClick={() => { if (selectedItems.length > 0) { const payload = isMultiSelect ? selectedItems : selectedItems[0]; onAdd(type, payload); } }} disabled={selectedItems.length === 0} variant="contained">
          {(() => {
            const baseLabel = type === 'cube' ? 'Add Cube' : type === 'measure' ? 'Add Measure' : 'Add Dimension';
            if (!isMultiSelect || selectedItems.length <= 1) return baseLabel;
            const plural = type === 'measure' ? 'Measures' : 'Dimensions';
            return `Add ${selectedItems.length} ${plural}`;
          })()}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default AddItemDialog;
