export function generateContext(schema: any) {
  const ctx: any = {}
  for (const field of schema.fields) {
    ctx[field.name] = sampleValue(field.type)
  }
  return ctx
}

function sampleValue(type: any) {
  switch (type) {
    case 'string': return 'sample text'
    case 'number': return 42
    case 'boolean': return true
    case 'array': return []
    case 'object': return {}
    default: return null
  }
}