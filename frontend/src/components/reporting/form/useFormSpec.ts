import { useState, useCallback } from 'react';
import type { FormTemplateSpec, FormSection, FormFieldItem } from './FormManagerTypes';

export interface UseFormSpecReturn {
  formSpec: FormTemplateSpec | null;
  formRegistry: Record<string, FormTemplateSpec>;
  setFormSpec: (spec: FormTemplateSpec | null) => void;
  registerTemplate: (spec: FormTemplateSpec) => void;
  addSection: () => void;
  updateSection: (sectionId: string, patch: Partial<FormSection>) => void;
  deleteSection: (sectionId: string) => void;
  addField: (sectionId: string, item: FormFieldItem) => void;
  updateField: (sectionId: string, itemId: string, patch: Partial<FormFieldItem>) => void;
  deleteField: (sectionId: string, itemId: string) => void;
}

export function useFormSpec(
  initialSpec?: FormTemplateSpec | null,
  initialRegistry?: Record<string, FormTemplateSpec>
): UseFormSpecReturn {
  const [formSpec, setFormSpecState] = useState<FormTemplateSpec | null>(initialSpec || null);
  const [formRegistry, setFormRegistry] = useState<Record<string, FormTemplateSpec>>(
    initialRegistry || {}
  );

  const registerTemplate = useCallback((spec: FormTemplateSpec) => {
    setFormRegistry((prev) => ({ ...prev, [spec.templateId]: spec }));
  }, []);

  const setFormSpec = useCallback(
    (spec: FormTemplateSpec | null) => {
      setFormSpecState(spec);
      if (spec) {
        setFormRegistry((prev) => ({ ...prev, [spec.templateId]: spec }));
      }
    },
    []
  );

  const addSection = useCallback(() => {
    if (!formSpec) return;
    const newSection: FormSection = {
      id: `sec_${Date.now()}`,
      title: 'New Form Section',
      description: 'Drag fields or static elements into this section',
      columns: 2,
      items: [],
    };
    const updated = { ...formSpec, sections: [...formSpec.sections, newSection] };
    setFormSpec(updated);
  }, [formSpec, setFormSpec]);

  const updateSection = useCallback(
    (sectionId: string, patch: Partial<FormSection>) => {
      if (!formSpec) return;
      const updated = {
        ...formSpec,
        sections: formSpec.sections.map((s) =>
          s.id === sectionId ? { ...s, ...patch } : s
        ),
      };
      setFormSpec(updated);
    },
    [formSpec, setFormSpec]
  );

  const deleteSection = useCallback(
    (sectionId: string) => {
      if (!formSpec) return;
      const updated = {
        ...formSpec,
        sections: formSpec.sections.filter((s) => s.id !== sectionId),
      };
      setFormSpec(updated);
    },
    [formSpec, setFormSpec]
  );

  const addField = useCallback(
    (sectionId: string, item: FormFieldItem) => {
      if (!formSpec) return;
      const updated = {
        ...formSpec,
        sections: formSpec.sections.map((s) =>
          s.id === sectionId ? { ...s, items: [...s.items, item] } : s
        ),
      };
      setFormSpec(updated);
    },
    [formSpec, setFormSpec]
  );

  const updateField = useCallback(
    (sectionId: string, itemId: string, patch: Partial<FormFieldItem>) => {
      if (!formSpec) return;
      const updated = {
        ...formSpec,
        sections: formSpec.sections.map((s) =>
          s.id === sectionId
            ? { ...s, items: s.items.map((i) => (i.id === itemId ? { ...i, ...patch } : i)) }
            : s
        ),
      };
      setFormSpec(updated);
    },
    [formSpec, setFormSpec]
  );

  const deleteField = useCallback(
    (sectionId: string, itemId: string) => {
      if (!formSpec) return;
      const updated = {
        ...formSpec,
        sections: formSpec.sections.map((s) =>
          s.id === sectionId ? { ...s, items: s.items.filter((i) => i.id !== itemId) } : s
        ),
      };
      setFormSpec(updated);
    },
    [formSpec, setFormSpec]
  );

  return {
    formSpec,
    formRegistry,
    setFormSpec,
    registerTemplate,
    addSection,
    updateSection,
    deleteSection,
    addField,
    updateField,
    deleteField,
  };
}
