export function safeEvaluate(fn: any) {
  try {
    return fn()
  } catch (e: any) {
    return { error: e.message }
  }
}