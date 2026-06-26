/* eslint-disable no-console */

import Ajv from 'ajv';
import addFormats from 'ajv-formats';
// eslint-disable-next-line @typescript-eslint/no-var-requires
const schema = require('../../../schema/cube-semantic.json');

describe('cube semantic schema', () => {
  const ajv = new Ajv({ allErrors: true });
  addFormats(ajv);
  ajv.addSchema(schema, 'cube-semantic');

  it('validates a minimal valid cube', () => {
    const doc = {
      name: 'my_cube',
      tenant_instance_id: 'ds_1',
      sql: 'select 1 as id',
      measures: {
        total: { sql: 'count(*)', type: 'count' }
      },
      dimensions: {
        id: { sql: 'id', type: 'number' }
      }
    };
    const valid = ajv.validate('cube-semantic', doc);
    if (!valid) console.error(ajv.errors);
    expect(valid).toBe(true);
  });

  it('rejects an invalid cube with missing required measure sql', () => {
    const doc = {
      name: 'bad_cube',
      measures: { bad: { type: 'sum' } }
    };
    const valid = ajv.validate('cube-semantic', doc);
    expect(valid).toBe(false);
    expect(ajv.errors && ajv.errors.length).toBeGreaterThan(0);
  });

  it('accepts a pre_aggregation example', () => {
    const doc = {
      name: 'agg_cube',
      pre_aggregations: [
        { name: 'pa1', type: 'rollup', time_dimension: 'created_at', dimensions: ['country'] }
      ]
    };
    const valid = ajv.validate('cube-semantic', doc);
    if (!valid) console.error(ajv.errors);
    expect(valid).toBe(true);
  });
});
