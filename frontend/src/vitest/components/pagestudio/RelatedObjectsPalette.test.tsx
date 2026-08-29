import { describe, it, expect, vi } from 'vitest';
import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { RelatedObjectsPalette } from '../../../components/pagestudio/RelatedObjectsPalette';
import type { RelationshipResult } from '../../../components/pagestudio/pageStudioTypes';

const rel = (over: Partial<RelationshipResult>): RelationshipResult => ({
  id: 'r1', relKey: 'rk', direction: 'outbound', relatedObjectName: 'Name', relatedBoKey: 'key',
  targetObjectId: 'bo-1', relationshipType: 'RELATED_TO', cardinality: '1:M', ...over,
});

describe('RelatedObjectsPalette', () => {
  it('splits to-many (draggable) from to-one (read-only) relationships', () => {
    const onAdd = vi.fn();
    const relationships = [
      rel({ id: 'r-1m', relatedObjectName: 'Allocations', cardinality: '1:M' }),
      rel({ id: 'r-mm', relatedObjectName: 'Tags', cardinality: 'M:M' }),
      rel({ id: 'r-11', relatedObjectName: 'PrimaryContact', cardinality: '1:1' }),
      rel({ id: 'r-m1', relatedObjectName: 'ParentAccount', cardinality: 'M:1' }),
    ];
    render(<RelatedObjectsPalette relationships={relationships} onAddRelatedObject={onAdd} />);

    expect(screen.getByText('Allocations')).toBeInTheDocument();
    expect(screen.getByText('Tags')).toBeInTheDocument();
    expect(screen.getByText('PrimaryContact')).toBeInTheDocument();
    expect(screen.getByText('ParentAccount')).toBeInTheDocument();
    expect(screen.getByText('Reference cards (shown automatically)')).toBeInTheDocument();

    fireEvent.click(screen.getByLabelText('Add Allocations'));
    expect(onAdd).toHaveBeenCalledWith(relationships[0]);
  });

  it('shows an empty state when there are no relationships', () => {
    render(<RelatedObjectsPalette relationships={[]} onAddRelatedObject={() => {}} />);
    expect(screen.getByText(/No relationships found/i)).toBeInTheDocument();
  });

  it('shows a loading state while relationships are null', () => {
    render(<RelatedObjectsPalette relationships={null} onAddRelatedObject={() => {}} />);
    expect(screen.getByText(/Loading relationships/i)).toBeInTheDocument();
  });
});
