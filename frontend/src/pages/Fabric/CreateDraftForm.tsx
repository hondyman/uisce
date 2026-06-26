// React import removed (automatic JSX runtime)
import { devLog } from '../../utils/devLogger';
import { gql, useMutation, ApolloCache, FetchResult } from '@apollo/client';
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

// ----- GraphQL mutation -----
export const CREATE_DRAFT = gql`
  mutation CreateDraft($input: fabric_defn_insert_input!) {
    insert_fabric_defn_one(object: $input) {
      id
      title
      model_key
      version
      description
    }
  }
`;

export type CreateDraftData = {
  insert_fabric_defn_one: {
    id: string;
    title: string;
    model_key: string;
    version: number;
    description?: string | null;
    // Add __typename for Apollo cache
    __typename?: string;
  };
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
  const [mutate, { loading, error }] = useMutation<CreateDraftData, CreateDraftVariables>(CREATE_DRAFT, {
    optimisticResponse: {
      insert_fabric_defn_one: {
        __typename: 'fabric_defn',
        id: 'temp-id',
        model_key: 'orders',
        version: 3,
        title: 'Orders v3',
        description: 'Adds revenue measures',
      },
    },
    update: (cache: ApolloCache<CreateDraftData>, result: FetchResult<CreateDraftData>) => {
      const newDraft = result.data?.insert_fabric_defn_one;
      if (!newDraft) return;
      cache.modify({
        fields: {
          fabric_defn(existingRefs = []) {
            const newRef = cache.writeFragment({
              data: newDraft,
              fragment: gql`
                fragment NewDraft on fabric_defn {
                  id
                  title
                  model_key
                  version
                  description
                }
              `,
            });
            return [newRef, ...existingRefs];
          },
        },
      });
    },
    onCompleted: (data: CreateDraftData) =>
  devLog('Created draft:', data.insert_fabric_defn_one?.id),
  });

  const hook = {
    submit: async (vars: Record<string, unknown>) => {
      await mutate({ variables: vars as CreateDraftVariables });
    },
    submitting: loading,
    error: error ? { message: error.message } : null,
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
      render={(ctx: FormContextValue<DraftFormValues>) => (
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