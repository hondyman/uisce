import type { CorePageDefinition } from '../../../../types/pageStudio';
import { masteringConsoleBlueprint } from './masteringConsole';
import { stagingBindingsBlueprint } from './stagingBindings';
import { dataPipelinesBlueprint, dataPipelineEditorBlueprint } from './dataPipelines';

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
];
