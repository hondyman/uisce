import React, { useCallback, useEffect, useRef, useState } from 'react';
import { Box, Dialog, DialogTitle, DialogContent, DialogActions, Button, IconButton, Tooltip, Typography, Alert } from '@mui/material';
import OpenInFullIcon from '@mui/icons-material/OpenInFull';
import CloseFullscreenIcon from '@mui/icons-material/CloseFullscreen';
import Editor, { OnMount } from '@monaco-editor/react';
import type * as Monaco from 'monaco-editor';
import { registerUisceExpressionLanguage, UISCE_EXPRESSION_LANGUAGE, setAslFields } from '../../rules/aslMonacoRegistry';
import { parseExpressionWasm, ExpressionParseError } from '../../rules/wasmRuntime';
import apiClient from '../../utils/apiClient';

interface ExpressionEditorFieldProps {
  label: string;
  value: string;
  onChange: (value: string) => void;
  /** BO key (e.g. "order") whose semantic terms drive intellisense - the same vocabulary the central Rule Builder offers, not raw columns. */
  boName?: string;
  minHeight?: number;
}

/**
 * The centralized expression editor (Monaco + the ASL grammar + semantic-
 * term intellisense from aslMonacoRegistry, same engine the Rule Builder's
 * Expression mode uses) as a drop-in field: renders inline at `minHeight`,
 * with a pop-out button that reopens the identical editor larger in a
 * modal for more room. Both surfaces edit the same `value`/`onChange` -
 * there is exactly one source of truth, never a separate draft that has
 * to be reconciled on close.
 */
const ExpressionEditorField: React.FC<ExpressionEditorFieldProps> = ({
  label, value, onChange, boName, minHeight = 120,
}) => {
  const [popoutOpen, setPopoutOpen] = useState(false);
  const [parseError, setParseError] = useState<{ message: string; pos: number } | null>(null);
  const monacoRef = useRef<typeof Monaco | null>(null);
  const editorRef = useRef<Monaco.editor.IStandaloneCodeEditor | null>(null);

  useEffect(() => {
    if (!boName) return;
    let cancelled = false;
    apiClient<{ fields: { name: string; dataType: string; cardinality?: string }[] }>(
      `/validation-rule-nodes/bo-fields?bo_name=${encodeURIComponent(boName)}`
    )
      .then((res) => {
        if (cancelled) return;
        setAslFields((res.fields || []).map((f) => ({ name: f.name, type: f.dataType })));
      })
      .catch(() => { /* intellisense degrades to functions-only, not fatal */ });
    return () => { cancelled = true; };
  }, [boName]);

  useEffect(() => {
    const handle = setTimeout(() => {
      if (!value.trim()) { setParseError(null); return; }
      parseExpressionWasm(value)
        .then(() => {
          setParseError(null);
          if (monacoRef.current && editorRef.current) {
            monacoRef.current.editor.setModelMarkers(editorRef.current.getModel()!, 'uisce-expr-field', []);
          }
        })
        .catch((err) => {
          if (err instanceof ExpressionParseError) {
            setParseError({ message: err.message, pos: err.pos });
          } else {
            setParseError({ message: err instanceof Error ? err.message : String(err), pos: 0 });
          }
        });
    }, 300);
    return () => clearTimeout(handle);
  }, [value]);

  const handleMount: OnMount = useCallback((editor, monaco) => {
    editorRef.current = editor;
    monacoRef.current = monaco;
  }, []);

  const editor = (height: number) => (
    <Editor
      height={height}
      language={UISCE_EXPRESSION_LANGUAGE}
      value={value}
      onChange={(v) => onChange(v ?? '')}
      beforeMount={(monaco) => { void registerUisceExpressionLanguage(monaco); }}
      onMount={handleMount}
      options={{
        minimap: { enabled: false },
        fontSize: 13,
        lineNumbers: 'off',
        folding: false,
        scrollBeyondLastLine: false,
      }}
    />
  );

  return (
    <Box>
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 0.5 }}>
        <Typography variant="caption" color="text.secondary">{label}</Typography>
        <Tooltip title="Pop out to a larger editor">
          <IconButton size="small" onClick={() => setPopoutOpen(true)}>
            <OpenInFullIcon sx={{ fontSize: 14 }} />
          </IconButton>
        </Tooltip>
      </Box>
      <Box sx={{ border: '1px solid', borderColor: 'divider', borderRadius: 1, overflow: 'hidden' }}>
        {editor(minHeight)}
      </Box>
      {parseError && (
        <Typography variant="caption" color="error" sx={{ display: 'block', mt: 0.5 }}>
          {parseError.message}
        </Typography>
      )}

      <Dialog open={popoutOpen} onClose={() => setPopoutOpen(false)} maxWidth="md" fullWidth>
        <DialogTitle sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          {label}
          <IconButton size="small" onClick={() => setPopoutOpen(false)}>
            <CloseFullscreenIcon sx={{ fontSize: 16 }} />
          </IconButton>
        </DialogTitle>
        <DialogContent>
          <Box sx={{ border: '1px solid', borderColor: 'divider', borderRadius: 1, overflow: 'hidden', mt: 1 }}>
            {editor(360)}
          </Box>
          {parseError && <Alert severity="error" sx={{ mt: 2 }}>{parseError.message}</Alert>}
        </DialogContent>
        <DialogActions>
          <Button variant="contained" onClick={() => setPopoutOpen(false)}>Done</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default ExpressionEditorField;
