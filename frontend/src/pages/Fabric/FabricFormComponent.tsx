import * as React from 'react'; // required: code uses React.createElement and React types
import { devLog, devError } from '../../utils/devLogger';
import { ZodSchema } from 'zod';
import { useMutationForm, Form } from '../../graphql/helpers/fabric_form_helpers';

// Example usage component
interface ExampleFormData {
  id: string;
  title: string;
}

interface ExampleFormVars {
  title: string;
  description?: string;
}

interface FabricFormComponentProps<TData, TVars extends Record<string, any>> {
  schema: ZodSchema<TVars>;
  initialValues: TVars;
  mutationFn: (variables: TVars) => Promise<TData>;
  onSuccess?: (data: TData) => void;
  onError?: (error: Error) => void;
  render: (props: {
    values: TVars;
    setValue: <K extends keyof TVars>(key: K, value: TVars[K]) => void;
    errors: Partial<Record<keyof TVars, string>>;
    submitting: boolean;
  }) => React.ReactNode;
}

export function FabricFormComponent<TData, TVars extends Record<string, any>>({
  schema,
  initialValues,
  mutationFn,
  onSuccess,
  onError,
  render,
}: FabricFormComponentProps<TData, TVars>): React.ReactElement {
  const useFormHook = useMutationForm<TData, TVars>({
    mutationFn,
    onSuccess,
    onError,
  });

  return React.createElement(Form<TData, TVars>, {
    schema,
    useFormHook,
    initialValues,
    render,
  });
}

// Example of how to use it
export function ExampleUsage(): React.ReactElement {
  // This would typically come from your GraphQL mutation
  const exampleMutationFn = async (variables: ExampleFormVars): Promise<ExampleFormData> => {
    // Simulate API call
    await new Promise(resolve => setTimeout(resolve, 1000));
    return {
      id: 'generated-id',
      title: variables.title,
    };
  };

  return React.createElement(FabricFormComponent<ExampleFormData, ExampleFormVars>, {
    schema: {} as any, // Placeholder for your Zod schema
    initialValues: {
      title: '',
      description: '',
    },
    mutationFn: exampleMutationFn,
  onSuccess: (data) => devLog('Success:', data),
  onError: (error) => { try { devError('Error:', error); } catch {} },
    render: ({ values, setValue, errors, submitting }) =>
      React.createElement(
        React.Fragment,
        null,
        React.createElement('input', {
          value: values.title,
          onChange: (e: React.ChangeEvent<HTMLInputElement>) => setValue('title', e.target.value),
          placeholder: 'Title',
        }),
        errors.title && React.createElement('div', { style: { color: 'red' } }, errors.title),
        React.createElement('textarea', {
          value: values.description || '',
          onChange: (e: React.ChangeEvent<HTMLTextAreaElement>) => setValue('description', e.target.value),
          placeholder: 'Description',
        }),
        React.createElement(
          'button',
          { type: 'submit', disabled: submitting },
          submitting ? 'Submitting...' : 'Submit'
        )
      ),
  });
}