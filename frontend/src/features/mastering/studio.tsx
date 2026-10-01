import i18n from '../../i18n';
import { fmt } from '../schedules/api';
import { registerOperations, type OperationDef } from '../../studio-core/operations/registry';
import { stagingBindingsApi } from '../staging-bindings/api';
import { masteringApi, isSeries, pct, showValue, type Counts, type Override, type Policy } from './api';

/**
 * The mastering domain's Page Studio surface: the operations a page may
 * call. Row-level derivations (who may vote, what a request says) live here - the domain decides, the page
 * displays - so pages stay declarative and the rules stay with the API.
 */

const t = (k: string, o?: Record<string, unknown>) => i18n.t(k, o) as string;
const str = (p: Record<string, unknown>, k: string) => (p[k] === undefined || p[k] === null ? '' : String(p[k]));

/** How the policy reads in plain words. */
function policyText(p?: Policy) {
  if (!p) return '';
  if (p.mode === 'DIRECT') return t('mastering.policy.directShort');
  return t('mastering.policy.approvalShort', { count: p.approvals_required });
}

/** Whether the viewer can still act on a request: their own, already voted, can vote. */
const requestState = (r: { mine?: boolean; voted?: boolean }) => (r.mine ? 'mine' : r.voted ? 'voted' : 'can_vote');

function overrideRow(o: Override) {
  const statusView = o.status === 'APPLIED' && o.action === 'SET' ? (o.active ? 'ACTIVE' : 'ENDED') : o.status;
  return {
    ...o,
    state: o.status === 'PENDING' ? requestState(o) : 'closed',
    record_label: o.golden_code ?? o.golden_id.slice(0, 8),
    open_ref: o.open_id ?? o.golden_id,
    status_view: statusView,
    status_label: statusView === 'ACTIVE' ? t('mastering.overrides.active') : statusView === 'ENDED' ? t('mastering.overrides.ended') : t(`mastering.overrides.status.${o.status}`),
    approvals_text: o.mode === 'DIRECT' ? t('mastering.overrides.direct') : t('mastering.overrides.approvals', { n: o.approvals, of: o.approvals_required }),
    before_text: showValue(o.previous_value),
    after_text: showValue(o.value),
  };
}

const E = { name: 'entity', type: 'string' as const, required: true, label: 'Entity' };

const operations: OperationDef[] = [
  {
    id: 'mastering.profiles', domain: 'mastering', kind: 'query', label: 'Mastered entities',
    description: 'Every entity with a mastering profile, in display order.', params: [],
    fields: [{ name: 'entity', type: 'string' }, { name: 'display_name', type: 'string' }, { name: 'series', type: 'boolean' }, { name: 'bo_key', type: 'string' }],
    run: async () => (await masteringApi.profiles()).profiles.map((p) => ({ ...p, entity: p.entity_cd.toLowerCase(), series: isSeries(p) })),
  },
  {
    id: 'mastering.profile', domain: 'mastering', kind: 'query', label: 'One mastered entity',
    description: 'The chosen entity\'s profile; series is true for time series (prices).', params: [E],
    fields: [{ name: 'display_name' }, { name: 'series', type: 'boolean' }, { name: 'bo_key' }, { name: 'kind' }],
    run: async (p) => {
      const found = (await masteringApi.profiles()).profiles.find((x) => x.entity_cd.toLowerCase() === str(p, 'entity'));
      return found ? { ...found, entity: found.entity_cd.toLowerCase(), series: isSeries(found) } : null;
    },
  },
  {
    id: 'mastering.policy', domain: 'mastering', kind: 'query', label: 'Override policy',
    description: 'The entity\'s steward policy; label reads it in plain words.', params: [E],
    fields: [{ name: 'label' }, { name: 'policy', type: 'object' }, { name: 'can_edit', type: 'boolean' }, { name: 'attributes', type: 'array' }, { name: 'caption' }],
    run: async (p) => {
      const r = await masteringApi.policy(str(p, 'entity'));
      return {
        ...r, label: policyText(r.policy), attributes: r.policy.attributes ?? [],
        caption: r.policy.inherited ? t('mastering.policy.inherited') : t('mastering.policy.own', { by: r.policy.updated_by ?? '—' }),
      };
    },
  },
  {
    id: 'mastering.policy.save', domain: 'mastering', kind: 'mutation', label: 'Change the override policy',
    description: 'Administrators only; audited. policy: {mode, approvals_required, high_risk_attributes, high_risk_approvals}.',
    params: [E, { name: 'policy', type: 'object', required: true }],
    run: (p) => {
      const { attributes: _known, ...policy } = (p.policy ?? {}) as Partial<Policy>;
      const n = (v: unknown, d: number) => (v === undefined || v === null || v === '' ? d : Number(v));
      return masteringApi.setPolicy(str(p, 'entity'), {
        ...policy,
        approvals_required: n(policy.approvals_required, 1),
        high_risk_approvals: n(policy.high_risk_approvals, 1),
        high_risk_attributes: policy.high_risk_attributes ?? [],
      });
    },
  },
  {
    id: 'mastering.price.detail', domain: 'mastering', kind: 'query', label: 'A golden price for its drawer',
    description: 'Header, the version shown, why this value, every quote considered, controls, variances, exceptions, versions, and the steward decision (override form data).',
    params: [E, { name: 'id', type: 'string', required: true }, { name: 'version', type: 'number' }],
    fields: [
      { name: 'title' }, { name: 'subtitle' }, { name: 'value_text' }, { name: 'header_chips', type: 'array' }, { name: 'historical', type: 'boolean' },
      { name: 'viewing_text' }, { name: 'reason_text' }, { name: 'prior_text' }, { name: 'candidates', type: 'array' }, { name: 'ranking_text' },
      { name: 'controls', type: 'array' }, { name: 'variances', type: 'array' }, { name: 'exceptions', type: 'array' }, { name: 'versions', type: 'array' },
      { name: 'decide', type: 'object' },
    ],
    run: (p) => priceDetail(str(p, 'entity'), str(p, 'id'), p.version === null || p.version === undefined || p.version === '' ? undefined : Number(p.version)),
  },
  {
    id: 'mastering.runs.setup', domain: 'mastering', kind: 'query', label: 'What can be mastered',
    description: 'The staging loads (label, caption) and the staging tables bound to the entity\'s business object; ready / can_run for the choice made.',
    params: [E, { name: 'bo_key', type: 'string', required: true }, { name: 'load_id', type: 'string' }, { name: 'table', type: 'string' }, { name: 'again', type: 'boolean' }],
    fields: [
      { name: 'loads', type: 'array' }, { name: 'tables', type: 'array' }, { name: 'initial', type: 'object' }, { name: 'loads_help' }, { name: 'table_help' },
      { name: 'already_mastered', type: 'boolean' }, { name: 'ready', type: 'boolean' }, { name: 'can_run', type: 'boolean' },
    ],
    run: (p) => runSetup(str(p, 'entity'), str(p, 'bo_key'), str(p, 'load_id'), str(p, 'table'), p.again === true || p.again === 'true'),
  },
  {
    id: 'mastering.runs.preview', domain: 'mastering', kind: 'mutation', label: 'Preview mastering a load (nothing kept)',
    description: 'result: chips (counts), exceptions ({code, severity, message}), unmastered_text.',
    params: [E, { name: 'table', type: 'string', required: true }, { name: 'load_id', type: 'string', required: true }, { name: 'again', type: 'boolean' }],
    invalidates: [],
    run: async (p) => {
      const { preview } = await masteringApi.preview(str(p, 'entity'), runRequest(p));
      return {
        chips: countChips(preview.counts),
        exceptions: preview.exceptions,
        unmastered_text: preview.unmastered_fields?.length ? t('mastering.runDialog.unmastered', { fields: preview.unmastered_fields.join(', ') }) : '',
      };
    },
  },
  {
    id: 'mastering.runs.run', domain: 'mastering', kind: 'mutation', label: 'Master a load',
    description: 'Runs to the end, reporting each stage. result: run, messageKey (what happened), tab (where to look).',
    params: [E, { name: 'table', type: 'string', required: true }, { name: 'load_id', type: 'string', required: true }, { name: 'again', type: 'boolean' }],
    run: async (p, ctx) => {
      const { run } = await masteringApi.runToEnd(str(p, 'entity'), runRequest(p),
        (r) => ctx?.progress(t('mastering.runDialog.stage', { stage: t(`mastering.stage.${r.stage}`, { defaultValue: r.stage }) })));
      return { run, messageKey: run.replayed ? 'mastering.replayed' : `mastering.finished.${run.status}`, tab: run.counts.exceptions ? 'exceptions' : 'golden' };
    },
  },
  {
    id: 'mastering.golden.list', domain: 'mastering', kind: 'query', label: 'Golden records',
    params: [E, { name: 'q', type: 'string', label: 'Search' }, { name: 'status', type: 'string' }],
    fields: [
      { name: 'id' }, { name: 'code' }, { name: 'name' }, { name: 'status' }, { name: 'version', type: 'number' }, { name: 'dq_score', type: 'number' },
      { name: 'identity_confidence', type: 'number' }, { name: 'winning_label', description: 'Distinct winning sources' },
      { name: 'winning_tooltip', description: 'attribute: source, one per line' }, { name: 'updated_at', type: 'datetime' },
    ],
    run: async (p) => (await masteringApi.golden(str(p, 'entity'), { q: str(p, 'q'), status: str(p, 'status') })).golden.map((g) => ({
      ...g,
      winning_label: Array.from(new Set(Object.values(g.winning_sources ?? {}))).join(', ') || String(g.sources),
      winning_tooltip: Object.entries(g.winning_sources ?? {}).map(([a, s]) => `${a}: ${s}`).join('\n'),
    })),
  },
  {
    id: 'mastering.prices.list', domain: 'mastering', kind: 'query', label: 'Golden prices for a date',
    description: 'Returns {date, dates, prices}: the date shown, the dates available, and the prices.',
    params: [E, { name: 'date', type: 'string' }, { name: 'q', type: 'string' }, { name: 'price_type', type: 'string' }, { name: 'status', type: 'string' }],
    fields: [
      { name: 'id' }, { name: 'name' }, { name: 'entity_id' }, { name: 'code' }, { name: 'price_type' }, { name: 'value', type: 'number' }, { name: 'currency' },
      { name: 'change_pct', type: 'number' }, { name: 'winner' }, { name: 'sources', type: 'number' }, { name: 'variance_pct', type: 'number' },
      { name: 'status' }, { name: 'is_stale', type: 'boolean' }, { name: 'version', type: 'number' }, { name: 'updated_at', type: 'datetime' },
      { name: 'winner_label' }, { name: 'sources_help' },
    ],
    run: async (p) => {
      const r = (await masteringApi.prices(str(p, 'entity'), { date: str(p, 'date'), q: str(p, 'q'), price_type: str(p, 'price_type'), status: str(p, 'status') })).prices;
      return {
        ...r,
        prices: r.prices.map((x) => ({ ...x, winner_label: `${x.winner ?? '—'} · ${x.sources}`, sources_help: t('mastering.prices.sourcesHelp', { n: x.sources }) })),
      };
    },
  },
  {
    id: 'mastering.prices.completeness', domain: 'mastering', kind: 'mutation', label: 'Check price completeness for a date',
    description: 'Raises missing-price exceptions (and resolves those priced since). result: severity, message, gaps (one label per gap).',
    params: [E, { name: 'date', type: 'string', required: true }],
    run: async (p) => {
      const { completeness: c } = await masteringApi.completeness(str(p, 'entity'), str(p, 'date'));
      const missing = c.missing - c.held;
      const summary = t('mastering.prices.completeness.summary', { date: c.date, expected: c.expected, priced: c.priced, held: c.held, missing, stale: c.stale });
      const raised = c.raised > 0 || c.resolved > 0 ? ` ${t('mastering.prices.completeness.raised', { raised: c.raised, resolved: c.resolved })}` : '';
      const gaps = c.gaps.map((g) => `${g.name ?? g.code ?? g.entity_id.slice(0, 8)} · ${g.price_type}${g.held ? ` (${t('mastering.goldenStatus.REVIEW')})` : ''}`);
      return {
        severity: missing > 0 ? 'warning' : c.held > 0 ? 'info' : 'success',
        message: summary + raised,
        gaps: gaps.length > 40 ? [...gaps.slice(0, 40), `+${gaps.length - 40}`] : gaps,
      };
    },
  },
  {
    id: 'mastering.golden.detail', domain: 'mastering', kind: 'query', label: 'A golden record for its drawer',
    description: 'Header chips, the version shown and what it changed, fields (with competing values, override notice), the side-by-side matrix (sources + rows.by_source), identifiers, sources, versions, overrides, exceptions. width follows view (compare is wider).',
    params: [E, { name: 'id', type: 'string', required: true }, { name: 'version', type: 'number' }, { name: 'view', type: 'string' }],
    fields: [
      { name: 'title' }, { name: 'code' }, { name: 'width', type: 'number' }, { name: 'historical', type: 'boolean' }, { name: 'viewing_text' },
      { name: 'changes_text' }, { name: 'fields', type: 'array' }, { name: 'identifiers', type: 'array' }, { name: 'sources_list', type: 'array' },
      { name: 'versions', type: 'array' }, { name: 'overrides', type: 'array' }, { name: 'exceptions', type: 'array' }, { name: 'matrix', type: 'object' },
    ],
    run: (p) => goldenDetail(str(p, 'entity'), str(p, 'id'), p.version === null || p.version === undefined || p.version === '' ? undefined : Number(p.version), str(p, 'view')),
  },
  {
    id: 'mastering.overrides.propose', domain: 'mastering', kind: 'mutation', label: 'Propose (or apply) a steward override',
    description: 'SET a value or CLEAR an active override; applies now or goes for approval per the entity\'s policy. Numbers go as numbers.',
    params: [E, { name: 'id', type: 'string', required: true }, { name: 'attribute', type: 'string', required: true },
      { name: 'action', type: 'string' }, { name: 'value', type: 'string' }, { name: 'reason', type: 'string', required: true }],
    run: (p) => {
      const action = (str(p, 'action') || 'SET') as 'SET' | 'CLEAR';
      const raw = str(p, 'value').trim();
      const value = raw !== '' && /^-?\d+(\.\d+)?$/.test(raw) ? Number(raw) : raw;
      return masteringApi.proposeOverride(str(p, 'entity'), str(p, 'id'), {
        attribute: str(p, 'attribute'), action, value: action === 'SET' ? value : undefined, reason: str(p, 'reason').trim(),
      });
    },
  },
  {
    id: 'mastering.runs.list', domain: 'mastering', kind: 'query', label: 'Mastering runs', params: [E],
    fields: [{ name: 'started_at', type: 'datetime' }, { name: 'status' }, { name: 'trigger' }, { name: 'counts', type: 'object' }, { name: 'error_detail' }, { name: 'started_by' }],
    run: async (p) => (await masteringApi.runs(str(p, 'entity'))).runs,
  },
  {
    id: 'mastering.exceptions.list', domain: 'mastering', kind: 'query', label: 'Exceptions',
    description: 'Open exceptions by default; status RESOLVED or WAIVED for closed ones.', params: [E, { name: 'status', type: 'string' }],
    fields: [{ name: 'id' }, { name: 'type' }, { name: 'severity' }, { name: 'description' }, { name: 'golden_id' }, { name: 'source_ref' }, { name: 'detected_at', type: 'datetime' }],
    run: async (p) => (await masteringApi.exceptions(str(p, 'entity'), str(p, 'status'))).exceptions.map((x) => ({
      ...x, source_ref: [x.source, x.source_key].filter(Boolean).join(' / '),
    })),
  },
  {
    id: 'mastering.exceptions.resolve', domain: 'mastering', kind: 'mutation', label: 'Resolve or waive an exception',
    params: [E, { name: 'id', type: 'string', required: true }, { name: 'status', type: 'string', required: true, description: 'RESOLVED or WAIVED' }, { name: 'note', type: 'string' }],
    run: (p) => masteringApi.resolve(str(p, 'entity'), str(p, 'id'), str(p, 'status') as 'RESOLVED' | 'WAIVED', str(p, 'note') || undefined),
  },
  {
    id: 'mastering.candidates.list', domain: 'mastering', kind: 'query', label: 'Possible duplicates',
    description: 'state: open (undecided), mine (my pending merge), voted, can_vote.', params: [E],
    fields: [{ name: 'id' }, { name: 'a' }, { name: 'a_code' }, { name: 'a_name' }, { name: 'b' }, { name: 'b_code' }, { name: 'b_name' }, { name: 'score', type: 'number' }, { name: 'rule' }, { name: 'state' }, { name: 'merge_caption' }],
    run: async (p) => (await masteringApi.candidates(str(p, 'entity'))).candidates.map((c) => ({
      ...c,
      state: c.merge_request ? requestState(c.merge_request) : 'open',
      merge_caption: c.merge_request ? t('mastering.review.mergeRequested', {
        by: c.merge_request.requested_by_name ?? '—', keep: c.merge_request.keep === 'a' ? c.a_code : c.b_code,
        n: c.merge_request.approvals, of: c.merge_request.approvals_required,
      }) : '',
      merge_request_id: c.merge_request?.id,
    })),
  },
  {
    id: 'mastering.candidates.toReview', domain: 'mastering', kind: 'query', label: 'Duplicates awaiting me (count)', params: [E],
    run: async (p) => (await masteringApi.candidates(str(p, 'entity'))).candidates
      .filter((c) => !c.merge_request || (!c.merge_request.mine && !c.merge_request.voted)).length,
  },
  {
    id: 'mastering.candidates.decide', domain: 'mastering', kind: 'mutation', label: 'Merge, or keep apart',
    description: 'result.message says what happened (merged, sent for approval, kept apart).',
    params: [E, { name: 'id', type: 'string', required: true }, { name: 'merge', type: 'boolean', required: true }, { name: 'keep', type: 'string' }, { name: 'note', type: 'string' }],
    run: async (p) => {
      const { decision } = await masteringApi.decide(str(p, 'entity'), str(p, 'id'), {
        merge: p.merge === true || p.merge === 'true', keep: (str(p, 'keep') || 'a') as 'a' | 'b', note: str(p, 'note') || undefined,
      });
      return { decision, message: decisionMessage(decision) };
    },
  },
  {
    id: 'mastering.merges.vote', domain: 'mastering', kind: 'mutation', label: 'Approve or reject a merge',
    params: [E, { name: 'id', type: 'string', required: true }, { name: 'approve', type: 'boolean', required: true }],
    run: async (p) => {
      const { decision } = await masteringApi.voteMerge(str(p, 'entity'), str(p, 'id'), p.approve === true || p.approve === 'true');
      return { decision, message: decisionMessage(decision) };
    },
  },
  {
    id: 'mastering.merges.withdraw', domain: 'mastering', kind: 'mutation', label: 'Withdraw my merge request',
    params: [E, { name: 'id', type: 'string', required: true }],
    run: (p) => masteringApi.withdrawMerge(str(p, 'entity'), str(p, 'id')),
  },
  {
    id: 'mastering.overrides.list', domain: 'mastering', kind: 'query', label: 'Steward overrides',
    description: 'status PENDING, ACTIVE, or empty for all. state: mine, voted, can_vote, closed.', params: [E, { name: 'status', type: 'string' }],
    fields: [
      { name: 'id' }, { name: 'record_label' }, { name: 'open_ref' }, { name: 'golden_name' }, { name: 'attribute' }, { name: 'action' },
      { name: 'before_text' }, { name: 'after_text' }, { name: 'reason' }, { name: 'requested_by_name' }, { name: 'requested_at', type: 'datetime' },
      { name: 'status_view' }, { name: 'status_label' }, { name: 'approvals_text' }, { name: 'voters' }, { name: 'state' },
    ],
    run: async (p) => (await masteringApi.overrides(str(p, 'entity'), { status: str(p, 'status') })).overrides.map(overrideRow),
  },
  {
    id: 'mastering.overrides.toDecide', domain: 'mastering', kind: 'query', label: 'Overrides awaiting me (count)', params: [E],
    run: async (p) => (await masteringApi.overrides(str(p, 'entity'), { status: 'PENDING' })).overrides.filter((o) => !o.mine && !o.voted).length,
  },
  {
    id: 'mastering.overrides.vote', domain: 'mastering', kind: 'mutation', label: 'Approve or reject an override',
    params: [E, { name: 'id', type: 'string', required: true }, { name: 'approve', type: 'boolean', required: true }, { name: 'comment', type: 'string' }],
    run: (p) => masteringApi.voteOverride(str(p, 'entity'), str(p, 'id'), p.approve === true || p.approve === 'true', str(p, 'comment') || undefined),
  },
  {
    id: 'mastering.overrides.withdraw', domain: 'mastering', kind: 'mutation', label: 'Withdraw my override',
    params: [E, { name: 'id', type: 'string', required: true }],
    run: (p) => masteringApi.withdrawOverride(str(p, 'entity'), str(p, 'id')),
  },
];

const GOLDEN_COLOR: Record<string, 'success' | 'warning' | 'default' | 'info' | 'error'> = {
  PUBLISHED: 'success', REVIEW: 'warning', SUPERSEDED: 'default', DRAFT: 'info', RETRACTED: 'error',
};
const dash = (v: unknown) => (v === undefined || v === null || v === '' ? '—' : String(v));

/** The strategy that decided a value: the reason's prefix ("SOURCE_PRIORITY: ..."). */
function strategyOf(reason?: string): { strategy: string; detail: string } {
  const r = reason ?? '';
  const i = r.indexOf(':');
  if (i > 0 && /^[A-Z_]+$/.test(r.slice(0, i))) return { strategy: r.slice(0, i), detail: r.slice(i + 1).trim() };
  return { strategy: '', detail: r };
}

function approvalsNeeded(p: Policy, attr: string) {
  if (p.mode === 'DIRECT') return 0;
  return p.high_risk_attributes.includes(attr) ? Math.max(p.high_risk_approvals, p.approvals_required) : p.approvals_required;
}

/**
 * A golden record shaped for its drawer (the provenance view): header,
 * version being viewed and what it changed, each attribute with its source,
 * confidence and the values that competed (and why), the side-by-side matrix
 * of every source, identifiers, sources, versions, overrides and exceptions.
 * The same derivations the hand-built drawer made - the page only displays.
 */
async function goldenDetail(entity: string, id: string, version: number | undefined, view: string) {
  const [{ golden: d }, { overrides }, pol] = await Promise.all([
    masteringApi.goldenById(entity, id, version),
    masteringApi.overrides(entity, { golden: id }),
    masteringApi.policy(entity).catch(() => null),
  ]);
  const latest = d.versions[0];
  const idx = Math.max(0, d.versions.findIndex((v) => v.version === d.selected_version));
  const current = d.versions[idx];
  const historical = !!latest && d.selected_version !== latest.version;
  const previous = d.versions[idx + 1];
  const changes = new Map<string, unknown>();
  if (current && previous) {
    for (const k of new Set([...Object.keys(current.attributes ?? {}), ...Object.keys(previous.attributes ?? {})])) {
      if (JSON.stringify(current.attributes?.[k] ?? null) !== JSON.stringify(previous.attributes?.[k] ?? null)) changes.set(k, previous.attributes?.[k]);
    }
  }
  const decisions = new Map(d.decisions.map((x) => [x.field, x]));
  const active = new Set(overrides.filter((o) => o.active).map((o) => o.attribute));
  const policy = pol?.policy;
  const fields = d.fields.map((f) => {
    const dec = decisions.get(f.name);
    const conf = f.confidence ?? 0;
    const need = policy ? Math.max(1, approvalsNeeded(policy, f.name)) : 1;
    return {
      id: f.name, attribute: f.name, value: dash(f.value),
      was_text: changes.has(f.name) ? t('mastering.golden.was', { v: showValue(changes.get(f.name)) }) : '',
      overridden: !historical && active.has(f.name), source: f.source ?? '',
      confidence: f.confidence, confidence_color: conf >= 0.8 ? 'success' : conf >= 0.5 ? 'warning' : 'error', confidence_label: pct(f.confidence),
      has_decision: !!dec, reason: dec?.reason ?? '',
      competing: (dec?.competing ?? []).map((c) => ({
        source: c.source, source_key: c.source_key, value: dash(showValue(c.value)), excluded: c.selected === false, note: c.note ?? '',
        as_of: c.as_of, stale: !!c.stale,
      })),
      can_override: !historical,
      override_title: t('mastering.overrides.title', { attribute: f.name }),
      value_raw: f.value ?? '',
      override_intro: `${t('mastering.overrides.current')}: ${dash(f.value)}. ${t('mastering.overrides.sticky')}`,
      override_notice: policy?.mode === 'DIRECT' ? t('mastering.overrides.appliesNow') : t('mastering.overrides.needsApproval', { count: need }),
      override_severity: policy?.mode === 'DIRECT' ? 'warning' : 'info',
      override_submit: policy?.mode === 'DIRECT' ? t('mastering.overrides.apply') : t('mastering.overrides.propose'),
    };
  });
  // Side by side: source columns, the most often selected first.
  const wins = new Map<string, number>();
  for (const x of d.decisions) {
    for (const c of x.competing ?? []) if (!wins.has(c.source)) wins.set(c.source, 0);
    if (x.source && wins.has(x.source)) wins.set(x.source, (wins.get(x.source) ?? 0) + 1);
  }
  const sources = [...wins.entries()].sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0])).map(([code]) => ({ code }));
  const matrix = [...d.fields].sort((a, b) => (d.terms?.[a.name] ?? a.name).localeCompare(d.terms?.[b.name] ?? b.name)).map((f) => {
    const dec = decisions.get(f.name);
    const { strategy, detail } = strategyOf(dec?.reason);
    const steward = strategy === 'OVERRIDE' || f.source === 'STEWARD' || dec?.source === 'STEWARD';
    const bySource: Record<string, unknown[]> = {};
    for (const { code } of sources) {
      const cands = (dec?.competing ?? []).filter((c) => c.source === code);
      bySource[code] = cands.map((c) => ({
        value: showValue(c.value), key: cands.length > 1 ? c.source_key : '', note: c.note ?? '',
        tone: dec?.source === code && showValue(c.value) === showValue(f.value) ? 'success' : '',
        strike: c.selected === false, stale: !!c.stale,
      }));
    }
    return {
      id: f.name, term: d.terms?.[f.name] ?? f.name, attribute: f.name, golden: showValue(f.value), steward, by_source: bySource,
      strategy, strategy_label: strategy ? t(`mastering.compare.strategy.${strategy}`, { defaultValue: strategy }) : '',
      rule: dec?.rule_id ? d.rules?.[dec.rule_id] ?? '' : '', detail: detail || '—',
    };
  });
  return {
    id: d.id, code: d.code, title: d.name ?? (current?.attributes?.name as string | undefined) ?? d.code,
    width: view === 'compare' ? 1440 : 760,
    header_chips: current ? [
      { label: t(`mastering.goldenStatus.${current.status}`), color: GOLDEN_COLOR[current.status] ?? 'default', variant: 'filled' },
      { label: t('mastering.golden.version', { v: current.version }) },
      { label: t('mastering.golden.dq', { v: dash(current.dq_score) }) },
      { label: t('mastering.golden.identity', { v: pct(current.identity_confidence) }) },
    ] : [],
    historical,
    viewing_text: historical && current && latest
      ? t('mastering.golden.viewingVersion', { v: current.version, latest: latest.version, at: fmt(current.published_at ?? current.knowledge_at, i18n.language) })
      : '',
    changes_text: previous ? (changes.size === 0
      ? t('mastering.golden.noChanges', { v: previous.version })
      : t('mastering.golden.changes', { count: changes.size, v: previous.version })) : '',
    fields_title: historical ? t('mastering.golden.fieldsAt', { v: d.selected_version }) : t('mastering.golden.fields'),
    compare_title: t('mastering.compare.title', { v: d.selected_version }),
    fields,
    identifiers: d.identifiers.map((i) => ({
      label: `${i.type} ${i.value}${i.source ? ` · ${i.source}` : ''}`, color: i.is_primary ? 'primary' : 'default', variant: i.is_primary ? 'filled' : 'outlined',
    })),
    sources_list: d.sources.map((x) => ({
      id: `${x.source}|${x.source_key}`, source: x.source, source_key: x.source_key,
      method: t(`mastering.method.${x.method}`, { defaultValue: x.method }), score: x.score, updated_at: x.updated_at,
    })),
    versions: d.versions.map((v) => ({
      id: v.id, version: v.version, status: v.status, dq_text: t('mastering.golden.dq', { v: dash(v.dq_score) }),
      at: v.published_at ?? v.knowledge_at, showing: v.version === d.selected_version,
      // Selecting the latest shows "latest" (no pinned version).
      target: latest && v.version === latest.version ? null : v.version,
    })),
    overrides: overrides.map(overrideRow),
    exceptions: d.exceptions,
    matrix: { sources, rows: matrix },
  };
}

const num = (v: number | undefined | null, digits = 4) =>
  v === undefined || v === null ? '—' : v.toLocaleString(i18n.language, { maximumFractionDigits: digits });

/**
 * A golden price shaped for its drawer: the version being viewed, why this
 * value won, every quote considered, controls, variances, versions, and
 * what a steward may do now (correct, change or clear, or wait for a
 * pending decision). The same derivations the hand-built drawer made.
 */
async function priceDetail(entity: string, id: string, version: number | undefined) {
  const { price: d } = await masteringApi.priceById(entity, id);
  const attr = `${d.price_type}@${d.date}`;
  const [{ overrides }, pol] = await Promise.all([
    masteringApi.overrides(entity, { golden: d.entity_id }),
    masteringApi.policy(entity).catch(() => null),
  ]);
  const latest = d.versions[0];
  const v = d.versions.find((x) => x.version === version) ?? latest;
  const historical = !!v && !!latest && v.version !== latest.version;
  const prov = v?.provenance;
  const th = prov?.threshold;
  const mine = overrides.filter((o) => o.attribute === attr);
  const active = mine.find((o) => o.active);
  const pending = mine.find((o) => o.status === 'PENDING');
  const policy = pol?.policy;
  const need = policy ? Math.max(1, approvalsNeeded(policy, attr)) : 1;
  const held = latest?.status === 'REVIEW';
  return {
    id: d.id, title: d.name ?? d.entity_id, subtitle: `${d.code ?? ''} · ${d.price_type} · ${d.date}`,
    value_text: v ? `${num(v.value)} ${v.currency ?? ''}`.trim() : '',
    header_chips: v ? [
      { label: t(`mastering.goldenStatus.${v.status}`, { defaultValue: v.status }), color: GOLDEN_COLOR[v.status] ?? 'default', variant: 'filled' },
      { label: t('mastering.golden.version', { v: v.version }) },
      ...(v.is_stale ? [{ label: t('mastering.golden.stale'), color: 'warning' }] : []),
      { label: `${t('mastering.golden.confidence')} ${pct(v.confidence)}` },
    ] : [],
    historical,
    viewing_text: historical && v && latest ? t('mastering.prices.viewingVersion', { v: v.version, latest: latest.version, at: fmt(v.knowledge_at, i18n.language) }) : '',
    reason_text: prov ? `${v.winner ?? t('mastering.prices.previous')} — ${prov.reason}` : '',
    prior_text: prov?.prior
      ? t('mastering.prices.prior', { date: prov.prior.date, value: num(prov.prior.value) }) + (prov.change_pct !== undefined ? ` · ${t('mastering.prices.moved', { pct: num(prov.change_pct, 2) })}` : '')
      : '',
    candidates: (prov?.candidates ?? []).map((c) => ({
      id: c.source, source: c.source, won: c.source === v?.winner, value_text: num(c.value), currency: c.currency ?? '',
      out: !!c.excluded || c.selected === false, diff_pct: c.diff_pct, rank_text: c.rank ? String(c.rank) : '—', as_of: c.as_of, stale: !!c.stale,
      excluded_text: c.excluded ? t('mastering.prices.excluded', { why: c.excluded }) : '', note: c.note ?? '',
    })),
    ranking_text: prov ? t('mastering.prices.ranking', { order: prov.ranking.join(' › ') || '—' })
      + (th && th.warning !== undefined ? ` · ${t('mastering.prices.thresholds', { w: th.warning, e: th.error, c: th.critical })}` : '') : '',
    controls: (prov?.controls ?? []).map((c) => ({
      label: t('mastering.prices.control', {
        control: t(`mastering.exceptionType.${c.control}`, { defaultValue: c.control }), level: c.level.toLowerCase(), pct: num(c.pct, 2), action: c.action.toLowerCase(),
      }),
      color: c.level === 'WARNING' ? 'warning' : 'error', variant: c.action === 'HOLD' ? 'filled' : 'outlined',
    })),
    variances: d.variances.map((x) => ({
      id: x.id, a_text: `${x.source_a ?? ''} ${num(x.price_a)}`, b_text: `${x.source_b ?? ''} ${num(x.price_b)}`,
      variance_pct: x.variance_pct, severity: x.severity, status: x.status,
    })),
    exceptions: d.exceptions,
    versions: d.versions.map((x) => ({
      id: x.id, version: x.version, value_text: `${num(x.value)} ${x.currency ?? ''}`.trim(), winner: x.winner ?? '—', status: x.status,
      at: x.knowledge_at, showing: x.version === v?.version, target: latest && x.version === latest.version ? null : x.version,
    })),
    // What a steward may do on the latest version.
    decide: {
      can: !historical && !!latest, held, blocked: !!pending,
      label: held ? t('mastering.prices.decideHeld') : active ? t('mastering.prices.changeOverride') : t('mastering.prices.correct'),
      chips: [
        ...(active ? [{ label: t('mastering.prices.stewardPrice'), color: 'secondary' }] : []),
        ...(pending ? [{ label: overrideRow(pending).status_label, color: 'warning' }] : []),
      ],
      pending_text: pending ? t('mastering.prices.pending', { value: String(pending.value ?? '—') }) : '',
      golden_id: d.id, attribute: attr, has_active: !!active, value_raw: latest ? String(latest.value) : '',
      title: t('mastering.overrides.title', { attribute: attr }),
      intro: `${t('mastering.overrides.current')}: ${latest ? String(latest.value) : '—'}. ${t('mastering.overrides.sticky')}`,
      notice: policy?.mode === 'DIRECT' ? t('mastering.overrides.appliesNow') : t('mastering.overrides.needsApproval', { count: need }),
      severity: policy?.mode === 'DIRECT' ? 'warning' : 'info',
      submit: policy?.mode === 'DIRECT' ? t('mastering.overrides.apply') : t('mastering.overrides.propose'),
    },
  };
}

const COUNT_KEYS: (keyof Counts)[] = ['records', 'invalid', 'xref', 'deterministic', 'fuzzy', 'review', 'new', 'restated', 'rechecked', 'conflicts', 'published', 'held_for_review', 'unchanged', 'exceptions'];

/** A run's counts as chips: the non-zero ones, problems coloured. */
function countChips(counts: Partial<Counts>) {
  return COUNT_KEYS.filter((k) => (counts[k] ?? 0) > 0).map((k) => ({
    label: `${t(`mastering.counts.${k}`)} ${counts[k]}`, variant: 'outlined',
    color: k === 'invalid' || k === 'conflicts' ? 'error' : k === 'review' || k === 'held_for_review' || k === 'exceptions' ? 'warning' : k === 'published' ? 'success' : 'default',
  }));
}

/** A load is mastered once per key; "master again" gives it a new one. */
const runRequest = (p: Record<string, unknown>) => ({
  staging_table: str(p, 'table'), load_run_id: str(p, 'load_id'),
  idempotency_key: p.again === true || p.again === 'true' ? `load:${str(p, 'load_id')}:${Date.now()}` : undefined,
});

/** The loads that can be mastered and the staging tables bound to the entity. */
async function runSetup(entity: string, boKey: string, loadId: string, table: string, again: boolean) {
  const [{ loads }, { bindings }] = await Promise.all([masteringApi.loads(entity), stagingBindingsApi.list()]);
  const tables = bindings.filter((b) => b.bo_key === boKey).map((b) => b.staging_table);
  const load = loads.find((l) => l.id === loadId);
  const alreadyMastered = !!load?.mastering_run_id;
  const ready = !!loadId && !!table;
  return {
    loads: loads.map((l) => ({
      id: l.id,
      label: `${l.source} · ${l.run_ref ?? l.id.slice(0, 8)} · ${fmt(l.started_at, i18n.language)}`,
      caption: [
        l.received_rows !== undefined && l.received_rows !== null ? t('mastering.runDialog.rows', { n: l.received_rows }) : '',
        l.mastering_status ? t('mastering.runDialog.mastered', { status: t(`mastering.runStatus.${l.mastering_status}`) }) : '',
      ].filter(Boolean).join(' · '),
    })),
    tables: tables.map((x) => ({ value: x })),
    // One bound table is chosen for you.
    initial: tables.length === 1 ? { table: tables[0] } : {},
    loads_help: loads.length === 0 ? t('mastering.runDialog.noLoads') : '',
    table_help: tables.length === 0 ? t('mastering.runDialog.noBinding', { bo: boKey }) : t('mastering.runDialog.tableHelp'),
    already_mastered: alreadyMastered, ready, can_run: ready && (!alreadyMastered || again),
  };
}

function decisionMessage(d: { status: string; moved?: { sources: number; identifiers: number } }) {
  if (d.status === 'APPROVED') return t('mastering.review.merged', { sources: d.moved?.sources ?? 0, identifiers: d.moved?.identifiers ?? 0 });
  if (d.status === 'PENDING_APPROVAL') return t('mastering.review.requested');
  if (d.status === 'MERGE_REJECTED') return t('mastering.review.mergeRejected');
  return t('mastering.review.rejected');
}

registerOperations(operations);
