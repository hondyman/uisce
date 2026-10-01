import React, { useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import {
  Badge, Box, Button, Chip, FormControl, InputAdornment, InputLabel, LinearProgress, List, ListItemButton,
  ListItemText, MenuItem, Paper, Select, Stack, Tab, Table, TableBody, TableCell, TableContainer, TableHead,
  TableRow, Tabs, TextField, Tooltip, Typography,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import SearchIcon from '@mui/icons-material/Search';
import TranslateIcon from '@mui/icons-material/Translate';
import { useLocale } from '../../i18n/useLocale';
import { effective, MessageView, msgcatApi } from './api';
import MessageEditor from './MessageEditor';
import ApprovalsPanel from './ApprovalsPanel';
import { CatalogErrorAlert, SeverityChip } from './parts';

export default function MessageCatalogPage() {
  const { t } = useTranslation();
  const uiLocale = useLocale();
  const [tab, setTab] = useState<'messages' | 'approvals'>('messages');
  const [setNbr, setSetNbr] = useState<number>(0);
  const [q, setQ] = useState('');
  const [lang, setLang] = useState<string>(uiLocale);
  const [editing, setEditing] = useState<MessageView | 'new' | null>(null);

  const me = useQuery({ queryKey: ['msgcat-me'], queryFn: msgcatApi.me });
  const languages = useQuery({ queryKey: ['msgcat-languages'], queryFn: msgcatApi.languages, staleTime: Infinity });
  const sets = useQuery({ queryKey: ['msgcat-sets'], queryFn: msgcatApi.sets });
  const messages = useQuery({ queryKey: ['msgcat-messages'], queryFn: () => msgcatApi.messages({}) });
  const canReview = !!me.data && (me.data.can_edit_core || me.data.can_edit_tenant);
  const pending = useQuery({
    queryKey: ['msgcat-changes', false],
    queryFn: () => msgcatApi.changes('pending'),
    enabled: canReview,
  });

  const all = messages.data?.messages ?? [];
  const counts = useMemo(() => {
    const c: Record<number, number> = {};
    for (const m of all) c[m.set_nbr] = (c[m.set_nbr] ?? 0) + 1;
    return c;
  }, [all]);
  const needle = q.trim().toLowerCase();
  const shown = all.filter((m) => {
    if (setNbr && m.set_nbr !== setNbr) return false;
    if (!needle) return true;
    if (m.code.includes(needle)) return true;
    return [...Object.values(m.core), ...Object.values(m.tenant)].some(
      (e) => e.text.toLowerCase().includes(needle) || (e.description ?? '').toLowerCase().includes(needle),
    );
  });
  const langs = languages.data?.languages ?? [];
  const langName = (code: string) => langs.find((l) => l.code === code)?.name ?? code;
  const pendingCount = pending.data?.changes.length ?? 0;
  const selectedSet = sets.data?.sets.find((s) => s.set_nbr === setNbr);

  return (
    // The shell's canvas is dark whatever the MUI mode; paint the page's own
    // themed surface so text contrast follows the theme.
    <Box sx={{ bgcolor: 'background.default', color: 'text.primary', minHeight: '100%' }}>
    <Box sx={{ p: { xs: 2, md: 3 }, maxWidth: 1400, mx: 'auto' }}>
      <Stack direction={{ xs: 'column', md: 'row' }} alignItems={{ md: 'center' }} spacing={2} sx={{ mb: 2 }}>
        <Stack direction="row" spacing={2} alignItems="center" sx={{ flex: 1 }}>
          <TranslateIcon color="primary" fontSize="large" />
          <Box>
            <Typography variant="h5" fontWeight={700}>{t('messageCatalog.title')}</Typography>
            <Typography color="text.secondary">{t('messageCatalog.subtitle')}</Typography>
          </Box>
        </Stack>
        {canReview && (
          <Button variant="contained" startIcon={<AddIcon />} onClick={() => setEditing('new')}>
            {t('messageCatalog.newMessage')}
          </Button>
        )}
      </Stack>

      <Tabs value={tab} onChange={(_, v) => setTab(v)} sx={{ mb: 2, borderBottom: 1, borderColor: 'divider' }}>
        <Tab value="messages" label={t('messageCatalog.tabs.messages')} />
        {canReview && (
          <Tab value="approvals" label={
            <Badge color="warning" badgeContent={pendingCount} sx={{ '& .MuiBadge-badge': { right: -14 } }}>
              {t('messageCatalog.tabs.approvals')}
            </Badge>
          } />
        )}
      </Tabs>

      {(me.error || messages.error) && <CatalogErrorAlert error={me.error ?? messages.error} />}

      {tab === 'approvals' && me.data && <ApprovalsPanel me={me.data} />}

      {tab === 'messages' && (
        <Stack direction={{ xs: 'column', md: 'row' }} spacing={2} alignItems="flex-start">
          <Paper variant="outlined" sx={{ width: { xs: '100%', md: 260 }, flexShrink: 0, maxHeight: { md: 'calc(100vh - 260px)' }, overflowY: 'auto' }}>
            <List dense disablePadding>
              <ListItemButton selected={setNbr === 0} onClick={() => setSetNbr(0)}>
                <ListItemText primary={t('messageCatalog.allSets')} secondary={t('messageCatalog.messageCount', { count: all.length })} />
              </ListItemButton>
              {(sets.data?.sets ?? []).map((s) => (
                <ListItemButton key={s.set_nbr} selected={setNbr === s.set_nbr} onClick={() => setSetNbr(s.set_nbr)}>
                  <ListItemText
                    primary={<Stack direction="row" spacing={1} alignItems="center">
                      <Typography variant="body2" fontFamily="monospace" color="text.secondary">{s.set_nbr}</Typography>
                      <Typography variant="body2" fontWeight={500} noWrap>{s.name}</Typography>
                    </Stack>}
                    secondary={t('messageCatalog.setSecondary', { type: t(`messageCatalog.setTypes.${s.type}`), count: counts[s.set_nbr] ?? 0 })}
                  />
                </ListItemButton>
              ))}
            </List>
          </Paper>

          <Box sx={{ flex: 1, minWidth: 0, width: '100%' }}>
            <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2} sx={{ mb: 2 }}>
              <TextField size="small" sx={{ flex: 1 }} placeholder={t('messageCatalog.searchPlaceholder')} value={q}
                onChange={(e) => setQ(e.target.value)}
                InputProps={{ startAdornment: <InputAdornment position="start"><SearchIcon fontSize="small" /></InputAdornment> }} />
              <FormControl size="small" sx={{ minWidth: 200 }}>
                <InputLabel>{t('messageCatalog.showIn')}</InputLabel>
                <Select label={t('messageCatalog.showIn')} value={langs.some((l) => l.code === lang) ? lang : ''} onChange={(e) => setLang(e.target.value)}>
                  {langs.map((l) => <MenuItem key={l.code} value={l.code}>{l.name}</MenuItem>)}
                </Select>
              </FormControl>
            </Stack>
            {selectedSet?.description && (
              <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>{selectedSet.description}</Typography>
            )}
            {messages.isLoading && <LinearProgress />}
            <TableContainer component={Paper} variant="outlined">
              <Table size="small" stickyHeader>
                <TableHead>
                  <TableRow>
                    <TableCell sx={{ width: 110 }}>{t('messageCatalog.columns.code')}</TableCell>
                    <TableCell sx={{ width: 110 }}>{t('messageCatalog.severity')}</TableCell>
                    <TableCell>{t('messageCatalog.columns.message')}</TableCell>
                    <TableCell sx={{ width: 220, display: { xs: 'none', sm: 'table-cell' } }}>{t('messageCatalog.columns.languages')}</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {shown.map((m) => {
                    const { entry, source } = effective(m, lang);
                    const override = Object.keys(m.tenant).length > 0;
                    return (
                      <TableRow key={m.code} hover sx={{ cursor: 'pointer' }} onClick={() => setEditing(m)}>
                        <TableCell sx={{ fontFamily: 'monospace', whiteSpace: 'nowrap' }}>{m.code}</TableCell>
                        <TableCell><SeverityChip severity={m.severity} /></TableCell>
                        <TableCell>
                          <Typography variant="body2" sx={{ fontStyle: source === 'fallback' ? 'italic' : 'normal' }}
                            dir={langs.find((l) => l.code === (entry?.language ?? lang))?.rtl ? 'rtl' : 'ltr'}>
                            {entry?.text ?? '—'}
                          </Typography>
                          {source === 'fallback' && (
                            <Typography variant="caption" color="warning.main">
                              {t('messageCatalog.untranslated', { language: langName(lang) })}
                            </Typography>
                          )}
                          <Stack direction="row" spacing={0.5} sx={{ mt: 0.5 }} flexWrap="wrap" useFlexGap>
                            <Chip size="small" variant="outlined" sx={{ display: { xs: 'inline-flex', sm: 'none' } }}
                              label={t('messageCatalog.languagesCount', { count: langs.filter((l) => m.tenant[l.code] || m.core[l.code]).length, total: langs.length })} />
                            {override && <Chip size="small" color="secondary" variant="outlined" label={t('messageCatalog.overridden')} />}
                            {m.pending > 0 && <Chip size="small" color="warning" label={t('messageCatalog.pendingCount', { count: m.pending })} />}
                          </Stack>
                        </TableCell>
                        <TableCell sx={{ display: { xs: 'none', sm: 'table-cell' } }}>
                          <Stack direction="row" spacing={0.5} flexWrap="wrap" useFlexGap>
                            {langs.map((l) => {
                              const has = m.tenant[l.code] || m.core[l.code];
                              return (
                                <Tooltip key={l.code} title={`${l.name}: ${m.tenant[l.code] ? t('messageCatalog.scope.tenant') : m.core[l.code] ? t('messageCatalog.scope.core') : t('messageCatalog.missing')}`}>
                                  <Chip size="small" label={l.code} variant={has ? 'filled' : 'outlined'}
                                    color={m.tenant[l.code] ? 'secondary' : has ? 'primary' : 'default'}
                                    sx={{ height: 20, fontSize: 11, opacity: has ? 1 : 0.5 }} />
                                </Tooltip>
                              );
                            })}
                          </Stack>
                        </TableCell>
                      </TableRow>
                    );
                  })}
                  {!messages.isLoading && shown.length === 0 && (
                    <TableRow><TableCell colSpan={4}><Typography color="text.secondary" sx={{ py: 2, textAlign: 'center' }}>{t('messageCatalog.noMatches')}</Typography></TableCell></TableRow>
                  )}
                </TableBody>
              </Table>
            </TableContainer>
          </Box>
        </Stack>
      )}

      {me.data && editing && (
        <MessageEditor
          open
          onClose={() => setEditing(null)}
          message={editing === 'new' ? undefined : all.find((m) => m.code === editing.code) ?? editing}
          sets={sets.data?.sets ?? []}
          languages={langs}
          me={me.data}
          initialSet={setNbr || undefined}
          initialLanguage={editing === 'new' ? 'en' : lang}
        />
      )}
    </Box>
    </Box>
  );
}
