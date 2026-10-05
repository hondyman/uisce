import React, { useEffect, useState } from 'react';
import {
  Box,
  Stack,
  TextField,
  Typography,
} from '@mui/material';
import { listCubes } from '../cubes/cubeDefinitionApi';
import type { CubeDefinition } from '../cubes/types';
import type { QuerySubject, QuerySubjectKind } from './types';
import { businessObjectSubject, cubeSubject } from './types';

export type SubjectPickerProps = {
  value: QuerySubject | null;
  /** BO options already loaded by the host shell. */
  businessObjects: Array<{ id: string; display_name: string; name?: string }>;
  /** Called when the user changes kind or selection. */
  onChange: (next: QuerySubject | null, meta?: { cube?: CubeDefinition }) => void;
  disabled?: boolean;
};

/**
 * Single subject picker for Query Builder / Report Builder: Business Object | Cube.
 */
export const SubjectPicker: React.FC<SubjectPickerProps> = ({
  value,
  businessObjects,
  onChange,
  disabled,
}) => {
  // Local override so switching to Cube is sticky while cubeId is still empty
  // (parent may synthesize a BO subject from leftover selectedBO).
  const [kindOverride, setKindOverride] = useState<QuerySubjectKind | null>(null);
  const kind: QuerySubjectKind =
    kindOverride
    ?? (value?.kind === 'cube' ? 'cube' : 'business_object');
  const [cubes, setCubes] = useState<CubeDefinition[]>([]);
  const [cubesError, setCubesError] = useState<string | null>(null);
  const [loadingCubes, setLoadingCubes] = useState(false);

  useEffect(() => {
    if (value?.kind === 'cube' && value.cubeId) {
      setKindOverride(null);
    }
  }, [value]);

  useEffect(() => {
    if (kind !== 'cube') return;
    let cancelled = false;
    setLoadingCubes(true);
    setCubesError(null);
    listCubes({ scope: 'all', limit: 200 })
      .then((res) => {
        if (!cancelled) setCubes(res.cubes || []);
      })
      .catch((err) => {
        if (!cancelled) {
          setCubes([]);
          setCubesError(err instanceof Error ? err.message : String(err));
        }
      })
      .finally(() => {
        if (!cancelled) setLoadingCubes(false);
      });
    return () => {
      cancelled = true;
    };
  }, [kind]);

  const handleKindChange = (next: QuerySubjectKind) => {
    if (next === 'business_object') {
      setKindOverride(null);
      const first = businessObjects[0];
      onChange(
        first
          ? businessObjectSubject(first.id, '')
          : null,
      );
      return;
    }
    setKindOverride('cube');
    onChange(null);
  };

  return (
    <Box sx={{ p: 2, borderBottom: '1px solid #eee' }} data-testid="subject-picker" data-kind={kind}>
      <Typography variant="overline" color="text.secondary">
        Subject
      </Typography>
      <Stack spacing={1.5} sx={{ mt: 1 }}>
        <TextField
          select
          fullWidth
          size="small"
          label="Kind"
          value={kind}
          onChange={(e) => handleKindChange(e.target.value as QuerySubjectKind)}
          SelectProps={{ native: true }}
          inputProps={{ 'aria-label': 'Subject kind' }}
          disabled={disabled}
        >
          <option value="business_object">Business Object</option>
          <option value="cube">Cube</option>
        </TextField>

        {kind === 'business_object' && (
          <TextField
            select
            fullWidth
            size="small"
            label="Business Object"
            value={value?.kind === 'business_object' ? value.boId : ''}
            onChange={(e) => {
              const boId = e.target.value;
              setKindOverride(null);
              onChange(businessObjectSubject(boId, value?.kind === 'business_object' ? value.bindingId : ''));
            }}
            SelectProps={{ native: true }}
            inputProps={{ 'aria-label': 'Business Object' }}
            disabled={disabled}
          >
            <option value="" disabled>
              Select Business Object...
            </option>
            {businessObjects.map((bo) => (
              <option key={bo.id} value={bo.id}>
                {bo.display_name || bo.name || bo.id}
              </option>
            ))}
          </TextField>
        )}

        {kind === 'cube' && (
          <TextField
            select
            fullWidth
            size="small"
            label={loadingCubes ? 'Loading cubes…' : 'Cube'}
            value={value?.kind === 'cube' ? value.cubeId : ''}
            onChange={(e) => {
              const cubeId = e.target.value;
              const cube = cubes.find((c) => c.id === cubeId);
              setKindOverride(null);
              onChange(cubeSubject(cubeId, 'latest'), { cube });
            }}
            SelectProps={{ native: true }}
            inputProps={{ 'aria-label': 'Cube' }}
            disabled={disabled || loadingCubes}
            helperText={cubesError || undefined}
            error={!!cubesError}
          >
            <option value="" disabled>
              {loadingCubes ? 'Loading…' : 'Select Cube...'}
            </option>
            {cubes.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
                {c.isCore ? ' (core)' : ''} · v{c.contractVersion}
              </option>
            ))}
          </TextField>
        )}
      </Stack>
    </Box>
  );
};

export default SubjectPicker;
