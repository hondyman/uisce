import { describe, expect, it } from 'vitest';
import '../../i18n';
import '../../studio-core/registerDomains';
import { PAGE_BLUEPRINTS } from '../../pages/page-studio/app/blueprints';
import { checkPage } from '../../pages/page-studio/app/pageChecker';

describe('page checker on every blueprint', () => {
  for (const b of PAGE_BLUEPRINTS) {
    it(`${b.id}: no errors`, () => {
      const issues = checkPage(b.build() as never);
      if (issues.length) console.log('ISSUES', b.id, JSON.stringify(issues.map((i) => `${i.severity} ${i.code} @ ${i.where}: ${i.message}`), null, 1));
      expect(issues.filter((i) => i.severity === 'error')).toEqual([]);
    });
  }
});

describe('page checker finds what would break a page', () => {
  const cond = (field: string, operator: string) => ({ type: 'condition', field, operator });
  const broken = {
    layout: { root: 'root', nodes: {
      root: { id: 'root', type: 'Column', children: ['grid', 'btn', 'ghost', 'dlg', 'drawer'] },
      dlg: { id: 'dlg', type: 'Dialog', children: [], props: { title: 'Edit', openWhen: cond('vars.stuck', 'is_true'), onClose: [] } },
      drawer: { id: 'drawer', type: 'Drawer', children: [], props: { title: 'Detail', openWhen: cond('vars.open', 'is_true') } },
    } },
    components: {
      grid: { id: 'grid', type: 'DataGrid', props: { query: 'nope', columns: [{ id: 'a', header: 'mastering.no.such.key', cell: { kind: 'text', value: '{{vars.missing}}' } }] } },
      btn: { id: 'btn', type: 'ActionButton', props: { label: '', onClick: [
        { kind: 'setVariable', name: 'undeclared', value: 1 },
        { kind: 'runOperation', operation: 'no.such.op', params: {} },
        { kind: 'runOperation', operation: 'schedules.remove', params: {} },
        { kind: 'setVariable', name: 'open', value: true },
      ] } },
      lonely: { id: 'lonely', type: 'TextBlock', props: { text: 'Never placed' } },
      dc: { id: 'dc', type: 'DomainComponent', props: { component: 'gone.Component' } },
    },
    app: {
      variables: [{ name: 'stuck' }, { name: 'open' }, { name: 'open' }],
      queries: [{ id: 'q', operation: 'schedules.targets', params: {} }, { id: 'bad', operation: 'x.y' }],
      tabVariable: 'tabz',
    },
  };
  const issues = checkPage(broken as never);
  const codes = (sev: string) => issues.filter((i) => i.severity === sev).map((i) => i.code).sort();

  it('reports each problem once, errors first', () => {
    expect(codes('error')).toEqual([
      'cannot-close', 'duplicate-variable', 'missing-node', 'missing-param', 'missing-param', 'unknown-component',
      'unknown-operation', 'unknown-operation', 'unknown-query', 'unknown-variable', 'unknown-variable', 'unknown-variable',
    ]);
    expect(codes('warning')).toEqual(['missing-translation', 'no-close', 'no-label', 'unplaced', 'unplaced']);
    expect(issues[0].severity).toBe('error');
  });

  it('says where, in words a designer recognises', () => {
    const v = issues.find((i) => i.message.includes('"missing"'))!;
    expect(v.where).toContain('component › grid › columns › 0 › cell › value');
    expect(v.componentId).toBe('grid');
    expect(issues.find((i) => i.code === 'cannot-close')!.message).toMatch(/"stuck"/);
  });
});

describe('page checker cube subject pin (PR1a)', () => {
  const pageWithCube = (subject: unknown, savedQueryId?: string) => ({
    layout: { root: 'root', nodes: { root: { id: 'root', type: 'Column', children: ['tile'] } } },
    components: {
      tile: {
        id: 'tile',
        type: 'Tile',
        props: {
          savedQueryId,
          subject,
        },
      },
    },
    app: {},
  }) as never;

  it('errors when cube subject uses contractVersion latest', () => {
    const issues = checkPage(pageWithCube({ kind: 'cube', cubeId: 'c1', contractVersion: 'latest' }, 'sq-1'));
    expect(issues.some((i) => i.code === 'cube-pin-latest' && i.severity === 'error')).toBe(true);
  });

  it('passes numeric cube pin with savedQueryId', () => {
    const issues = checkPage(pageWithCube({ kind: 'cube', cubeId: 'c1', contractVersion: 2 }, 'sq-1'));
    expect(issues.filter((i) => i.code.startsWith('cube-pin'))).toEqual([]);
  });

  it('errors when cube subject has no savedQueryId', () => {
    const issues = checkPage(pageWithCube({ kind: 'cube', cubeId: 'c1', contractVersion: 1 }));
    expect(issues.some((i) => i.code === 'cube-pin-no-saved-query')).toBe(true);
  });
});

describe('page checker and conditions', () => {
  const page = (when: unknown) => ({
    layout: { root: 'root', nodes: { root: { id: 'root', type: 'Column', children: ['t'] } } },
    components: { t: { id: 't', type: 'TextBlock', visibleWhen: when, props: { text: 'x' } } },
    app: { variables: [{ name: 'a' }] },
  }) as never;
  const bad = (when: unknown) => checkPage(page(when)).filter((i) => i.code === 'bad-condition').map((i) => `${i.severity}: ${i.message}`);

  it('blocks a condition the engine cannot run or that is half built', () => {
    expect(bad({ type: 'condition', field: 'vars.a', operator: 'sounds_like', value: 'x' })).toEqual(['error: The rule engine has no operator "sounds_like".']);
    expect(bad({ type: 'condition', field: 'vars.a', operator: 'equals' })).toEqual(['error: "equals" needs a value.']);
    expect(bad({ type: 'group', operator: 'OR', conditions: [{ type: 'condition', field: '', operator: 'is_true' }] })).toEqual(['error: Choose what to check.']);
  });
  it('passes complete conditions, including 0 and false as values', () => {
    expect(bad({ type: 'condition', field: 'vars.a', operator: 'equals', value: 0 })).toEqual([]);
    expect(bad({ type: 'condition', field: 'vars.a', operator: 'equals', value: false })).toEqual([]);
    expect(bad({ type: 'condition', field: 'vars.a', operator: 'is_not_null' })).toEqual([]);
  });
});
