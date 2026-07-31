export async function simulatePortfolio(bundle: any, contexts: any[], pool: any) {
  const results: any[] = []
  for (const ctx of contexts) {
    const row: any[] = []
    for (const rule of bundle.rules) {
      row.push(await pool.evaluate(rule, ctx))
    }
    results.push(row)
  }
  return results
}