-- Persist the calc engine's resolved execution tier on the calculation
-- itself, rather than recomputing it on every read. This is the "single
-- source of truth" piece of the centralized calc engine consolidation:
-- CreateCalculation/UpdateCalculation now compute tier via
-- boresolver.ValidateFormula (parse + ResolveTier) at save time and store
-- it here, so the execute endpoint dispatches by a stored fact instead of
-- re-deriving it on every request.
--
-- ADD COLUMN IF NOT EXISTS is additive and idempotent -- safe regardless
-- of when/how the calculations table itself was originally created.

ALTER TABLE calculations
    ADD COLUMN IF NOT EXISTS tier TEXT NOT NULL DEFAULT 'pushdown',
    ADD COLUMN IF NOT EXISTS execution_preference TEXT NOT NULL DEFAULT 'auto';

COMMENT ON COLUMN calculations.tier IS
    'Resolved by boresolver.ResolveTier at save time: "pushdown" (compiles to SQL) or "host_runtime" (must run via boresolver.HostRuntimeExecutor, e.g. xirr).';

COMMENT ON COLUMN calculations.execution_preference IS
    'boresolver.ExecutionPreference: "auto" (best available), "pushdown" (require SQL, fail loudly if not possible), or "host_runtime" (always use the Go engine, e.g. for auditability of published/official values).';
