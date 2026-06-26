export class WorkerPoolService {
  constructor() {
    this.workers = 0
  }

  async spawn(count) {
    this.workers = count
  }

  evaluate(_rule, _context) {
    return this.workers > 0 ? { result: true } : null
  }
}