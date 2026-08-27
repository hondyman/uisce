import { useState, useEffect } from 'react';
import { Tenant } from '../types';
import { JSX } from 'react';
import apiClient from '../utils/apiClient';
import { useTheme } from '@mui/material/styles';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import TextField from '@mui/material/TextField';
import Button from '@mui/material/Button';
import Paper from '@mui/material/Paper';

export default function HomePage(): JSX.Element {
  const theme = useTheme();
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [newTenantName, setNewTenantName] = useState('');
  const [newTenantInstance, setNewTenantInstance] = useState('');

  useEffect(() => {
    apiClient('tenants')
      .then((res) => res.json())
      .then((data: Tenant[]) => setTenants(data));
  }, []);

  const handleCreateTenant = () => {
    apiClient('tenants', {
      method: 'POST',
      body: JSON.stringify({
        name: newTenantName,
        instance: newTenantInstance,
      }),
    })
      .then((res) => res.json())
      .then((newTenant: Tenant) => {
        setTenants((prevTenants) => [...prevTenants, newTenant]);
        setNewTenantName('');
        setNewTenantInstance('');
      });
  };

  const isDark = theme.palette.mode === 'dark';

  return (
    <Box
      sx={{
        minHeight: '100vh',
        background: isDark
          ? 'linear-gradient(to bottom right, #020617, #0f172a, #020617)'
          : 'linear-gradient(to bottom right, #f8fafc, #eff6ff, #f1f5f9)',
        p: 4,
      }}
    >
      <Box sx={{ maxWidth: 900, mx: 'auto' }}>
        <Paper
          sx={{
            p: 4,
            borderRadius: '12px',
            border: '1px solid',
            borderColor: isDark ? 'rgba(255,255,255,0.1)' : 'rgb(226, 232, 240)',
            boxShadow: isDark ? 'none' : '0 10px 15px -3px rgba(0, 0, 0, 0.1)',
            backgroundColor: isDark ? '#1e293b' : 'white',
          }}
        >
          <Typography
            variant="h4"
            sx={{
              fontWeight: 700,
              color: isDark ? '#f1f5f9' : '#0f172a',
              mb: 1,
            }}
          >
            Tenants and Instances
          </Typography>
          <Typography
            sx={{
              color: isDark ? '#94a3b8' : '#475569',
              mb: 4,
            }}
          >
            Manage your tenant configurations and instances
          </Typography>

          <Box sx={{ mb: 3 }}>
            <Typography
              variant="body2"
              sx={{
                color: isDark ? '#94a3b8' : '#64748b',
                mb: 1,
              }}
            >
              Loaded tenants:{' '}
              <Typography
                component="span"
                sx={{
                  fontWeight: 600,
                  color: isDark ? '#f1f5f9' : '#0f172a',
                }}
              >
                {tenants.length}
              </Typography>
            </Typography>
          </Box>

          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
            <Box
              sx={{
                display: 'grid',
                gridTemplateColumns: { xs: '1fr', md: '1fr 1fr' },
                gap: 2,
              }}
            >
              <Box>
                <Typography
                  component="label"
                  variant="body2"
                  sx={{
                    display: 'block',
                    fontWeight: 500,
                    color: isDark ? '#f1f5f9' : '#334155',
                    mb: 1,
                  }}
                >
                  New Tenant Name
                </Typography>
                <TextField
                  fullWidth
                  size="small"
                  value={newTenantName}
                  onChange={(e) => setNewTenantName(e.target.value)}
                  placeholder="Enter tenant name"
                  sx={{
                    '& .MuiOutlinedInput-root': {
                      borderRadius: '8px',
                      backgroundColor: isDark ? '#1e293b' : 'white',
                      color: isDark ? '#f1f5f9' : '#0f172a',
                      '& fieldset': {
                        borderColor: isDark ? 'rgba(255,255,255,0.1)' : '#cbd5e1',
                      },
                      '&:hover fieldset': {
                        borderColor: isDark ? 'rgba(255,255,255,0.2)' : '#94a3b8',
                      },
                      '&.Mui-focused fieldset': {
                        borderColor: '#3b82f6',
                      },
                    },
                    '& input::placeholder': {
                      color: isDark ? '#64748b' : '#94a3b8',
                      opacity: 1,
                    },
                  }}
                />
              </Box>
              <Box>
                <Typography
                  component="label"
                  variant="body2"
                  sx={{
                    display: 'block',
                    fontWeight: 500,
                    color: isDark ? '#f1f5f9' : '#334155',
                    mb: 1,
                  }}
                >
                  New Instance Name
                </Typography>
                <TextField
                  fullWidth
                  size="small"
                  value={newTenantInstance}
                  onChange={(e) => setNewTenantInstance(e.target.value)}
                  placeholder="Enter instance name"
                  sx={{
                    '& .MuiOutlinedInput-root': {
                      borderRadius: '8px',
                      backgroundColor: isDark ? '#1e293b' : 'white',
                      color: isDark ? '#f1f5f9' : '#0f172a',
                      '& fieldset': {
                        borderColor: isDark ? 'rgba(255,255,255,0.1)' : '#cbd5e1',
                      },
                      '&:hover fieldset': {
                        borderColor: isDark ? 'rgba(255,255,255,0.2)' : '#94a3b8',
                      },
                      '&.Mui-focused fieldset': {
                        borderColor: '#3b82f6',
                      },
                    },
                    '& input::placeholder': {
                      color: isDark ? '#64748b' : '#94a3b8',
                      opacity: 1,
                    },
                  }}
                />
              </Box>
            </Box>

            <Button
              variant="contained"
              onClick={handleCreateTenant}
              sx={{
                display: 'inline-flex',
                alignItems: 'center',
                px: 3,
                py: 1.5,
                backgroundColor: isDark ? '#3b82f6' : '#2563eb',
                '&:hover': {
                  backgroundColor: isDark ? '#2563eb' : '#1d4ed8',
                },
                fontWeight: 500,
                borderRadius: '8px',
                boxShadow: isDark ? 'none' : '0 1px 2px 0 rgba(0, 0, 0, 0.05)',
                '&:hover': {
                  boxShadow: isDark ? 'none' : '0 4px 6px -1px rgba(0, 0, 0, 0.1)',
                },
                alignSelf: 'flex-start',
              }}
            >
              Create Tenant
            </Button>
          </Box>
        </Paper>
      </Box>
    </Box>
  );
}
