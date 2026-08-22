import { describe, it, expect } from 'vitest';

describe('BusinessObjectDetailsPage', () => {
  describe('dedupeFields regression', () => {
    it('removes duplicate fields when core and custom arrays overlap', async () => {
      const { dedupeFields } = await import('../../utils/dedupeFields');

      const coreFields = [
        { key: 'account_id', name: 'Account ID', technicalName: 'account_id' },
        { key: 'account_name', name: 'Account Name', technicalName: 'account_name' },
      ];

      const customFields = [
        { key: 'account_id', name: 'Account ID', technicalName: 'account_id' },
        { key: 'account_name', name: 'Account Name', technicalName: 'account_name' },
        { key: 'custodian_name', name: 'Custodian Name', technicalName: 'custodian_name' },
      ];

      const merged = [...coreFields, ...customFields];
      expect(merged).toHaveLength(5);

      const result = dedupeFields(merged);
      expect(result).toHaveLength(3);
      expect(result.map((f) => f.key)).toEqual(['account_id', 'account_name', 'custodian_name']);
    });
  });
});
