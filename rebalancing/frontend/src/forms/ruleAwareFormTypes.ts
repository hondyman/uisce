export type ConditionJson = any;

export interface FieldValidationRule {
    condition?: ConditionJson;  // when this validation applies
    type: "required" | "pattern" | "min" | "max" | "custom";
    message: string;
    value?: any;                 // e.g., min value, regex pattern
}

export interface FieldVisibilityRule {
    condition?: ConditionJson;   // when field is visible
    rolesAllowed?: string[];     // RBAC
}

export interface DependentValue {
    field: string;              // which field to set
    valueExpr: ConditionJson;   // Starlark to compute value
}

export interface StepFieldSchema {
    key: string;
    label: string;
    type: "text" | "number" | "select" | "checkbox" | "textarea";
    placeholder?: string;
    options?: Array<{ value: string; label: string }>;
    visibility?: FieldVisibilityRule;
    validations?: FieldValidationRule[];
    dependents?: DependentValue[];  // cascade updates to other fields
    helpText?: string;
}

export interface StepSectionSchema {
    key: string;
    label: string;
    collapsible?: boolean;
    fields: StepFieldSchema[];
}

export interface StepSettingsSchema {
    stepKey: string;
    sections: StepSectionSchema[];
}

export interface StepBranch {
    id: string;
    label: string;
    condition: ConditionJson;
    nextStepKey: string;
}

export interface WizardStepDef {
    key: string;
    label: string;
    schema: StepSettingsSchema;
    branches?: StepBranch[];
    defaultNextStep?: string;
}
