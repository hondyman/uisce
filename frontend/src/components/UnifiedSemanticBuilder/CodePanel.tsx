// Monaco-only implementation with JSON/YAML toggle and semantic-layer completions
import React, { useEffect, useState, useRef, useCallback } from 'react';
import yaml from 'js-yaml';
import { Box, IconButton, Tooltip, ToggleButton, ToggleButtonGroup } from '@mui/material';
import { ShowCode } from './types';
import MonacoCodeEditor from './MonacoCodeEditor.lazy';
import { devLog } from '../../utils/devLogger';
import './CodePanel.css';
import { useGlobalSearch } from '../../contexts/GlobalSearchContext';

interface CodePanelProps {
  showCode: ShowCode;
  modelName: string;
  generateJSON: () => string;
  generateYAML: () => string;
  generateCustomJSON?: () => string;
  generateCustomYAML?: () => string;
  generateCoreJSON?: () => string;
  generateCoreYAML?: () => string;
  generateMergedModelObject?: () => any;
  searchTerm?: string;
  setMatchIndex?: React.Dispatch<React.SetStateAction<number>>;
  setMatchCount?: React.Dispatch<React.SetStateAction<number>>;
  codeEditable?: boolean;
  onImportCode?: (text: string, format: ShowCode) => Promise<void> | void;
  extendsModel?: string | null;
  onToggleFormat?: (format: ShowCode) => void;
  semanticModel?: any;
  selectedModel?: any;
}

export const CodePanel: React.FC<CodePanelProps> = ({
  showCode,
  modelName,
  generateJSON,
  generateYAML,
  generateCustomJSON,
  generateCustomYAML,
  generateCoreJSON,
  generateCoreYAML,
  generateMergedModelObject,
  searchTerm,
  setMatchIndex,
  setMatchCount,
  codeEditable = false,
  onImportCode: _onImportCode,
  extendsModel = null,
  onToggleFormat: _onToggleFormat,
  semanticModel,
  selectedModel,
}) => {
  const monacoApiRef = React.useRef<any | null>(null);
  // Ensure hooks run unconditionally to avoid conditional hook ordering
  // Issues. Compute initial code text lazily to avoid expensive work when
  // no format is selected.
  // Hidden debug toggle for code source: 'auto' (default), 'custom', 'core', 'merged'
  const [codeSource, setCodeSource] = useState<'auto' | 'custom' | 'core' | 'merged'>('auto');

  const initial = React.useMemo(() => {
    if (!showCode) return '';
    const fmtIsJson = showCode === 'json';
    const pickAuto = () => {
      if (selectedModel?.is_custom) {
        return fmtIsJson ? (generateCustomJSON ? generateCustomJSON() : generateJSON()) : (generateCustomYAML ? generateCustomYAML() : generateYAML());
      }
      return fmtIsJson ? generateJSON() : generateYAML();
    };
    const pickBySource = (src: typeof codeSource) => {
      switch (src) {
        case 'custom': return fmtIsJson ? (generateCustomJSON ? generateCustomJSON() : generateJSON()) : (generateCustomYAML ? generateCustomYAML() : generateYAML());
        case 'core': return fmtIsJson ? (generateCoreJSON ? generateCoreJSON() : generateJSON()) : (generateCoreYAML ? generateCoreYAML() : generateYAML());
        case 'merged': {
          const obj = generateMergedModelObject ? generateMergedModelObject() : semanticModel;
          try { return fmtIsJson ? JSON.stringify(obj, null, 2) : yaml.dump(obj); } catch { return fmtIsJson ? '{}' : ''; }
        }
        case 'auto':
        default: return pickAuto();
      }
    };
    return pickBySource(codeSource);
  }, [showCode, generateJSON, generateYAML, generateCustomJSON, generateCustomYAML, generateCoreJSON, generateCoreYAML, generateMergedModelObject, semanticModel, selectedModel, codeSource]);
  const [codeText, setCodeText] = useState<string>(initial);
  const lastFormatRef = useRef<ShowCode>(showCode);
  const lastParsedObjRef = useRef<any>(null);
  const isDirtyRef = useRef<boolean>(false);
  const prevFormatRef = useRef<ShowCode>(showCode);
  const previousNonEmptyRef = useRef<string>(initial);
  const wrapperRef = useRef<HTMLDivElement | null>(null);
  const { searchTerm: globalSearchTerm } = useGlobalSearch();
  const importDebounceRef = React.useRef<number | null>(null);
  // Hidden testing/assistive mirror for Monaco: keeps a textarea synced for selection/jump tests
  const hiddenMirrorRef = useRef<HTMLTextAreaElement | null>(null);

  // Build semantic-aware dynamic completions (dimensions, measures, filters, joins)
  const dynamicCompletions = React.useMemo(() => {
    try {
      const out: Array<{ label: string; insertText: string; kind?: number }> = [];
      const add = (prefix: string, name: string) => {
        if (!name) return;
        // Provide simple name completions that work in both JSON and YAML contexts
        out.push({ label: `${prefix}.${name}`, insertText: `${name}` });
      };
      const sm = semanticModel || {};
      const dims = Array.isArray(sm.dimensions) ? sm.dimensions : [];
      const meas = Array.isArray(sm.measures) ? sm.measures : [];
      const filts = Array.isArray(sm.filters) ? sm.filters : [];
      const joins = Array.isArray(sm.joins) ? sm.joins : [];
      dims.forEach((d: any) => add('dimension', String(d?.name || d?.id || '')));
      meas.forEach((m: any) => add('measure', String(m?.name || m?.id || '')));
      filts.forEach((f: any) => add('filter', String(f?.name || f?.id || '')));
      joins.forEach((j: any) => add('join', String(j?.name || j?.id || '')));
      return out;
    } catch {
      return [] as Array<{ label: string; insertText: string; kind?: number }>;
    }
  }, [semanticModel]);

  // Initial parse
  useEffect(() => {
    try {
      const parsed = showCode === 'json' ? JSON.parse(codeText) : yaml.load(codeText);
      if (parsed && typeof parsed === 'object') lastParsedObjRef.current = parsed;
    } catch (_) {}
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Debug log
  useEffect(() => {
    devLog('[CodePanel] format', showCode, 'len', codeText.length);
  }, [showCode, codeText]);

  // Handle format toggles & underlying model refresh
  useEffect(() => {
    const formatChanged = prevFormatRef.current !== showCode;
  if (formatChanged) {
      let converted = '';
      const obj = lastParsedObjRef.current;
      if (obj) {
        try {
          converted = showCode === 'json' ? JSON.stringify(obj, null, 2) : yaml.dump(obj);
        } catch (_) {}
      }
      if (!converted) {
        const fmtIsJson = showCode === 'json';
        const bySrc = (src: typeof codeSource) => {
          switch (src) {
            case 'custom': return fmtIsJson ? (generateCustomJSON ? generateCustomJSON() : generateJSON()) : (generateCustomYAML ? generateCustomYAML() : generateYAML());
            case 'core': return fmtIsJson ? (generateCoreJSON ? generateCoreJSON() : generateJSON()) : (generateCoreYAML ? generateCoreYAML() : generateYAML());
            case 'merged': {
              const obj = generateMergedModelObject ? generateMergedModelObject() : semanticModel;
              try { return fmtIsJson ? JSON.stringify(obj, null, 2) : yaml.dump(obj); } catch { return fmtIsJson ? '{}' : ''; }
            }
            case 'auto':
            default:
              if (selectedModel?.is_custom) return fmtIsJson ? (generateCustomJSON ? generateCustomJSON() : generateJSON()) : (generateCustomYAML ? generateCustomYAML() : generateYAML());
              return fmtIsJson ? generateJSON() : generateYAML();
          }
        };
        converted = bySrc(codeSource);
      }
      if (showCode === 'yaml' && (!converted || converted.trim().length === 0)) {
        const regen = selectedModel?.is_custom ? (generateCustomYAML ? generateCustomYAML() : generateYAML()) : generateYAML();
        if (regen && regen.trim().length > 0) converted = regen; else if (obj) { try { converted = yaml.dump(obj); } catch (_) {} }
      }
      if ((!converted || converted.trim().length === 0) && previousNonEmptyRef.current.trim().length) converted = previousNonEmptyRef.current;
  setCodeText(converted);
  isDirtyRef.current = false;
  lastFormatRef.current = showCode;
      prevFormatRef.current = showCode;
    devLog('[CodePanel] toggled format ->', showCode, 'len', converted.length);
  devLog('[CodePanel] code after toggle to', showCode, '\n', converted);
      try { const parsed = showCode === 'json' ? JSON.parse(converted) : yaml.load(converted); if (parsed && typeof parsed === 'object') lastParsedObjRef.current = parsed; } catch (_) {}
      if (converted.trim().length) previousNonEmptyRef.current = converted;
    } else if (!isDirtyRef.current) {
      let regenerated = '';
      const fmtIsJson = showCode === 'json';
      const regenBySrc = (src: typeof codeSource) => {
        switch (src) {
          case 'custom': return fmtIsJson ? (generateCustomJSON ? generateCustomJSON() : generateJSON()) : (generateCustomYAML ? generateCustomYAML() : generateYAML());
          case 'core': return fmtIsJson ? (generateCoreJSON ? generateCoreJSON() : generateJSON()) : (generateCoreYAML ? generateCoreYAML() : generateYAML());
          case 'merged': {
            const obj = generateMergedModelObject ? generateMergedModelObject() : semanticModel;
            try { return fmtIsJson ? JSON.stringify(obj, null, 2) : yaml.dump(obj); } catch { return fmtIsJson ? '{}' : ''; }
          }
          case 'auto':
          default:
            if (selectedModel?.is_custom) return fmtIsJson ? (generateCustomJSON ? generateCustomJSON() : generateJSON()) : (generateCustomYAML ? generateCustomYAML() : generateYAML());
            return fmtIsJson ? generateJSON() : generateYAML();
        }
      };
      regenerated = regenBySrc(codeSource);
      if (!(showCode === 'yaml' && regenerated.trim().length === 0)) {
        setCodeText(regenerated);
  devLog('[CodePanel] regenerated code for', showCode, 'format', '\n', regenerated);
        if (regenerated.trim().length) previousNonEmptyRef.current = regenerated;
      }
      devLog('[CodePanel] refreshed (model change, clean) format', showCode, 'len', regenerated.length);
      try { const parsed = showCode === 'json' ? JSON.parse(regenerated) : yaml.load(regenerated); if (parsed && typeof parsed === 'object') lastParsedObjRef.current = parsed; } catch (_) {}
    }
  }, [showCode, generateJSON, generateYAML, generateCustomJSON, generateCustomYAML, generateCoreJSON, generateCoreYAML, generateMergedModelObject, extendsModel, semanticModel, selectedModel, codeSource]);

  // Mount/unmount log
  useEffect(() => {
    devLog('[CodePanel] mounted for model', modelName, 'extends', extendsModel);
    return () => { devLog('[CodePanel] unmounted for model', modelName); };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [modelName, extendsModel]);

  // Hidden debug hotkey: Ctrl+Shift+D cycles code source (auto -> custom -> core -> merged)
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.shiftKey && (e.key === 'D' || e.key === 'd')) {
        setCodeSource(prev => prev === 'auto' ? 'custom' : prev === 'custom' ? 'core' : prev === 'core' ? 'merged' : 'auto');
        e.preventDefault();
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, []);

  // Jump to section (approximate scroll) for Prism editor
  useEffect(() => {
    const escapeRegex = (s: string) => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');

    const handler = (e: any) => {
      const section = e?.detail?.section; if (!section) return;
      const key = e?.detail?.key as string | undefined;
  const ta = hiddenMirrorRef.current || wrapperRef.current?.querySelector('textarea');
      if (!ta) return;
      const lines = codeText.split('\n');
      const isJson = showCode === 'json';
      const sectionRegex = isJson ? new RegExp(`"${escapeRegex(section)}"\\s*:`) : new RegExp(`^\\s*${escapeRegex(section)}:`);
      let sectionStart = lines.findIndex(l => sectionRegex.test(l));
      if (sectionStart < 0) return;

      if (key) {
        const parts = key.split('.').filter(Boolean);
        const last = parts[parts.length - 1];

        if (isJson) {
          const lastRegex = new RegExp(`"${escapeRegex(last)}"\\s*:`, 'i');
          // find all indices with last key
          const candIdx: number[] = [];
          for (let i = sectionStart; i < lines.length; i++) {
            if (lastRegex.test(lines[i])) candIdx.push(i);
            // break if we hit a new top-level section
            if (i > sectionStart && /^\w+\s*:/m.test(lines[i])) break;
          }
          // try to validate parent chain for each candidate by searching upward
          const maxLookback = 80;
          let chosen: number | null = null;
          for (const idx of candIdx) {
            let ok = true;
            let searchPos = idx;
            for (let p = parts.length - 2; p >= 0; p--) {
              const parent = parts[p];
              // search upward up to maxLookback lines for parent
              let found = false;
              for (let j = Math.max(sectionStart, searchPos - maxLookback); j >= sectionStart; j--) {
                if (new RegExp(`"${escapeRegex(parent)}"\\s*:`, 'i').test(lines[j])) { found = true; searchPos = j; break; }
              }
              if (!found) { ok = false; break; }
            }
            if (ok) { chosen = idx; break; }
          }
          if (chosen !== null) sectionStart = chosen;
          else if (candIdx.length > 0) sectionStart = candIdx[0];
        } else {
          // YAML: look for last key and try to find parent keys above
          const lastRegex = new RegExp(`^\\s*${escapeRegex(last)}\\s*:`, 'i');
          const candIdx: number[] = [];
          for (let i = sectionStart; i < lines.length; i++) {
            if (lastRegex.test(lines[i])) candIdx.push(i);
            if (i > sectionStart && /^\w+\s*:/m.test(lines[i])) break;
          }
          const maxLookback = 80;
          let chosen: number | null = null;
          for (const idx of candIdx) {
            let ok = true;
            let searchPos = idx;
            for (let p = parts.length - 2; p >= 0; p--) {
              const parent = parts[p];
              let found = false;
              for (let j = Math.max(sectionStart, searchPos - maxLookback); j >= sectionStart; j--) {
                if (new RegExp(`^\\s*${escapeRegex(parent)}\\s*:`, 'i').test(lines[j])) { found = true; searchPos = j; break; }
              }
              if (!found) { ok = false; break; }
            }
            if (ok) { chosen = idx; break; }
          }
          if (chosen !== null) sectionStart = chosen;
          else if (candIdx.length > 0) sectionStart = candIdx[0];
        }
      }

      const lineHeight = parseFloat(getComputedStyle(ta).lineHeight || '18');
      ta.scrollTop = Math.max(0, sectionStart * lineHeight - 40);
      let pos = 0; for (let i = 0; i < sectionStart; i++) pos += lines[i].length + 1;
      ta.selectionStart = ta.selectionEnd = pos;
      ta.focus();
    };
    window.addEventListener('semlayer.jumpToSection', handler as EventListener);
    return () => window.removeEventListener('semlayer.jumpToSection', handler as EventListener);
  }, [codeText, showCode]);

  // Listen for validation issues from other components and convert to Monaco markers
  useEffect(() => {
  // store api exposed by MonacoCodeEditor via monacoApiRef
  const onMount = (api: any) => { monacoApiRef.current = api; };
  // referenced so compiler doesn't mark as unused in some build variants
  void onMount;
    // Attach a handler that uses the apiRef to set markers
    const handler = (e: any) => {
      const issues = e?.detail?.issues as Array<any> | undefined;
      if (!issues || !monacoApiRef.current) return;
      try {
        const monaco = (window as any).monaco;
        const text = monacoApiRef.current.getValue?.() || '';
        const lines = text.split('\n');
        const escapeRegex = (s: string) => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
        const markers = issues.map((it) => {
          const severity = it.level === 'warning' ? (monaco?.MarkerSeverity?.Warning ?? 4) : (monaco?.MarkerSeverity?.Error ?? 8);
          const msg = it.message || it.code || String(it);
          let startLineNumber = 1, startColumn = 1, endLineNumber = 1, endColumn = 1;

          // Prefer explicit line/col from backend
          if (typeof it.line === 'number' && typeof it.col === 'number') {
            startLineNumber = Math.max(1, Math.floor(it.line));
            startColumn = Math.max(1, Math.floor(it.col));
            if (typeof it.endLine === 'number' && typeof it.endCol === 'number') {
              endLineNumber = Math.max(startLineNumber, Math.floor(it.endLine));
              endColumn = Math.max(1, Math.floor(it.endCol));
            } else {
              endLineNumber = startLineNumber;
              endColumn = startColumn + (typeof it.length === 'number' ? Math.max(1, Math.floor(it.length)) : 1);
            }
          } else {
            // Prefer element_id when available (more deterministic)
            const elementId = it.element_id || it.elementId || it.id;
            if (elementId && typeof elementId === 'string') {
              const last = String(elementId).split('.').pop() || String(elementId);
              const regexes = [
                new RegExp(`\\"id\\"\\s*:\\s*\\\"${escapeRegex(last)}\\\"`, 'i'),
                new RegExp(`\\"name\\"\\s*:\\s*\\\"${escapeRegex(last)}\\\"`, 'i'),
                new RegExp(`^\\s*-\\s*name:\\s*${escapeRegex(last)}\\b`, 'i'),
                new RegExp(`\\b${escapeRegex(last)}\\b`, 'i'),
              ];
              let foundIdx = -1;
              for (let i = 0; i < lines.length; i++) {
                for (const r of regexes) {
                  if (r.test(lines[i])) { foundIdx = i; break; }
                }
                if (foundIdx !== -1) break;
              }
              if (foundIdx >= 0) {
                startLineNumber = foundIdx + 1;
                const lineLower = lines[foundIdx];
                const col = Math.max(0, lineLower.toLowerCase().indexOf(last.toLowerCase()));
                startColumn = col >= 0 ? col + 1 : 1;
                endLineNumber = startLineNumber;
                endColumn = startColumn + last.length;
              }
            }

            // Fallback to key-based heuristic if element_id didn't match
            if (startLineNumber === 1 && startColumn === 1 && it.key && typeof it.key === 'string') {
              const last = it.key.split('.').pop() || it.key;
              const idx = lines.findIndex((l: string) => l.toLowerCase().includes(`\"${last.toLowerCase()}\"`) || l.toLowerCase().includes(last.toLowerCase()));
              if (idx >= 0) {
                startLineNumber = idx + 1;
                startColumn = Math.max(1, (lines[idx].toLowerCase().indexOf(last.toLowerCase()) + 1));
                endLineNumber = startLineNumber;
                endColumn = startColumn + last.length;
              }
            }
          }

          // include element identifier on the marker object so consumers (and tests)
          // can map back to the originating element when present
          const elementIdOut = it.element_id || it.elementId || it.id || undefined;
          return { startLineNumber, startColumn, endLineNumber, endColumn, message: msg, severity, code: it.code || it.code_id || undefined, element_id: elementIdOut };
        });
        try { monacoApiRef.current?.setMarkers?.(markers); } catch (_) {}
      } catch (_) {}
    };
    window.addEventListener('semlayer.validationIssues', handler as EventListener);
  return () => window.removeEventListener('semlayer.validationIssues', handler as EventListener);
  }, []);
  const navigateMatch = useCallback((dir: number) => {
    const term = (searchTerm ?? globalSearchTerm ?? '').trim();
    if (!term) return;
    const lines = codeText.split(/\n/);
    const matches: Array<{ line: number }> = [];
    lines.forEach((l, i) => { if (l.toLowerCase().includes(term.toLowerCase())) matches.push({ line: i + 1 }); });
    if (matches.length === 0) { setMatchCount?.(0); setMatchIndex?.(0); return; }

    setMatchCount?.(matches.length);
    setMatchIndex?.((prev: number) => (prev + dir + matches.length) % matches.length);
  }, [searchTerm, globalSearchTerm, codeText, setMatchCount, setMatchIndex]);

  // Recalculate matches when search term changes (reset to first match)
  useEffect(() => {
    const term = (searchTerm ?? globalSearchTerm ?? '').trim();
    if (!term) { setMatchCount?.(0); setMatchIndex?.(0); return; }
    const lines = codeText.split(/\n/);
    const matchLines = lines.filter(l => l.toLowerCase().includes(term.toLowerCase()));
    if (matchLines.length === 0) { setMatchCount?.(0); setMatchIndex?.(0); return; }
    setMatchCount?.(matchLines.length);
    setMatchIndex?.(0);
  }, [searchTerm, globalSearchTerm, codeText, setMatchCount, setMatchIndex]);

  // copy/download controls are provided in the workspace header; removed here to avoid duplication

  useEffect(() => {
    const handler = (e: any) => {
      if (e.detail?.direction) navigateMatch(e.detail.direction);
    };
    window.addEventListener('semlayer.navigateMatch', handler);
    return () => window.removeEventListener('semlayer.navigateMatch', handler);
  }, [navigateMatch]);

  // If no code view is requested, don't render anything. Hooks already ran
  // above so hook order remains stable across renders.
  if (!showCode) return null;

  return (
    <div className="code-panel" ref={wrapperRef}>
      <div className="code-actions" aria-label="Code actions">
        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 1, width: '100%' }}>
          {/* Source toggle: Auto / Custom / Core / Merged */}
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Tooltip title="Choose code source: Auto selects Custom for custom models, Core for core models">
              <ToggleButtonGroup
                size="small"
                exclusive
                value={codeSource}
                onChange={(_e, val) => { if (val) setCodeSource(val); }}
                aria-label="Code source selector"
              >
                <ToggleButton value="auto" aria-label="Auto source">Auto</ToggleButton>
                <ToggleButton value="custom" aria-label="Custom source">Custom</ToggleButton>
                <ToggleButton value="core" aria-label="Core source">Core</ToggleButton>
                <ToggleButton value="merged" aria-label="Merged source">Merged</ToggleButton>
              </ToggleButtonGroup>
            </Tooltip>
            {/* Format toggle: JSON / YAML */}
            <Tooltip title="Toggle code format">
              <ToggleButtonGroup
                size="small"
                exclusive
                value={showCode}
                onChange={(_e, val) => { if (val === 'json' || val === 'yaml') { try { _onToggleFormat?.(val); } catch {} } }}
                aria-label="Code format selector"
              >
                <ToggleButton value="json" aria-label="JSON format">JSON</ToggleButton>
                <ToggleButton value="yaml" aria-label="YAML format">YAML</ToggleButton>
              </ToggleButtonGroup>
            </Tooltip>
          </Box>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Tooltip title="Apply code to tiles">
              <IconButton aria-label="Apply" size="small" onClick={() => { try { _onImportCode?.(codeText, showCode); } catch {} }}>
                <svg width="14" height="14" viewBox="0 0 24 24"><path d="M12 2L2 7l10 5 10-5-10-5zm0 7.5L4.2 7 12 3.5 19.8 7 12 9.5zM2 17l10 5 10-5v-2l-10 5L2 15v2z" fill="currentColor"/></svg>
              </IconButton>
            </Tooltip>
          </Box>
          {/* Copy/Download moved to the workspace header */}
        </Box>
      </div>
      {extendsModel && (
        <div className="extends-header">
          <label>extends</label>
          <div className="extends-value">{extendsModel}</div>
        </div>
      )}
  <div className="editor-wrapper-full">
          {/* Monaco is lazily loaded; if unavailable, the container will be empty. */}
          {/* eslint-disable-next-line @typescript-eslint/ban-ts-comment */}
          {/* @ts-ignore */}
          <MonacoCodeEditor
            value={codeText}
            language={showCode}
            readOnly={!codeEditable}
            onMount={(api: any) => { monacoApiRef.current = api; }}
            dynamicCompletions={dynamicCompletions}
            onChange={(val: string) => {
              setCodeText(val);
              isDirtyRef.current = true;
              try { window.dispatchEvent(new CustomEvent('semlayer.markDirty')); } catch(_) {}
              // Debounced live import
              try {
                if (importDebounceRef.current) window.clearTimeout(importDebounceRef.current);
                importDebounceRef.current = window.setTimeout(() => { try { _onImportCode?.(val, showCode); } catch {} }, 600);
              } catch {}
              // Update lastParsedObjRef for in-memory conversion between JSON/YAML
              try {
                const parsed = showCode === 'json' ? JSON.parse(val) : yaml.load(val);
                if (parsed && typeof parsed === 'object') lastParsedObjRef.current = parsed;
              } catch {}
              // Sync hidden mirror for tests
              try { if (hiddenMirrorRef.current) hiddenMirrorRef.current.value = val; } catch {}
            }}
          />
          {/* Hidden mirror textarea used in tests and for screen readers if needed.
              In tests (no Monaco), we accept edits here and propagate to state/import handler. */}
          <textarea
            ref={hiddenMirrorRef}
            aria-label={`${showCode?.toUpperCase() || 'CODE'} code editor`}
            value={codeText}
            onChange={(e) => {
              const val = e.target.value;
              setCodeText(val);
              isDirtyRef.current = true;
              try { window.dispatchEvent(new CustomEvent('semlayer.markDirty')); } catch (_) {}
              try { _onImportCode?.(val, showCode); } catch {}
            }}
            className="hidden-mirror"
          />
      </div>
    </div>
  );
};
