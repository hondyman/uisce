import React, { useState, useEffect, useCallback, useRef } from 'react';
import {
  Box, Container, Typography, Paper, Button, TextField,
  MenuItem, Select, InputLabel, FormControl, Stack, Alert, Chip,
  Divider, CircularProgress,
  ToggleButton, ToggleButtonGroup, Accordion, AccordionSummary, AccordionDetails,
  Avatar, Tooltip, IconButton,
} from '@mui/material';
import { alpha } from '@mui/material/styles';
import RuleIcon from '@mui/icons-material/Rule';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import CodeIcon from '@mui/icons-material/Code';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import SaveIcon from '@mui/icons-material/Save';
import FunctionsIcon from '@mui/icons-material/Functions';
import StorageIcon from '@mui/icons-material/Storage';
import WarningAmberIcon from '@mui/icons-material/WarningAmber';
import BlockIcon from '@mui/icons-material/Block';
import RefreshIcon from '@mui/icons-material/Refresh';
import ChecklistIcon from '@mui/icons-material/Checklist';
import Editor, { OnMount } from '@monaco-editor/react';
import type * as Monaco from 'monaco-editor';
import AdvancedConditionBuilder, { ConditionGroup, ConditionNode, EntityDefinition, FieldDefinition } from '../components/ExpressionBuilder/AdvancedConditionBuilder';
import {
  evaluateRuleWasm, parseExpressionWasm, evaluateExpressionTextWasm,
  ExpressionParseError,
} from '../rules/wasmRuntime';
import { registerUisceExpressionLanguage, UISCE_EXPRESSION_LANGUAGE, setAslFields } from '../rules/aslMonacoRegistry';
import apiClient from '../utils/apiClient';

// Small section-header pattern shared by every card on this page - an
// icon-in-a-tinted-circle plus title/subtitle, matching the visual
// language BusinessObjectDetailsPage/components/PageHeader.tsx already
// established elsewhere in the app, rather than the bare Typography
// headers this page had before.
function SectionHeader({
  icon, title, subtitle, action,
}: { icon: React.ReactNode; title: string; subtitle?: string; action?: React.ReactNode }) {
  return (
    <Stack direction="row" alignItems="center" spacing={1.5} sx={{ mb: 2.5 }}>
      <Avatar
        variant="rounded"
        sx={{
          bgcolor: (theme) => alpha(theme.palette.primary.main, theme.palette.mode === 'dark' ? 0.2 : 0.1),
          color: 'primary.main',
          width: 40, height: 40,
        }}
      >
        {icon}
      </Avatar>
      <Box sx={{ flex: 1, minWidth: 0 }}>
        <Typography variant="h6" sx={{ fontWeight: 700, lineHeight: 1.2 }}>{title}</Typography>
        {subtitle && <Typography variant="body2" color="text.secondary">{subtitle}</Typography>}
      </Box>
      {action}
    </Stack>
  );
}

const SEVERITY_META: Record<string, { color: 'error' | 'warning'; icon: React.ReactNode }> = {
  BLOCK: { color: 'error', icon: <BlockIcon fontSize="small" /> },
  WARN: { color: 'warning', icon: <WarningAmberIcon fontSize="small" /> },
};

// Converts the editor's ConditionNode shape into the wire format
// internal/rules/vm.RuleNode.UnmarshalJSON expects (flat "type" +
// sibling fields, not nested under a "Condition"/"Group" key - see
// backend/internal/rules/vm/ast.go). Structural discrimination
// ("conditions" in node) rather than trusting node.type, since Condition
// nodes from the builder don't always set an explicit type.
function toRuleNode(node: ConditionNode): unknown {
  if ('conditions' in node) {
    return {
      type: 'group',
      id: node.id,
      operator: node.operator,
      conditions: node.conditions.map(toRuleNode),
    };
  }
  return {
    type: 'condition',
    id: node.id,
    field: node.fieldPath || node.field,
    operator: node.operator,
    value: node.value,
  };
}

const INITIAL_RULE: ConditionGroup = {
  id: 'root',
  type: 'group',
  operator: 'AND',
  conditions: [
    { id: 'c1', type: 'condition', field: '', operator: 'greater_than', value: 0 },
  ],
};

const SAMPLE_CONTEXT = { total: 150, status: 'pending', is_gift: false };

// Postgres data_type -> the editor's field-type vocabulary.
function toFieldType(dataType: string): FieldDefinition['type'] {
  if (dataType.startsWith('numeric') || dataType.startsWith('integer') || dataType.startsWith('double')) return 'number';
  if (dataType === 'boolean') return 'boolean';
  if (dataType.startsWith('timestamp') || dataType === 'date') return 'date';
  return 'string';
}

interface BOOption {
  id: string;
  key: string;
  name: string;
}

interface SavedRule {
  id: string;
  name: string;
  bo_name: string;
  severity: string;
  timing: string;
  category?: string;
}

interface ViolationRow {
  id: string;
  rule_name: string;
  bo_key: string;
  severity: string;
  record_id: string;
  message: string;
  write_blocked: boolean;
  rule_error: boolean;
  created_at: string;
}

const SAMPLE_EXPRESSION = 'SUM(ExecQuantity * ExecPrice)';

const AdvancedRuleBuilderPage: React.FC = () => {
  const [rule, setRule] = useState<ConditionGroup>(INITIAL_RULE);
  const [contextJson, setContextJson] = useState(JSON.stringify(SAMPLE_CONTEXT, null, 2));
  const [evalResult, setEvalResult] = useState<{ result?: boolean | number; resultType?: string; error?: string } | null>(null);
  const [evaluating, setEvaluating] = useState(false);

  // Authoring mode: the structured condition/group builder (dropdowns,
  // no FuncCall/Expression authoring surface) vs. free-text expression
  // mode (Monaco, bound to the real function registry via
  // asl.monaco.json) - the surface item D in the handoff named as the
  // calc side's missing mirror. Expression mode produces the same
  // vm.Expression AST either way it's saved: as a validation rule's
  // rule_ast (wrapped in {type:"expression", root:...}) or as a calc
  // term's rule_ast (via /calc-terms, unwrapped).
  const [mode, setMode] = useState<'structured' | 'expression'>('structured');
  const [expressionText, setExpressionText] = useState(SAMPLE_EXPRESSION);
  const [exprParseError, setExprParseError] = useState<{ message: string; pos: number } | null>(null);
  const monacoRef = useRef<typeof Monaco | null>(null);
  const editorRef = useRef<Monaco.editor.IStandaloneCodeEditor | null>(null);

  const [calcTermSaving, setCalcTermSaving] = useState(false);
  const [calcTermSaveResult, setCalcTermSaveResult] = useState<{ id?: string; error?: string } | null>(null);
  const [sqlPreview, setSqlPreview] = useState<{ sql?: string; error?: string } | null>(null);
  const [sqlPreviewLoading, setSqlPreviewLoading] = useState(false);

  // Real BO catalog data, replacing MOCK_ENTITIES.
  const [businessObjects, setBusinessObjects] = useState<BOOption[]>([]);
  const [selectedBOKey, setSelectedBOKey] = useState<string>('');
  const [fields, setFields] = useState<FieldDefinition[]>([]);
  const [loadingFields, setLoadingFields] = useState(false);

  // Save form fields - the editor previously collected none of these,
  // even though ValidationRuleProperties requires them.
  const [ruleName, setRuleName] = useState('');
  const [severity, setSeverity] = useState('BLOCK');
  const [timing, setTiming] = useState('pre_write');
  const [category, setCategory] = useState('');
  const [domain, setDomain] = useState('validation');
  const [saving, setSaving] = useState(false);
  const [saveResult, setSaveResult] = useState<{ id?: string; error?: string } | null>(null);

  const [savedRules, setSavedRules] = useState<SavedRule[]>([]);
  const [violations, setViolations] = useState<ViolationRow[]>([]);

  const entities: EntityDefinition[] = selectedBOKey
    ? [{ name: selectedBOKey, label: selectedBOKey, fields, relationships: [] }]
    : [];

  // Load the real BO catalog (replaces MOCK_ENTITIES). Preselects
  // ?bo_name= from the URL when present - the "open in editor" link from
  // the BO Validations tab (BusinessObjectDetailsPage) lands here with
  // that param set, so the editor opens already scoped to the BO the
  // user came from instead of the catalog's first entry.
  useEffect(() => {
    apiClient<BOOption[]>('/business-objects?format=array')
      .then((bos) => {
        setBusinessObjects(bos);
        const fromURL = new URLSearchParams(window.location.search).get('bo_name');
        if (fromURL && bos.some((bo) => bo.key === fromURL)) {
          setSelectedBOKey(fromURL);
        } else if (bos.length > 0) {
          setSelectedBOKey((prev) => prev || bos[0].key);
        }
      })
      .catch((err) => console.error('Failed to load business objects:', err));
  }, []);

  // Load physical fields (the vocabulary rules actually evaluate against
  // - see ValidationRuleService.ListPhysicalFields) whenever the BO
  // selection changes.
  const loadFieldsAndRules = useCallback(async () => {
    if (!selectedBOKey) return;
    setLoadingFields(true);
    try {
      const fieldsRes = await apiClient<{ fields: { name: string; dataType: string }[] }>(
        `/validation-rule-nodes/bo-fields?bo_name=${encodeURIComponent(selectedBOKey)}`
      );
      setFields(
        (fieldsRes.fields || []).map((f) => ({
          name: f.name,
          label: f.name,
          type: toFieldType(f.dataType || ''),
        }))
      );
      const rulesRes = await apiClient<{ validationRules: SavedRule[] }>(
        `/validation-rule-nodes?bo_name=${encodeURIComponent(selectedBOKey)}`
      );
      setSavedRules(rulesRes.validationRules || []);
      const violationsRes = await apiClient<{ violations: ViolationRow[] }>(
        `/validation-rule-nodes/violations?bo_name=${encodeURIComponent(selectedBOKey)}&limit=25`
      );
      setViolations(violationsRes.violations || []);
    } catch (err) {
      console.error('Failed to load fields/rules/violations for BO:', err);
    } finally {
      setLoadingFields(false);
    }
  }, [selectedBOKey]);

  useEffect(() => {
    loadFieldsAndRules();
  }, [loadFieldsAndRules]);

  // Keep the expression editor's field-completion source in sync with
  // the BO's real fields (dot notation: "client.risk_score" would need
  // an `entity`-tagged field here - fields is flat/single-entity today,
  // so entityScope filtering in aslMonacoRegistry currently just means
  // "no dotted fields offered yet for this BO," not a limitation of the
  // completion provider itself).
  useEffect(() => {
    setAslFields(fields.map((f) => ({ name: f.name, type: f.type, entity: f.entity, description: f.description })));
  }, [fields]);

  // Live syntax checking: reparse on every edit (debounced) and render
  // the result as an inline Monaco marker at the real character offset
  // vm.ParseExpression's *ParseError reports - not a toast the user has
  // to correlate with a position themselves.
  useEffect(() => {
    if (mode !== 'expression') return;
    const handle = setTimeout(() => {
      parseExpressionWasm(expressionText)
        .then(() => {
          setExprParseError(null);
          if (monacoRef.current && editorRef.current) {
            monacoRef.current.editor.setModelMarkers(editorRef.current.getModel()!, 'uisce-expr', []);
          }
        })
        .catch((err) => {
          if (err instanceof ExpressionParseError) {
            setExprParseError({ message: err.message, pos: err.pos });
            if (monacoRef.current && editorRef.current) {
              const model = editorRef.current.getModel()!;
              const posAt = model.getPositionAt(err.pos);
              monacoRef.current.editor.setModelMarkers(model, 'uisce-expr', [{
                startLineNumber: posAt.lineNumber, startColumn: posAt.column,
                endLineNumber: posAt.lineNumber, endColumn: posAt.column + 1,
                message: err.message,
                severity: monacoRef.current.MarkerSeverity.Error,
              }]);
            }
          } else {
            setExprParseError({ message: err instanceof Error ? err.message : String(err), pos: 0 });
          }
        });
    }, 300);
    return () => clearTimeout(handle);
  }, [expressionText, mode]);

  const handleExpressionEditorMount: OnMount = (editor, monaco) => {
    editorRef.current = editor;
    monacoRef.current = monaco;
  };

  const runBackendPreview = async () => {
    setEvaluating(true);
    setEvalResult(null);
    try {
      const ctx = JSON.parse(contextJson);
      if (mode === 'expression') {
        const { result, resultType } = await evaluateExpressionTextWasm(expressionText, ctx);
        setEvalResult({ result, resultType });
      } else {
        const ruleNode = toRuleNode(rule);
        const result = await evaluateRuleWasm(ruleNode, ctx);
        setEvalResult({ result, resultType: 'boolean' });
      }
    } catch (err) {
      setEvalResult({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      setEvaluating(false);
    }
  };

  // Save path 1: expression mode -> validation rule. The parsed
  // Expression's root becomes a RuleNode of type "expression" (see
  // internal/rules/vm/ast.go's RuleNode/Expression shapes) - the same
  // /validation-rule-nodes endpoint the structured builder already
  // posts to, just a different rule_ast shape.
  const handleSaveExpressionRule = async () => {
    setSaving(true);
    setSaveResult(null);
    try {
      const ast = await parseExpressionWasm(expressionText) as { root: unknown };
      const desc = await apiClient<{ id: string }>('/validation-rule-nodes', {
        method: 'POST',
        body: JSON.stringify({
          bo_name: selectedBOKey,
          name: ruleName,
          severity,
          timing,
          category,
          domain,
          rule_ast: { type: 'expression', root: ast.root },
        }),
      });
      setSaveResult({ id: desc.id });
      await loadFieldsAndRules();
    } catch (err) {
      setSaveResult({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      setSaving(false);
    }
  };

  // Save path 2: expression mode -> calc term. Server-side parsing
  // (POST /calc-terms, internal/handlers/calc_term_handler.go) rather
  // than posting the client-parsed AST - the server is the single place
  // that validates and stores rule_ast, so a client/server parser
  // disagreement can't silently save something that evaluates
  // differently than it was authored.
  const handleSaveCalcTerm = async () => {
    setCalcTermSaving(true);
    setCalcTermSaveResult(null);
    try {
      const desc = await apiClient<{ id: string }>('/calc-terms', {
        method: 'POST',
        body: JSON.stringify({
          bo_name: selectedBOKey,
          name: ruleName,
          expression: expressionText,
        }),
      });
      setCalcTermSaveResult({ id: desc.id });
    } catch (err) {
      setCalcTermSaveResult({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      setCalcTermSaving(false);
    }
  };

  // "Does this pushdown, and to what?" - compiles via the real backend
  // resolver (BO field bindings -> physical columns), the same chain
  // GenerateDDL uses for a saved pre-aggregation - not a client-side
  // guess at column names.
  const handlePreviewSQL = async () => {
    setSqlPreviewLoading(true);
    setSqlPreview(null);
    try {
      const res = await apiClient<{ sql: string }>('/calc-terms/preview-sql', {
        method: 'POST',
        body: JSON.stringify({ bo_name: selectedBOKey, expression: expressionText }),
      });
      setSqlPreview({ sql: res.sql });
    } catch (err) {
      setSqlPreview({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      setSqlPreviewLoading(false);
    }
  };

  const handleSave = async () => {
    setSaving(true);
    setSaveResult(null);
    try {
      const ruleAst = toRuleNode(rule);
      const desc = await apiClient<{ id: string }>('/validation-rule-nodes', {
        method: 'POST',
        body: JSON.stringify({
          bo_name: selectedBOKey,
          name: ruleName,
          severity,
          timing,
          category,
          rule_ast: ruleAst,
        }),
      });
      setSaveResult({ id: desc.id });
      await loadFieldsAndRules();
    } catch (err) {
      setSaveResult({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      setSaving(false);
    }
  };

  const codeBg = (theme: import('@mui/material/styles').Theme) =>
    theme.palette.mode === 'dark' ? alpha(theme.palette.common.white, 0.06) : theme.palette.grey[50];

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: 'background.default', pb: 6 }}>
    <Container maxWidth="lg" sx={{ py: 4 }}>
      {/* Page header - matches the icon+title+subtitle language used by
          BusinessObjectDetailsPage/components/PageHeader.tsx elsewhere in
          the app, plus the BO badge so it's always visible which object
          every panel below is scoped to. */}
      <Stack direction={{ xs: 'column', sm: 'row' }} justifyContent="space-between" alignItems={{ xs: 'flex-start', sm: 'center' }} spacing={2} sx={{ mb: 4 }}>
        <Stack direction="row" spacing={2} alignItems="center">
          <Avatar variant="rounded" sx={{ bgcolor: 'primary.main', width: 48, height: 48 }}>
            <RuleIcon />
          </Avatar>
          <Box>
            <Stack direction="row" spacing={1.5} alignItems="center">
              <Typography variant="h4" sx={{ fontWeight: 900 }}>Rule Builder</Typography>
              {selectedBOKey && <Chip label={selectedBOKey} size="small" color="primary" variant="outlined" />}
            </Stack>
            <Typography variant="body2" color="text.secondary">
              Author validation rules and calc terms with nested conditions, cross-entity traversal, and type-aware operators.
            </Typography>
          </Box>
        </Stack>
      </Stack>

      {/* Primary authoring card - the one thing on this page that isn't
          secondary/supporting, so it gets a visible border instead of
          living at the same flat elevation as everything below it. */}
      <Paper
        variant="outlined"
        sx={{ p: 3, mb: 3, borderColor: 'primary.main', borderWidth: 1.5, borderRadius: 2 }}
      >
        <Stack direction="row" spacing={2} alignItems="center" sx={{ mb: 2.5 }} flexWrap="wrap" useFlexGap>
          <FormControl size="small" sx={{ minWidth: 240 }}>
            <InputLabel id="bo-select-label">Business Object</InputLabel>
            <Select
              labelId="bo-select-label"
              label="Business Object"
              value={selectedBOKey}
              onChange={(e) => setSelectedBOKey(e.target.value)}
              startAdornment={<StorageIcon fontSize="small" sx={{ mr: 1, color: 'text.secondary' }} />}
            >
              {businessObjects.map((bo) => (
                <MenuItem key={bo.key} value={bo.key}>{bo.name || bo.key}</MenuItem>
              ))}
            </Select>
          </FormControl>
          {loadingFields && <CircularProgress size={20} />}
          {!loadingFields && fields.length === 0 && selectedBOKey && (
            <Alert severity="warning" sx={{ py: 0 }} icon={<WarningAmberIcon fontSize="small" />}>
              No physical fields found for this BO's driver_table_name.
            </Alert>
          )}
          <ToggleButtonGroup
            size="small"
            value={mode}
            exclusive
            onChange={(_, v) => v && setMode(v)}
            sx={{ ml: 'auto' }}
          >
            <ToggleButton value="structured" sx={{ px: 2, gap: 0.75 }}>
              <ChecklistIcon fontSize="small" /> Structured Conditions
            </ToggleButton>
            <ToggleButton value="expression" sx={{ px: 2, gap: 0.75 }}>
              <FunctionsIcon fontSize="small" /> Expression
            </ToggleButton>
          </ToggleButtonGroup>
        </Stack>

        {mode === 'expression' && (
          <Box>
            <Typography variant="body2" color="text.secondary" sx={{ mb: 1.5 }}>
              Author a FuncCall/Expression tree directly - the structured
              builder can't produce these. Start typing a function name
              (e.g. "XI") for autocomplete with its signature and
              wasm-only/pushdown badge.
            </Typography>
            <Paper variant="outlined" sx={{ mb: 1.5, borderRadius: 1.5, overflow: 'hidden' }}>
              <Editor
                height="140px"
                language={UISCE_EXPRESSION_LANGUAGE}
                value={expressionText}
                onChange={(v) => setExpressionText(v ?? '')}
                theme="vs-light"
                beforeMount={(monaco) => { void registerUisceExpressionLanguage(monaco); }}
                onMount={handleExpressionEditorMount}
                options={{
                  minimap: { enabled: false }, fontSize: 14, lineNumbers: 'off', folding: false, scrollBeyondLastLine: false,
                  quickSuggestions: { other: true, comments: false, strings: false },
                  quickSuggestionsDelay: 10,
                  suggestOnTriggerCharacters: true,
                  parameterHints: { enabled: true },
                  wordBasedSuggestions: false,
                  suggest: { showFunctions: true, showFields: true, snippetsPreventQuickSuggestions: false },
                }}
              />
            </Paper>
            {exprParseError ? (
              <Alert severity="error" sx={{ mb: 1 }}>
                Syntax error at position {exprParseError.pos}: {exprParseError.message}
              </Alert>
            ) : (
              <Alert severity="success" sx={{ mb: 1 }}>Parses cleanly.</Alert>
            )}
          </Box>
        )}

        {mode === 'structured' && entities.length > 0 && (
          <AdvancedConditionBuilder
            // AdvancedConditionBuilder seeds its own currentEntity state
            // from the primaryEntity prop via useState(primaryEntity) once,
            // at mount, with no effect resyncing it when the prop changes
            // later - switching the BO dropdown here changed `entities`/
            // `primaryEntity` without the field picker noticing, so it kept
            // looking up fields on the old entity name (which no longer
            // exists in the new single-entity `entities` array) and showed
            // "No fields found". Forcing a remount on BO change is the
            // correct fix scoped to this page - fixing the shared
            // component's internal state sync is a separate, larger change
            // with its own other consumers to consider.
            key={selectedBOKey}
            value={rule}
            onChange={setRule}
            entities={entities}
            primaryEntity={selectedBOKey}
            enableCrossEntity={false}
            enableDragDrop={true}
            showValidation={true}
          />
        )}
      </Paper>

      <Paper sx={{ p: 3, mb: 3, borderRadius: 2 }}>
        <SectionHeader icon={<SaveIcon fontSize="small" />} title="Save" subtitle="Persist this rule against the selected business object." />
        <Stack direction="row" spacing={2} flexWrap="wrap" useFlexGap sx={{ mb: 2.5 }}>
          <TextField label="Rule name" value={ruleName} onChange={(e) => setRuleName(e.target.value)} size="small" sx={{ minWidth: 260 }} />
          <FormControl size="small" sx={{ minWidth: 150 }}>
            <InputLabel id="severity-label">Severity</InputLabel>
            <Select labelId="severity-label" label="Severity" value={severity} onChange={(e) => setSeverity(e.target.value)}>
              <MenuItem value="BLOCK">
                <Stack direction="row" spacing={1} alignItems="center"><BlockIcon fontSize="small" color="error" /> BLOCK</Stack>
              </MenuItem>
              <MenuItem value="WARN">
                <Stack direction="row" spacing={1} alignItems="center"><WarningAmberIcon fontSize="small" color="warning" /> WARN</Stack>
              </MenuItem>
            </Select>
          </FormControl>
          <FormControl size="small" sx={{ minWidth: 160 }}>
            <InputLabel id="timing-label">Timing</InputLabel>
            <Select labelId="timing-label" label="Timing" value={timing} onChange={(e) => setTiming(e.target.value)}>
              <MenuItem value="pre_write">pre_write</MenuItem>
              <MenuItem value="reconcile">reconcile</MenuItem>
            </Select>
          </FormControl>
          <FormControl size="small" sx={{ minWidth: 150 }}>
            <InputLabel id="domain-label">Domain</InputLabel>
            <Select labelId="domain-label" label="Domain" value={domain} onChange={(e) => setDomain(e.target.value)}>
              <MenuItem value="validation">validation</MenuItem>
              <MenuItem value="mdm">mdm</MenuItem>
              <MenuItem value="compliance">compliance</MenuItem>
            </Select>
          </FormControl>
          <TextField label="Category" value={category} onChange={(e) => setCategory(e.target.value)} size="small" sx={{ minWidth: 160 }} disabled={mode === 'expression'} />
        </Stack>

        {mode === 'structured' && (
          <Button
            variant="contained"
            startIcon={saving ? <CircularProgress size={16} color="inherit" /> : <SaveIcon />}
            onClick={handleSave}
            disabled={saving || !ruleName || !selectedBOKey}
          >
            {saving ? 'Saving...' : 'Save Rule'}
          </Button>
        )}

        {mode === 'expression' && (
          <Stack direction="row" spacing={2} flexWrap="wrap" useFlexGap>
            <Button
              variant="contained"
              startIcon={saving ? <CircularProgress size={16} color="inherit" /> : <SaveIcon />}
              onClick={handleSaveExpressionRule}
              disabled={saving || !ruleName || !selectedBOKey || !!exprParseError}
            >
              {saving ? 'Saving...' : 'Save as Validation Rule'}
            </Button>
            <Button
              variant="outlined"
              startIcon={<FunctionsIcon />}
              onClick={handleSaveCalcTerm}
              disabled={calcTermSaving || !ruleName || !selectedBOKey || !!exprParseError}
            >
              {calcTermSaving ? 'Saving...' : 'Save as Calc Term'}
            </Button>
            <Button
              variant="outlined"
              startIcon={<CodeIcon />}
              onClick={handlePreviewSQL}
              disabled={sqlPreviewLoading || !selectedBOKey || !!exprParseError}
            >
              {sqlPreviewLoading ? 'Compiling...' : 'Preview SQL'}
            </Button>
          </Stack>
        )}

        {saveResult && (
          <Alert severity={saveResult.error ? 'error' : 'success'} sx={{ mt: 2 }}>
            {saveResult.error || `Saved as validation rule ${saveResult.id}`}
          </Alert>
        )}
        {calcTermSaveResult && (
          <Alert severity={calcTermSaveResult.error ? 'error' : 'success'} sx={{ mt: 2 }}>
            {calcTermSaveResult.error || `Saved as calc term ${calcTermSaveResult.id}`}
          </Alert>
        )}
        {sqlPreview && (
          <Box sx={{ mt: 2 }}>
            {sqlPreview.error ? (
              <Alert severity="error">{sqlPreview.error}</Alert>
            ) : (
              <Box sx={{ bgcolor: codeBg, p: 2, borderRadius: 1.5, overflow: 'auto', fontFamily: 'monospace', fontSize: 13 }}>
                {sqlPreview.sql}
              </Box>
            )}
          </Box>
        )}

        <Divider sx={{ my: 2.5 }} />
        <Typography variant="subtitle2" sx={{ fontWeight: 700, mb: 1 }}>
          Saved rules for {selectedBOKey || '...'} ({savedRules.length})
        </Typography>
        <Stack spacing={1}>
          {savedRules.map((r) => {
            const meta = SEVERITY_META[r.severity] ?? SEVERITY_META.WARN;
            return (
              <Stack
                key={r.id}
                direction="row"
                spacing={1.5}
                alignItems="center"
                sx={{
                  py: 1, px: 1.5, borderRadius: 1.5,
                  bgcolor: codeBg,
                }}
              >
                <Avatar sx={{ width: 28, height: 28, bgcolor: `${meta.color}.main` }}>{meta.icon}</Avatar>
                <Box sx={{ flex: 1, minWidth: 0 }}>
                  <Typography variant="body2" sx={{ fontWeight: 600 }} noWrap>{r.name}</Typography>
                  <Typography variant="caption" color="text.secondary">
                    {r.severity} · {r.timing}{r.category ? ` · ${r.category}` : ''}
                  </Typography>
                </Box>
              </Stack>
            );
          })}
          {savedRules.length === 0 && (
            <Typography variant="body2" color="text.secondary">No rules saved yet for this BO.</Typography>
          )}
        </Stack>
      </Paper>

      <Paper sx={{ p: 3, mb: 3, borderRadius: 2 }}>
        <SectionHeader
          icon={<WarningAmberIcon fontSize="small" />}
          title="Recent Violations"
          subtitle={`For ${selectedBOKey || '...'}`}
          action={
            <Tooltip title="Refresh">
              <IconButton size="small" onClick={loadFieldsAndRules}><RefreshIcon fontSize="small" /></IconButton>
            </Tooltip>
          }
        />
        <Stack spacing={1}>
          {violations.map((v) => (
            <Box key={v.id} sx={{ py: 1, px: 1.5, borderRadius: 1.5, bgcolor: codeBg }}>
              <Stack direction="row" spacing={1} alignItems="center" flexWrap="wrap" useFlexGap>
                <Typography variant="body2" sx={{ fontWeight: 600 }}>{v.rule_name}</Typography>
                <Chip size="small" label={v.severity} color={v.severity === 'BLOCK' ? 'error' : 'warning'} />
                <Chip size="small" label={v.write_blocked ? 'write blocked' : 'logged only'} variant="outlined" />
                {v.rule_error && <Chip size="small" label="rule error" color="secondary" />}
              </Stack>
              <Typography variant="caption" color="text.secondary">
                record {v.record_id} · {v.created_at}{v.rule_error ? ` · could not evaluate: ${v.message}` : ''}
              </Typography>
            </Box>
          ))}
          {violations.length === 0 && (
            <Typography variant="body2" color="text.secondary">No violations recorded yet for this BO.</Typography>
          )}
        </Stack>
      </Paper>

      {/* Debug/output surfaces - useful, but secondary to the authoring
          flow above, so they're collapsed accordions rather than a Tabs
          panel competing for the same visual weight as Save/Violations. */}
      <Accordion variant="outlined" sx={{ borderRadius: 2, '&:before': { display: 'none' } }}>
        <AccordionSummary expandIcon={<ExpandMoreIcon />}>
          <Stack direction="row" spacing={1} alignItems="center">
            <CodeIcon fontSize="small" color="action" />
            <Typography variant="subtitle2" sx={{ fontWeight: 700 }}>JSON Output</Typography>
          </Stack>
        </AccordionSummary>
        <AccordionDetails>
          <Box sx={{ bgcolor: codeBg, p: 2, borderRadius: 1.5, overflow: 'auto' }}>
            <pre style={{ margin: 0, fontSize: 13 }}>
              {mode === 'expression' ? expressionText : JSON.stringify(rule, null, 2)}
            </pre>
          </Box>
        </AccordionDetails>
      </Accordion>

      <Accordion variant="outlined" sx={{ mt: 1.5, borderRadius: 2, '&:before': { display: 'none' } }}>
        <AccordionSummary expandIcon={<ExpandMoreIcon />}>
          <Stack direction="row" spacing={1} alignItems="center">
            <PlayArrowIcon fontSize="small" color="action" />
            <Typography variant="subtitle2" sx={{ fontWeight: 700 }}>Backend Preview</Typography>
          </Stack>
        </AccordionSummary>
        <AccordionDetails>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
            Runs this {mode === 'expression' ? 'expression' : 'rule'} against
            internal/rules/vm.AdvancedEvaluator via the same rule_engine.wasm
            build the browser live-preview panel uses (see
            frontend/src/rules/wasmRuntime.ts) - the real evaluator, not a claim about it.
          </Typography>
          <TextField
            label="Sample data (JSON)"
            value={contextJson}
            onChange={(e) => setContextJson(e.target.value)}
            multiline
            minRows={4}
            fullWidth
            sx={{ mb: 2, fontFamily: 'monospace' }}
          />
          <Button variant="contained" startIcon={<PlayArrowIcon />} onClick={runBackendPreview} disabled={evaluating}>
            {evaluating ? 'Evaluating...' : 'Evaluate against sample data'}
          </Button>
          {evalResult && (
            <Box sx={{ mt: 2 }}>
              {evalResult.error ? (
                <Alert severity="error">{evalResult.error}</Alert>
              ) : evalResult.resultType === 'number' ? (
                <Alert severity="success">Result: {evalResult.result}</Alert>
              ) : (
                <Alert severity={evalResult.result ? 'success' : 'warning'}>
                  Result: {evalResult.result ? 'PASS' : 'FAIL'}
                </Alert>
              )}
            </Box>
          )}
        </AccordionDetails>
      </Accordion>
    </Container>
    </Box>
  );
};

export default AdvancedRuleBuilderPage;
