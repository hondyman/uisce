import i18n from '../../../i18n';
import type { CorePageDefinition, PageLayout } from '../../../types/pageStudio';
import { getOperation } from '../../../studio-core/operations/registry';
import { getDomainComponent } from '../../../studio-core/components/registry';
import { shapeProblems } from './conditionOps';

/**
 * The page checker: finds what would break a page before anyone uses it -
 * bindings to variables or queries that do not exist, operations nobody
 * registered, required params never set, layout pointing at nothing, an
 * overlay nothing can close, labels that look like translation keys but have
 * no translation, buttons with no label. Pure: it reads the page definition
 * (and the operation / component registries) and returns issues, so the
 * designer shows them and publishing is blocked while any is an error.
 */

export type IssueSeverity = 'error' | 'warning';

export interface PageIssue {
  severity: IssueSeverity;
  /** Stable code, e.g. unknown-variable. */
  code: string;
  message: string;
  /** Where: "component grid › columns › 2 › cell", "query golden", "tab Runs". */
  where: string;
  /** The widget or layout node it belongs to, when there is one (to select it). */
  componentId?: string;
}

/** Settings whose text is shown to users (checked for missing translations). */
const TEXT_KEYS = new Set([
  'label', 'title', 'subtitle', 'text', 'header', 'placeholder', 'emptyText', 'helperText', 'intro', 'emptyLabel',
  'confirmLabel', 'submitLabel', 'doneLabel', 'warningTitle', 'busyText', 'emptyHint', 'keyHeader', 'valueHeader', 'description',
]);
/** An i18n key: dotted words, no spaces (mastering.tabs.golden). */
const KEY_LIKE = /^[a-z][A-Za-z0-9_]*(\.[A-Za-z0-9_]+)+$/;
const TEMPLATE = /\{\{\s*([^}]+?)\s*\}\}/g;

type Visit = (value: unknown, path: (string | number)[]) => void;
function walk(value: unknown, path: (string | number)[], visit: Visit) {
  visit(value, path);
  if (Array.isArray(value)) value.forEach((v, i) => walk(v, [...path, i], visit));
  else if (value && typeof value === 'object') for (const [k, v] of Object.entries(value)) walk(v, [...path, k], visit);
}
const describe = (path: (string | number)[]) => path.join(' › ');

export function checkPage(page: Pick<CorePageDefinition, 'components' | 'layout' | 'tabs' | 'filterBar' | 'app'>): PageIssue[] {
  const issues: PageIssue[] = [];
  const add = (severity: IssueSeverity, code: string, message: string, where: string, componentId?: string) =>
    issues.push({ severity, code, message, where, componentId });

  const app = page.app ?? {};
  const components = page.components ?? {};
  const variables = new Set<string>();
  const queries = new Set<string>();

  // --- declarations: names are unique -------------------------------------------------------
  for (const v of app.variables ?? []) {
    if (variables.has(v.name)) add('error', 'duplicate-variable', `The variable "${v.name}" is declared twice.`, `variable ${v.name}`);
    variables.add(v.name);
    if (v.initFrom && !(app.queries ?? []).some((q) => q.id === v.initFrom!.query)) {
      add('error', 'unknown-query', `"${v.name}" starts from query "${v.initFrom.query}", which the page does not declare.`, `variable ${v.name}`);
    }
  }
  for (const q of app.queries ?? []) {
    if (queries.has(q.id)) add('error', 'duplicate-query', `The query "${q.id}" is declared twice.`, `query ${q.id}`);
    queries.add(q.id);
  }
  if (app.tabVariable && !variables.has(app.tabVariable)) {
    add('error', 'unknown-variable', `The tab variable "${app.tabVariable}" is not declared.`, 'page tabs');
  }

  // --- queries: operations exist and get their required params ----------------------------------
  for (const q of app.queries ?? []) {
    if (q.kind === 'savedQuery') {
      if (!q.savedQueryId) {
        add('error', 'missing-saved-query', `Query "${q.id}" is savedQuery-backed but has no savedQueryId.`, `query ${q.id}`);
      }
      continue;
    }
    const op = getOperation(q.operation);
    if (!op) {
      add('error', 'unknown-operation', `Query "${q.id}" uses operation "${q.operation}", which is not registered.`, `query ${q.id}`);
      continue;
    }
    if (op.kind === 'mutation') add('warning', 'mutation-as-query', `Query "${q.id}" runs "${q.operation}", which changes data; it would run every time the page loads.`, `query ${q.id}`);
    for (const p of op.params.filter((x) => x.required)) {
      if (!(p.name in (q.params ?? {}))) add('error', 'missing-param', `Query "${q.id}" never sets "${p.name}", which "${q.operation}" requires - it will never run.`, `query ${q.id}`);
    }
  }

  // --- every place the page names things --------------------------------------------------------
  const scan = (root: unknown, base: (string | number)[], componentId?: string) => walk(root, base, (value, path) => {
    const where = describe(path);
    const last = path[path.length - 1];
    // Bindings: {{vars.x}}, {{queries.y}}
    if (typeof value === 'string') {
      for (const m of value.matchAll(TEMPLATE)) checkRef(m[1], where, componentId);
      if (typeof last === 'string' && TEXT_KEYS.has(last) && KEY_LIKE.test(value) && !i18n.exists(value)) {
        add('warning', 'missing-translation', `"${value}" looks like a translation key but has no translation; users would see the key.`, where, componentId);
      }
    }
    if (!value || typeof value !== 'object' || Array.isArray(value)) return;
    const o = value as Record<string, unknown>;
    // Conditions read paths too.
    if (o.type === 'condition' && typeof o.field === 'string') {
      checkRef(o.field, where, componentId);
      // The same rules the condition builder shows: a field, an operator the engine has, the value it takes.
      for (const p of shapeProblems(o as unknown as Parameters<typeof shapeProblems>[0])) add(p.severity, 'bad-condition', p.message, where, componentId);
    }
    // Actions.
    if (o.kind === 'setVariable' && typeof o.name === 'string' && !variables.has(o.name)) {
      add('error', 'unknown-variable', `An action sets "${o.name}", which is not a declared variable.`, where, componentId);
    }
    if (o.kind === 'runOperation' && typeof o.operation === 'string') {
      const op = getOperation(o.operation);
      if (!op) add('error', 'unknown-operation', `An action runs "${o.operation}", which is not registered.`, where, componentId);
      else {
        const params = (o.params ?? {}) as Record<string, unknown>;
        for (const p of op.params.filter((x) => x.required)) {
          if (!(p.name in params)) add('error', 'missing-param', `An action runs "${o.operation}" without "${p.name}", which it requires.`, where, componentId);
        }
      }
      if (typeof o.progressVariable === 'string' && !variables.has(o.progressVariable)) {
        add('error', 'unknown-variable', `An action reports progress into "${o.progressVariable}", which is not declared.`, where, componentId);
      }
    }
  });

  function checkRef(expr: string, where: string, componentId?: string) {
    const parts = expr.trim().split('.');
    if (parts[0] === 'vars' && parts[1] && !variables.has(parts[1])) {
      add('error', 'unknown-variable', `"${expr}" reads the variable "${parts[1]}", which is not declared.`, where, componentId);
    }
    if (parts[0] === 'queries' && parts[1] && !queries.has(parts[1])) {
      add('error', 'unknown-query', `"${expr}" reads the query "${parts[1]}", which is not declared.`, where, componentId);
    }
  }

  scan(app.queries, ['query']);
  scan(app.variables, ['variable']);

  // --- widgets --------------------------------------------------------------------------------
  for (const [id, c] of Object.entries(components)) {
    const p = (c.props ?? {}) as Record<string, unknown>;
    scan(c.props, ['component', id], id);
    scan(c.visibleWhen, ['component', id, 'show when'], id);
    for (const key of ['variable', 'selectedVariable']) {
      const v = p[key];
      if (typeof v === 'string' && v && !variables.has(v)) add('error', 'unknown-variable', `This ${c.type} uses the variable "${v}", which is not declared.`, `component ${id}`, id);
    }
    const query = p.query;
    if (typeof query === 'string' && query && !queries.has(query)) {
      add('error', 'unknown-query', `This ${c.type} reads the query "${query}", which is not declared.`, `component ${id}`, id);
    }
    const from = p.optionsFrom as { query?: string } | undefined;
    if (from && typeof from === 'object' && from.query && !queries.has(from.query)) {
      add('error', 'unknown-query', `This ${c.type} takes options from "${from.query}", which is not declared.`, `component ${id}`, id);
    }
    if (c.type === 'DomainComponent' && typeof p.component === 'string' && !getDomainComponent(p.component)) {
      add('error', 'unknown-component', `"${p.component}" is not a registered domain component.`, `component ${id}`, id);
    }
    if (c.type === 'ActionButton' && !p.label && p.variant !== 'chip') {
      add('warning', 'no-label', 'This button has no label; screen readers and users cannot tell what it does.', `component ${id}`, id);
    }
    // Cube consume (PR1a): mirrored subject on props — publish requires numeric pin.
    const subject = p.subject as { kind?: string; cubeId?: string; contractVersion?: unknown } | undefined;
    if (subject && typeof subject === 'object' && subject.kind === 'cube') {
      if (!subject.cubeId || typeof subject.cubeId !== 'string' || !subject.cubeId.trim()) {
        add('error', 'cube-pin-missing-id', 'This cube tile has no cubeId on its mirrored subject.', `component ${id}`, id);
      }
      const ver = subject.contractVersion;
      const numeric = typeof ver === 'number' && ver > 0;
      if (!numeric) {
        add(
          'error',
          'cube-pin-latest',
          'Published pages must pin a numeric cube contractVersion (not "latest").',
          `component ${id}`,
          id,
        );
      }
      if (!p.savedQueryId || typeof p.savedQueryId !== 'string') {
        add(
          'error',
          'cube-pin-no-saved-query',
          'This cube subject has no savedQueryId; v1 cube tiles run only via saved queries.',
          `component ${id}`,
          id,
        );
      }
    }
  }

  // --- layout: references resolve, every widget is placed, overlays can close -------------------
  const trees: { name: string; layout?: PageLayout }[] = [
    ...(page.tabs?.length ? page.tabs.map((t) => ({ name: `tab ${t.label}`, layout: t.layout })) : [{ name: 'page', layout: page.layout }]),
    { name: 'filter bar', layout: page.filterBar },
  ];
  const placed = new Set<string>();
  const setters = new Set<string>();
  walk({ app, components, trees: trees.map((t) => t.layout) }, [], (v) => {
    if (v && typeof v === 'object' && (v as { kind?: string }).kind === 'setVariable') setters.add(String((v as { name?: string }).name));
  });
  for (const t of trees) {
    if (!t.layout) continue;
    const nodes = t.layout.nodes ?? {};
    if (t.layout.root && !nodes[t.layout.root] && !components[t.layout.root]) {
      add('error', 'missing-node', `The ${t.name} starts at "${t.layout.root}", which does not exist.`, t.name);
    }
    for (const [nid, n] of Object.entries(nodes)) {
      scan(n.props, [t.name, nid], nid);
      for (const child of n.children ?? []) {
        if (!nodes[child] && !components[child]) add('error', 'missing-node', `"${nid}" holds "${child}", which is neither a layout node nor a widget.`, `${t.name} › ${nid}`, nid);
        if (components[child]) placed.add(child);
      }
      if (n.type === 'Drawer' || n.type === 'Dialog') {
        const props = (n.props ?? {}) as { openWhen?: unknown; onClose?: unknown[] };
        const opened: string[] = [];
        walk(props.openWhen, [], (v) => {
          if (v && typeof v === 'object' && typeof (v as { field?: unknown }).field === 'string') {
            const f = (v as { field: string }).field.split('.');
            if (f[0] === 'vars' && f[1]) opened.push(f[1]);
          }
        });
        if (!props.openWhen) add('warning', 'never-opens', `This ${n.type} has no Open when, so it never opens.`, `${t.name} › ${nid}`, nid);
        else if (opened.length && !opened.some((v) => setters.has(v))) {
          add('error', 'cannot-close', `Nothing ever changes ${opened.map((v) => `"${v}"`).join(' or ')}, so this ${n.type} can never close (or open).`, `${t.name} › ${nid}`, nid);
        } else if (!(props.onClose ?? []).length) {
          add('warning', 'no-close', `This ${n.type} has no On close actions: the close button and Escape do nothing.`, `${t.name} › ${nid}`, nid);
        }
      }
    }
  }
  for (const id of Object.keys(components)) {
    if (!placed.has(id) && !trees.some((t) => t.layout?.root === id)) add('warning', 'unplaced', `The widget "${id}" is not placed anywhere; it never shows.`, `component ${id}`, id);
  }

  return issues.sort((a, b) => (a.severity === b.severity ? 0 : a.severity === 'error' ? -1 : 1));
}
