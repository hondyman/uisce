import React, { useMemo, useState } from 'react';
import {
  Box,
  Typography,
  TextField,
  InputAdornment,
  Chip,
  Card,
  CardContent,
  List,
  ListItem,
  ListItemButton,
  ListItemText,
  ListItemIcon,
  Collapse,
  IconButton,
  Tooltip,
  Badge,
  Paper,
  Divider,
} from '@mui/material';
import {
  Search as SearchIcon,
  ExpandMore as ExpandMoreIcon,
  ChevronRight as ChevronRightIcon,
  Category as CategoryIcon,
  Label as LabelIcon,
  CheckCircle as CheckCircleIcon,
  Info as InfoIcon,
} from '@mui/icons-material';
import { useTenant } from '../../../contexts/TenantContext';
import { useNodeTypes } from '../../../api/nodeTypes';
import { usePropertyLookupMaps } from '../../../hooks/usePropertyLookupMaps';
import { EnhancedSelectedAsset } from '../../../types/SemanticTypes';

interface SemanticAsset {
  id: string;
  node_name: string;
  description: string;
  parent_id?: string;
  properties: Record<string, unknown>;
}

interface BusinessTermTreeProps {
  businessTerms: SemanticAsset[];
  onAssetSelect: (_asset: EnhancedSelectedAsset) => void;
  searchTerm: string;
  highlightedItem: string | null;
}

interface CategoryGroup {
  name: string;
  terms: (SemanticAsset & { type: 'business_term' })[];
  subcategories: Map<string, CategoryGroup>;
}

const BusinessTermTree: React.FC<BusinessTermTreeProps> = ({
  businessTerms,
  onAssetSelect,
  searchTerm,
  highlightedItem
}) => {
  const [expandedCategories, setExpandedCategories] = useState<Set<string>>(new Set(['root']));
  const [localSearch, setLocalSearch] = useState('');
  const { tenant } = useTenant();
  const { data: nodeTypes } = useNodeTypes(tenant?.id || '');
  
  const businessTermNodeType = useMemo(() => {
    if (!nodeTypes || nodeTypes.length === 0) return null;
    const found = (nodeTypes as any[]).find((nt) => {
      const name = String(nt.catalog_type_name || '').toLowerCase();
      return name === 'business_term' || name === 'business term' || name.includes('business_term') || name.includes('business term');
    });
    return found || null;
  }, [nodeTypes]);

  const topLevelLookupMaps = usePropertyLookupMaps(businessTermNodeType);

  const nodeNameMap = useMemo(() => {
    const m = new Map<string, string>();
    businessTerms.forEach(b => {
      if (b.id && b.node_name) m.set(b.id, b.node_name);
    });
    return m;
  }, [businessTerms]);

  const resolveCategoryValue = (propKeys: string[], value: any): string => {
    if (!value) return '';
    const valStr = String(value);
    const uuidRegex = /^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$/;
    if (!uuidRegex.test(valStr)) return valStr;

    for (const k of propKeys) {
      if (topLevelLookupMaps?.[k]) {
        const found = topLevelLookupMaps[k].get(valStr);
        if (found) return found;
      }
    }

    const fromNode = nodeNameMap.get(valStr);
    if (fromNode) return fromNode;

    return `Unknown (${valStr.substring(0, 8)}...)`;
  };

  const getCategoryLevels = (asset: SemanticAsset): string[] => {
    const props = asset.properties || {};
    const level1 = resolveCategoryValue(['category_1', 'category1', 'category_level_1', 'category'], props.category_level_1 || props.category1 || props.category_1 || props.category);
    const level2 = resolveCategoryValue(['category_2', 'category2', 'category_level_2', 'sub_category'], props.category_level_2 || props.category2 || props.category_2 || props.sub_category);
    const level3 = resolveCategoryValue(['category_3', 'category3', 'category_level_3'], props.category_level_3 || props.category3 || props.category_3);
    return [level1, level2, level3].filter(Boolean);
  };

  // Build hierarchical structure
  const categoryTree = useMemo(() => {
    const terms = businessTerms.map(bt => ({ ...bt, type: 'business_term' as const }));
    const root: CategoryGroup = { name: 'root', terms: [], subcategories: new Map() };

    terms.forEach(term => {
      const categories = getCategoryLevels(term);
      
      if (categories.length === 0) {
        root.terms.push(term);
        return;
      }

      let current = root;
      categories.forEach((category, index) => {
        if (!current.subcategories.has(category)) {
          current.subcategories.set(category, {
            name: category,
            terms: [],
            subcategories: new Map()
          });
        }
        current = current.subcategories.get(category)!;
        
        // Add term to the deepest category
        if (index === categories.length - 1) {
          current.terms.push(term);
        }
      });
    });

    return root;
  }, [businessTerms]);

  // Filter terms
  const effectiveSearch = searchTerm || localSearch;
  const filteredTerms = useMemo(() => {
    if (!effectiveSearch.trim()) return businessTerms.map(bt => ({ ...bt, type: 'business_term' as const }));
    
    const lower = effectiveSearch.toLowerCase();
    return businessTerms
      .filter(bt => 
        bt.node_name.toLowerCase().includes(lower) ||
        bt.description?.toLowerCase().includes(lower) ||
        getCategoryLevels(bt).some(cat => cat.toLowerCase().includes(lower))
      )
      .map(bt => ({ ...bt, type: 'business_term' as const }));
  }, [businessTerms, effectiveSearch]);

  const handleToggleCategory = (categoryPath: string) => {
    setExpandedCategories(prev => {
      const next = new Set(prev);
      if (next.has(categoryPath)) {
        next.delete(categoryPath);
      } else {
        next.add(categoryPath);
      }
      return next;
    });
  };

  const handleAssetClick = (asset: SemanticAsset) => {
    // Determine the actual type of the asset from the data
    // Priority: catalog_type > node_type > lookup by node_type_id
    let assetType = (asset as any).catalog_type || (asset as any).node_type;
    
    // If not found, try to look up by node_type_id
    if (!assetType && (asset as any).node_type_id) {
      const nodeTypeId = (asset as any).node_type_id;
      const nodeType = nodeTypes?.find((nt: any) => nt.id === nodeTypeId);
      if (nodeType) {
        assetType = nodeType.catalog_type_name;
        console.log(`[BusinessTermTree] Resolved type from node_type_id: ${nodeTypeId} -> ${assetType}`);
      }
    }
    
    if (!assetType) {
      console.error('[BusinessTermTree] Asset is missing catalog_type/node_type and could not resolve from node_type_id:', asset);
      return;
    }
    
    const enhancedAsset: EnhancedSelectedAsset = {
      type: assetType as any,
      id: `${assetType}-${asset.id}`,
      nodeId: asset.id,
      name: asset.node_name,
      node: asset
    };
    
    console.log('[BusinessTermTree] Selected asset:', { type: assetType, name: asset.node_name });
    onAssetSelect(enhancedAsset);
  };

  const renderCategoryGroup = (group: CategoryGroup, path: string, level: number = 0) => {
    const isExpanded = expandedCategories.has(path);
    const hasSubcategories = group.subcategories.size > 0;
    const totalTerms = group.terms.length;

    return (
      <Box key={path}>
        {group.name !== 'root' && (
          <ListItem disablePadding sx={{ pl: level * 2 }}>
            <ListItemButton
              onClick={() => handleToggleCategory(path)}
              sx={{
                borderRadius: 1,
                '&:hover': { bgcolor: 'action.hover' },
              }}
            >
              <ListItemIcon sx={{ minWidth: 32 }}>
                {hasSubcategories ? (
                  isExpanded ? <ExpandMoreIcon /> : <ChevronRightIcon />
                ) : (
                  <CategoryIcon color="primary" fontSize="small" />
                )}
              </ListItemIcon>
              <ListItemText
                primary={group.name}
                primaryTypographyProps={{
                  fontWeight: 600,
                  fontSize: '0.9rem',
                }}
              />
              <Chip
                label={totalTerms}
                size="small"
                sx={{ height: 20, fontSize: '0.7rem' }}
              />
            </ListItemButton>
          </ListItem>
        )}

        <Collapse in={group.name === 'root' || isExpanded} timeout="auto">
          {/* Render terms in this category */}
          {group.terms.map(term => {
            const assetId = `${term.type}-${term.id}`;
            const isSelected = highlightedItem === assetId;
            const isMapped = term.properties?.mapped === true;

            return (
              <ListItem
                key={assetId}
                disablePadding
                sx={{ pl: (level + 1) * 2 }}
              >
                <ListItemButton
                  selected={isSelected}
                  onClick={() => handleAssetClick(term)}
                  sx={{
                    borderRadius: 1,
                    borderLeft: isSelected ? '4px solid' : '4px solid transparent',
                    borderLeftColor: isSelected ? 'primary.main' : 'transparent',
                    '&.Mui-selected': {
                      bgcolor: 'primary.light',
                      '&:hover': { bgcolor: 'primary.light' },
                    },
                  }}
                >
                  <ListItemIcon sx={{ minWidth: 32 }}>
                    {isMapped ? (
                      <CheckCircleIcon color="success" fontSize="small" />
                    ) : null}
                  </ListItemIcon>
                  <ListItemText
                    primary={term.node_name}
                    secondary={term.description}
                    primaryTypographyProps={{
                      fontSize: '0.875rem',
                      fontWeight: isSelected ? 600 : 400,
                    }}
                    secondaryTypographyProps={{
                      fontSize: '0.75rem',
                      noWrap: true,
                    }}
                  />
                  {term.description && (
                    <Tooltip title={term.description} placement="right">
                      <IconButton size="small" sx={{ ml: 1 }}>
                        <InfoIcon fontSize="small" />
                      </IconButton>
                    </Tooltip>
                  )}
                </ListItemButton>
              </ListItem>
            );
          })}

          {/* Render subcategories */}
          {Array.from(group.subcategories.entries()).map(([name, subgroup]) =>
            renderCategoryGroup(subgroup, `${path}/${name}`, level + 1)
          )}
        </Collapse>
      </Box>
    );
  };

  if (filteredTerms.length === 0) {
    return (
      <Box sx={{ p: 4, textAlign: 'center' }}>
        <LabelIcon sx={{ fontSize: 64, color: 'text.disabled', mb: 2 }} />
        <Typography variant="h6" color="text.secondary">
          {effectiveSearch ? 'No terms found' : 'No Business Terms'}
        </Typography>
        <Typography variant="body2" color="text.secondary">
          {effectiveSearch ? 'Try adjusting your search' : 'No business terms configured'}
        </Typography>
      </Box>
    );
  }

  return (
    <Box sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      {/* Header with Stats */}
      <Paper elevation={0} sx={{ p: 2, borderBottom: 1, borderColor: 'divider' }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
          <Typography variant="h6" fontWeight={600}>
            Business Terms
          </Typography>
          <Box sx={{ display: 'flex', gap: 1 }}>
            <Chip
              icon={<LabelIcon />}
              label={`${filteredTerms.length} terms`}
              size="small"
              color="primary"
              variant="outlined"
            />
            <Chip
              icon={<CheckCircleIcon />}
              label={`${filteredTerms.filter(t => t.properties?.mapped).length} mapped`}
              size="small"
              color="success"
              variant="outlined"
            />
          </Box>
        </Box>

        {/* Search */}
        <TextField
          fullWidth
          size="small"
          placeholder="Search terms..."
          value={localSearch}
          onChange={(e) => setLocalSearch(e.target.value)}
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <SearchIcon />
              </InputAdornment>
            ),
          }}
        />
      </Paper>

      {/* Tree View */}
      <Box sx={{ flex: 1, overflow: 'auto', p: 1 }}>
        <List dense>
          {effectiveSearch ? (
            // Flat list when searching
            filteredTerms.map(term => {
              const assetId = `${term.type}-${term.id}`;
              const isSelected = highlightedItem === assetId;
              const isMapped = term.properties?.mapped === true;

              return (
                <ListItem key={assetId} disablePadding>
                  <ListItemButton
                    selected={isSelected}
                    onClick={() => handleAssetClick(term)}
                    sx={{
                      borderRadius: 1,
                      '&.Mui-selected': {
                        bgcolor: 'primary.light',
                        '&:hover': { bgcolor: 'primary.light' },
                      },
                    }}
                  >
                    <ListItemIcon sx={{ minWidth: 32 }}>
                      {isMapped ? (
                        <CheckCircleIcon color="success" fontSize="small" />
                      ) : null}
                    </ListItemIcon>
                    <ListItemText
                      primary={term.node_name}
                      secondary={getCategoryLevels(term).join(' > ')}
                      primaryTypographyProps={{
                        fontSize: '0.875rem',
                        fontWeight: isSelected ? 600 : 400,
                      }}
                      secondaryTypographyProps={{
                        fontSize: '0.75rem',
                      }}
                    />
                  </ListItemButton>
                </ListItem>
              );
            })
          ) : (
            // Hierarchical tree when not searching
            renderCategoryGroup(categoryTree, 'root')
          )}
        </List>
      </Box>
    </Box>
  );
};

export default BusinessTermTree;