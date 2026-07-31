export class WorkerPoolService {
  workers: number;
  constructor() {
    this.workers = 0
  }

  async spawn(count: any) {
    this.workers = count
  }

  evaluate(_rule: any, _context: any) {
    return this.workers > 0 ? { result: true } : null
  }
}