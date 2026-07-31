export function exportRule(rule: any) {
  return JSON.stringify(rule, null, 2)
}

export function importRule(json: any) {
  return JSON.parse(json)
}