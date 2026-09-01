import React, { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useTheme } from "@mui/material/styles";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import TextField from "@mui/material/TextField";
import Paper from "@mui/material/Paper";
import Table from "@mui/material/Table";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell";
import TableContainer from "@mui/material/TableContainer";
import TableHead from "@mui/material/TableHead";
import TableRow from "@mui/material/TableRow";
import FormControl from "@mui/material/FormControl";
import InputLabel from "@mui/material/InputLabel";
import Select from "@mui/material/Select";
import MenuItem from "@mui/material/MenuItem";
import Chip from "@mui/material/Chip";
import { observabilityApi, RuleLineage } from "../../api/observabilityApi";
import { format } from "date-fns";

export function RuleLineageExplorerPage() {
  const theme = useTheme();
  const [ruleId, setRuleId] = useState("");
  const [tenantId, setTenantId] = useState("");
  const [status, setStatus] = useState("");

  const { data, isLoading, error } = useQuery({
    queryKey: ["rule-lineage", ruleId, tenantId, status],
    queryFn: () => observabilityApi.getRuleLineage(ruleId, { tenant_id: tenantId || undefined, status: status || undefined }),
    enabled: !!ruleId
  });

  const getStatusBadge = (runStatus: string) => {
    switch (runStatus) {
      case "PASS":
        return <Chip label="PASS" size="small" sx={{ bgcolor: 'success.light', color: 'success.dark', fontWeight: 700, fontSize: '0.75rem' }} />;
      case "FAIL":
        return <Chip label="FAIL" size="small" sx={{ bgcolor: 'error.light', color: 'error.dark', fontWeight: 700, fontSize: '0.75rem' }} />;
      case "WARNING":
        return <Chip label="WARNING" size="small" sx={{ bgcolor: 'warning.light', color: 'warning.dark', fontWeight: 700, fontSize: '0.75rem' }} />;
      default:
        return <Chip label={runStatus} size="small" sx={{ bgcolor: 'grey.100', color: 'grey.800', fontWeight: 700, fontSize: '0.75rem' }} />;
    }
  };

  return (
    <Box sx={{ p: 3 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Box>
          <Typography variant="h5" sx={{ fontWeight: 700, color: 'grey.900' }}>
            Rule Lineage Trace
          </Typography>
          <Typography variant="body2" sx={{ color: 'grey.500', mt: 0.5 }}>
            Explore historical executions and semantic evaluation lineage for Compliance Rules.
          </Typography>
        </Box>
      </Box>

      <Paper sx={{ mb: 3, p: 2, display: 'flex', gap: 2, alignItems: 'flex-start' }}>
        <TextField
          label="Rule ID (Required)"
          placeholder="UUID of the rule..."
          value={ruleId}
          onChange={(e) => setRuleId(e.target.value)}
          size="small"
          sx={{ width: 288, '& input': { fontFamily: 'monospace' } }}
        />
        <TextField
          label="Tenant ID"
          placeholder="Optional"
          value={tenantId}
          onChange={(e) => setTenantId(e.target.value)}
          size="small"
          sx={{ width: 192, '& input': { fontFamily: 'monospace' } }}
        />
        <FormControl size="small" sx={{ minWidth: 160 }}>
          <InputLabel>Outcome</InputLabel>
          <Select
            value={status}
            label="Outcome"
            onChange={(e) => setStatus(e.target.value)}
          >
            <MenuItem value="">All</MenuItem>
            <MenuItem value="PASS">Pass</MenuItem>
            <MenuItem value="FAIL">Fail</MenuItem>
            <MenuItem value="WARNING">Warning</MenuItem>
          </Select>
        </FormControl>
      </Paper>

      <Paper sx={{ overflow: 'hidden' }}>
        {!ruleId ? (
          <Box sx={{ p: 4, textAlign: 'center', color: 'grey.500' }}>Please enter a Rule ID to view execution lineage.</Box>
        ) : isLoading ? (
          <Box sx={{ p: 4, textAlign: 'center', color: 'grey.500' }}>Fetching lineage traces...</Box>
        ) : error ? (
          <Box sx={{ p: 4, textAlign: 'center', color: 'error.main' }}>Error loading rule lineage.</Box>
        ) : (
          <TableContainer>
            <Table>
              <TableHead>
                <TableRow sx={{ bgcolor: 'grey.50' }}>
                  <TableCell sx={{ fontWeight: 600, color: 'grey.500', fontSize: '0.75rem' }}>Date</TableCell>
                  <TableCell sx={{ fontWeight: 600, color: 'grey.500', fontSize: '0.75rem' }}>Portfolio / Target</TableCell>
                  <TableCell sx={{ fontWeight: 600, color: 'grey.500', fontSize: '0.75rem' }}>Outcome</TableCell>
                  <TableCell sx={{ fontWeight: 600, color: 'grey.500', fontSize: '0.75rem' }}>Metric vs Threshold</TableCell>
                  <TableCell sx={{ fontWeight: 600, color: 'grey.500', fontSize: '0.75rem' }}>WASM Perf</TableCell>
                  <TableCell sx={{ fontWeight: 600, color: 'grey.500', fontSize: '0.75rem' }}>Semantics Hit</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {data?.lineage?.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={6} sx={{ p: 4, textAlign: 'center', color: 'grey.500', fontSize: '0.875rem' }}>
                      No traces found for this Rule ID.
                    </TableCell>
                  </TableRow>
                ) : (
                  data?.lineage?.map((trace: RuleLineage) => (
                    <TableRow key={trace.id} hover sx={{ '&:hover': { bgcolor: 'grey.50' } }}>
                      <TableCell sx={{ fontWeight: 500, color: 'grey.900', fontSize: '0.875rem' }}>
                        {format(new Date(trace.valuation_date), "MMM d, yyyy")}
                      </TableCell>
                      <TableCell sx={{ color: 'grey.500', fontSize: '0.875rem' }}>
                        <Box sx={{ fontFamily: 'monospace', fontSize: '0.75rem', mb: 0.5 }} title="Portfolio ID">{trace.portfolio_id.split('-')[0]}</Box>
                        {trace.security_id && <Box sx={{ fontFamily: 'monospace', fontSize: '0.75rem', color: 'primary.main' }} title="Security Focus">{trace.security_id.split('-')[0]}</Box>}
                      </TableCell>
                      <TableCell>
                        {getStatusBadge(trace.status)}
                      </TableCell>
                      <TableCell>
                         {(trace.metric_value !== undefined && trace.threshold_value !== undefined) ? (
                           <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                             <Box component="span" sx={{ fontFamily: 'monospace', fontSize: '0.875rem', color: trace.status === 'FAIL' ? 'error.main' : 'grey.900', fontWeight: trace.status === 'FAIL' ? 700 : 400 }}>{trace.metric_value}</Box>
                             <Box component="span" sx={{ color: 'grey.400', fontSize: '0.75rem' }}>/</Box>
                             <Box component="span" sx={{ fontFamily: 'monospace', fontSize: '0.875rem', color: 'grey.500' }}>{trace.threshold_value}</Box>
                           </Box>
                         ) : <Box sx={{ color: 'grey.400' }}>-</Box>}
                      </TableCell>
                      <TableCell>
                        <Box sx={{ fontSize: '0.875rem', color: 'grey.900' }}>{trace.duration_ms} ms</Box>
                        <Box sx={{ fontFamily: 'monospace', fontSize: '0.75rem', color: 'grey.400' }} title="WASM Engine Version">{trace.wasm_version_id.split('-')[0]}</Box>
                      </TableCell>
                      <TableCell sx={{ color: 'grey.500', fontSize: '0.875rem' }}>
                         <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                            {trace.semantic_terms_used?.length > 0 ? trace.semantic_terms_used.map(term => (
                              <Chip key={term} label={term} size="small" sx={{ bgcolor: 'primary.light', color: 'primary.dark', border: '1px solid', borderColor: 'primary.200', fontSize: '0.75rem' }} />
                            )) : <Box sx={{ color: 'grey.400' }}>None mapped</Box>}
                         </Box>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </TableContainer>
        )}
      </Paper>
    </Box>
  );
}
