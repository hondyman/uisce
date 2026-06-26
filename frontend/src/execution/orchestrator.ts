export class ExecutionOrchestrator {
  constructor(pool, router, budget) {
    this.pool = pool
    this.router = router
    this.budget = budget
  }

  async execute(plan) {
    this.budget.check()
    return await executePlan(plan, this.router)
  }
}