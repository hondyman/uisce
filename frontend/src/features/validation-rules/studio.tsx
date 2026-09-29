import React from 'react';
import { registerOperations, type OperationDef } from '../../studio-core/operations/registry';
import { registerDomainComponents } from '../../studio-core/components/registry';
import { validationRulesApi } from './api';
import { RuleBundleManager } from './components/RuleBundleManager';
import { RuleTesterPanel } from './components/RuleTesterPanel';
import { ViolationsLiveViewer } from './components/ViolationsLiveViewer';

/**
 * Validation Rules Page Studio Operations:
 * Exposes queries and mutations to the Page Studio runtime.
 */

const operations: OperationDef[] = [
  {
    id: 'validationRules.list',
    domain: 'validation_rules',
    kind: 'query',
    label: 'List Validation Rules',
    description: 'Lists all active and draft validation rules for a given Business Object.',
    params: [
      { name: 'bo_name', type: 'string', required: false },
      { name: 'domain', type: 'string', required: false },
    ],
    fields: [
      { name: 'rules' },
      { name: 'total', type: 'number' },
    ],
    run: async (params) => {
      const boName = params.bo_name ? String(params.bo_name) : undefined;
      const domain = params.domain ? String(params.domain) : undefined;
      const rules = await validationRulesApi.listRules(boName, domain);
      return { rules, total: rules.length };
    },
  },
  {
    id: 'validationRules.export',
    domain: 'validation_rules',
    kind: 'query',
    label: 'Export Rule Bundle',
    description: 'Exports portable AST rule bundles in JSON or YAML format.',
    params: [
      { name: 'bo_name', type: 'string', required: false },
      { name: 'domain', type: 'string', required: false },
      { name: 'format', type: 'string', required: false },
    ],
    run: async (params) => {
      const boName = params.bo_name ? String(params.bo_name) : undefined;
      const domain = params.domain ? String(params.domain) : undefined;
      const format = params.format === 'yaml' ? 'yaml' : 'json';
      return validationRulesApi.exportBundle(boName, domain, undefined, format);
    },
  },
  {
    id: 'validationRules.preflight',
    domain: 'validation_rules',
    kind: 'mutation',
    label: 'Preflight Rule Bundle Check',
    description: 'Executes a dry-run check of an import bundle returning AST diffs and error reports.',
    params: [
      { name: 'bundle', type: 'string', required: true },
      { name: 'overwrite_policy', type: 'string', required: false },
    ],
    run: async (params) => {
      let bundle = params.bundle;
      if (typeof bundle === 'string' && bundle.startsWith('{')) {
        bundle = JSON.parse(bundle);
      }
      return validationRulesApi.preflightImport(
        bundle,
        params.overwrite_policy ? String(params.overwrite_policy) : 'fail'
      );
    },
  },
  {
    id: 'validationRules.import',
    domain: 'validation_rules',
    kind: 'mutation',
    label: 'Import Rule Bundle',
    description: 'Imports and applies a portable rule bundle to the tenant catalog.',
    params: [
      { name: 'bundle', type: 'string', required: true },
      { name: 'overwrite_policy', type: 'string', required: false },
    ],
    invalidates: [['validationRules.list']],
    run: async (params) => {
      let bundle = params.bundle;
      if (typeof bundle === 'string' && bundle.startsWith('{')) {
        bundle = JSON.parse(bundle);
      }
      return validationRulesApi.importBundle(
        bundle,
        params.overwrite_policy ? String(params.overwrite_policy) : 'fail'
      );
    },
  },
  {
    id: 'validationRules.evaluateRecord',
    domain: 'validation_rules',
    kind: 'mutation',
    label: 'Evaluate Record Payload',
    description: 'Evaluates a single data record against active rules for a Business Object.',
    params: [
      { name: 'bo_name', type: 'string', required: true },
      { name: 'record', type: 'string', required: true },
      { name: 'timing', type: 'string', required: false },
    ],
    run: async (params) => {
      const boName = String(params.bo_name);
      let record = params.record;
      if (typeof record === 'string') {
        record = JSON.parse(record);
      }
      return validationRulesApi.evaluateRecord(
        boName,
        record as Record<string, any>,
        undefined,
        params.timing ? String(params.timing) : undefined
      );
    },
  },
  {
    id: 'validationRules.violations',
    domain: 'validation_rules',
    kind: 'query',
    label: 'List Rule Violations',
    description: 'Fetches recent validation violations from public.validation_rule_violations.',
    params: [
      { name: 'bo_key', type: 'string', required: false },
      { name: 'limit', type: 'number', required: false },
    ],
    run: async (params) => {
      const boKey = params.bo_key ? String(params.bo_key) : undefined;
      const limit = params.limit ? Number(params.limit) : 100;
      return validationRulesApi.listViolations(boKey, limit);
    },
  },
];

registerOperations(operations);

registerDomainComponents([
  {
    id: 'validationRules.RuleBundleManager',
    domain: 'validation_rules',
    label: 'Rule Bundle Porter (GitOps)',
    description: 'Export and import portable AST rule bundles with preflight diff review.',
    inputs: [
      { name: 'bo_name', label: 'Business Object', type: 'string', required: false },
      { name: 'domain', label: 'Domain', type: 'string', required: false },
    ],
    events: ['onRefresh'],
    render: ({ inputs, onEvent }) => (
      <RuleBundleManager
        boName={inputs.bo_name ? String(inputs.bo_name) : undefined}
        domain={inputs.domain ? String(inputs.domain) : undefined}
        onRefresh={() => onEvent && onEvent('onRefresh')}
      />
    ),
  },
  {
    id: 'validationRules.RuleTesterPanel',
    domain: 'validation_rules',
    label: 'Rule Evaluator & Test Simulator',
    description: 'Interactive test panel for validating payload payloads against rules in-memory.',
    inputs: [
      { name: 'default_bo_name', label: 'Default BO', type: 'string', required: false },
    ],
    events: [],
    render: ({ inputs }) => (
      <RuleTesterPanel
        defaultBOName={inputs.default_bo_name ? String(inputs.default_bo_name) : 'order'}
      />
    ),
  },
  {
    id: 'validationRules.ViolationsLiveViewer',
    domain: 'validation_rules',
    label: 'Validation Violations Live Monitor',
    description: 'Authoritative telemetry viewer querying public.validation_rule_violations.',
    inputs: [
      { name: 'limit', label: 'Max Violations Limit', type: 'number', required: false },
    ],
    events: [],
    render: ({ inputs }) => (
      <ViolationsLiveViewer
        limit={inputs.limit ? Number(inputs.limit) : 100}
      />
    ),
  },
]);
