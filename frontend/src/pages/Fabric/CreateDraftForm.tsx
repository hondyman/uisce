// React import removed (automatic JSX runtime)
import { devLog } from '../../utils/devLogger';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { apiFetch } from '../../lib/apiClient';
import { z } from 'zod';
import type { JSONValue } from '../../types/json';

// Hooks & components from your fabric_form.ts file
import {
  Form,
  InputField,
  NumberField,
  TextAreaField,
  FormContextValue,
} from '../../graphql/helpers/fabric_form';

export type CreateDraftData = {
  id: string;
  title: string;
  model_key: string;
  version: number;
  description?: string | null;
};

export type CreateDraftVariables = {
  input: {
    model_key: string;
    version: number;
    title: string;
    description?: string;
  source_config: JSONValue;
  resolved_config: JSONValue;
  };
};

// Define the form values type
type DraftFormValues = {
  input: {
    model_key: string;
    version: number;
    title: string;
    description?: string;
  source_config: JSONValue;
  resolved_config: JSONValue;
  };
};

// ----- Zod schema -----
const draftSchema = z.object({
  input: z.object({
    model_key: z.string().min(1, 'Model key required'),
    version: z.number().int().positive('Must be positive'),
    title: z.string().min(1, 'Title required'),
    description: z.string().optional(),
  source_config: z.any() as z.ZodType<JSONValue>,
  resolved_config: z.any() as z.ZodType<JSONValue>,
  }),
});

// ----- Component -----
export function CreateDraftForm(): JSX.Element {
  const queryClient = useQueryClient();

  const { mutate, isPending: loading, error } = useMutation({
    mutationFn: (input: CreateDraftVariables['input']) =>
      apiFetch('/api/rest/fabric-defns', {
        method: 'POST',
        body: JSON.stringify(input),
      }).then(r => r.json()),
    onSuccess: (data: CreateDraftData) => {
      devLog('Created draft:', data.id);
      queryClient.invalidateQueries({ queryKey: ['fabric-defns'] });
    },
  });

  const hook = {
    submit: async (vars: Record<string, unknown>) => {
      await mutate(vars as CreateDraftVariables['input']);
    },
    submitting: loading,
    error: error ? { message: (error as Error).message } : null,
  };

  return (
    <Form<DraftFormValues>
      schema={draftSchema}
      initialValues={{
        input: {
          model_key: '',
          version: 1,
          title: '',
          description: '',
          source_config: {},
          resolved_config: {},
        },
      }}
      hook={hook}
      render={(ctx: any) => (
        <>
          {/* Model key with ILIKE mask */}
          <InputField<DraftFormValues, 'input.model_key'>
            name="input.model_key"
            label="Model Key"
            required
            transform={(v: unknown) => `%${String(v).trim()}%`} // ILIKE wrap
          />

          <NumberField<DraftFormValues, 'input.version'>
            name="input.version"
            label="Version"
            required
          />

          <InputField<DraftFormValues, 'input.title'>
            name="input.title"
            label="Title"
            required
            transform={(v: unknown) => String(v).trim()} // simple trim
          />

          <TextAreaField<DraftFormValues, 'input.description'>
            name="input.description"
            label="Description"
            transform={(v: unknown) => String(v).trim()}
          />

          <button type="submit" disabled={ctx.submitting}>
            {ctx.submitting ? 'Creating…' : 'Create Draft'}
          </button>
        </>
      )}
    />
  );
}