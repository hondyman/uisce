import { describe, expect, it } from 'vitest';
import {
  applyActions,
  isTruthy,
  targetKey,
} from '../../pages/page-studio/presentationEvents';
import type { PresentationAction } from '../../types/pageStudio';

describe('presentationEvents', () => {
  it('builds field target keys separately from widget ids', () => {
    expect(targetKey({ kind: 'widget', id: 'w1' })).toBe('w1');
    expect(targetKey({ kind: 'section', id: 'row1' })).toBe('row1');
    expect(targetKey({ kind: 'field', id: 'form1', fieldName: 'Status' })).toBe('field:form1:Status');
  });

  it('treats wasm boolean and 1 as true', () => {
    expect(isTruthy(true)).toBe(true);
    expect(isTruthy(1)).toBe(true);
    expect(isTruthy('true')).toBe(true);
    expect(isTruthy(false)).toBe(false);
    expect(isTruthy(0)).toBe(false);
    expect(isTruthy('false')).toBe(false);
    expect(isTruthy('')).toBe(false);
  });

  it('merges later actions over earlier ones without dropping unrelated keys', () => {
    const first: PresentationAction[] = [
      { target: { kind: 'widget', id: 'alloc' }, hidden: true, style: { color: '#111' } },
    ];
    const second: PresentationAction[] = [
      { target: { kind: 'widget', id: 'alloc' }, style: { backgroundColor: '#eee' } },
      { target: { kind: 'widget', id: 'hdr' }, label: 'Cancelled' },
    ];
    const merged = applyActions(second, applyActions(first));
    expect(merged.alloc.hidden).toBe(true);
    expect(merged.alloc.style).toEqual({ color: '#111', backgroundColor: '#eee' });
    expect(merged.hdr.label).toBe('Cancelled');
  });
});
