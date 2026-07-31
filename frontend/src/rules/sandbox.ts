export class RuleSandbox {
  pool: any;
  constructor(pool: any) {
    this.pool = pool
  }

  async run(rule: any, context: any) {
    return await this.pool.evaluate(rule, context)
  }
}