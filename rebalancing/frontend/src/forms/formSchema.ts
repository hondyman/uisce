export type ConditionJson = any;

export interface FieldVisibilityRule {
    condition?: ConditionJson;       // when to show
    rolesAllowed?: string[];         // who can see it
}

export interface FieldValidationRule {
    condition?: ConditionJson;       // when this validation applies
    message: string;
}

export type FieldComponent = "text" | "number" | "select" | "checkbox";

export interface FieldSchema {
    key: string;
    label: string;
    component: FieldComponent;
    props?: Record<string, any>;
    visibility?: FieldVisibilityRule;
    validations?: FieldValidationRule[];
}

export interface SectionSchema {
    key: string;
    label: string;
    fields: FieldSchema[];
}

export interface FormSchema {
    sections: SectionSchema[];
}
