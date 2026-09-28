import type { CorePageDefinition } from '../../../../types/pageStudio';
import { masteringConsoleBlueprint } from './masteringConsole';

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
];
