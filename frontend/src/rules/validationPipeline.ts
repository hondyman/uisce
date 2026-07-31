export async function validateRule(rule: any, contexts: any[], pool: any) {
  return {
    schema: validateSchema(rule),
    lint: lintRule(rule),
    migration: migrateRule(rule),
    simulation: await simulateBundle({ rules: [rule] }, contexts, pool),
  }
}