import React from 'react';
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import {
  ReportTemplateGallery,
  scaffoldBlankGridReport,
  BUILT_IN_REPORT_TEMPLATES,
} from '../../pages/page-studio/ReportTemplateGallery';

describe('ReportTemplateGallery & Scaffolding (Phase 6.3)', () => {
  it('scaffoldBlankGridReport returns a layoutKind grid with parameterBar and starter tiles', () => {
    const draft = scaffoldBlankGridReport('Monthly Sales Overview');
    expect(draft.name).toBe('Monthly Sales Overview');
    expect(draft.layout?.nodes['root'].props?.layoutKind).toBe('grid');
    expect(draft.layout?.nodes['root'].props?.columns).toBe(12);

    const components = draft.components || [];
    const paramBar = components.find((c) => c.type === 'report_parameter_bar');
    expect(paramBar).toBeDefined();

    const kpiTile = components.find((c) => c.type === 'kpi_tile');
    expect(kpiTile).toBeDefined();

    const chart = components.find((c) => c.type === 'saved_query_widget');
    expect(chart).toBeDefined();

    expect(draft.app?.variables).toEqual([{ name: 'v_date_range', type: 'string', defaultValue: 'YTD' }]);
  });

  it('renders built-in templates with categories, parameter chips, and tile counts', () => {
    const onSelect = vi.fn();
    const onClose = vi.fn();

    render(<ReportTemplateGallery open={true} onClose={onClose} onSelectTemplate={onSelect} />);

    expect(screen.getByText('Report Template Gallery')).toBeDefined();
    expect(screen.getByText('Start Blank Grid Report')).toBeDefined();

    BUILT_IN_REPORT_TEMPLATES.forEach((tmpl) => {
      expect(screen.getByText(tmpl.name)).toBeDefined();
    });

    expect(screen.getByText('Executive KPI Dashboard')).toBeDefined();
    expect(screen.getByText('Financial & Trading Ledger')).toBeDefined();
    expect(screen.getByText('Portfolio Risk & Performance')).toBeDefined();
  });

  it('filters templates by category and search keyword', () => {
    const onSelect = vi.fn();
    const onClose = vi.fn();

    render(<ReportTemplateGallery open={true} onClose={onClose} onSelectTemplate={onSelect} />);

    // Filter by Trading category
    fireEvent.click(screen.getByText('Trading'));
    expect(screen.getByText('Financial & Trading Ledger')).toBeDefined();
    expect(screen.queryByText('Executive KPI Dashboard')).toBeNull();

    // Search input
    fireEvent.change(screen.getByPlaceholderText(/Search templates/i), { target: { value: 'Portfolio' } });
    fireEvent.click(screen.getByText('ALL'));
    expect(screen.getByText('Portfolio Risk & Performance')).toBeDefined();
    expect(screen.queryByText('Financial & Trading Ledger')).toBeNull();
  });

  it('instantiates selected template and passes constructed draft to onSelectTemplate', () => {
    const onSelect = vi.fn();
    const onClose = vi.fn();

    render(<ReportTemplateGallery open={true} onClose={onClose} onSelectTemplate={onSelect} />);

    fireEvent.click(screen.getByText('Executive KPI Dashboard'));
    expect(onSelect).toHaveBeenCalledTimes(1);

    const draft = onSelect.mock.calls[0][0];
    expect(draft.name).toBe('Executive KPI Dashboard');
    expect(draft.layout.nodes['root'].props.layoutKind).toBe('grid');
    expect(draft.components.length).toBe(6);
    expect(draft.app.variables.length).toBe(3);
    expect(onClose).toHaveBeenCalled();
  });

  it('clicking Start Blank Grid Report scaffolds a blank grid report', () => {
    const onSelect = vi.fn();
    const onClose = vi.fn();

    render(<ReportTemplateGallery open={true} onClose={onClose} onSelectTemplate={onSelect} />);

    fireEvent.click(screen.getByText('Start Blank Grid Report'));
    expect(onSelect).toHaveBeenCalledTimes(1);

    const draft = onSelect.mock.calls[0][0];
    expect(draft.name).toBe('New Report');
    expect(draft.layout.nodes['root'].props.layoutKind).toBe('grid');
    expect(onClose).toHaveBeenCalled();
  });
});
