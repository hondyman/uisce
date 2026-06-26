import { getJsonSchemaForProperty } from '../propertyHelpers';

describe('getJsonSchemaForProperty', () => {
  test('returns null for non-json properties', () => {
    const schema = getJsonSchemaForProperty({ name: 'a', data_type: 'string', input_type: 'text' } as any);
    expect(schema).toBeNull();
  });

  test('returns user-provided jsonSchema when present', () => {
    const custom = { type: 'object', properties: { foo: { type: 'string' } } };
    const schema = getJsonSchemaForProperty({ name: 'a', data_type: 'json', input_type: 'json-editor', validation: { jsonSchema: custom } } as any);
    expect(schema).toEqual(custom);
  });

  test('produces permissive schema for json-editor', () => {
    const schema = getJsonSchemaForProperty({ name: 'x', data_type: 'json', input_type: 'json-editor' } as any);
    expect(schema).not.toBeNull();
    expect(schema).toHaveProperty('type');
  });

  test('includes enum values when enumValues provided', () => {
    const schema = getJsonSchemaForProperty({ name: 'a', data_type: 'json', input_type: 'json-editor', enumValues: ['x','y'] } as any);
    expect(schema).toHaveProperty('enum');
    expect((schema as any).enum).toEqual(['x','y']);
  });

  test('creates item schema for typed arrays', () => {
    const schema = getJsonSchemaForProperty({ name: 'arr', data_type: 'json', input_type: 'json-editor', format: 'array', itemsType: 'string' } as any);
    expect(schema).toHaveProperty('type', 'array');
    expect(schema).toHaveProperty('items');
    expect((schema as any).items).toHaveProperty('type', 'string');
  });
});
