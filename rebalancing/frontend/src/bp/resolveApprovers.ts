import { ApproverRule } from "./approvers";

export function resolveApproverRole(
    rules: ApproverRule[],
    fallbackRole: string | undefined,
    ctx: Record<string, any>,                    // flattened context (entity/fields)
    evalConditionJson: (cond: any, ctx: any) => boolean
): string | undefined {
    for (const rule of rules) {
        if (rule.condition && evalConditionJson(rule.condition, ctx)) {
            return rule.actorRole;
        }
    }
    return fallbackRole;
}
