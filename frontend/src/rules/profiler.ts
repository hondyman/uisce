export async function profileRule(rule: any, contexts: any[], pool: any) {
  const times: number[] = []
  for (const ctx of contexts) {
    const start = performance.now()
    await pool.evaluate(rule, ctx)
    times.push(performance.now() - start)
  }
  return {
    avg: times.reduce((a: number, b: number) => a + b, 0) / times.length,
    max: Math.max(...times),
    min: Math.min(...times),
  }
}