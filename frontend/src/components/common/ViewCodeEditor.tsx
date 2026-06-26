import React, { useEffect, useState } from 'react';
import { Box, ToggleButton, ToggleButtonGroup, Alert } from '@mui/material';
import CodeEditor from './CodeEditor';
import CodeSearch from './CodeSearch';
import { load as yamlLoad, dump as yamlDump } from 'js-yaml';

interface ViewCodeEditorProps {
  viewData: any;
  setViewData: (d: any) => void;
  initialFormat?: 'json' | 'yaml';
}

const ViewCodeEditor: React.FC<ViewCodeEditorProps> = ({ viewData, setViewData, initialFormat = 'json' }) => {
  const [codeFormat, setCodeFormat] = useState<'json' | 'yaml'>(initialFormat);
  const [codeValue, setCodeValue] = useState<string>('');
  const [codeError, setCodeError] = useState<string | null>(null);
    const [editorApi, setEditorApi] = useState<any | null>(null);
    const [monacoApi, setMonacoApi] = useState<any | null>(null);

  useEffect(() => {
    try {
      const asJson = JSON.stringify(viewData || {}, null, 2);
      if (codeFormat === 'json') setCodeValue(asJson);
      else {
        try {
          const yaml = yamlDump(viewData || {});
          setCodeValue(yaml);
        } catch (e: any) {
          // fallback to JSON string if dump fails
          setCodeValue(asJson);
        }
      }
      setCodeError(null);
    } catch (e: any) {
      setCodeValue(String(viewData || ''));
    }
  }, [viewData, codeFormat]);

  return (
    <Box>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1 }}>
        <ToggleButtonGroup
          value={codeFormat}
          exclusive
          onChange={(_, v) => { if (v) setCodeFormat(v); }}
          size="small"
        >
          <ToggleButton value="json">JSON</ToggleButton>
          <ToggleButton value="yaml">YAML</ToggleButton>
        </ToggleButtonGroup>

        <Box component="small" sx={{ color: 'gray' }}>Edit the view as JSON or view YAML export</Box>
      </Box>

      {codeError && <Alert severity="error" sx={{ mb: 1 }}>{codeError}</Alert>}

      <Box component="style">{`.code-search-highlight { background-color: rgba(255,236,179,0.9); border-radius: 2px; }`}</Box>
      <CodeSearch editor={editorApi} monaco={monacoApi} />

      <CodeEditor
        value={codeValue}
        onChange={(val: string) => {
          setCodeValue(val);
          setCodeError(null);
          if (codeFormat === 'json') {
            try {
              const parsed = JSON.parse(val);
              setViewData(parsed);
            } catch (e: any) {
              setCodeError(`JSON parse error: ${String(e?.message || e)}`);
            }
          } else {
            try {
              const maybeJson = val.trim();
              if (maybeJson.startsWith('{') || maybeJson.startsWith('[')) {
                const parsed = JSON.parse(maybeJson);
                setViewData(parsed);
                return;
              }
              const parsed = yamlLoad(val);
              setViewData(parsed as any);
            } catch (e: any) {
              setCodeError(`YAML parse error: ${String(e?.message || e)}`);
            }
          }
        }}
        language={codeFormat === 'json' ? 'json' : 'yaml'}
        height="60vh"
        onEditorMount={(editor, monaco) => { setEditorApi(editor); setMonacoApi(monaco); }}
      />
    </Box>
  );
};

export default ViewCodeEditor;
