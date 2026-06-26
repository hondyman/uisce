export function detectRegressions(oldResults, newResults) {
  const regressions = []
  for (let i = 0; i < oldResults.length; i++) {
    if (oldResults[i] !== newResults[i]) {
      regressions.push(i)
    }
  }
  return regressions
}