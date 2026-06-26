export function generateContext(schema) {
  const ctx = {}
  for (const field of schema.fields) {
    ctx[field.name] = sampleValue(field.type)
  }
  return ctx
}

function sampleValue(type) {
  switch (type) {
    case 'string': return 'sample text'
    case 'number': return 42
    case 'boolean': return true
    case 'array': return []
    case 'object': return {}
    default: return null
  }
}