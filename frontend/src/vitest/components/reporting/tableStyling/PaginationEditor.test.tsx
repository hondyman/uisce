import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import React from 'react';
import { PaginationEditor } from '../../../../../src/components/reporting/tableStyling/PaginationEditor';
import { createDefaultPaginationConfig } from '../../../../../src/components/reporting/tableColumnModel';

describe('PaginationEditor', () => {
  const defaultPagination = createDefaultPaginationConfig();

  it('renders mode buttons with expand selected by default', () => {
    render(<PaginationEditor pagination={defaultPagination} onChange={vi.fn()} />);
    expect(screen.getByText('Show All Rows')).toBeInTheDocument();
    expect(screen.getByText('Paginate')).toBeInTheDocument();
  });

  it('does not show rows per page field when mode is expand', () => {
    render(<PaginationEditor pagination={defaultPagination} onChange={vi.fn()} />);
    expect(screen.queryByLabelText('Rows per page')).not.toBeInTheDocument();
  });

  it('shows rows per page field when mode is paginate', () => {
    const paginatePagination = { ...defaultPagination, mode: 'paginate' as const };
    render(<PaginationEditor pagination={paginatePagination} onChange={vi.fn()} />);
    expect(screen.getByLabelText('Rows per page')).toBeInTheDocument();
  });

  it('calls onChange when expand mode is selected', () => {
    const handleChange = vi.fn();
    const paginatePagination = { ...defaultPagination, mode: 'paginate' as const };
    render(<PaginationEditor pagination={paginatePagination} onChange={handleChange} />);
    fireEvent.click(screen.getByText('Show All Rows'));
    expect(handleChange).toHaveBeenCalledWith(
      expect.objectContaining({ mode: 'expand' })
    );
  });

  it('calls onChange when paginate mode is selected', () => {
    const handleChange = vi.fn();
    render(<PaginationEditor pagination={defaultPagination} onChange={handleChange} />);
    fireEvent.click(screen.getByText('Paginate'));
    expect(handleChange).toHaveBeenCalledWith(
      expect.objectContaining({ mode: 'paginate' })
    );
  });

  it('does not show repeat headers switch when mode is expand', () => {
    render(<PaginationEditor pagination={defaultPagination} onChange={vi.fn()} />);
    expect(screen.queryByText('Repeat headers on each page')).not.toBeInTheDocument();
  });

  it('shows repeat headers switch when mode is paginate', () => {
    const paginatePagination = { ...defaultPagination, mode: 'paginate' as const };
    render(<PaginationEditor pagination={paginatePagination} onChange={vi.fn()} />);
    expect(screen.getByText('Repeat headers on each page')).toBeInTheDocument();
  });

  it('does not show page total section when mode is expand', () => {
    render(<PaginationEditor pagination={defaultPagination} onChange={vi.fn()} />);
    expect(screen.queryByText('Page total row')).not.toBeInTheDocument();
  });

  it('shows page total switch when mode is paginate', () => {
    const paginatePagination = { ...defaultPagination, mode: 'paginate' as const };
    render(<PaginationEditor pagination={paginatePagination} onChange={vi.fn()} />);
    expect(screen.getByText('Page total row')).toBeInTheDocument();
  });

  it('shows page total label and position when page total is enabled', () => {
    const paginatePagination = {
      ...defaultPagination,
      mode: 'paginate' as const,
      pageTotalEnabled: true,
    };
    render(<PaginationEditor pagination={paginatePagination} onChange={vi.fn()} />);
    expect(screen.getByLabelText('Label')).toBeInTheDocument();
    expect(screen.getByText('Top')).toBeInTheDocument();
    expect(screen.getByText('Bottom')).toBeInTheDocument();
  });

  it('calls onChange with updated rowsPerPage when changed', () => {
    const handleChange = vi.fn();
    const paginatePagination = { ...defaultPagination, mode: 'paginate' as const };
    render(<PaginationEditor pagination={paginatePagination} onChange={handleChange} />);
    const input = screen.getByLabelText('Rows per page');
    fireEvent.change(input, { target: { value: '30' } });
    expect(handleChange).toHaveBeenCalledWith(
      expect.objectContaining({ rowsPerPage: 30 })
    );
  });

  it('calls onChange with pageTotalPosition top when Top button clicked', () => {
    const handleChange = vi.fn();
    const paginatePagination = {
      ...defaultPagination,
      mode: 'paginate' as const,
      pageTotalEnabled: true,
    };
    render(<PaginationEditor pagination={paginatePagination} onChange={handleChange} />);
    fireEvent.click(screen.getByText('Top'));
    expect(handleChange).toHaveBeenCalledWith(
      expect.objectContaining({ pageTotalPosition: 'top' })
    );
  });

  it('calls onChange with pageTotalPosition bottom when Bottom button clicked', () => {
    const handleChange = vi.fn();
    const paginatePagination = {
      ...defaultPagination,
      mode: 'paginate' as const,
      pageTotalEnabled: true,
      pageTotalPosition: 'top' as const,
    };
    render(<PaginationEditor pagination={paginatePagination} onChange={handleChange} />);
    fireEvent.click(screen.getByText('Bottom'));
    expect(handleChange).toHaveBeenCalledWith(
      expect.objectContaining({ pageTotalPosition: 'bottom' })
    );
  });

  it('calls onChange with pageTotalLabel when label changed', () => {
    const handleChange = vi.fn();
    const paginatePagination = {
      ...defaultPagination,
      mode: 'paginate' as const,
      pageTotalEnabled: true,
    };
    render(<PaginationEditor pagination={paginatePagination} onChange={handleChange} />);
    const input = screen.getByLabelText('Label');
    fireEvent.change(input, { target: { value: 'Running Total' } });
    expect(handleChange).toHaveBeenCalledWith(
      expect.objectContaining({ pageTotalLabel: 'Running Total' })
    );
  });
});
