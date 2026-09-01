import { useCallback } from 'react';
import { ExplorerQueryDefinition } from '../types/explorerTypes';
import { apiFetch } from '../../../lib/apiClient';
import { devWarn } from '../../../utils/devLogger';

interface TelemetryOptions {
  tenantId?: string;
  userId?: string;
  userRole?: string;
}

export function useQueryTelemetry(options: TelemetryOptions = {}) {
  const { tenantId = 'default', userId = 'usr_analyst', userRole = 'Senior Portfolio Manager' } = options;

  const logInteraction = useCallback(
    async (params: {
      prompt: string;
      generatedQuery: ExplorerQueryDefinition;
      executedQuery: ExplorerQueryDefinition;
      wasEdited: boolean;
      wasSaved?: boolean;
      wasExported?: boolean;
      clonedToReport?: boolean;
      rating?: number;
      feedbackNotes?: string;
      executionDurationMs?: number;
    }) => {
      try {
        await apiFetch('/api/v1/ai/telemetry', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            tenantId,
            userId,
            userRole,
            prompt: params.prompt,
            generatedQuery: params.generatedQuery,
            executedQuery: params.executedQuery,
            wasEdited: params.wasEdited,
            wasSaved: params.wasSaved || false,
            wasExported: params.wasExported || false,
            clonedToReport: params.clonedToReport || false,
            rating: params.rating || 0,
            feedbackNotes: params.feedbackNotes || '',
            executionDurationMs: params.executionDurationMs || 0,
          }),
        });
      } catch (err) {
        devWarn('Asynchronous query telemetry logging failed:', err);
      }
    },
    [tenantId, userId, userRole]
  );

  return { logInteraction };
}

export default useQueryTelemetry;
