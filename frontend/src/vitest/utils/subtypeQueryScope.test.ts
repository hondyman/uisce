import { describe, it, expect } from 'vitest';
import { resolveEffectiveQueryScope, type BusinessObject, type BORelationship } from '../../utils/subtypeQueryScope';

const makeField = (key: string, name: string): import('../subtypeFieldResolver').FieldDefinition => ({
  id: `f-${key}`,
  key,
  technicalName: key,
  displayName: name,
  dataType: 'string',
});

const makeRelationship = (
  key: string,
  opts: Partial<BORelationship> = {}
): BORelationship => ({
  id: `r-${key}`,
  key,
  name: key,
  sourceTable: 'oms.account',
  targetBoKey: 'oms.benchmark',
  targetTable: 'oms.benchmark',
  joinType: 'LEFT',
  joinCondition: '${SOURCE}.benchmark_id = ${TARGET}.id',
  ...opts,
});

const sampleBO: BusinessObject = {
  id: 'bo-account',
  key: 'oms.account',
  name: 'account',
  displayName: 'Investment Account',
  coreFields: [
    makeField('account_id', 'Account ID'),
    makeField('account_number', 'Account Number'),
    makeField('base_currency', 'Base Currency'),
  ],
  subtypes: {
    institutional: {
      id: 'st-inst',
      key: 'institutional',
      name: 'institutional',
      displayName: 'Institutional',
      technicalName: 'institutional',
      subtypeFields: [
        makeField('mandate_type', 'Mandate Type'),
        makeField('erisa_flag', 'ERISA Flag'),
        makeField('benchmark_id', 'Benchmark ID'),
      ],
      relationshipAllowlist: ['account_to_benchmark', 'account_to_mandate_schedule'],
    },
    retail_wealth: {
      id: 'st-retail',
      key: 'retail_wealth',
      name: 'retail_wealth',
      displayName: 'Retail Wealth',
      technicalName: 'retail_wealth',
      subtypeFields: [
        makeField('risk_profile', 'Risk Profile'),
        makeField('advisor_id', 'Advisor ID'),
      ],
      relationshipAllowlist: ['account_to_advisor'],
    },
  },
  relationships: [
    makeRelationship('account_to_custodian', {
      scopedSubtypeKey: null,
    }),
    makeRelationship('account_to_benchmark', {
      scopedSubtypeKey: 'institutional',
    }),
    makeRelationship('account_to_mandate_schedule', {
      scopedSubtypeKey: 'institutional',
    }),
    makeRelationship('account_to_advisor', {
      scopedSubtypeKey: 'retail_wealth',
    }),
  ],
};

describe('resolveEffectiveQueryScope', () => {
  it('root scope: null discriminator, only root-level joins', () => {
    const scope = resolveEffectiveQueryScope(sampleBO, null);

    expect(scope.boKey).toBe('oms.account');
    expect(scope.selectedSubtypeKey).toBeNull();
    expect(scope.discriminatorClause).toBeNull();
    expect(scope.fields).toHaveLength(3);
    expect(scope.fields.every((f) => f.scope === 'ASSIGNED')).toBe(true);
    expect(scope.relationships).toHaveLength(1);
    expect(scope.relationships[0].key).toBe('account_to_custodian');
  });

  it('subtype scope: sets discriminator clause and subtype fields', () => {
    const scope = resolveEffectiveQueryScope(sampleBO, 'institutional');

    expect(scope.selectedSubtypeKey).toBe('institutional');
    expect(scope.discriminatorClause).toBe("t0.subtype_code = 'institutional'");
    expect(scope.fields).toHaveLength(6);
    const assigned = scope.fields.filter((f) => f.scope === 'ASSIGNED');
    const inherited = scope.fields.filter((f) => f.scope === 'INHERITED');
    expect(assigned).toHaveLength(3);
    expect(inherited).toHaveLength(3);
    expect(assigned.map((f) => f.key)).toEqual([
      'mandate_type',
      'erisa_flag',
      'benchmark_id',
    ]);
  });

  it('subtype scope: restricts relationships to allowlisted + root', () => {
    const scope = resolveEffectiveQueryScope(sampleBO, 'institutional');

    const relKeys = scope.relationships.map((r) => r.key);
    expect(relKeys).toContain('account_to_custodian');
    expect(relKeys).toContain('account_to_benchmark');
    expect(relKeys).toContain('account_to_mandate_schedule');
    expect(relKeys).not.toContain('account_to_advisor');
  });

  it('subtype scope: retail_wealth has its own edges', () => {
    const scope = resolveEffectiveQueryScope(sampleBO, 'retail_wealth');

    const relKeys = scope.relationships.map((r) => r.key);
    expect(relKeys).toContain('account_to_custodian');
    expect(relKeys).toContain('account_to_advisor');
    expect(relKeys).not.toContain('account_to_benchmark');
  });

  it('unknown subtype key falls back to root scope', () => {
    const scope = resolveEffectiveQueryScope(sampleBO, 'unknown_subtype');

    expect(scope.selectedSubtypeKey).toBeNull();
    expect(scope.discriminatorClause).toBeNull();
    expect(scope.relationships).toHaveLength(1);
  });

  it('BO with no subtypes falls back to root scope', () => {
    const boNoSubtypes: BusinessObject = {
      ...sampleBO,
      subtypes: undefined,
    };
    const scope = resolveEffectiveQueryScope(boNoSubtypes, 'institutional');

    expect(scope.selectedSubtypeKey).toBeNull();
    expect(scope.discriminatorClause).toBeNull();
  });

  it('deduplicates fields: subtype field shadows root field with same key', () => {
    const boWithShadow: BusinessObject = {
      ...sampleBO,
      coreFields: [
        makeField('account_number', 'Account Number'),
        makeField('base_currency', 'Base Currency'),
      ],
      subtypes: {
        institutional: {
          ...sampleBO.subtypes!.institutional,
          subtypeFields: [
            makeField('account_number', 'Account Number (Override)'),
          ],
        },
      },
    };
    const scope = resolveEffectiveQueryScope(boWithShadow, 'institutional');
    const assigned = scope.fields.filter((f) => f.scope === 'ASSIGNED');
    const inherited = scope.fields.filter((f) => f.scope === 'INHERITED');
    expect(assigned).toHaveLength(1);
    expect(inherited).toHaveLength(1);
    expect(scope.fields).toHaveLength(2);
    expect(assigned.find((f) => f.key === 'account_number')?.displayName).toBe(
      'Account Number (Override)'
    );
  });

  it('empty relationshipAllowlist allows only root joins', () => {
    const boEmptyAllowlist: BusinessObject = {
      ...sampleBO,
      subtypes: {
        ...sampleBO.subtypes,
        retail_wealth: {
          ...sampleBO.subtypes!['retail_wealth'],
          relationshipAllowlist: [],
        },
      },
    };
    const scope = resolveEffectiveQueryScope(boEmptyAllowlist, 'retail_wealth');
    expect(scope.relationships).toHaveLength(1);
    expect(scope.relationships[0].key).toBe('account_to_custodian');
  });
});
