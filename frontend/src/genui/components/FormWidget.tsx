import React from "react";
import type { FormComponent } from "../schema";
import { useMutation, gql } from "@apollo/client";

interface FormWidgetProps {
  def: FormComponent;
}

export function FormWidget({ def }: FormWidgetProps) {
  const [formData, setFormData] = React.useState<Record<string, any>>({});

  const [submitMutation, { loading }] = useMutation(gql(def.submitAction.mutation));

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    try {
      await submitMutation({ variables: formData });
      if (def.submitAction.successMessage) {
        alert(def.submitAction.successMessage);
      }
    } catch (error) {
      console.error("Form submission error:", error);
    }
  };

  return (
    <div className="bg-white rounded-lg shadow p-6">
      {def.title && <h3 className="text-lg font-semibold mb-4">{def.title}</h3>}
      
      <form onSubmit={handleSubmit} className="space-y-4">
        {def.fields.map((field) => (
          <div key={field.name}>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              {field.label}
              {field.required && <span className="text-red-500 ml-1">*</span>}
            </label>
            
            {renderField(field, formData, setFormData)}
          </div>
        ))}

        <button
          type="submit"
          disabled={loading}
          className="w-full bg-blue-600 text-white py-2 px-4 rounded-lg hover:bg-blue-700 disabled:opacity-50"
        >
          {loading ? "Submitting..." : "Submit"}
        </button>
      </form>
    </div>
  );
}

function renderField(
  field: FormComponent["fields"][0],
  formData: Record<string, any>,
  setFormData: React.Dispatch<React.SetStateAction<Record<string, any>>>
) {
  const value = formData[field.name] || "";
  const onChange = (val: any) => setFormData((prev) => ({ ...prev, [field.name]: val }));

  switch (field.type) {
    case "text":
    case "number":
    case "date":
      return (
        <input
          type={field.type}
          required={field.required}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          className="w-full border border-gray-300 rounded-lg px-3 py-2"
        />
      );

    case "textarea":
      return (
        <textarea
          required={field.required}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          className="w-full border border-gray-300 rounded-lg px-3 py-2"
          rows={4}
        />
      );

    case "select":
      return (
        <select
          required={field.required}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          className="w-full border border-gray-300 rounded-lg px-3 py-2"
        >
          <option value="">Select...</option>
          {field.options?.map((opt) => (
            <option key={opt.value} value={opt.value}>
              {opt.label}
            </option>
          ))}
        </select>
      );

    case "checkbox":
      return (
        <input
          type="checkbox"
          checked={!!value}
          onChange={(e) => onChange(e.target.checked)}
          className="w-4 h-4"
        />
      );

    default:
      return null;
  }
}
