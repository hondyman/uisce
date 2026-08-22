import React from 'react';

export interface ValidationSummaryItem {
  id: string;
  label: string;
  detail: string;
  status: 'PASS' | 'WARN' | 'FAIL';
  isBlocking?: boolean;
}

export interface ValidationPublishRailProps {
  status: 'DRAFT' | 'PUBLISHABLE' | 'PUBLISHED' | 'CUSTOM_OVERRIDE';
  summaryItems: ValidationSummaryItem[];
  onSaveDraft?: () => void;
  onRunValidation?: () => void;
  onPublish?: () => void;
  isPublishing?: boolean;
}

export const ValidationPublishRail: React.FC<ValidationPublishRailProps> = ({
  status,
  summaryItems,
  onSaveDraft,
  onRunValidation,
  onPublish,
  isPublishing = false,
}) => {
  const hasBlockingFailures = summaryItems.some((item) => item.status === 'FAIL' && item.isBlocking);
  const warningCount = summaryItems.filter((item) => item.status === 'WARN').length;

  return (
    <div className="fixed bottom-0 left-0 right-0 z-40 bg-slate-950/95 border-t border-slate-800 p-4 backdrop-blur-lg shadow-2xl">
      <div className="max-w-7xl mx-auto flex flex-col md:flex-row items-center justify-between gap-4">
        {/* Validation Summary List */}
        <div className="flex items-center gap-4 flex-wrap text-xs font-mono">
          <span className="font-semibold text-slate-300 tracking-wider">VALIDATION SUMMARY:</span>
          {summaryItems.map((item, idx) => (
            <div key={item.id || `${item.label}-${idx}`} className="flex items-center gap-1.5 bg-slate-900/80 px-2.5 py-1 rounded border border-slate-800">
              <span
                className={
                  item.status === 'PASS'
                    ? 'text-emerald-400 font-bold'
                    : item.status === 'WARN'
                    ? 'text-amber-400 font-bold'
                    : 'text-red-400 font-bold'
                }
              >
                {item.status === 'PASS' ? '✓' : item.status === 'WARN' ? '⚠️' : '✗'}
              </span>
              <span className="text-slate-300 font-medium">{item.label}:</span>
              <span className="text-slate-400">{item.detail}</span>
            </div>
          ))}
        </div>

        {/* Action Controls */}
        <div className="flex items-center gap-3">
          {onSaveDraft && (
            <button
              onClick={onSaveDraft}
              className="px-4 py-2 rounded-lg text-xs font-medium bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700 transition-colors"
            >
              Save Draft
            </button>
          )}

          {onRunValidation && (
            <button
              onClick={onRunValidation}
              className="px-4 py-2 rounded-lg text-xs font-medium bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700 transition-colors"
            >
              Run Validation
            </button>
          )}

          <button
            onClick={onPublish}
            disabled={hasBlockingFailures || isPublishing}
            className={`px-5 py-2 rounded-lg text-xs font-bold transition-all shadow-lg flex items-center gap-2 ${
              hasBlockingFailures
                ? 'bg-slate-800 text-slate-500 border border-slate-700 cursor-not-allowed'
                : warningCount > 0
                ? 'bg-gradient-to-r from-amber-500 to-amber-600 text-slate-950 hover:from-amber-400 hover:to-amber-500 shadow-amber-500/20'
                : 'bg-gradient-to-r from-amber-500 to-amber-600 text-slate-950 hover:from-amber-400 hover:to-amber-500 shadow-amber-500/25'
            }`}
          >
            {isPublishing ? (
              <span>Publishing...</span>
            ) : hasBlockingFailures ? (
              <span>Required Fields Unresolved</span>
            ) : status === 'PUBLISHED' ? (
              <span>✓ Published (Edit Draft)</span>
            ) : warningCount > 0 ? (
              <span>🚀 Publish BO ({warningCount} Warning)</span>
            ) : (
              <span>🚀 Publish BO</span>
            )}
          </button>
        </div>
      </div>
    </div>
  );
};
