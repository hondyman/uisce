
import { useState, useEffect, useMemo } from 'react';
import type { FC } from 'react';
import { devLog, devError } from '../../utils/devLogger';
import { fetchRelatedObjects, RelatedEntity, applyRelationship, unlinkRelationship } from '../../api/relationships';
import RelationshipDetailsPanel from './RelationshipDetailsPanel';
import './RelatedObjectsTab.css';
import RelationshipCard from './RelationshipCard';
import type { RelationshipSuggestion } from '../../services/businessEntitySemanticService';
import { AlertCircle, Search } from 'lucide-react';

interface Relationship extends RelatedEntity {
  isApplied?: boolean;
  isSuggestion?: boolean;
  suggestionData?: RelationshipSuggestion;
}

interface RelationshipsTabProps {
  tenantId: string;
  datasourceId: string;
  entityId: string;
  entityName: string;
  suggestions: RelationshipSuggestion[];
  suggestionsLoading: boolean;
  suggestionsError: Error | null;
  onApplySuggestion: (suggestion: RelationshipSuggestion) => Promise<void>;
}

const RelationshipsTab: FC<RelationshipsTabProps> = ({
  tenantId,
  datasourceId,
  entityId,
  entityName,
  suggestions,
  suggestionsLoading,
  suggestionsError,
  onApplySuggestion,
}) => {
  devLog('RelationshipsTab rendered', { tenantId, datasourceId, entityId, entityName });
  const [relationships, setRelationships] = useState<Relationship[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedObject, setSelectedObject] = useState<{ type: 'node' | 'edge'; data: Relationship } | null>(null);
  const [pendingIds, setPendingIds] = useState<string[]>([]);
  const [toast, setToast] = useState<{ type: 'success' | 'error'; message: string } | null>(null);

  const handleCloseDetails = () => {
    setSelectedObject(null);
  };

  const isPending = (id: string) => pendingIds.includes(id);

  const handleLink = async (rel: Relationship) => {
    if (rel.isSuggestion && rel.suggestionData) {
      setPendingIds((p) => [...p, rel.id]);
      try {
        await onApplySuggestion(rel.suggestionData);
        setRelationships((prevR) => prevR.map((r) => (r.id === rel.id ? { ...r, isApplied: true } : r)));
        setToast({ type: 'success', message: `Accepted suggestion for ${rel.targetEntity}` });
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Failed to apply suggestion';
        setError(message);
        setToast({ type: 'error', message });
        devError('Error applying suggestion', err);
      } finally {
        setPendingIds((p) => p.filter((id) => id !== rel.id));
      }
    } else {
      setPendingIds((p) => [...p, rel.id]);
      const prev = relationships;
      setRelationships((prevR) => prevR.map((r) => (r.id === rel.id ? { ...r, isApplied: true } : r)));

      try {
        const resp = await applyRelationship(
          tenantId,
          datasourceId,
          rel.sourceEntity || entityId,
          rel.targetEntity,
          rel.edgeType || 'entity_relationship',
          rel.cardinality || 'One-to-Many'
        );
        if (!resp || !resp.success) {
          throw new Error(resp?.error || 'Failed to link relationship');
        }
        setToast({ type: 'success', message: `Linked ${rel.targetEntity}` });
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Failed to link relationship';
        setError(message);
        setToast({ type: 'error', message });
        devError('Error linking relationship', err);
        setRelationships(prev);
      } finally {
        setPendingIds((p) => p.filter((id) => id !== rel.id));
      }
    }
  };

  const handleUnlink = async (rel: Relationship) => {
    setPendingIds((p) => [...p, rel.id]);
    const prev = relationships;
    setRelationships((prevR) => prevR.map((r) => (r.id === rel.id ? { ...r, isApplied: false } : r)));

    try {
      const resp = await unlinkRelationship(tenantId, datasourceId, rel.sourceEntity || entityId, rel.targetEntity);
      if (!resp || !resp.success) {
        throw new Error(resp?.error || 'Failed to unlink relationship');
      }
      setToast({ type: 'success', message: `Unlinked ${rel.targetEntity}` });
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to unlink relationship';
      setError(message);
      setToast({ type: 'error', message });
      devError('Error unlinking relationship', err);
      setRelationships(prev);
    } finally {
      setPendingIds((p) => p.filter((id) => id !== rel.id));
    }
  };

  useEffect(() => {
    devLog('RelationshipsTab useEffect triggered', { tenantId, datasourceId, entityId });
    const fetchRelationships = async () => {
      devLog('fetchRelationships called');
      try {
        setLoading(true);
        setError(null);
        devLog('🔗 Fetching relationships for entity:', { entityId, tenantId, datasourceId });
        const entities = await fetchRelatedObjects(tenantId, datasourceId, entityId);
        devLog('✅ Relationships fetched:', entities);
        const deduplicatedMap = new Map<string, Relationship>();
        entities.forEach((entity, index) => {
          const uniqueKey = `${entity.sourceEntity || entityId}|${entity.targetEntity}|${entity.cardinality}`;
          if (!deduplicatedMap.has(uniqueKey)) {
            const uniqueId = entity.id && entity.id !== uniqueKey ? `${entity.id}-${index}` : `rel-${index}-${Date.now()}`;
            deduplicatedMap.set(uniqueKey, {
              ...entity,
              id: uniqueId,
              isApplied: entity.isApplied || false, // Use the isApplied field from API response
              isSuggestion: false,
            } as Relationship);
          }
        });
        setRelationships(Array.from(deduplicatedMap.values()));
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Unknown error';
        devError('Error fetching relationships:', message, err);
        setError(message);
        setRelationships([]);
      } finally {
        setLoading(false);
      }
    };
    if (tenantId && datasourceId && entityId) {
      fetchRelationships();
    } else {
      devLog('Skipping fetchRelationships, missing params', { tenantId, datasourceId, entityId });
    }
  }, [tenantId, datasourceId, entityId]);

  const combinedRelationships = useMemo(() => {
    const existingRels = relationships.map(r => ({...r, isSuggestion: false}));
    const suggestionRels = suggestions.map(s => ({
        id: s.id,
        targetEntity: s.target_entity_id,
        sourceEntity: entityId,
        cardinality: 'One-to-Many', // Default cardinality for suggestions
        description: s.rationale,
        isApplied: false,
        isSuggestion: true,
        suggestionData: s,
    } as Relationship));

    // simple combination, could be improved with de-duplication
    return [...existingRels, ...suggestionRels];
  }, [relationships, suggestions, entityId]);


  const filteredRelationships = combinedRelationships.filter((rel) => {
    if (!searchTerm.trim()) {
      return true;
    }
    const query = searchTerm.toLowerCase();
    return (
      rel.targetEntity.toLowerCase().includes(query) ||
      (rel.description && rel.description.toLowerCase().includes(query)) ||
      rel.cardinality.toLowerCase().includes(query)
    );
  });

  useEffect(() => {
    if (!toast) return;
    const id = setTimeout(() => setToast(null), 3000);
    return () => clearTimeout(id);
  }, [toast]);

  if (loading || suggestionsLoading) {
    return (
      <div className="flex items-center justify-center gap-3 p-12">
        <div className="relationship-spinner"></div>
        <span className="text-slate-700 dark:text-slate-300 font-medium">Loading relationships...</span>
      </div>
    );
  }

  if (error || suggestionsError) {
    return (
      <div className="flex flex-col px-4 py-6 mt-8">
        <div className="flex flex-col items-center gap-6 rounded-xl bg-red-50 dark:bg-red-900/20 p-12 border border-red-200 dark:border-red-900/30">
          <div className="flex h-16 w-16 items-center justify-center rounded-full bg-red-100 dark:bg-red-900/30">
            <AlertCircle className="text-3xl text-red-600 dark:text-red-400" />
          </div>
          <div className="flex max-w-[480px] flex-col items-center gap-2">
            <p className="text-red-900 dark:text-red-200 text-lg font-bold leading-tight tracking-[-0.015em] max-w-[480px] text-center">Could not load relationships</p>
            <p className="text-red-700 dark:text-red-300 text-sm font-normal leading-normal max-w-[480px] text-center">{error || suggestionsError?.message}</p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="w-full">
      <div className="p-4 sm:p-0">
        <div className="flex flex-wrap justify-between gap-4 items-center mb-6">
          <div className="flex min-w-72 flex-col gap-1">
            <p className="text-[#0d141b] dark:text-slate-100 text-4xl font-black leading-tight tracking-[-0.033em]">
              {entityName}: Relationships
            </p>
            <p className="text-slate-500 dark:text-slate-400 text-base font-normal leading-normal">
              Showing {filteredRelationships.length} of {combinedRelationships.length} relationships
            </p>
          </div>
        </div>

        {toast && (
          <div className={`p-3 rounded-md mb-4 flex items-center gap-3 ${toast.type === 'success' ? 'bg-emerald-50 text-emerald-800' : 'bg-red-50 text-red-800'}`}>
            <span className="text-sm font-medium">{toast.message}</span>
          </div>
        )}

        <div className="rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 p-4">
          <div className="flex flex-wrap justify-between items-center gap-4">
            <div className="flex grow items-center gap-4">
              <p className="text-sm font-medium text-slate-800 dark:text-slate-200">
                {combinedRelationships.length} Relationships
              </p>
              <div className="relative">
                <span className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400">
                  <Search className="h-4 w-4" />
                </span>
                <input
                  type="search"
                  value={searchTerm}
                  onChange={(event) => setSearchTerm(event.target.value)}
                  placeholder="Filter relationships..."
                  className="relationship-filter-input"
                />
              </div>
            </div>
          </div>

          <div className="relationship-card-grid grid grid-cols-[repeat(auto-fit,minmax(280px,1fr))] gap-4 p-4">
            {filteredRelationships.map((rel) => (
              <RelationshipCard
                key={rel.id}
                rel={rel}
                appearance={getRelationshipAppearance(rel.cardinality || '')}
                isPending={isPending}
                handleLink={handleLink}
                handleUnlink={handleUnlink}
                setSelectedObject={setSelectedObject}
              />
            ))}
          </div>
        </div>
      </div>
      <RelationshipDetailsPanel
        selectedObject={selectedObject}
        onClose={handleCloseDetails}
        entityName={entityName}
      />
    </div>
  );
};

// Helper for card appearance
const getRelationshipAppearance = (cardinality: string): any => {
    const normalized = (cardinality || '').toLowerCase();
    const normalizedWithHyphen = normalized.replace(/[_\s]+/g, '-');
  
    const isOneToOne =
      normalized.includes('1:1') ||
      normalizedWithHyphen.includes('one-to-one') ||
      normalized.includes('one to one');
  
    const isOneToMany =
      normalized.includes('1:m') ||
      normalizedWithHyphen.includes('one-to-many') ||
      normalized.includes('one to many') ||
      normalized.includes('1-to-many') ||
      normalized.includes('1 to many');
  
    const isManyToOne =
      normalized.includes('m:1') ||
      normalizedWithHyphen.includes('many-to-one') ||
      normalized.includes('many to one');
  
    const isManyToMany =
      normalized.includes('m:m') ||
      normalizedWithHyphen.includes('many-to-many') ||
      normalized.includes('many to many');
  
    if (isOneToOne) {
      return {
        badgeClassName: 'relationship-badge--one-to-one',
        iconName: 'person',
        label: '1:1 Relationship',
        tailwindBadgeClass:
          'inline-flex items-center rounded-full bg-green-100 dark:bg-green-900/50 px-2.5 py-0.5 text-xs font-semibold text-green-800 dark:text-green-300',
      };
    }
  
    if (isOneToMany) {
      return {
        badgeClassName: 'relationship-badge--one-to-many',
        iconName: 'school',
        label: '1:M Relationship',
        tailwindBadgeClass:
          'inline-flex items-center rounded-full bg-orange-100 dark:bg-orange-900/50 px-2.5 py-0.5 text-xs font-semibold text-orange-800 dark:text-orange-300',
      };
    }
  
    if (isManyToOne) {
      return {
        badgeClassName: 'relationship-badge--many-to-one',
        iconName: 'corporate_fare',
        label: 'M:1 Relationship',
        tailwindBadgeClass:
          'inline-flex items-center rounded-full bg-amber-100 dark:bg-amber-900/40 px-2.5 py-0.5 text-xs font-semibold text-amber-800 dark:text-amber-300',
      };
    }
  
    if (isManyToMany) {
      return {
        badgeClassName: 'relationship-badge--many-to-many',
        iconName: 'verified',
        label: 'M:M Relationship',
        tailwindBadgeClass:
          'inline-flex items-center rounded-full bg-green-100 dark:bg-green-900/50 px-2.5 py-0.5 text-xs font-semibold text-green-800 dark:text-green-300',
      };
    }
  
    return {
      badgeClassName: 'relationship-badge--default',
      iconName: 'link',
      label: 'Relationship',
      tailwindBadgeClass:
        'inline-flex items-center rounded-full bg-slate-100 dark:bg-slate-700 px-2.5 py-0.5 text-xs font-semibold text-slate-800 dark:text-slate-300',
    };
  };

export default RelationshipsTab;
