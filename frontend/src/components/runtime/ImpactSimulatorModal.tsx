import React, { useState } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  TextField,
  Box,
  Typography,
  LinearProgress,
  List,
  ListItem,
  ListItemText,
  IconButton,
  Divider,
  Chip,
} from '@mui/material';
import { Warning, Close, TrendingUp, Layers, Group, Storage, ShowChart } from "@mui/icons-material";;

interface ImpactSimulatorModalProps {
  open: boolean;
  onClose: () => void;
  initialBOK?: string;
}

interface BlastRadiusItem {
  category: string;
  icon: React.ReactNode;
  count: number;
  detail: string;
  severity: 'high' | 'medium' | 'low';
}

export const ImpactSimulatorModal: React.FC<ImpactSimulatorModalProps> = ({
  open,
  onClose,
  initialBOK = '',
}) => {
  const [boKey, setBoKey] = useState(initialBOK);
  const [isSimulating, setIsSimulating] = useState(false);
  const [blastRadius, setBlastRadius] = useState<BlastRadiusItem[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  const simulate = async () => {
    if (!boKey.trim()) return;

    setIsSimulating(true);
    setError(null);
    setBlastRadius(null);

    try {
      const res = await fetch(`/api/v1/ai/simulate?bo_key=${encodeURIComponent(boKey)}`, {
        credentials: 'include',
      });

      if (!res.ok) {
        const errData = await res.json().catch(() => ({}));
        throw new Error(errData.message || `HTTP ${res.status}`);
      }

      const data = await res.json();
      setBlastRadius(data.blast_radius || []);
    } catch (e: any) {
      setError(e.message || 'Simulation failed');
    } finally {
      setIsSimulating(false);
    }
  };

  const handleClose = () => {
    setBoKey('');
    setBlastRadius(null);
    setError(null);
    setIsSimulating(false);
    onClose();
  };

  const severityColor = (s: 'high' | 'medium' | 'low') => {
    switch (s) {
      case 'high': return '#ef4444';
      case 'medium': return '#f59e0b';
      case 'low': return '#22c55e';
    }
  };

  return (
    <Dialog
      open={open}
      onClose={handleClose}
      maxWidth="sm"
      fullWidth
      PaperProps={{
        sx: {
          bgcolor: '#0f172a',
          border: '1px solid #334155',
          color: '#f8fafc',
        },
      }}
    >
      <DialogTitle sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
          <Warning className="w-5 h-5 text-amber-400" />
          <Typography variant="h6" fontWeight="600">
            Impact Simulator
          </Typography>
        </Box>
        <IconButton onClick={handleClose} size="small" sx={{ color: '#94a3b8' }}>
          <Close className="w-4 h-4" />
        </IconButton>
      </DialogTitle>

      <Divider sx={{ borderColor: '#1e293b' }} />

      <DialogContent>
        <Box sx={{ mb: 3 }}>
          <Typography variant="body2" color="#94a3b8" mb={1.5}>
            Enter a Business Object key to calculate its blast radius — downstream consumers,
            dependent calculations, and cross-tenant exposure.
          </Typography>
          <Box sx={{ display: 'flex', gap: 1.5 }}>
            <TextField
              fullWidth
              size="small"
              placeholder="e.g. bo_customer, bo_order, bo_invoice"
              value={boKey}
              onChange={(e) => setBoKey(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && simulate()}
              disabled={isSimulating}
              sx={{
                input: { color: '#f8fafc', bgcolor: '#1e293b' },
              }}
            />
            <Button
              variant="contained"
              onClick={simulate}
              disabled={isSimulating || !boKey.trim()}
              sx={{ bgcolor: '#6366f1', '&:hover': { bgcolor: '#4f46e5' }, whiteSpace: 'nowrap' }}
            >
              Simulate
            </Button>
          </Box>
        </Box>

        {isSimulating && <LinearProgress sx={{ mb: 2, bgcolor: '#1e293b' }} />}

        {error && (
          <Box sx={{ p: 2, bgcolor: '#7f1d1d', borderRadius: 1, mb: 2 }}>
            <Typography variant="body2" color="#fca5a5">{error}</Typography>
          </Box>
        )}

        {blastRadius && (
          <Box>
            <Typography variant="subtitle2" color="#94a3b8" mb={1.5}>
              BLAST RADIUS for{' '}
              <Chip label={boKey} size="small" sx={{ bgcolor: '#1e293b', color: '#38bdf8', ml: 0.5 }} />
            </Typography>

            <List dense disablePadding>
              {blastRadius.map((item, idx) => (
                <ListItem
                  key={idx}
                  sx={{
                    bgcolor: '#1e293b',
                    borderRadius: 1,
                    mb: 0.75,
                    borderLeft: `3px solid ${severityColor(item.severity)}`,
                  }}
                >
                  <Box sx={{ mr: 1.5, color: '#94a3b8' }}>{item.icon}</Box>
                  <ListItemText
                    primary={
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                        <Typography variant="body2" fontWeight="500">
                          {item.category}
                        </Typography>
                        <Chip
                          label={item.count}
                          size="small"
                          sx={{
                            height: 16,
                            fontSize: '0.65rem',
                            bgcolor: severityColor(item.severity),
                            color: '#fff',
                          }}
                        />
                      </Box>
                    }
                    secondary={item.detail}
                    secondaryTypographyProps={{ variant: 'caption', color: '#64748b' }}
                  />
                </ListItem>
              ))}
            </List>

            <Box sx={{ mt: 2, p: 1.5, bgcolor: '#1e293b', borderRadius: 1 }}>
              <Typography variant="caption" color="#64748b">
                Impact calculated at {new Date().toLocaleTimeString()} — results reflect current
                metadata graph state and may differ from runtime behavior.
              </Typography>
            </Box>
          </Box>
        )}

        {!blastRadius && !isSimulating && !error && (
          <Box
            sx={{
              textAlign: 'center',
              py: 4,
              color: '#475569',
            }}
          >
            <ShowChart className="w-10 h-10 mx-auto mb-2 opacity-30" />
            <Typography variant="body2">
              Enter a BO key and click Simulate to see blast radius
            </Typography>
          </Box>
        )}
      </DialogContent>

      <Divider sx={{ borderColor: '#1e293b' }} />

      <DialogActions sx={{ px: 3, py: 1.5 }}>
        <Button onClick={handleClose} size="small" sx={{ color: '#94a3b8' }}>
          Close
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export const defaultBlastRadiusItems: BlastRadiusItem[] = [
  {
    category: 'Downstream Reports',
    icon: <TrendingUp style={{ fontSize: 16  }}  />,
    count: 14,
    detail: 'Dashboards, scheduled reports, and ad-hoc queries referencing this BO',
    severity: 'high',
  },
  {
    category: 'Dependent BOs',
    icon: <Layers style={{ fontSize: 16  }}  />,
    count: 7,
    detail: 'Business Objects that reference columns or calculations from this BO',
    severity: 'medium',
  },
  {
    category: 'Active Users',
    icon: <Group style={{ fontSize: 16  }}  />,
    count: 342,
    detail: 'Users with at least one query or page referencing this BO in the last 30 days',
    severity: 'medium',
  },
  {
    category: 'Schema Dependencies',
    icon: <Storage style={{ fontSize: 16  }}  />,
    count: 3,
    detail: 'Source tables and external schemas this BO depends on',
    severity: 'low',
  },
];
