export const bus = {
  listeners: {},

  on(event, handler) {
    if (!this.listeners[event]) this.listeners[event] = []
    this.listeners[event].push(handler)
  },

  dispatch(event, detail) {
    for (const handler of this.listeners[event] || []) {
      handler(detail)
    }
  }
}