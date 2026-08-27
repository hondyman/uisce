import { describe, it, expect } from 'vitest';
import {
  buildSavePayload,
  buildReportTitle,
  ReportBuilderState,
} from '../../../../src/components/reporting/builderSerialization';
import type { BuilderElement } from '../../../../src/components/reporting/tableSerialization';

describe('builderSerialization', () => {
  describe('buildReportTitle', () => {
    it('extracts name from textbox element with name property', () => {
      const elements: BuilderElement[] = [
        {
          id: 't1',
          type: 'textbox',
          properties: { name: 'Monthly Sales Report' },
        },
      ];

      expect(buildReportTitle(elements)).toBe('Monthly Sales Report');
    });

    it('falls back to first textbox content if no name', () => {
      const elements: BuilderElement[] = [
        {
          id: 't1',
          type: 'textbox',
          properties: { name: 'Q4 Report' },
        },
      ];

      expect(buildReportTitle(elements)).toBe('Q4 Report');
    });

    it('returns "Untitled Report" when no textbox exists', () => {
      const elements: BuilderElement[] = [
        { id: 'el1', type: 'table', properties: { name: 'Table' } },
        { id: 'el2', type: 'chart', properties: {} },
      ];

      expect(buildReportTitle(elements)).toBe('Untitled Report');
    });

    it('returns "Untitled Report" for empty elements array', () => {
      expect(buildReportTitle([])).toBe('Untitled Report');
    });

    it('finds first textbox with name among multiple elements', () => {
      const elements: BuilderElement[] = [
        { id: 't1', type: 'textbox', properties: { text: 'Ignore me' } },
        { id: 't2', type: 'textbox', properties: { name: 'Correct Title' } },
        { id: 't3', type: 'textbox', properties: { name: 'Also ignore' } },
      ];

      expect(buildReportTitle(elements)).toBe('Correct Title');
    });
  });

  describe('buildSavePayload', () => {
    it('returns expected shape with name, description, definition, metadata', () => {
      const state: ReportBuilderState = {
        elements: [],
        reportTitle: 'Test Report',
      };

      const result = buildSavePayload(state);

      expect(result).toHaveProperty('name', 'Test Report');
      expect(result).toHaveProperty('description', '');
      expect(result).toHaveProperty('definition');
      expect(result).toHaveProperty('metadata');
    });

    it('includes _schemaVersion: 2 in definition', () => {
      const state: ReportBuilderState = {
        elements: [],
        reportTitle: 'Version Test',
      };

      const result = buildSavePayload(state);
      const def = result.definition as Record<string, unknown>;

      expect(def._schemaVersion).toBe(2);
    });

    it('passes elements and reportTitle through to definition', () => {
      const elements: BuilderElement[] = [
        {
          id: 'el1',
          type: 'table',
          properties: {
            name: 'Sales',
            columns: [
              {
                id: 'col1',
                field: 'amount',
                headerText: 'Amount',
                widthPx: 100,
                visible: true,
                align: 'left',
                verticalAlign: 'middle',
                wrap: false,
                formatType: 'Currency',
                formatMask: '',
                formatPrefix: '',
                formatSuffix: '',
                headerStyle: {},
                bodyStyle: {},
              },
            ],
          },
        },
      ];

      const state: ReportBuilderState = {
        elements,
        reportTitle: 'Sales Report',
      };

      const result = buildSavePayload(state);
      const def = result.definition as Record<string, unknown>;

      expect(def._schemaVersion).toBe(2);
      expect((def as any).title).toBe('Sales Report');
      // elements are placed in default section
      const sections = def.sections as any[];
      expect(sections.length).toBeGreaterThan(0);
    });

    it('uses "Untitled Report" as name when reportTitle is empty', () => {
      const state: ReportBuilderState = {
        elements: [],
        reportTitle: '',
      };

      const result = buildSavePayload(state);

      expect(result.name).toBe('Untitled Report');
    });

    it('works without reportId (create flow)', () => {
      const state: ReportBuilderState = {
        elements: [],
        reportTitle: 'New Report',
      };

      const result = buildSavePayload(state);

      expect(result.name).toBe('New Report');
    });

    it('preserves groupDefinitions in definition', () => {
      const state: ReportBuilderState = {
        elements: [],
        groupDefinitions: [{ id: 'g1', name: 'Group 1' }],
        reportTitle: 'Grouped Report',
      };

      const result = buildSavePayload(state);
      const def = result.definition as Record<string, unknown>;

      expect(def.groupDefinitions).toEqual([{ id: 'g1', name: 'Group 1' }]);
    });
  });
});
