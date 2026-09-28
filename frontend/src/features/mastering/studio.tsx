import React from 'react';
import i18n from '../../i18n';
import { fmt } from '../schedules/api';
import { registerOperations, type OperationDef } from '../../studio-core/operations/registry';
import { registerDomainComponents } from '../../studio-core/components/registry';
import { masteringApi, isSeries, pct, showValue, type Override, type Policy, type Profile, type Run } from './api';
import { PriceDrawer } from './prices';
import RunDialog from './RunDialog';
import { PolicyDialog, policyText } from './overrides';

/**
 * The mastering domain's Page Studio surface: the operations a page may call
 * and the hand-built components it may place. Row-level derivations (who
 * may vote, what a request says) live here - the domain decides, the page
 * displays - so pages stay declarative and the rules stay with the API.
 */

const t = (k: string, o?: Record<string, unknown>) => i18n.t(k, o) as string;
const str = (p: Record<string, unknown>, k: string) => (p[k] === undefined || p[k] === null ? '' : String(p[k]));

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
    fields: [{ name: 'label' }, { name: 'policy', type: 'object' }, { name: 'can_edit', type: 'boolean' }],
    run: async (p) => {
      const r = await masteringApi.policy(str(p, 'entity'));
      return { ...r, label: policyText(t, r.policy) };
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

function decisionMessage(d: { status: string; moved?: { sources: number; identifiers: number } }) {
  if (d.status === 'APPROVED') return t('mastering.review.merged', { sources: d.moved?.sources ?? 0, identifiers: d.moved?.identifiers ?? 0 });
  if (d.status === 'PENDING_APPROVAL') return t('mastering.review.requested');
  if (d.status === 'MERGE_REJECTED') return t('mastering.review.mergeRejected');
  return t('mastering.review.rejected');
}

registerOperations(operations);

const entityInput = { name: 'entity', type: 'string' as const, required: true };

registerDomainComponents([
  {
    id: 'mastering.PriceDrawer', domain: 'mastering', label: 'Golden price drawer', overlay: true,
    description: 'A golden price: every quote considered, controls, variances, versions, steward decision.',
    inputs: [entityInput, { name: 'id', type: 'string' }],
    events: [{ name: 'close' }],
    render: ({ inputs, emit }) => (
      <PriceDrawer entity={String(inputs.entity ?? '')} id={(inputs.id as string) || null} onClose={() => emit('close')} />
    ),
  },
  {
    id: 'mastering.RunDialog', domain: 'mastering', label: 'Master a load', overlay: true,
    description: 'Pick a staging load, preview (nothing kept), then run. done carries the run, a message key and the tab to show.',
    inputs: [{ name: 'profile', type: 'object', required: true, description: 'The entity profile (mastering.profile)' }, { name: 'open', type: 'boolean' }],
    events: [{ name: 'close' }, { name: 'done', payload: ['run', 'messageKey', 'tab'] }],
    render: ({ inputs, emit }) => {
      const profile = inputs.profile as Profile | null;
      if (!profile || !inputs.open) return null;
      return (
        <RunDialog profile={profile} open onClose={() => emit('close')}
          onDone={(r: Run) => emit('done', {
            run: r,
            messageKey: r.replayed ? 'mastering.replayed' : `mastering.finished.${r.status}`,
            tab: r.counts.exceptions ? 'exceptions' : 'golden',
          })} />
      );
    },
  },
  {
    id: 'mastering.PolicyDialog', domain: 'mastering', label: 'Override policy', overlay: true,
    description: 'The entity\'s steward policy; administrators can change it (audited).',
    inputs: [entityInput, { name: 'open', type: 'boolean' }],
    events: [{ name: 'close' }],
    render: ({ inputs, emit }) => (inputs.open && inputs.entity
      ? <PolicyDialog entity={String(inputs.entity)} open onClose={() => emit('close')} />
      : null),
  },
]);
