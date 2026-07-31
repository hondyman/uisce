export const bus: any = {
  listeners: {} as any,

  on(event: any, handler: any) {
    if (!this.listeners[event]) this.listeners[event] = []
    this.listeners[event].push(handler)
  },

  dispatch(event: any, detail: any) {
    for (const handler of this.listeners[event] || []) {
      handler(detail)
    }
  }
}