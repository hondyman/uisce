import React from 'react';
import { Box, Typography } from '@mui/material';
import type { BOSchema } from '../types/queryDef';

/** Minimal stub — schema-driven form entry is out of scope for CUBE-1.6. */
const AutoFormRenderer: React.FC<{
  schema?: BOSchema | null;
  onAddField?: (fieldId: string) => void;
  onAddFilter?: (fieldId: string) => void;
  isInQuery?: (fieldId: string) => boolean;
}> = ({ schema }) => (
  <Box sx={{ p: 1 }}>
    <Typography variant="body2" color="text.secondary">
      {schema?.fields?.length
        ? `${schema.fields.length} schema fields (form renderer stub)`
        : 'Form renderer unavailable in this build.'}
    </Typography>
  </Box>
);

export default AutoFormRenderer;
