import React, { useState, useEffect } from 'react';
import {
  Dialog, DialogTitle, DialogContent, DialogActions,
  Button, TextField, Typography, Box, CircularProgress, Alert
} from '@mui/material';

interface GenericBOFormModalProps {
  boKey: string;
  recordId: string | null;
  open: boolean;
  onClose: () => void;
  onSaved: () => void;
}

export const GenericBOFormModal: React.FC<GenericBOFormModalProps> = ({
  boKey,
  recordId,
  open,
  onClose,
  onSaved,
}) => {
  const [formData, setFormData] = useState<Record<string, any>>({});
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!open || !recordId || !boKey) {
      setFormData({});
      return;
    }

    setLoading(true);
    setError(null);
    fetch(`/api/v1/bo/${boKey}/records/${recordId}`)
      .then(async (res) => {
        if (!res.ok) throw new Error(await res.text());
        return res.json();
      })
      .then((data) => setFormData(data || {}))
      .catch((err) => setError(err.message || 'Failed to load record'))
      .finally(() => setLoading(false));
  }, [open, recordId, boKey]);

  const handleSave = async () => {
    setSaving(true);
    try {
      const res = await fetch(`/api/v1/bo/${boKey}/records/${recordId}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(formData),
      });

      if (!res.ok) throw new Error(await res.text());
      onSaved();
      onClose();
    } catch (err: any) {
      setError(err.message || 'Save failed');
    } finally {
      setSaving(false);
    }
  };

  return (
    <Dialog
      open={open}
      onClose={onClose}
      maxWidth="sm"
      fullWidth
      PaperProps={{ sx: { bgcolor: '#071526', color: '#F8FAFC', border: '1px solid #1E293B' } }}
    >
      <DialogTitle sx={{ borderBottom: '1px solid #1E293B' }}>
        <Typography variant="subtitle1" fontWeight={700} color="#00D4FF">
          Inspect & Edit: {boKey} ({recordId})
        </Typography>
      </DialogTitle>

      <DialogContent sx={{ py: 2 }}>
        {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
        {loading ? (
          <Box sx={{ display: 'flex', justifyContent: 'center', p: 4 }}>
            <CircularProgress size={24} />
          </Box>
        ) : (
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, mt: 1 }}>
            {Object.keys(formData).map((key) => {
              const isReadOnly = key === 'id' || key === 'tenant_id' || key === 'created_at';
              return (
                <TextField
                  key={key}
                  size="small"
                  label={key.replace(/_/g, ' ').toUpperCase()}
                  value={formData[key] ?? ''}
                  onChange={(e) => setFormData((prev) => ({ ...prev, [key]: e.target.value }))}
                  disabled={isReadOnly}
                  sx={{
                    bgcolor: '#0B1E36',
                    input: { color: '#F8FAFC', fontSize: 12 },
                    label: { color: '#94A3B8', fontSize: 12 },
                    '& .MuiOutlinedInput-notchedOutline': { borderColor: '#1E293B' },
                  }}
                />
              );
            })}
          </Box>
        )}
      </DialogContent>

      <DialogActions sx={{ borderTop: '1px solid #1E293B', px: 3, py: 1.5 }}>
        <Button onClick={onClose} sx={{ color: '#94A3B8', textTransform: 'none' }}>Cancel</Button>
        <Button
          variant="contained"
          onClick={handleSave}
          disabled={saving || loading}
          sx={{ bgcolor: '#0284C7', textTransform: 'none' }}
        >
          {saving ? 'Saving...' : 'Save Changes'}
        </Button>
      </DialogActions>
    </Dialog>
  );
};
