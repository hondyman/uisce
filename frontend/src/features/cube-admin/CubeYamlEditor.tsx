import React, { useRef, useState, useCallback } from 'react';
import Editor, { DiffEditor, Monaco, OnMount } from '@monaco-editor/react';
import {
  Box,
  Paper,
  Typography,
  IconButton,
  Tooltip,
  Button,
  Chip,
  Alert,
  Tabs,
  Tab,
  CircularProgress,
  Snackbar,
} from '@mui/material';
import {
  ContentCopy as CopyIcon,
  Download as DownloadIcon,
  Check as CheckIcon,
  Error as ErrorIcon,
  Code as CodeIcon,
  Save as SaveIcon,
} from '@mui/icons-material';
import * as monaco from 'monaco-editor';

// Cube YAML Schema for IntelliSense
const CUBE_YAML_SCHEMA = {
  cubes: {
    properties: {
      name: { type: 'string', description: 'Unique cube identifier (PascalCase)' },
      title: { type: 'string', description: 'Human-readable title' },
      description: { type: 'string', description: 'Documentation for this cube' },
      sql_table: { type: 'string', description: 'SQL table reference: schema.table' },
      sql: { type: 'string', description: 'Custom SQL query for the cube' },
      data_source: { type: 'string', description: 'Database connection name' },
      public: { type: 'boolean', description: 'Whether cube is exposed via API' },
      refresh_key: { type: 'object', description: 'Pre-aggregation refresh configuration' },
      extends: { type: 'string', description: 'Parent cube to extend' },
    },
  },
  measures: {
    properties: {
      name: { type: 'string', description: 'Measure identifier (snake_case)' },
      sql: { type: 'string', description: 'SQL expression for measure' },
      type: {
        type: 'string',
        enum: ['count', 'count_distinct', 'count_distinct_approx', 'sum', 'avg', 'min', 'max', 'number', 'string', 'time', 'boolean', 'running_total'],
        description: 'Aggregation type',
      },
      title: { type: 'string', description: 'Display title' },
      description: { type: 'string', description: 'Documentation' },
      format: { type: 'string', enum: ['number', 'currency', 'percent'], description: 'Display format' },
      shown: { type: 'boolean', description: 'Visibility in API' },
      drill_members: { type: 'array', description: 'Dimensions for drill-down' },
      filters: { type: 'array', description: 'Measure-level filters' },
    },
  },
  dimensions: {
    properties: {
      name: { type: 'string', description: 'Dimension identifier (snake_case)' },
      sql: { type: 'string', description: 'SQL expression for dimension' },
      type: {
        type: 'string',
        enum: ['string', 'number', 'boolean', 'time', 'geo'],
        description: 'Data type',
      },
      title: { type: 'string', description: 'Display title' },
      description: { type: 'string', description: 'Documentation' },
      primary_key: { type: 'boolean', description: 'Mark as primary key' },
      shown: { type: 'boolean', description: 'Visibility in API' },
      sub_query: { type: 'boolean', description: 'Use as subquery dimension' },
      propagate_filters_to_sub_query: { type: 'boolean', description: 'Propagate filters' },
    },
  },
  joins: {
    properties: {
      name: { type: 'string', description: 'Join identifier' },
      sql: { type: 'string', description: 'Join condition SQL' },
      relationship: {
        type: 'string',
        enum: ['one_to_one', 'one_to_many', 'many_to_one'],
        description: 'Relationship type',
      },
    },
  },
  pre_aggregations: {
    properties: {
      name: { type: 'string', description: 'Pre-aggregation identifier' },
      type: {
        type: 'string',
        enum: ['rollup', 'rollupLambda', 'rollupJoin', 'originalSql', 'autoRollup'],
        description: 'Pre-aggregation type',
      },
      measures: { type: 'array', description: 'Measures to include' },
      dimensions: { type: 'array', description: 'Dimensions to include' },
      time_dimension: { type: 'string', description: 'Time dimension reference' },
      granularity: {
        type: 'string',
        enum: ['second', 'minute', 'hour', 'day', 'week', 'month', 'quarter', 'year'],
        description: 'Time granularity',
      },
      partition_granularity: { type: 'string', description: 'Partition granularity' },
      refresh_key: { type: 'object', description: 'Refresh configuration' },
      external: { type: 'boolean', description: 'Store in external database' },
      scheduled_refresh: { type: 'boolean', description: 'Enable scheduled refresh' },
      build_range_start: { type: 'object', description: 'Build range start' },
      build_range_end: { type: 'object', description: 'Build range end' },
      indexes: { type: 'object', description: 'Index definitions' },
    },
  },
};

// Keywords for syntax highlighting and completion
const CUBE_KEYWORDS = [
  'cubes', 'name', 'title', 'description', 'sql', 'sql_table', 'data_source', 'public',
  'measures', 'dimensions', 'joins', 'pre_aggregations', 'segments',
  'type', 'format', 'shown', 'drill_members', 'filters', 'primary_key',
  'relationship', 'time_dimension', 'granularity', 'partition_granularity',
  'refresh_key', 'every', 'sql', 'external', 'scheduled_refresh',
  'build_range_start', 'build_range_end', 'indexes', 'extends', 'sub_query',
];

const MEASURE_TYPES = ['count', 'count_distinct', 'count_distinct_approx', 'sum', 'avg', 'min', 'max', 'number', 'string', 'time', 'boolean', 'running_total'];
const DIMENSION_TYPES = ['string', 'number', 'boolean', 'time', 'geo'];
const RELATIONSHIP_TYPES = ['one_to_one', 'one_to_many', 'many_to_one'];
const GRANULARITY_TYPES = ['second', 'minute', 'hour', 'day', 'week', 'month', 'quarter', 'year'];
const PREAGG_TYPES = ['rollup', 'rollupLambda', 'rollupJoin', 'originalSql', 'autoRollup'];

interface CubeYamlEditorProps {
  value: string;
  onChange?: (value: string) => void;
  onSave?: (value: string) => void;
  onValidate?: (errors: ValidationError[]) => void;
  readOnly?: boolean;
  height?: number | string;
  showDiff?: boolean;
  originalValue?: string;
  catalogTables?: string[];
  catalogColumns?: Record<string, string[]>;
  existingCubes?: string[];
}

interface ValidationError {
  line: number;
  column: number;
  message: string;
  severity: 'error' | 'warning' | 'info';
}

export const CubeYamlEditor: React.FC<CubeYamlEditorProps> = ({
  value,
  onChange,
  onSave,
  onValidate,
  readOnly = false,
  height = 500,
  showDiff = false,
  originalValue,
  catalogTables = [],
  catalogColumns: _catalogColumns = {},
  existingCubes = [],
}) => {
  const editorRef = useRef<monaco.editor.IStandaloneCodeEditor | null>(null);
  const monacoRef = useRef<Monaco | null>(null);
  const [errors, setErrors] = useState<ValidationError[]>([]);
  const [activeTab, setActiveTab] = useState(0);
  const [copied, setCopied] = useState(false);
  const [saving, setSaving] = useState(false);

  // Configure Monaco for Cube YAML
  const configureMonaco = useCallback((monaco: Monaco) => {
    // Register YAML language configuration
    monaco.languages.setLanguageConfiguration('yaml', {
      comments: { lineComment: '#' },
      brackets: [['{', '}'], ['[', ']']],
      autoClosingPairs: [
        { open: '{', close: '}' },
        { open: '[', close: ']' },
        { open: '"', close: '"' },
        { open: "'", close: "'" },
      ],
      indentationRules: {
        increaseIndentPattern: /^.*:\s*$/,
        decreaseIndentPattern: /^\s*$/,
      },
    });

    // Register completion provider
    monaco.languages.registerCompletionItemProvider('yaml', {
      provideCompletionItems: (model, position) => {
        const word = model.getWordUntilPosition(position);
        const range = {
          startLineNumber: position.lineNumber,
          endLineNumber: position.lineNumber,
          startColumn: word.startColumn,
          endColumn: word.endColumn,
        };

        const lineContent = model.getLineContent(position.lineNumber);
        const textUntilPosition = model.getValueInRange({
          startLineNumber: 1,
          startColumn: 1,
          endLineNumber: position.lineNumber,
          endColumn: position.column,
        });

        const suggestions: monaco.languages.CompletionItem[] = [];

        // Context-aware suggestions
        const context = detectContext(textUntilPosition, lineContent);

        if (context === 'root') {
          suggestions.push(
            ...['cubes', 'views'].map(keyword => ({
              label: keyword,
              kind: monaco.languages.CompletionItemKind.Keyword,
              insertText: `${keyword}:\n  - name: `,
              range,
              documentation: `Define ${keyword}`,
            }))
          );
        }

        if (context === 'cube') {
          suggestions.push(
            ...Object.keys(CUBE_YAML_SCHEMA.cubes.properties).map(prop => ({
              label: prop,
              kind: monaco.languages.CompletionItemKind.Property,
              insertText: `${prop}: `,
              range,
              documentation: (CUBE_YAML_SCHEMA.cubes.properties as any)[prop].description,
            }))
          );
          suggestions.push(
            { label: 'measures', kind: monaco.languages.CompletionItemKind.Property, insertText: 'measures:\n    - ', range, documentation: 'Define cube measures' },
            { label: 'dimensions', kind: monaco.languages.CompletionItemKind.Property, insertText: 'dimensions:\n    - ', range, documentation: 'Define cube dimensions' },
            { label: 'joins', kind: monaco.languages.CompletionItemKind.Property, insertText: 'joins:\n    - ', range, documentation: 'Define cube joins' },
            { label: 'pre_aggregations', kind: monaco.languages.CompletionItemKind.Property, insertText: 'pre_aggregations:\n    - ', range, documentation: 'Define pre-aggregations' },
          );
        }

        if (context === 'measure') {
          suggestions.push(
            ...Object.keys(CUBE_YAML_SCHEMA.measures.properties).map(prop => ({
              label: prop,
              kind: monaco.languages.CompletionItemKind.Property,
              insertText: `${prop}: `,
              range,
              documentation: (CUBE_YAML_SCHEMA.measures.properties as any)[prop].description,
            }))
          );
        }

        if (context === 'measure_type') {
          suggestions.push(
            ...MEASURE_TYPES.map(type => ({
              label: type,
              kind: monaco.languages.CompletionItemKind.EnumMember,
              insertText: type,
              range,
              documentation: `${type} aggregation`,
            }))
          );
        }

        if (context === 'dimension') {
          suggestions.push(
            ...Object.keys(CUBE_YAML_SCHEMA.dimensions.properties).map(prop => ({
              label: prop,
              kind: monaco.languages.CompletionItemKind.Property,
              insertText: `${prop}: `,
              range,
              documentation: (CUBE_YAML_SCHEMA.dimensions.properties as any)[prop].description,
            }))
          );
        }

        if (context === 'dimension_type') {
          suggestions.push(
            ...DIMENSION_TYPES.map(type => ({
              label: type,
              kind: monaco.languages.CompletionItemKind.EnumMember,
              insertText: type,
              range,
              documentation: `${type} dimension type`,
            }))
          );
        }

        if (context === 'join') {
          suggestions.push(
            ...Object.keys(CUBE_YAML_SCHEMA.joins.properties).map(prop => ({
              label: prop,
              kind: monaco.languages.CompletionItemKind.Property,
              insertText: `${prop}: `,
              range,
              documentation: (CUBE_YAML_SCHEMA.joins.properties as any)[prop].description,
            }))
          );
        }

        if (context === 'relationship_type') {
          suggestions.push(
            ...RELATIONSHIP_TYPES.map(type => ({
              label: type,
              kind: monaco.languages.CompletionItemKind.EnumMember,
              insertText: type,
              range,
              documentation: `${type} relationship`,
            }))
          );
        }

        if (context === 'pre_aggregation') {
          suggestions.push(
            ...Object.keys(CUBE_YAML_SCHEMA.pre_aggregations.properties).map(prop => ({
              label: prop,
              kind: monaco.languages.CompletionItemKind.Property,
              insertText: `${prop}: `,
              range,
              documentation: (CUBE_YAML_SCHEMA.pre_aggregations.properties as any)[prop].description,
            }))
          );
        }

        if (context === 'preagg_type') {
          suggestions.push(
            ...PREAGG_TYPES.map(type => ({
              label: type,
              kind: monaco.languages.CompletionItemKind.EnumMember,
              insertText: type,
              range,
              documentation: `${type} pre-aggregation type`,
            }))
          );
        }

        if (context === 'granularity') {
          suggestions.push(
            ...GRANULARITY_TYPES.map(type => ({
              label: type,
              kind: monaco.languages.CompletionItemKind.EnumMember,
              insertText: type,
              range,
              documentation: `${type} granularity`,
            }))
          );
        }

        if (context === 'sql_table' && catalogTables.length > 0) {
          suggestions.push(
            ...catalogTables.map(table => ({
              label: table,
              kind: monaco.languages.CompletionItemKind.Value,
              insertText: table,
              range,
              documentation: `Table: ${table}`,
            }))
          );
        }

        if (context === 'cube_reference' && existingCubes.length > 0) {
          suggestions.push(
            ...existingCubes.map(cube => ({
              label: cube,
              kind: monaco.languages.CompletionItemKind.Reference,
              insertText: cube,
              range,
              documentation: `Reference to cube: ${cube}`,
            }))
          );
        }

        // SQL snippets
        if (context === 'sql_expression') {
          suggestions.push(
            { label: '{CUBE}', kind: monaco.languages.CompletionItemKind.Snippet, insertText: '{CUBE}.', range, documentation: 'Reference current cube' },
            { label: 'CASE', kind: monaco.languages.CompletionItemKind.Snippet, insertText: 'CASE WHEN ${1:condition} THEN ${2:result} ELSE ${3:default} END', insertTextRules: monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet, range, documentation: 'SQL CASE expression' },
            { label: 'COALESCE', kind: monaco.languages.CompletionItemKind.Snippet, insertText: 'COALESCE(${1:value}, ${2:default})', insertTextRules: monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet, range, documentation: 'SQL COALESCE function' },
            { label: 'CONCAT', kind: monaco.languages.CompletionItemKind.Snippet, insertText: "CONCAT(${1:string1}, ${2:string2})", insertTextRules: monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet, range, documentation: 'SQL CONCAT function' },
          );
        }

        return { suggestions };
      },
    });

    // Register hover provider
    monaco.languages.registerHoverProvider('yaml', {
      provideHover: (model, position) => {
        const word = model.getWordAtPosition(position);
        if (!word) return null;

        const text = word.word;

        // Find documentation for the word
        let doc = '';
        
        if (CUBE_KEYWORDS.includes(text)) {
          for (const section of Object.values(CUBE_YAML_SCHEMA)) {
            if ((section.properties as any)[text]) {
              doc = (section.properties as any)[text].description;
              break;
            }
          }
        }

        if (MEASURE_TYPES.includes(text)) {
          doc = `Measure type: ${text}`;
        }
        if (DIMENSION_TYPES.includes(text)) {
          doc = `Dimension type: ${text}`;
        }
        if (GRANULARITY_TYPES.includes(text)) {
          doc = `Time granularity: ${text}`;
        }

        if (doc) {
          return {
            contents: [{ value: doc }],
            range: {
              startLineNumber: position.lineNumber,
              endLineNumber: position.lineNumber,
              startColumn: word.startColumn,
              endColumn: word.endColumn,
            },
          };
        }

        return null;
      },
    });

    // Register code actions for quick fixes
    monaco.languages.registerCodeActionProvider('yaml', {
      provideCodeActions: (model, range, context) => {
        const actions: monaco.languages.CodeAction[] = [];

        for (const marker of context.markers) {
          if (marker.message.includes('unknown property')) {
            // Suggest similar property names
            actions.push({
              title: `Remove unknown property`,
              kind: 'quickfix',
              edit: {
                edits: [{
                  resource: model.uri,
                  textEdit: {
                    range: {
                      startLineNumber: marker.startLineNumber,
                      startColumn: 1,
                      endLineNumber: marker.endLineNumber + 1,
                      endColumn: 1,
                    },
                    text: '',
                  },
                  versionId: model.getVersionId(),
                }],
              },
            });
          }
        }

        return { actions, dispose: () => {} };
      },
    });

    monacoRef.current = monaco;
  }, [catalogTables, existingCubes]);

  // Detect editing context for smart completions
  const detectContext = (textUntilPosition: string, currentLine: string): string => {
    const lines = textUntilPosition.split('\n');
    const trimmedLine = currentLine.trim();

    // Check for type: field
    if (trimmedLine.startsWith('type:')) {
      // Determine which type based on context
      const contextLines = textUntilPosition.split('\n').slice(-10).join('\n');
      if (contextLines.includes('measures:')) return 'measure_type';
      if (contextLines.includes('dimensions:')) return 'dimension_type';
      if (contextLines.includes('pre_aggregations:')) return 'preagg_type';
    }

    if (trimmedLine.startsWith('relationship:')) return 'relationship_type';
    if (trimmedLine.startsWith('granularity:') || trimmedLine.startsWith('partition_granularity:')) return 'granularity';
    if (trimmedLine.startsWith('sql_table:')) return 'sql_table';
    if (trimmedLine.startsWith('sql:')) return 'sql_expression';
    if (trimmedLine.startsWith('extends:') || trimmedLine.startsWith('name:')) return 'cube_reference';

    // Check indentation level and context
    let inMeasures = false;
    let inDimensions = false;
    let inJoins = false;
    let inPreAggs = false;
    let inCube = false;

    for (let i = lines.length - 1; i >= 0; i--) {
      const line = lines[i].trim();
      if (line.startsWith('measures:')) { inMeasures = true; break; }
      if (line.startsWith('dimensions:')) { inDimensions = true; break; }
      if (line.startsWith('joins:')) { inJoins = true; break; }
      if (line.startsWith('pre_aggregations:')) { inPreAggs = true; break; }
      if (line.startsWith('cubes:')) { inCube = true; break; }
      if (line.startsWith('- name:')) {
        // Determine from previous section markers
        for (let j = i - 1; j >= 0; j--) {
          const prevLine = lines[j].trim();
          if (prevLine.startsWith('measures:')) { inMeasures = true; break; }
          if (prevLine.startsWith('dimensions:')) { inDimensions = true; break; }
          if (prevLine.startsWith('joins:')) { inJoins = true; break; }
          if (prevLine.startsWith('pre_aggregations:')) { inPreAggs = true; break; }
          if (prevLine.startsWith('cubes:') || prevLine.startsWith('- name:')) { inCube = true; break; }
        }
        break;
      }
    }

    if (inMeasures) return 'measure';
    if (inDimensions) return 'dimension';
    if (inJoins) return 'join';
    if (inPreAggs) return 'pre_aggregation';
    if (inCube) return 'cube';

    return 'root';
  };

  // Validate YAML content
  const validateYaml = useCallback((content: string) => {
    const validationErrors: ValidationError[] = [];
    const lines = content.split('\n');

    // Basic YAML structure validation
    let hasRoot = false;
    // const indentStack: number[] = [0]; // Reserved for future indent validation

    lines.forEach((line, index) => {
      const lineNum = index + 1;
      const trimmed = line.trim();

      if (!trimmed || trimmed.startsWith('#')) return;

      // Check for root element
      if (trimmed.startsWith('cubes:') || trimmed.startsWith('views:')) {
        hasRoot = true;
      }

      // Check indentation
      const indent = line.search(/\S/);
      if (indent === -1) return;

      // Validate required properties
      if (trimmed.startsWith('- name:') && !trimmed.includes(':')) {
        validationErrors.push({
          line: lineNum,
          column: 1,
          message: 'Name property requires a value',
          severity: 'error',
        });
      }

      // Validate measure types
      if (trimmed.startsWith('type:')) {
        const value = trimmed.split(':')[1]?.trim();
        if (value && ![...MEASURE_TYPES, ...DIMENSION_TYPES, ...PREAGG_TYPES].includes(value)) {
          validationErrors.push({
            line: lineNum,
            column: trimmed.indexOf(value) + 1,
            message: `Unknown type: ${value}`,
            severity: 'warning',
          });
        }
      }

      // Validate SQL references
      if (trimmed.includes('{CUBE}') && !trimmed.includes('{CUBE}.')) {
        validationErrors.push({
          line: lineNum,
          column: trimmed.indexOf('{CUBE}') + 1,
          message: '{CUBE} should be followed by a column reference like {CUBE}.column_name',
          severity: 'warning',
        });
      }
    });

    if (!hasRoot && content.trim().length > 0) {
      validationErrors.push({
        line: 1,
        column: 1,
        message: 'YAML must start with cubes: or views:',
        severity: 'error',
      });
    }

    setErrors(validationErrors);
    onValidate?.(validationErrors);

    // Set markers in editor
    if (editorRef.current && monacoRef.current) {
      const model = editorRef.current.getModel();
      if (model) {
        monacoRef.current.editor.setModelMarkers(model, 'cube-validator', 
          validationErrors.map(err => ({
            startLineNumber: err.line,
            startColumn: err.column,
            endLineNumber: err.line,
            endColumn: err.column + 10,
            message: err.message,
            severity: err.severity === 'error' 
              ? monacoRef.current!.MarkerSeverity.Error 
              : err.severity === 'warning'
              ? monacoRef.current!.MarkerSeverity.Warning
              : monacoRef.current!.MarkerSeverity.Info,
          }))
        );
      }
    }
  }, [onValidate]);

  // Editor mount handler
  const handleEditorDidMount: OnMount = (editor, monaco) => {
    editorRef.current = editor;
    configureMonaco(monaco);

    // Initial validation
    validateYaml(value);

    // Add keyboard shortcuts
    editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS, () => {
      handleSave();
    });

    editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyD, () => {
      // Duplicate line
      const selection = editor.getSelection();
      if (selection) {
        const lineContent = editor.getModel()?.getLineContent(selection.startLineNumber);
        editor.executeEdits('duplicate-line', [{
          range: {
            startLineNumber: selection.startLineNumber,
            startColumn: 1,
            endLineNumber: selection.startLineNumber,
            endColumn: 1,
          },
          text: lineContent + '\n',
        }]);
      }
    });
  };

  // Handle content change
  const handleChange = (newValue: string | undefined) => {
    const content = newValue || '';
    onChange?.(content);
    
    // Debounce validation
    const timeout = setTimeout(() => validateYaml(content), 500);
    return () => clearTimeout(timeout);
  };

  // Handle save
  const handleSave = async () => {
    if (errors.some(e => e.severity === 'error')) {
      return;
    }
    setSaving(true);
    try {
      await onSave?.(value);
    } finally {
      setSaving(false);
    }
  };

  // Copy to clipboard
  const handleCopy = () => {
    navigator.clipboard.writeText(value);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  // Download YAML
  const handleDownload = () => {
    const blob = new Blob([value], { type: 'application/x-yaml' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'cube-model.yaml';
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <Paper sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      {/* Toolbar */}
      <Box sx={{ p: 1, display: 'flex', justifyContent: 'space-between', alignItems: 'center', borderBottom: 1, borderColor: 'divider' }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <CodeIcon color="primary" />
          <Typography variant="subtitle2">Cube YAML Editor</Typography>
          {errors.length > 0 && (
            <Chip
              size="small"
              icon={errors.some(e => e.severity === 'error') ? <ErrorIcon /> : <CheckIcon />}
              label={`${errors.length} ${errors.length === 1 ? 'issue' : 'issues'}`}
              color={errors.some(e => e.severity === 'error') ? 'error' : 'warning'}
            />
          )}
        </Box>
        <Box>
          {showDiff && originalValue && (
            <Tabs value={activeTab} onChange={(_, v) => setActiveTab(v)} sx={{ minHeight: 32 }}>
              <Tab label="Editor" sx={{ minHeight: 32, py: 0 }} />
              <Tab label="Diff" sx={{ minHeight: 32, py: 0 }} />
            </Tabs>
          )}
          <Tooltip title="Copy">
            <IconButton size="small" onClick={handleCopy}>
              {copied ? <CheckIcon color="success" /> : <CopyIcon />}
            </IconButton>
          </Tooltip>
          <Tooltip title="Download">
            <IconButton size="small" onClick={handleDownload}>
              <DownloadIcon />
            </IconButton>
          </Tooltip>
          {onSave && (
            <Button
              size="small"
              variant="contained"
              startIcon={saving ? <CircularProgress size={16} /> : <SaveIcon />}
              onClick={handleSave}
              disabled={saving || errors.some(e => e.severity === 'error')}
              sx={{ ml: 1 }}
            >
              Save
            </Button>
          )}
        </Box>
      </Box>

      {/* Editor */}
      <Box sx={{ flex: 1, position: 'relative' }}>
        {activeTab === 0 && (
          <Editor
            height={height}
            defaultLanguage="yaml"
            value={value}
            onChange={handleChange}
            onMount={handleEditorDidMount}
            options={{
              readOnly,
              minimap: { enabled: false },
              fontSize: 13,
              lineNumbers: 'on',
              folding: true,
              wordWrap: 'on',
              automaticLayout: true,
              scrollBeyondLastLine: false,
              tabSize: 2,
              insertSpaces: true,
              renderWhitespace: 'selection',
              quickSuggestions: true,
              suggestOnTriggerCharacters: true,
              acceptSuggestionOnEnter: 'on',
              formatOnPaste: true,
              formatOnType: true,
            }}
            theme="vs-dark"
          />
        )}
        {activeTab === 1 && showDiff && originalValue && (
          <DiffEditor
            height={height}
            language="yaml"
            modified={value}
            original={originalValue}
            options={{
              readOnly: true,
              renderSideBySide: true,
              minimap: { enabled: false },
            }}
            theme="vs-dark"
          />
        )}
      </Box>

      {/* Errors panel */}
      {errors.length > 0 && (
        <Box sx={{ maxHeight: 150, overflow: 'auto', borderTop: 1, borderColor: 'divider' }}>
          {errors.map((error, index) => (
            <Alert
              key={index}
              severity={error.severity}
              sx={{ py: 0, borderRadius: 0 }}
              onClick={() => {
                editorRef.current?.revealLineInCenter(error.line);
                editorRef.current?.setPosition({ lineNumber: error.line, column: error.column });
                editorRef.current?.focus();
              }}
            >
              <Typography variant="caption">
                Line {error.line}: {error.message}
              </Typography>
            </Alert>
          ))}
        </Box>
      )}

      <Snackbar
        open={copied}
        autoHideDuration={2000}
        message="YAML copied to clipboard"
        anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
      />
    </Paper>
  );
};

export default CubeYamlEditor;
