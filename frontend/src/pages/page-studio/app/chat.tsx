import React, { useEffect, useRef, useState } from 'react';
import {
  Alert, Box, Button, Chip, CircularProgress, IconButton, Paper, Stack, TextField, Typography,
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import SendIcon from '@mui/icons-material/Send';
import type { Action, Binding, ConditionNode, TextSpec } from './appModel';
import { resolve, text, type Scope } from './bindings';
import { useCondition } from './conditions';
import { useAppRuntime } from './AppRuntime';
import { PageIcon } from './icons';

/**
 * A conversation panel (an assistant that drafts or explains): messages held
 * in a page variable, starters when it is empty, and a composer. Sending
 * appends the user's message and runs onSend with {{text}} and {{messages}}
 * (the conversation before it) - usually an operation that returns the next
 * messages. An assistant message may carry detail lines, a warning and an
 * action (Apply), which marks itself done once it has run.
 */

export interface ChatMessage {
  role: 'user' | 'assistant';
  text: string;
  error?: boolean;
  /** The message's action has been taken (set by the widget). */
  done?: boolean;
  [k: string]: unknown;
}

export interface ChatProps {
  /** The page variable holding the messages. */
  variable: string;
  title?: TextSpec;
  icon?: string;
  /** Shown while the conversation is empty, above the starters. */
  intro?: TextSpec;
  /** Suggestions that send themselves: a list of texts, or a binding to one. */
  starters?: TextSpec[] | Binding;
  placeholder?: TextSpec;
  busyText?: TextSpec;
  /** Runs on send with {{text}} and {{messages}}. */
  onSend?: Action[];
  /** An assistant message's extras; templates see {{message}} and {{index}}. */
  message?: {
    details?: Binding;
    warning?: Binding;
    warningTitle?: TextSpec;
    action?: { label: TextSpec; doneLabel?: TextSpec; visibleWhen?: ConditionNode; onClick: Action[] };
  };
  /** Makes the panel closable. */
  onClose?: Action[];
}

export const DEFAULT_CHAT_PROPS: ChatProps = {
  variable: 'messages', title: 'Assistant', icon: 'assistant', intro: 'Ask a question.', starters: [], placeholder: 'Type a message', onSend: [],
};

const lines = (v: unknown): string[] => (Array.isArray(v) ? v.map((x) => (x && typeof x === 'object' ? JSON.stringify(x) : String(x))) : v ? [String(v)] : []);

function MessageAction({ a, scope, done, onDone }: { a: NonNullable<NonNullable<ChatProps['message']>['action']>; scope: Scope; done: boolean; onDone: () => void }) {
  const { runActions } = useAppRuntime();
  const shown = useCondition(a.visibleWhen, scope, true);
  const [busy, setBusy] = useState(false);
  if (a.visibleWhen && !shown) return null;
  const click = async () => {
    setBusy(true);
    try { await runActions(a.onClick, scope); onDone(); } finally { setBusy(false); }
  };
  return (
    <Button size="small" variant={done ? 'outlined' : 'contained'} disabled={done || busy} onClick={() => void click()}>
      {done ? text(a.doneLabel ?? 'Done', scope) : text(a.label, scope)}
    </Button>
  );
}

export function Chat({ p, scope }: { p: ChatProps; scope: Scope }) {
  const { setVariable, runActions } = useAppRuntime();
  const raw = (scope.vars as Record<string, unknown> | undefined)?.[p.variable];
  const messages = (Array.isArray(raw) ? raw : []) as ChatMessage[];
  const [draft, setDraft] = useState('');
  const [busy, setBusy] = useState(false);
  const end = useRef<HTMLDivElement>(null);
  // Block body: scrollIntoView may return a Promise, which must not become the effect's cleanup.
  useEffect(() => { end.current?.scrollIntoView?.({ behavior: 'smooth' }); }, [messages.length, busy]);

  const send = async (msg: string) => {
    if (!msg.trim() || busy) return;
    setDraft('');
    setVariable(p.variable, [...messages, { role: 'user', text: msg }]);
    setBusy(true);
    try { await runActions(p.onSend, { text: msg, messages }); } finally { setBusy(false); }
  };
  const markDone = (i: number) => {
    const now = ((scope.vars as Record<string, unknown>)?.[p.variable] ?? messages) as ChatMessage[];
    setVariable(p.variable, now.map((m, j) => (j === i ? { ...m, done: true } : m)));
  };
  const starters = Array.isArray(p.starters) ? p.starters.map((s) => text(s, scope)) : lines(resolve(p.starters, scope));

  return (
    <Paper square elevation={0} sx={{ height: '100%', minHeight: 320, borderLeft: 1, borderColor: 'divider', display: 'flex', flexDirection: 'column' }}>
      <Stack direction="row" alignItems="center" spacing={1} sx={{ px: 2, py: 1, borderBottom: 1, borderColor: 'divider' }}>
        {p.icon && <PageIcon name={p.icon} color="secondary" fontSize="small" />}
        <Typography variant="subtitle1" sx={{ flex: 1, fontWeight: 700 }}>{text(p.title ?? '', scope)}</Typography>
        {p.onClose?.length ? (
          <IconButton size="small" onClick={() => void runActions(p.onClose, scope)} aria-label="close"><CloseIcon fontSize="small" /></IconButton>
        ) : null}
      </Stack>
      <Box sx={{ flex: 1, overflowY: 'auto', p: 2 }}>
        {messages.length === 0 && (
          <Stack spacing={1}>
            {p.intro && <Typography variant="body2" color="text.secondary">{text(p.intro, scope)}</Typography>}
            {starters.map((s) => (
              <Chip key={s} label={s} onClick={() => void send(s)} sx={{ height: 'auto', '& .MuiChip-label': { whiteSpace: 'normal', py: 0.75 } }} />
            ))}
          </Stack>
        )}
        <Stack spacing={1.5}>
          {messages.map((m, i) => {
            const s: Scope = { ...scope, message: m, index: i };
            if (m.role === 'user') {
              return (
                <Box key={i} sx={{ alignSelf: 'flex-end', maxWidth: '85%' }}>
                  <Paper sx={{ px: 1.5, py: 1, bgcolor: 'primary.main', color: 'primary.contrastText' }}><Typography variant="body2">{m.text}</Typography></Paper>
                </Box>
              );
            }
            const details = p.message?.details !== undefined ? lines(resolve(p.message.details, s)) : [];
            const warning = p.message?.warning !== undefined ? lines(resolve(p.message.warning, s)) : [];
            return (
              <Box key={i} sx={{ alignSelf: 'stretch' }}>
                <Paper variant="outlined" sx={{ px: 1.5, py: 1 }}>
                  <Typography variant="body2" color={m.error ? 'error' : undefined} sx={{ whiteSpace: 'pre-wrap' }}>{m.text}</Typography>
                  {(details.length > 0 || warning.length > 0 || p.message?.action) && !m.error && (
                    <Stack spacing={1} sx={{ mt: details.length || warning.length ? 1 : 0 }}>
                      {details.map((d, k) => <Typography key={k} variant="caption" sx={{ display: 'block' }}>• {d}</Typography>)}
                      {warning.length > 0 && (
                        <Alert severity="warning" sx={{ py: 0 }}>
                          {p.message?.warningTitle ? text(p.message.warningTitle, s) : ''}
                          {warning.map((x, k) => <Typography key={k} variant="caption" sx={{ display: 'block' }}>{x}</Typography>)}
                        </Alert>
                      )}
                      {p.message?.action && <MessageAction a={p.message.action} scope={s} done={!!m.done} onDone={() => markDone(i)} />}
                    </Stack>
                  )}
                </Paper>
              </Box>
            );
          })}
          {busy && (
            <Stack direction="row" spacing={1} alignItems="center">
              <CircularProgress size={16} /><Typography variant="body2" color="text.secondary">{text(p.busyText ?? 'Working it out…', scope)}</Typography>
            </Stack>
          )}
        </Stack>
        <div ref={end} />
      </Box>
      <Stack direction="row" spacing={1} sx={{ p: 1.5, borderTop: 1, borderColor: 'divider' }}>
        <TextField fullWidth size="small" multiline maxRows={4} placeholder={p.placeholder ? text(p.placeholder, scope) : undefined}
          value={draft} onChange={(e) => setDraft(e.target.value)}
          onKeyDown={(e) => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); void send(draft); } }} />
        <IconButton color="primary" disabled={!draft.trim() || busy} onClick={() => void send(draft)} aria-label="send"><SendIcon /></IconButton>
      </Stack>
    </Paper>
  );
}
