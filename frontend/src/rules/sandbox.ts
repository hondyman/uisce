export class RuleSandbox {
  constructor(pool) {
    this.pool = pool
  }

  async run(rule, context) {
    return await this.pool.evaluate(rule, context)
  }
}