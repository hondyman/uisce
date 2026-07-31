export class CommandRegistry {
  commands: Map<string, any>;
  constructor() {
    this.commands = new Map()
  }

  register(cmd: any) {
    this.commands.set(cmd.id, cmd)
  }

  execute(id: string, kernel: any) {
    const cmd = this.commands.get(id)
    if (cmd) cmd.run(kernel)
  }

  search(query: string) {
    const results: any[] = []
    for (const [id, cmd] of this.commands) {
      if (cmd.title.toLowerCase().includes(query.toLowerCase())) {
        results.push(cmd)
      }
    }
    return results
  }
}