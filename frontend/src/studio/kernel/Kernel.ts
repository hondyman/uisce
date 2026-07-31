import { KernelServices } from "./KernelServices"
import { KernelLifecycle } from "./KernelLifecycle"
import { StudioState } from "../state/StudioState"
import { bus } from "../events/StudioEvents"

export class Kernel {
  services: any;
  lifecycle: any;
  state: any;
  events: any;
  ready: boolean = false;

  constructor() {
    this.services = new KernelServices()
    this.lifecycle = new KernelLifecycle()
    this.state = new StudioState()
    this.events = bus
    this.ready = false
  }

  async start() {
    await this.lifecycle.start(this)
    this.ready = true
    this.events.dispatch("kernel.ready")
  }
}