// src/runtime/evalExpression.ts

export function evalExpression(
    expr: string,
    ctx: { state: any; roles?: string[]; data: any }
): any {
    try {
        // Safe-ish evaluator using Function
        // In production, consider a real expression parser like 'jsep'
        const fn = new Function("state", "roles", "data", `return (${expr});`);
        return fn(ctx.state, ctx.roles || [], ctx.data);
    } catch (err) {
        console.error(`Failed to evaluate expression: ${expr}`, err);
        return false;
    }
}
