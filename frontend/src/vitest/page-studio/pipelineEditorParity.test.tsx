import React from 'react';
import { beforeAll, describe, expect, it, vi } from 'vitest';
import { render } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { loadRuleEngine } from './loadRuleEngine';
import { arrange, scenario } from './pipelineEditorScenario';

/**
 * The data pipeline editor, built in Page Studio (Canvas widget, step forms
 * shaped by the domain, problems / preview / runs tabs), against what the
 * hand-built PipelineEditorPage showed and saved for the same pipeline -
 * recorded below by running the same scenario on it before it was retired.
 * Every step's values, alerts and chips, the canvas, palette, problems,
 * the saved spec after an edit, the preview and a run are the same; the
 * deliberate differences are listed in `comparable`.
 */

// React Flow measures its viewport; jsdom has no layout.
class RO { observe() {} unobserve() {} disconnect() {} }
(globalThis as { ResizeObserver?: unknown }).ResizeObserver ??= RO;

const dp = vi.hoisted(() => ({ get: vi.fn(), nodeTypes: vi.fn(), validate: vi.fn(), stagingTables: vi.fn(), files: vi.fn(), preview: vi.fn(), runs: vi.fn(), run: vi.fn(), update: vi.fn(), suggestMapping: vi.fn(), create: vi.fn(), startRun: vi.fn(), profile: vi.fn(), upload: vi.fn() }));
const pf = vi.hoisted(() => ({ businessObjects: vi.fn(), boSchema: vi.fn(), rules: vi.fn() }));
vi.mock('../../features/data-pipelines/api', async (orig) => {
  const real = await orig<typeof import('../../features/data-pipelines/api')>();
  return { ...real, pipelinesApi: { ...real.pipelinesApi, ...dp }, platformApi: { ...real.platformApi, ...pf } };
});
const sched = vi.hoisted(() => ({ list: vi.fn() }));
vi.mock('../../features/schedules/api', async (orig) => {
  const real = await orig<typeof import('../../features/schedules/api')>();
  return { ...real, schedulesApi: { ...real.schedulesApi, ...sched } };
});
const mast = vi.hoisted(() => ({ profiles: vi.fn() }));
vi.mock('../../features/mastering/api', async (orig) => {
  const real = await orig<typeof import('../../features/mastering/api')>();
  return { ...real, masteringApi: { ...real.masteringApi, ...mast } };
});

import '../../studio-core/registerDomains';
import RuntimePage from '../../pages/page-studio/app/RuntimePage';
import { dataPipelineEditorBlueprint } from '../../pages/page-studio/app/blueprints/dataPipelines';

beforeAll(loadRuleEngine, 30000);

/** What the hand-built editor showed and saved (squashed text; see pipelineEditorScenario). */
const HAND_BUILT = {
  "nodes": {
    "file_1": "Vendorfileuploads/prices.csv·2columns",
    "map_2": "Mapfields2field(s)mapped",
    "staging_sink_3": "Loadstagingstaging.ff_price·FACTSET/PRICE",
    "master_4": "Mastermasterintopricegoldenrecords",
    "bo_source_5": "Fundsfund·3filter(s)",
    "validate_6": "Check1required",
    "rule_check_7": "Rules1rule(s)offund",
    "bo_sink_8": "Writefundsupdateorcreatefund",
    "file_sink_9": "Exportexports/funds.csv(csv)",
    "iceberg_sink_10": "Archivedefault.raw(iceberg)"
  },
  "palette": [
    "Read a file:on",
    "Map fields:on",
    "Archive to lakehouse:off"
  ],
  "problems": "PickatleastoneuniquefieldCheckTwosourcesfeednothingPipeline",
  "steps": {
    "file_1": {
      "labels": [
        "File",
        "Format",
        "Headerrow",
        "Separator",
        "Stepname"
      ],
      "inputs": [
        ",",
        "Vendorfile",
        "check:false",
        "check:true",
        "check:true",
        "csv",
        "decimal",
        "string",
        "uploads/prices.csv"
      ],
      "alerts": [],
      "chips": [],
      "buttons": [
        "Readfile"
      ]
    },
    "map_2": {
      "labels": [
        "AlsopassthroughfieldsIdidn'tmap",
        "Stepname"
      ],
      "inputs": [
        "",
        "A=Alpha",
        "Mapfields",
        "check:false",
        "instrument",
        "isin",
        "lookup",
        "price",
        "px"
      ],
      "alerts": [
        "Requiredandnotmappedyet:isin"
      ],
      "chips": [],
      "buttons": [
        "Addmapping",
        "Suggestmappings"
      ]
    },
    "staging_sink_3": {
      "labels": [
        "Domain",
        "Loadreference(optional)",
        "Sourcesystem",
        "Stagingtable",
        "Stepname"
      ],
      "inputs": [
        "",
        "",
        "FACTSET",
        "Loadstaging",
        "PRICE",
        "instrument",
        "staging.ff_price"
      ],
      "alerts": [],
      "chips": [],
      "buttons": []
    },
    "master_4": {
      "labels": [
        "Entity",
        "Stepname"
      ],
      "inputs": [
        "Master",
        "price"
      ],
      "alerts": [],
      "chips": [],
      "buttons": []
    },
    "bo_source_5": {
      "labels": [
        "Businessobject",
        "Condition",
        "Condition",
        "Condition",
        "Field",
        "Field",
        "Field",
        "Readatmost(optional)",
        "Stepname",
        "from",
        "to",
        "value",
        "values"
      ],
      "inputs": [
        "",
        "",
        "1000",
        "2020",
        "2024",
        "Fund",
        "Funds",
        "aum",
        "between",
        "greater_than",
        "in",
        "launch",
        "region"
      ],
      "alerts": [],
      "chips": [
        "EU",
        "US"
      ],
      "buttons": [
        "Addcondition"
      ]
    },
    "validate_6": {
      "labels": [
        "Mustbeuniquetogether",
        "Musthaveavalue",
        "Stepname"
      ],
      "inputs": [
        "",
        "",
        "Check"
      ],
      "alerts": [
        "Pickatleastoneuniquefield"
      ],
      "chips": [
        "aum"
      ],
      "buttons": []
    },
    "rule_check_7": {
      "labels": [
        "AUMpositiveBlockcoreAUMmustbeabovezero",
        "RegionknownWarn",
        "Rulesofbusinessobject",
        "Stepname"
      ],
      "inputs": [
        "Fund",
        "Rules",
        "check:false",
        "check:true"
      ],
      "alerts": [],
      "chips": [
        "Block",
        "Warn",
        "core"
      ],
      "buttons": []
    },
    "bo_sink_8": {
      "labels": [
        "Businessobject",
        "Matchrecordson",
        "Rehearsal:checkeveryrecordagainsttherules,butsavenothing",
        "Stepname",
        "Whentherecordalreadyexists"
      ],
      "inputs": [
        "",
        "Fund",
        "Writefunds",
        "check:false",
        "upsert"
      ],
      "alerts": [],
      "chips": [
        "code"
      ],
      "buttons": []
    },
    "file_sink_9": {
      "labels": [
        "Filename",
        "Format",
        "Separator",
        "Stepname"
      ],
      "inputs": [
        ",",
        "Export",
        "csv",
        "exports/funds.csv"
      ],
      "alerts": [],
      "chips": [],
      "buttons": []
    },
    "iceberg_sink_10": {
      "labels": [
        "Namespace/CatalogDB",
        "Stepname",
        "Tablename"
      ],
      "inputs": [
        "Archive",
        "default",
        "raw"
      ],
      "alerts": [],
      "chips": [
        "Format:Parquet(IcebergREST)"
      ],
      "buttons": []
    }
  },
  "saved": {
    "name": "Funds load",
    "map": {
      "id": "map_2",
      "type": "map",
      "label": "Map it",
      "position": {
        "x": 280,
        "y": 0
      },
      "config": {
        "fields": [
          {
            "from": "isin",
            "to": "instrument"
          },
          {
            "from": "px",
            "to": "price",
            "transform": "lookup",
            "lookup": {
              "A": "Alpha"
            }
          },
          {
            "from": "",
            "to": ""
          }
        ]
      }
    }
  },
  "preview": {
    "labels": [],
    "inputs": [],
    "alerts": [],
    "chips": [
      "1rejected",
      "1rejected",
      "2wouldbewritten",
      "3read",
      "Allsteps",
      "Mapit:2",
      "Vendorfile:3",
      "in3",
      "in3",
      "out2",
      "out3"
    ],
    "buttons": [],
    "rejects": "3Mapitpxisnotanumber",
    "sample": "2US2—"
  },
  "run": {
    "labels": [],
    "inputs": [],
    "alerts": [],
    "chips": [
      "Master:completed",
      "completedwitherrors"
    ],
    "buttons": []
  }
} as const;

type Inv = { labels: string[]; inputs: string[]; alerts: string[]; chips: string[]; buttons: string[] };
const drop = (list: readonly string[], gone: string[]) => list.filter((x) => !gone.includes(x));

/**
 * The studio page's inventory made comparable. Deliberate differences:
 * step actions are labelled buttons (Close, Remove this step, Upload a
 * file, Create a rule) rather than icons or links; a file's columns and a
 * map's rows are labelled inputs rather than table cells (so the column
 * names are read-only inputs); a business-object picker is a select, whose
 * input holds the key (fund) rather than the name shown; preview steps are
 * toggle buttons, not chips; a run's mastering link is a button; the
 * Iceberg format is a line of text, not a chip.
 */
function comparable(studio: Record<string, unknown>): Record<string, unknown> {
  const steps = studio.steps as Record<string, Inv>;
  const out: Record<string, Inv> = {};
  for (const [id, inv] of Object.entries(steps)) {
    out[id] = {
      labels: drop(inv.labels, id === 'file_1' ? ['Column', 'Type', 'Required'] : id === 'map_2' ? ['From', 'To', 'Transform', 'Lookup'] : []),
      inputs: drop(inv.inputs, id === 'file_1' ? ['isin', 'px'] : []),
      alerts: inv.alerts, chips: inv.chips,
      buttons: drop(inv.buttons, ['Close', 'Removethisstep', 'Uploadafile', 'Createarule']),
    };
  }
  const pv = studio.preview as Inv & { rejects: string; sample: string };
  const run = studio.run as Inv;
  return {
    ...studio, steps: out,
    preview: { ...pv, chips: [...pv.chips, ...pv.buttons].sort(), buttons: [] },
    run: { ...run, buttons: drop(run.buttons, ['Openpricemastering']) },
  };
}
function expected() {
  const hb = structuredClone(HAND_BUILT) as unknown as Record<string, unknown>;
  const steps = hb.steps as Record<string, Inv>;
  for (const id of ['bo_source_5', 'rule_check_7', 'bo_sink_8']) steps[id].inputs = steps[id].inputs.map((x) => (x === 'Fund' ? 'fund' : x)).sort();
  steps.iceberg_sink_10.chips = [];
  return hb;
}

describe('data pipeline editor built in Page Studio', () => {
  it('shows, edits, saves, previews and follows runs as the hand-built editor did', async () => {
    arrange({ dp, pf, sched, mast });
    const bp = dataPipelineEditorBlueprint();
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <MemoryRouter initialEntries={['/data/pipelines/p1']}>
          <Routes>
            <Route path="/data/pipelines/:id" element={
              <RuntimePage name={bp.name} slug={bp.slug} tabs={bp.tabs!} components={bp.components} dataSources={[]} tenantId="t1" app={bp.app} />
            } />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>,
    );
    const got = comparable(await scenario({ dp, pf, sched, mast }));
    const want = expected();
    expect(got.nodes).toEqual(want.nodes);
    expect(got.palette).toEqual(want.palette);
    expect(got.problems).toEqual(want.problems);
    for (const id of Object.keys(want.steps as object)) expect({ id, ...(got.steps as Record<string, Inv>)[id] }).toEqual({ id, ...(want.steps as Record<string, Inv>)[id] });
    expect(got.saved).toEqual(want.saved);
    expect(got.preview).toEqual(want.preview);
    expect(got.run).toEqual(want.run);
  }, 60000);
});
