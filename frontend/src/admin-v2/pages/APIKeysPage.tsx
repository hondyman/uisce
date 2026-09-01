import React, { useState } from "react";
import { useTheme } from "@mui/material/styles";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import Button from "@mui/material/Button";
import Paper from "@mui/material/Paper";
import Table from "@mui/material/Table";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell";
import TableContainer from "@mui/material/TableContainer";
import TableHead from "@mui/material/TableHead";
import TableRow from "@mui/material/TableRow";
import Chip from "@mui/material/Chip";
import CircularProgress from "@mui/material/CircularProgress";
import { Card } from "../components/Card";
import { Spinner } from "../components/Feedback";
import { CreateAPIKeyModal } from "../components/CreateAPIKeyModal";
import { useAPIKeys } from "../hooks/useAPIKeys";

export function APIKeysPage() {
  const theme = useTheme();
  const [showCreateModal, setShowCreateModal] = useState(false);
  const apiKeysQuery = useAPIKeys();

  const apiKeys = apiKeysQuery.data?.data || [];

  const columns = ["Name", "Key Preview", "Created", "Last Used", "Status"];

  return (
    <Box sx={{ p: 3 }}>
      <Box
        sx={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          mb: 3,
        }}
      >
        <Box>
          <Typography variant="h4" component="h1" sx={{ fontWeight: 600, mb: 0.5 }}>
            API Keys
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Manage authentication tokens
          </Typography>
        </Box>
        <Button
          variant="contained"
          onClick={() => setShowCreateModal(true)}
          sx={{ textTransform: "none" }}
        >
          + Create Key
        </Button>
      </Box>

      <Card>
        {apiKeysQuery.isLoading ? (
          <Box sx={{ display: "flex", justifyContent: "center", p: 4 }}>
            <CircularProgress size="medium" />
          </Box>
        ) : (
          <TableContainer>
            <Table>
              <TableHead>
                <TableRow>
                  {columns.map((col) => (
                    <TableCell key={col} sx={{ fontWeight: 600 }}>
                      {col}
                    </TableCell>
                  ))}
                </TableRow>
              </TableHead>
              <TableBody>
                {apiKeys.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={columns.length} align="center" sx={{ py: 4, color: "text.secondary" }}>
                      No API keys yet
                    </TableCell>
                  </TableRow>
                ) : (
                  apiKeys.map((key) => (
                    <TableRow key={key.id} hover>
                      <TableCell>{key.name}</TableCell>
                      <TableCell>
                        <Box
                          component="code"
                          sx={{
                            fontFamily: "monospace",
                            fontSize: "0.875rem",
                            color: "text.secondary",
                          }}
                        >
                          {key.key.substring(0, 20)}...
                        </Box>
                      </TableCell>
                      <TableCell>{new Date(key.createdAt).toLocaleDateString()}</TableCell>
                      <TableCell>
                        {key.lastUsedAt ? new Date(key.lastUsedAt).toLocaleDateString() : "Never"}
                      </TableCell>
                      <TableCell>
                        <Chip
                          label={key.revoked ? "Revoked" : "Active"}
                          color={key.revoked ? "error" : "success"}
                          size="small"
                          sx={{ textTransform: "capitalize" }}
                        />
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </TableContainer>
        )}
      </Card>

      <CreateAPIKeyModal
        open={showCreateModal}
        onClose={() => setShowCreateModal(false)}
        onSuccess={() => {}}
      />
    </Box>
  );
}
