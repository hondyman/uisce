import { registerOperations, type OperationDef } from '../../studio-core/operations/registry';
import { lakehouseStatusApi, type TenantBlock, type ResourceGroup, type ClusterStatus } from './api';

/**
 * Read-only operations behind Platform > Lakehouse status and Tenant menu > My data.
 * The page carries no URLs, no DDL, no mutations — every operation is a query, and the
 * panel surfaces cross-check warnings rather than fix the panel themselves.
 *
 * The domain decides what the row means; the page only binds. See features/system-lakehouse/
 * studio.ts for the writable-page precedent.
 */

const DOMAIN = 'lakehouseStatus';

const operations: OperationDef[] = [
  {
    id: `${DOMAIN}.platform`,
    domain: DOMAIN,
    kind: 'query',
    label: 'Platform lakehouse status',
    description:
      'Cluster health, resource groups, and per-tenant wiring for every tenant in the lakehouse registry. Global admin only.',
    params: [],
    fields: [
      { name: 'generated_at', label: 'Generated at', type: 'datetime' },
      { name: 'cluster_rows', label: 'Cluster rows', type: 'object' },
      { name: 'resource_groups', label: 'Resource groups', type: 'array' },
      { name: 'tenants', label: 'Tenants', type: 'array' },
      { name: 'notes_text', label: 'Notes', type: 'string' },
    ],
    run: async () => {
      const raw = await lakehouseStatusApi.platform();
      return shapePlatform(raw);
    },
  },
  {
    id: `${DOMAIN}.tenant`,
    domain: DOMAIN,
    kind: 'query',
    label: 'Tenant lakehouse status',
    description:
      "The session tenant's lakehouse status row, scoped by AuthContextMiddleware. No cluster or resource-group sections.",
    params: [],
    fields: [
      { name: 'generated_at', label: 'Generated at', type: 'datetime' },
      { name: 'tenant', label: 'Tenant', type: 'object' },
      { name: 'notes_text', label: 'Notes', type: 'string' },
    ],
    run: async () => {
      const raw = await lakehouseStatusApi.tenant();
      return shapeTenant(raw);
    },
  },
];

registerOperations(operations);

// ---- row shaping: keep the page free of business logic ----

interface ClusterRow {
  label: string;
  value: string;
  sub: string;
}

interface PlatformShape {
  generated_at: string;
  /** Flat list of {label, value, sub} rows for the cluster KeyValue widget. */
  cluster_rows: ClusterRow[];
  resource_groups: ResourceGroupRow[];
  tenants: TenantRow[];
  /** Joined notes text for the AlertBanner (multi-line). Empty when no notes. */
  notes_text: string;
}

interface TenantShape {
  generated_at: string;
  tenant: TenantRow;
  notes_text: string;
}

interface ResourceGroupRow {
  name: string;
  cpu_weight?: number;
  mem_limit?: string;
  concurrency_limit?: number;
  big_query_mem_limit?: number;
  /** "tenant_northwinds_svc (1.0), tenant_crd_bakeoff_svc (1.0)" — the classifier routing. */
  classifiers_text: string;
}

interface TenantRow extends TenantBlock {
  /** "DSN configured" | "Legacy pool" | "No DSN" */
  dsn_label: string;
  /** 'success' | 'warning' | 'default' — chip colour. */
  dsn_color: 'success' | 'warning' | 'default';
  /** Init resource group value, or "" if not set. */
  init_rg_text: string;
  /** DB size formatted for display; "—" if unknown, "no DB" if !exists. */
  db_size_text: string;
  /** "3 tables" | "" — table count text. */
  db_table_text: string;
  /** "Audit pipeline not installed" | last_run_status etc. */
  audit_text: string;
  warnings_count: number;
  /** "warning" | "warning" — pluralised for the chip label. */
  warnings_label: string;
}

/** shapePlatform renders the platform payload into the row shapes the page binds to. */
function shapePlatform(raw: Awaited<ReturnType<typeof lakehouseStatusApi.platform>>): PlatformShape {
  return {
    generated_at: raw.generated_at,
    cluster_rows: shapeClusterRows(raw.cluster),
    resource_groups: raw.resource_groups.map(shapeResourceGroup),
    tenants: raw.tenants.map(shapeTenantBlock),
    notes_text: (raw.notes ?? []).join('\n'),
  };
}

/** shapeTenant renders the tenant-scoped payload. The tenant handler strips cluster/RGs. */
function shapeTenant(raw: Awaited<ReturnType<typeof lakehouseStatusApi.tenant>>): TenantShape {
  return {
    generated_at: raw.generated_at,
    tenant: shapeTenantBlock(raw.tenant),
    notes_text: (raw.notes ?? []).join('\n'),
  };
}

/** shapeClusterRows flattens the cluster payload into rows for the KeyValue widget. */
function shapeClusterRows(cluster: ClusterStatus | undefined): ClusterRow[] {
  if (!cluster) return [];
  const rows: ClusterRow[] = [];
  for (const fe of cluster.frontends ?? []) {
    rows.push({
      label: `FE ${fe.name ?? ''}`.trim(),
      value: fe.alive ? 'alive' : 'down',
      sub: [fe.version, fe.host].filter(Boolean).join(' · '),
    });
  }
  for (let i = 0; i < (cluster.backends ?? []).length; i++) {
    const be = cluster.backends[i];
    const pct = typeof be.used_pct === 'number' ? `${(be.used_pct * 100).toFixed(1)}%` : '?';
    rows.push({
      label: `BE ${be.host ?? `#${i}`}`,
      value: be.alive ? 'alive' : 'down',
      sub: `used ${pct}`,
    });
  }
  return rows;
}

/** shapeResourceGroup flattens one resource group into the row shape the DataGrid binds to. */
function shapeResourceGroup(g: ResourceGroup): ResourceGroupRow {
  const classifiers = (g.classifiers ?? [])
    .map((c) => {
      const parts = [c.user, c.role].filter(Boolean).join('/');
      const weight = typeof c.weight === 'number' ? ` (${c.weight})` : '';
      return `${parts}${weight}`;
    })
    .filter((s) => s !== '')
    .join(', ');
  return {
    name: g.name,
    cpu_weight: g.cpu_weight,
    mem_limit: g.mem_limit,
    concurrency_limit: g.concurrency_limit,
    big_query_mem_limit: g.big_query_mem_limit,
    classifiers_text: classifiers || '—',
  };
}

/** shapeTenantBlock adds the derived fields the panel renders (DSN colour, DB text, etc.). */
function shapeTenantBlock(t: TenantBlock): TenantRow {
  const dsnConfigured = t.env?.dsn_configured ?? false;
  const initRg = t.env?.init_resource_group ?? '';
  const db = t.database ?? ({ exists: false } as TenantBlock['database']);
  const audit = t.audit_copy ?? {};
  const warnings = t.warnings ?? [];

  let dsnLabel = 'No DSN';
  let dsnColor: 'success' | 'warning' | 'default' = 'default';
  if (dsnConfigured && initRg) {
    dsnLabel = 'DSN + init rg';
    dsnColor = 'success';
  } else if (dsnConfigured) {
    dsnLabel = 'DSN configured';
    dsnColor = 'success';
  } else if (initRg) {
    dsnLabel = 'init rg only';
    dsnColor = 'warning';
  }

  let dbSize = '—';
  if (db.exists && db.total_bytes) {
    dbSize = db.size_text || `${db.total_bytes} B`;
  } else if (db.exists) {
    dbSize = 'empty';
  } else if (db.name) {
    dbSize = 'no DB';
  }

  const dbTableText = db.exists && db.table_count ? `${db.table_count} ${db.table_count === 1 ? 'table' : 'tables'}` : '';

  // Audit-copy posture. v1 ships "Audit pipeline not installed" until the temporal
  // visibility wiring lands (per the runbook gap 1). The panel intentionally shows
  // this as a placeholder rather than a fake zero — a panel that shows zeros for
  // unmeasured things teaches operators to ignore it.
  let auditText = 'Audit pipeline not installed';
  if (audit.last_run_status) {
    auditText = `${audit.last_run_status}${audit.last_run_time ? ` · ${audit.last_run_time}` : ''}`;
  }

  const count = warnings.length;
  const warningsLabel = count === 0 ? '' : `${count} ${count === 1 ? 'warning' : 'warnings'}`;

  return {
    ...t,
    dsn_label: dsnLabel,
    dsn_color: dsnColor,
    init_rg_text: initRg,
    db_size_text: dbSize,
    db_table_text: dbTableText,
    audit_text: auditText,
    warnings_count: count,
    warnings_label: warningsLabel,
  };
}