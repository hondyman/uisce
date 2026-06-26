import type { FC } from 'react';
import MonacoCodeEditor from './UnifiedSemanticBuilder/MonacoCodeEditor.lazy';
import './SqlMonacoEditor.css';

interface SqlMonacoEditorProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  height?: string | number;
  readOnly?: boolean;
  className?: string;
}


// Retained filename / component name to avoid broad refactor; implementation now Monaco-based.
const SqlMonacoEditor: FC<SqlMonacoEditorProps> = ({
  value,
  onChange,
  placeholder: _placeholder = 'Enter SQL expression...',
  height = 120,
  readOnly = false,
  className = '',
}) => {
  return (
    <div className={`sql-monaco-editor ${className}`}>
  <div className={`editor-wrapper-full ${typeof height === 'number' ? '' : ''}`}> {/* external container controls height */}
  {/* Use json as generic language; could be extended to sql with a dedicated provider */}
  <MonacoCodeEditor value={value} language="sql" readOnly={readOnly} onChange={(val: string) => onChange(val)} />
      </div>
    </div>
  );
};

export default SqlMonacoEditor;
