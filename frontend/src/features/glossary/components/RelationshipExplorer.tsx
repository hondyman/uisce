import * as React from 'react';
import { useAccess } from '../../../contexts/AccessContext';
import type { CatalogNode } from '../../../api/glossary';
import {
  useEntityRelationships,
  type EntityType,
  type NormalizedRelationships,
} from '../hooks/useEntityRelationships';
import { RelationshipList } from './RelationshipList';
import { AISuggestionCard } from './AISuggestionCard';

/**
 * Props for the unified RelationshipExplorer.
 *
 * `entityType` discriminates which backend endpoint is hit and which styling
 * applies. `entityId` is the focal catalog node. `focalNode` is passed through
 * so children (AISuggestionCard, CognitiveStudioLauncher) have a stable
 * reference even before the relationships data resolves.
 *
 * `onNavigate` and `onMutated` are escape hatches for the parent explorer
 * (e.g. /core/business-terms) so this component stays a pure renderer.
 */
export interface RelationshipExplorerProps {
  entityType: EntityType;
  entityId: string;
  focalNode: CatalogNode;
  onNavigate?: (id: string, type?: string) => void;
  onMutated?: () => void;
  /**
   * PR 2 placeholder — when true the shell reserves a slot at the top of
   * the Relationships tab for the AISuggestionCard. Default true so
   * consumers can drop the explorer in today and have the slot ready.
   */
  showAISuggestionsStrip?: boolean;
  /**
   * When true, render a button that launches the Cognitive Studio tab
   * (PR 3 will wire it to navigate). Default true.
   */
  showCognitiveButton?: boolean;
  /**
   * Permission predicate. Defaults to "any writer can edit / admin can delete".
   * Pass false to lock the whole explorer read-only.
   */
  canEdit?: boolean;
  canDelete?: boolean;
}

export const RelationshipExplorer: React.FC<RelationshipExplorerProps> = ({
  entityType,
  entityId,
  focalNode,
  onNavigate,
  onMutated,
  showAISuggestionsStrip = true,
  showCognitiveButton = true,
  canEdit,
  canDelete,
}) => {
  const { accessLevel, isPlatformOperator } = useAccess();

  // Permission resolution: writers can edit edges, admin/steward can delete.
  const roleAllowsEdit = canEdit ?? (accessLevel === 'tenant_admin' || accessLevel === 'platform_operator' || isPlatformOperator);
  const roleAllowsDelete = canDelete ?? (accessLevel === 'tenant_admin' || accessLevel === 'platform_operator' || isPlatformOperator);

  const { data, isLoading, error, refetch } = useEntityRelationships(entityType, entityId);

  const handleNavigate = React.useCallback(
    (id: string) => {
      if (onNavigate) onNavigate(id, entityType);
    },
    [onNavigate, entityType]
  );

  const handleMutated = React.useCallback(() => {
    void refetch();
    if (onMutated) onMutated();
  }, [refetch, onMutated]);

  return (
    <div
      data-testid="relationship-explorer"
      data-entity-type={entityType}
      data-entity-id={entityId}
      style={{ display: 'flex', flexDirection: 'column', gap: 16, padding: '16px 24px' }}
    >
      {showAISuggestionsStrip && (
        <div
          data-testid="ai-suggestions-slot"
          style={{
            border: '1px solid rgba(99,102,241,0.25)',
            borderRadius: 10,
            padding: '12px 16px',
            background: 'rgba(99,102,241,0.04)',
          }}
        >
          <AISuggestionCard
            entityType={entityType}
            entityId={entityId}
            focalNode={focalNode}
            onSuggestionApplied={handleMutated}
          />
        </div>
      )}

      {showCognitiveButton && (
        <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
          <button
            type="button"
            onClick={() => handleNavigate(focalNode.id)}
            style={{
              border: '1px solid rgba(168,85,247,0.45)',
              background: 'rgba(168,85,247,0.12)',
              color: '#C084FC',
              padding: '6px 14px',
              borderRadius: 8,
              cursor: 'pointer',
              fontSize: 12,
              fontWeight: 700,
              letterSpacing: '0.04em',
              textTransform: 'uppercase',
            }}
            title="Open in Cognitive Studio"
          >
            🧠 Open in Cognitive Studio
          </button>
        </div>
      )}

      {isLoading && !data && (
        <div data-testid="relationship-explorer-loading" style={{ color: '#8892A4', fontSize: 13, padding: 16 }}>
          Loading relationships…
        </div>
      )}

      {error && (
        <div
          data-testid="relationship-explorer-error"
          style={{
            color: '#EF4444',
            fontSize: 13,
            padding: 12,
            background: 'rgba(239,68,68,0.06)',
            border: '1px solid rgba(239,68,68,0.3)',
            borderRadius: 8,
          }}
        >
          Failed to load relationships: {(error as Error).message}
        </div>
      )}

      {data && (
        <RelationshipList
          edges={data.edges}
          nodes={data.nodes}
          selectedNodeId={focalNode.id}
          darkMode={true}
          canEdit={roleAllowsEdit}
          canDelete={roleAllowsDelete}
          onDeleted={handleMutated}
          onUpdated={handleMutated}
          getNodeName={(id) => data.nodes.find((n) => n.id === id)?.node_name ?? id.substring(0, 8)}
          getNodePath={(id) => data.nodes.find((n) => n.id === id)?.qualified_path}
        />
      )}
    </div>
  );
};

export default RelationshipExplorer;
