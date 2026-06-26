// React import not required with the new JSX runtime; keep file minimal
import MonacoCodeEditor from '../../../components/UnifiedSemanticBuilder/MonacoCodeEditor.lazy';

interface JsonTabProps {
  config: any;
}

export default function JsonTab({ config }: JsonTabProps) {
  const jsonString = JSON.stringify(config, null, 2);

  return (
  <div className="editor-wrapper-full editor-h-400">
    <MonacoCodeEditor
        value={jsonString}
        language="json"
        readOnly
        onChange={() => {}}
      />
    </div>
  );
}