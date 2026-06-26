import { buildMatchSet, filterValidationRulesForEntity } from '../validationRules';

describe('validationRules utils', () => {
  const entityKey = 'orders';
  const entity = {
    name: 'Orders',
    businessName: 'Order',
    technicalName: 'orders',
    subtypes: {
      line_item: { name: 'LineItem', technicalName: 'line_item' },
      discount: { name: 'Discount', technicalName: 'discount' }
    }
  };

  const rules = [
    { id: 'r1', target_entity: 'Orders' },
    { id: 'r2', target_entities: ['line_item'] },
    { id: 'r3', entity: 'Other' },
    { id: 'r4', sub_entity_type: 'discount' },
    { id: 'r5', target_entity: 'orders' },
    { id: 'r6', target_entity_id: 'orders' }
  ];

  test('buildMatchSet includes keys and subtype identifiers', () => {
    const s = buildMatchSet(entityKey, entity as any);
    expect(s.has('orders')).toBeTruthy();
    expect(s.has('order')).toBeTruthy();
    expect(s.has('line_item')).toBeTruthy();
    expect(s.has('discount')).toBeTruthy();
  });

  test('filterValidationRulesForEntity selects only matching rules', () => {
    const filtered = filterValidationRulesForEntity(entityKey, entity as any, rules as any[]);
    const ids = filtered.map((r: any) => r.id).sort();
    expect(ids).toEqual(['r1', 'r2', 'r4', 'r5', 'r6'].sort());
  });
});
