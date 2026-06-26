import React, { Suspense } from 'react';
import './MonacoCodeEditor.css';

const MonacoImpl = React.lazy(() => import('./MonacoCodeEditor.impl'));

const MonacoCodeEditorLazy: React.FC<any> = (props) => (
  <Suspense fallback={<div className="monaco-loading">Loading editor...</div>}>
    <MonacoImpl {...props} />
  </Suspense>
);

export default MonacoCodeEditorLazy;
