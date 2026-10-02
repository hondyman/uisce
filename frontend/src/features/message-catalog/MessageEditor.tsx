import React, { useEffect, useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import {
  Alert, Box, Button, Chip, Divider, Drawer, FormControl, IconButton, InputLabel, MenuItem, Select, Stack,
  Tab, Tabs, TextField, ToggleButton, ToggleButtonGroup, Tooltip, Typography,
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import {
  Change, ChangeRequest, Entry, formatMessage, formatWhen, Language, Me, MessageSet, MessageView, msgcatApi,
  placeholders, samePlaceholders, Scope, SEVERITIES, Severity,
} from './api';
import { alertSeverity, CatalogErrorAlert, SeverityChip, StatusChip } from './parts';

interface Draft {
  text: string;
  user_action: string;
  description: string;
}

const EMPTY: Draft = { text: '', user_action: '', description: '' };

function draftOf(e?: Entry): Draft {
  return e ? { text: e.text, user_action: e.user_action ?? '', description: e.description ?? '' } : EMPTY;
}

const same = (a: Draft, b: Draft) =>
  a.text === b.text && a.user_action === b.user_action && a.description === b.description;

interface Props {
  open: boolean;
  onClose: () => void;
  /** The message to edit; undefined creates a new one. */
  message?: MessageView;
  sets: MessageSet[];
  languages: Language[];
  me: Me;
  initialSet?: number;
  initialLanguage: string;
}

export default function MessageEditor({ open, onClose, message, sets, languages, me, initialSet, initialLanguage }: Props) {
  const { t, i18n } = useTranslation();
  const qc = useQueryClient();
  const isNew = !message;
  const [scope, setScope] = useState<Scope>(me.can_edit_tenant ? 'tenant' : 'core');
  const [lang, setLang] = useState(initialLanguage);
  const [setNbr, setSetNbr] = useState<number>(message?.set_nbr ?? initialSet ?? 0);
  const [msgNbr, setMsgNbr] = useState<string>(message ? String(message.message_nbr) : '');
  const [severity, setSeverity] = useState<Severity>(message?.severity ?? 'Error');
  const [drafts, setDrafts] = useState<Record<string, Draft>>({});
  const [reason, setReason] = useState('');
  const [sample, setSample] = useState<string[]>([]);
  const [result, setResult] = useState<'applied' | 'pending' | null>(null);

  // The rows in the chosen scope, which edits are compared against.
  const scopeRows: Record<string, Entry> = useMemo(
    () => (message ? (scope === 'tenant' ? message.tenant : message.core) : {}),
    [message, scope],
  );

  // What the editor starts from, and what an edit is compared against: the
  // scope's own row, or for a tenant without one, the core text it would
  // replace. A language counts as changed only if it differs from this.
  const startingRow = (code: string): Entry | undefined =>
    scopeRows[code] ?? (scope === 'tenant' ? message?.core[code] : undefined);

  useEffect(() => {
    if (!open) return;
    // Start from the scope's own text; a new tenant override starts from
    // the core text it replaces.
    const d: Record<string, Draft> = {};
    for (const l of languages) {
      d[l.code] = draftOf(startingRow(l.code));
    }
    setDrafts(d);
    setSeverity((scope === 'tenant' ? message?.tenant.en?.severity : undefined) ?? message?.severity ?? 'Error');
    setResult(null);
    // Reset only when a different message or scope is opened: a refetch after
    // saving hands us a new message object and must not wipe the result.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, scope, message?.code, languages.length]);

  useEffect(() => {
    if (open) setLang(initialLanguage);
  }, [open, initialLanguage]);

  const history = useQuery({
    queryKey: ['msgcat-message', message?.set_nbr, message?.message_nbr],
    queryFn: () => msgcatApi.message(message!.set_nbr, message!.message_nbr),
    enabled: open && !!message && (me.can_edit_core || me.can_edit_tenant),
  });

  const english = drafts.en?.text ?? '';
  const current = drafts[lang] ?? EMPTY;
  const baseline = (code: string) => draftOf(startingRow(code));
  const changed = languages
    .map((l) => l.code)
    .filter((code) => drafts[code] && !same(drafts[code], baseline(code)) && drafts[code].text.trim() !== '');
  const startEN = startingRow('en');
  const severityChanged = !isNew && !!startEN && severity !== startEN.severity;
  if (severityChanged && !changed.includes('en')) changed.unshift('en');
  changed.sort((a, b) => (a === 'en' ? -1 : b === 'en' ? 1 : 0));

  const mismatched = (code: string) => code !== 'en' && english && drafts[code]?.text && !samePlaceholders(drafts[code].text, english);
  const blocking = changed.some(mismatched) || (isNew && (!setNbr || !Number(msgNbr) || !english.trim()));

  const requiresApproval = scope === 'core' ? me.core_requires_approval : me.tenant_requires_approval !== false;
  const editable = scope === 'core' ? me.can_edit_core : me.can_edit_tenant;
  const setChoices = sets.filter((s) => (scope === 'core' ? s.type !== 'client' || me.can_edit_core : s.type === 'client'));
  const params = placeholders(english);
  // Unfilled parameters stay visible in the preview.
  const previewParams = ['%1', '%2', '%3', '%4', '%5', '%6', '%7', '%8', '%9'].map((p, i) => sample[i] || p);

  const save = useMutation({
    mutationFn: (reqs: ChangeRequest[]) => msgcatApi.propose(reqs),
    onSuccess: (res) => {
      setResult(res.applied ? 'applied' : 'pending');
      qc.invalidateQueries({ queryKey: ['msgcat-messages'] });
      qc.invalidateQueries({ queryKey: ['msgcat-changes'] });
      qc.invalidateQueries({ queryKey: ['msgcat-message'] });
    },
  });

  const submit = () => {
    const set = message?.set_nbr ?? setNbr;
    const number = message?.message_nbr ?? Number(msgNbr);
    save.mutate(
      changed.map((code) => ({
        scope, set_nbr: set, message_nbr: number, language: code, action: 'upsert',
        severity: code === 'en' ? severity : undefined,
        text: drafts[code].text, user_action: drafts[code].user_action || undefined,
        description: drafts[code].description || undefined, reason: reason || undefined,
      })),
    );
  };

  const remove = () => {
    if (!message) return;
    save.mutate([{ scope, set_nbr: message.set_nbr, message_nbr: message.message_nbr, language: lang, action: 'delete', reason: reason || undefined }]);
  };

  const setDraft = (patch: Partial<Draft>) => setDrafts((d) => ({ ...d, [lang]: { ...(d[lang] ?? EMPTY), ...patch } }));
  const langMeta = languages.find((l) => l.code === lang);
  const code = message?.code ?? (setNbr && msgNbr ? `${setNbr}-${msgNbr}` : '—');

  return (
    <Drawer anchor="right" open={open} onClose={onClose} PaperProps={{ sx: { width: { xs: '100%', md: 720 } } }}>
      <Stack direction="row" alignItems="center" spacing={1.5} sx={{ px: 3, py: 2, borderBottom: 1, borderColor: 'divider' }}>
        <Box sx={{ flex: 1, minWidth: 0 }}>
          <Typography variant="overline" color="text.secondary">
            {isNew ? t('messageCatalog.editor.newTitle') : t('messageCatalog.editor.editTitle')}
          </Typography>
          <Stack direction="row" alignItems="center" spacing={1}>
            <Typography variant="h6" fontFamily="monospace">{code}</Typography>
            <SeverityChip severity={severity} />
          </Stack>
        </Box>
        <IconButton onClick={onClose} aria-label={t('messageCatalog.close')}><CloseIcon /></IconButton>
      </Stack>

      <Box sx={{ px: 3, py: 2, overflowY: 'auto', flex: 1 }}>
        <Stack spacing={2.5}>
          {(me.can_edit_core && me.can_edit_tenant) && (
            <Box>
              <Typography variant="subtitle2" gutterBottom>{t('messageCatalog.editor.scope')}</Typography>
              <ToggleButtonGroup exclusive size="small" value={scope} onChange={(_, v) => v && setScope(v)}>
                <ToggleButton value="tenant">{t('messageCatalog.scope.tenant')}</ToggleButton>
                <ToggleButton value="core">{t('messageCatalog.scope.core')}</ToggleButton>
              </ToggleButtonGroup>
              <Typography variant="caption" color="text.secondary" display="block" sx={{ mt: 0.5 }}>
                {scope === 'tenant' ? t('messageCatalog.editor.scopeTenantHelp') : t('messageCatalog.editor.scopeCoreHelp')}
              </Typography>
            </Box>
          )}
          {!editable && <Alert severity="info">{t('messageCatalog.editor.readOnly')}</Alert>}

          {isNew && (
            <Stack direction="row" spacing={2}>
              <FormControl size="small" sx={{ flex: 2 }}>
                <InputLabel>{t('messageCatalog.editor.set')}</InputLabel>
                <Select label={t('messageCatalog.editor.set')} value={setNbr || ''} onChange={(e) => setSetNbr(Number(e.target.value))}>
                  {setChoices.map((s) => (
                    <MenuItem key={s.set_nbr} value={s.set_nbr}>{s.set_nbr} · {s.name}</MenuItem>
                  ))}
                </Select>
              </FormControl>
              <TextField size="small" sx={{ flex: 1 }} label={t('messageCatalog.editor.number')} value={msgNbr}
                onChange={(e) => setMsgNbr(e.target.value.replace(/\D/g, '').slice(0, 5))} inputProps={{ inputMode: 'numeric' }} />
            </Stack>
          )}

          <FormControl size="small" sx={{ maxWidth: 240 }}>
            <InputLabel>{t('messageCatalog.severity')}</InputLabel>
            <Select label={t('messageCatalog.severity')} value={severity} disabled={!editable}
              onChange={(e) => setSeverity(e.target.value as Severity)}>
              {SEVERITIES.map((s) => <MenuItem key={s} value={s}>{t(`messageCatalog.severities.${s}`)}</MenuItem>)}
            </Select>
          </FormControl>

          <Box>
            <Tabs value={lang} onChange={(_, v) => setLang(v)} variant="scrollable" scrollButtons="auto">
              {languages.map((l) => {
                const has = !!drafts[l.code]?.text;
                const dirty = changed.includes(l.code);
                return (
                  <Tab key={l.code} value={l.code} sx={{ minWidth: 0, px: 1.5 }} label={
                    <Stack direction="row" spacing={0.75} alignItems="center">
                      <span>{l.name}</span>
                      <Box component="span" sx={{
                        width: 8, height: 8, borderRadius: '50%',
                        bgcolor: mismatched(l.code) ? 'error.main' : dirty ? 'warning.main' : has ? 'success.main' : 'action.disabled',
                      }} />
                    </Stack>
                  } />
                );
              })}
            </Tabs>
            <Divider />
          </Box>

          {lang !== 'en' && english && (
            <Box sx={{ p: 1.5, borderRadius: 1, bgcolor: 'action.hover' }}>
              <Typography variant="caption" color="text.secondary">{t('messageCatalog.editor.englishReference')}</Typography>
              <Typography variant="body2">{english}</Typography>
            </Box>
          )}

          <TextField label={t('messageCatalog.editor.text')} value={current.text} multiline minRows={2} disabled={!editable}
            onChange={(e) => setDraft({ text: e.target.value })} inputProps={{ dir: langMeta?.rtl ? 'rtl' : 'ltr', maxLength: 1000 }}
            error={!!mismatched(lang)}
            helperText={mismatched(lang)
              ? t('messageCatalog.editor.paramMismatch', { params: params.join(', ') || '—' })
              : t('messageCatalog.editor.textHelp')} />
          <TextField label={t('messageCatalog.editor.userAction')} value={current.user_action} multiline minRows={2} disabled={!editable}
            onChange={(e) => setDraft({ user_action: e.target.value })} inputProps={{ dir: langMeta?.rtl ? 'rtl' : 'ltr', maxLength: 1000 }}
            helperText={t('messageCatalog.editor.userActionHelp')} />
          <TextField label={t('messageCatalog.editor.description')} value={current.description} multiline minRows={2} disabled={!editable}
            onChange={(e) => setDraft({ description: e.target.value })} inputProps={{ maxLength: 4000 }}
            helperText={t('messageCatalog.editor.descriptionHelp')} />

          <Box>
            <Typography variant="subtitle2" gutterBottom>{t('messageCatalog.editor.preview')}</Typography>
            {params.length > 0 && (
              <Stack direction="row" spacing={1} sx={{ mb: 1 }} flexWrap="wrap" useFlexGap>
                {params.map((p) => {
                  const i = Number(p[1]) - 1;
                  return (
                    <TextField key={p} size="small" label={p} value={sample[i] ?? ''} sx={{ width: 140 }}
                      onChange={(e) => setSample((s) => { const n = [...s]; n[i] = e.target.value; return n; })} />
                  );
                })}
              </Stack>
            )}
            <Alert severity={alertSeverity(severity)}
              sx={{ '& .MuiAlert-message': { width: '100%' } }} dir={langMeta?.rtl ? 'rtl' : 'ltr'}>
              <Typography variant="body2">{formatMessage(current.text || english, previewParams) || '—'}</Typography>
              {(current.user_action || drafts.en?.user_action) && (
                <Typography variant="body2" sx={{ mt: 0.5, opacity: 0.85 }}>
                  {formatMessage(current.user_action || drafts.en?.user_action || '', previewParams)}
                </Typography>
              )}
              <Typography variant="caption" sx={{ mt: 0.5, display: 'block', opacity: 0.7 }}>
                {t('messageCatalog.editor.previewFooter', { code })}
              </Typography>
            </Alert>
          </Box>

          {editable && (
            <TextField label={t('messageCatalog.editor.reason')} value={reason} onChange={(e) => setReason(e.target.value)}
              size="small" helperText={requiresApproval ? t('messageCatalog.editor.reasonHelpApproval') : undefined} />
          )}

          {save.error && <CatalogErrorAlert error={save.error} />}
          {result === 'pending' && <Alert severity="info">{t('messageCatalog.editor.submitted')}</Alert>}
          {result === 'applied' && <Alert severity="success">{t('messageCatalog.editor.saved')}</Alert>}

          {history.data?.history && history.data.history.length > 0 && (
            <Box>
              <Typography variant="subtitle2" gutterBottom>{t('messageCatalog.editor.history')}</Typography>
              <Stack spacing={1}>
                {history.data.history.map((c: Change) => (
                  <Stack key={c.id} direction="row" spacing={1} alignItems="center" sx={{ fontSize: 13 }}>
                    <StatusChip status={c.status} />
                    <Chip size="small" variant="outlined" label={c.language} />
                    <Typography variant="body2" sx={{ flex: 1, minWidth: 0 }} noWrap title={c.text}>
                      {c.action === 'delete' ? t('messageCatalog.approvals.deleteAction') : c.text}
                    </Typography>
                    <Typography variant="caption" color="text.secondary" noWrap>
                      {c.requested_by_name || c.requested_by} · {formatWhen(c.requested_at, i18n.language)}
                    </Typography>
                  </Stack>
                ))}
              </Stack>
            </Box>
          )}
        </Stack>
      </Box>

      {editable && (
        <Stack direction="row" spacing={1} alignItems="center" sx={{ px: 3, py: 2, borderTop: 1, borderColor: 'divider' }}>
          {!isNew && scopeRows[lang] && !(scope === 'core' && lang === 'en') && (
            <Button color="error" onClick={remove} disabled={save.isPending}>
              {scope === 'tenant' ? t('messageCatalog.editor.removeOverride') : t('messageCatalog.editor.removeTranslation')}
            </Button>
          )}
          <Box sx={{ flex: 1 }} />
          <Typography variant="caption" color="text.secondary">
            {changed.length > 0 ? t('messageCatalog.editor.changedCount', { count: changed.length }) : t('messageCatalog.editor.noChanges')}
          </Typography>
          <Tooltip title={requiresApproval ? t('messageCatalog.editor.approvalTooltip') : ''}>
            <span>
              <Button variant="contained" onClick={submit} disabled={!changed.length || blocking || save.isPending}>
                {requiresApproval ? t('messageCatalog.editor.submitForApproval') : t('messageCatalog.editor.save')}
              </Button>
            </span>
          </Tooltip>
        </Stack>
      )}
    </Drawer>
  );
}
