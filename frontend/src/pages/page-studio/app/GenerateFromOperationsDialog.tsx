import React, { useMemo, useState } from 'react';
import { Alert, Button, Dialog, DialogActions, DialogContent, DialogTitle, MenuItem, Stack, TextField, Typography } from '@mui/material';
import '../../../studio-core/registerDomains';
import { listOperations, type OperationDef } from '../../../studio-core/operations/registry';
import { GenerateError, generateFromOperations } from './generateFromOperations';
import { pageFromFragment } from './fragment';
import type { CorePageDefinition } from '../../../types/pageStudio';

/**
 * "From operations": pick what lists the rows (and, if wanted, what creates,
 * updates and deletes them) and get a page - list, detail drawer, edit dialog -
 * opened as an unsaved draft, like a blueprint. Nothing is saved until the
 * author saves it, and every widget is open to the usual editors.
 */

const slugOf = (s: string) => s.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '');
const idOf = (s: string) => s.replace(/[^A-Za-z0-9]+(.)?/g, (_, c: string) => (c ? c.toUpperCase() : '')).replace(/^[^A-Za-z]+/, '') || 'page';
// Only operations that declare their rows: `fields` of an envelope result describes the envelope, not a row.
const describes = (o: OperationDef) => !!o.rowFields?.length;

export function GenerateFromOperationsDialog({ open, onClose, onGenerated }: {
  open: boolean;
  onClose: () => void;
  onGenerated: (draft: Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'>) => void;
}) {
  const queries = useMemo(() => listOperations('query').filter(describes), [open]); // eslint-disable-line react-hooks/exhaustive-deps
  const mutations = useMemo(() => listOperations('mutation'), [open]); // eslint-disable-line react-hooks/exhaustive-deps
  const [title, setTitle] = useState('');
  const [list, setList] = useState('');
  const [create, setCreate] = useState('');
  const [update, setUpdate] = useState('');
  const [remove, setRemove] = useState('');
  const [keyField, setKeyField] = useState('id');
  const [error, setError] = useState<string | null>(null);

  const generate = () => {
    setError(null);
    try {
      const fragment = generateFromOperations({
        id: idOf(title), title: title.trim(), list,
        create: create || undefined, update: update || undefined, remove: remove || undefined, keyField: keyField.trim() || 'id',
      });
      onGenerated(pageFromFragment(fragment, { name: title.trim(), slug: slugOf(title) }) as Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'>);
    } catch (err) {
      setError(err instanceof GenerateError ? err.message : 'Could not generate a page from these operations.');
    }
  };

  const picker = (label: string, value: string, set: (v: string) => void, ops: OperationDef[], optional: boolean, help?: string) => (
    <TextField select size="small" label={label} value={value} onChange={(e) => set(e.target.value)} helperText={help}>
      {optional && <MenuItem value=""><em>None</em></MenuItem>}
      {ops.map((o) => <MenuItem key={o.id} value={o.id}>{o.label} ({o.id})</MenuItem>)}
    </TextField>
  );

  return (
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="sm">
      <DialogTitle>Page from operations</DialogTitle>
      <DialogContent>
        <Stack spacing={2} sx={{ pt: 1 }}>
          <Typography variant="body2" color="text.secondary">
            Choose the operation that lists the rows, and any that create, update or delete them. You get a list with a detail drawer and an edit dialog, opened as a draft to adjust before saving.
          </Typography>
          <TextField size="small" label="Page title" value={title} onChange={(e) => setTitle(e.target.value)} />
          {picker('Lists the rows', list, setList, queries, false, queries.length ? 'Only operations that say what their rows carry are offered.' : 'No registered operation describes its rows yet.')}
          {picker('Creates a row', create, setCreate, mutations, true)}
          {picker('Updates a row', update, setUpdate, mutations, true)}
          {picker('Deletes a row', remove, setRemove, mutations, true)}
          <TextField size="small" label="Row key field" value={keyField} onChange={(e) => setKeyField(e.target.value)} helperText="The field that identifies a row." />
          {error && <Alert severity="warning">{error}</Alert>}
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button variant="contained" disabled={!title.trim() || !list} onClick={generate}>Generate</Button>
      </DialogActions>
    </Dialog>
  );
}
