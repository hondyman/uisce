import React, { useEffect, useState, useCallback } from 'react';
import {
  Box, Paper, Table, TableHead, TableBody, TableRow, TableCell, IconButton, Typography, Tooltip, CircularProgress, Alert,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import OpenInNewIcon from '@mui/icons-material/OpenInNew';
import { GenericBOFormModal } from '../pagedesigner/widgets/GenericBOFormModal';
import type { RelatedObjectWidget } from './pageStudioTypes';

export interface RelatedObjectGridProps {
  rootBoKey: string;
  rootRecordId: string;
  widget: RelatedObjectWidget;
}

export const RelatedObjectGrid: React.FC<RelatedObjectGridProps> = ({ rootBoKey, rootRecordId, widget }) => {
  const [records, setRecords] = useState<Record<string, any>[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [modalRecordId, setModalRecordId] = useState<string | null | undefined>(undefined); // undefined = closed

  const listUrl = `/api/v1/bo/${encodeURIComponent(rootBoKey)}/records/${encodeURIComponent(rootRecordId)}/relationships/${encodeURIComponent(widget.relKey)}`;

  const load = useCallback(() => {
    setError(null);
    fetch(listUrl)
      .then(async (res) => {
        if (!res.ok) throw new Error(await res.text());
        return res.json();
      })
      .then((data) => setRecords(data?.records || []))
      .catch((err) => setError(err.message || 'Failed to load related records'));
  }, [listUrl]);

  useEffect(() => {
    load();
  }, [load]);

  const columns = widget.displayColumns.length > 0 ? widget.displayColumns : ['id'];

  return (
    <Paper variant="outlined" sx={{ p: 1.5 }}>
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1 }}>
        <Typography variant="subtitle2" fontWeight={700}>{widget.title}</Typography>
        <Tooltip title={`Add ${widget.targetBoKey}`}>
          <IconButton size="small" onClick={() => setModalRecordId(null)}>
            <AddIcon fontSize="small" />
          </IconButton>
        </Tooltip>
      </Box>

      {error && <Alert severity="error" sx={{ mb: 1 }}>{error}</Alert>}

      {records === null ? (
        <Box sx={{ display: 'flex', justifyContent: 'center', p: 2 }}>
          <CircularProgress size={20} />
        </Box>
      ) : records.length === 0 ? (
        <Typography variant="caption" color="text.secondary">No related records yet.</Typography>
      ) : (
        <Table size="small">
          <TableHead>
            <TableRow>
              {columns.map((c) => <TableCell key={c}>{c}</TableCell>)}
              <TableCell align="right" />
            </TableRow>
          </TableHead>
          <TableBody>
            {records.map((rec) => (
              <TableRow key={rec.id}>
                {columns.map((c) => <TableCell key={c}>{String(rec[c] ?? '')}</TableCell>)}
                <TableCell align="right">
                  <Tooltip title="Open">
                    <IconButton size="small" onClick={() => setModalRecordId(rec.id)}>
                      <OpenInNewIcon fontSize="inherit" />
                    </IconButton>
                  </Tooltip>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      {modalRecordId !== undefined && (
        <GenericBOFormModal
          boKey={widget.targetBoKey}
          recordId={modalRecordId}
          open
          onClose={() => setModalRecordId(undefined)}
          onSaved={load}
          createUrl={listUrl}
          fieldKeys={columns}
        />
      )}
    </Paper>
  );
};

export default RelatedObjectGrid;
