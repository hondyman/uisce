import { devWarn } from '../../utils/devLogger';
import { useEffect, FC } from 'react';
import RawCodeModal from '../../components/UnifiedSemanticBuilder/RawCodeModal';
import { CodePanel } from './CodePanel';

import { ShowCode } from './types';
// CodeSearch removed; controls live in workspace header

interface CodeSidebarProps {
  formatType: 'json' | 'yaml';
  setFormatType: (fmt: 'json' | 'yaml') => void;
  rawOpen: boolean;
  setRawOpen: (v: boolean) => void;
  rawFormat: 'json' | 'yaml';
  setRawFormat: (f: 'json' | 'yaml') => void;
  activeEditorTab: 'core' | 'custom';
  setActiveEditorTab: (t: 'core' | 'custom') => void;
  generateCoreJSON: () => string;
  generateCoreYAML: () => string;
  generateCustomJSON: () => string;
  generateCustomYAML: () => string;
  generateMergedModelObject: () => any;
  generateJSON: () => string;
  generateYAML: () => string;
  modelName: string;
  selectedModel: any;
}

const CodeSidebar: FC<CodeSidebarProps> = ({
  formatType,
  setFormatType: _setFormatType,
  rawOpen,
  setRawOpen,
  rawFormat,
  setRawFormat,
  activeEditorTab,
  setActiveEditorTab: _setActiveEditorTab,
  generateCoreJSON,
  generateCoreYAML,
  generateCustomJSON,
  generateCustomYAML,
  generateMergedModelObject: _generateMergedModelObject,
  generateJSON: _generateJSON,
  generateYAML: _generateYAML,
  modelName,
  selectedModel,
  
}) => {
  // match navigation handled by workspace header search
  // Global search handled in workspace header

  const getCurrentCode = () => {
    if (formatType === 'json') {
      switch (activeEditorTab) {
        case 'core': return generateCoreJSON();
        case 'custom': return generateCustomJSON();
        default: return generateCustomJSON();
      }
    } else { // yaml
      switch (activeEditorTab) {
        case 'core': return generateCoreYAML();
        case 'custom': return generateCustomYAML();
        default: return generateCustomYAML();
      }
    }
  };

  const handleCopy = () => {
  navigator.clipboard.writeText(getCurrentCode()).catch(err => devWarn("Copy failed", err));
  };

  const handleDownload = () => {
    const code = getCurrentCode();
    const blob = new Blob([code], { type: `application/${formatType}` });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${modelName}-${activeEditorTab}.${formatType}`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  // Listen for global copy/download events triggered from workspace header
  useEffect(() => {
    const copyHandler = () => handleCopy();
    const downloadHandler = () => handleDownload();
    window.addEventListener('semlayer.copyCode', copyHandler);
    window.addEventListener('semlayer.downloadCode', downloadHandler);
    return () => {
      window.removeEventListener('semlayer.copyCode', copyHandler);
      window.removeEventListener('semlayer.downloadCode', downloadHandler);
    };
  }, [formatType, activeEditorTab, selectedModel, modelName]);

  return (
    <>
      <RawCodeModal
        open={rawOpen}
        onClose={() => setRawOpen(false)}
        format={rawFormat}
        onFormatChange={(f) => setRawFormat(f)}
        generateJSON={() => {
          switch (activeEditorTab) {
            case 'core': return generateCoreJSON();
            case 'custom': return generateCustomJSON();
            default: return generateCustomJSON();
          }
        }}
        generateYAML={() => {
          switch (activeEditorTab) {
            case 'core': return generateCoreYAML();
            case 'custom': return generateCustomYAML();
            default: return generateCustomYAML();
          }
        }}
        title={`Raw ${rawFormat.toUpperCase()} Output`}
      />

      <CodePanel
        key={`${selectedModel ? (selectedModel.id || selectedModel.model_key) : modelName}-${activeEditorTab}-${formatType}`}
        showCode={formatType as ShowCode}
        modelName={modelName}
        generateJSON={() => {
          switch (activeEditorTab) {
            case 'core':
              return generateCoreJSON();
            case 'custom':
              return generateCustomJSON();
            default:
              return generateCustomJSON();
          }
        }}
        generateYAML={() => {
          switch (activeEditorTab) {
            case 'core':
              return generateCoreYAML();
            case 'custom':
              return generateCustomYAML();
            default:
              return generateCustomYAML();
          }
        }}
      />
    </>
  );
};

export default CodeSidebar;
