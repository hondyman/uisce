/**
 * Registered operations: the only way a Page Studio page reads or changes
 * data outside a Business Object binding. A page names an operation id and
 * supplies params; the domain that owns the API registers what the id
 * means. Pages never carry URLs, so tenant isolation, auth and validation
 * stay where they already are - in the domain's own endpoints - and the
 * studio can list, document and govern everything a page can do.
 */

export interface OperationParam {
  name: string;
  label?: string;
  type: 'string' | 'number' | 'boolean' | 'object';
  required?: boolean;
  description?: string;
}

/** A field a query's rows carry - what the studio offers when binding columns. */
export interface OperationField {
  name: string;
  label?: string;
  type?: 'string' | 'number' | 'boolean' | 'datetime' | 'object' | 'array';
  description?: string;
}

/** What a running operation may report while it works (a long run's stage). */
export interface OperationContext {
  progress: (text: string) => void;
}

export interface OperationDef {
  id: string;
  /** Owning domain; also the cache prefix, so the domain's own invalidations refresh pages. */
  domain: string;
  label: string;
  description?: string;
  kind: 'query' | 'mutation';
  params: OperationParam[];
  run: (params: Record<string, unknown>, ctx?: OperationContext) => Promise<unknown>;
  /** Query only: row fields, for the column editor. */
  fields?: OperationField[];
  /** Mutation only: cache prefixes to refresh after success (default: [domain]). */
  invalidates?: string[][];
}

const operations = new Map<string, OperationDef>();

export function registerOperations(defs: OperationDef[]): void {
  for (const d of defs) operations.set(d.id, d);
}

export function getOperation(id: string): OperationDef | undefined {
  return operations.get(id);
}

export function listOperations(kind?: OperationDef['kind']): OperationDef[] {
  return [...operations.values()].filter((o) => !kind || o.kind === kind).sort((a, b) => a.id.localeCompare(b.id));
}

/** Missing required params, so a query waits instead of calling the API with holes. */
export function missingParams(op: OperationDef, params: Record<string, unknown>): string[] {
  return op.params.filter((p) => p.required && (params[p.name] === undefined || params[p.name] === null || params[p.name] === '')).map((p) => p.name);
}
