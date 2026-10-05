import React from 'react';
import { Alert, Box } from '@mui/material';
import { registerOperations, type OperationDef } from '../../studio-core/operations/registry';
import { registerDomainComponents } from '../../studio-core/components/registry';
import { listBusinessObjects } from '../../studio-core/binding/businessObjectApi';
import type { BusinessObjectOption } from '../../studio-core/binding/businessObjectApi';
import {
  createCube,
  deployCube,
  getCube,
  getCubeImpact,
  listCubeMetrics,
  listCubes,
  patchCube,
  previewCubeImpact,
  refreshCube,
  validateCube,
} from './cubeDefinitionApi';
import { cubeDraftFromDefinition, cubeDraftPayload, emptyCubeDraft } from './draft';
import FederationEditor from './FederationEditor';
import ImpactPanel from './ImpactPanel';
import type {
  CubeDraft,
  CubeFederation,
  CubeImpactPreviewAction,
  FederationKeySample,
} from './types';

/**
 * Cubes Page Studio surface: catalog/designer ops + small DomainComponents
 * (`cubes.FederationEditor`, `cubes.ImpactPanel`). Do **not** register a
 * full-page `cubes.Designer` DomainComponent that rehosts a coded designer shell.
 */

const DOMAIN = 'cubes';

function need(p: Record<string, unknown>, name: string): string {
  const v = p[name];
  if (typeof v !== 'string' || !v.trim()) throw new Error(`${name} is required`);
  return v.trim();
}

function asDraft(p: Record<string, unknown>): CubeDraft {
  const d = p.draft;
  if (!d || typeof d !== 'object') throw new Error('draft is required');
  return d as CubeDraft;
}

function asFederation(value: unknown): CubeFederation {
  if (!value || typeof value !== 'object') return {};
  return value as CubeFederation;
}

function asKeySamples(value: unknown): FederationKeySample[] {
  return Array.isArray(value) ? (value as FederationKeySample[]) : [];
}

function optionalGrain(p: Record<string, unknown>): string[] | undefined {
  const g = p.grain;
  if (Array.isArray(g)) return g.map(String);
  if (typeof g === 'string' && g.trim()) {
    try {
      const parsed = JSON.parse(g);
      if (Array.isArray(parsed)) return parsed.map(String);
    } catch {
      return g.split(',').map((x) => x.trim()).filter(Boolean);
    }
  }
  return undefined;
}

const operations: OperationDef[] = [
  {
    id: 'cubes.list',
    domain: DOMAIN,
    kind: 'query',
    label: 'List cubes',
    description: 'Cube definitions visible to the tenant (core / custom / adopted / all).',
    params: [
      { name: 'scope', type: 'string', description: 'all | core | custom | adopted' },
      { name: 'boId', type: 'string' },
      { name: 'cursor', type: 'string' },
      { name: 'limit', type: 'number' },
    ],
    rowsPath: 'rows',
    rowFields: [
      { name: 'id' },
      { name: 'name' },
      { name: 'boId' },
      { name: 'contractVersion', type: 'number' },
      { name: 'isCore', type: 'boolean' },
      { name: 'status' },
      { name: 'updatedAt', type: 'datetime' },
    ],
    run: async (p) => {
      const res = await listCubes({
        scope: (typeof p.scope === 'string' ? p.scope : 'all') as 'all' | 'core' | 'custom' | 'adopted',
        boId: typeof p.boId === 'string' ? p.boId : undefined,
        cursor: typeof p.cursor === 'string' ? p.cursor : undefined,
        limit: typeof p.limit === 'number' ? p.limit : undefined,
      });
      return {
        rows: res.cubes,
        scope: res.scope,
        nextCursor: res.nextCursor,
        empty_text: 'No cubes yet. Create one from Build → Models → Cubes.',
      };
    },
  },
  {
    id: 'cubes.get',
    domain: DOMAIN,
    kind: 'query',
    label: 'Get a cube',
    params: [{ name: 'id', type: 'string', required: true }],
    fields: [
      { name: 'cube', type: 'object' },
      { name: 'draft', type: 'object' },
    ],
    run: async (p) => {
      const cube = await getCube(need(p, 'id'));
      return { cube, draft: cubeDraftFromDefinition(cube) };
    },
  },
  {
    id: 'cubes.editorStart',
    domain: DOMAIN,
    kind: 'query',
    label: 'Start the cube editor',
    description:
      'Load an existing cube by id, or return { is_new: true, draft } when id is absent/empty/new.',
    params: [{ name: 'id', type: 'string', description: 'Cube id, or empty / "new" for create' }],
    fields: [
      { name: 'is_new', type: 'boolean' },
      { name: 'draft', type: 'object' },
      { name: 'cube', type: 'object' },
      { name: 'key', type: 'string' },
      { name: 'title', type: 'string' },
      { name: 'contractVersion', type: 'number' },
    ],
    run: async (p) => {
      const raw = typeof p.id === 'string' ? p.id.trim() : '';
      const isNew = !raw || raw === 'new';
      if (isNew) {
        const draft = emptyCubeDraft();
        return {
          is_new: true,
          draft,
          cube: null,
          key: 'new',
          title: 'New cube',
          contractVersion: 1,
          federationKeySamples: draft.federationKeySamples,
        };
      }
      const cube = await getCube(raw);
      const draft = cubeDraftFromDefinition(cube);
      return {
        is_new: false,
        draft,
        cube,
        key: `${cube.id}:${cube.contractVersion}`,
        title: cube.name,
        contractVersion: cube.contractVersion,
        federationKeySamples: draft.federationKeySamples,
      };
    },
  },
  {
    id: 'cubes.patchDraft',
    domain: DOMAIN,
    kind: 'query',
    label: 'Merge into cube draft (client)',
    description:
      'Returns draft with optional federation / keySamples / patch merged. Used by DomainComponent events and Form onChange.',
    params: [
      { name: 'draft', type: 'object', required: true },
      { name: 'federation', type: 'object' },
      { name: 'federationKeySamples', type: 'object' },
      { name: 'patch', type: 'object' },
    ],
    fields: [{ name: 'draft', type: 'object' }],
    run: async (p) => {
      const draft: CubeDraft = { ...asDraft(p) };
      if (p.federation !== undefined) draft.federation = asFederation(p.federation);
      if (p.federationKeySamples !== undefined) {
        draft.federationKeySamples = asKeySamples(p.federationKeySamples);
      }
      if (p.patch && typeof p.patch === 'object' && !Array.isArray(p.patch)) {
        Object.assign(draft, p.patch as Partial<CubeDraft>);
      }
      return { draft };
    },
  },
  {
    id: 'cubes.create',
    domain: DOMAIN,
    kind: 'mutation',
    label: 'Create a cube',
    params: [{ name: 'draft', type: 'object', required: true }],
    invalidates: [['cubes.list']],
    run: async (p) => {
      const cube = await createCube(cubeDraftPayload(asDraft(p)));
      return { cube, id: cube.id, draft: cubeDraftFromDefinition(cube) };
    },
  },
  {
    id: 'cubes.patch',
    domain: DOMAIN,
    kind: 'mutation',
    label: 'Patch a cube',
    params: [
      { name: 'id', type: 'string', required: true },
      { name: 'draft', type: 'object', required: true },
    ],
    invalidates: [['cubes.list'], ['cubes.get']],
    run: async (p) => {
      const res = await patchCube(need(p, 'id'), cubeDraftPayload(asDraft(p)));
      return {
        cube: res.cube,
        draft: cubeDraftFromDefinition(res.cube),
        noop: res.noop,
        reason: res.reason,
      };
    },
  },
  {
    id: 'cubes.validate',
    domain: DOMAIN,
    kind: 'mutation',
    label: 'Validate a cube',
    description: 'Structural + metrics + federation orphan gate. Pass draft and optional key samples.',
    params: [
      { name: 'id', type: 'string', required: true },
      { name: 'draft', type: 'object' },
      { name: 'federationKeySamples', type: 'object' },
    ],
    run: async (p) => {
      const id = need(p, 'id');
      let body: Record<string, unknown> | undefined;
      if (p.draft && typeof p.draft === 'object') {
        const draft = { ...(p.draft as CubeDraft) };
        if (Array.isArray(p.federationKeySamples)) {
          draft.federationKeySamples = p.federationKeySamples as FederationKeySample[];
        }
        body = cubeDraftPayload(draft, { includeKeySamples: true });
      } else if (Array.isArray(p.federationKeySamples)) {
        body = { federationKeySamples: p.federationKeySamples };
      }
      return validateCube(id, body);
    },
  },
  {
    id: 'cubes.deploy',
    domain: DOMAIN,
    kind: 'mutation',
    label: 'Deploy a cube',
    params: [
      { name: 'id', type: 'string', required: true },
      { name: 'grain', type: 'object' },
      { name: 'force', type: 'boolean' },
    ],
    invalidates: [['cubes.get']],
    run: async (p) =>
      deployCube(need(p, 'id'), {
        grain: optionalGrain(p),
        force: p.force === true,
      }),
  },
  {
    id: 'cubes.refresh',
    domain: DOMAIN,
    kind: 'mutation',
    label: 'Refresh a cube materialization',
    params: [
      { name: 'id', type: 'string', required: true },
      { name: 'grain', type: 'object' },
      { name: 'force', type: 'boolean' },
    ],
    invalidates: [['cubes.get']],
    run: async (p) =>
      refreshCube(need(p, 'id'), {
        grain: optionalGrain(p),
        force: p.force === true,
      }),
  },
  {
    id: 'cubes.metrics',
    domain: DOMAIN,
    kind: 'query',
    label: 'Cube metric options',
    description: 'Metric definition ids available for a Business Object.',
    params: [{ name: 'boId', type: 'string' }],
    rowsPath: 'rows',
    rowFields: [{ name: 'id' }, { name: 'name' }, { name: 'boId' }],
    run: async (p) => {
      const metrics = await listCubeMetrics(typeof p.boId === 'string' ? p.boId : undefined);
      return { rows: metrics, metrics };
    },
  },
  {
    id: 'cubes.businessObjects',
    domain: DOMAIN,
    kind: 'query',
    label: 'Business objects for cube federation',
    description: 'BO options for FederationEditor bos input (wraps listBusinessObjects).',
    params: [],
    rowsPath: 'rows',
    rowFields: [
      { name: 'id' },
      { name: 'key' },
      { name: 'name' },
      { name: 'displayName' },
    ],
    run: async () => {
      const rows = await listBusinessObjects();
      return { rows, bos: rows };
    },
  },
  {
    id: 'cubes.impact',
    domain: DOMAIN,
    kind: 'query',
    label: 'Cube impact inventory',
    description: 'GET /api/cubes/{id}/impact — composition + consumers + physical grains.',
    params: [
      { name: 'id', type: 'string', required: true },
      { name: 'includePhysical', type: 'boolean' },
    ],
    fields: [
      { name: 'cubeId', type: 'string' },
      { name: 'composition', type: 'object' },
      { name: 'consumers', type: 'object' },
      { name: 'summary', type: 'object' },
    ],
    run: async (p) =>
      getCubeImpact(need(p, 'id'), {
        includePhysical: p.includePhysical === false ? false : true,
      }),
  },
  {
    id: 'cubes.impactPreview',
    domain: DOMAIN,
    kind: 'mutation',
    label: 'Cube impact preview',
    description:
      'POST /api/cubes/{id}/impact/preview — changeClass + confirmToken. Cascade apply is A4.',
    params: [
      { name: 'id', type: 'string', required: true },
      { name: 'action', type: 'string', required: true },
      { name: 'draft', type: 'object' },
      { name: 'patch', type: 'object' },
    ],
    fields: [
      { name: 'changeClass', type: 'string' },
      { name: 'breakReasons', type: 'object' },
      { name: 'blockingCount', type: 'number' },
      { name: 'confirmToken', type: 'string' },
      { name: 'allowedModes', type: 'object' },
      { name: 'recommendedMode', type: 'string' },
    ],
    run: async (p) => {
      const action = need(p, 'action') as CubeImpactPreviewAction;
      let patch: Record<string, unknown> | undefined;
      if (p.patch && typeof p.patch === 'object' && !Array.isArray(p.patch)) {
        patch = p.patch as Record<string, unknown>;
      } else if (p.draft && typeof p.draft === 'object') {
        patch = cubeDraftPayload(p.draft as CubeDraft);
      }
      return previewCubeImpact(need(p, 'id'), { action, patch });
    },
  },
];

registerOperations(operations);

function asBos(value: unknown): BusinessObjectOption[] {
  if (!Array.isArray(value)) return [];
  return value.filter(
    (x): x is BusinessObjectOption =>
      !!x && typeof x === 'object' && typeof (x as BusinessObjectOption).id === 'string',
  );
}

const FederationEditorAdapter: React.FC<{
  inputs: Record<string, unknown>;
  emit: (event: string, payload?: Record<string, unknown>) => void;
}> = ({ inputs, emit }) => {
  const primaryBoId = typeof inputs.primaryBoId === 'string' ? inputs.primaryBoId : '';
  const bos = asBos(inputs.bos);
  const federation = asFederation(inputs.federation);
  const keySamples = asKeySamples(inputs.keySamples);
  const readOnly = inputs.readOnly === true;

  if (!primaryBoId.trim()) {
    return <Alert severity="info">Pick a primary Business Object before editing federation.</Alert>;
  }
  if (bos.length === 0) {
    return (
      <Alert severity="warning">
        No Business Object options bound. Wire a <code>cubes.businessObjects</code> query to{' '}
        <code>inputs.bos</code>.
      </Alert>
    );
  }

  return (
    <Box sx={{ pointerEvents: readOnly ? 'none' : 'auto', opacity: readOnly ? 0.7 : 1 }}>
      <FederationEditor
        primaryBoId={primaryBoId}
        bos={bos}
        federation={federation}
        keySamples={keySamples}
        onChange={(next) => emit('onChange', { federation: next })}
        onKeySamplesChange={(next) => emit('onKeySamplesChange', { keySamples: next })}
      />
    </Box>
  );
};

const ImpactPanelAdapter: React.FC<{
  inputs: Record<string, unknown>;
  emit: (event: string, payload?: Record<string, unknown>) => void;
}> = ({ inputs }) => {
  const cubeId = typeof inputs.cubeId === 'string' ? inputs.cubeId : '';
  const draft =
    inputs.draft && typeof inputs.draft === 'object' && !Array.isArray(inputs.draft)
      ? (inputs.draft as CubeDraft)
      : null;
  const readOnly = inputs.readOnly === true;
  const title = typeof inputs.title === 'string' && inputs.title.trim() ? inputs.title : 'Cube impact';
  if (!cubeId.trim()) {
    return <Alert severity="info">Bind a cube id to assess impact.</Alert>;
  }
  return <ImpactPanel cubeId={cubeId} draft={draft} readOnly={readOnly} title={title} />;
};

registerDomainComponents([
  {
    id: 'cubes.FederationEditor',
    domain: DOMAIN,
    label: 'Cube federation editor',
    description:
      'Sources, term joins, orphan max %, and key samples for a cube draft. Maps 1:1 to live FederationEditor props.',
    inputs: [
      { name: 'primaryBoId', label: 'Primary BO id', type: 'string', required: true },
      { name: 'bos', label: 'Business object options', type: 'object', required: true },
      { name: 'federation', label: 'Federation draft', type: 'object', required: true },
      { name: 'keySamples', label: 'Federation key samples (session)', type: 'object', required: true },
      { name: 'readOnly', label: 'Read only', type: 'boolean' },
    ],
    events: [
      { name: 'onChange', payload: ['federation'], description: 'Federation JSON changed' },
      {
        name: 'onKeySamplesChange',
        payload: ['keySamples'],
        description: 'Session orphan-gate key samples changed',
      },
    ],
    render: FederationEditorAdapter,
  },
  {
    id: 'cubes.ImpactPanel',
    domain: DOMAIN,
    label: 'Cube impact panel',
    description:
      'Composition + consumers inventory and dry-run impact preview (archive/patch/publish). Cascade Confirm is A4.',
    inputs: [
      { name: 'cubeId', label: 'Cube id', type: 'string', required: true },
      { name: 'draft', label: 'Working draft (optional for patch/publish preview)', type: 'object' },
      { name: 'readOnly', label: 'Hide preview actions', type: 'boolean' },
      { name: 'title', label: 'Panel title', type: 'string' },
    ],
    events: [],
    render: ImpactPanelAdapter,
  },
]);

export { emptyCubeDraft, cubeDraftFromDefinition, cubeDraftPayload };
export { operations as cubesStudioOperations };
