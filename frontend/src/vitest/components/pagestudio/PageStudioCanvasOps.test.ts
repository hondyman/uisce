import { describe, it, expect } from 'vitest';
import {
  addFieldToContainer,
  addWidget,
  moveItem,
  removeItem,
  newContainerWidget,
  type CanvasWidget,
} from '../../../components/pagestudio/pageStudioTypes';
import type { BOField } from '../../../components/reporting/BOFieldsPalette';

const field = (name: string, dataType = 'string'): BOField => ({ name, technicalName: name, label: name, dataType });

describe('pageStudioTypes canvas ops', () => {
  it('adds a field to the root canvas', () => {
    const canvas: CanvasWidget[] = [];
    const next = addFieldToContainer(canvas, null, field('sponsor_id'));
    expect(next).toHaveLength(1);
    expect(next[0].type).toBe('field');
    expect(canvas).toHaveLength(0); // original untouched
  });

  it('adds a field into a nested container', () => {
    const section = newContainerWidget('section', 'Sec 1');
    const canvas: CanvasWidget[] = [section];
    const next = addFieldToContainer(canvas, section.id, field('mandate_type'));
    const nextSection = next[0] as any;
    expect(nextSection.children).toHaveLength(1);
    expect(nextSection.children[0].fieldKey).toBe('mandate_type');
    // original tree is untouched (no shared references)
    expect((canvas[0] as any).children).toHaveLength(0);
  });

  it('adds a widget at root', () => {
    const next = addWidget([], null, 'row');
    expect(next).toHaveLength(1);
    expect(next[0].type).toBe('row');
  });

  it('moves a root-level item up and down', () => {
    let canvas: CanvasWidget[] = [];
    canvas = addFieldToContainer(canvas, null, field('a'));
    canvas = addFieldToContainer(canvas, null, field('b'));
    const moved = moveItem(canvas, [1], -1);
    expect((moved[0] as any).fieldKey).toBe('b');
    expect((moved[1] as any).fieldKey).toBe('a');
    // moving the first item up is a no-op
    const noop = moveItem(canvas, [0], -1);
    expect(noop).toBe(canvas);
  });

  it('moves a nested item within its container', () => {
    const section = newContainerWidget('section');
    let canvas: CanvasWidget[] = [section];
    canvas = addFieldToContainer(canvas, section.id, field('a'));
    canvas = addFieldToContainer(canvas, section.id, field('b'));
    const moved = moveItem(canvas, [0, 0], 1);
    const children = (moved[0] as any).children;
    expect(children[0].fieldKey).toBe('b');
    expect(children[1].fieldKey).toBe('a');
  });

  it('removes a root-level and nested item', () => {
    let canvas: CanvasWidget[] = [];
    canvas = addFieldToContainer(canvas, null, field('a'));
    canvas = addFieldToContainer(canvas, null, field('b'));
    const afterRemove = removeItem(canvas, [0]);
    expect(afterRemove).toHaveLength(1);
    expect((afterRemove[0] as any).fieldKey).toBe('b');

    const section = newContainerWidget('section');
    let nested: CanvasWidget[] = [section];
    nested = addFieldToContainer(nested, section.id, field('a'));
    const afterNestedRemove = removeItem(nested, [0, 0]);
    expect((afterNestedRemove[0] as any).children).toHaveLength(0);
  });
});
