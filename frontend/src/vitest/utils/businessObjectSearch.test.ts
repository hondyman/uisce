import { describe, expect, it } from 'vitest';
import { filterBusinessObjectsBySearch } from '../../utils/businessObjectSearch';

const bos = [
  { id: '1', name: 'security', display_name: 'Security', driver_table_name: '/orm/security', description: 'Security master' },
  { id: '2', name: 'party', display_name: 'Party', driver_table_name: '/mdm/party', category: 'MDM' },
  { id: '3', name: 'issuer', display_name: 'Issuer', driver_table_name: '/mdm/issuer' },
  { id: '4', name: 'orphan', display_name: null, driver_table_name: undefined },
];

const ids = (q: string) => filterBusinessObjectsBySearch(bos, q).map((b) => b.id);

describe('filterBusinessObjectsBySearch', () => {
  it('returns everything for an empty or blank query', () => {
    expect(ids('')).toEqual(['1', '2', '3', '4']);
    expect(ids('   ')).toEqual(['1', '2', '3', '4']);
  });
  it('matches as you type, case-insensitively, on name and display name', () => {
    expect(ids('sec')).toEqual(['1']);
    expect(ids('ISS')).toEqual(['3']);
  });
  it('matches the driving table and category', () => {
    expect(ids('/mdm/')).toEqual(['2', '3']);
    expect(ids('mdm')).toEqual(['2', '3']);
  });
  it('requires every word to match', () => {
    expect(ids('mdm party')).toEqual(['2']);
    expect(ids('mdm nothing')).toEqual([]);
  });
  it('matches the description', () => {
    expect(ids('master')).toEqual(['1']);
  });
  it('does not throw on objects with missing fields', () => {
    expect(ids('orph')).toEqual(['4']);
  });
});
