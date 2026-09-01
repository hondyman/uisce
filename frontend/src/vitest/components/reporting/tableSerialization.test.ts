import { describe, it, expect } from 'vitest';
import {
  migrateV1ToV2,
  needsMigration,
  deserializeFromBackend,
  serializeForBackend,
  BuilderDefinition,
} from '../../../../src/components/reporting/tableSerialization';
import { createDefaultPaginationConfig } from '../../../../src/components/reporting/tableColumnModel';

describe('tableSerialization', () => {
  describe('needsMigration', () => {
    it('returns true for v1 schema (no version field)', () => {
      const v1: BuilderDefinition = {
        _schemaVersion: 1,
        elements: [],
      };
      expect(needsMigration(v1)).toBe(true);
    });

    it('returns true for v0 schema', () => {
      const v0 = { elements: [] } as unknown as BuilderDefinition;
      expect(needsMigration(v0)).toBe(true);
    });

    it('returns false for v2 schema', () => {
      const v2: BuilderDefinition = {
        _schemaVersion: 2,
        elements: [],
      };
      expect(needsMigration(v2)).toBe(false);
    });

    it('returns false when schema version equals current', () => {
      const current: BuilderDefinition = {
        _schemaVersion: 2,
        elements: [],
      };
      expect(needsMigration(current)).toBe(false);
    });
  });

  describe('migrateV1ToV2', () => {
    it('converts string[] columns to ColumnConfig[]', () => {
      const v1: BuilderDefinition = {
        _schemaVersion: 1,
        elements: [
          {
            id: 'el1',
            type: 'table',
            properties: {
              name: 'Test Table',
              columns: ['fieldA', 'fieldB', 'fieldC'],
            },
          },
        ],
      };

      const migrated = migrateV1ToV2(v1);

      expect(migrated._schemaVersion).toBe(2);
      const tableEl = migrated.elements[0];
      const cols = (tableEl.properties as any).columns;
      expect(cols).toBeInstanceOf(Array);
      expect(cols.length).toBe(3);
      expect(cols[0]).toHaveProperty('id');
      expect(cols[0]).toHaveProperty('field', 'fieldA');
      expect(cols[0]).toHaveProperty('headerText', 'fieldA');
      expect(cols[0]).toHaveProperty('visible', true);
      expect(cols[0]).toHaveProperty('headerStyle');
      expect(cols[0]).toHaveProperty('bodyStyle');
    });

    it('migrates fontSize from v1 to bodyStyle and headerStyle', () => {
      const v1: BuilderDefinition = {
        _schemaVersion: 1,
        elements: [
          {
            id: 'el1',
            type: 'table',
            properties: {
              name: 'Test',
              columns: ['Amount'],
              fontSize: 14,
            },
          },
        ],
      };

      const migrated = migrateV1ToV2(v1);
      const col = ((migrated.elements[0].properties as any).columns as any[])[0];
      expect(col.bodyStyle.fontSize).toBe(14);
      expect(col.headerStyle.fontSize).toBe(14);
    });

    it('migrates textColor from v1 to bodyStyle and headerStyle', () => {
      const v1: BuilderDefinition = {
        _schemaVersion: 1,
        elements: [
          {
            id: 'el1',
            type: 'table',
            properties: {
              name: 'Test',
              columns: ['Amount'],
              textColor: '#FF0000',
            },
          },
        ],
      };

      const migrated = migrateV1ToV2(v1);
      const col = ((migrated.elements[0].properties as any).columns as any[])[0];
      expect(col.bodyStyle.color).toBe('#FF0000');
      expect(col.headerStyle.color).toBe('#FF0000');
    });

    it('migrates backgroundColor from v1 to bodyStyle', () => {
      const v1: BuilderDefinition = {
        _schemaVersion: 1,
        elements: [
          {
            id: 'el1',
            type: 'table',
            properties: {
              name: 'Test',
              columns: ['Amount'],
              backgroundColor: '#FFFF00',
            },
          },
        ],
      };

      const migrated = migrateV1ToV2(v1);
      const col = ((migrated.elements[0].properties as any).columns as any[])[0];
      expect(col.bodyStyle.backgroundColor).toBe('#FFFF00');
    });

    it('leaves non-table/matrix elements unchanged', () => {
      const v1: BuilderDefinition = {
        _schemaVersion: 1,
        elements: [
          {
            id: 'text1',
            type: 'textbox',
            properties: { text: 'Hello', fontSize: 10 },
          },
          {
            id: 'chart1',
            type: 'chart',
            properties: { chartType: 'bar' },
          },
        ],
      };

      const migrated = migrateV1ToV2(v1);

      expect(migrated.elements.length).toBe(2);
      expect(migrated.elements[0]).toEqual(v1.elements[0]);
      expect(migrated.elements[1]).toEqual(v1.elements[1]);
    });

    it('is idempotent — running twice produces same result', () => {
      const v1: BuilderDefinition = {
        _schemaVersion: 1,
        elements: [
          {
            id: 'el1',
            type: 'table',
            properties: {
              name: 'Test',
              columns: ['A', 'B'],
              fontSize: 12,
            },
          },
        ],
      };

      const first = migrateV1ToV2(v1);
      const second = migrateV1ToV2(first);

      expect(second._schemaVersion).toBe(2);
      const firstCols = (first.elements[0].properties as any).columns;
      const secondCols = (second.elements[0].properties as any).columns;
      expect(secondCols.length).toBe(firstCols.length);
      expect(secondCols.length).toBe(2);
    });

    it('sets _schemaVersion to 2 on migrated definition', () => {
      const v1: BuilderDefinition = {
        _schemaVersion: 1,
        elements: [
          {
            id: 'el1',
            type: 'table',
            properties: { name: 'Test', columns: ['X'] },
          },
        ],
      };

      const migrated = migrateV1ToV2(v1);
      expect(migrated._schemaVersion).toBe(2);
      expect((migrated.elements[0].properties as any)._schemaVersion).toBe(2);
    });

    it('creates default totals and banding configs for migrated table', () => {
      const v1: BuilderDefinition = {
        _schemaVersion: 1,
        elements: [
          {
            id: 'el1',
            type: 'table',
            properties: { name: 'Test', columns: ['X'] },
          },
        ],
      };

      const migrated = migrateV1ToV2(v1);
      const props = migrated.elements[0].properties as any;
      expect(props.totals).toBeDefined();
      expect(props.banding).toBeDefined();
    });

    it('migrated columns have aggregate.autoEnabled: false for future opt-in', () => {
      const v1: BuilderDefinition = {
        _schemaVersion: 1,
        elements: [
          {
            id: 'el1',
            type: 'table',
            properties: { name: 'Test', columns: ['A', 'B'] },
          },
        ],
      };

      const migrated = migrateV1ToV2(v1);
      const cols = (migrated.elements[0].properties as any).columns;
      expect(cols.length).toBe(2);
      expect(cols[0].aggregate).toEqual({ enabled: false, function: 'SUM', scope: 'column' });
      expect(cols[1].aggregate).toEqual({ enabled: false, function: 'SUM', scope: 'column' });
    });
  });

  describe('serializeForBackend', () => {
    it('produces v2 schema with _schemaVersion: 2', () => {
      const def: BuilderDefinition = {
        _schemaVersion: 2,
        elements: [
          {
            id: 'el1',
            type: 'table',
            properties: {
              name: 'Sales Table',
              columns: [
                {
                  id: 'col1',
                  field: 'amount',
                  headerText: 'Amount',
                  widthPx: 120,
                  visible: true,
                  align: 'right',
                  verticalAlign: 'middle',
                  wrap: false,
                  formatType: 'Currency',
                  formatMask: '$#,##0.00',
                  formatPrefix: '$',
                  formatSuffix: '',
                  headerStyle: {},
                  bodyStyle: {},
                },
              ],
              totals: { grandTotal: { enabled: true, label: 'Grand Total' }, subtotals: { enabled: false } },
              banding: { type: 'none' },
            },
          },
        ],
        reportTitle: 'My Report',
      };

      const result = serializeForBackend(def);

      expect(result._schemaVersion).toBe(2);
      expect(result.title).toBe('My Report');
    });

    it('maps ColumnConfig fields correctly in serialized table element', () => {
      const def: BuilderDefinition = {
        _schemaVersion: 2,
        sections: [{ id: 's1', title: 'Main' }],
        elements: [
          {
            id: 'el1',
            type: 'table',
            section: 's1',
            properties: {
              name: 'Test',
              columns: [
                {
                  id: 'col1',
                  field: 'revenue',
                  headerText: 'Revenue',
                  widthPx: 150,
                  visible: true,
                  align: 'right',
                  verticalAlign: 'top',
                  wrap: true,
                  formatType: 'Decimal',
                  formatMask: '#,##0.00',
                  formatPrefix: '',
                  formatSuffix: 'K',
                  headerStyle: { color: '#000' },
                  bodyStyle: { fontSize: 11 },
                },
              ],
            },
          },
        ],
        reportTitle: 'Report',
      };

      const result = serializeForBackend(def);
      expect((result.sections as any[]).length).toBeGreaterThan(0);
      const section = (result.sections as any[])[0];
      expect(section.elements).toBeDefined();
      expect(section.elements.length).toBeGreaterThan(0);
      const tableEl = section.elements[0] as any;

      expect(tableEl).toBeDefined();
      expect(tableEl.id).toBe('el1');
      expect(tableEl.type).toBe('table');
      expect(tableEl.columns).toBeDefined();
      expect(tableEl.columns.length).toBe(1);
      expect(tableEl.columns[0].dimension).toBe('revenue');
      expect(tableEl.columns[0].label).toBe('Revenue');
      expect(tableEl.columns[0].width).toBe(150);
      expect(tableEl.columns[0].format).toBe('Decimal');
      expect(tableEl.columns[0].alignment).toBe('right');
      expect(tableEl.columns[0].verticalAlign).toBe('top');
      expect(tableEl.columns[0].wrap).toBe(true);
      expect(tableEl.columns[0].visible).toBe(true);
    });
  });

  describe('deserializeFromBackend', () => {
    it('triggers migration for v1 raw payload', () => {
      const raw = {
        _schemaVersion: 1,
        title: 'Legacy Report',
        sections: [
          {
            id: 'default',
            elements: [
              {
                id: 'el1',
                type: 'table',
                columns: ['Alpha', 'Beta'],
              },
            ],
          },
        ],
      };

      const def = deserializeFromBackend(raw as Record<string, unknown>);

      expect(def._schemaVersion).toBe(2);
      expect(def.elements.length).toBe(1);
      const cols = (def.elements[0].properties as any).columns;
      expect(cols.length).toBe(2);
      expect(cols[0].field).toBe('Alpha');
      expect(cols[1].field).toBe('Beta');
    });

    it('does not migrate v2 raw payload', () => {
      const raw = {
        _schemaVersion: 2,
        title: 'New Report',
        sections: [
          {
            id: 'default',
            elements: [
              {
                id: 'el1',
                type: 'table',
                columns: [
                  {
                    dimension: 'Gamma',
                    label: 'Gamma Label',
                    width: 200,
                    visible: true,
                  },
                ],
              },
            ],
          },
        ],
      };

      const def = deserializeFromBackend(raw as Record<string, unknown>);

      expect(def._schemaVersion).toBe(2);
      const cols = (def.elements[0].properties as any).columns;
      expect(cols.length).toBe(1);
      expect(cols[0].field).toBe('Gamma');
      expect(cols[0].headerText).toBe('Gamma Label');
    });

    it('maps backend column shape to ColumnConfig correctly on deserialization', () => {
      const raw = {
        _schemaVersion: 2,
        title: 'Test',
        sections: [
          {
            id: 'default',
            elements: [
              {
                id: 'el1',
                type: 'table',
                columns: [
                  {
                    dimension: 'delta_field',
                    label: 'Delta Field',
                    width: 300,
                    visible: false,
                    alignment: 'center',
                    verticalAlign: 'bottom',
                    wrap: true,
                    format: 'Integer',
                    formatMask: '0',
                    aggregate: { function: 'SUM', scope: 'column', enabled: true },
                    headerStyle: { color: '#FFF', backgroundColor: '#333' },
                    bodyStyle: { fontSize: 12 },
                  },
                ],
                totals: { grandTotal: { enabled: true, label: 'Total' }, subtotals: { enabled: true } },
              },
            ],
          },
        ],
      };

      const def = deserializeFromBackend(raw as Record<string, unknown>);
      const col = ((def.elements[0].properties as any).columns as any[])[0];

      expect(col.id).toBeDefined();
      expect(col.field).toBe('delta_field');
      expect(col.headerText).toBe('Delta Field');
      expect(col.widthPx).toBe(300);
      expect(col.visible).toBe(false);
      expect(col.align).toBe('center');
      expect(col.verticalAlign).toBe('bottom');
      expect(col.wrap).toBe(true);
      expect(col.formatType).toBe('Integer');
      expect(col.formatMask).toBe('0');
      expect(col.aggregate).toEqual({ function: 'SUM', scope: 'column', enabled: true });
      expect(col.headerStyle).toEqual({ color: '#FFF', backgroundColor: '#333' });
      expect(col.bodyStyle).toEqual({ fontSize: 12 });
    });
  });

  describe('pagination round-trip', () => {
    it('migrateV1ToV2 sets pagination mode to expand', () => {
      const v1: BuilderDefinition = {
        _schemaVersion: 1,
        elements: [
          {
            id: 'el1',
            type: 'table',
            properties: { name: 'Test', columns: ['A'] },
          },
        ],
      };
      const migrated = migrateV1ToV2(v1);
      const props = migrated.elements[0].properties as any;
      expect(props.pagination).toBeDefined();
      expect(props.pagination.mode).toBe('expand');
    });

    it('serializeForBackend includes pagination in output', () => {
      const def: BuilderDefinition = {
        _schemaVersion: 2,
        sections: [{ id: 's1', title: 'Main' }],
        elements: [
          {
            id: 'el1',
            type: 'table',
            section: 's1',
            properties: {
              name: 'Test',
              columns: [],
              pagination: createDefaultPaginationConfig(),
            },
          },
        ],
        reportTitle: 'Report',
      };
      const result = serializeForBackend(def);
      const section = (result.sections as any[])[0];
      const tableEl = section.elements[0] as any;
      expect(tableEl.pagination).toBeDefined();
      expect(tableEl.pagination.mode).toBe('expand');
      expect(tableEl.pagination.rowsPerPage).toBe(20);
      expect(tableEl.pagination.repeatHeadersOnEachPage).toBe(true);
    });

    it('deserializeFromBackend parses pagination from backend shape', () => {
      const raw = {
        _schemaVersion: 2,
        title: 'Test',
        sections: [
          {
            id: 'default',
            elements: [
              {
                id: 'el1',
                type: 'table',
                columns: [],
                pagination: {
                  mode: 'paginate',
                  rowsPerPage: 15,
                  repeatHeadersOnEachPage: true,
                  pageTotalEnabled: true,
                  pageTotalPosition: 'bottom',
                  pageTotalLabel: 'Running Total',
                },
              },
            ],
          },
        ],
      };
      const def = deserializeFromBackend(raw as Record<string, unknown>);
      const props = def.elements[0].properties as any;
      expect(props.pagination).toBeDefined();
      expect(props.pagination.mode).toBe('paginate');
      expect(props.pagination.rowsPerPage).toBe(15);
      expect(props.pagination.pageTotalEnabled).toBe(true);
      expect(props.pagination.pageTotalPosition).toBe('bottom');
      expect(props.pagination.pageTotalLabel).toBe('Running Total');
    });

    it('deserializeFromBackend uses default pagination when not present', () => {
      const raw = {
        _schemaVersion: 2,
        title: 'Test',
        sections: [
          {
            id: 'default',
            elements: [
              {
                id: 'el1',
                type: 'table',
                columns: [],
              },
            ],
          },
        ],
      };
      const def = deserializeFromBackend(raw as Record<string, unknown>);
      const props = def.elements[0].properties as any;
      expect(props.pagination).toBeDefined();
      expect(props.pagination.mode).toBe('expand');
    });
  });
});
