import { computeQuickFixActions, convertActionsToMonacoEdits } from '../MonacoCodeEditor';

describe('convertActionsToMonacoEdits', () => {
  const mockModel = {
    getValue: () => '{\n  "name": "x"\n}',
    getLineCount: () => 2,
    getLineMaxColumn: (n: number) => n === 2 ? 12 : 1,
    uri: { toString: () => 'inmemory://1' },
  } as any;

  const mockMonaco = {
    Range: function(s: any, sc: any, e: any, ec: any) { return { start: s, startCol: sc, end: e, endCol: ec }; }
  } as any;
  // ensure global monaco Range used by buildAstReplacement is available
  (window as any).monaco = mockMonaco;

  it('produces Monaco edits for datasource AST action', () => {
    const markers = [{ code: 'MISSING_DATASOURCE', message: 'datasource missing' }];
    const actions = computeQuickFixActions(markers, mockModel.getValue(), 'json');
    const edits = convertActionsToMonacoEdits(actions, mockModel, mockMonaco, 'json');
    expect(edits.length).toBeGreaterThan(0);
    expect(edits[0].edit).toBeDefined();
    expect(Array.isArray(edits[0].edit.edits)).toBe(true);
  });

  it('produces Monaco edits for join AST action', () => {
    const markers = [{ code: 'MISSING_JOIN', message: 'join missing' }];
    const actions = computeQuickFixActions(markers, mockModel.getValue(), 'json');
    const edits = convertActionsToMonacoEdits(actions, mockModel, mockMonaco, 'json');
    expect(edits.length).toBeGreaterThan(0);
    expect(edits[0].edit.edits[0].edit.text).toContain('joins');
  });
});
