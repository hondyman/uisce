import React, { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useTheme } from "@mui/material/styles";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import Paper from "@mui/material/Paper";
import TextField from "@mui/material/TextField";
import Select from "@mui/material/Select";
import MenuItem from "@mui/material/MenuItem";
import FormControl from "@mui/material/FormControl";
import InputLabel from "@mui/material/InputLabel";
import Table from "@mui/material/Table";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell";
import TableContainer from "@mui/material/TableContainer";
import TableHead from "@mui/material/TableHead";
import TableRow from "@mui/material/TableRow";
import Chip from "@mui/material/Chip";
import CircularProgress from "@mui/material/CircularProgress";
import { observabilityApi, ETLRun } from "../../api/observabilityApi";
import { format } from "date-fns";

export function ETLRunDashboardPage() {
  const theme = useTheme();
  const [tenantId, setTenantId] = useState("");
  const [status, setStatus] = useState("");

  const { data, isLoading, error } = useQuery({
    queryKey: ["etl-runs", tenantId, status],
    queryFn: () => observabilityApi.listETLRuns({ tenant_id: tenantId || undefined, status: status || undefined })
  });

  const getStatusBadge = (runStatus: string) => {
    const config: Record<string, { color: "success" | "error" | "info" | "default"; label: string }> = {
      SUCCESS: { color: "success", label: "Success" },
      FAILED: { color: "error", label: "Failed" },
      STARTED: { color: "info", label: "Running" },
    };
    const { color, label } = config[runStatus] || { color: "default", label: runStatus };
    return <Chip label={label} color={color} size="small" sx={{ fontWeight: 600 }} />;
  };

  return (
    <Box sx={{ p: 3 }}>
      <Box sx={{ display: "flex", justifyContent: "space-between", alignItems: "center", mb: 3 }}>
        <Box>
          <Typography variant="h5" sx={{ fontWeight: 700, mb: 0.5 }}>
            ETL Telemetry Dashboard
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Monitor the semantic execution fabric runs across all tenants.
          </Typography>
        </Box>
      </Box>

      <Paper sx={{ mb: 3, p: 2, display: "flex", gap: 3, flexWrap: "wrap" }}>
        <TextField
          label="Tenant ID"
          variant="outlined"
          size="small"
          placeholder="Filter by Tenant..."
          value={tenantId}
          onChange={(e) => setTenantId(e.target.value)}
          sx={{ minWidth: 250 }}
        />
        <FormControl size="small" sx={{ minWidth: 150 }}>
          <InputLabel>Status</InputLabel>
          <Select
            value={status}
            label="Status"
            onChange={(e) => setStatus(e.target.value)}
          >
            <MenuItem value="">All Statuses</MenuItem>
            <MenuItem value="SUCCESS">Success</MenuItem>
            <MenuItem value="FAILED">Failed</MenuItem>
            <MenuItem value="STARTED">Started</MenuItem>
          </Select>
        </FormControl>
      </Paper>

      <Paper sx={{ overflow: "hidden" }}>
        {isLoading ? (
          <Box sx={{ p: 4, textAlign: "center", color: "text.secondary" }}>
            <CircularProgress size={24} sx={{ mr: 1 }} />
            Loading telemetry data...
          </Box>
        ) : error ? (
          <Box sx={{ p: 4, textAlign: "center", color: "error.main" }}>
            Error loading ETL runs.
          </Box>
        ) : (
          <TableContainer>
            <Table>
              <TableHead>
                <TableRow sx={{ bgcolor: "action.hover" }}>
                  <TableCell sx={{ fontWeight: 600 }}>Run ID</TableCell>
                  <TableCell sx={{ fontWeight: 600 }}>Valuation Date</TableCell>
                  <TableCell sx={{ fontWeight: 600 }}>Status</TableCell>
                  <TableCell sx={{ fontWeight: 600 }}>Duration</TableCell>
                  <TableCell sx={{ fontWeight: 600 }}>Evaluations</TableCell>
                  <TableCell sx={{ fontWeight: 600 }}>Version</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {data?.runs?.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={6} align="center" sx={{ py: 4, color: "text.secondary" }}>
                      No ETL telemetry found matching criteria.
                    </TableCell>
                  </TableRow>
                ) : (
                  data?.runs?.map((run: ETLRun) => (
                    <TableRow key={run.id} hover sx={{ cursor: "pointer" }}>
                      <TableCell>
                        <Box sx={{ fontWeight: 500, color: "primary.main", fontFamily: "monospace" }}>
                          {run.id.split('-')[0]}
                        </Box>
                        <Box sx={{ fontSize: "0.75rem", color: "text.secondary", mt: 0.5 }}>
                          Tenant: {run.tenant_id.split('-')[0]}
                        </Box>
                      </TableCell>
                      <TableCell>
                        {format(new Date(run.valuation_date), "MMM d, yyyy")}
                      </TableCell>
                      <TableCell>
                        {getStatusBadge(run.status)}
                      </TableCell>
                      <TableCell sx={{ color: "text.secondary" }}>
                        {run.duration_ms ? `${run.duration_ms} ms` : "-"}
                      </TableCell>
                      <TableCell>
                        <Box sx={{ fontSize: "0.875rem" }}>{run.rules_evaluated} rules</Box>
                        <Box sx={{ fontSize: "0.75rem", color: "text.secondary" }}>{run.scenarios_evaluated} scen.</Box>
                      </TableCell>
                      <TableCell sx={{ fontFamily: "monospace", color: "text.secondary" }}>
                        {run.wasm_orchestrator_version ? run.wasm_orchestrator_version.split('-')[0] : "-"}
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
