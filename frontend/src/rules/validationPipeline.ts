export async function validateRule(rule, contexts, pool) {
  return {
    schema: validateSchema(rule),
    lint: lintRule(rule),
    migration: migrateRule(rule),
    simulation: await simulateBundle({ rules: [rule] }, contexts, pool),
  }
}