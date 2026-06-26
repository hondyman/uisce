import { useState, useEffect, useCallback } from 'react';
import { devError } from '../../../utils/devLogger';
import type { AliasMap } from '../../../types/upgrade-generated';
import { getAliasMap, analyzeExtensionFixes, applyExtensionFixes } from '../api';
import getErrorMessage from '../../../utils/errors';

interface ExtensionFix {
  file_path: string;
  fixes: ExtensionFixEntry[];
}

interface ExtensionFixEntry {
  line_number: number;
  old_code: string;
  new_code: string;
  alias_used?: string;
  confidence: 'high' | 'medium' | 'low';
}

interface ExtensionFixerProps {
  fromVersion: string;
  toVersion: string;
  onClose?: () => void;
}

export default function ExtensionFixer({ fromVersion, toVersion, onClose }: ExtensionFixerProps) {
  const [aliasMap, setAliasMap] = useState<AliasMap | null>(null);
  const [fixes, setFixes] = useState<ExtensionFix[]>([]);
  const [loading, setLoading] = useState(false);
  const [applying, setApplying] = useState(false);

  const loadAliasMap = useCallback(async () => {
    try {
      setLoading(true);
      const map = await getAliasMap(fromVersion, toVersion);
      setAliasMap(map);
    } catch (error: unknown) {
      devError('Failed to load alias map:', getErrorMessage(error));
    } finally {
      setLoading(false);
    }
  }, [fromVersion, toVersion]);

  const analyzeFixes = useCallback(async () => {
    if (!aliasMap) return;

    try {
      setLoading(true);
      // Derive candidate extension files; if server needs file paths, provide an empty list (backend may infer)
      const extensionFiles: string[] = Array.isArray((aliasMap as any).aliases)
        ? []
        : Object.keys(((aliasMap as unknown as Record<string, unknown>).aliases) ?? {});
      const analyzedFixes = await analyzeExtensionFixes(toVersion, extensionFiles);
      setFixes(analyzedFixes);
    } catch (error: unknown) {
      devError('Failed to analyze fixes:', getErrorMessage(error));
    } finally {
      setLoading(false);
    }
  }, [aliasMap, toVersion]);

  useEffect(() => {
    loadAliasMap();
  }, [loadAliasMap]);

  useEffect(() => {
    if (aliasMap) {
      analyzeFixes();
    }
  }, [aliasMap, analyzeFixes]);

  const handleApplyFixes = async () => {
    try {
      setApplying(true);
    await applyExtensionFixes(toVersion, fixes);
      await analyzeFixes();
    } catch (error: unknown) {
      devError('Failed to apply fixes:', getErrorMessage(error));
    } finally {
      setApplying(false);
    }
  };

  return (
    <div className="extension-fixer">
      <header className="flex justify-between items-center mb-4">
        <h2 className="text-xl font-semibold">Extension Fixer</h2>
        <div className="flex gap-2">
          <span className="text-sm text-gray-600">{fixes.length} files</span>
          <button
            className="px-3 py-1 bg-blue-500 text-white rounded"
            onClick={loadAliasMap}
            disabled={loading}
          >
            Refresh
          </button>
        </div>
      </header>

      <p className="text-sm text-gray-600 mb-4">
        Comparing {fromVersion} → {toVersion}
      </p>

      {loading && <p>Analyzing extensions...</p>}

      <section className="space-y-4">
        {fixes.map((fix) => (
          <div key={fix.file_path} className="border rounded p-4">
            <h3 className="font-semibold">{fix.file_path}</h3>
            <p className="text-sm text-gray-600">{fix.fixes.length} fixes</p>

            <ul className="mt-2 space-y-1">
              {fix.fixes.map((fixEntry, idx) => (
                <li key={idx} className="text-sm">
                  <span className="font-mono">Line {fixEntry.line_number}:</span>{' '}
                  <span className={`px-2 py-1 rounded text-xs ${
                    fixEntry.confidence === 'high' ? 'bg-green-200' :
                    fixEntry.confidence === 'medium' ? 'bg-yellow-200' : 'bg-red-200'
                  }`}>
                    {fixEntry.confidence}
                  </span>
                  {fixEntry.alias_used && (
                    <span className="text-gray-500"> (using {fixEntry.alias_used})</span>
                  )}
                </li>
              ))}
            </ul>
          </div>
        ))}

        {fixes.length === 0 && !loading && (
          <p className="text-gray-500">No extension fixes needed.</p>
        )}
      </section>

      {fixes.length > 0 && (
        <div className="mt-4 flex justify-end">
          <button
            className="px-4 py-2 bg-green-500 text-white rounded"
            onClick={handleApplyFixes}
            disabled={applying}
          >
            {applying ? 'Applying...' : `Apply ${fixes.length} Files`}
          </button>
        </div>
      )}

      {onClose && (
        <div className="mt-4 flex justify-end">
          <button
            className="px-4 py-2 bg-gray-500 text-white rounded"
            onClick={onClose}
          >
            Close
          </button>
        </div>
      )}
    </div>
  );
}