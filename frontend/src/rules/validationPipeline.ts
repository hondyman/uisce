export async function validateRule(rule: any, contexts: any[], pool: any) {
  const validateSchema = (r: any) => ({ valid: true });
  const lintRule = (r: any) => ({ errors: [] });
  const migrateRule = (r: any) => ({ migrated: r });
  const simulateBundle = async (bundle: any, ctx: any[], p: any) => ({ result: true });
  return {
    schema: validateSchema(rule),
    lint: lintRule(rule),
    migration: migrateRule(rule),
    simulation: await simulateBundle({ rules: [rule] }, contexts, pool),
  }
}