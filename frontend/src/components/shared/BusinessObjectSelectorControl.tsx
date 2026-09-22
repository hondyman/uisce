/**
 * BusinessObjectSelectorControl - the centralized, reusable UI for "pick a
 * primary Business Object + binding, then add/remove relevant related
 * Business Objects and their fields", built on
 * studio-core/binding/useBusinessObjectSelector.
 *
 * Query Builder's SavedQueryEditor, Page Studio's Data Binding panel, and
 * the BO Binding wizard all grew their own version of this picker
 * independently; this is the one to import going forward instead of
 * writing a fourth. Field selection is opt-in via `onToggleField` (a
 * report/page consumer that only needs the object+binding pair, not field
 * checkboxes, can render only `renderFields={false}`).
 */
import React from 'react';
import {
  Box, Paper, Typography, TextField, Autocomplete, Chip, Stack, Button,
  Table, TableBody, TableCell, TableContainer, TableHead, TableRow,
  Checkbox, FormControl, InputLabel, Select, MenuItem, Tooltip, CircularProgress, Alert,
} from '@mui/material';
import LockIcon from '@mui/icons-material/Lock';
import { useBusinessObjectSelector } from '../../studio-core/binding/useBusinessObjectSelector';
import { fetchBusinessObjectBindings, type BindingView, type BusinessObjectOption } from '../../studio-core/binding/businessObjectApi';

export type BusinessObjectSelector = ReturnType<typeof useBusinessObjectSelector>;

export interface BusinessObjectSelectorControlProps {
  /** The hook instance to render - call useBusinessObjectSelector in the
   * parent so it can also read primary/related/objects for its own save
   * logic, and pass the same instance here. */
  selector: BusinessObjectSelector;
  /** Show the per-object field table with checkboxes (default true). Set
   * false for a consumer that only needs the object/binding pair. */
  renderFields?: boolean;
  /** Called whenever a field checkbox is toggled, after the hook's own
   * state update - most consumers don't need this, since `selector.objects`
   * already reflects the new selection reactively. */
  onToggleField?: (boId: string, termNodeId: string) => void;
  /** Override what happens when a primary object + binding is picked for
   * the first time. Default: the hook adopts it into local state
   * immediately (fine for a picker that edits an already-persisted
   * selection). A creation flow that needs to call a "create" API first
   * (e.g. Query Builder minting the saved-query row before anything else
   * can be edited) passes this instead of letting the hook self-select. */
  onSelectPrimary?: (bo: BusinessObjectOption, bindingId: string) => void | Promise<void>;
}

export default function BusinessObjectSelectorControl({ selector, renderFields = true, onToggleField, onSelectPrimary }: BusinessObjectSelectorControlProps) {
  const {
    boOptions, boOptionsLoading, primary, related, objects, locked,
    availableRelationships, relationshipsLoading,
    selectPrimary, addRelated, removeRelated, toggleField, error,
  } = selector;

  const handleToggle = (boId: string, termNodeId: string) => {
    toggleField(boId, termNodeId);
    onToggleField?.(boId, termNodeId);
  };

  if (!primary) {
    return <PrimaryObjectPicker boOptions={boOptions} boOptionsLoading={boOptionsLoading} onSelect={onSelectPrimary || selectPrimary} error={error} />;
  }

  return (
    <Box>
      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

      <Paper variant="outlined" sx={{ p: 2, mb: 2 }}>
        <Stack direction="row" alignItems="center" spacing={1} mb={1}>
          {locked && <LockIcon fontSize="small" color="disabled" />}
          <Typography variant="subtitle2">Primary object: {primary.boName}</Typography>
          <Tooltip title={locked ? 'The primary object and binding are locked after creation' : 'Binding used for this object'}>
            <Chip size="small" label={`Binding: ${primary.bindingId}`} />
          </Tooltip>
        </Stack>

        <Typography variant="subtitle2" gutterBottom sx={{ mt: 1 }}>Related objects</Typography>
        <Stack direction="row" spacing={1} flexWrap="wrap" mb={1} useFlexGap>
          {related.map((r) => (
            <Chip key={r.boId} label={r.boName} onDelete={() => removeRelated(r.boId)} />
          ))}
        </Stack>
        <Autocomplete
          options={availableRelationships}
          loading={relationshipsLoading}
          getOptionLabel={(r) => r.relatedObjectName}
          value={null}
          onChange={(_, v) => v && addRelated(v.targetObjectId, v.relatedObjectName)}
          renderOption={(props, r) => (
            <li {...props} key={r.targetObjectId}>
              <Box>
                <Typography variant="body2">{r.relatedObjectName}</Typography>
                <Typography variant="caption" color="text.secondary">{r.cardinality}</Typography>
              </Box>
            </li>
          )}
          renderInput={(params) => <TextField {...params} size="small" label="Add related object" placeholder="Search relevant objects…" />}
          sx={{ maxWidth: 360 }}
        />
      </Paper>

      {renderFields && objects.map((obj) => (
        <Paper key={obj.boId} variant="outlined" sx={{ p: 2, mb: 2 }}>
          <Typography variant="subtitle2" gutterBottom>{obj.boName} — fields</Typography>
          {obj.termsLoading ? (
            <Box display="flex" justifyContent="center" p={2}><CircularProgress size={20} /></Box>
          ) : obj.termsError ? (
            <Alert severity="warning">{obj.termsError}</Alert>
          ) : (
            <TableContainer sx={{ overflowX: 'auto' }}>
              <Table size="small" sx={{ minWidth: 500 }}>
                <TableHead>
                  <TableRow>
                    <TableCell padding="checkbox" sx={{ width: 48 }} />
                    <TableCell>Field</TableCell>
                    <TableCell sx={{ width: 120 }}>Role</TableCell>
                    <TableCell sx={{ width: 100 }}>Type</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {obj.terms.map((t) => (
                    <TableRow key={t.termNodeId} hover onClick={() => handleToggle(obj.boId, t.termNodeId)} sx={{ cursor: 'pointer' }}>
                      <TableCell padding="checkbox">
                        <Checkbox checked={!!t.selected} size="small" />
                      </TableCell>
                      <TableCell>{t.displayName}</TableCell>
                      <TableCell>{t.role}</TableCell>
                      <TableCell>{t.dataType}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          )}
        </Paper>
      ))}
    </Box>
  );
}

/** Shown before a primary object has been chosen - BO + binding pickers,
 * locked in on selection. */
function PrimaryObjectPicker({ boOptions, boOptionsLoading, onSelect, error }: {
  boOptions: BusinessObjectOption[];
  boOptionsLoading: boolean;
  onSelect: (bo: BusinessObjectOption, bindingId: string) => void;
  error: string | null;
}) {
  const [bo, setBo] = React.useState<BusinessObjectOption | null>(null);
  const [bindings, setBindings] = React.useState<BindingView[]>([]);
  const [bindingId, setBindingId] = React.useState('');
  const [loadingBindings, setLoadingBindings] = React.useState(false);

  React.useEffect(() => {
    if (!bo) { setBindings([]); setBindingId(''); return; }
    setLoadingBindings(true);
    fetchBusinessObjectBindings(bo.id)
      .then((b) => {
        setBindings(b);
        const def = b.find((x) => x.isDefault) || b[0];
        setBindingId(def?.bindingId || '');
      })
      .finally(() => setLoadingBindings(false));
  }, [bo]);

  return (
    <Box maxWidth={480}>
      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
      <Stack spacing={2}>
        <Autocomplete
          options={boOptions}
          loading={boOptionsLoading}
          getOptionLabel={(o) => o.displayName || o.name}
          value={bo}
          onChange={(_, v) => setBo(v)}
          renderInput={(params) => <TextField {...params} label="Business Object" />}
        />
        <FormControl fullWidth disabled={!bo || bindings.length === 0}>
          <InputLabel>Binding</InputLabel>
          <Select value={bindingId} label="Binding" onChange={(e) => setBindingId(e.target.value)}>
            {bindings.map((b) => (
              <MenuItem key={b.bindingId} value={b.bindingId}>
                {b.bindingName} {b.isDefault ? '(default)' : ''}
              </MenuItem>
            ))}
          </Select>
        </FormControl>
        <Box>
          <Button
            variant="contained"
            disabled={!bo || !bindingId || loadingBindings}
            onClick={() => bo && bindingId && onSelect(bo, bindingId)}
          >
            {loadingBindings ? <CircularProgress size={20} /> : 'Select'}
          </Button>
        </Box>
      </Stack>
    </Box>
  );
}
