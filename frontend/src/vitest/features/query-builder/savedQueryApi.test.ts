import { describe, it, expect } from 'vitest';
import {
  isRowGrainIntact,
  isSafeToRollUpAcrossRows,
  isAdditiveSafe,
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

// isAdditiveSafe is the SECOND of two independent signals (LINEARITY, not
// grain/ownership). Same fail-safe polarity as the row-grain tests above:
// every non-"sum" value, missing field, and unrecognized string must read
// false. If any of these ever return true, the gate has silently accepted
// an aggregation whose row values don't add up under `+`.
describe('isAdditiveSafe', () => {
  it('is true only for the single recognized additive aggregation: "sum"', () => {
    expect(isAdditiveSafe({ aggregation: 'sum' })).toBe(true);
  });

  // "avg" looks aggregate-y but a sum of averages is NOT the average of
  // sums unless every contributing group is the same size, which a
  // client-side widget cannot verify from column metadata alone.
  it('is false for "avg" - sum of averages is not additive unless groups are equipopulated', () => {
    expect(isAdditiveSafe({ aggregation: 'avg' })).toBe(false);
  });

  // The one that looks most intuitive to sum but isn't: COUNT-of-rows
  // summed across groups double-counts every row that contributes to
  // multiple groups (which is the common case once any related-BO
  // join fans out). The widget cannot verify group disjointness; the
  // gate must refuse by default.
  it('is false for "count" - summed counts double-count rows under any non-disjoint partitioning', () => {
    expect(isAdditiveSafe({ aggregation: 'count' })).toBe(false);
  });

  it('is false for "count_distinct" - distinct counts are not additive across overlapping groups', () => {
    expect(isAdditiveSafe({ aggregation: 'count_distinct' })).toBe(false);
  });

  it('is false for "min" and "max" - sums of minimums / maximums have no natural meaning', () => {
    expect(isAdditiveSafe({ aggregation: 'min' })).toBe(false);
    expect(isAdditiveSafe({ aggregation: 'max' })).toBe(false);
  });

  // QueryResultColumn's `json:"aggregation,omitempty"` drops the field
  // entirely for dimensions (Go marshals "" + omitempty as no key), so
  // an omitted-on-the-wire column arrives here as aggregation: undefined,
  // not as aggregation: "". The empty-string case is tested below as
  // defense-in-depth for any producer that emits the key with an empty
  // value rather than omitting it.
  it('is false for empty aggregation string - the "this column is not aggregated" sentinel', () => {
    expect(isAdditiveSafe({ aggregation: '' })).toBe(false);
  });

  it('is false when the aggregation field is missing entirely (undefined)', () => {
    expect(isAdditiveSafe({})).toBe(false);
  });

  // Unknown is unsafe, NOT lenient. A future backend field value the
  // frontend doesn't yet recognize must read as unsafe, not silently
  // permissive. This is the same fail-safe polarity rule as the
  // row-grain gate ("absence/missing ⇒ unsafe"): the day someone adds
  // an aggregation to the backend without teaching the frontend about
  // it, the gate stays defensive until they do.
  it('is false for unrecognized aggregation strings - unknown is unsafe, not lenient', () => {
    expect(isAdditiveSafe({ aggregation: 'median' })).toBe(false);
    expect(isAdditiveSafe({ aggregation: 'percentile_95' })).toBe(false);
    // And "none" - not a recognized aggregation string on this tree.
    // Same fail-safe polarity as the cases above.
    expect(isAdditiveSafe({ aggregation: 'none' })).toBe(false);
  });
});

// The polarity of hasRelatedBOs is deliberately asymmetric: absent is
// strict (treated as unsafe), explicit false is permissive (treated as
// safe-by-construction). That only holds if the two survive as distinct
// values across the actual wire transport, not just as distinct object
// literals built by hand in the tests above - a "sparse object builder"
// on either the backend or a future frontend refactor could collapse
// "key omitted" and "key present with value false" into the same thing,
// which would either silently disable the single-BO fast path (false
// read as absent) or silently open the permissive branch on real
// exposure (absent read as false) depending on which direction it drifts.
// This goes through actual JSON.stringify/JSON.parse, the same
// serialization boundary the real fetchJSON response crosses, rather
// than asserting on the object literal directly.
describe('hasRelatedBOs survives JSON round-trip as distinct from absence', () => {
  it('explicit false survives JSON.stringify/parse as false, not becoming undefined', () => {
    const wire = JSON.stringify({ columns: [{ name: 'x', type: 'number' }], hasRelatedBOs: false });
    const parsed = JSON.parse(wire) as SavedQueryRunResult;
    expect(parsed.hasRelatedBOs).toBe(false);
    expect(isRowGrainIntact(parsed)).toBe(true);
  });

  it('an omitted key survives JSON.stringify/parse as undefined, not becoming false', () => {
    const wire = JSON.stringify({ columns: [{ name: 'x', type: 'number' }] });
    const parsed = JSON.parse(wire) as SavedQueryRunResult;
    expect(parsed.hasRelatedBOs).toBeUndefined();
    expect(isRowGrainIntact(parsed)).toBe(false);
  });

  it('the two parsed results disagree on isRowGrainIntact - proving they were not collapsed to the same value', () => {
    const withFalse = JSON.parse(JSON.stringify({ columns: [{ name: 'x', type: 'number' }], hasRelatedBOs: false })) as SavedQueryRunResult;
    const omitted = JSON.parse(JSON.stringify({ columns: [{ name: 'x', type: 'number' }] })) as SavedQueryRunResult;
    expect(isRowGrainIntact(withFalse)).not.toBe(isRowGrainIntact(omitted));
  });
});
