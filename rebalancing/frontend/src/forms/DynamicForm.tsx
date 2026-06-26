import React from "react";
import { FormSchema } from "./formSchema";
import { isFieldVisible, validateField } from "./formLogic";

interface Props {
  schema: FormSchema;
  values: Record<string, any>;
  onChange: (key: string, value: any) => void;
  userRoles: string[];
  evalConditionJson: (cond: any, values: Record<string, any>) => boolean;
}

export const DynamicForm: React.FC<Props> = ({
  schema,
  values,
  onChange,
  userRoles,
  evalConditionJson,
}) => {
  return (
    <div className="space-y-6">
      {schema.sections.map((section) => (
        <div key={section.key} className="form-section border p-4 rounded bg-white shadow-sm">
          <h3 className="text-lg font-bold mb-4 border-b pb-2">{section.label}</h3>
          <div className="space-y-4">
            {section.fields.map((f) => {
              if (!isFieldVisible(f.visibility, values, userRoles, evalConditionJson)) {
                return null;
              }

              const fieldErrors = validateField(f.validations, values, evalConditionJson);
              const value = values[f.key];

              const common = {
                className: "w-full border rounded p-2 text-sm",
                value: value ?? "",
                onChange: (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) =>
                  onChange(f.key, f.component === "checkbox" ? (e.target as HTMLInputElement).checked : e.target.value),
                ...f.props,
              };

              return (
                <div key={f.key} className="form-field">
                  <label className="block text-sm font-semibold text-gray-700 mb-1">
                    {f.label}
                    {f.component === "checkbox" && (
                         <input type="checkbox" className="ml-2" checked={!!value} {...common} value={undefined} />
                    )}
                  </label>
                  
                  {f.component === "text" && <input type="text" {...common} />}
                  {f.component === "number" && <input type="number" {...common} />}
                  
                  {f.component === "select" && (
                    <select {...common}>
                      <option value="">Select...</option>
                      {f.props?.options?.map((opt: any) => (
                        <option key={opt.value} value={opt.value}>
                          {opt.label}
                        </option>
                      ))}
                    </select>
                  )}
                  
                  {fieldErrors.map((msg) => (
                    <div key={msg} className="text-red-500 text-xs mt-1">
                      {msg}
                    </div>
                  ))}
                </div>
              );
            })}
          </div>
        </div>
      ))}
    </div>
  );
};
