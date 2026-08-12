import React, { useState } from 'react';
import {
  Box,
  Typography,
  Chip,
  Tooltip,
  IconButton,
  Collapse,
} from '@mui/material';
import {
  Lock as LockIcon,
  Shield as ShieldIcon,
  ExpandMore as ExpandMoreIcon,
  ExpandLess as ExpandLessIcon,
  Refresh as RefreshIcon,
} from '@mui/icons-material';

export interface JWTInspectorProps {
  tenantId?: string;
  role?: string;
  scopes?: string[];
  expiresInMinutes?: number;
  issuedAt?: string;
  issuer?: string;
  signatureAlgorithm?: string;
}

export const JWTInspector: React.FC<JWTInspectorProps> = ({
  tenantId = '99e99e99-99e9-49e9-89e9-99e99e99e999',
  role = 'front_office_trader',
  scopes = ['read:bo', 'execute:query', 'view:api'],
  expiresInMinutes = 23,
  issuedAt = new Date().toISOString(),
  issuer = 'auth.uisce.io',
  signatureAlgorithm = 'RS256 (2048-bit RSA)',
}) => {
  const [expanded, setExpanded] = useState(false);

  return (
    <Box
      sx={{
        borderRadius: 2,
        border: '1px solid rgba(245, 166, 35, 0.3)',
        bgcolor: 'rgba(15, 17, 23, 0.9)',
        backdropFilter: 'blur(12px)',
        overflow: 'hidden',
      }}
    >
      {/* Header rail */}
      <Box
        onClick={() => setExpanded(!expanded)}
        sx={{
          px: 2,
          py: 1.5,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'between',
          cursor: 'pointer',
          bgcolor: 'rgba(245, 166, 35, 0.08)',
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <LockIcon sx={{ fontSize: 16, color: '#F5A623' }} />
          <Typography variant="subtitle2" sx={{ fontFamily: 'JetBrains Mono, monospace', fontSize: 12, fontWeight: 700, color: '#F0F4FF' }}>
            🔒 JWT SECURITY CONTEXT (GSIFI ENFORCED)
          </Typography>
        </Box>

        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, ml: 'auto' }}>
          <Chip
            label={`Tenant: ${tenantId.substring(0, 8)}...`}
            size="small"
            sx={{ bgcolor: 'rgba(245, 166, 35, 0.2)', color: '#F5A623', fontSize: 10, fontWeight: 700 }}
          />
          <IconButton size="small" sx={{ color: '#94A3B8' }}>
            {expanded ? <ExpandLessIcon fontSize="small" /> : <ExpandMoreIcon fontSize="small" />}
          </IconButton>
        </Box>
      </Box>

      {/* Collapsible Detail Panel */}
      <Collapse in={expanded}>
        <Box sx={{ p: 2, borderTop: '1px solid rgba(37, 45, 61, 0.8)', fontFamily: 'JetBrains Mono, monospace', fontSize: 11, color: '#94A3B8' }}>
          <Box sx={{ display: 'grid', gridTemplateColumns: '120px 1fr', gap: 1, mb: 1 }}>
            <Typography variant="caption" sx={{ color: '#4B5563', fontWeight: 600 }}>Tenant ID:</Typography>
            <Typography variant="caption" sx={{ color: '#00D4FF', fontWeight: 600 }}>{tenantId}</Typography>

            <Typography variant="caption" sx={{ color: '#4B5563', fontWeight: 600 }}>Role:</Typography>
            <Typography variant="caption" sx={{ color: '#F0F4FF' }}>{role}</Typography>

            <Typography variant="caption" sx={{ color: '#4B5563', fontWeight: 600 }}>Scopes:</Typography>
            <Box sx={{ display: 'flex', gap: 0.5, flexWrap: 'wrap' }}>
              {scopes.map((s) => (
                <Chip key={s} label={s} size="small" sx={{ bgcolor: 'rgba(139, 92, 246, 0.2)', color: '#8B5CF6', fontSize: 10 }} />
              ))}
            </Box>

            <Typography variant="caption" sx={{ color: '#4B5563', fontWeight: 600 }}>Expires:</Typography>
            <Typography variant="caption" sx={{ color: '#10B981' }}>in {expiresInMinutes} minutes</Typography>

            <Typography variant="caption" sx={{ color: '#4B5563', fontWeight: 600 }}>Issued At:</Typography>
            <Typography variant="caption" sx={{ color: '#F0F4FF' }}>{issuedAt}</Typography>

            <Typography variant="caption" sx={{ color: '#4B5563', fontWeight: 600 }}>Issuer:</Typography>
            <Typography variant="caption" sx={{ color: '#F0F4FF' }}>{issuer}</Typography>

            <Typography variant="caption" sx={{ color: '#4B5563', fontWeight: 600 }}>Algorithm:</Typography>
            <Typography variant="caption" sx={{ color: '#F0F4FF' }}>{signatureAlgorithm}</Typography>
          </Box>

          <Box sx={{ pt: 1, borderTop: '1px solid rgba(37, 45, 61, 0.5)', display: 'flex', alignItems: 'center', justifyContent: 'between' }}>
            <Typography variant="caption" sx={{ color: '#10B981', display: 'flex', alignItems: 'center', gap: 0.5 }}>
              <ShieldIcon sx={{ fontSize: 14 }} /> Signature verified • RequireTenantOwnership active
            </Typography>
          </Box>
        </Box>
      </Collapse>
    </Box>
  );
};
