import { mergeProperties } from '../mergeProperties';

describe('mergeProperties', () => {
  test('returns undefined when both inputs missing', () => {
    expect(mergeProperties(undefined, undefined)).toBeUndefined();
  });

  test('prefers instance values over type defaults (shallow)', () => {
    const def = { a: 1, b: 2 };
    const inst = { b: 20, c: 3 };
    const merged = mergeProperties(def, inst);
    expect(merged).toEqual({ a: 1, b: 20, c: 3 });
  });

  test('deep merges plain object values one level', () => {
    const def = { settings: { nullable: false, max: 100 }, a: 1 };
    const inst = { settings: { max: 50, extra: true } };
    const merged = mergeProperties(def, inst);
    expect(merged).toEqual({ a: 1, settings: { nullable: false, max: 50, extra: true } });
  });

  test('arrays and primitives from instance override defaults', () => {
    const def = { list: [1,2,3], x: { y: 1 } };
    const inst = { list: [9], x: 'string' };
    const merged = mergeProperties(def, inst);
    expect(merged).toEqual({ list: [9], x: 'string' });
  });
});
