import React, { useEffect, useState, Suspense, lazy } from 'react';
import { Dialog, DialogTitle, DialogContent, DialogActions, Button, CircularProgress, Typography, Divider, Box } from '@mui/material';

// Lazy-load the Monaco DiffEditor (safe for test environments where monaco is mocked/unavailable)
const DiffEditorLazy = lazy(async () => {
  try {
    const mod = await import('@monaco-editor/react');
    return { default: mod.DiffEditor ? mod.DiffEditor : () => <Typography variant="body2">Diff editor unavailable in this environment.</Typography> };
  } catch (e) {
    return { default: () => <Typography variant="body2">Diff editor unavailable in this environment.</Typography> };
  }
});

// A typed 'any' wrapper to avoid compile-time prop checking on the lazy component
const DiffEditorAny: any = DiffEditorLazy as unknown as any;
import { useTenant } from '../../contexts/TenantContext';
import { RuleDiff, DiffField } from '../../types/rules';
import { rulesApi } from '../../services/rulesApi';

interface RuleDiffViewerProps {
  isOpen: boolean;
  onClose: () => void;
  boId: string;
  ruleId: string;
}

export const RuleDiffViewer: React.FC<RuleDiffViewerProps> = ({ isOpen, onClose, boId, ruleId }) => {
  const { tenant, datasource } = useTenant();
  const [diff, setDiff] = useState<RuleDiff | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (isOpen && boId && ruleId) {
      loadDiff();
    }
  }, [isOpen, boId, ruleId]);

  const loadDiff = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await rulesApi.fetchRuleDiff(boId, ruleId, tenant?.id, datasource?.id);
      setDiff(data);
    } catch (err) {
      console.error('Failed to load diff:', err);
      setError('Failed to load rule comparison.');
    } finally {
      setLoading(false);
    }
  };

  const renderDiffLine = (label: string, oldVal: string, newVal: string, isChanged: boolean) => (
    <div className={`grid grid-cols-12 gap-4 py-2 border-b ${isChanged ? 'bg-yellow-50' : ''}`}>
      <div className="col-span-2 font-medium text-gray-700">{label}</div>
      <div className={`col-span-5 text-sm font-mono whitespace-pre-wrap ${isChanged ? 'text-red-700 bg-red-50' : 'text-gray-500'}`}>
        {oldVal || <span className="text-gray-400 italic">None</span>}
      </div>
      <div className={`col-span-5 text-sm font-mono whitespace-pre-wrap ${isChanged ? 'text-green-700 bg-green-50' : 'text-gray-900'}`}>
        {newVal || <span className="text-gray-400 italic">None</span>}
      </div>
    </div>
  );

  const [showDslDiff, setShowDslDiff] = useState(false);

  return (
    <Dialog open={isOpen} onClose={onClose} maxWidth="lg" fullWidth>
      <DialogTitle>
        Rule Comparison
        <Typography variant="body2" color="textSecondary">
          Comparing current rule version against its base (core/inherited).
        </Typography>
      </DialogTitle>
      <DialogContent>
        {loading ? (
          <div className="flex justify-center p-8">
            <CircularProgress />
          </div>
        ) : error ? (
          <div className="text-red-600 p-4">{error}</div>
        ) : diff ? (
          <div className="mt-4">
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <div>
                <Typography variant="subtitle1" sx={{ fontWeight: 700 }}>Comparison</Typography>
                <Typography variant="caption" color="text.secondary">Field-level and expression differences</Typography>
              </div>
              <div>
                <Button variant="outlined" size="small" onClick={() => setShowDslDiff(true)} sx={{ mr: 1 }}>Show DSL Diff</Button>
                <Button variant="text" size="small" onClick={loadDiff}>Refresh</Button>
              </div>
            </Box>

            <Box sx={{ mt: 1.5 }}>
              <div className="grid grid-cols-12 gap-4 pb-2 border-b-2 font-bold mb-2">
                <div className="col-span-2">Field</div>
                <div className="col-span-5">Base (Core/Inherited)</div>
                <div className="col-span-5">Current (Local/Override)</div>
              </div>

              {renderDiffLine(
                "Name", 
                diff.base?.name, 
                diff.current?.name, 
                diff.base?.name !== diff.current?.name
              )}

              {renderDiffLine(
                "Severity", 
                diff.base?.severity, 
                diff.current?.severity, 
                diff.base?.severity !== diff.current?.severity
              )}

              {renderDiffLine(
                "Scope", 
                diff.base?.scope, 
                diff.current?.scope, 
                diff.base?.scope !== diff.current?.scope
              )}

              {renderDiffLine(
                "Expression", 
                diff.base?.expression, 
                diff.current?.expression, 
                diff.base?.expression !== diff.current?.expression
              )}

              {diff.diffs.length === 0 && (
                <div className="text-center py-8 text-gray-500">
                  No differences found. The rules are identical.
                </div>
              )}
            </Box>

            {/* DSL Diff Modal */}
            <Dialog open={showDslDiff} onClose={() => setShowDslDiff(false)} fullWidth maxWidth="xl">
              <DialogTitle>DSL Diff</DialogTitle>
              <DialogContent>
                <Box sx={{ height: 600 }}>
                  <Suspense fallback={<Box sx={{ p: 2 }}><Typography variant="body2">Loading diff editor...</Typography></Box>}>
                    {/* Using an any-typed wrapper to avoid compile-time prop checking */}
                    <DiffEditorAny
                      original={diff.base?.expression || ''}
                      modified={diff.current?.expression || ''}
                      language="semrule"
                      options={{ readOnly: true, minimap: { enabled: false } }}
                      height="100%"
                    />
                  </Suspense>
                </Box>
              </DialogContent>
              <DialogActions>
                <Button onClick={() => setShowDslDiff(false)}>Close</Button>
              </DialogActions>
            </Dialog>

          </div>
        ) : null}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Close</Button>
      </DialogActions>
    </Dialog>
  );
};
