import { FieldVisibilityRule, FieldValidationRule } from "./formSchema";

export function isFieldVisible(
    rule: FieldVisibilityRule | undefined,
    values: Record<string, any>,
    userRoles: string[],
    evalConditionJson: (cond: any, values: Record<string, any>) => boolean
): boolean {
    if (!rule) return true;

    if (rule.rolesAllowed && !rule.rolesAllowed.some((r) => userRoles.includes(r))) {
        return false;
    }
    if (rule.condition && !evalConditionJson(rule.condition, values)) {
        return false;
    }
    return true;
}

export function validateField(
    validations: FieldValidationRule[] | undefined,
    values: Record<string, any>,
    evalConditionJson: (cond: any, values: Record<string, any>) => boolean
): string[] {
    if (!validations) return [];
    const errors: string[] = [];
    for (const v of validations) {
        if (!v.condition || evalConditionJson(v.condition, values)) {
            // condition true => validation rule active
            errors.push(v.message);
        }
    }
    return errors;
}
