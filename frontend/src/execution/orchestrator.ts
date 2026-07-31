export class ExecutionOrchestrator {
  pool: any;
  router: any;
  budget: any;

  constructor(pool: any, router: any, budget: any) {
    this.pool = pool
    this.router = router
    this.budget = budget
  }

  async execute(plan: any) {
    this.budget.check()
    return await executePlan(plan, this.router)
  }
}

async function executePlan(plan: any, router: any) {
  return { success: true, plan, router };
}