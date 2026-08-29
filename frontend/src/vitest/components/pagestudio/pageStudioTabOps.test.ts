import { describe, it, expect } from 'vitest';
import { addTab, renameTab, removeTab, moveTab, newTab, type PageTab } from '../../../components/pagestudio/pageStudioTypes';

describe('pageStudioTypes tab ops', () => {
  it('adds a new tab', () => {
    const tabs = [newTab('Details')];
    const next = addTab(tabs, 'Extra');
    expect(next).toHaveLength(2);
    expect(next[1].title).toBe('Extra');
    expect(tabs).toHaveLength(1); // original untouched
  });

  it('renames a tab by id', () => {
    const tabs: PageTab[] = [newTab('Details')];
    const next = renameTab(tabs, tabs[0].id, 'Renamed');
    expect(next[0].title).toBe('Renamed');
    expect(tabs[0].title).toBe('Details');
  });

  it('removes a tab, but never the last one', () => {
    const tabs = addTab([newTab('A')], 'B');
    const next = removeTab(tabs, tabs[0].id);
    expect(next).toHaveLength(1);
    expect(next[0].title).toBe('B');

    const noop = removeTab(next, next[0].id);
    expect(noop).toBe(next); // no-op: would leave zero tabs
  });

  it('moves a tab left/right', () => {
    const tabs = addTab(addTab([newTab('A')], 'B'), 'C');
    const moved = moveTab(tabs, 0, 1);
    expect(moved.map((t) => t.title)).toEqual(['B', 'A', 'C']);

    const noop = moveTab(tabs, 0, -1);
    expect(noop).toBe(tabs); // can't move first tab left
  });
});
