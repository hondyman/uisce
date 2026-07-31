export function computeCoverage(trace: any) {
  const covered = new Set()
  walk(trace)
  return covered

  function walk(node: any) {
    covered.add(node.nodeType)
    for (const child of node.children || []) walk(child)
  }
}