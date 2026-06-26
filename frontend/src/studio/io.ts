export function exportRule(rule) {
  return JSON.stringify(rule, null, 2)
}

export function importRule(json) {
  return JSON.parse(json)
}