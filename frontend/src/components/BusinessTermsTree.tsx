import React, { useMemo, useState, useEffect, useCallback } from 'react';
import { EnhancedSelectedAsset } from '../types/SemanticTypes';
import { CatalogNode } from '../api/glossary';
import { IconButton, Tooltip } from '@mui/material';
import { useTenant } from '../contexts/TenantContext';
import { usePropertyLookupMaps } from '../hooks/usePropertyLookupMaps';
import { useNodeTypes } from '../api/nodeTypes';
import { 
  EditOutlined as EditIcon, 
  DeleteOutline as DeleteIcon, 
  UnfoldMoreOutlined as UnfoldMoreIcon, 
  UnfoldLessOutlined as UnfoldLessIcon 
} from '@mui/icons-material';
import './BusinessTermsTree.css';
import { useTranslation } from 'react-i18next';
import { devDebug } from '../utils/devLogger';

interface BusinessTermsTreeProps {
  businessTerms: any[];
  semanticTerms: any[];
  semanticViews: any[];
  semanticEdges?: any[];
  selectedAsset: EnhancedSelectedAsset | null;
  onAssetSelect: (asset: EnhancedSelectedAsset) => void;
  searchTerm?: string;
  highlightedItem: string | null;
  onEditTerm?: (term: CatalogNode) => void;
  onDeleteTerm?: (term: CatalogNode) => void;
  filterType?: 'all' | 'with_relationships' | 'without_relationships';
}

interface SemanticAsset {
  id: string;
  node_name: string;
  description: string;
  parent_id?: string;
  properties: Record<string, unknown>;
}

interface CategoryNode {
  name: string;
  level: 1 | 2 | 3;
  terms: SemanticAsset[];
  children?: CategoryNode[];
  parent?: CategoryNode;
}

const BusinessTermsTree: React.FC<BusinessTermsTreeProps> = ({
  businessTerms = [],
  semanticEdges = [],
  onAssetSelect,
  highlightedItem,
  searchTerm = '',
  onEditTerm,
  onDeleteTerm,
  filterType = 'all',
}) => {
  const [isFlatView, setIsFlatView] = useState(false);
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(new Set());
  const [isDefaultExpansion, setIsDefaultExpansion] = useState(true);

  // Create a map of all available nodes for lookup name resolution
  const nodeNameMap = useMemo(() => {
    const map = new Map<string, string>();
    businessTerms.forEach((term: any) => {
      if (term.id && term.node_name) {
        map.set(term.id, term.node_name);
      }
    });
    devDebug('[BusinessTermsTree] nodeNameMap size:', map.size);
    return map;
  }, [businessTerms]);

  const { tenant } = useTenant();
  const { data: nodeTypes } = useNodeTypes(tenant?.id || '');
  const businessTermNodeType = useMemo(() => {
    if (!nodeTypes) return null;
    return (nodeTypes as any[]).find((nt) => {
      const name = String(nt.catalog_type_name || '').toLowerCase();
      return name === 'business_term' || name === 'business term' || name.includes('business_term') || name.includes('business term');
    });
  }, [nodeTypes]);

  const topLevelLookupMaps = usePropertyLookupMaps(businessTermNodeType);
  const [cascadingLookupCache, setCascadingLookupCache] = useState<Map<string, Map<string, string>>>(new Map());

  useEffect(() => {
    if (businessTermNodeType && topLevelLookupMaps) {
      devDebug('[BusinessTermsTree] Business term node type:', businessTermNodeType.catalog_type_name);
      devDebug('[BusinessTermsTree] Top-level lookup maps keys:', Object.keys(topLevelLookupMaps));
      Object.entries(topLevelLookupMaps).forEach(([key, map]) => {
        devDebug(`[BusinessTermsTree] Lookup map '${key}' has ${map?.size || 0} entries`);
        if (map && map.size > 0) {
          const entries = Array.from(map.entries()).slice(0, 5);
          devDebug(`[BusinessTermsTree]   Sample entries for '${key}':`, entries);
        }
      });
    }
  }, [businessTermNodeType, topLevelLookupMaps]);

  // Get category levels from business term properties with proper UUID resolution
  const getCategoryLevels = useCallback((term: SemanticAsset): [string, string, string] => {
    const props = term.properties || {};
    const uuidRegex = /^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$/;
    
    const resolveValue = (propKeyCandidates: string[], val: any, level: string): string => {
      if (!val) return '';
      const strVal = String(val);
      
      // If it's not a UUID, return as-is
      if (!uuidRegex.test(strVal)) {
        devDebug(`[BusinessTermsTree] ${level} value '${strVal}' is not a UUID, using as-is`);
        return strVal;
      }

      devDebug(`[BusinessTermsTree] Resolving ${level} UUID: ${strVal}`);
      
      // Try lookup maps first with all candidate keys
      for (const key of propKeyCandidates) {
        if (topLevelLookupMaps?.[key]?.has(strVal)) {
          const mapped = topLevelLookupMaps[key].get(strVal);
          if (mapped) {
            devDebug(`[BusinessTermsTree] ✓ Resolved ${level} via lookup '${key}': ${strVal} -> ${mapped}`);
            return mapped;
          }
        }
      }

      // Try nodeNameMap
      const fromNodeMap = nodeNameMap.get(strVal);
      if (fromNodeMap) {
        devDebug(`[BusinessTermsTree] ✓ Resolved ${level} via nodeNameMap: ${strVal} -> ${fromNodeMap}`);
        return fromNodeMap;
      }

      // Try cascading lookups
      const propDef = (businessTermNodeType?.properties as any[])?.find((p: any) => propKeyCandidates.includes(p.name));
      if (propDef?.cascade_from && propDef?.lookup_id) {
        const parentProperty = propDef.cascade_from;
        const parentVal = props[parentProperty];
        if (parentVal) {
          const cacheKey = `${propDef.lookup_id}_${parentVal}`;
          if (cascadingLookupCache.has(cacheKey)) {
            const cachedMap = cascadingLookupCache.get(cacheKey);
            const mapped = cachedMap?.get(strVal);
            if (mapped) {
              devDebug(`[BusinessTermsTree] ✓ Resolved ${level} via cascade: ${strVal} -> ${mapped}`);
              return mapped;
            }
          }
        }
      }
      
      devDebug(`[BusinessTermsTree] ✗ Could not resolve ${level} UUID: ${strVal}`);
      // Return "Unknown Category" instead of the UUID for better UX
      return `Unknown (${strVal.substring(0, 8)}...)`;
    };
    
    // Try all possible property name variations for each level
    const level1Candidates = ['category_1', 'category1', 'category_level_1', 'category'];
    const level2Candidates = ['category_2', 'category2', 'category_level_2', 'sub_category'];
    const level3Candidates = ['category_3', 'category3', 'category_level_3'];
    
    // Get raw values
    const level1Val = props.category_level_1 || props.category1 || props.category_1 || props.category;
    const level2Val = props.category_level_2 || props.category2 || props.category_2 || props.sub_category;
    const level3Val = props.category_level_3 || props.category3 || props.category_3;
    
    const level1 = resolveValue(level1Candidates, level1Val || 'Uncategorized', 'Level 1');
    const level2 = resolveValue(level2Candidates, level2Val || '', 'Level 2');
    const level3 = resolveValue(level3Candidates, level3Val || '', 'Level 3');
    
    return [level1, level2, level3];
  }, [topLevelLookupMaps, nodeNameMap, cascadingLookupCache, businessTermNodeType]);

  const filteredBusinessTerms = useMemo(() => {
    if (filterType === 'all' || !businessTerms) {
      return businessTerms;
    }

    const businessTermIds = new Set(businessTerms.map((term: any) => term.id));
    const termsWithRelationships = new Set<string>();

    semanticEdges.forEach((edge: any) => {
      if (edge.relationship_type === 'business_term_to_semantic_term') {
        if (businessTermIds.has(edge.source_node_id)) {
          termsWithRelationships.add(edge.source_node_id);
        }
      }
    });

    if (filterType === 'with_relationships') {
      return businessTerms.filter((term: any) => termsWithRelationships.has(term.id));
    } else if (filterType === 'without_relationships') {
      return businessTerms.filter((term: any) => !termsWithRelationships.has(term.id));
    }

    return businessTerms || [];
  }, [businessTerms, semanticEdges, filterType]);

  // Prefetch cascading values
  useEffect(() => {
    if (!businessTermNodeType?.properties || !filteredBusinessTerms?.length || !tenant?.id) return;

    const cascadeProps = (businessTermNodeType.properties as any[]).filter((p: any) => p.cascade_from && p.lookup_id);
    if (!cascadeProps?.length) return;

    filteredBusinessTerms.forEach((term: any) => {
      cascadeProps.forEach(async (p: any) => {
        const val = term.properties?.[p.name];
        const parentVal = term.properties?.[p.cascade_from];
        if (val && parentVal) {
          const cacheKey = `${p.lookup_id}_${parentVal}`;
          if (cascadingLookupCache.has(cacheKey)) return;
          
          setCascadingLookupCache(prev => {
            if (prev.has(cacheKey)) return prev;
            const newCache = new Map(prev);
            newCache.set(cacheKey, new Map());
            return newCache;
          });

          try {
            const url = `/api/lookups/${p.lookup_id}/values?tenant_id=${tenant.id}&parent_id=${encodeURIComponent(String(parentVal))}`;
            const res = await fetch(url, { credentials: 'include' });
            if (res.ok) {
              const data = await res.json();
              const lookupMap = new Map<string, string>();
              (data.items || []).forEach((item: any) => {
                if (item.id && item.label) {
                  lookupMap.set(item.id, item.label);
                }
              });
              
              setCascadingLookupCache(prev => {
                const newCache = new Map(prev);
                newCache.set(cacheKey, lookupMap);
                return newCache;
              });
            }
          } catch (err) {
            devDebug('[BusinessTermsTree] Error fetching cascading lookup:', err);
          }
        }
      });
    });
  }, [businessTermNodeType, filteredBusinessTerms, tenant?.id, cascadingLookupCache]);

  // Build hierarchical tree structure with accurate counts
  const categoryTree = useMemo(() => {
    const root: CategoryNode = {
      name: 'Root',
      level: 1,
      terms: [],
      children: []
    };

    const level1Map = new Map<string, CategoryNode>();

    (filteredBusinessTerms || []).forEach(term => {
      const [level1, level2, level3] = getCategoryLevels(term);

      let level1Node = level1Map.get(level1);
      if (!level1Node) {
        level1Node = {
          name: level1,
          level: 1,
          terms: [],
          children: []
        };
        level1Map.set(level1, level1Node);
        root.children!.push(level1Node);
      }

      if (!level2 && !level3) {
        // Term belongs directly to level 1
        level1Node.terms.push(term);
      } else if (level2 && !level3) {
        // Level 2 category
        let level2Node = level1Node.children!.find(child => child.name === level2);
        if (!level2Node) {
          level2Node = {
            name: level2,
            level: 2,
            terms: [],
            children: [],
            parent: level1Node
          };
          level1Node.children!.push(level2Node);
        }
        level2Node.terms.push(term);
      } else if (level2 && level3) {
        // Level 3 category
        let level2Node = level1Node.children!.find(child => child.name === level2);
        if (!level2Node) {
          level2Node = {
            name: level2,
            level: 2,
            terms: [],
            children: [],
            parent: level1Node
          };
          level1Node.children!.push(level2Node);
        }

        let level3Node = level2Node.children!.find(child => child.name === level3);
        if (!level3Node) {
          level3Node = {
            name: level3,
            level: 3,
            terms: [],
            children: [],
            parent: level2Node
          };
          level2Node.children!.push(level3Node);
        }
        level3Node.terms.push(term);
      }
    });

    return root;
  }, [filteredBusinessTerms, getCategoryLevels]);

  // Helper function to count all terms in a node and its children recursively
  const countAllTerms = useCallback((node: CategoryNode): number => {
    let count = node.terms.length;
    if (node.children) {
      node.children.forEach(child => {
        count += countAllTerms(child);
      });
    }
    return count;
  }, []);

  // Filter tree based on search
  const filteredTree = useMemo(() => {
    if (!searchTerm.trim()) {
      return categoryTree;
    }

    const searchLower = searchTerm.toLowerCase();

    const filterNode = (node: CategoryNode): CategoryNode | null => {
      const matchingTerms = node.terms.filter(term =>
        term.node_name.toLowerCase().includes(searchLower) ||
        term.description?.toLowerCase().includes(searchLower) ||
        getCategoryLevels(term).some(level => level.toLowerCase().includes(searchLower))
      );

      const matchingChildren = node.children?.map(filterNode).filter(Boolean) as CategoryNode[] || [];

      if (matchingTerms.length > 0 || matchingChildren.length > 0) {
        return {
          ...node,
          terms: matchingTerms,
          children: matchingChildren
        };
      }

      return null;
    };

    const filtered = filterNode(categoryTree);
    return filtered || { name: 'Root', level: 1, terms: [], children: [] };
  }, [categoryTree, searchTerm, getCategoryLevels]);

  // Flatten tree for flat view
  const flattenedTerms = useMemo(() => {
    if (!searchTerm.trim()) {
      return filteredBusinessTerms || [];
    }

    const searchLower = searchTerm.toLowerCase();
    return (filteredBusinessTerms || []).filter(term =>
      term.node_name.toLowerCase().includes(searchLower) ||
      term.description?.toLowerCase().includes(searchLower) ||
      getCategoryLevels(term).some(level => level.toLowerCase().includes(searchLower))
    );
  }, [filteredBusinessTerms, searchTerm, getCategoryLevels]);

  const handleNodeToggle = (nodePath: string) => {
    setExpandedNodes(prev => {
      const newSet = new Set(prev);
      if (newSet.has(nodePath)) {
        newSet.delete(nodePath);
      } else {
        newSet.add(nodePath);
      }
      return newSet;
    });
  };

  const handleBusinessTermSelect = (businessTerm: SemanticAsset) => {
    const asset: EnhancedSelectedAsset = {
      type: 'business_term',
      id: `business_term-${businessTerm.id}`,
      nodeId: businessTerm.id,
      name: businessTerm.node_name,
      businessTerm: businessTerm.node_name,
      node: businessTerm,
    };
    onAssetSelect(asset);
  };

  const handleExpandAll = useCallback(() => {
    setIsDefaultExpansion(false);
    const allPaths = new Set<string>();
    const collectPaths = (node: CategoryNode, path = ''): boolean => {
      const nodePath = path ? `${path}-${node.name}` : node.name;
      const hasTerms = node.terms && node.terms.length > 0;
      const childrenAreExpandable = (node.children || []).map(child => collectPaths(child, nodePath)).some(Boolean);
      const isExpandable = hasTerms || childrenAreExpandable;

      if (isExpandable) {
        allPaths.add(nodePath);
      }
      return isExpandable;
    };
  
    filteredTree.children?.forEach(child => collectPaths(child));
    setExpandedNodes(allPaths);
  }, [filteredTree]);

  const handleCollapseAll = () => {
    setIsDefaultExpansion(false);
    setExpandedNodes(new Set());
  };

  const renderCategoryNode = (node: CategoryNode, depth = 0, path = ''): React.ReactNode => {
    const nodePath = path ? `${path}-${node.name}` : node.name;
    const isExpanded = expandedNodes.has(nodePath);
    const hasChildren = node.children && node.children.length > 0;
    const hasTerms = node.terms.length > 0;
    const isExpandable = hasChildren || hasTerms;
    
    // Calculate total count including all descendants
    const totalCount = countAllTerms(node);

    return (
      <div key={nodePath} className="business-category-node">
        <div
          className={`business-category-header level-${node.level} depth-${depth} ${isExpandable ? 'expandable' : ''}`}
          onClick={() => isExpandable && handleNodeToggle(nodePath)}
        >
          {isExpandable && (
            <span className={`category-toggle ${isExpanded ? 'expanded' : 'collapsed'}`}>
              ▶
            </span>
          )}
          {!isExpandable && <span className="category-spacer">•</span>}
          <span className="category-name">{node.name}</span>
          <span className="category-count">({totalCount})</span>
        </div>

        {isExpanded && (
          <div className="category-children">
            {node.children?.map(child => renderCategoryNode(child, depth + 1, nodePath))}
            {node.terms.map(term => {
              const assetId = `business_term-${term.id}`;
              const isSelected = highlightedItem === assetId;
              return (
                <div
                  key={assetId}
                  className={`business-term-item depth-${depth + 1} ${isSelected ? 'selected' : ''}`}
                >
                  <div 
                    className="term-content"
                    onClick={() => handleBusinessTermSelect(term)}
                  >
                    <span className="term-name">{term.node_name}</span>
                    {term.description && (
                      <span className="term-description">{term.description}</span>
                    )}
                  </div>
                  <div className="term-actions">
                    {onEditTerm && (
                      <Tooltip title={t('term.edit', 'Edit Term')}>
                        <IconButton
                          size="small"
                          onClick={(e) => {
                            e.stopPropagation();
                            onEditTerm({ ...term, catalog_type: 'business_term' } as CatalogNode);
                          }}
                        >
                          <EditIcon fontSize="small" />
                        </IconButton>
                      </Tooltip>
                    )}
                    {onDeleteTerm && (
                      <Tooltip title={t('term.delete', 'Delete Term')}>
                        <IconButton
                          size="small"
                          onClick={(e) => {
                            e.stopPropagation();
                            onDeleteTerm({ ...term, catalog_type: 'business_term' } as CatalogNode);
                          }}
                        >
                          <DeleteIcon fontSize="small" />
                        </IconButton>
                      </Tooltip>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>
    );
  };

  const renderFlatView = () => {
    return (
      <div className="business-flat-view">
        {flattenedTerms.map(term => {
          const assetId = `business_term-${term.id}`;
          const isSelected = highlightedItem === assetId;
          const [level1, level2, level3] = getCategoryLevels(term);
          const categories = [level1, level2, level3].filter(Boolean);

          return (
            <div
              key={assetId}
              className={`business-term-item-flat ${isSelected ? 'selected' : ''}`}
            >
              <div 
                className="term-content"
                onClick={() => handleBusinessTermSelect(term)}
              >
                <div className="term-header">
                  <span className="term-name">{term.node_name}</span>
                  {categories.length > 0 && (
                    <span className="term-categories">{categories.join(' > ')}</span>
                  )}
                </div>
                {term.description && (
                  <div className="term-description">{term.description}</div>
                )}
              </div>
              <div className="term-actions">
                {onEditTerm && (
                  <Tooltip title={t('term.edit', 'Edit Term')}>
                    <IconButton
                      size="small"
                      onClick={(e) => {
                        e.stopPropagation();
                        onEditTerm({ ...term, catalog_type: 'business_term' } as CatalogNode);
                      }}
                    >
                      <EditIcon fontSize="small" />
                    </IconButton>
                  </Tooltip>
                )}
                {onDeleteTerm && (
                  <Tooltip title={t('term.delete', 'Delete Term')}>
                    <IconButton
                      size="small"
                      onClick={(e) => {
                        e.stopPropagation();
                        onDeleteTerm({ ...term, catalog_type: 'business_term' } as CatalogNode);
                      }}
                    >
                      <DeleteIcon fontSize="small" />
                    </IconButton>
                  </Tooltip>
                )}
              </div>
            </div>
          );
        })}
      </div>
    );
  };

  // Initialize expanded nodes
  useEffect(() => {
    if (isDefaultExpansion && filteredTree.children && filteredTree.children.length > 0) {
      const initialExpanded = new Set<string>();
      filteredTree.children.forEach(child => {
        initialExpanded.add(child.name);
      });

      setExpandedNodes(prevExpanded => {
        if (prevExpanded.size === initialExpanded.size) {
          const allPresent = [...initialExpanded].every(item => prevExpanded.has(item));
          if (allPresent) {
            return prevExpanded;
          }
        }
        return initialExpanded;
      });
    }
  }, [filteredTree, isDefaultExpansion]);

  useEffect(() => {
    setIsDefaultExpansion(true);
  }, [searchTerm, filterType]);

  useEffect(() => {
    if (highlightedItem) {
      setTimeout(() => {
        const element = document.getElementById(highlightedItem);
        element?.scrollIntoView({ behavior: 'smooth', block: 'center' });
      }, 150);
    }
  }, [highlightedItem]);

  const { t } = useTranslation();

  return (
    <div className="business-terms-tree-container">
      <div className="business-tree-header">
        <div className="tree-controls">
          <button
            className={`view-toggle ${isFlatView ? 'flat' : 'tree'}`}
            onClick={() => setIsFlatView(!isFlatView)}
          >
            {isFlatView ? `🌳 ${t('view.tree', 'Tree View')}` : `📋 ${t('view.flat', 'Flat View')}`}
          </button>
          {!isFlatView && (
            <>
              <Tooltip title={t('view.expand_all', 'Expand All')}>
                <IconButton onClick={handleExpandAll} size="small">
                  <UnfoldMoreIcon />
                </IconButton>
              </Tooltip>
              <Tooltip title={t('view.collapse_all', 'Collapse All')}>
                <IconButton onClick={handleCollapseAll} size="small">
                  <UnfoldLessIcon />
                </IconButton>
              </Tooltip>
            </>
          )}
        </div>
      </div>

      <div className="business-tree-content">
        {isFlatView ? (
          flattenedTerms.length === 0 && searchTerm ? (
            <div className="no-results">
              <div className="no-results-icon">🔍</div>
              <h4>{t('no_results.title', 'No terms found')}</h4>
              <p>{t('no_results.description', 'No business terms match your search')}</p>
            </div>
          ) : (
            renderFlatView()
          )
        ) : (
          filteredTree.children && filteredTree.children.length > 0 ? (
            <div className="business-tree-nodes">
              {filteredTree.children.map(child => renderCategoryNode(child))}
            </div>
          ) : searchTerm ? (
            <div className="no-results">
              <div className="no-results-icon">🔍</div>
              <h4>No categories found</h4>
              <p>No business term categories match your search</p>
            </div>
          ) : (
            <div className="no-results">
              <div className="no-results-icon">📁</div>
              <h4>No Business Terms</h4>
              <p>No business terms are available</p>
            </div>
          )
        )}
      </div>
    </div>
  );
};

export default BusinessTermsTree;