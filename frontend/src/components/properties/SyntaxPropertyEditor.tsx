import React, { lazy, Suspense } from 'react';
import { Box, TextField, CircularProgress, Typography } from '@mui/material';

interface SyntaxPropertyEditorProps {
  value: string;
  onChange: (value: string) => void;
  language: 'sql' | 'yaml' | 'json' | null;
  label?: string;
  placeholder?: string;
  readOnly?: boolean;
  height?: string;
}

// Lazy load MonacoCodeEditor to reduce bundle size
const MonacoCodeEditor = lazy(() => import('../UnifiedSemanticBuilder/MonacoCodeEditor.lazy'));

/**
 * Component for editing string properties with optional syntax highlighting.
 * Uses Monaco Editor when a language is specified, falls back to TextField otherwise.
 */
export const SyntaxPropertyEditor: React.FC<SyntaxPropertyEditorProps> = ({
  value,
  onChange,
  language,
  label = 'Value',
  placeholder = '',
  readOnly = false,
  height = '200px',
}) => {
  // If no language is specified, use a simple text field
  if (!language) {
    return (
      <TextField
        fullWidth
        multiline
        rows={5}
        label={label}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        InputProps={{
          readOnly,
        }}
      />
    );
  }

  // Use Monaco Editor for syntax highlighting
  return (
    <Box>
      {label && (
        <Typography variant="caption" display="block" gutterBottom sx={{ fontWeight: 600 }}>
          {label}
        </Typography>
      )}
      <Suspense
        fallback={
          <Box
            sx={{
              height,
              border: 1,
              borderColor: 'divider',
              borderRadius: 1,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              bgcolor: 'action.hover',
            }}
          >
            <CircularProgress size={32} />
          </Box>
        }
      >
        <Box sx={{ height, border: 1, borderColor: 'divider', borderRadius: 1, overflow: 'hidden' }}>
          <MonacoCodeEditor
            value={value}
            language={language}
            readOnly={readOnly}
            onChange={onChange}
          />
        </Box>
      </Suspense>
    </Box>
  );
};

export default SyntaxPropertyEditor;
