export async function simulatePortfolio(bundle, contexts, pool) {
  const results = []
  for (const ctx of contexts) {
    const row = []
    for (const rule of bundle.rules) {
      row.push(await pool.evaluate(rule, ctx))
    }
    results.push(row)
  }
  return results
}