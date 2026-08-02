import { useState, useEffect, useMemo } from 'react';
import {
  Box,
  Button,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Typography,
  IconButton,
  Tooltip,
  TextField,
  InputAdornment,
  Chip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Menu,
  MenuItem,
  ListItemIcon,
  ListItemText,
} from '@mui/material';
import {
  Add as AddIcon,
  Edit as EditIcon,
  Delete as DeleteIcon,
  Search as SearchIcon,
  Download as DownloadIcon,
} from '@mui/icons-material';
import { useIPWhitelistAPI } from '../../fabric/hooks/useIPWhitelist';
import { IPWhitelistEntry, Tenant } from '../../fabric/types/ipWhitelist';
import IPAddEditDialog from '../../fabric/components/IPAddEditDialog';
import ModalHeader from '../../../components/ModalHeader';

interface IPWhitelistTabProps {
  tenantId: string;
}

export default function IPWhitelistTab({ tenantId }: IPWhitelistTabProps): JSX.Element {
  const [search, setSearch] = useState('');
  const [ipAddresses, setIPAddresses] = useState<IPWhitelistEntry[]>([]);
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editEntry, setEditEntry] = useState<IPWhitelistEntry | null>(null);
  const [exportAnchor, setExportAnchor] = useState<null | HTMLElement>(null);

  const api = useIPWhitelistAPI();

  const fetchData = async () => {
    setLoading(true);
    try {
      const [ips, tenantList] = await Promise.all([
        api.fetchTenantIPWhitelist(tenantId),
        api.fetchTenants(),
      ]);
      setIPAddresses(ips ?? []);
      setTenants(tenantList ?? []);
    } catch (err) {
      console.error('Failed to fetch IP whitelist:', err);
      setIPAddresses([]);
      setTenants([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (tenantId) {
      fetchData();
    }
  }, [tenantId]);

  const filteredIPs = useMemo(() => {
    const q = search.trim().toLowerCase();
    const ips = ipAddresses ?? [];
    if (!q) return ips;
    return ips.filter(
      (ip) =>
        ip.ipAddress?.toLowerCase().includes(q) ||
        (ip.label || '').toLowerCase().includes(q) ||
        (ip.description || '').toLowerCase().includes(q)
    );
  }, [search, ipAddresses]);

  const handleAdd = () => {
    setEditEntry(null);
    setModalOpen(true);
  };

  const handleEdit = (entry: IPWhitelistEntry) => {
    setEditEntry(entry);
    setModalOpen(true);
  };

  const handleDelete = async (entry: IPWhitelistEntry) => {
    try {
      await api.removeIPWhitelist(tenantId, entry.ipAddress);
      await fetchData();
    } catch (err) {
      console.error('Failed to delete IP:', err);
    }
  };

  const handleSave = async (data: {
    ipAddress: string;
    label?: string;
    description?: string;
    tenantIds: string[];
    allTenants?: boolean;
  }) => {
    try {
      if (editEntry) {
        await api.updateIPAssignments(
          editEntry.ipAddress,
          tenantId,
          data.tenantIds,
          { allTenants: data.allTenants, prevTenantIds: editEntry.tenantIds || [] }
        );
      } else {
        await api.addIPWhitelist(
          tenantId,
          data.ipAddress,
          data.label,
          data.description,
          data.tenantIds.filter((id) => id !== tenantId),
          { allTenants: data.allTenants }
        );
      }
      await fetchData();
    } catch (err) {
      console.error('Failed to save IP:', err);
      throw err;
    }
  };

  const handleExportCSV = () => {
    const headers = ['IP Address', 'Label', 'Description', 'Created At', 'Updated At'];
    const rows = filteredIPs.map((ip) => [
      ip.ipAddress,
      ip.label || '',
      ip.description || '',
      ip.createdAt || '',
      ip.updatedAt || '',
    ]);
    const csv = [headers, ...rows].map((r) => r.join(',')).join('\n');
    downloadFile(csv, `${tenantId}-ip-whitelist.csv`, 'text/csv');
    setExportAnchor(null);
  };

  const handleExportJSON = () => {
    const json = JSON.stringify({ tenantId, ipAddresses: filteredIPs }, null, 2);
    downloadFile(json, `${tenantId}-ip-whitelist.json`, 'application/json');
    setExportAnchor(null);
  };

  const downloadFile = (content: string, filename: string, mimeType: string) => {
    const blob = new Blob([content], { type: mimeType });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    a.click();
    URL.revokeObjectURL(url);
  };

  const currentTenant: Tenant = useMemo(
    () => ({
      id: tenantId,
      displayName: tenants.find((t) => t.id === tenantId)?.displayName || tenantId,
    }),
    [tenantId, tenants]
  );

  return (
    <Box sx={{ p: 0 }}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, mb: 2 }}>
        <Tooltip title="Add IP Address">
          <Button variant="contained" startIcon={<AddIcon />} onClick={handleAdd}>
            Add IP
          </Button>
        </Tooltip>
        <TextField
          size="small"
          placeholder="Search IP, label, description..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <SearchIcon sx={{ color: 'action.active' }} />
              </InputAdornment>
            ),
          }}
          sx={{ width: 300 }}
        />
        <Box sx={{ flexGrow: 1 }} />
        <Tooltip title="Export">
          <IconButton onClick={(e) => setExportAnchor(e.currentTarget)}>
            <DownloadIcon />
          </IconButton>
        </Tooltip>
        <Menu
          anchorEl={exportAnchor}
          open={Boolean(exportAnchor)}
          onClose={() => setExportAnchor(null)}
        >
          <MenuItem onClick={handleExportCSV}>
            <ListItemIcon>
              <DownloadIcon fontSize="small" />
            </ListItemIcon>
            <ListItemText>Export CSV</ListItemText>
          </MenuItem>
          <MenuItem onClick={handleExportJSON}>
            <ListItemIcon>
              <DownloadIcon fontSize="small" />
            </ListItemIcon>
            <ListItemText>Export JSON</ListItemText>
          </MenuItem>
        </Menu>
      </Box>

      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        Showing {filteredIPs?.length ?? 0} of {ipAddresses?.length ?? 0} IP addresses
      </Typography>

      <TableContainer component={Paper}>
        <Table>
          <TableHead>
            <TableRow>
              <TableCell>IP Address</TableCell>
              <TableCell>Label</TableCell>
              <TableCell>Description</TableCell>
              <TableCell>Status</TableCell>
              <TableCell align="right">Actions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {loading ? (
              <TableRow>
                <TableCell colSpan={5} align="center">
                  Loading...
                </TableCell>
              </TableRow>
            ) : filteredIPs.length === 0 ? (
              <TableRow>
                <TableCell colSpan={5} align="center">
                  No IP addresses found
                </TableCell>
              </TableRow>
            ) : (
              filteredIPs.map((ip, idx) => (
                <TableRow key={`${ip.ipAddress}-${idx}`}>
                  <TableCell>
                    <Typography variant="body2" fontFamily="monospace">
                      {ip.ipAddress}
                    </Typography>
                  </TableCell>
                  <TableCell>{ip.label || '—'}</TableCell>
                  <TableCell>
                    <Typography
                      variant="body2"
                      sx={{
                        maxWidth: 300,
                        overflow: 'hidden',
                        textOverflow: 'ellipsis',
                        whiteSpace: 'nowrap',
                      }}
                    >
                      {ip.description || '—'}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    {(ip as any).allTenants === true ? (
                      <Chip label="All Tenants" size="small" color="info" />
                    ) : (ip.tenantIds?.length ?? 0) > 0 ? (
                      <Chip
                        label={`${ip.tenantIds?.length ?? 0} tenant(s)`}
                        size="small"
                        variant="outlined"
                      />
                    ) : (
                      <Chip label="Unassigned" size="small" />
                    )}
                  </TableCell>
                  <TableCell align="right">
                    <Tooltip title="Edit">
                      <IconButton size="small" onClick={() => handleEdit(ip)}>
                        <EditIcon fontSize="small" />
                      </IconButton>
                    </Tooltip>
                    <Tooltip title="Delete">
                      <IconButton
                        size="small"
                        color="error"
                        onClick={() => handleDelete(ip)}
                      >
                        <DeleteIcon fontSize="small" />
                      </IconButton>
                    </Tooltip>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableContainer>

      <IPAddEditDialog
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        onSave={handleSave}
        tenants={tenants}
        editingEntry={editEntry}
        initialTenantId={tenantId}
      />
    </Box>
  );
}
