import { describe, it, expect } from 'vitest';
import { resolveSubtypeFields, BusinessObject } from '../../utils/subtypeFieldResolver';

describe('subtypeFieldResolver', () => {
  const sampleBO: BusinessObject = {
    id: 'bo-1',
    key: 'account',
    name: 'Account',
    displayName: 'Account BO',
    coreFields: [
      { id: 'f1', key: 'account_number', technicalName: 'account_number', displayName: 'Account Number', dataType: 'string' },
      { id: 'f2', key: 'account_name', technicalName: 'account_name', displayName: 'Account Name', dataType: 'string' },
    ],
    subtypes: {
      institutional: {
        id: 'st-1',
        key: 'institutional',
        name: 'Institutional',
        displayName: 'Institutional Account',
        technicalName: 'institutional',
        subtypeFields: [
          { id: 'f3', key: 'sponsor_id', technicalName: 'sponsor_id', displayName: 'Sponsor ID', dataType: 'uuid' },
          { id: 'f4', key: 'mandate_type', technicalName: 'mandate_type', displayName: 'Mandate Type', dataType: 'string' },
        ],
      },
    },
  };

  it('should return root fields when no subtype is selected', () => {
    const fields = resolveSubtypeFields(sampleBO, null, false);
    expect(fields).toHaveLength(2);
    expect(fields.every((f) => f.scope === 'ASSIGNED')).toBe(true);
    expect(fields.map((f) => f.key)).toEqual(['account_number', 'account_name']);
  });

  it('should return strictly assigned subtype fields when showInheritedFields is false', () => {
    const fields = resolveSubtypeFields(sampleBO, 'institutional', false);
    expect(fields).toHaveLength(2);
    expect(fields.every((f) => f.scope === 'ASSIGNED')).toBe(true);
    expect(fields.map((f) => f.key)).toEqual(['sponsor_id', 'mandate_type']);
  });

  it('should return union of assigned and inherited root fields when showInheritedFields is true', () => {
    const fields = resolveSubtypeFields(sampleBO, 'institutional', true);
    expect(fields).toHaveLength(4);
    const assigned = fields.filter((f) => f.scope === 'ASSIGNED');
    const inherited = fields.filter((f) => f.scope === 'INHERITED');
    expect(assigned).toHaveLength(2);
    expect(inherited).toHaveLength(2);
    expect(assigned.map((f) => f.key)).toEqual(['sponsor_id', 'mandate_type']);
    expect(inherited.map((f) => f.key)).toEqual(['account_number', 'account_name']);
  });
});
