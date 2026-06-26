export async function profileRule(rule, contexts, pool) {
  const times = []
  for (const ctx of contexts) {
    const start = performance.now()
    await pool.evaluate(rule, ctx)
    times.push(performance.now() - start)
  }
  return {
    avg: times.reduce((a, b) => a + b, 0) / times.length,
    max: Math.max(...times),
    min: Math.min(...times),
  }
}