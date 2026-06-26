export type ConditionJson = any; // your AdvancedConditionBuilder JSON schema

export interface ApproverRule {
    id: string;
    label: string;         // e.g. "US High Value"
    condition: ConditionJson;
    actorRole: string;     // "Manager", "Director", "Compliance"
}

export interface ApprovalConfig {
    stepKey: string;
    rules: ApproverRule[];
    fallbackRole?: string;
}
