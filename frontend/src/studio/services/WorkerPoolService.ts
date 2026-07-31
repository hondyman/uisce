export class WorkerPoolService {
  workers: number;
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