export class KernelLifecycle {
  async start(kernel: any) {
    await kernel.services.wasm.init()
    await kernel.services.pool.init()
    await kernel.services.monaco.init()
    await kernel.services.plugins.load(kernel)
    await kernel.services.persistence.restore(kernel)
    kernel.services.telemetry.start()
  }
}