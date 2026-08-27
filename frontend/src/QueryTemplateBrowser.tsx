import { useState, useEffect } from 'react';
import { listQueryTemplates } from './api';
import type { QueryTemplateMeta } from './types';
import { useTheme } from '@mui/material/styles';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import Paper from '@mui/material/Paper';
import Chip from '@mui/material/Chip';
import CircularProgress from '@mui/material/CircularProgress';

const Badge = ({ children }: { children: React.ReactNode }) => (
  <Chip label={children} size="small" color="primary" variant="outlined" />
);

interface QueryTemplateBrowserProps {
  datasourceId: string;
  onSelect: (template: QueryTemplateMeta) => void;
}

export default function QueryTemplateBrowser({ datasourceId, onSelect }: QueryTemplateBrowserProps) {
  const theme = useTheme();
  const isDark = theme.palette.mode === 'dark';
  const [templates, setTemplates] = useState<QueryTemplateMeta[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    listQueryTemplates(datasourceId)
      .then(setTemplates)
      .catch((e) => { import('./utils/devLogger').then(({ devError }) => devError(e)).catch(() => {}); })
      .finally(() => setLoading(false));
  }, [datasourceId]);

  if (loading) {
    return (
      <Box sx={{ textAlign: 'center', py: 2 }}>
        <CircularProgress size={24} />
      </Box>
    );
  }

  return (
    <Box sx={{ mt: 2 }}>
      <Typography variant="subtitle1" sx={{ fontWeight: 600, mb: 2 }}>
        Query Templates
      </Typography>
      <Box
        sx={{
          display: 'grid',
          gridTemplateColumns: { xs: '1fr', sm: 'repeat(2, 1fr)', md: 'repeat(3, 1fr)' },
          gap: 2,
        }}
      >
        {templates.map((t) => (
          <Paper
            key={t.id}
            onClick={() => onSelect(t)}
            title={t.description}
            sx={{
              p: 2,
              cursor: 'pointer',
              transition: 'transform 0.2s, box-shadow 0.2s',
              '&:hover': {
                transform: 'translateY(-2px)',
                boxShadow: 3,
              },
              display: 'flex',
              flexDirection: 'column',
              gap: 1,
            }}
          >
            <Typography variant="subtitle2" fontWeight={600}>
              {t.name}
            </Typography>
            <Typography variant="body2" color="text.secondary" sx={{ flex: 1 }}>
              {t.description}
            </Typography>
            <Box sx={{ mt: 1 }}>
              {t.certified && <Badge>Certified</Badge>}
            </Box>
          </Paper>
        ))}
      </Box>
    </Box>
  );
}
