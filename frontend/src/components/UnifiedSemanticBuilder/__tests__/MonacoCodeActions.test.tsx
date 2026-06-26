import { computeQuickFixActions } from '../MonacoCodeEditor';

describe('computeQuickFixActions', () => {
  it('maps MISSING_DATASOURCE code to AST datasource action', () => {
    const markers = [{ code: 'MISSING_DATASOURCE', message: 'datasource is missing' }];
    const actions = computeQuickFixActions(markers, '{}', 'json');
    expect(actions.some(a => a.title && a.title.includes('datasource'))).toBe(true);
    const astAction = actions.find(a => a.title === 'Insert tenant_instance_id (AST)');
    expect(astAction).toBeDefined();
    expect(typeof astAction.updater).toBe('function');
  });

  it('maps MISSING_JOIN code to scaffold join action', () => {
    const markers = [{ code: 'MISSING_JOIN', message: 'join missing' }];
    const actions = computeQuickFixActions(markers, '{}', 'json');
    const joinAction = actions.find(a => a.title === 'Scaffold join (AST)');
    expect(joinAction).toBeDefined();
    expect(typeof joinAction.updater).toBe('function');
  });

  it('falls back to message-based datasource action when code absent', () => {
    const markers = [{ message: 'No tenant_instance_id found for model' }];
    const actions = computeQuickFixActions(markers, '{}', 'json');
    const fallback = actions.find(a => a.title && a.title.includes('Insert tenant_instance_id'));
    expect(fallback).toBeDefined();
    expect(fallback!.rawText).toBeDefined();
  });
});
