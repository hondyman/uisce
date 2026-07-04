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
  UnfoldLessOutlined as UnfoldLessIcon,
} from '@mui/icons-material';
import './BusinessTermsTree.css';
import { useTranslation } from 'react-i18next';
import { devDebug } from '../utils/devLogger';

interface BusinessTermsTreeProps {
  businessTerms: any[];
  semanticTerms?: any[];          // accepted but no longer rendered here (semantics live in SemanticTermsTab)
  semanticViews?: any[];
  semanticEdges?: any[];
  selectedAsset: EnhancedSelectedAsset | null;
  onAssetSelect: (asset: EnhancedSelectedAsset) => void;
  searchTerm?: string;
  highlightedItem: string | null;
  onEditTerm?: (term: CatalogNode) => void;
  onDeleteTerm?: (term: CatalogNode) => void;
  onEditSemanticTerm?: (term: CatalogNode) => void;
  onDeleteSemanticTerm?: (term: CatalogNode) => void;
  filterType?: 'all' | 'with_relationships' | 'without_relationships';
  tenantId?: string;
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
  semanticTerms: _semanticTerms = [],            // accepted for prop compatibility but not rendered
  semanticEdges = [],
  onAssetSelect,
  highlightedItem,
  searchTerm = '',
  onEditTerm,
  onDeleteTerm,
  onEditSemanticTerm: _onEditSemanticTerm,       // accepted for prop compatibility but not rendered
  onDeleteSemanticTerm: _onDeleteSemanticTerm,   // accepted for prop compatibility but not rendered
  filterType = 'all',
  tenantId: tenantIdProp,
}) => {
  const [isFlatView, setIsFlatView] = useState(false);
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(() => new Set());

  // Create a map of all available business term nodes for lookup name resolution.
  // We only resolve against business terms here; semantic terms belong to the SemanticTermsTab.
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
  const effectiveTenantId = tenantIdProp || tenant?.id || '';
  const { data: nodeTypes } = useNodeTypes({ tenantId: effectiveTenantId });
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
      });
    }
  }, [businessTermNodeType, topLevelLookupMaps]);

  const getCategoryLevels = useCallback((term: SemanticAsset): [string, string, string] => {
    const props = term.properties || {};
    const uuidRegex = /^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$/;

    const resolveValue = (propKeyCandidates: string[], val: any, _level: string): string => {
      if (!val) return '';
      const strVal = String(val);
      if (!uuidRegex.test(strVal)) {
        return strVal;
      }

      for (const key of propKeyCandidates) {
        if (topLevelLookupMaps?.[key]?.has(strVal)) {
          const mapped = topLevelLookupMaps[key].get(strVal);
          if (mapped) return mapped;
        }
      }

      const fromNodeMap = nodeNameMap.get(strVal);
      if (fromNodeMap) return fromNodeMap;

      const propDef = (businessTermNodeType?.properties as any[])?.find((p: any) => propKeyCandidates.includes(p.name));
      if (propDef?.cascade_from && propDef?.lookup_id) {
        const parentProperty = propDef.cascade_from;
        const parentVal = props[parentProperty];
        if (parentVal) {
          const cacheKey = `${propDef.lookup_id}_${parentVal}`;
          if (cascadingLookupCache.has(cacheKey)) {
            const cachedMap = cascadingLookupCache.get(cacheKey);
            const mapped = cachedMap?.get(strVal);
            if (mapped) return mapped;
          }
        }
      }

      return `Unknown (${strVal.substring(0, 8)}...)`;
    };

    const level1Candidates = ['category_1', 'category1', 'category_level_1', 'category'];
    const level2Candidates = ['category_2', 'category2', 'category_level_2', 'sub_category'];
    const level3Candidates = ['category_3', 'category3', 'category_level_3'];

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

  useEffect(() => {
    if (!businessTermNodeType?.properties || !filteredBusinessTerms?.length || !effectiveTenantId) return;

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
            const url = `/api/lookups/${p.lookup_id}/values?tenant_id=${effectiveTenantId}&parent_id=${encodeURIComponent(String(parentVal))}`;
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
  }, [businessTermNodeType, filteredBusinessTerms, effectiveTenantId, cascadingLookupCache]);

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
        level1Node.terms.push(term);
      } else if (level2 && !level3) {
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

  const countAllTerms = useCallback((node: CategoryNode): number => {
    let count = node.terms.length;
    if (node.children) {
      node.children.forEach(child => {
        count += countAllTerms(child);
      });
    }
    return count;
  }, []);

  const filteredTree = useMemo(() => {
    if (!searchTerm.trim()) {
      return categoryTree;
    }

    const searchLower = searchTerm.toLowerCase();

    const filterNode = (node: CategoryNode): CategoryNode | null => {
      const matchingTerms = node.terms.filter(term =>
        term.node_name.toLowerCase().includes(searchLower) ||
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

  // Flat view: only business terms (semantic terms belong to SemanticTermsTab).
  const flattenedBusinessTerms = useMemo(() => {
    if (!searchTerm.trim()) {
      return filteredBusinessTerms || [];
    }

    const searchLower = searchTerm.toLowerCase();
    return (filteredBusinessTerms || []).filter(term =>
      term.node_name.toLowerCase().includes(searchLower) ||
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
    setExpandedNodes(new Set());
  };

  const renderCategoryNode = (node: CategoryNode, depth = 0, path = ''): React.ReactNode => {
    const nodePath = path ? `${path}-${node.name}` : node.name;
    const isExpanded = expandedNodes.has(nodePath);
    const hasChildren = node.children && node.children.length > 0;
    const hasTerms = node.terms.length > 0;
    const isExpandable = hasChildren || hasTerms;

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
        {flattenedBusinessTerms.map(term => {
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

  // Auto-expand BUSINESS term categories on first render of each new tree.
  // To avoid the "Maximum update depth" infinite loop, we compute the desired
  // Set via useMemo (stable reference until filteredTree.children changes)
  // and the updater function bails out when prev already equals desired.
  const desiredExpansion = useMemo(() => {
    const desired = new Set<string>();
    const children = filteredTree.children;
    if (children) {
      for (const child of children) {
        desired.add(child.name);
      }
    }
    return desired;
  }, [filteredTree.children]);

  useEffect(() => {
    setExpandedNodes(prev => {
      if (prev.size === desiredExpansion.size) {
        let allPresent = true;
        desiredExpansion.forEach(k => { if (!prev.has(k)) allPresent = false; });
        if (allPresent) return prev;
      }
      return new Set(desiredExpansion);
    });
  }, [desiredExpansion]);

  useEffect(() => {
    if (highlightedItem) {
      setTimeout(() => {
        const element = document.getElementById(highlightedItem);
        element?.scrollIntoView({ behavior: 'smooth', block: 'center' });
      }, 150);
    }
  }, [highlightedItem]);

  const { t } = useTranslation();

  const noBusinessCategories = !filteredTree.children || filteredTree.children.length === 0;
  devDebug('[BusinessTermsTree] render — businessTerms:', filteredBusinessTerms?.length, 'isFlatView:', isFlatView, 'categories:', filteredTree.children?.length ?? 0);

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
          flattenedBusinessTerms.length === 0 && searchTerm ? (
            <div className="no-results">
              <div className="no-results-icon">🔍</div>
              <h4>{t('no_results.title', 'No terms found')}</h4>
              <p>{t('no_results.description', 'No business terms match your search')}</p>
            </div>
          ) : flattenedBusinessTerms.length === 0 ? (
            <div className="no-results">
              <div className="no-results-icon">📁</div>
              <h4>{t('no_terms.title', 'No Business Terms')}</h4>
              <p>{t('no_terms.description', 'No business terms are available')}</p>
            </div>
          ) : (
            renderFlatView()
          )
        ) : noBusinessCategories && !searchTerm ? (
          <div className="no-results">
            <div className="no-results-icon">📁</div>
            <h4>{t('no_terms.title', 'No Business Terms')}</h4>
            <p>{t('no_terms.description', 'No business terms are available')}</p>
          </div>
        ) : noBusinessCategories && searchTerm ? (
          <div className="no-results">
            <div className="no-results-icon">🔍</div>
            <h4>{t('no_results.title', 'No terms found')}</h4>
            <p>{t('no_results.description', 'No business terms match your search')}</p>
          </div>
        ) : (
          <div className="business-tree-nodes">
            {filteredTree.children?.map(child => renderCategoryNode(child))}
          </div>
        )}
      </div>
    </div>
  );
};

export default BusinessTermsTree;
