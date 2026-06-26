import { useCallback, useEffect, useState } from 'react';
import type {
  Annotation,
  AddAnnotationRequest,
} from '../types/scenarios';

/**
 * API Service for annotation operations
 */
const annotationService = {
  async getAnnotations(simulationId: string): Promise<Annotation[]> {
    const response = await fetch(
      `/api/v1/annotations?simulationId=${encodeURIComponent(simulationId)}`
    );
    if (!response.ok) throw new Error('Failed to fetch annotations');
    return response.json();
  },

  async addAnnotation(request: AddAnnotationRequest): Promise<Annotation> {
    const response = await fetch('/api/v1/annotations', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(request),
    });
    if (!response.ok) throw new Error('Failed to add annotation');
    return response.json();
  },

  async updateAnnotation(annotationId: string, text: string): Promise<Annotation> {
    const response = await fetch(`/api/v1/annotations/${annotationId}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ text }),
    });
    if (!response.ok) throw new Error('Failed to update annotation');
    return response.json();
  },

  async deleteAnnotation(annotationId: string): Promise<void> {
    const response = await fetch(`/api/v1/annotations/${annotationId}`, {
      method: 'DELETE',
    });
    if (!response.ok) throw new Error('Failed to delete annotation');
  },

  async pinAnnotation(annotationId: string, isPinned: boolean): Promise<Annotation> {
    const response = await fetch(`/api/v1/annotations/${annotationId}/pin`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ isPinned }),
    });
    if (!response.ok) throw new Error('Failed to pin annotation');
    return response.json();
  },

  async replyToAnnotation(
    annotationId: string,
    request: AddAnnotationRequest
  ): Promise<Annotation> {
    const response = await fetch(`/api/v1/annotations/${annotationId}/replies`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(request),
    });
    if (!response.ok) throw new Error('Failed to add reply');
    return response.json();
  },
};

/**
 * Hook for managing scenario annotations and comments
 * Handles fetching, adding, updating, deleting, and pinning annotations
 *
 * @param simulationId - ID of the simulation to get annotations for
 * @param enabled - Whether to automatically fetch annotations
 *
 * @example
 * const { annotations, isLoading, error, add, update, delete: deleteAnnotation } =
 *   useScenarioAnnotations(simulationId);
 *
 * // Add a new annotation
 * await add({
 *   simulationId,
 *   userId: currentUser.id,
 *   text: "Interesting result on Tech portfolio",
 *   cellReference: "Tech - Equity Move",
 *   mentions: [userId1, userId2],
 * });
 */
export function useScenarioAnnotations(simulationId: string | null, enabled: boolean = true) {
  const [annotations, setAnnotations] = useState<Annotation[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  // Fetch annotations on mount or when simulationId changes
  useEffect(() => {
    if (!simulationId || !enabled) return;

    setIsLoading(true);
    setError(null);

    annotationService
      .getAnnotations(simulationId)
      .then(setAnnotations)
      .catch(err => {
        const error = err instanceof Error ? err : new Error('Failed to fetch annotations');
        setError(error);
        console.error('Fetch annotations error:', error);
      })
      .finally(() => setIsLoading(false));
  }, [simulationId, enabled]);

  // Add new annotation
  const add = useCallback(
    async (request: AddAnnotationRequest): Promise<Annotation> => {
      try {
        const annotation = await annotationService.addAnnotation(request);
        setAnnotations(prev => [...prev, annotation]);
        return annotation;
      } catch (err) {
        const error = err instanceof Error ? err : new Error('Failed to add annotation');
        setError(error);
        throw error;
      }
    },
    []
  );

  // Update annotation text
  const update = useCallback(async (annotationId: string, text: string): Promise<void> => {
    try {
      const updated = await annotationService.updateAnnotation(annotationId, text);
      setAnnotations(prev =>
        prev.map(a => (a.id === annotationId ? updated : a))
      );
    } catch (err) {
      const error = err instanceof Error ? err : new Error('Failed to update annotation');
      setError(error);
      throw error;
    }
  }, []);

  // Delete annotation
  const remove = useCallback(async (annotationId: string): Promise<void> => {
    try {
      await annotationService.deleteAnnotation(annotationId);
      setAnnotations(prev => prev.filter(a => a.id !== annotationId));
    } catch (err) {
      const error = err instanceof Error ? err : new Error('Failed to delete annotation');
      setError(error);
      throw error;
    }
  }, []);

  // Pin/unpin annotation
  const togglePin = useCallback(async (annotationId: string): Promise<void> => {
    try {
      const annotation = annotations.find(a => a.id === annotationId);
      const updated = await annotationService.pinAnnotation(
        annotationId,
        !(annotation?.isPinned ?? false)
      );
      setAnnotations(prev =>
        prev.map(a => (a.id === annotationId ? updated : a))
      );
    } catch (err) {
      const error = err instanceof Error ? err : new Error('Failed to pin annotation');
      setError(error);
      throw error;
    }
  }, [annotations]);

  // Reply to annotation
  const reply = useCallback(
    async (annotationId: string, request: AddAnnotationRequest): Promise<Annotation> => {
      try {
        const reply = await annotationService.replyToAnnotation(annotationId, request);
        setAnnotations(prev =>
          prev.map(a => (a.id === annotationId ? { ...a, replies: [...(a.replies ?? []), reply] } : a))
        );
        return reply;
      } catch (err) {
        const error = err instanceof Error ? err : new Error('Failed to reply to annotation');
        setError(error);
        throw error;
      }
    },
    []
  );

  // Refresh annotations
  const refresh = useCallback(async (): Promise<void> => {
    if (!simulationId) return;

    setIsLoading(true);
    try {
      const updated = await annotationService.getAnnotations(simulationId);
      setAnnotations(updated);
    } catch (err) {
      const error = err instanceof Error ? err : new Error('Failed to refresh annotations');
      setError(error);
    } finally {
      setIsLoading(false);
    }
  }, [simulationId]);

  return {
    annotations,
    isLoading,
    error,
    add,
    update,
    delete: remove,
    togglePin,
    reply,
    refresh,
  };
}

export type UseScenarioAnnotationsReturn = ReturnType<typeof useScenarioAnnotations>;
