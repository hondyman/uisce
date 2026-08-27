import React, { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
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
import Button from "@mui/material/Button";
import Chip from "@mui/material/Chip";
import { observabilityApi, WasmModuleVersion } from "../../api/observabilityApi";
import { format } from "date-fns";

export function WasmVersionRegistryPage() {
  const theme = useTheme();
  const [moduleName, setModuleName] = useState("risk-compliance-engine");
  const queryClient = useQueryClient();

  const { data, isLoading, error } = useQuery({
    queryKey: ["wasm-versions", moduleName],
    queryFn: () => observabilityApi.listWasmVersions(moduleName || undefined)
  });

  const activateMutation = useMutation({
    mutationFn: (id: string) => observabilityApi.activateWasmVersion(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["wasm-versions"] });
    }
  });

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return "0 Bytes";
    const k = 1024;
    const sizes = ["Bytes", "KB", "MB", "GB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
  };

  return (
    <Box sx={{ p: 3 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Box>
          <Typography variant="h5" sx={{ fontWeight: 700, color: 'grey.900' }}>
            WASM Registry
          </Typography>
          <Typography variant="body2" sx={{ color: 'grey.500', mt: 0.5 }}>
            Manage and activate WASM execution artifacts for the SemLayer compute engine.
          </Typography>
        </Box>
      </Box>

      <Paper sx={{ mb: 3, p: 2, display: 'flex', gap: 2, alignItems: 'flex-start' }}>
        <TextField
          label="Module Namespace"
          placeholder="e.g. risk-compliance-engine"
          value={moduleName}
          onChange={(e) => setModuleName(e.target.value)}
          size="small"
          sx={{ width: 256 }}
        />
      </Paper>

      <Paper sx={{ overflow: 'hidden' }}>
        {isLoading ? (
          <Box sx={{ p: 4, textAlign: 'center', color: 'grey.500' }}>Loading registry data...</Box>
        ) : error ? (
          <Box sx={{ p: 4, textAlign: 'center', color: 'error.main' }}>Error loading WASM versions.</Box>
        ) : (
          <TableContainer>
            <Table>
              <TableHead>
                <TableRow sx={{ bgcolor: 'grey.50' }}>
                  <TableCell sx={{ fontWeight: 600, color: 'grey.500', fontSize: '0.75rem' }}>Version ID</TableCell>
                  <TableCell sx={{ fontWeight: 600, color: 'grey.500', fontSize: '0.75rem' }}>Tag</TableCell>
                  <TableCell sx={{ fontWeight: 600, color: 'grey.500', fontSize: '0.75rem' }}>Hash (SHA-256)</TableCell>
                  <TableCell sx={{ fontWeight: 600, color: 'grey.500', fontSize: '0.75rem' }}>Size</TableCell>
                  <TableCell sx={{ fontWeight: 600, color: 'grey.500', fontSize: '0.75rem' }}>Status</TableCell>
                  <TableCell sx={{ fontWeight: 600, color: 'grey.500', fontSize: '0.75rem' }}>Uploaded By</TableCell>
                  <TableCell align="right" sx={{ fontWeight: 600, color: 'grey.500', fontSize: '0.75rem' }}>Actions</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {data?.versions?.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={7} sx={{ p: 4, textAlign: 'center', color: 'grey.500', fontSize: '0.875rem' }}>
                      No WASM bundles found for this module.
                    </TableCell>
                  </TableRow>
                ) : (
                  data?.versions?.map((v: WasmModuleVersion) => (
                    <TableRow key={v.id} hover sx={{ '&:hover': { bgcolor: 'grey.50' } }}>
                      <TableCell sx={{ fontWeight: 500, color: 'grey.900', fontSize: '0.875rem', fontFamily: 'monospace' }}>
                        {v.id.split('-')[0]}
                      </TableCell>
                      <TableCell>
                        <Chip label={v.version_tag} size="small" sx={{ fontFamily: 'monospace', bgcolor: 'grey.100', color: 'grey.800' }} />
                      </TableCell>
                      <TableCell sx={{ fontFamily: 'monospace', fontSize: '0.75rem', color: 'grey.500' }}>
                        {v.wasm_hash.substring(0, 16)}...
                      </TableCell>
                      <TableCell sx={{ color: 'grey.500', fontSize: '0.875rem' }}>
                        {formatBytes(v.size_bytes)}
                      </TableCell>
                      <TableCell>
                        {v.is_active ? (
                          <Chip label="ACTIVE" size="small" sx={{ fontWeight: 700, bgcolor: 'success.light', color: 'success.dark', border: '1px solid', borderColor: 'success.200' }} />
                        ) : (
                          <Chip label="INACTIVE" size="small" sx={{ bgcolor: 'grey.100', color: 'grey.500' }} />
                        )}
                      </TableCell>
                      <TableCell sx={{ color: 'grey.500', fontSize: '0.875rem' }}>
                        <Box>{v.uploaded_by}</Box>
                        <Box sx={{ fontSize: '0.75rem', color: 'grey.400' }}>{format(new Date(v.created_at), "MMM d, yyyy HH:mm")}</Box>
                      </TableCell>
                      <TableCell align="right" sx={{ fontWeight: 500, fontSize: '0.875rem' }}>
                        {!v.is_active && (
                          <Button
                            onClick={() => activateMutation.mutate(v.id)}
                            disabled={activateMutation.isPending}
                            variant="outlined"
                            size="small"
                            sx={{ 
                              color: 'primary.main', 
                              borderColor: 'primary.main',
                              fontSize: '0.75rem',
                              fontWeight: 600,
                              '&:hover': { color: 'primary.dark', borderColor: 'primary.dark' },
                              '&:disabled': { opacity: 0.5 }
                            }}
                          >
                            {activateMutation.isPending ? "Activating..." : "Activate Now"}
                          </Button>
                        )}
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
