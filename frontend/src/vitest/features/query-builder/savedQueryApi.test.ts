import { describe, it, expect } from 'vitest';
import {
  isRowGrainIntact,
  isSafeToRollUpAcrossRows,
  type SavedQueryRunResult,
} from '../../../features/query-builder/services/savedQueryApi';

// These pin the one property the fan-out cardinality trace kept insisting
// on at every seam: ABSENCE of a safety signal must degrade to "unsafe,"
// never to "probably fine." Every test below either sets a field to
// undefined/missing or a value that isn't the recognized safe verdict, and
// expects false - if any of these ever return true, the gate has silently
// flipped its failure direction.

function resultWith(columns: SavedQueryRunResult['columns'], hasRelatedBOs?: boolean): SavedQueryRunResult {
  return { columns, rows: [], rowCount: 0, chartType: 'bar', name: 'test', hasRelatedBOs };
}

describe('isRowGrainIntact', () => {
  it('is true when hasRelatedBOs is explicitly false, regardless of column metadata', () => {
    expect(isRowGrainIntact(resultWith([{ name: 'x', type: 'number' }], false))).toBe(true);
  });

  it('is true when every column is explicitly cardinality "one"', () => {
    expect(isRowGrainIntact(resultWith([{ name: 'x', type: 'number', cardinality: 'one' }], true))).toBe(true);
  });

  it('is false when a column has cardinality "many"', () => {
    expect(isRowGrainIntact(resultWith([{ name: 'x', type: 'number', cardinality: 'many' }], true))).toBe(false);
  });

  it('is false when a column has cardinality "unresolved"', () => {
    expect(isRowGrainIntact(resultWith([{ name: 'x', type: 'number', cardinality: 'unresolved' }], true))).toBe(false);
  });

  // The absence case - the one this whole test file exists to pin.
  it('is false when a column is MISSING cardinality entirely, even with hasRelatedBOs true', () => {
    expect(isRowGrainIntact(resultWith([{ name: 'x', type: 'number' }], true))).toBe(false);
  });

  it('is false when hasRelatedBOs itself is missing (undefined, not false) and a column has no cardinality', () => {
    expect(isRowGrainIntact(resultWith([{ name: 'x', type: 'number' }], undefined))).toBe(false);
  });

  it('is false when ONE column of several lacks cardinality, even if the rest are clean', () => {
    expect(
      isRowGrainIntact(
        resultWith(
          [
            { name: 'a', type: 'number', cardinality: 'one' },
            { name: 'b', type: 'number' }, // missing
          ],
          true,
        ),
      ),
    ).toBe(false);
  });
});

describe('isSafeToRollUpAcrossRows', () => {
  it('is true for a unique-ownership column when hasRelatedBOs is false', () => {
    const result = resultWith([{ name: 'x', type: 'number' }], false);
    expect(isSafeToRollUpAcrossRows(result, result.columns[0])).toBe(true);
  });

  it('is true for a unique-ownership column when the query row grain is intact', () => {
    const result = resultWith([{ name: 'x', type: 'number', cardinality: 'one', rootOwnership: 'unique' }], true);
    expect(isSafeToRollUpAcrossRows(result, result.columns[0])).toBe(true);
  });

  it('is false when rootOwnership is MISSING, even though cardinality is "one"', () => {
    const result = resultWith([{ name: 'x', type: 'number', cardinality: 'one' }], true);
    expect(isSafeToRollUpAcrossRows(result, result.columns[0])).toBe(false);
  });

  it('is false when rootOwnership is "shared"', () => {
    const result = resultWith([{ name: 'x', type: 'number', cardinality: 'one', rootOwnership: 'shared' }], true);
    expect(isSafeToRollUpAcrossRows(result, result.columns[0])).toBe(false);
  });

  it('is false when rootOwnership is "unresolved"', () => {
    const result = resultWith([{ name: 'x', type: 'number', cardinality: 'one', rootOwnership: 'unresolved' }], true);
    expect(isSafeToRollUpAcrossRows(result, result.columns[0])).toBe(false);
  });

  // The worked example from the trace: a clean column (unique ownership,
  // "one" cardinality) sitting next to a DIFFERENT column whose join fans
  // out. The measure column's own metadata is perfect; the query's row
  // grain is not. Must still be false.
  it('is false when the target column is clean but a DIFFERENT column in the result fans out', () => {
    const result = resultWith(
      [
        { name: 'seats', type: 'number', cardinality: 'one', rootOwnership: 'unique' }, // the measure
        { name: 'subcategory_label', type: 'string', cardinality: 'many', rootOwnership: 'unique' }, // a different, fanning-out column
      ],
      true,
    );
    const measureCol = result.columns[0];
    expect(isSafeToRollUpAcrossRows(result, measureCol)).toBe(false);
  });

  it('is false when both cardinality and rootOwnership are entirely absent (single-BO metadata gap without hasRelatedBOs stated)', () => {
    const result = resultWith([{ name: 'x', type: 'number' }], undefined);
    expect(isSafeToRollUpAcrossRows(result, result.columns[0])).toBe(false);
  });
});
