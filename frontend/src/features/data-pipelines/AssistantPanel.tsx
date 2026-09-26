import React, { useEffect, useRef, useState } from 'react';
import {
  Alert, Box, Button, Chip, CircularProgress, IconButton, Paper, Stack, TextField, Typography,
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import SendIcon from '@mui/icons-material/Send';
import AutoAwesomeIcon from '@mui/icons-material/AutoAwesome';
import apiClient from '../../utils/apiClient';
import type { PreviewResult, Spec } from './api';

interface Proposal {
  reply: string;
  spec?: Spec;
  changes?: string[];
  issues?: { node_id?: string; message: string }[];
}
interface Message { role: 'user' | 'assistant'; text: string; proposal?: Proposal; applied?: boolean; error?: boolean }

const STARTERS = [
  'Load the uploaded FactSet file into the Fund business object, updating funds that already exist',
  'Add a step that applies the fund validation rules before writing',
  'Load this file into a staging table instead of a business object',
  'Why were rows rejected in the preview?',
];

export function AssistantPanel({ spec, selectedNodeId, preview, onApply, onClose }: {
  spec: Spec;
  selectedNodeId: string | null;
  preview: PreviewResult | null;
  onApply: (spec: Spec, focusNodeId?: string) => void;
  onClose: () => void;
}) {
  const [messages, setMessages] = useState<Message[]>([]);
  const [text, setText] = useState('');
  const [busy, setBusy] = useState(false);
  const end = useRef<HTMLDivElement>(null);
  // Block body: scrollIntoView may return a Promise, which must not become the effect's cleanup.
  useEffect(() => { end.current?.scrollIntoView({ behavior: 'smooth' }); }, [messages, busy]);

  const send = async (msg: string) => {
    if (!msg.trim() || busy) return;
    const history = messages.map(m => ({ role: m.role, text: m.text }));
    setMessages(m => [...m, { role: 'user', text: msg }]);
    setText('');
    setBusy(true);
    try {
      const res = await apiClient<Proposal>('/api/data-pipelines/assist', {
        method: 'POST',
        body: JSON.stringify({
          message: msg, spec, selected_node_id: selectedNodeId ?? undefined, history,
          preview_rejects: preview?.rejects.slice(0, 30).map(r => `row ${r.row} (${r.node_id}): ${r.reason}`),
        }),
        headers: { 'Content-Type': 'application/json' },
      });
      setMessages(m => [...m, { role: 'assistant', text: res.reply, proposal: res.spec ? res : undefined }]);
    } catch (e) {
      setMessages(m => [...m, { role: 'assistant', text: friendly((e as Error).message), error: true }]);
    } finally {
      setBusy(false);
    }
  };

  const apply = (i: number) => {
    const p = messages[i].proposal;
    if (!p?.spec) return;
    const firstChanged = p.spec.nodes.find(n => !spec.nodes.some(o => o.id === n.id))?.id;
    onApply(p.spec, firstChanged);
    setMessages(m => m.map((x, j) => (j === i ? { ...x, applied: true } : x)));
  };

  return (
    <Paper square elevation={0} sx={{ width: { xs: 300, lg: 380 }, flexShrink: 0, borderLeft: 1, borderColor: 'divider', display: 'flex', flexDirection: 'column' }}>
      <Stack direction="row" alignItems="center" spacing={1} sx={{ px: 2, py: 1, borderBottom: 1, borderColor: 'divider' }}>
        <AutoAwesomeIcon color="secondary" fontSize="small" />
        <Typography variant="subtitle1" sx={{ flex: 1, fontWeight: 700 }}>Pipeline assistant</Typography>
        <IconButton size="small" onClick={onClose} aria-label="close assistant"><CloseIcon fontSize="small" /></IconButton>
      </Stack>
      <Box sx={{ flex: 1, overflowY: 'auto', p: 2 }}>
        {messages.length === 0 && (
          <Stack spacing={1}>
            <Typography variant="body2" color="text.secondary">
              Describe what you want to load or change. I only use business objects, rules, files and tables that exist for you,
              and nothing changes until you press <b>Apply</b>.
            </Typography>
            {STARTERS.map(s => <Chip key={s} label={s} onClick={() => send(s)} sx={{ height: 'auto', '& .MuiChip-label': { whiteSpace: 'normal', py: 0.75 } }} />)}
          </Stack>
        )}
        <Stack spacing={1.5}>
          {messages.map((m, i) => (
            <Box key={i} sx={{ alignSelf: m.role === 'user' ? 'flex-end' : 'stretch', maxWidth: m.role === 'user' ? '85%' : '100%' }}>
              {m.role === 'user' ? (
                <Paper sx={{ px: 1.5, py: 1, bgcolor: 'primary.main', color: 'primary.contrastText' }}><Typography variant="body2">{m.text}</Typography></Paper>
              ) : (
                <Paper variant="outlined" sx={{ px: 1.5, py: 1 }}>
                  <Typography variant="body2" color={m.error ? 'error' : undefined} sx={{ whiteSpace: 'pre-wrap' }}>{m.text}</Typography>
                  {m.proposal && (
                    <Stack spacing={1} sx={{ mt: 1 }}>
                      {(m.proposal.changes ?? []).map((c, k) => <Typography key={k} variant="caption" sx={{ display: 'block' }}>• {c}</Typography>)}
                      {(m.proposal.issues ?? []).length > 0 && (
                        <Alert severity="warning" sx={{ py: 0 }}>
                          Still needs attention:
                          {m.proposal.issues!.map((x, k) => <Typography key={k} variant="caption" sx={{ display: 'block' }}>{x.node_id ? `${x.node_id}: ` : ''}{x.message}</Typography>)}
                        </Alert>
                      )}
                      <Button size="small" variant={m.applied ? 'outlined' : 'contained'} disabled={m.applied} onClick={() => apply(i)}>
                        {m.applied ? 'Applied' : 'Apply to canvas'}
                      </Button>
                    </Stack>
                  )}
                </Paper>
              )}
            </Box>
          ))}
          {busy && <Stack direction="row" spacing={1} alignItems="center"><CircularProgress size={16} /><Typography variant="body2" color="text.secondary">Working it out…</Typography></Stack>}
        </Stack>
        <div ref={end} />
      </Box>
      <Stack direction="row" spacing={1} sx={{ p: 1.5, borderTop: 1, borderColor: 'divider' }}>
        <TextField
          fullWidth size="small" multiline maxRows={4} placeholder="e.g. only load funds with AUM over 1m"
          value={text} onChange={e => setText(e.target.value)}
          onKeyDown={e => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); send(text); } }}
        />
        <IconButton color="primary" disabled={!text.trim() || busy} onClick={() => send(text)} aria-label="send"><SendIcon /></IconButton>
      </Stack>
    </Paper>
  );
}

function friendly(msg: string): string {
  if (msg.includes('503')) return 'The AI assistant is not set up here yet - an administrator can configure it under System > LLM Config.';
  return `Sorry, that did not work: ${msg.replace(/^API Error: \d+ [^-]*- /, '')}`;
}
