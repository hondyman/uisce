// Capture the blueprint JSON for embedding into a seed migration. Run from the repo root:
//   node scripts/capture_blueprint.ts lakehouseStatus
// Prints the layout/tabs/components/dataSources/presentationEvents/filterBar/app blocks
// as JSON strings, ready to paste into a migration's $tag$ ... $tag$ sections.
import { lakehouseStatusBlueprint } from '../frontend/src/pages/page-studio/app/blueprints/lakehouseStatus';

const bp = lakehouseStatusBlueprint();
const sections = ['layout', 'tabs', 'components', 'dataSources', 'presentationEvents', 'filterBar', 'app'];
for (const key of sections) {
  const v = (bp as Record<string, unknown>)[key];
  process.stdout.write(`--- ${key} ---\n${JSON.stringify(v, null, 2)}\n`);
}