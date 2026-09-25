import React from 'react';
import { useNavigate } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Alert, Box, Button, Card, CardActionArea, CardContent, Chip, IconButton, LinearProgress, Stack, Tooltip, Typography,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import DeleteIcon from '@mui/icons-material/Delete';
import AccountTreeIcon from '@mui/icons-material/AccountTree';
import { Definition, pipelinesApi } from './api';
import { NODE_META } from './PipelineNode';

function describe(d: Definition): string {
  const src = d.spec.nodes.filter(n => NODE_META[n.type]?.category === 'source').map(n => n.label || n.type);
  const dst = d.spec.nodes.filter(n => NODE_META[n.type]?.category === 'destination').map(n => n.label || n.type);
  if (!src.length && !dst.length) return 'Empty pipeline';
  return `${src.join(', ') || '?'} → ${dst.join(', ') || '?'}`;
}

export default function PipelinesListPage() {
  const navigate = useNavigate();
  const qc = useQueryClient();
  const list = useQuery({ queryKey: ['dp-list'], queryFn: pipelinesApi.list });
  const remove = useMutation({
    mutationFn: (id: string) => pipelinesApi.remove(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['dp-list'] }),
  });

  return (
    <Box sx={{ p: 3, maxWidth: 1200, mx: 'auto' }}>
      <Stack direction="row" alignItems="center" spacing={2} sx={{ mb: 3 }}>
        <AccountTreeIcon color="primary" fontSize="large" />
        <Box sx={{ flex: 1 }}>
          <Typography variant="h5" fontWeight={700}>Data pipelines</Typography>
          <Typography color="text.secondary">
            Load files and business objects, check them against your rules, and write them to business objects, staging tables or files - no code.
          </Typography>
        </Box>
        <Button variant="contained" startIcon={<AddIcon />} onClick={() => navigate('/data/pipelines/new')}>New pipeline</Button>
      </Stack>
      {list.isLoading && <LinearProgress />}
      {list.isError && <Alert severity="error">Could not load pipelines: {String((list.error as Error).message)}</Alert>}
      {list.data?.length === 0 && (
        <Alert severity="info" action={<Button onClick={() => navigate('/data/pipelines/new')}>Create one</Button>}>
          No pipelines yet. Start from a blank canvas, or describe what you want to the assistant.
        </Alert>
      )}
      <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(320px, 1fr))', gap: 2 }}>
        {(list.data ?? []).map(d => (
          <Card key={d.id} variant="outlined">
            <CardActionArea onClick={() => navigate(`/data/pipelines/${d.id}`)}>
              <CardContent>
                <Typography variant="subtitle1" fontWeight={700} noWrap>{d.name}</Typography>
                <Typography variant="body2" color="text.secondary" noWrap>{describe(d)}</Typography>
                <Stack direction="row" spacing={1} sx={{ mt: 1 }}>
                  <Chip size="small" label={`${d.spec.nodes.length} steps`} />
                  <Chip size="small" variant="outlined" label={`edited ${new Date(d.last_modified_at).toLocaleDateString()}`} />
                </Stack>
              </CardContent>
            </CardActionArea>
            <Stack direction="row" justifyContent="flex-end" sx={{ px: 1, pb: 1 }}>
              <Tooltip title="Delete (run history is kept)">
                <IconButton size="small" onClick={() => window.confirm(`Delete "${d.name}"?`) && remove.mutate(d.id)}><DeleteIcon fontSize="small" /></IconButton>
              </Tooltip>
            </Stack>
          </Card>
        ))}
      </Box>
    </Box>
  );
}
