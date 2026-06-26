import React, { useEffect, useState } from 'react';
import { Box, LinearProgress, Alert } from '@mui/material';

// Dynamically import Monaco Editor
const loadMonaco = () => import('@monaco-editor/react');

interface CodeEditorProps {
  value: string;
  onChange: (value: string) => void;
  language: 'json' | 'yaml' | 'javascript' | 'typescript' | 'sql';
  height?: string;
  readOnly?: boolean;
  theme?: 'light' | 'dark' | 'vs-dark';
  minimap?: boolean;
  lineNumbers?: boolean;
  fontSize?: number;
  formatOnPaste?: boolean;
  formatOnType?: boolean;
  // optional callback to expose editor and monaco instances to parent for integrations (search, etc.)
  onEditorMount?: (editor: any, monaco: any) => void;
}

const CodeEditor: React.FC<CodeEditorProps> = ({
  value,
  onChange,
  language = 'json',
  height = '300px',
  readOnly = false,
  theme = 'light',
  minimap = false,
  lineNumbers = true,
  fontSize = 14,
  formatOnPaste = true,
  formatOnType = true,
  onEditorMount,
}) => {
  const [Monaco, setMonaco] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let isMounted = true;

    const loadEditor = async () => {
      try {
        const { default: Editor } = await loadMonaco();
        
        if (isMounted) {
          setMonaco(() => Editor);
          setLoading(false);
        }
      } catch (err) {
        if (isMounted) {
          setError('Failed to load code editor');
          setLoading(false);
        }
      }
    };

    loadEditor();

    return () => {
      isMounted = false;
    };
  }, []);

  const editorOptions = {
    readOnly,
    minimap: { enabled: minimap },
    lineNumbers: lineNumbers ? 'on' : 'off',
    fontSize,
    formatOnPaste,
    formatOnType,
    scrollBeyondLastLine: false,
    automaticLayout: true,
    wordWrap: 'on' as const,
    glyphMargin: false,
    folding: true,
    lineDecorationsWidth: 10,
    lineNumbersMinChars: 3,
    renderWhitespace: 'selection' as const,
    tabSize: 2,
    insertSpaces: true,
    bracketPairColorization: { enabled: true },
    guides: {
      bracketPairs: true,
      indentation: true,
    },
  };

  const getMonacoTheme = () => {
    switch (theme) {
      case 'dark':
        return 'dark';
      case 'vs-dark':
        return 'vs-dark';
      default:
        return 'light';
    }
  };

  if (loading) {
    return (
      <Box sx={{ width: '100%', height }}>
        <LinearProgress />
        <Box sx={{ p: 2, textAlign: 'center' }}>Loading editor...</Box>
      </Box>
    );
  }

  if (error) {
    return (
      <Box sx={{ height }}>
        <Alert severity="error">{error}</Alert>
      </Box>
    );
  }

  return (
    <Box sx={{ height, border: 1, borderColor: 'divider', borderRadius: 1 }}>
      <Monaco
        height={height}
        language={language}
        value={value}
        onChange={onChange}
        theme={getMonacoTheme()}
        options={editorOptions}
        onMount={(editor: any, monaco: any) => {
          if (typeof onEditorMount === 'function') {
            try {
              onEditorMount(editor, monaco);
            } catch (e) {
              // swallow
            }
          }
          // Configure JSON schema validation for Cube.js models
          if (language === 'json') {
            monaco.languages.json.jsonDefaults.setDiagnosticsOptions({
              validate: true,
              allowComments: false,
              schemas: [
                {
                  uri: 'http://cube.js/cube-schema.json',
                  fileMatch: ['*'],
                  schema: {
                    type: 'object',
                    properties: {
                      sql: { type: 'string' },
                      measures: {
                        type: 'object',
                        additionalProperties: {
                          type: 'object',
                          properties: {
                            type: { type: 'string', enum: ['count', 'sum', 'avg', 'min', 'max', 'countDistinct'] },
                            sql: { type: 'string' },
                            title: { type: 'string' },
                            description: { type: 'string' },
                          }
                        }
                      },
                      dimensions: {
                        type: 'object',
                        additionalProperties: {
                          type: 'object',
                          properties: {
                            sql: { type: 'string' },
                            type: { type: 'string', enum: ['string', 'number', 'boolean', 'time', 'geo'] },
                            title: { type: 'string' },
                            description: { type: 'string' },
                            primaryKey: { type: 'boolean' },
                          }
                        }
                      },
                      hierarchies: {
                        type: 'object',
                        additionalProperties: {
                          type: 'object',
                          properties: {
                            title: { type: 'string' },
                            levels: {
                              type: 'array',
                              items: { type: 'string' }
                            }
                          }
                        }
                      },
                      drillMembers: {
                        type: 'array',
                        items: { type: 'string' }
                      },
                      joins: {
                        type: 'object',
                        additionalProperties: {
                          type: 'object',
                          properties: {
                            relationship: { type: 'string', enum: ['belongsTo', 'hasMany', 'hasOne'] },
                            sql: { type: 'string' }
                          }
                        }
                      }
                    }
                  }
                }
              ]
            });
          }

          // Configure YAML diagnostics/completion using monaco-yaml when editing YAML.
          if (language === 'yaml') {
            (async () => {
              try {
                let monacoYaml: any = null;
                try {
                  monacoYaml = await import('monaco-yaml');
                } catch (e) {
                  monacoYaml = null;
                }
                if (!monacoYaml) return; // silently skip YAML diagnostics when not available
                // Reuse the same schema shape as JSON; monaco-yaml expects a YAML schema array
                const yamlSchema = {
                  uri: 'http://cube.js/cube-schema.json',
                  fileMatch: ['*'],
                  schema: {
                    type: 'object',
                    properties: {
                      sql: { type: 'string' },
                      measures: {
                        type: 'object',
                        additionalProperties: {
                          type: 'object',
                          properties: {
                            type: { type: 'string', enum: ['count', 'sum', 'avg', 'min', 'max', 'countDistinct'] },
                            sql: { type: 'string' },
                            title: { type: 'string' },
                            description: { type: 'string' },
                          }
                        }
                      },
                      dimensions: {
                        type: 'object',
                        additionalProperties: {
                          type: 'object',
                          properties: {
                            sql: { type: 'string' },
                            type: { type: 'string', enum: ['string', 'number', 'boolean', 'time', 'geo'] },
                            title: { type: 'string' },
                            description: { type: 'string' },
                            primaryKey: { type: 'boolean' },
                          }
                        }
                      },
                      hierarchies: {
                        type: 'object',
                        additionalProperties: {
                          type: 'object',
                          properties: {
                            title: { type: 'string' },
                            levels: {
                              type: 'array',
                              items: { type: 'string' }
                            }
                          }
                        }
                      },
                      drillMembers: {
                        type: 'array',
                        items: { type: 'string' }
                      },
                      joins: {
                        type: 'object',
                        additionalProperties: {
                          type: 'object',
                          properties: {
                            relationship: { type: 'string', enum: ['belongsTo', 'hasMany', 'hasOne'] },
                            sql: { type: 'string' }
                          }
                        }
                      }
                    }
                  }
                };

                monacoYaml.setDiagnosticsOptions(monaco, {
                  enableSchemaRequest: false,
                  hover: true,
                  completion: true,
                  validate: true,
                  schemas: [yamlSchema]
                });
              } catch (e) {
                // ignore if monaco-yaml can't be loaded
              }
            })();
          }

          // Auto-format on mount for JSON
          if (language === 'json' && value) {
            try {
              const parsed = JSON.parse(value);
              const formatted = JSON.stringify(parsed, null, 2);
              if (formatted !== value) {
                onChange(formatted);
              }
            } catch {
              // Invalid JSON, don't format
            }
          }

          // Focus the editor
          editor.focus();
        }}
      />
    </Box>
  );
};

export default CodeEditor;
