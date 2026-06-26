import React, { useMemo, useRef } from 'react';
import Editor from '@monaco-editor/react';
import { Box } from '@mui/material';

interface JsonMonacoEditorProps {
  value: string | object | undefined;
  onChange: (v: string) => void;
  height?: string;
  readOnly?: boolean;
  schema?: object | null;
  schemaUrn?: string;
  'data-testid'?: string;
}

const JsonMonacoEditor: React.FC<JsonMonacoEditorProps> = ({ value, onChange, height = '200px', readOnly = false, schema = null, schemaUrn, 'data-testid': dataTestId }) => {
  const initial = useMemo(() => {
    if (typeof value === 'string') return value;
    try {
      return value ? JSON.stringify(value, null, 2) : '';
    } catch (err) {
      return String(value ?? '');
    }
  }, [value]);

  const monacoRef = useRef<any>(null);

  return (
    <Box sx={{ '& .monaco-editor': { borderRadius: 1 } }}>
      <Editor
        defaultLanguage="json"
        value={initial}
        height={height}
        options={{ readOnly, minimap: { enabled: false } }}
        onChange={(v) => onChange(v ?? '')}
        theme="vs-light"
        beforeMount={(monaco) => {
          monacoRef.current = monaco;
          try {
            // If the caller passed a JSON schema use it for inline diagnostics
            if (schema && monaco?.languages?.json?.jsonDefaults && typeof monaco.languages.json.jsonDefaults.setDiagnosticsOptions === 'function') {
              monaco.languages.json.jsonDefaults.setDiagnosticsOptions({
                validate: true,
                allowComments: false,
                schemas: [
                  {
                    uri: schemaUrn || `inmemory://schema/${Date.now()}`,
                    fileMatch: ['*'],
                    schema: schema,
                  },
                ],
              });
            }
          } catch (e) {
            // silently ignore schema failures
          }
        }}
        data-testid={dataTestId}
      />
    </Box>
  );
};

export default JsonMonacoEditor;
