import { describe, it, expect, vi } from 'vitest';
import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { PageStudioCanvas } from '../../../components/pagestudio/PageStudioCanvas';
import { addFieldToContainer, type CanvasWidget } from '../../../components/pagestudio/pageStudioTypes';
import type { BOField } from '../../../components/reporting/BOFieldsPalette';

function makeDataTransfer(payload: any) {
  return {
    getData: (fmt: string) => (fmt === 'application/json' ? JSON.stringify(payload) : ''),
    setData: () => {},
  };
}

const field: BOField = { name: 'sponsor_id', technicalName: 'sponsor_id', label: 'Sponsor ID', dataType: 'string' };

describe('PageStudioCanvas', () => {
  it('shows an empty-state hint when there are no widgets', () => {
    render(<PageStudioCanvas canvas={[]} onChange={() => {}} />);
    expect(screen.getByText(/Drag fields from the BO Fields tab/i)).toBeInTheDocument();
  });

  it('adds a field widget on native drop onto the root canvas', () => {
    const onChange = vi.fn();
    render(<PageStudioCanvas canvas={[]} onChange={onChange} />);

    const root = screen.getByTestId('page-studio-canvas-root');
    fireEvent.drop(root, { dataTransfer: makeDataTransfer({ type: 'bofield', field }) });

    expect(onChange).toHaveBeenCalledTimes(1);
    const next = onChange.mock.calls[0][0] as CanvasWidget[];
    expect(next).toHaveLength(1);
    expect((next[0] as any).fieldKey).toBe('sponsor_id');
  });

  it('moves and removes a field widget via inline controls', () => {
    let canvas: CanvasWidget[] = [];
    canvas = addFieldToContainer(canvas, null, { ...field, name: 'a', technicalName: 'a', label: 'A' });
    canvas = addFieldToContainer(canvas, null, { ...field, name: 'b', technicalName: 'b', label: 'B' });

    const onChange = vi.fn();
    const { rerender } = render(<PageStudioCanvas canvas={canvas} onChange={onChange} />);

    fireEvent.click(screen.getAllByLabelText('move down')[0]);
    let next = onChange.mock.calls[0][0] as CanvasWidget[];
    expect((next[0] as any).fieldKey).toBe('b');

    rerender(<PageStudioCanvas canvas={next} onChange={onChange} />);
    fireEvent.click(screen.getAllByLabelText('remove')[0]);
    next = onChange.mock.calls[1][0] as CanvasWidget[];
    expect(next).toHaveLength(1);
  });

  it('delegates a relatedobject drop to onAddRelatedObject instead of mutating the canvas directly', () => {
    const onChange = vi.fn();
    const onAddRelatedObject = vi.fn();
    const relationship = { id: 'rel-1', relKey: 'allocations', direction: 'outbound', relatedObjectName: 'Account', relatedBoKey: 'account', targetObjectId: 'bo-account', relationshipType: 'RELATED_TO', cardinality: '1:M' };
    render(<PageStudioCanvas canvas={[]} onChange={onChange} onAddRelatedObject={onAddRelatedObject} />);

    const root = screen.getByTestId('page-studio-canvas-root');
    fireEvent.drop(root, { dataTransfer: makeDataTransfer({ type: 'relatedobject', relationship }) });

    expect(onAddRelatedObject).toHaveBeenCalledWith(null, relationship);
    expect(onChange).not.toHaveBeenCalled();
  });

  it('renders a relatedObject widget as a bound-config preview card', () => {
    const canvas: CanvasWidget[] = [{
      id: 'ro-1', type: 'relatedObject', title: 'Account', relationshipId: 'rel-1', relKey: 'allocations',
      targetBoId: 'bo-account', targetBoKey: 'account', cardinality: '1:M', displayColumns: ['id'],
    }];
    render(<PageStudioCanvas canvas={canvas} onChange={() => {}} />);
    expect(screen.getByText('Account')).toBeInTheDocument();
    expect(screen.getByText(/1:M.*account/)).toBeInTheDocument();
  });
});
