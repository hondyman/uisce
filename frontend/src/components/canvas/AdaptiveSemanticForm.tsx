import React, { useEffect, useMemo, useState } from 'react';
import type {
  MutationResponse,
  ResolvedCapabilities,
  ResolvedField,
} from '@/types/mutability';
import { dispatchMutationApi } from '@/api/layoutResolver';

export interface AdaptiveSemanticFormProps {
  capabilities: ResolvedCapabilities;
  schemaFields: ResolvedField[];
  initialValues: Record<string, unknown>;
  businessObjectKey: string;
  bindingId?: string;
  businessObjectId?: string;
  tenantId: string;
  userId?: string;
  mutationType?: 'CREATE' | 'UPDATE' | 'DELETE';
  onSuccess?: (resp: MutationResponse) => void;
  onError?: (err: Error) => void;
  className?: string;
}

/**
 * AdaptiveSemanticForm renders a hydration-aware edit form. It locks fields
 * that are not bound to a physical column on the active backend and routes
 * the submission through the CQRS dispatcher when the binding's backend is
 * not directly writeable (Cardinal Rule 1: config-before-code).
 *
 * The component is intentionally backend-agnostic — it never branches on
 * engine_type by name. Every visual decision is driven by the resolved
 * `capabilities` and `schemaFields` payload.
 */
export const AdaptiveSemanticForm: React.FC<AdaptiveSemanticFormProps> = ({
  capabilities,
  schemaFields,
  initialValues,
  businessObjectKey,
  bindingId,
  businessObjectId,
  tenantId,
  userId,
  mutationType = 'UPDATE',
  onSuccess,
  onError,
  className,
}) => {
  const [formState, setFormState] = useState<Record<string, unknown>>(initialValues);
  const [submitting, setSubmitting] = useState(false);
  const [lastResponse, setLastResponse] = useState<MutationResponse | null>(null);

  // Reset state when initialValues change (e.g., navigating between pages).
  useEffect(() => {
    setFormState(initialValues);
  }, [initialValues]);

  const isCqrsRoute =
    capabilities.mutabilityMode === 'ASYNCHRONOUS_CQRS_QUEUE';

  const fieldGrid = useMemo(() => schemaFields, [schemaFields]);

  const handleFieldChange = (key: string, val: unknown) => {
    setFormState((prev) => ({ ...prev, [key]: val }));
  };

  const executeMutation = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setLastResponse(null);
    try {
      const resp = await dispatchMutationApi({
        businessObjectKey,
        businessObjectId,
        bindingId,
        mutationType,
        tenantId,
        userId,
        statePayload: formState,
      });
      setLastResponse(resp);
      if (resp.status === 'success' || resp.status === 'pending') {
        onSuccess?.(resp);
      } else {
        const err = new Error(resp.error || 'mutation failed');
        onError?.(err);
      }
    } catch (err) {
      const e2 = err instanceof Error ? err : new Error(String(err));
      setLastResponse({
        commandId: '',
        correlationId: '',
        route: 'REJECTED',
        status: 'failed',
        timestamp: new Date().toISOString(),
        error: e2.message,
      });
      onError?.(e2);
    } finally {
      setSubmitting(false);
    }
  };

  const isFieldDisabled = (field: ResolvedField): boolean => {
    if (!field.isEditable) return true;
    if (field.hydrationState === 'UNBOUND_FALLBACK_NULL') return true;
    return false;
  };

  return (
    <form
      onSubmit={executeMutation}
      className={
        className ||
        'space-y-4 p-6 bg-slate-950 border border-slate-800 rounded-xl'
      }
    >
      <div className="flex justify-between items-center border-b border-slate-900 pb-3">
        <h4 className="text-xs font-bold font-mono text-slate-200">
          Data Viewport Controller
        </h4>
        <span
          className={`text-[10px] font-mono px-2 py-0.5 rounded border ${
            isCqrsRoute
              ? 'bg-blue-500/10 text-blue-400 border-blue-500/20'
              : 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20'
          }`}
          data-testid="mutability-mode-badge"
        >
          {isCqrsRoute ? 'CQRS Buffered Pipe' : 'Direct Relational Mode'}
        </span>
      </div>

      <div className="grid grid-cols-2 gap-4">
        {fieldGrid.map((field) => {
          const disabled = isFieldDisabled(field);
          const displayValue =
            field.hydrationState === 'UNBOUND_FALLBACK_NULL'
              ? '--'
              : (formState[field.semanticTermKey] as string | undefined) ?? '';
          return (
            <div key={field.semanticTermKey} className="flex flex-col gap-1.5">
              <label className="text-xs font-semibold font-mono text-slate-400 flex justify-between">
                <span>{field.displayLabel}</span>
                {field.hydrationState === 'UNBOUND_FALLBACK_NULL' && (
                  <span className="text-[9px] text-amber-500 font-sans italic">
                    Field Missing from Source
                  </span>
                )}
              </label>
              <input
                type="text"
                disabled={disabled}
                value={displayValue}
                onChange={(e) =>
                  handleFieldChange(field.semanticTermKey, e.target.value)
                }
                className={`p-2 rounded bg-slate-900 border font-mono text-sm transition-all ${
                  disabled
                    ? 'border-slate-800/80 text-slate-600 cursor-not-allowed bg-slate-900/20'
                    : 'border-slate-700 text-slate-100 focus:border-emerald-500'
                }`}
              />
            </div>
          );
        })}
      </div>

      {lastResponse && (
        <div
          className={`text-[10px] font-mono px-3 py-2 rounded border ${
            lastResponse.status === 'success'
              ? 'bg-emerald-500/10 text-emerald-300 border-emerald-500/20'
              : lastResponse.status === 'pending'
                ? 'bg-blue-500/10 text-blue-300 border-blue-500/20'
                : 'bg-red-500/10 text-red-300 border-red-500/20'
          }`}
        >
          {lastResponse.status === 'pending' && lastResponse.topic && (
            <>📤 Queued → {lastResponse.topic}</>
          )}
          {lastResponse.status === 'success' && (
            <>✅ Applied (correlation: {lastResponse.correlationId})</>
          )}
          {lastResponse.status === 'failed' && (
            <>❌ {lastResponse.error || 'mutation failed'}</>
          )}
        </div>
      )}

      <div className="flex justify-end gap-3 mt-6 border-t border-slate-900 pt-4">
        <button
          type="submit"
          disabled={submitting}
          className="px-4 py-2 text-xs font-mono bg-slate-100 text-slate-950 font-bold rounded hover:bg-slate-200 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {submitting ? 'Submitting…' : 'Publish State Mutation'}
        </button>
      </div>
    </form>
  );
};

export default AdaptiveSemanticForm;