import React, { useEffect, useMemo, useState } from "react";
import { useForm, FormProvider, Controller, useWatch } from "react-hook-form";
import { StepSettingsSchema, StepFieldSchema } from "./ruleAwareFormTypes";

interface Props {
  schema: StepSettingsSchema;
  initialValues?: Record<string, any>;
  userRoles: string[];
  evalConditionJson: (cond: any, values: Record<string, any>) => boolean;
  onSubmit: (values: Record<string, any>) => Promise<void>;
}

export const RuleAwareStepForm: React.FC<Props> = ({
  schema,
  initialValues = {},
  userRoles,
  evalConditionJson,
  onSubmit,
}) => {
  const methods = useForm({
    defaultValues: initialValues,
    mode: "onChange",
  });

  const { control, handleSubmit, watch, formState: { errors }, setValue } = methods;
  const watchedValues = watch();

  // Apply dependent field updates when values change
  useEffect(() => {
    for (const section of schema.sections) {
      for (const field of section.fields) {
        if (field.dependents && watchedValues[field.key] !== undefined) {
          for (const dep of field.dependents) {
            if (dep.valueExpr) {
              try {
                const computed = evalDependentValue(dep.valueExpr, watchedValues);
                setValue(dep.field, computed, { shouldValidate: true });
              } catch (e) {
                console.error(`Failed to compute dependent ${dep.field}:`, e);
              }
            }
          }
        }
      }
    }
  }, [watchedValues, schema, setValue]);

  return (
    <FormProvider {...methods}>
      <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
        {schema.sections.map((section) => (
          <FormSection
            key={section.key}
            section={section}
            watchedValues={watchedValues}
            control={control}
            errors={errors}
            userRoles={userRoles}
            evalConditionJson={evalConditionJson}
          />
        ))}
        <div className="pt-4 border-t">
            <button type="submit" className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700">
            Save
            </button>
        </div>
      </form>
    </FormProvider>
  );
};

interface FormSectionProps {
  section: any;
  watchedValues: Record<string, any>;
  control: any;
  errors: any;
  userRoles: string[];
  evalConditionJson: (cond: any, values: Record<string, any>) => boolean;
}

const FormSection: React.FC<FormSectionProps> = ({
  section,
  watchedValues,
  control,
  errors,
  userRoles,
  evalConditionJson,
}) => {
  const [isCollapsed, setIsCollapsed] = useState(false);

  return (
    <div className="border rounded bg-white shadow-sm overflow-hidden">
      <div
        className={`px-4 py-3 bg-gray-50 border-b flex items-center justify-between ${section.collapsible ? 'cursor-pointer hover:bg-gray-100' : ''}`}
        onClick={() => section.collapsible && setIsCollapsed(!isCollapsed)}
      >
        <h3 className="font-semibold text-gray-800">
            {section.label}
        </h3>
         {section.collapsible && <span className="text-gray-500">{isCollapsed ? "▼" : "▲"}</span>}
      </div>
      {!isCollapsed && (
        <div className="p-4 space-y-4">
          {section.fields.map((field: StepFieldSchema) => (
            <FormField
              key={field.key}
              field={field}
              control={control}
              watchedValues={watchedValues}
              errors={errors}
              userRoles={userRoles}
              evalConditionJson={evalConditionJson}
            />
          ))}
        </div>
      )}
    </div>
  );
};

interface FormFieldProps {
  field: StepFieldSchema;
  control: any;
  watchedValues: Record<string, any>;
  errors: any;
  userRoles: string[];
  evalConditionJson: (cond: any, values: Record<string, any>) => boolean;
}

const FormField: React.FC<FormFieldProps> = ({
  field,
  control,
  watchedValues,
  errors,
  userRoles,
  evalConditionJson,
}) => {
  // Check visibility
  const isVisible = useMemo(() => {
    if (!field.visibility) return true;
    if (field.visibility.rolesAllowed) {
      if (!field.visibility.rolesAllowed.some((r) => userRoles.includes(r))) {
        return false;
      }
    }
    if (field.visibility.condition) {
      try {
        return evalConditionJson(field.visibility.condition, watchedValues);
      } catch (e) {
        console.error(`Visibility check failed for ${field.key}:`, e);
        return false;
      }
    }
    return true;
  }, [field.visibility, watchedValues, userRoles, evalConditionJson]);

  // Compute applicable validations
  const applicableValidations = useMemo(() => {
    return (field.validations ?? []).filter((v) => {
      if (!v.condition) return true;
      try {
        return evalConditionJson(v.condition, watchedValues);
      } catch {
        return false;
      }
    });
  }, [field.validations, watchedValues, evalConditionJson]);

  // Build React Hook Form validation rules
  const rules = useMemo(() => buildValidationRules(applicableValidations), [applicableValidations]);
  
  if (!isVisible) return null;

  return (
    <div className="mb-4">
      <Controller
        name={field.key}
        control={control}
        rules={rules}
        render={({ field: fieldProps }) => (
          <div className="flex flex-col">
            <label htmlFor={field.key} className="text-sm font-medium text-gray-700 mb-1">
                {field.label}
                {field.type !== 'checkbox' && rules.required && <span className="text-red-500 ml-1">*</span>}
            </label>
            {renderField(field, fieldProps)}
            {field.helpText && <div className="text-xs text-gray-500 mt-1">{field.helpText}</div>}
          </div>
        )}
      />
      {errors[field.key] && (
        <div className="text-red-500 text-xs mt-1">
          {errors[field.key]?.message?.toString() ?? `${field.label} is invalid`}
        </div>
      )}
    </div>
  );
};

function renderField(field: StepFieldSchema, fieldProps: any) {
    const commonClasses = "w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm border p-2";
    
  switch (field.type) {
    case "text":
      return (
        <input
          {...fieldProps}
          type="text"
          placeholder={field.placeholder}
          className={commonClasses}
        />
      );
    case "number":
      return (
        <input
          {...fieldProps}
          type="number"
          placeholder={field.placeholder}
          className={commonClasses}
        />
      );
    case "textarea":
      return (
        <textarea
          {...fieldProps}
          placeholder={field.placeholder}
          className={commonClasses}
          rows={4}
        />
      );
    case "select":
      return (
        <select {...fieldProps} className={commonClasses}>
          <option value="">-- Select --</option>
          {field.options?.map((opt) => (
            <option key={opt.value} value={opt.value}>
              {opt.label}
            </option>
          ))}
        </select>
      );
    case "checkbox":
      return (
        <div className="flex items-center h-5">
            <input
            {...fieldProps}
            checked={!!fieldProps.value}
            id={field.key}
            type="checkbox"
            className="h-4 w-4 rounded border-gray-300 text-indigo-600 focus:ring-indigo-500"
            />
        </div>
      );
    default:
      return null;
  }
}

function buildValidationRules(applicableValidations: any[]) {
  const rules: any = {};

  for (const v of applicableValidations) {
    switch (v.type) {
      case "required":
        rules.required = v.message;
        break;
      case "min":
        rules.min = { value: v.value, message: v.message };
        break;
      case "max":
        rules.max = { value: v.value, message: v.message };
        break;
      case "pattern":
        rules.pattern = { value: new RegExp(v.value), message: v.message };
        break;
      case "custom":
        rules.validate = () => v.message;
        break;
    }
  }

  return rules;
}

function evalDependentValue(expr: any, ctx: Record<string, any>): any {
  // Simple literal value
  if (expr.type === "literal") {
    return expr.value;
  }
  // Could extend to Starlark evaluation if needed
  return null;
}
