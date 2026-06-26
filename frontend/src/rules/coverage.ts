export function computeCoverage(trace) {
  const covered = new Set()
  walk(trace)
  return covered

  function walk(node) {
    covered.add(node.nodeType)
    for (const child of node.children || []) walk(child)
  }
}