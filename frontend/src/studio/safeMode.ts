export function safeEvaluate(fn) {
  try {
    return fn()
  } catch (e) {
    return { error: e.message }
  }
}