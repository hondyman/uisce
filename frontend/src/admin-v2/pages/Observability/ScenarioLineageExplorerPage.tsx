import React, { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useTheme } from "@mui/material/styles";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import Paper from "@mui/material/Paper";
import TextField from "@mui/material/TextField";
import Table from "@mui/material/Table";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell";
import TableContainer from "@mui/material/TableContainer";
import TableHead from "@mui/material/TableHead";
import TableRow from "@mui/material/TableRow";
import Chip from "@mui/material/Chip";
import CircularProgress from "@mui/material/CircularProgress";
import { observabilityApi, ScenarioLineage } from "../../api/observabilityApi";
import { format } from "date-fns";

export function ScenarioLineageExplorerPage() {
  const theme = useTheme();
  const [scenarioId, setScenarioId] = useState("");
  const [tenantId, setTenantId] = useState("");

  const { data, isLoading, error } = useQuery({
    queryKey: ["scenario-lineage", scenarioId, tenantId],
    queryFn: () => observabilityApi.getScenarioLineage(scenarioId, { tenant_id: tenantId || undefined }),
    enabled: !!scenarioId
  });

  const formatCurrency = (val: number) => {
    return new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: 0 }).format(val);
  };

  const getPnlBadge = (pnlAmount: number, pnlPercent: number) => {
    const isLoss = pnlAmount < 0;
    return (
      <Box
        sx={{
          display: "flex",
          alignItems: "center",
          gap: 1,
          px: 1,
          py: 0.5,
          borderRadius: 1,
          border: 1,
          bgcolor: isLoss ? "error.light" : "success.light",
          borderColor: isLoss ? "error.light" : "success.light",
          color: isLoss ? "error.dark" : "success.dark",
        }}
      >
        <Box component="span" sx={{ fontFamily: "monospace", fontSize: "0.875rem", fontWeight: 600 }}>
          {formatCurrency(pnlAmount)}
        </Box>
        <Box component="span" sx={{ fontSize: "0.75rem" }}>
          ({(pnlPercent * 100).toFixed(2)}%)
        </Box>
      </Box>
    );
  };

  return (
    <Box sx={{ p: 3 }}>
      <Box sx={{ display: "flex", justifyContent: "space-between", alignItems: "center", mb: 3 }}>
        <Box>
          <Typography variant="h5" sx={{ fontWeight: 700, mb: 0.5 }}>
            Scenario Lineage Trace
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Explore historical stress test results and VaR projections mapped by semantic factors.
          </Typography>
        </Box>
      </Box>

      <Paper sx={{ mb: 3, p: 2, display: "flex", gap: 3, flexWrap: "wrap" }}>
        <TextField
          label="Scenario ID (Required)"
          variant="outlined"
          size="small"
          placeholder="UUID of the scenario..."
          value={scenarioId}
          onChange={(e) => setScenarioId(e.target.value)}
          sx={{ width: 300 }}
          InputProps={{ sx: { fontFamily: "monospace" } }}
        />
        <TextField
          label="Tenant ID"
          variant="outlined"
          size="small"
          placeholder="Optional"
          value={tenantId}
          onChange={(e) => setTenantId(e.target.value)}
          sx={{ width: 200 }}
          InputProps={{ sx: { fontFamily: "monospace" } }}
        />
      </Paper>

      <Paper sx={{ overflow: "hidden" }}>
        {!scenarioId ? (
          <Box sx={{ p: 4, textAlign: "center", color: "text.secondary" }}>
            Please enter a Scenario ID to view execution lineage.
          </Box>
        ) : isLoading ? (
          <Box sx={{ p: 4, textAlign: "center", color: "text.secondary" }}>
            <CircularProgress size={24} sx={{ mr: 1 }} />
            Fetching lineage traces...
          </Box>
        ) : error ? (
          <Box sx={{ p: 4, textAlign: "center", color: "error.main" }}>
            Error loading scenario lineage.
          </Box>
        ) : (
          <TableContainer>
            <Table>
              <TableHead>
                <TableRow sx={{ bgcolor: "action.hover" }}>
                  <TableCell sx={{ fontWeight: 600 }}>Date</TableCell>
                  <TableCell sx={{ fontWeight: 600 }}>Portfolio</TableCell>
                  <TableCell align="right" sx={{ fontWeight: 600 }}>Base Value</TableCell>
                  <TableCell align="right" sx={{ fontWeight: 600 }}>Stressed Value</TableCell>
                  <TableCell sx={{ fontWeight: 600 }}>PnL Impact</TableCell>
                  <TableCell sx={{ fontWeight: 600 }}>WASM Perf</TableCell>
                  <TableCell sx={{ fontWeight: 600 }}>Semantic Trace</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {data?.lineage?.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={7} align="center" sx={{ py: 4, color: "text.secondary" }}>
                      No traces found for this Scenario ID.
                    </TableCell>
                  </TableRow>
                ) : (
                  data?.lineage?.map((trace: ScenarioLineage) => (
                    <TableRow key={trace.id} hover>
                      <TableCell sx={{ fontWeight: 500 }}>
                        {format(new Date(trace.valuation_date), "MMM d, yyyy")}
                      </TableCell>
                      <TableCell>
                        <Box sx={{ fontFamily: "monospace", fontSize: "0.75rem" }} title="Portfolio ID">
                          {trace.portfolio_id.split('-')[0]}
                        </Box>
                      </TableCell>
                      <TableCell align="right" sx={{ fontFamily: "monospace" }}>
                        {formatCurrency(trace.total_base_value)}
                      </TableCell>
                      <TableCell align="right" sx={{ fontFamily: "monospace" }}>
                        {formatCurrency(trace.total_stressed_value)}
                      </TableCell>
                      <TableCell>
                        {getPnlBadge(trace.pnl_amount, trace.pnl_percent)}
                      </TableCell>
                      <TableCell>
                        <Box sx={{ fontSize: "0.875rem" }}>{trace.duration_ms} ms</Box>
                        <Box sx={{ fontSize: "0.75rem", fontFamily: "monospace", color: "text.secondary" }} title="WASM Engine Version">
                          {trace.wasm_version_id.split('-')[0]}
                        </Box>
                      </TableCell>
                      <TableCell>
                        <Box sx={{ display: "flex", flexWrap: "wrap", gap: 0.5, maxWidth: 200 }}>
                          {trace.semantic_terms_used?.length > 0 ? trace.semantic_terms_used.map(term => (
                            <Chip
                              key={term}
                              label={term}
                              size="small"
                              sx={{
                                bgcolor: "primary.light",
                                color: "primary.dark",
                                fontSize: "0.75rem",
                                height: 22,
                              }}
                            />
                          )) : <Typography variant="caption" sx={{ color: "text.disabled" }}>None mapped</Typography>}
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
