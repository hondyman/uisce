import React from 'react';
import { useTranslation } from 'react-i18next';
import { Box, Button, Chip, IconButton, Stack, TextField, Tooltip, Typography } from '@mui/material';
import { alpha } from '@mui/material/styles';
import { fmt } from '../../../features/schedules/api';
import type { CellBody, CellSpec, ChipColor, RowButton } from './appModel';
import { resolve, text, type Scope } from './bindings';
import { useCondition } from './conditions';
import { useAppRuntime } from './AppRuntime';
import { PageIcon } from './icons';

const DASH = '—';
const blank = (v: unknown) => v === undefined || v === null || v === '';
const show = (v: unknown) => (blank(v) ? DASH : typeof v === 'object' ? JSON.stringify(v) : String(v));
const numFmt = (v: unknown, lang: string, digits: number) =>
  typeof v === 'number' ? v.toLocaleString(lang, { maximumFractionDigits: digits }) : show(v);

function chipLabel(value: unknown, labelKey: string | undefined, t: (k: string, o?: Record<string, unknown>) => string) {
  const v = show(value);
  return labelKey ? t(`${labelKey}${v}`, { defaultValue: v }) : v;
}

function RowButtonView({ b, scope }: { b: RowButton; scope: Scope }) {
  const { runActions, mode } = useAppRuntime();
  const visible = useCondition(b.visibleWhen, scope, false);
  if (!visible) return null;
  const click = (e: React.MouseEvent) => { e.stopPropagation(); void runActions(b.onClick, scope); };
  if (b.icon) {
    const label = text(b.label, scope);
    return (
      <Tooltip title={label}>
        <span>
          <IconButton size="small" color={b.color === 'inherit' ? 'default' : b.color ?? 'default'} disabled={mode === 'design'} onClick={click} aria-label={label}>
            <PageIcon name={b.icon} fontSize="small" />
          </IconButton>
        </span>
      </Tooltip>
    );
  }
  return (
    <Button size="small" variant={b.variant ?? 'text'} color={b.color ?? 'primary'} disabled={mode === 'design'} onClick={click}>
      {text(b.label, scope)}
    </Button>
  );
}

function Caption({ c, scope }: { c: NonNullable<Extract<CellBody, { kind: 'actions' }>['caption']>; scope: Scope }) {
  const visible = useCondition(c.visibleWhen, scope, false);
  return visible ? <Typography variant="caption" color="text.secondary">{text(c.text, scope)}</Typography> : null;
}

function InputCell({ spec, scope }: { spec: Extract<CellBody, { kind: 'input' }>; scope: Scope }) {
  const rowState = (scope.rowState ?? {}) as Record<string, unknown>;
  const setRowState = scope.setRowState as ((name: string, v: unknown) => void) | undefined;
  return (
    <TextField size="small" placeholder={spec.placeholder ? text(spec.placeholder, scope) : undefined} value={rowState[spec.name] ?? ''}
      onClick={(e) => e.stopPropagation()} onChange={(e) => setRowState?.(spec.name, e.target.value)} />
  );
}

/** One table cell, rendered from its spec against the row's scope ({...page, row, rowState}). */
export function Cell({ spec, scope }: { spec: CellSpec; scope: Scope }) {
  return spec.visibleWhen ? <GatedCell spec={spec} scope={scope} /> : <CellBodyView spec={spec} scope={scope} />;
}

function GatedCell({ spec, scope }: { spec: CellSpec; scope: Scope }) {
  const visible = useCondition(spec.visibleWhen, scope, false);
  return visible ? <CellBodyView spec={spec} scope={scope} /> : null;
}

function CellBodyView({ spec, scope }: { spec: CellBody; scope: Scope }) {
  const { t, i18n } = useTranslation();
  const { runActions } = useAppRuntime();
  const lang = i18n.language;
  switch (spec.kind) {
    case 'text': {
      const v = spec.text !== undefined ? text(spec.text, scope) : resolve(spec.value, scope);
      if (spec.caption && blank(v)) return null;
      const tone = spec.tone !== undefined ? String(resolve(spec.tone, scope) ?? '') : '';
      const struck = spec.strike !== undefined && !!resolve(spec.strike, scope) && resolve(spec.strike, scope) !== 'false';
      const tip = spec.tooltip !== undefined ? resolve(spec.tooltip, scope) : undefined;
      const toned = ['success', 'warning', 'error', 'info'].includes(tone);
      const body = (
        <Typography variant={spec.caption ? 'caption' : 'body2'} component={spec.caption || toned ? 'div' : 'span'} fontWeight={spec.bold || toned ? 600 : undefined}
          color={struck ? 'text.secondary' : spec.color ? `${spec.color}.main` : spec.caption ? 'text.secondary' : undefined}
          sx={(th) => ({
            fontFamily: spec.mono ? 'monospace' : undefined, whiteSpace: spec.nowrap ? 'nowrap' : undefined, wordBreak: 'break-word',
            textDecoration: struck ? 'line-through' : undefined,
            ...(toned ? {
              px: 1, py: 0.25, borderRadius: 1,
              bgcolor: alpha(th.palette[tone as 'success'].main, 0.22), outline: `1px solid ${alpha(th.palette[tone as 'success'].main, 0.6)}`,
            } : {}),
          })}>
          {show(v)}
        </Typography>
      );
      return !blank(tip) ? <Tooltip title={String(tip)}><Box component="span">{body}</Box></Tooltip> : body;
    }
    case 'list': {
      const v = resolve(spec.value, scope);
      const items = Array.isArray(v) ? v : [];
      if (items.length === 0) return <Typography variant="body2" color="text.disabled">{spec.empty ? text(spec.empty, scope) : DASH}</Typography>;
      return (
        <Stack direction={spec.direction ?? 'column'} spacing={0.5} useFlexGap flexWrap="wrap">
          {items.map((item, i) => <Cell key={i} spec={spec.item} scope={{ ...scope, item }} />)}
        </Stack>
      );
    }
    case 'diff':
      return <Typography variant="body2"><s>{show(resolve(spec.before, scope))}</s> → <b>{show(resolve(spec.after, scope))}</b></Typography>;
    case 'twoLine': {
      const p = resolve(spec.primary, scope);
      const s2 = spec.secondary !== undefined ? resolve(spec.secondary, scope) : undefined;
      return (
        <>
          <Typography variant="body2" fontWeight={500}>{show(p)}</Typography>
          {!blank(s2) && <Typography variant="caption" color="text.secondary" fontFamily={spec.secondaryMono ? 'monospace' : undefined}>{show(s2)}</Typography>}
        </>
      );
    }
    case 'number': {
      const v = resolve(spec.value, scope);
      const suffix = spec.suffix !== undefined ? resolve(spec.suffix, scope) : undefined;
      return (
        <Typography variant="body2" component="span" sx={{ fontVariantNumeric: 'tabular-nums', whiteSpace: 'nowrap' }}>
          {numFmt(v, lang, spec.digits ?? 4)}{!blank(suffix) && <Typography component="span" variant="caption" color="text.secondary"> {show(suffix)}</Typography>}
        </Typography>
      );
    }
    case 'percent': {
      const v = resolve(spec.value, scope);
      return <>{typeof v === 'number' ? `${Math.round(v * 100)}%` : DASH}</>;
    }
    case 'delta': {
      const v = resolve(spec.value, scope);
      if (typeof v !== 'number') return <Typography variant="body2" color="text.secondary">{DASH}</Typography>;
      const a = Math.abs(v);
      const bad = spec.bad ?? 5;
      const warn = spec.warn ?? 1;
      return (
        <Typography variant="body2" sx={{ fontVariantNumeric: 'tabular-nums', color: a >= bad ? 'error.main' : a >= warn ? 'warning.main' : 'text.primary' }}>
          {numFmt(v, lang, 2)}%
        </Typography>
      );
    }
    case 'datetime': {
      const d = fmt(resolve(spec.value, scope) as string | undefined, lang);
      return spec.caption ? <Typography variant="caption" color="text.secondary" component="div">{d}</Typography> : <>{d}</>;
    }
    case 'chip': {
      const v = resolve(spec.value, scope);
      if (blank(v) && !spec.label) return <>{DASH}</>;
      const label = spec.label ? text(spec.label, scope) : chipLabel(v, spec.labelKey, t);
      const by = spec.colorBy !== undefined ? resolve(spec.colorBy, scope) : v;
      const color: ChipColor = spec.colorMap?.[String(by)] ?? spec.colorMap?.['*'] ?? spec.color ?? 'default';
      const tip = spec.tooltip !== undefined ? resolve(spec.tooltip, scope) : undefined;
      const caption = spec.caption !== undefined ? resolve(spec.caption, scope) : undefined;
      const chip = <Chip size="small" color={color} variant={spec.variant ?? 'filled'} label={label} />;
      return (
        <>
          {!blank(tip) ? <Tooltip title={String(tip)} componentsProps={{ tooltip: { sx: { whiteSpace: 'pre-line' } } }}>{chip}</Tooltip> : chip}
          {!blank(caption) && <Typography component="span" variant="caption" color="text.secondary" sx={{ ml: 0.5 }}>{show(caption)}</Typography>}
        </>
      );
    }
    case 'chips': {
      const v = resolve(spec.value, scope);
      // Entries may carry their own look: {label, color, variant}.
      if (Array.isArray(v) && v.some((x) => x && typeof x === 'object')) {
        return (
          <Stack direction="row" spacing={0.75} flexWrap="wrap" useFlexGap>
            {v.map((x, i) => {
              const o = (x && typeof x === 'object' ? x : { label: x }) as { label?: unknown; color?: ChipColor; variant?: 'filled' | 'outlined'; tooltip?: unknown };
              const chip = <Chip key={i} size="small" color={o.color ?? 'default'} variant={o.variant ?? 'outlined'} label={show(o.label)} />;
              return !blank(o.tooltip) ? <Tooltip key={i} title={String(o.tooltip)}>{chip}</Tooltip> : chip;
            })}
          </Stack>
        );
      }
      const entries: [string, unknown][] = Array.isArray(v)
        ? v.map((x) => [String(x), undefined])
        : v && typeof v === 'object'
          ? (spec.keys ? spec.keys.map((k) => [k, (v as Record<string, unknown>)[k]] as [string, unknown]) : Object.entries(v as Record<string, unknown>))
              .filter(([, n]) => typeof n === 'number' ? n > 0 : !blank(n))
          : [];
      return (
        <Stack direction="row" spacing={0.75} flexWrap="wrap" useFlexGap>
          {entries.map(([k, n]) => (
            <Chip key={k} size="small" variant="outlined" color={spec.colorMap?.[k] ?? spec.colorMap?.['*'] ?? 'default'}
              label={`${spec.labelKey ? t(`${spec.labelKey}${k}`, { defaultValue: k }) : k}${spec.showCount !== false && n !== undefined ? ` ${n}` : ''}`} />
          ))}
        </Stack>
      );
    }
    case 'link': {
      const after = spec.after !== undefined ? resolve(spec.after, scope) : undefined;
      return (
        <>
          <Button size="small" onClick={(e) => { e.stopPropagation(); void runActions(spec.onClick, scope); }}>{text(spec.label, scope)}</Button>
          {!blank(after) && <> {show(after)}</>}
        </>
      );
    }
    case 'actions':
      return (
        <Stack spacing={0.5} alignItems="flex-end">
          {spec.caption && <Caption c={spec.caption} scope={scope} />}
          <Stack direction="row" spacing={1} justifyContent="flex-end">
            {spec.buttons.map((b, i) => <RowButtonView key={i} b={b} scope={scope} />)}
          </Stack>
        </Stack>
      );
    case 'input':
      return <InputCell spec={spec} scope={scope} />;
    default:
      return null;
  }
}
