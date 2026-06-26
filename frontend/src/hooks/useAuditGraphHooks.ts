import { useQuery, useMutation } from '@tanstack/react-query';
import { graphqlClient } from '@/lib/graphql-client';
import { useAuth } from '@/hooks/useAuth';
import { useCallback } from 'react';

// Types
export interface AuditEvent {
  id: string;
  type: AuditEventType;
  tenantId: string;
  timestamp: Date;
  status?: string;
  artifactType?: string;
  artifactId?: string;
  title?: string;
  semanticContext?: Record<string, any>;
  complianceContext?: Record<string, any>;
  aiNarrative?: Record<string, any>;
  metadata?: Record<string, any>;
}

export interface ChangeSet {
  id: string;
  tenantId: string;
  title: string;
  description: string;
  status: ChangeSetStatus;
  source: ChangeSetSource;
  risk?: RiskLevel;
  createdBy: string;
  createdAt: Date;
  updatedAt: Date;
  approvedBy?: string;
  approvedAt?: Date;
  rejectionReason?: string;
  impactedEntities: ImpactedEntity[];
  history?: ChangeSetEvent[];
}

export interface ImpactedEntity {
  id: string;
  nodeId: string;
  entityType: ImpactedEntityType;
  name?: string;
  qualifiedPath?: string;
}

export interface AIExplanation {
  narrative: string;
  rootCause: string;
  blastRadius: string;
  impactedTenants?: string[];
  recommendedFix: string;
  suggestedChangeSetSummary: string;
  confidence?: number;
}

export enum AuditEventType {
  AUDIT_EVENT = 'AUDIT_EVENT',
  JOB_RUN = 'JOB_RUN',
  DAG_RUN = 'DAG_RUN',
  CHANGESET_EVENT = 'CHANGESET_EVENT',
  COMPLIANCE_EVENT = 'COMPLIANCE_EVENT',
  INCIDENT = 'INCIDENT',
  SEMANTIC_SNAPSHOT = 'SEMANTIC_SNAPSHOT',
  AI_SUGGESTION = 'AI_SUGGESTION',
}

export enum ChangeSetStatus {
  PENDING = 'PENDING',
  APPROVED = 'APPROVED',
  REJECTED = 'REJECTED',
  APPLIED = 'APPLIED',
  FAILED = 'FAILED',
}

export enum ChangeSetSource {
  MANUAL = 'MANUAL',
  AI_PROPOSED = 'AI_PROPOSED',
  SYSTEM = 'SYSTEM',
}

export enum ImpactedEntityType {
  SEMANTIC_TERM = 'SEMANTIC_TERM',
  JOB = 'JOB',
  DAG = 'DAG',
  BUSINESS_TERM = 'BUSINESS_TERM',
  PAGE = 'PAGE',
  API = 'API',
  BO = 'BO',
  DATASET = 'DATASET',
}

export enum RiskLevel {
  LOW = 'LOW',
  MEDIUM = 'MEDIUM',
  HIGH = 'HIGH',
  CRITICAL = 'CRITICAL',
}

// GraphQL Queries
const AUDIT_EVENTS_QUERY = `
  query AuditEvents(
    $tenantIds: [String!]!
    $from: DateTime!
    $to: DateTime!
    $filter: AuditEventsFilter
    $limit: Int
    $offset: Int
  ) {
    auditEvents(
      tenantIds: $tenantIds
      from: $from
      to: $to
      filter: $filter
      limit: $limit
      offset: $offset
    ) {
      id
      type
      tenantId
      timestamp
      status
      artifactType
      artifactId
      title
      semanticContext
      complianceContext
      aiNarrative
      metadata
    }
  }
`;

const CHANGESET_LIST_QUERY = `
  query ChangeSets(
    $filter: ChangeSetFilter
    $pagination: Pagination
    $sort: Sort
  ) {
    changeSets(filter: $filter, pagination: $pagination, sort: $sort) {
      items {
        id
        tenantId
        title
        description
        status
        source
        risk
        createdBy
        createdAt
        updatedAt
        approvedBy
        approvedAt
        rejectionReason
        impactedEntities {
          id
          nodeId
          entityType
          name
          qualifiedPath
        }
      }
      totalCount
    }
  }
`;

const CHANGESET_DETAIL_QUERY = `
  query ChangeSet($id: ID!) {
    changeSet(id: $id) {
      id
      tenantId
      title
      description
      status
      source
      risk
      createdBy
      createdAt
      updatedAt
      approvedBy
      approvedAt
      rejectionReason
      impactedEntities {
        id
        nodeId
        entityType
        name
        qualifiedPath
      }
      history {
        id
        type
        actor
        timestamp
        details
      }
    }
  }
`;

const EXPLAIN_AUDIT_MUTATION = `
  mutation ExplainAudit(
    $tenantIds: [String!]!
    $records: [AuditRecordInput!]!
  ) {
    explainAudit(tenantIds: $tenantIds, records: $records) {
      narrative
      rootCause
      blastRadius
      impactedTenants
      recommendedFix
      suggestedChangeSetSummary
      confidence
    }
  }
`;

const CREATE_CHANGESET_FROM_AI_MUTATION = `
  mutation CreateChangeSetFromAI($input: ChangeSetFromAIInput!) {
    createChangeSetFromAI(input: $input) {
      id
      tenantId
      title
      status
      source
      createdAt
    }
  }
`;

const APPROVE_CHANGESET_MUTATION = `
  mutation ApproveChangeSet($id: ID!) {
    approveChangeSet(id: $id) {
      id
      status
      approvedAt
    }
  }
`;

const REJECT_CHANGESET_MUTATION = `
  mutation RejectChangeSet($id: ID!, $reason: String!) {
    rejectChangeSet(id: $id, reason: $reason) {
      id
      status
      rejectionReason
    }
  }
`;

// ============================================================================
// Hooks
// ============================================================================

/**
 * useAuditEvents - Fetch audit events with filtering and pagination
 */
export function useAuditEvents(
  tenantIds: string[],
  from: Date,
  to: Date,
  filters?: {
    types?: AuditEventType[];
    status?: string[];
    riskLevels?: RiskLevel[];
  },
  pagination = { limit: 50, offset: 0 }
) {
  return useQuery({
    queryKey: ['auditEvents', tenantIds, from, to, filters, pagination],
    queryFn: async () => {
      const response = await graphqlClient.request(AUDIT_EVENTS_QUERY, {
        tenantIds,
        from: from.toISOString(),
        to: to.toISOString(),
        filter: filters,
        limit: pagination.limit,
        offset: pagination.offset,
      });
      return response.auditEvents as AuditEvent[];
    },
    enabled: tenantIds.length > 0,
  });
}

/**
 * useChangeSets - Fetch ChangeSets with filtering and pagination
 */
export function useChangeSets(
  tenantIds?: string[],
  status?: ChangeSetStatus[],
  pagination = { limit: 50, offset: 0 }
) {
  return useQuery({
    queryKey: ['changeSets', tenantIds, status, pagination],
    queryFn: async () => {
      const response = await graphqlClient.request(CHANGESET_LIST_QUERY, {
        filter: {
          tenantIds,
          status,
        },
        pagination,
      });
      return response.changeSets;
    },
  });
}

/**
 * useChangeSet - Fetch a single ChangeSet by ID
 */
export function useChangeSet(id: string) {
  return useQuery({
    queryKey: ['changeSet', id],
    queryFn: async () => {
      const response = await graphqlClient.request(CHANGESET_DETAIL_QUERY, { id });
      return response.changeSet as ChangeSet;
    },
    enabled: !!id,
  });
}

/**
 * useExplainAudit - Trigger AI explanation for audit events
 */
export function useExplainAudit() {
  return useMutation({
    mutationFn: async (params: {
      tenantIds: string[];
      records: Array<{
        id: string;
        type: AuditEventType;
        tenantId: string;
        timestamp: Date;
        status?: string;
        artifactType?: string;
        artifactId?: string;
        semanticContext?: Record<string, any>;
        complianceContext?: Record<string, any>;
        metadata?: Record<string, any>;
      }>;
    }) => {
      const response = await graphqlClient.request(EXPLAIN_AUDIT_MUTATION, {
        tenantIds: params.tenantIds,
        records: params.records.map(r => ({
          ...r,
          timestamp: r.timestamp.toISOString(),
        })),
      });
      return response.explainAudit as AIExplanation;
    },
  });
}

/**
 * useCreateChangeSetFromAI - Create a ChangeSet from AI suggestion
 */
export function useCreateChangeSetFromAI() {
  return useMutation({
    mutationFn: async (params: {
      title: string;
      description: string;
      tenantId: string;
      sourceEventId: string;
      impactedEntities: Array<{
        id: string;
        nodeId: string;
        entityType: ImpactedEntityType;
      }>;
    }) => {
      const response = await graphqlClient.request(CREATE_CHANGESET_FROM_AI_MUTATION, {
        input: params,
      });
      return response.createChangeSetFromAI as ChangeSet;
    },
  });
}

/**
 * useApproveChangeSet - Approve a ChangeSet
 */
export function useApproveChangeSet() {
  return useMutation({
    mutationFn: async (changeSetId: string) => {
      const response = await graphqlClient.request(APPROVE_CHANGESET_MUTATION, {
        id: changeSetId,
      });
      return response.approveChangeSet as ChangeSet;
    },
  });
}

/**
 * useRejectChangeSet - Reject a ChangeSet
 */
export function useRejectChangeSet() {
  return useMutation({
    mutationFn: async (params: { changeSetId: string; reason: string }) => {
      const response = await graphqlClient.request(REJECT_CHANGESET_MUTATION, {
        id: params.changeSetId,
        reason: params.reason,
      });
      return response.rejectChangeSet as ChangeSet;
    },
  });
}

/**
 * useAuditExplainerFlow - High-level hook for "Explain with AI → ChangeSet proposal" flow
 */
export function useAuditExplainerFlow(event: AuditEvent) {
  const { data: tenantIds } = useTenantScope();
  const explainMutation = useExplainAudit();
  const createChangeSetMutation = useCreateChangeSetFromAI();

  const handleExplain = useCallback(async () => {
    const explanation = await explainMutation.mutateAsync({
      tenantIds: tenantIds || [],
      records: [
        {
          id: event.id,
          type: event.type,
          tenantId: event.tenantId,
          timestamp: event.timestamp,
          status: event.status,
          artifactType: event.artifactType,
          artifactId: event.artifactId,
          semanticContext: event.semanticContext,
          complianceContext: event.complianceContext,
          metadata: event.metadata,
        },
      ],
    });
    return explanation;
  }, [event, tenantIds, explainMutation]);

  const handleProposeChangeSet = useCallback(
    async (explanation: AIExplanation, impactedEntities: ImpactedEntity[]) => {
      const changeSet = await createChangeSetMutation.mutateAsync({
        title: explanation.suggestedChangeSetSummary,
        description: explanation.narrative + '\n\nRoot Cause: ' + explanation.rootCause,
        tenantId: event.tenantId,
        sourceEventId: event.id,
        impactedEntities: impactedEntities.map(e => ({
          id: e.id,
          nodeId: e.nodeId,
          entityType: e.entityType,
        })),
      });
      return changeSet;
    },
    [event.tenantId, event.id, createChangeSetMutation]
  );

  return {
    explain: {
      mutate: handleExplain,
      isLoading: explainMutation.isPending,
      error: explainMutation.error,
    },
    proposeChangeSet: {
      mutate: handleProposeChangeSet,
      isLoading: createChangeSetMutation.isPending,
      error: createChangeSetMutation.error,
    },
  };
}

/**
 * useTenantScope - Get allowed tenants from auth context
 */
function useTenantScope() {
  const { user } = useAuth();

  return useQuery({
    queryKey: ['tenantScope', user?.id],
    queryFn: () => {
      // Extract from auth context or localStorage
      const stored = localStorage.getItem('selected_tenant');
      if (stored) {
        const tenant = JSON.parse(stored);
        return [tenant.id];
      }
      return [];
    },
    enabled: !!user,
  });
}

/**
 * useApprovalFlow - High-level hook for ChangeSet approval workflow
 */
export function useApprovalFlow(changeSetId: string) {
  const changeSet = useChangeSet(changeSetId);
  const approveMutation = useApproveChangeSet();
  const rejectMutation = useRejectChangeSet();

  const handleApprove = useCallback(async () => {
    await approveMutation.mutateAsync(changeSetId);
    // Invalidate queries to refresh
    await changeSet.refetch();
  }, [changeSetId, approveMutation, changeSet]);

  const handleReject = useCallback(
    async (reason: string) => {
      await rejectMutation.mutateAsync({
        changeSetId,
        reason,
      });
      // Invalidate queries to refresh
      await changeSet.refetch();
    },
    [changeSetId, rejectMutation, changeSet]
  );

  return {
    changeSet: changeSet.data,
    isLoading: changeSet.isLoading || approveMutation.isPending || rejectMutation.isPending,
    approve: {
      mutate: handleApprove,
      isLoading: approveMutation.isPending,
      error: approveMutation.error,
    },
    reject: {
      mutate: handleReject,
      isLoading: rejectMutation.isPending,
      error: rejectMutation.error,
    },
  };
}
