export function diffRules(a, b) {
  const changes = []

  function walk(path, aNode, bNode) {
    if (JSON.stringify(aNode) === JSON.stringify(bNode)) return

    if (!aNode) {
      changes.push({ path, type: "added", value: bNode })
      return
    }

    if (!bNode) {
      changes.push({ path, type: "removed", value: aNode })
      return
    }

    if (aNode.Type !== bNode.Type) {
      changes.push({ path, type: "changed-type", from: aNode.Type, to: bNode.Type })
      return
    }

    for (const key of new Set([...Object.keys(aNode), ...Object.keys(bNode)])) {
      walk([...path, key], aNode[key], bNode[key])
    }
  }

  walk([], a, b)
  return changes
}