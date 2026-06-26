/**
 * React Hooks for Audit Semantic Graph Queries
 * 
 * These hooks provide convenient access to audit events, incidents, and AI-generated
 * explanations from the semantic graph. All queries are automatically tenant-scoped.
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { gql } from 'graphql-request';

import type { GraphQLClient } from 'graphql-request';
let graphqlClient: GraphQLClient | null = null; // Injected by app context

export function setGraphQLClient(client: GraphQLClient) {
  graphqlClient = client;
}

function ensureGraphQLClient() {
  if (!graphqlClient) {
    throw new Error(
      'GraphQL client has not been initialized. Call setGraphQLClient() in app context.'
    );
  }
  return graphqlClient;
}

/**
 * Query audit events with filters
 * Returns all events matching the criteria across allowed tenants
 */
export function useAuditEvents(filters: {
  tenantIds: string[];
  types?: string[];
  statuses?: string[];
  severities?: string[];
  from: Date;
  to: Date;
  limit?: number;
  offset?: number;
}) {
  const query = gql`
    query AuditEvents(
      $tenantIds: [String!]!
      $types: [String!]
      $statuses: [String!]
      $severities: [String!]
      $from: DateTime!
      $to: DateTime!
      $limit: Int
      $offset: Int
    ) {
      auditEvents(
        filter: {
          tenantIds: $tenantIds
          types: $types
          statuses: $statuses
          severities: $severities
          from: $from
          to: $to
          limit: $limit
          offset: $offset
        }
      ) {
        id
        type
        timestamp
        status
        severity
        errorMessage
        properties
        tenantId
        catalogNodeId
        relatedEntity {
          type
          id
          name
        }
        aiNarratives {
          id
          type
          narrative
          confidence
          generatedBy
          generatedAt
        }
      }
    }
  `;

  return useQuery({
    queryKey: ['auditEvents', filters],
    queryFn: () =>
      ensureGraphQLClient().request(query, {
        tenantIds: filters.tenantIds,
        types: filters.types,
        statuses: filters.statuses,
        severities: filters.severities,
        from: filters.from.toISOString(),
        to: filters.to.toISOString(),
        limit: filters.limit || 100,
        offset: filters.offset || 0,
      }),
    staleTime: 30000, // 30 seconds
    gcTime: 5 * 60 * 1000, // 5 minutes
  });
}

/**
 * Get complete audit timeline for an entity
 * Shows all events that directly or indirectly affected the entity
 */
export function useEntityAudit(
  entityType: string,
  entityId: string,
  filters: {
    tenantIds: string[];
    from: Date;
    to: Date;
  }
) {
  const query = gql`
    query EntityAudit(
      $entityType: String!
      $entityId: String!
      $tenantIds: [String!]!
      $from: DateTime!
      $to: DateTime!
    ) {
      entityAudit(
        filter: {
          entityType: $entityType
          entityId: $entityId
          tenantIds: $tenantIds
          from: $from
          to: $to
        }
      ) {
        entity {
          type
          id
          name
        }
        events {
          id
          type
          timestamp
          status
          severity
          errorMessage
          properties
        }
        summary {
          totalEvents
          eventsByType {
            type
            count
          }
          eventsByStatus {
            status
            count
          }
          lastEvent
          firstEvent
        }
        impactAnalysis {
          affectedTerms
          riskScore
          downstreamEntities {
            type
            id
            name
          }
          relatedIncidents {
            id
            title
            severity
            status
          }
        }
      }
    }
  `;

  return useQuery({
    queryKey: ['entityAudit', entityType, entityId, filters],
    queryFn: () =>
      ensureGraphQLClient().request(query, {
        entityType,
        entityId,
        tenantIds: filters.tenantIds,
        from: filters.from.toISOString(),
        to: filters.to.toISOString(),
      }),
    staleTime: 60000, // 1 minute
  });
}

/**
 * Query incidents with root cause and impact information
 */
export function useIncidents(filters: {
  tenantIds: string[];
  statuses?: string[];
  severities?: string[];
  from: Date;
  to: Date;
  limit?: number;
  offset?: number;
}) {
  const query = gql`
    query Incidents(
      $tenantIds: [String!]!
      $statuses: [String!]
      $severities: [String!]
      $from: DateTime!
      $to: DateTime!
      $limit: Int
      $offset: Int
    ) {
      incidents(
        filter: {
          tenantIds: $tenantIds
          statuses: $statuses
          severities: $severities
          from: $from
          to: $to
          limit: $limit
          offset: $offset
        }
      ) {
        id
        title
        description
        status
        severity
        detectedAt
        resolvedAt
        blastRadius
        affectedTerms
        recommendedActions
        rootCauseEvents {
          id
          type
          timestamp
          status
          errorMessage
        }
        rootCauseAnalysis {
          narrative
          confidence
          generatedBy
        }
      }
    }
  `;

  return useQuery({
    queryKey: ['incidents', filters],
    queryFn: () =>
      ensureGraphQLClient().request(query, {
        tenantIds: filters.tenantIds,
        statuses: filters.statuses,
        severities: filters.severities,
        from: filters.from.toISOString(),
        to: filters.to.toISOString(),
        limit: filters.limit || 50,
        offset: filters.offset || 0,
      }),
    staleTime: 30000,
  });
}

/**
 * Get AI-powered explanation for an audit event or incident
 */
export function useExplainAudit(
  entityId: string,
  entityType: string,
  tenantIds: string[]
) {
  const query = gql`
    query ExplainAudit(
      $entityId: String!
      $entityType: String!
      $tenantIds: [String!]!
    ) {
      explainAudit(
        request: {
          entityId: $entityId
          entityType: $entityType
          tenantIds: $tenantIds
        }
      ) {
        whatHappened
        rootCause
        severity
        blastRadius
        confidence
        recommendedActions
        proposedChangeSet
        affectedEntities {
          type
          id
          name
        }
        relatedEvents {
          id
          type
          timestamp
          status
        }
      }
    }
  `;

  return useQuery({
    queryKey: ['explainAudit', entityId, entityType, tenantIds],
    queryFn: () =>
      ensureGraphQLClient().request(query, {
        entityId,
        entityType,
        tenantIds,
      }),
    enabled: !!entityId && !!entityType,
    staleTime: 120000, // 2 minutes (AI responses are relatively stable)
  });
}

/**
 * Analyze the impact of a ChangeSet on downstream entities
 */
export function useChangeSetImpact(changeSetId: string, tenantIds: string[]) {
  const query = gql`
    query ChangeSetImpact(
      $changeSetId: String!
      $tenantIds: [String!]!
    ) {
      analyzeChangeSetImpact(
        changeSetId: $changeSetId
        tenantIds: $tenantIds
      ) {
        changeSetId
        summary
        riskScore
        affectedTerms
        downstreamImpacts {
          type
          id
          name
        }
        potentialIncidents {
          id
          title
          severity
          status
        }
      }
    }
  `;

  return useQuery({
    queryKey: ['changeSetImpact', changeSetId, tenantIds],
    queryFn: () =>
      ensureGraphQLClient().request(query, {
        changeSetId,
        tenantIds,
      }),
    enabled: !!changeSetId,
  });
}

/**
 * Get compliance status and violations
 */
export function useComplianceStatus(
  tenantIds: string[],
  from: Date,
  to: Date
) {
  const query = gql`
    query ComplianceStatus(
      $tenantIds: [String!]!
      $from: DateTime!
      $to: DateTime!
    ) {
      complianceStatus(
        tenantIds: $tenantIds
        from: $from
        to: $to
      ) {
        totalChecks
        violations
        passing
        critical
        high
        lastViolation
        affectedTerms
      }
    }
  `;

  return useQuery({
    queryKey: ['complianceStatus', tenantIds, from, to],
    queryFn: () =>
      ensureGraphQLClient().request(query, {
        tenantIds,
        from: from.toISOString(),
        to: to.toISOString(),
      }),
    staleTime: 60000,
  });
}

/**
 * Get critical events in the last N hours (for real-time dashboard)
 */
export function useCriticalEventsRealtime(tenantIds: string[], hoursBack = 1) {
  const query = gql`
    query CriticalEventsRealtime(
      $tenantIds: [String!]!
      $hoursBack: Int!
    ) {
      criticalEventsRealtime(
        tenantIds: $tenantIds
        hoursBack: $hoursBack
      ) {
        id
        type
        timestamp
        status
        severity
        errorMessage
        properties
      }
    }
  `;

  return useQuery({
    queryKey: ['criticalEventsRealtime', tenantIds, hoursBack],
    queryFn: () =>
      ensureGraphQLClient().request(query, {
        tenantIds,
        hoursBack,
      }),
    staleTime: 10000, // 10 seconds (real-time)
    refetchInterval: 10000, // Poll every 10 seconds
  });
}

/**
 * Get event statistics for dashboard
 */
export function useAuditEventStats(
  tenantIds: string[],
  from: Date,
  to: Date
) {
  const query = gql`
    query AuditEventStats(
      $tenantIds: [String!]!
      $from: DateTime!
      $to: DateTime!
    ) {
      auditEventStats(
        tenantIds: $tenantIds
        from: $from
        to: $to
      ) {
        totalEvents
        eventsByType {
          type
          count
        }
        eventsByStatus {
          status
          count
        }
        eventsBySeverity {
          severity
          count
        }
        topImpactedEntities {
          type
          id
          eventCount
          lastEvent
        }
        incidentCount
        violationCount
        avgDurationMs
        criticalCount
      }
    }
  `;

  return useQuery({
    queryKey: ['auditEventStats', tenantIds, from, to],
    queryFn: () =>
      ensureGraphQLClient().request(query, {
        tenantIds,
        from: from.toISOString(),
        to: to.toISOString(),
      }),
    staleTime: 60000,
  });
}

/**
 * Create a manual audit event (admin only)
 */
export function useCreateAuditEvent() {
  const queryClient = useQueryClient();
  const mutation = gql`
    mutation CreateAuditEvent(
      $type: String!
      $tenantIds: [String!]!
      $properties: JSON!
    ) {
      createAuditEvent(
        type: $type
        tenantIds: $tenantIds
        properties: $properties
      ) {
        id
        type
        timestamp
        properties
      }
    }
  `;

  return useMutation({
    mutationFn: (params: { type: string; tenantIds: string[]; properties: any }) =>
      ensureGraphQLClient().request(mutation, params),
    onSuccess: () => {
      // Invalidate affected queries
      queryClient.invalidateQueries({ queryKey: ['auditEvents'] });
      queryClient.invalidateQueries({ queryKey: ['auditEventStats'] });
    },
  });
}

/**
 * Create a manual incident (admin only)
 */
export function useCreateIncident() {
  const queryClient = useQueryClient();
  const mutation = gql`
    mutation CreateIncident(
      $title: String!
      $description: String!
      $severity: String!
      $tenantIds: [String!]!
      $rootCauseEventIds: [String!]
      $affectedTerms: [String!]
    ) {
      createIncident(
        title: $title
        description: $description
        severity: $severity
        tenantIds: $tenantIds
        rootCauseEventIds: $rootCauseEventIds
        affectedTerms: $affectedTerms
      ) {
        id
        title
        severity
        status
      }
    }
  `;

  return useMutation({
    mutationFn: (params: {
      title: string;
      description: string;
      severity: string;
      tenantIds: string[];
      rootCauseEventIds?: string[];
      affectedTerms?: string[];
    }) => ensureGraphQLClient().request(mutation, params),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['incidents'] });
      queryClient.invalidateQueries({ queryKey: ['criticalEventsRealtime'] });
    },
  });
}

/**
 * Compound hook for common dashboard use case
 * Fetches all data needed for a complete audit dashboard
 */
export function useAuditDashboard(
  tenantIds: string[],
  dateRange: { from: Date; to: Date },
  options = { enableRealtime: true }
) {
  const eventsQuery = useAuditEventStats(tenantIds, dateRange.from, dateRange.to);
  const incidentsQuery = useIncidents({
    tenantIds,
    from: dateRange.from,
    to: dateRange.to,
    severities: ['CRITICAL', 'HIGH'],
  });
  const complianceQuery = useComplianceStatus(tenantIds, dateRange.from, dateRange.to);
  const realtimeQuery = options.enableRealtime
    ? useCriticalEventsRealtime(tenantIds, 1)
    : null;

  return {
    stats: eventsQuery,
    incidents: incidentsQuery,
    compliance: complianceQuery,
    realtime: realtimeQuery,
    isLoading:
      eventsQuery.isLoading ||
      incidentsQuery.isLoading ||
      complianceQuery.isLoading ||
      (realtimeQuery?.isLoading ?? false),
    error:
      eventsQuery.error ||
      incidentsQuery.error ||
      complianceQuery.error ||
      realtimeQuery?.error,
  };
}
