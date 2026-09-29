import type { CorePageDefinition } from '../../../../types/pageStudio';
import { masteringConsoleBlueprint } from './masteringConsole';
import { stagingBindingsBlueprint } from './stagingBindings';
import { dataPipelinesBlueprint, dataPipelineEditorBlueprint } from './dataPipelines';
import { matchRulesBlueprint, sourceHierarchyBlueprint, vendorRegistryBlueprint } from './mdmConfig';
import { schedulesBlueprint } from './schedules';
import { sourceScoringBlueprint } from './sourceScoring';
import { validationRulesBlueprint } from './validationRules';

/**
 * Pages the studio can start from: complete, working pages built entirely
 * from the page model - editable like any other page once created.
 */
export interface PageBlueprint {
  id: string;
  name: string;
  description: string;
  build: () => Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'>;
}

export const PAGE_BLUEPRINTS: PageBlueprint[] = [
  {
    id: 'mastering-console',
    name: 'Mastering console',
    description: 'Golden records, prices, runs, exceptions, duplicate review and steward overrides for every mastered entity.',
    build: masteringConsoleBlueprint,
  },
  {
    id: 'staging-bindings',
    name: 'Staging bindings',
    description: 'Map vendor staging tables to business object fields for mastering; changes need a second administrator.',
    build: stagingBindingsBlueprint,
  },
  {
    id: 'data-pipelines',
    name: 'Data pipelines',
    description: 'The list of data pipelines that load files and business objects into staging for mastering.',
    build: dataPipelinesBlueprint,
  },
  {
    id: 'data-pipeline-editor',
    name: 'Data pipeline editor',
    description: 'The visual pipeline editor (canvas, preview, assistant, runs), served at /data/pipelines/:id.',
    build: dataPipelineEditorBlueprint,
  },
  {
    id: 'mdm-source-hierarchy',
    name: 'Source hierarchy',
    description: 'Per mastered entity: which source wins for each field group or price type. Maker-checker.',
    build: sourceHierarchyBlueprint,
  },
  {
    id: 'mdm-match-rules',
    name: 'Match rules',
    description: 'Per record entity: exact and fuzzy match keys with auto-match and review thresholds. Maker-checker.',
    build: matchRulesBlueprint,
  },
  {
    id: 'mdm-vendors',
    name: 'Vendor registry',
    description: 'The one list of data sources mastering ranks and matches on. Maker-checker.',
    build: vendorRegistryBlueprint,
  },
  {
    id: 'schedules',
    name: 'Schedules',
    description: 'The platform scheduler: every schedule and its run history, with the schedule editor.',
    build: schedulesBlueprint,
  },
  {
    id: 'mdm-source-scoring',
    name: 'Source scoring & displacement',
    description: 'Vendor quality scoring, substitution rates, override endorsements, value-for-money efficient frontier, and displacement readiness simulation.',
    build: sourceScoringBlueprint,
  },
  {
    id: 'validation-rules',
    name: 'Validation Rules & Rule Studio',
    description: 'Centralized single-store validation catalog, portable AST bundles, and live evaluation engine.',
    build: validationRulesBlueprint,
  },
];


