import React, { useState, useEffect } from 'react';
import {
  Box,
  Typography,
  TextField,
  FormControlLabel,
  Switch,
  Paper,
  Chip,
  IconButton,
  Tooltip,
  Collapse,
  CircularProgress,
  Button,
} from '@mui/material';
import { useTheme } from '@mui/material/styles';
import {
  SlidersHorizontal,
  Play,
  RotateCcw,
  Terminal,
  Copy,
  Check,
  Lock,
  UserCheck,
} from 'lucide-react';
import { ReportParameter } from './builderSerialization';
import { resolveParameterDefaults, UserSessionProfile } from './ParameterContextEngine';

interface ReportParametersToolbarProps {
  parameters: ReportParameter[];
  values: Record<string, any>;
  onChange: (paramName: string, value: any) => void;
  onRun: (paramValues: Record<string, any>) => void;
  currentUserProfile?: UserSessionProfile;
  loading?: boolean;
  reportId?: string;
  reportKey?: string;
}

export const ReportParametersToolbar: React.FC<ReportParametersToolbarProps> = ({
  parameters,
  values,
  onChange,
  onRun,
  currentUserProfile,
  loading = false,
  reportId = 'rep-custom-001',
  reportKey = 'daily_institutional_client_valuation',
}) => {
  const theme = useTheme();
  const isDark = theme.palette.mode === 'dark';

  const [showCodeSnippet, setShowCodeSnippet] = useState(false);
  const [copiedLang, setCopiedLang] = useState<string | null>(null);
  const [selectedLang, setSelectedLang] = useState<'curl' | 'ts' | 'python'>('curl');
  const [collapsed, setCollapsed] = useState(false);

  const C = {
    bg: isDark ? '#0B132B' : '#FFFFFF',
    bgAlt: isDark ? 'rgba(15, 23, 42, 0.6)' : '#F8FAFC',
    inputBg: isDark ? 'rgba(15, 23, 42, 0.6)' : '#FFFFFF',
    text: isDark ? '#F8FAFC' : '#0F172A',
    textMuted: isDark ? '#94A3B8' : '#64748B',
    border: isDark ? 'rgba(255,255,255,0.08)' : 'rgba(0,0,0,0.08)',
    accent: '#0D9488',
    accentLight: isDark ? '#2DD4BF' : '#0D9488',
    accentBg: isDark ? 'rgba(13, 148, 136, 0.15)' : 'rgba(13, 148, 136, 0.08)',
    codeBg: isDark ? 'rgba(0,0,0,0.5)' : '#0F172A',
  };

  // Prepopulate with user context & relative date defaults on mount or profile change
  useEffect(() => {
    const defaults = resolveParameterDefaults(parameters, currentUserProfile);
    parameters.forEach((p) => {
      if (values[p.name] === undefined && defaults[p.name] !== undefined) {
        onChange(p.name, defaults[p.name]);
      }
    });
  }, [parameters, currentUserProfile, values, onChange]);

  const handleResetDefaults = () => {
    const defaults = resolveParameterDefaults(parameters, currentUserProfile);
    parameters.forEach((p) => {
      onChange(p.name, defaults[p.name] !== undefined ? defaults[p.name] : (p.defaultValue || ''));
    });
  };

  const handleCopyCode = (text: string, lang: string) => {
    navigator.clipboard.writeText(text);
    setCopiedLang(lang);
    setTimeout(() => setCopiedLang(null), 2000);
  };

  const targetId = reportId || reportKey || 'rep-custom-001';

  const codeSnippets = {
    curl: `curl -X POST "http://localhost:8080/api/v1/reports/${targetId}/render" \\
  -H "Authorization: Bearer <YOUR_JWT_TOKEN>" \\
  -H "X-Tenant-ID: <TENANT_ID>" \\
  -H "Content-Type: application/json" \\
  -d '${JSON.stringify({ parameters: values }, null, 2)}'`,

    ts: `import { UuisceClient } from '@uuisce/sdk';

const client = new UuisceClient({
  apiKey: process.env.UUISCE_API_KEY,
  tenantId: 'acme_wealth'
});

// Run report with runtime parameter overrides
const reportResult = await client.reports.render('${targetId}', {
  parameters: ${JSON.stringify(values, null, 4)}
});

console.log(\`Rendered \${reportResult.rowCount} rows:\`, reportResult.rows);`,

    python: `from uuisce import UuisceClient

client = UuisceClient(api_key="your_api_key", tenant_id="acme_wealth")

# Execute report with runtime parameters
result = client.reports.render(
    report_id="${targetId}",
    parameters=${JSON.stringify(values, null, 4).replace(/true/g, 'True').replace(/false/g, 'False')}
)

print(f"Generated {len(result['rows'])} rows")`,
  };

  return (
    <Paper
      elevation={0}
      sx={{
        p: 2,
        mb: 2,
        bgcolor: C.bg,
        color: C.text,
        borderRadius: 2,
        border: `1px solid ${C.border}`,
        boxShadow: isDark ? 'none' : '0 1px 3px rgba(0,0,0,0.05)',
      }}
    >
      {/* Header Bar */}
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2, pb: 1.2, borderBottom: `1px solid ${C.border}` }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
          <SlidersHorizontal size={18} color={C.accentLight} />
          <Box>
            <Typography variant="subtitle2" sx={{ fontWeight: 800, color: C.text, display: 'flex', alignItems: 'center', gap: 1 }}>
              Runtime Execution Parameters
              <Chip
                size="small"
                label={`${parameters.length} Active`}
                sx={{ height: 18, fontSize: '0.65rem', fontWeight: 700, bgcolor: C.accentBg, color: C.accentLight }}
              />
            </Typography>
            <Typography variant="caption" sx={{ color: C.textMuted }}>
              Provide parameter inputs to drive report query filters, dynamic expressions, and subtotal groupings.
            </Typography>
          </Box>
        </Box>

        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <Button
            size="small"
            variant="text"
            onClick={() => setCollapsed(!collapsed)}
            startIcon={<SlidersHorizontal size={14} style={{ transform: collapsed ? 'rotate(-90deg)' : 'none', transition: 'transform 0.2s' }} />}
            sx={{ textTransform: 'none', fontSize: '0.72rem', color: C.textMuted, '&:hover': { color: C.accentLight } }}
          >
            {collapsed ? 'Expand' : 'Collapse'}
          </Button>
          <Button
            size="small"
            variant="text"
            onClick={() => setShowCodeSnippet(!showCodeSnippet)}
            startIcon={<Terminal size={14} />}
            sx={{ textTransform: 'none', fontSize: '0.72rem', color: C.textMuted, '&:hover': { color: C.accentLight } }}
          >
            {showCodeSnippet ? 'Hide SDK / API' : 'SDK & API Guide'}
          </Button>
        </Box>
      </Box>

      {/* Parameter Controls Grid */}
      {parameters.length === 0 ? (
        <Box sx={{ p: 1.5, textAlign: 'center', bgcolor: C.bgAlt, borderRadius: 1.5 }}>
          <Typography variant="caption" sx={{ color: C.textMuted, display: 'block' }}>
            No parameters configured. Parameters can be added in the <strong>Design</strong> tab (Parameters palette) and bound to query filters in the <strong>Filters</strong> tab.
          </Typography>
        </Box>
      ) : (
        <Box sx={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 1.5 }}>
          {parameters.map((p) => {
            const currentVal = values[p.name] !== undefined ? values[p.name] : (p.defaultValue || '');
            const isLocked = Boolean(p.lockForUser);

            if (p.type === 'boolean') {
              return (
                <Box
                  key={p.id}
                  sx={{
                    p: 0.8,
                    px: 1.5,
                    bgcolor: C.bgAlt,
                    borderRadius: 1.5,
                    border: `1px solid ${C.border}`,
                    display: 'flex',
                    alignItems: 'center',
                  }}
                >
                  <FormControlLabel
                    control={
                      <Switch
                        size="small"
                        disabled={isLocked}
                        checked={Boolean(currentVal === true || currentVal === 'true')}
                        onChange={(e) => onChange(p.name, e.target.checked)}
                      />
                    }
                    label={
                      <Box>
                        <Typography variant="caption" sx={{ fontWeight: 700, color: C.text, display: 'flex', alignItems: 'center', gap: 0.5 }}>
                          {p.prompt || p.name}
                          {isLocked && <Lock size={11} color="#F59E0B" />}
                        </Typography>
                        <Typography variant="caption" sx={{ color: C.textMuted, fontSize: '0.65rem' }}>
                          @{p.name}
                        </Typography>
                      </Box>
                    }
                  />
                </Box>
              );
            }

            return (
              <Box key={p.id} sx={{ minWidth: 180, flexGrow: 1, maxWidth: 260 }}>
                <TextField
                  fullWidth
                  size="small"
                  disabled={isLocked}
                  label={p.prompt || p.name}
                  type={p.type === 'number' ? 'number' : p.type === 'date' ? 'date' : 'text'}
                  value={Array.isArray(currentVal) ? currentVal.join(', ') : currentVal}
                  onChange={(e) => {
                    const v = e.target.value;
                    if (p.allowMultiple && v.includes(',')) {
                      onChange(p.name, v.split(',').map(s => s.trim()).filter(Boolean));
                    } else {
                      onChange(p.name, v);
                    }
                  }}
                  helperText={
                    <span style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                      {isLocked ? (
                        <>
                          <Lock size={10} color="#F59E0B" />
                          <span style={{ color: '#F59E0B' }}>Locked for user profile</span>
                        </>
                      ) : p.sourceType === 'context' ? (
                        <>
                          <UserCheck size={10} color="#2DD4BF" />
                          <span>Context: @{p.name}</span>
                        </>
                      ) : (
                        `Variable: @${p.name}${p.allowMultiple ? ' (Array)' : ''}`
                      )}
                    </span>
                  }
                  InputLabelProps={p.type === 'date' ? { shrink: true } : undefined}
                  InputProps={{
                    sx: {
                      fontFamily: 'monospace',
                      fontSize: '0.82rem',
                      fontWeight: 600,
                      bgcolor: isLocked ? (isDark ? 'rgba(255,255,255,0.02)' : '#F1F5F9') : C.inputBg,
                      color: C.text,
                    },
                  }}
                  FormHelperTextProps={{
                    sx: { color: C.textMuted, fontSize: '0.65rem', mt: 0.3 },
                  }}
                />
              </Box>
            );
          })}

          {/* Action Buttons */}
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, ml: 'auto', alignSelf: 'flex-start', pt: 0.5 }}>
            <Tooltip title="Reset to Default Values">
              <IconButton size="small" onClick={handleResetDefaults} sx={{ color: C.textMuted }}>
                <RotateCcw size={16} />
              </IconButton>
            </Tooltip>

            <Button
              variant="contained"
              size="small"
              onClick={() => onRun(values)}
              disabled={loading}
              startIcon={loading ? <CircularProgress size={14} color="inherit" /> : <Play size={14} />}
              sx={{
                bgcolor: C.accent,
                color: '#FFF',
                fontWeight: 800,
                textTransform: 'none',
                fontSize: '0.78rem',
                px: 2,
                py: 0.8,
                borderRadius: 1.5,
                boxShadow: '0 2px 10px rgba(13, 148, 136, 0.3)',
                '&:hover': { bgcolor: '#0F766E' },
              }}
            >
              {loading ? 'Running...' : 'Run with Parameters'}
            </Button>
          </Box>
        </Box>
      )}

      {/* Expandable SDK & API Guide */}
      <Collapse in={showCodeSnippet}>
        <Box sx={{ mt: 2, pt: 2, borderTop: `1px solid ${C.border}` }}>
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1.5 }}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
              <Terminal size={16} color={isDark ? '#38BDF8' : '#0284C7'} />
              <Typography variant="caption" sx={{ fontWeight: 700, color: isDark ? '#38BDF8' : '#0284C7' }}>
                Programmatic Execution (SDK &amp; REST API)
              </Typography>
            </Box>

            <Box sx={{ display: 'flex', gap: 0.5 }}>
              {(['curl', 'ts', 'python'] as const).map((lang) => (
                <Chip
                  key={lang}
                  size="small"
                  label={lang === 'curl' ? 'cURL' : lang === 'ts' ? 'TypeScript SDK' : 'Python SDK'}
                  onClick={() => setSelectedLang(lang)}
                  sx={{
                    fontSize: '0.65rem',
                    fontWeight: 700,
                    cursor: 'pointer',
                    bgcolor: selectedLang === lang ? C.accent : C.bgAlt,
                    color: selectedLang === lang ? '#FFF' : C.textMuted,
                  }}
                />
              ))}
            </Box>
          </Box>

          <Paper
            sx={{
              p: 1.5,
              bgcolor: C.codeBg,
              border: `1px solid ${C.border}`,
              borderRadius: 1.5,
              position: 'relative',
            }}
          >
            <IconButton
              size="small"
              onClick={() => handleCopyCode(codeSnippets[selectedLang], selectedLang)}
              sx={{ position: 'absolute', top: 8, right: 8, color: '#94A3B8', '&:hover': { color: '#FFF' } }}
            >
              {copiedLang === selectedLang ? <Check size={14} color="#10B981" /> : <Copy size={14} />}
            </IconButton>
            <pre style={{ margin: 0, fontFamily: 'monospace', fontSize: '0.75rem', color: '#A5B4FC', overflowX: 'auto' }}>
              {codeSnippets[selectedLang]}
            </pre>
          </Paper>
        </Box>
      </Collapse>
    </Paper>
  );
};

export default ReportParametersToolbar;
