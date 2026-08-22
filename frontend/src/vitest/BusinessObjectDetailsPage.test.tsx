import { describe, it, expect } from 'vitest';
import { dedupeFields } from '../utils/dedupeFields';

describe('BusinessObjectDetailsPage Field Deduplication', () => {
  it('deduplicates overlapping coreFields and customFields by key', () => {
    const mockBO = {
      coreFields: [
        { id: '1', key: 'account_id', technicalName: 'account_id', name: 'Account ID' },
        { id: '2', key: 'account_name', technicalName: 'account_name', name: 'Account Name' },
      ],
      customFields: [
        { id: '1-dup', key: 'account_id', technicalName: 'account_id', name: 'Account ID' },
        { id: '3', key: 'custom_tax_status', technicalName: 'custom_tax_status', name: 'Tax Status' },
      ],
    };

    const raw = [...mockBO.coreFields, ...mockBO.customFields];
    const deduplicated = dedupeFields(raw);

    expect(deduplicated).toHaveLength(3);
    expect(deduplicated.map((f) => f.key)).toEqual([
      'account_id',
      'account_name',
      'custom_tax_status',
    ]);
  });

  it('preserves fields that lack key but have technicalName or id', () => {
    const fields = [
      { technicalName: 'cust_code', name: 'Customer Code' },
      { id: 'uuid-1234', name: 'Unnamed Field' },
      { technicalName: 'cust_code', name: 'Duplicate Customer Code' },
    ];

    const deduplicated = dedupeFields(fields);
    expect(deduplicated).toHaveLength(2);
    expect(deduplicated[0].technicalName).toBe('cust_code');
    expect(deduplicated[1].id).toBe('uuid-1234');
  });
});
