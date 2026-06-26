import React, { useState } from 'react';
import Editor from '@monaco-editor/react';
import { Box, Typography,  ToggleButton, ToggleButtonGroup, Paper } from '@mui/material';
import AutoAwesomeIcon from '@mui/icons-material/AutoAwesome';
import VerticalSplitIcon from '@mui/icons-material/VerticalSplit';

interface ValidationRuleScriptEditorProps {
  value: string;
  onChange: (value: string | undefined) => void;
  height?: string;
  readOnly?: boolean;
  language?: string;
  theme?: string;
  schemaContext?: string; // Generated CUE schema for reference/IntelliSense
}

export const ValidationRuleScriptEditor: React.FC<ValidationRuleScriptEditorProps> = ({
  value,
  onChange,
  height = "400px",
  readOnly = false,
  language = "python",
  theme = "vs-dark",
  schemaContext
}) => {
  const [showSchema, setShowSchema] = useState(false);

  const handleEditorDidMount = (editor: any, monaco: any) => {
    // Future: Configure language server capabilities using schemaContext
  };

  return (
    <Box>
      {schemaContext && (
        <Box sx={{ display: 'flex', justifyContent: 'flex-end', mb: 1 }}>
          <ToggleButtonGroup 
            size="small" 
            value={showSchema ? 'split' : 'editor'}
            exclusive
            onChange={(_, val) => val && setShowSchema(val === 'split')}
            sx={{ bgcolor: 'background.paper' }}
          >
            <ToggleButton value="editor">
              Editor
            </ToggleButton>
            <ToggleButton value="split">
              <VerticalSplitIcon sx={{ mr: 1, fontSize: 16 }} />
              Split View (Schema)
            </ToggleButton>
          </ToggleButtonGroup>
        </Box>
      )}
      
      <div className="monaco-editor-container" style={{ borderRadius: '8px', overflow: 'hidden', border: '1px solid #444', display: 'flex', height }}>
        <Box sx={{ flex: 1, height: '100%' }}>
            <Editor
                height="100%"
                defaultLanguage="python"
                language={language === 'cue' ? 'go' : language}
                value={value}
                onChange={onChange}
                theme={theme}
                options={{
                minimap: { enabled: false },
                fontSize: 14,
                lineNumbers: 'on',
                scrollBeyondLastLine: false,
                readOnly: readOnly,
                automaticLayout: true,
                fontFamily: "'Fira Code', 'Roboto Mono', monospace",
                renderWhitespace: 'selection',
                }}
                onMount={handleEditorDidMount}
            />
        </Box>
        
        {showSchema && schemaContext && (
            <Box sx={{ flex: 1, height: '100%', borderLeft: '1px solid #444', bgcolor: '#1e1e1e' }}>
                <Box sx={{ px: 2, py: 0.5, bg: '#252526', borderBottom: '1px solid #444' }}>
                    <Typography variant="caption" sx={{ color: '#aaa', display: 'flex', alignItems: 'center', gap: 1 }}>
                        <AutoAwesomeIcon sx={{ fontSize: 14 }} /> Generated Schema (Read-Only)
                    </Typography>
                </Box>
                <Editor
                    height="100%"
                    defaultLanguage="go" 
                    value={schemaContext}
                    theme={theme}
                    options={{
                        minimap: { enabled: false },
                        fontSize: 12, // Slightly smaller for reference
                        lineNumbers: 'off',
                        readOnly: true,
                        domReadOnly: true,
                        automaticLayout: true,
                        fontFamily: "'Fira Code', 'Roboto Mono', monospace",
                        renderWhitespace: 'none',
                    }}
                />
            </Box>
        )}
      </div>
    </Box>
  );
};

export default ValidationRuleScriptEditor;
