import { useEffect, useState } from "react";
import { JITRequestPanel } from "./JITRequestPanel";
import { AccessExplanation } from "./AccessExplanation";
import apiClient from "./utils/apiClient";
import {
  Box,
  Typography,
  TextField,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Button,
  Paper,
  Modal,
  List,
  ListItem,
  ListItemText,
} from "@mui/material";

export interface MicroBundle {
  id: string;
  name: string;
  description: string;
  claims: any[];
  domain: string;
  version: number;
}

export function MicroBundleCatalog() {
  const [bundles, setBundles] = useState<MicroBundle[]>([]);
  const [filter, setFilter] = useState({ domain: "", permission: "" });
  const [selected, setSelected] = useState<MicroBundle | null>(null);
  const [showJIT, setShowJIT] = useState(false);

  useEffect(() => {
    apiClient("micro-bundles")
      .then((r) => r.json())
      .then(setBundles);
  }, []);

  const filtered = bundles.filter(
    (b) =>
      (!filter.domain || b.domain.includes(filter.domain)) &&
      (!filter.permission || b.claims.some((c) => c.permission?.includes(filter.permission)))
  );

  return (
    <Box sx={{ p: 2 }}>
      <Typography variant="h5" sx={{ fontWeight: 700, mb: 1 }}>
        Micro-Bundle Catalog
      </Typography>
      <Box sx={{ display: "flex", gap: 2, mb: 2 }}>
        <TextField
          size="small"
          placeholder="Domain"
          value={filter.domain}
          onChange={(e) => setFilter({ ...filter, domain: e.target.value })}
          sx={{
            "& .MuiOutlinedInput-root": {
              borderRadius: 1,
            },
          }}
        />
        <TextField
          size="small"
          placeholder="Permission"
          value={filter.permission}
          onChange={(e) => setFilter({ ...filter, permission: e.target.value })}
          sx={{
            "& .MuiOutlinedInput-root": {
              borderRadius: 1,
            },
          }}
        />
      </Box>
      <TableContainer component={Paper} sx={{ mb: 2, border: "1px solid", borderColor: "divider" }}>
        <Table>
          <TableHead>
            <TableRow sx={{ backgroundColor: "#f5f5f5" }}>
              <TableCell>Name</TableCell>
              <TableCell>Domain</TableCell>
              <TableCell>Claims</TableCell>
              <TableCell></TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {filtered.map((b) => (
              <TableRow key={b.id}>
                <TableCell>{b.name}</TableCell>
                <TableCell>{b.domain}</TableCell>
                <TableCell>{b.claims.length}</TableCell>
                <TableCell>
                  <Button
                    size="small"
                    onClick={() => setSelected(b)}
                    sx={{ color: "primary.main", textDecoration: "underline" }}
                  >
                    Details
                  </Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
      <Modal open={!!selected} onClose={() => setSelected(null)}>
        <Box
          sx={{
            position: "absolute",
            top: "50%",
            left: "50%",
            transform: "translate(-50%, -50%)",
            width: 500,
            bgcolor: "background.paper",
            p: 3,
            borderRadius: 2,
            boxShadow: 24,
          }}
        >
          <Button
            onClick={() => setSelected(null)}
            sx={{ position: "absolute", top: 8, right: 8 }}
          >
            ×
          </Button>
          <Typography variant="h6" sx={{ fontWeight: 700, mb: 1 }}>
            {selected?.name}
          </Typography>
          <Typography sx={{ mb: 2 }}>{selected?.description}</Typography>
          <Box sx={{ mb: 2 }}>
            <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
              Claims:
            </Typography>
            <List dense>
              {selected?.claims.map((c, i) => (
                <ListItem key={i} sx={{ py: 0 }}>
                  <ListItemText primary={JSON.stringify(c)} />
                </ListItem>
              ))}
            </List>
          </Box>
          <Box sx={{ mb: 2 }}>
            <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
              Version:{" "}
            </Typography>
            {selected?.version}
          </Box>
          <Box sx={{ mb: 2 }}>
            <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
              Usage Example:{" "}
            </Typography>
            <code>GET /api/micro-bundles/{selected?.id}</code>
          </Box>
          <Box sx={{ mb: 2 }}>
            <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
              Expiry Policy:{" "}
            </Typography>
            JIT add-ons expire per policy (see below)
          </Box>
          <Button
            variant="contained"
            onClick={() => {
              setShowJIT(true);
              setSelected(null);
            }}
            sx={{ mt: 1 }}
          >
            Request JIT Add-On
          </Button>
        </Box>
      </Modal>
      {showJIT && <JITRequestPanel onClose={() => setShowJIT(false)} />}
      <AccessExplanation />
    </Box>
  );
}
