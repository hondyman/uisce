import React, { useState } from "react";
import { useTheme } from "@mui/material/styles";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import Chip from "@mui/material/Chip";
import Button from "@mui/material/Button";
import { Card } from "./Card";
import { Table } from "./Table";
import { Spinner, ErrorBanner } from "./Feedback";
import { useErrorFingerprints, useErrorFingerprintHistory } from "../hooks/useOps";
import type { ErrorFingerprint } from "@/admin-v2/types";

export function ErrorFingerprints() {
  const theme = useTheme();
  const isDark = theme.palette.mode === 'dark';
  const fingerprintsQuery = useErrorFingerprints(50);
  const [selectedFingerprintId, setSelectedFingerprintId] = useState<string | null>(null);
  const historyQuery = useErrorFingerprintHistory(selectedFingerprintId, 50);

  const fingerprints = fingerprintsQuery.data?.data || [];
  const history = historyQuery.data?.data || [];

  const columns = ["Path", "Status", "Sample", "Count", "Last Seen"];
  const rows = fingerprints.map((fp) => [
    <Box component="code" sx={{ fontFamily: 'monospace', fontSize: '0.875rem' }}>{fp.path}</Box>,
    <Chip
      label={fp.status_code}
      size="small"
      color={Math.floor(fp.status_code / 100) === 2 ? 'success' : Math.floor(fp.status_code / 100) === 4 ? 'warning' : 'error'}
      variant="outlined"
    />,
    <Typography variant="body2" sx={{ maxWidth: 200, overflow: 'hidden', textOverflow: 'ellipsis' }}>{fp.sample_message}</Typography>,
    <Typography sx={{ fontWeight: 600 }}>{fp.count}</Typography>,
    new Date(fp.last_seen).toLocaleString(),
  ]);

  const eventColumns = ["Tenant", "Endpoint", "Message", "Time"];
  const eventRows = history.map((event) => [
    event.tenant_id || "N/A",
    event.endpoint,
    <Typography variant="body2" sx={{ color: 'error.main' }}>{event.message}</Typography>,
    new Date(event.occurred_at).toLocaleString(),
  ]);

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
      <Card title="Error Fingerprints" subtitle="Grouped error patterns">
        {fingerprintsQuery.isLoading ? (
          <Spinner size="sm" />
        ) : (
          <Table
            columns={columns}
            rows={rows}
            loading={fingerprintsQuery.isLoading}
            empty="No errors recorded"
          />
        )}
      </Card>

      {selectedFingerprintId && (
        <Card title="Recent Occurrences">
          {historyQuery.isError && (
            <ErrorBanner message="Failed to load error history" />
          )}
          {historyQuery.isLoading ? (
            <Spinner size="sm" />
          ) : (
            <>
              <Table
                columns={eventColumns}
                rows={eventRows}
                loading={historyQuery.isLoading}
                empty="No recent occurrences"
              />
              <Button
                variant="outlined"
                onClick={() => setSelectedFingerprintId(null)}
                sx={{ mt: 2 }}
              >
                Clear Selection
              </Button>
            </>
          )}
        </Card>
      )}
    </Box>
  );
}
