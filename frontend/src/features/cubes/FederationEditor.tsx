import React, { useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Autocomplete,
  Box,
  Button,
  FormControl,
  IconButton,
  InputLabel,
  MenuItem,
  Paper,
  Select,
  Stack,
  TextField,
  Typography,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import type { BusinessObjectOption } from '../../studio-core/binding/businessObjectApi';
import {
  fetchBusinessObjectBindings,
  fetchBOTerms,
} from '../../studio-core/binding/businessObjectApi';
import type {
  CubeFederation,
  CubeFederationJoin,
  CubeFederationSource,
  FederationKeySample,
} from './types';
import { federationIsActive } from './types';

type Props = {
  primaryBoId: string;
  bos: BusinessObjectOption[];
  federation: CubeFederation;
  keySamples: FederationKeySample[];
  onChange: (federation: CubeFederation) => void;
  onKeySamplesChange: (samples: FederationKeySample[]) => void;
};

function aliasFromBo(boId: string): string {
  const raw = (boId || '').trim().toLowerCase();
  if (!raw) return 'src';
  const seg = raw.includes('.') ? raw.slice(raw.lastIndexOf('.') + 1) : raw;
  return seg.replace(/[^a-z0-9_]+/g, '_').replace(/^_+|_+$/g, '') || 'src';
}

function emptyJoin(leftAlias: string, rightAlias: string): CubeFederationJoin {
  return {
    leftAlias,
    rightAlias,
    keyKind: 'common',
    leftTermIds: [],
    rightTermIds: [],
  };
}

const FederationEditor: React.FC<Props> = ({
  primaryBoId,
  bos,
  federation,
  keySamples,
  onChange,
  onKeySamplesChange,
}) => {
  const sources = federation.sources || [];
  const joins = federation.joins || [];
  const active = federationIsActive(federation);

  const [termOptionsByBo, setTermOptionsByBo] = useState<Record<string, string[]>>({});
  const [bindingHintsByBo, setBindingHintsByBo] = useState<Record<string, string[]>>({});

  const aliases = useMemo(() => sources.map((s) => s.alias).filter(Boolean), [sources]);

  useEffect(() => {
    let cancelled = false;
    const load = async () => {
      const nextTerms: Record<string, string[]> = { ...termOptionsByBo };
      const nextHints: Record<string, string[]> = { ...bindingHintsByBo };
      for (const src of sources) {
        const bo = (src.boId || '').trim();
        if (!bo || nextTerms[bo]) continue;
        try {
          const bindings = await fetchBusinessObjectBindings(bo);
          if (cancelled) return;
          nextHints[bo] = bindings
            .map((b) => (b.bindingName || '').trim())
            .filter(Boolean);
          const preferred = bindings.find((b) => b.isDefault) || bindings[0];
          if (!preferred?.bindingId) {
            nextTerms[bo] = [];
            continue;
          }
          const terms = await fetchBOTerms(bo, preferred.bindingId);
          if (cancelled) return;
          nextTerms[bo] = terms
            .map((t) => t.termNodeId || t.termKey || t.termName)
            .filter(Boolean) as string[];
        } catch {
          nextTerms[bo] = [];
          nextHints[bo] = nextHints[bo] || [];
        }
      }
      if (!cancelled) {
        setTermOptionsByBo(nextTerms);
        setBindingHintsByBo(nextHints);
      }
    };
    void load();
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps -- reload when source boIds change
  }, [sources.map((s) => s.boId).join('|')]);

  const setSources = (next: CubeFederationSource[]) => {
    onChange({ ...federation, sources: next });
  };

  const setJoins = (next: CubeFederationJoin[]) => {
    onChange({ ...federation, joins: next });
    // Keep key samples aligned to join edges.
    const edges = new Set(next.map((j) => `${j.leftAlias}→${j.rightAlias}`));
    onKeySamplesChange(
      keySamples.filter((s) => edges.has(`${s.leftAlias}→${s.rightAlias}`)),
    );
  };

  const enableFederation = () => {
    const bo = (primaryBoId || '').trim() || 'account';
    const first: CubeFederationSource = {
      boId: bo,
      alias: aliasFromBo(bo),
      bindingHint: '',
    };
    onChange({
      sources: [first],
      joins: [],
      orphanRateMaxPercent: federation.orphanRateMaxPercent,
    });
  };

  const clearFederation = () => {
    onChange({});
    onKeySamplesChange([]);
  };

  const updateSample = (leftAlias: string, rightAlias: string, patch: Partial<FederationKeySample>) => {
    const idx = keySamples.findIndex(
      (s) => s.leftAlias === leftAlias && s.rightAlias === rightAlias,
    );
    if (idx >= 0) {
      const next = keySamples.slice();
      next[idx] = { ...next[idx], ...patch };
      onKeySamplesChange(next);
      return;
    }
    onKeySamplesChange([
      ...keySamples,
      {
        leftAlias,
        rightAlias,
        leftKeys: 0,
        rightKeys: 0,
        matched: 0,
        ...patch,
      },
    ]);
  };

  if (!active) {
    return (
      <Stack spacing={2}>
        <Alert severity="info">
          Single-BO cube — federation is empty. Add sources to join additional Business Objects on
          shared semantic terms (bindings resolve physical columns at materialize time).
        </Alert>
        <Button variant="outlined" startIcon={<AddIcon />} onClick={enableFederation}>
          Enable federation
        </Button>
      </Stack>
    );
  }

  return (
    <Stack spacing={3}>
      <Alert severity="info">
        Joins use semantic term IDs (`leftTermIds` / `rightTermIds`). Physical columns come from each
        source&apos;s BO binding (`bindingHint`). Transform keys must reference a deterministic
        term/calc — never a metric aggregate.
      </Alert>

      <Box>
        <Stack direction="row" justifyContent="space-between" alignItems="center" mb={1}>
          <Typography variant="subtitle1">Sources</Typography>
          <Button
            size="small"
            startIcon={<AddIcon />}
            onClick={() =>
              setSources([
                ...sources,
                { boId: '', alias: `s${sources.length + 1}`, bindingHint: '' },
              ])
            }
          >
            Add source
          </Button>
        </Stack>
        <Stack spacing={1.5}>
          {sources.map((src, i) => (
            <Paper key={`${src.alias}-${i}`} variant="outlined" sx={{ p: 1.5 }}>
              <Stack direction={{ xs: 'column', md: 'row' }} spacing={1} alignItems="flex-start">
                <Autocomplete
                  sx={{ flex: 2, minWidth: 200 }}
                  options={bos}
                  getOptionLabel={(o) => `${o.displayName || o.name} (${o.key})`}
                  value={bos.find((b) => b.key === src.boId) || null}
                  onChange={(_, v) => {
                    const next = sources.slice();
                    const boId = v?.key || '';
                    next[i] = {
                      ...src,
                      boId,
                      alias: src.alias || aliasFromBo(boId),
                    };
                    setSources(next);
                  }}
                  renderInput={(params) => (
                    <TextField {...params} label="Business Object" size="small" required />
                  )}
                />
                <TextField
                  label="Alias"
                  size="small"
                  required
                  value={src.alias}
                  onChange={(e) => {
                    const next = sources.slice();
                    next[i] = { ...src, alias: e.target.value.trim().toLowerCase() };
                    setSources(next);
                  }}
                  sx={{ width: 120 }}
                />
                <Autocomplete
                  freeSolo
                  sx={{ flex: 1, minWidth: 140 }}
                  options={bindingHintsByBo[src.boId] || []}
                  inputValue={src.bindingHint || ''}
                  onInputChange={(_, v) => {
                    const next = sources.slice();
                    next[i] = { ...src, bindingHint: v };
                    setSources(next);
                  }}
                  renderInput={(params) => (
                    <TextField
                      {...params}
                      label="Binding hint"
                      size="small"
                      placeholder="orm, mdm, …"
                    />
                  )}
                />
                <IconButton
                  aria-label="Remove source"
                  onClick={() => {
                    const removed = src.alias;
                    setSources(sources.filter((_, j) => j !== i));
                    setJoins(
                      joins.filter((j) => j.leftAlias !== removed && j.rightAlias !== removed),
                    );
                  }}
                  disabled={sources.length <= 1}
                >
                  <DeleteOutlineIcon />
                </IconButton>
              </Stack>
            </Paper>
          ))}
        </Stack>
      </Box>

      <Box>
        <Stack direction="row" justifyContent="space-between" alignItems="center" mb={1}>
          <Typography variant="subtitle1">Joins</Typography>
          <Button
            size="small"
            startIcon={<AddIcon />}
            disabled={aliases.length < 2}
            onClick={() =>
              setJoins([
                ...joins,
                emptyJoin(aliases[0], aliases[1] || aliases[0]),
              ])
            }
          >
            Add join
          </Button>
        </Stack>
        {aliases.length < 2 && (
          <Typography variant="body2" color="text.secondary" mb={1}>
            Add at least two sources before declaring joins.
          </Typography>
        )}
        <Stack spacing={1.5}>
          {joins.map((jn, ji) => {
            const leftSrc = sources.find((s) => s.alias === jn.leftAlias);
            const rightSrc = sources.find((s) => s.alias === jn.rightAlias);
            const leftTerms = termOptionsByBo[leftSrc?.boId || ''] || [];
            const rightTerms = termOptionsByBo[rightSrc?.boId || ''] || [];
            const sample =
              keySamples.find(
                (s) => s.leftAlias === jn.leftAlias && s.rightAlias === jn.rightAlias,
              ) || null;
            return (
              <Paper key={ji} variant="outlined" sx={{ p: 1.5 }}>
                <Stack spacing={1.5}>
                  <Stack direction={{ xs: 'column', md: 'row' }} spacing={1}>
                    <FormControl size="small" sx={{ minWidth: 120 }}>
                      <InputLabel id={`left-${ji}`}>Left</InputLabel>
                      <Select
                        labelId={`left-${ji}`}
                        label="Left"
                        value={jn.leftAlias}
                        onChange={(e) => {
                          const next = joins.slice();
                          next[ji] = { ...jn, leftAlias: String(e.target.value) };
                          setJoins(next);
                        }}
                      >
                        {aliases.map((a) => (
                          <MenuItem key={a} value={a}>
                            {a}
                          </MenuItem>
                        ))}
                      </Select>
                    </FormControl>
                    <FormControl size="small" sx={{ minWidth: 120 }}>
                      <InputLabel id={`right-${ji}`}>Right</InputLabel>
                      <Select
                        labelId={`right-${ji}`}
                        label="Right"
                        value={jn.rightAlias}
                        onChange={(e) => {
                          const next = joins.slice();
                          next[ji] = { ...jn, rightAlias: String(e.target.value) };
                          setJoins(next);
                        }}
                      >
                        {aliases.map((a) => (
                          <MenuItem key={a} value={a} disabled={a === jn.leftAlias}>
                            {a}
                          </MenuItem>
                        ))}
                      </Select>
                    </FormControl>
                    <FormControl size="small" sx={{ minWidth: 140 }}>
                      <InputLabel id={`kind-${ji}`}>Key kind</InputLabel>
                      <Select
                        labelId={`kind-${ji}`}
                        label="Key kind"
                        value={jn.keyKind || 'common'}
                        onChange={(e) => {
                          const kind = String(e.target.value);
                          const next = joins.slice();
                          next[ji] = {
                            ...jn,
                            keyKind: kind,
                            transformTermId: kind === 'transform' ? jn.transformTermId : undefined,
                          };
                          setJoins(next);
                        }}
                      >
                        <MenuItem value="common">common</MenuItem>
                        <MenuItem value="transform">transform</MenuItem>
                      </Select>
                    </FormControl>
                    <IconButton
                      aria-label="Remove join"
                      onClick={() => setJoins(joins.filter((_, j) => j !== ji))}
                    >
                      <DeleteOutlineIcon />
                    </IconButton>
                  </Stack>

                  <Autocomplete
                    multiple
                    freeSolo
                    options={leftTerms}
                    value={jn.leftTermIds || []}
                    onChange={(_, v) => {
                      const next = joins.slice();
                      next[ji] = { ...jn, leftTermIds: v.map(String) };
                      setJoins(next);
                    }}
                    renderInput={(params) => (
                      <TextField
                        {...params}
                        label="Left term IDs"
                        size="small"
                        helperText="Semantic terms on the left source (parallel to right)"
                      />
                    )}
                  />
                  <Autocomplete
                    multiple
                    freeSolo
                    options={rightTerms}
                    value={jn.rightTermIds || []}
                    onChange={(_, v) => {
                      const next = joins.slice();
                      next[ji] = { ...jn, rightTermIds: v.map(String) };
                      setJoins(next);
                    }}
                    renderInput={(params) => (
                      <TextField
                        {...params}
                        label="Right term IDs"
                        size="small"
                        helperText="Semantic terms on the right source (same length/order as left)"
                      />
                    )}
                  />
                  {jn.keyKind === 'transform' && (
                    <Autocomplete
                      freeSolo
                      options={[...new Set([...leftTerms, ...rightTerms])]}
                      inputValue={jn.transformTermId || ''}
                      onInputChange={(_, v) => {
                        const next = joins.slice();
                        next[ji] = { ...jn, transformTermId: v };
                        setJoins(next);
                      }}
                      renderInput={(params) => (
                        <TextField
                          {...params}
                          label="Transform term ID"
                          size="small"
                          required
                          helperText="Deterministic term/calc — never a metric aggregate"
                        />
                      )}
                    />
                  )}

                  <Typography variant="caption" color="text.secondary">
                    Orphan-gate sample for validate (optional for shape-only; required to pass deploy gate)
                  </Typography>
                  <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1}>
                    <TextField
                      label="Left keys"
                      type="number"
                      size="small"
                      value={sample?.leftKeys ?? ''}
                      onChange={(e) =>
                        updateSample(jn.leftAlias, jn.rightAlias, {
                          leftKeys: Number(e.target.value) || 0,
                        })
                      }
                      sx={{ width: 120 }}
                    />
                    <TextField
                      label="Right keys"
                      type="number"
                      size="small"
                      value={sample?.rightKeys ?? ''}
                      onChange={(e) =>
                        updateSample(jn.leftAlias, jn.rightAlias, {
                          rightKeys: Number(e.target.value) || 0,
                        })
                      }
                      sx={{ width: 120 }}
                    />
                    <TextField
                      label="Matched"
                      type="number"
                      size="small"
                      value={sample?.matched ?? ''}
                      onChange={(e) =>
                        updateSample(jn.leftAlias, jn.rightAlias, {
                          matched: Number(e.target.value) || 0,
                        })
                      }
                      sx={{ width: 120 }}
                    />
                  </Stack>
                </Stack>
              </Paper>
            );
          })}
        </Stack>
      </Box>

      <TextField
        label="Orphan rate max %"
        type="number"
        size="small"
        value={federation.orphanRateMaxPercent ?? ''}
        onChange={(e) => {
          const raw = e.target.value;
          if (raw === '') {
            onChange({ ...federation, orphanRateMaxPercent: undefined });
            return;
          }
          const n = Number(raw);
          onChange({
            ...federation,
            orphanRateMaxPercent: Number.isFinite(n) ? n : undefined,
          });
        }}
        helperText="Blank uses platform default 1.0%. Deploy fails when unmatched keys exceed this."
        sx={{ maxWidth: 240 }}
        inputProps={{ min: 0, step: 0.1 }}
      />

      <Button color="inherit" onClick={clearFederation}>
        Clear federation (back to single-BO)
      </Button>
    </Stack>
  );
};

export default FederationEditor;
