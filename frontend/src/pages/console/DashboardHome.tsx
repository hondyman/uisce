import { Grid, Card, CardContent, Typography, Stack, Box, Alert } from '@mui/material';
import { useEffect, useState } from 'react';
import { ConsoleLayout } from '../../layout/ConsoleLayout';
import { ConsoleBreadcrumbs } from '../../layout/ConsoleBreadcrumbs';
import {
  useComplianceSummary,
  useRiskSummary,
  useSparklines,
  useETLHealth,
  useAlerts,
} from '../../api/dashboard';
import { SparklineCard } from '../../components/charts/SparklineCard';
import { StatusBadge } from '../../components/design/StatusBadge';

export function DashboardHome() {
  const [tenantId, setTenantId] = useState<string>('');
  const valuationDate = new Date().toISOString().slice(0, 10);

  useEffect(() => {
    const saved = localStorage.getItem('selectedTenant') || 'tenant-1';
    setTenantId(saved);
  }, []);

  const compliance = useComplianceSummary(tenantId, valuationDate);
  const risk = useRiskSummary(tenantId, valuationDate);
  const sparklines = useSparklines(tenantId);
  const etl = useETLHealth(tenantId);
  const alerts = useAlerts(tenantId, valuationDate);

  return (
    <ConsoleLayout>
      <ConsoleBreadcrumbs items={[{ label: 'Dashboard', href: '/console/dashboard' }]} />

      <Grid container spacing={3}>
        {/* Compliance KPIs */}
        <Grid item xs={12} md={6}>
          {compliance.isLoading && <Typography>Loading compliance data…</Typography>}
          {compliance.data && (
            <Card>
              <CardContent>
                <Typography variant="h6" gutterBottom>
                  Compliance
                </Typography>
                <Stack spacing={2}>
                  <Stack direction="row" justifyContent="space-between">
                    <Typography color="textSecondary">Rules Evaluated</Typography>
                    <Typography fontWeight="bold">{compliance.data.total_rules}</Typography>
                  </Stack>
                  <Stack direction="row" justifyContent="space-between">
                    <Typography color="textSecondary">Pass Rate</Typography>
                    <Typography
                      fontWeight="bold"
                      sx={{
                        color: compliance.data.pass_rate > 0.9 ? '#2ECC71' : '#F39C12',
                      }}
                    >
                      {(compliance.data.pass_rate * 100).toFixed(1)}%
                    </Typography>
                  </Stack>
                  <Stack direction="row" justifyContent="space-between">
                    <Typography color="textSecondary">Hard Breaches</Typography>
                    <StatusBadge status={compliance.data.hard_breaches > 0 ? 'FAIL' : 'PASS'} />
                  </Stack>
                  <Typography variant="caption" color="textSecondary">
                    {compliance.data.hard_breaches} hard, {compliance.data.soft_breaches} soft
                  </Typography>
                </Stack>
              </CardContent>
            </Card>
          )}
        </Grid>

        {/* Risk KPIs */}
        <Grid item xs={12} md={6}>
          {risk.isLoading && <Typography>Loading risk data…</Typography>}
          {risk.data && (
            <Card>
              <CardContent>
                <Typography variant="h6" gutterBottom>
                  Risk Metrics
                </Typography>
                <Stack spacing={2}>
                  <Stack direction="row" justifyContent="space-between">
                    <Typography color="textSecondary">Avg Volatility</Typography>
                    <Typography fontWeight="bold">{risk.data.avg_volatility.toFixed(4)}</Typography>
                  </Stack>
                  <Stack direction="row" justifyContent="space-between">
                    <Typography color="textSecondary">VaR 95%</Typography>
                    <Typography fontWeight="bold">{risk.data.avg_var_95.toFixed(4)}</Typography>
                  </Stack>
                  <Stack direction="row" justifyContent="space-between">
                    <Typography color="textSecondary">VaR 99%</Typography>
                    <Typography fontWeight="bold">{risk.data.avg_var_99.toFixed(4)}</Typography>
                  </Stack>
                  {risk.data.worst_scenario && (
                    <Box
                      sx={{
                        p: 1.5,
                        backgroundColor: '#ffe0e0',
                        borderRadius: 1,
                        border: '1px solid #ffcccc',
                      }}
                    >
                      <Typography variant="caption" sx={{ display: 'block', mb: 0.5 }}>
                        Worst Scenario
                      </Typography>
                      <Typography variant="body2" fontWeight="bold">
                        {risk.data.worst_scenario.name}
                      </Typography>
                      <Typography
                        variant="body2"
                        sx={{ color: risk.data.worst_scenario.pnl > 0 ? '#2ECC71' : '#E74C3C' }}
                      >
                        P&L: {risk.data.worst_scenario.pnl.toFixed(0)}
                      </Typography>
                    </Box>
                  )}
                </Stack>
              </CardContent>
            </Card>
          )}
        </Grid>

        {/* Sparklines */}
        <Grid item xs={12}>
          {sparklines.isLoading && <Typography>Loading sparklines…</Typography>}
          {sparklines.data && (
            <Grid container spacing={2}>
              <Grid item xs={12} md={3}>
                <SparklineCard
                  title="Pass Rate (7d)"
                  data={sparklines.data.pass_rate}
                  metricKey="value"
                  color="#2ECC71"
                />
              </Grid>
              <Grid item xs={12} md={3}>
                <SparklineCard
                  title="Hard Breaches (7d)"
                  data={sparklines.data.hard_breaches}
                  metricKey="value"
                  color="#E74C3C"
                />
              </Grid>
              <Grid item xs={12} md={3}>
                <SparklineCard
                  title="Volatility (7d)"
                  data={sparklines.data.volatility}
                  metricKey="value"
                  color="#3498DB"
                />
              </Grid>
              <Grid item xs={12} md={3}>
                <SparklineCard
                  title="ETL Duration (7d)"
                  data={sparklines.data.etl_duration}
                  metricKey="value"
                  color="#9B59B6"
                />
              </Grid>
            </Grid>
          )}
        </Grid>

        {/* ETL Health */}
        <Grid item xs={12} md={6}>
          {etl.isLoading && <Typography>Loading ETL health…</Typography>}
          {etl.data && (
            <Card>
              <CardContent>
                <Typography variant="h6" gutterBottom>
                  ETL Health
                </Typography>
                <Stack spacing={2}>
                  <Stack direction="row" alignItems="center" spacing={1}>
                    <Typography color="textSecondary">Last Run Status</Typography>
                    <StatusBadge status={etl.data.last_run.status} />
                  </Stack>
                  <Stack direction="row" justifyContent="space-between">
                    <Typography color="textSecondary">Duration</Typography>
                    <Typography fontWeight="bold">
                      {(etl.data.last_run.duration_ms / 1000).toFixed(1)}s
                    </Typography>
                  </Stack>
                  <Stack direction="row" justifyContent="space-between">
                    <Typography color="textSecondary">Rules Evaluated</Typography>
                    <Typography fontWeight="bold">
                      {etl.data.last_run.rules_evaluated}
                    </Typography>
                  </Stack>
                  <Stack direction="row" justifyContent="space-between">
                    <Typography color="textSecondary">Success Rate</Typography>
                    <Typography fontWeight="bold">
                      {(etl.data.success_rate * 100).toFixed(1)}%
                    </Typography>
                  </Stack>
                </Stack>
              </CardContent>
            </Card>
          )}
        </Grid>

        {/* Alerts */}
        <Grid item xs={12} md={6}>
          {alerts.isLoading && <Typography>Loading alerts…</Typography>}
          {alerts.data && (
            <Card>
              <CardContent>
                <Typography variant="h6" gutterBottom>
                  Alerts & Breaches
                </Typography>
                <Stack spacing={1} sx={{ maxHeight: 300, overflow: 'auto' }}>
                  {alerts.data.total_alerts === 0 ? (
                    <Alert severity="success">No active alerts</Alert>
                  ) : (
                    <>
                      {alerts.data.hard_breaches.map((b, i) => (
                        <Alert key={`hard-${i}`} severity="error" sx={{ mb: 1 }}>
                          <Typography variant="caption">
                            <strong>{b.rule_code}</strong> (hard breach in {b.portfolio_id})
                          </Typography>
                        </Alert>
                      ))}
                      {alerts.data.soft_breaches.map((b, i) => (
                        <Alert key={`soft-${i}`} severity="warning" sx={{ mb: 1 }}>
                          <Typography variant="caption">
                            <strong>{b.rule_code}</strong> (soft breach in {b.portfolio_id})
                          </Typography>
                        </Alert>
                      ))}
                      {alerts.data.scenario_losses.map((s, i) => (
                        <Alert key={`scenario-${i}`} severity="info" sx={{ mb: 1 }}>
                          <Typography variant="caption">
                            Scenario <strong>{s.name}</strong>: {s.pnl.toFixed(0)} P&L loss
                          </Typography>
                        </Alert>
                      ))}
                      {alerts.data.etl_failures.map((e, i) => (
                        <Alert key={`etl-${i}`} severity="error" sx={{ mb: 1 }}>
                          <Typography variant="caption">
                            ETL run {e.etl_run_id} failed
                          </Typography>
                        </Alert>
                      ))}
                    </>
                  )}
                </Stack>
              </CardContent>
            </Card>
          )}
        </Grid>
      </Grid>
    </ConsoleLayout>
  );
}
