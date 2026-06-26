import { useState, useEffect } from 'react';
import type { FC } from 'react';
import { devLog, devError } from '../../utils/devLogger';
import { fetchRelatedObjects, RelatedEntity, applyRelationship, unlinkRelationship } from '../../api/relationships';
import RelationshipDetailsPanel from './RelationshipDetailsPanel';
import './RelatedObjectsTab.css';
import SVGIcon from './SVGIcon';
import ActionButton from '../ui/ActionButton';
import RelationshipCard from './RelationshipCard';

interface Relationship extends RelatedEntity {
  isApplied?: boolean;
}

interface RelatedObjectsTabProps {
  tenantId: string;
  datasourceId: string;
  entityName: string;
}

type ViewType = 'card' | 'diagram';

type RelationshipAppearance = {
  badgeClassName: string;
  iconName: string;
  label: string;
  tailwindBadgeClass?: string;
};

// CardView component extracted to prevent hook-order issues
interface CardViewProps {
  filteredRelationships: Relationship[];
  relationships: Relationship[];
  isPending: (id: string) => boolean;
  handleLink: (rel: Relationship) => Promise<void>;
  handleUnlink: (rel: Relationship) => Promise<void>;
  setSelectedObject: (obj: { type: 'node' | 'edge'; data: Relationship } | null) => void;
}

const CardView: FC<CardViewProps> = ({
  filteredRelationships,
  relationships,
  isPending,
  handleLink,
  handleUnlink,
  setSelectedObject,
}) => {
  if (filteredRelationships.length === 0) {
    return (
      <div className="relationship-empty-state mt-6">
        <div className="flex max-w-[480px] flex-col items-center gap-2">
          <p className="text-slate-900 dark:text-slate-100 text-lg font-bold leading-tight tracking-[-0.015em] text-center">
            {relationships.length === 0 ? 'No Relationships Found' : 'No Matches for This Filter'}
          </p>
          <p className="text-slate-600 dark:text-slate-400 text-sm font-normal leading-normal text-center">
            {relationships.length === 0
              ? 'There are no related objects configured for this entity. Get started by creating a new relationship.'
              : 'Try adjusting your keywords or clearing the filter to see all relationships.'}
          </p>
        </div>
        <ActionButton variant="primary">
          <span className="truncate">Create Relationship</span>
        </ActionButton>
      </div>
    );
  }

  return (
    <div className="relationship-card-grid grid grid-cols-[repeat(auto-fit,minmax(280px,1fr))] gap-4 p-4">
      {filteredRelationships.map((rel) => {
        const appearance = getRelationshipAppearance(rel.cardinality || '');

        return (
          <RelationshipCard
            key={rel.id}
            rel={rel}
            appearance={appearance}
            isPending={isPending}
            handleLink={handleLink}
            handleUnlink={handleUnlink}
            setSelectedObject={setSelectedObject}
          />
        );
      })}
    </div>
  );
};

// Helper for card appearance
const getRelationshipAppearance = (cardinality: string): RelationshipAppearance => {
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

const DiagramView: FC = () => (
  <div className="text-center py-12 text-slate-500 dark:text-slate-400">Diagram view coming soon!</div>
);

const RelatedObjectsTab: FC<RelatedObjectsTabProps> = ({
  tenantId,
  datasourceId,
  entityName,
}) => {
  const [relationships, setRelationships] = useState<Relationship[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [viewType, setViewType] = useState<ViewType>('card');
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedObject, setSelectedObject] = useState<{ type: 'node' | 'edge'; data: Relationship } | null>(null);
  const [pendingIds, setPendingIds] = useState<string[]>([]);
  const [toast, setToast] = useState<{ type: 'success' | 'error'; message: string } | null>(null);

  const handleCloseDetails = () => {
    setSelectedObject(null);
  };

  const isPending = (id: string) => pendingIds.includes(id);

  const handleLink = async (rel: Relationship) => {
    setPendingIds((p) => [...p, rel.id]);
    const prev = relationships;
    setRelationships((prevR) => prevR.map((r) => (r.id === rel.id ? { ...r, isApplied: true } : r)));

    try {
      const resp = await applyRelationship(
        tenantId,
        datasourceId,
        rel.sourceEntity || entityName,
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
      setRelationships(prev);
    } finally {
      setPendingIds((p) => p.filter((id) => id !== rel.id));
    }
  };

  const handleUnlink = async (rel: Relationship) => {
    setPendingIds((p) => [...p, rel.id]);
    const prev = relationships;
    setRelationships((prevR) => prevR.map((r) => (r.id === rel.id ? { ...r, isApplied: false } : r)));

    try {
      const resp = await unlinkRelationship(tenantId, datasourceId, rel.sourceEntity || entityName, rel.targetEntity);
      if (!resp || !resp.success) {
        throw new Error(resp?.error || 'Failed to unlink relationship');
      }
      setToast({ type: 'success', message: `Unlinked ${rel.targetEntity}` });
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to unlink relationship';
      setError(message);
      setToast({ type: 'error', message });
      setRelationships(prev);
    } finally {
      setPendingIds((p) => p.filter((id) => id !== rel.id));
    }
  };

  useEffect(() => {
    const fetchRelationships = async () => {
      try {
        setLoading(true);
        setError(null);
        devLog('🔗 Fetching relationships for entity:', { entityName, tenantId, datasourceId });
        const entities = await fetchRelatedObjects(tenantId, datasourceId, entityName);
        devLog('✅ Relationships fetched:', entities);
        const deduplicatedMap = new Map<string, Relationship>();
        entities.forEach((entity, index) => {
          const uniqueKey = `${entity.sourceEntity || entityName}|${entity.targetEntity}|${entity.cardinality}`;
          if (!deduplicatedMap.has(uniqueKey)) {
            const uniqueId = entity.id && entity.id !== uniqueKey ? `${entity.id}-${index}` : `rel-${index}-${Date.now()}`;
            deduplicatedMap.set(uniqueKey, {
              ...entity,
              id: uniqueId,
              isApplied: Math.random() > 0.5,
            } as Relationship);
          }
        });
        setRelationships(Array.from(deduplicatedMap.values()));
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Unknown error';
        devError('Error fetching relationships:', message);
        setError(message);
        setRelationships([]);
      } finally {
        setLoading(false);
      }
    };
    if (tenantId && datasourceId && entityName) {
      fetchRelationships();
    }
  }, [tenantId, datasourceId, entityName]);

  const filteredRelationships = relationships.filter((rel) => {
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

  if (loading) {
    return (
      <div className="flex items-center justify-center gap-3 p-12">
        <div className="relationship-spinner"></div>
        <span className="text-slate-700 dark:text-slate-300 font-medium">Loading relationships...</span>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex flex-col px-4 py-6 mt-8">
        <div className="flex flex-col items-center gap-6 rounded-xl bg-red-50 dark:bg-red-900/20 p-12 border border-red-200 dark:border-red-900/30">
          <div className="flex h-16 w-16 items-center justify-center rounded-full bg-red-100 dark:bg-red-900/30">
            <SVGIcon name="error" className="text-3xl text-red-600 dark:text-red-400" ariaLabel="error" />
          </div>
          <div className="flex max-w-[480px] flex-col items-center gap-2">
            <p className="text-red-900 dark:text-red-200 text-lg font-bold leading-tight tracking-[-0.015em] max-w-[480px] text-center">Could not load relationships</p>
            <p className="text-red-700 dark:text-red-300 text-sm font-normal leading-normal max-w-[480px] text-center">{error}</p>
          </div>
          <ActionButton variant="danger" onClick={() => window.location.reload()}>
            <SVGIcon name="refresh" className="mr-2 text-base" ariaLabel="refresh" />
            <span className="truncate">Retry</span>
          </ActionButton>
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
              {entityName}: Related Objects
            </p>
            <p className="text-slate-500 dark:text-slate-400 text-base font-normal leading-normal">
              Showing {filteredRelationships.length} of {relationships.length} relationships
            </p>
          </div>
        </div>

        {/* Toast */}
        {toast && (
          <div className={`p-3 rounded-md mb-4 flex items-center gap-3 ${toast.type === 'success' ? 'bg-emerald-50 text-emerald-800' : 'bg-red-50 text-red-800'}`}>
            <span className="text-sm font-medium">{toast.message}</span>
          </div>
        )}

        <div className="rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 p-4">
          <div className="flex flex-wrap justify-between items-center gap-4">
            <div className="flex grow items-center gap-4">
              <p className="text-sm font-medium text-slate-800 dark:text-slate-200">
                {relationships.length} Relationships
              </p>
              <div className="relative">
                <span className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400">
                  <SVGIcon name="search" className="relationship-input-icon" ariaLabel="search" />
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
            <div className="relationship-toggle-container">
              <button
                type="button"
                className={`relationship-view-toggle ${viewType === 'card' ? 'relationship-view-toggle--active' : ''}`}
                onClick={() => setViewType('card')}
              >
                <SVGIcon name="grid_view" className="" ariaLabel="grid view" />
                <span className="sr-only">Card view</span>
              </button>
              <button
                type="button"
                className={`relationship-view-toggle ${viewType === 'diagram' ? 'relationship-view-toggle--active' : ''}`}
                onClick={() => setViewType('diagram')}
              >
                <SVGIcon name="schema" className="" ariaLabel="diagram view" />
                <span className="sr-only">Diagram view</span>
              </button>
            </div>
          </div>

          {viewType === 'card' ? (
            <CardView
              filteredRelationships={filteredRelationships}
              relationships={relationships}
              isPending={isPending}
              handleLink={handleLink}
              handleUnlink={handleUnlink}
              setSelectedObject={setSelectedObject}
            />
          ) : (
            <DiagramView />
          )}
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

export default RelatedObjectsTab;
