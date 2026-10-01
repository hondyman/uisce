import React, { useMemo } from 'react';
import {
  Autocomplete, Box, Button, Chip, IconButton, MenuItem, Paper, Stack, TextField, ToggleButton, ToggleButtonGroup, Tooltip, Typography,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import DeleteIcon from '@mui/icons-material/DeleteOutline';
import AccountTreeIcon from '@mui/icons-material/AccountTree';
import type { ConditionNode } from './appModel';
import { useAppRuntime } from './AppRuntime';
import { useCondition } from './conditions';
import { BindingPicker, browsableScope } from './bindingPicker';
import {
  TYPE_LABEL, coerceValue, describeCondition, fieldTypeOf, fits, leavesOf, operatorInfo, operatorsFor, shapeProblems, showValue, simplify,
  type ConditionProblem, type FieldType, type Group, type Leaf,
} from './conditionOps';

/**
 * The condition builder: what shows, hides, enables or opens a thing, as a
 * sentence you build rather than JSON you write. Each condition picks a
 * field from the page's live data (with its kind read from the value it
 * holds), then only the operators that fit that kind, then the value the
 * operator takes (a switch, a number, a list, a range). Conditions nest in
 * all/any groups. Problems show beside the condition they belong to, and the
 * whole condition reads back in words with whether it holds right now.
 */

/** What the builder knows about the page, to type fields and flag ones that do not exist. */
interface Context {
  variables: Set<string>;
  queries: Set<string>;
  /** Roots that exist only where the setting is used (row, form, item...). */
  roots: Set<string>;
  typeOf: (path: string) => FieldType;
}

const valueAt = (root: unknown, path: string): unknown =>
  path.split('.').reduce<unknown>((v, k) => (v === null || v === undefined ? undefined : (v as Record<string, unknown>)[k]), root);

function usePageContext(paths: string[]): Context {
  const { scope } = useAppRuntime();
  const live = useMemo(() => browsableScope(scope), [scope]);
  return useMemo(() => ({
    variables: new Set(paths.filter((p) => p.startsWith('vars.')).map((p) => p.split('.')[1])),
    queries: new Set(paths.filter((p) => p.startsWith('queries.')).map((p) => p.split('.')[1])),
    roots: new Set(paths.map((p) => p.split('.')[0]).filter((r) => !['vars', 'queries', 'route'].includes(r))),
    typeOf: (path) => fieldTypeOf(valueAt(live, path)),
  }), [paths, live]);
}

/** Problems with a field that no operator or value can fix: a variable or query the page does not declare. */
function fieldProblems(leaf: Leaf, ctx: Context): ConditionProblem[] {
  const [root, name] = leaf.field.split('.');
  if (root === 'vars' && name && !ctx.variables.has(name)) return [{ severity: 'error', message: `The page has no variable "${name}".` }];
  if (root === 'queries' && name && !ctx.queries.has(name)) return [{ severity: 'error', message: `The page has no query "${name}".` }];
  if (root && !['vars', 'queries', 'route'].includes(root) && !ctx.roots.has(root)) {
    return [{ severity: 'warning', message: `"${root}" is not available where this setting is used.` }];
  }
  return [];
}

function problemsOf(leaf: Leaf, ctx: Context, type: FieldType): ConditionProblem[] {
  const out = [...fieldProblems(leaf, ctx), ...shapeProblems(leaf)];
  const op = operatorInfo(leaf.operator);
  if (op && !fits(op, type)) out.push({ severity: 'warning', message: `"${op.label}" is meant for ${op.fits === 'any' ? 'any' : op.fits.map((t) => TYPE_LABEL[t]).join(' or ')} fields, and this one is ${TYPE_LABEL[type]}.` });
  return out;
}

/** The value box(es) for the chosen operator. */
function ValueInputs({ leaf, type, onChange }: { leaf: Leaf; type: FieldType; onChange: (l: Leaf) => void }) {
  const op = operatorInfo(leaf.operator);
  if (!op || op.kind === 'none') return null;
  if (op.kind === 'list') {
    const list = Array.isArray(leaf.value) ? leaf.value.map(String) : leaf.value ? [String(leaf.value)] : [];
    return (
      <Autocomplete multiple freeSolo size="small" options={[]} value={list} sx={{ flex: 1, minWidth: 160 }}
        onChange={(_, v) => onChange({ ...leaf, value: v.map((x) => coerceValue(String(x), 'one', type)) })}
        renderTags={(vals, getTagProps) => vals.map((x, i) => <Chip size="small" label={x} {...getTagProps({ index: i })} key={`${x}${i}`} />)}
        renderInput={(p) => <TextField {...p} label="Values" placeholder="type, then Enter" />} />
    );
  }
  if (op.kind === 'range') {
    return (
      <Stack direction="row" spacing={1} sx={{ flex: 1 }}>
        <TextField size="small" type="number" label="From" value={showValue(leaf.value)} onChange={(e) => onChange({ ...leaf, value: coerceValue(e.target.value, 'range', type) })} />
        <TextField size="small" type="number" label="To" value={showValue(leaf.secondValue)} onChange={(e) => onChange({ ...leaf, secondValue: coerceValue(e.target.value, 'range', type) })} />
      </Stack>
    );
  }
  if (type === 'yesno' && op.kind === 'one') {
    return (
      <TextField select size="small" label="Value" sx={{ minWidth: 120 }} value={leaf.value === true ? 'true' : leaf.value === false ? 'false' : ''}
        onChange={(e) => onChange({ ...leaf, value: e.target.value === 'true' })}>
        <MenuItem value="true">true</MenuItem><MenuItem value="false">false</MenuItem>
      </TextField>
    );
  }
  const numeric = op.kind === 'number' || type === 'number';
  return (
    <TextField size="small" label="Value" sx={{ flex: 1, minWidth: 120 }} type={numeric ? 'number' : type === 'date' ? 'date' : 'text'}
      InputLabelProps={type === 'date' ? { shrink: true } : undefined}
      value={showValue(leaf.value)} onChange={(e) => onChange({ ...leaf, value: coerceValue(e.target.value, op.kind, type) })} />
  );
}

function LeafBox({ leaf, ctx, paths, onChange, onRemove }: { leaf: Leaf; ctx: Context; paths: string[]; onChange: (l: Leaf) => void; onRemove: () => void }) {
  const type = ctx.typeOf(leaf.field);
  const ops = useMemo(() => {
    const list = operatorsFor(type);
    const current = operatorInfo(leaf.operator);
    return current && !list.includes(current) ? [current, ...list] : list;
  }, [type, leaf.operator]);
  const problems = problemsOf(leaf, ctx, type);
  // Changing the operator keeps what still means something to the new one.
  const setOperator = (operator: string) => {
    const next = operatorInfo(operator);
    const keep = next && next.kind !== 'none' && next.kind === operatorInfo(leaf.operator)?.kind;
    onChange({ type: 'condition', field: leaf.field, operator, ...(keep ? { value: leaf.value, secondValue: leaf.secondValue } : {}) });
  };
  return (
    <Paper variant="outlined" sx={{ p: 1 }}>
      <Stack spacing={1}>
        <Stack direction="row" spacing={0.5} alignItems="flex-start">
          <Autocomplete freeSolo size="small" sx={{ flex: 1, minWidth: 0 }} options={paths} value={leaf.field} inputValue={leaf.field}
            onInputChange={(_, v) => onChange({ ...leaf, field: v })}
            renderInput={(p) => <TextField {...p} label="Check" />} />
          <BindingPicker contextPaths={paths} label="Browse data for the condition" onPick={(path) => onChange({ ...leaf, field: path })} />
          <Tooltip title="Remove this condition"><IconButton size="small" aria-label="Remove condition" onClick={onRemove}><DeleteIcon fontSize="small" /></IconButton></Tooltip>
        </Stack>
        <Stack direction="row" spacing={1} alignItems="flex-start" flexWrap="wrap" useFlexGap>
          <TextField select size="small" label="Is" sx={{ minWidth: 170 }} value={leaf.operator} onChange={(e) => setOperator(e.target.value)}
            helperText={leaf.field ? `a ${TYPE_LABEL[type]} field` : undefined}>
            {ops.map((o) => <MenuItem key={o.value} value={o.value}>{o.label}</MenuItem>)}
            {!operatorInfo(leaf.operator) && <MenuItem value={leaf.operator}>{leaf.operator} (unknown)</MenuItem>}
          </TextField>
          <ValueInputs leaf={leaf} type={type} onChange={onChange} />
        </Stack>
        {problems.map((p, i) => (
          <Typography key={i} variant="caption" color={p.severity === 'error' ? 'error' : 'warning.main'}>{p.message}</Typography>
        ))}
      </Stack>
    </Paper>
  );
}

function GroupBox({ group, depth, ctx, paths, onChange, onRemove }: {
  group: Group; depth: number; ctx: Context; paths: string[]; onChange: (g: Group) => void; onRemove?: () => void;
}) {
  const set = (i: number, c: ConditionNode) => onChange({ ...group, conditions: group.conditions.map((x, j) => (j === i ? c : x)) });
  const drop = (i: number) => onChange({ ...group, conditions: group.conditions.filter((_, j) => j !== i) });
  const blank = (): Leaf => ({ type: 'condition', field: paths.find((p) => p.startsWith('vars.')) ?? paths[0] ?? '', operator: 'is_not_empty' });
  return (
    <Box sx={depth > 0 ? { pl: 1.5, borderLeft: 2, borderColor: 'divider' } : undefined}>
      <Stack direction="row" alignItems="center" spacing={1} sx={{ mb: 1 }}>
        <Typography variant="caption" color="text.secondary">Match</Typography>
        <ToggleButtonGroup size="small" exclusive value={group.operator} onChange={(_, v) => v && onChange({ ...group, operator: v })} aria-label="Match all or any">
          <ToggleButton value="AND" sx={{ py: 0, px: 1 }}>all</ToggleButton>
          <ToggleButton value="OR" sx={{ py: 0, px: 1 }}>any</ToggleButton>
        </ToggleButtonGroup>
        <Typography variant="caption" color="text.secondary" sx={{ flex: 1 }}>of these</Typography>
        {onRemove && <Tooltip title="Remove this group"><IconButton size="small" aria-label="Remove group" onClick={onRemove}><DeleteIcon fontSize="small" /></IconButton></Tooltip>}
      </Stack>
      <Stack spacing={1}>
        {group.conditions.map((c, i) => (c.type === 'condition'
          ? <LeafBox key={i} leaf={c} ctx={ctx} paths={paths} onChange={(l) => set(i, l)} onRemove={() => drop(i)} />
          : <GroupBox key={i} group={c} depth={depth + 1} ctx={ctx} paths={paths} onChange={(g) => set(i, g)} onRemove={() => drop(i)} />))}
        <Stack direction="row" spacing={1}>
          <Button size="small" startIcon={<AddIcon />} onClick={() => onChange({ ...group, conditions: [...group.conditions, blank()] })}>Add condition</Button>
          <Button size="small" startIcon={<AccountTreeIcon />} onClick={() => onChange({ ...group, conditions: [...group.conditions, { type: 'group', operator: group.operator === 'AND' ? 'OR' : 'AND', conditions: [blank()] }] })}>Add group</Button>
        </Stack>
      </Stack>
    </Box>
  );
}

/** The condition read back in words, and whether it holds on the page right now (when it only reads page data). */
function Summary({ value }: { value: ConditionNode | undefined }) {
  const { scope } = useAppRuntime();
  const global = leavesOf(value).every((l) => /^(vars|queries|route)\./.test(l.field));
  const holds = useCondition(global ? value : undefined, scope, false);
  return (
    <Stack direction="row" spacing={1} alignItems="center" flexWrap="wrap" useFlexGap>
      <Typography variant="body2" sx={{ flex: 1, minWidth: 0, fontStyle: value ? 'normal' : 'italic' }} color={value ? 'text.primary' : 'text.secondary'}>
        {value ? `When ${describeCondition(value)}` : 'Always'}
      </Typography>
      {value && global && <Chip size="small" color={holds ? 'success' : 'default'} variant="outlined" label={holds ? 'true now' : 'false now'} />}
    </Stack>
  );
}

export function ConditionBuilder({ label, value, onChange, paths }: {
  label: string; value: ConditionNode | undefined; onChange: (v: ConditionNode | undefined) => void; paths: string[];
}) {
  const ctx = usePageContext(paths);
  // A single condition is edited as a group of one, and stored as that condition again.
  const group: Group = !value ? { type: 'group', operator: 'AND', conditions: [] } : value.type === 'group' ? value : { type: 'group', operator: 'AND', conditions: [value] };
  return (
    <Box>
      <Typography variant="caption" color="text.secondary" component="div" sx={{ mb: 0.5 }}>{label}</Typography>
      <Box sx={{ mb: 1 }}><Summary value={simplify(value)} /></Box>
      <GroupBox group={group} depth={0} ctx={ctx} paths={paths} onChange={(g) => onChange(simplify(g))} />
    </Box>
  );
}
