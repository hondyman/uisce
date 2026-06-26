import React, { useState, useEffect } from 'react';
import { FieldDefinition } from './types';

interface DynamicFormProps {
    fields: FieldDefinition[];
    onSubmit: (data: Record<string, any>) => void;
    initialData?: Record<string, any>;
    submitLabel?: string;
}

export const DynamicForm: React.FC<DynamicFormProps> = ({ fields, onSubmit, initialData = {}, submitLabel = 'Submit' }) => {
    const [formData, setFormData] = useState<Record<string, any>>(initialData);

    useEffect(() => {
        setFormData(initialData);
    }, [initialData]);

    const handleChange = (field: string, value: any) => {
        setFormData(prev => ({ ...prev, [field]: value }));
    };

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault();
        onSubmit(formData);
    };

    return (
        <form onSubmit={handleSubmit} className="space-y-4 p-4 border rounded shadow-sm bg-white">
            {fields.map((fieldDef) => (
                <div key={fieldDef.field} className="flex flex-col">
                    <label className="mb-1 font-medium text-gray-700">
                        {fieldDef.label}
                        {fieldDef.required && <span className="text-red-500 ml-1">*</span>}
                    </label>
                    {fieldDef.type === 'enum' ? (
                        <select
                            className="border rounded p-2 focus:ring-2 focus:ring-blue-500"
                            value={formData[fieldDef.field] || ''}
                            onChange={(e) => handleChange(fieldDef.field, e.target.value)}
                            required={fieldDef.required}
                        >
                            <option value="">Select...</option>
                            {fieldDef.options?.map((opt) => (
                                <option key={opt} value={opt}>{opt}</option>
                            ))}
                        </select>
                    ) : (
                        <input
                            type={fieldDef.type === 'number' ? 'number' : fieldDef.type === 'date' ? 'date' : 'text'}
                            className="border rounded p-2 focus:ring-2 focus:ring-blue-500"
                            value={formData[fieldDef.field] || ''}
                            onChange={(e) => handleChange(fieldDef.field, fieldDef.type === 'number' ? Number(e.target.value) : e.target.value)}
                            required={fieldDef.required}
                        />
                    )}
                </div>
            ))}
            <button
                type="submit"
                className="bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700 transition-colors"
            >
                {submitLabel}
            </button>
        </form>
    );
};
