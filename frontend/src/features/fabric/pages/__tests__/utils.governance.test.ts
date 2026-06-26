import { describe, it, expect } from 'vitest';
import { generateCoreYAML, generateFinalYAML } from '../utils';
import { generateCompanionGovernanceJSON, validateCubeConfig } from '../governance';
import type { SemanticModelConfig } from '../types';

const baseConfig: SemanticModelConfig = {
  core: {
    dimensions: [
      { id: '1', name: 'id', title: 'Id', sourceTable: 't', sourceColumn: 'id', type: 'number', sql: 'id', public: true, description: '', primary_key: true },
      { id: '2', name: 'tenant_id', title: 'Tenant Id', sourceTable: 't', sourceColumn: 'tenant_id', type: 'number', sql: 'tenant_id', public: true, description: '' },
    ],
    measures: [],
    joins: [
      { id: 'j1', name: 'users', title: 'users', sourceTable: 't', sourceColumn: '', type: '', sql: '{CUBE.tenant_id} = {users.tenant_id} AND {CUBE.user_id} = {users.id}', public: true, description: '', relationship: 'many_to_one' }
    ],
    options: { governance: { tenantField: 'tenant_id' } }
  },
  custom: {
    dimensions: [
      { id: '3', name: 'user_id', title: 'User Id', sourceTable: 't', sourceColumn: 'user_id', type: 'number', sql: 'user_id', public: true, description: '' },
    ],
    measures: [],
    joins: [],
    overrides: {},
    options: { governance: { pkFields: ['id'] } }
  }
};

describe('PK synthesis and tenant join injection', () => {
  it('synthesizes composite PK when multiple pkFields listed', () => {
    const cfg: SemanticModelConfig = JSON.parse(JSON.stringify(baseConfig));
    cfg.core.options = { governance: { pkFields: ['id', 'tenant_id'] } } as any;
    const yaml = generateCoreYAML(cfg, 'Orders');
    expect(yaml).toContain('name: Orders_core');
    expect(yaml).toMatch(/Primary Key/);
    expect(yaml).toContain("sql: CONCAT_WS('-', {CUBE}.id, {CUBE}.tenant_id)");
  });

  it('injects tenant filter into joins when missing', () => {
    const cfg: SemanticModelConfig = JSON.parse(JSON.stringify(baseConfig));
    cfg.core.joins = [
      { id: 'j2', name: 'users', title: 'users', sourceTable: 't', sourceColumn: '', type: '', sql: '{CUBE.user_id} = {users.id}', public: true, description: '', relationship: 'many_to_one' }
    ];
    const yaml = generateFinalYAML(cfg, 'Orders');
    expect(yaml).toContain('{CUBE.tenant_id} = {users.tenant_id}');
  });
});

describe('Governance JSON', () => {
  it('emits companion governance json with pkFields and steward', () => {
    const cfg: SemanticModelConfig = JSON.parse(JSON.stringify(baseConfig));
    cfg.core.options = { governance: { pkFields: ['id'], steward: 'data-team' } } as any;
    const json = generateCompanionGovernanceJSON(cfg, 'Orders');
    const obj = JSON.parse(json);
    expect(obj.cube).toBe('Orders');
    expect(obj.core.pkFields).toEqual(['id']);
    expect(obj.final.pkFields).toEqual(['id']);
    expect(obj.core.steward).toBe('data-team');
  });
});

describe('Validator', () => {
  it('flags missing FK field and naming inconsistency', () => {
    const cfg: SemanticModelConfig = JSON.parse(JSON.stringify(baseConfig));
    // remove user_id dimension, but keep it in join
    cfg.custom.dimensions = cfg.custom.dimensions.filter(d => d.name !== 'user_id');
    cfg.core.joins = [
      { id: 'j3', name: 'accounts', title: 'accounts', sourceTable: 't', sourceColumn: '', type: '', sql: '{CUBE.userRef} = {accounts.id}', public: true, description: '', relationship: 'many_to_one' }
    ];
    const issues = validateCubeConfig(cfg, 'Orders');
    const codes = issues.map(i => i.code);
    expect(codes).toContain('FK_FIELD_MISSING');
    expect(codes).toContain('NAMING_INCONSISTENT');
  });
});
