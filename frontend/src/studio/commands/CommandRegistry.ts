export class CommandRegistry {
  constructor() {
    this.commands = new Map()
  }

  register(cmd) {
    this.commands.set(cmd.id, cmd)
  }

  execute(id, kernel) {
    const cmd = this.commands.get(id)
    if (cmd) cmd.run(kernel)
  }

  search(query) {
    const results = []
    for (const [id, cmd] of this.commands) {
      if (cmd.title.toLowerCase().includes(query.toLowerCase())) {
        results.push(cmd)
      }
    }
    return results
  }
}