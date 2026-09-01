import { useEffect, useState } from "react";
import {
  Box,
  Typography,
  TextField,
  Button,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  CircularProgress,
} from "@mui/material";

interface JITGrantAuditEvent {
  id: string;
  grantId: string;
  userId: string;
  eventType: string;
  reason: string;
  occurredAt: string;
}

export function JITAuditDashboard() {
  const theme = useTheme();
  const [events, setEvents] = useState<JITGrantAuditEvent[]>([]);
  const [userId, setUserId] = useState("");
  const [bundleId, setBundleId] = useState("");
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    setLoading(true);
    let url = "/api/jit-audit";
    const params: string[] = [];
    if (userId) params.push(`user_id=${encodeURIComponent(userId)}`);
    if (bundleId) params.push(`bundle_id=${encodeURIComponent(bundleId)}`);
    if (params.length) url += "?" + params.join("&");
    fetch(url)
      .then((r) => r.json())
      .then(setEvents)
      .finally(() => setLoading(false));
  }, [userId, bundleId]);

  const exportCSV = () => {
    const header = "ID,Grant ID,User ID,Event Type,Reason,Occurred At\n";
    const rows = events.map(e =>
      [e.id, e.grantId, e.userId, e.eventType, e.reason, e.occurredAt].map(x => `"${x}"`).join(",")
    ).join("\n");
    const blob = new Blob([header + rows], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "jit_audit_log.csv";
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <Box sx={{ p: 3 }}>
      <Typography variant="h5" sx={{ mb: 2, fontWeight: 600 }}>
        JIT Audit & Compliance Dashboard
      </Typography>
      <Box sx={{ display: 'flex', gap: 2, mb: 2 }}>
        <TextField
          label="Filter by User ID"
          value={userId}
          onChange={e => setUserId(e.target.value)}
          size="small"
          sx={{ minWidth: 200 }}
        />
        <TextField
          label="Filter by Bundle ID"
          value={bundleId}
          onChange={e => setBundleId(e.target.value)}
          size="small"
          sx={{ minWidth: 200 }}
        />
        <Button
          variant="contained"
          onClick={exportCSV}
          disabled={!events.length}
        >
          Export CSV
        </Button>
      </Box>
      {loading ? (
        <Box sx={{ display: 'flex', justifyContent: 'center', py: 4 }}>
          <CircularProgress />
        </Box>
      ) : !events.length ? (
        <Typography color="text.secondary">No audit events found.</Typography>
      ) : (
        <TableContainer component={Paper} sx={{ borderRadius: 1, boxShadow: 1 }}>
          <Table>
            <TableHead>
              <TableRow sx={{ backgroundColor: theme.palette.grey[100] }}>
                <TableCell sx={{ fontWeight: 600 }}>ID</TableCell>
                <TableCell sx={{ fontWeight: 600 }}>Grant ID</TableCell>
                <TableCell sx={{ fontWeight: 600 }}>User ID</TableCell>
                <TableCell sx={{ fontWeight: 600 }}>Event Type</TableCell>
                <TableCell sx={{ fontWeight: 600 }}>Reason</TableCell>
                <TableCell sx={{ fontWeight: 600 }}>Occurred At</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {events.map(e => (
                <TableRow key={e.id} hover>
                  <TableCell>{e.id}</TableCell>
                  <TableCell>{e.grantId}</TableCell>
                  <TableCell>{e.userId}</TableCell>
                  <TableCell>{e.eventType}</TableCell>
                  <TableCell>{e.reason}</TableCell>
                  <TableCell>{e.occurredAt}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}
    </Box>
  );
}
