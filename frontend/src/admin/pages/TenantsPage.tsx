import React, { useCallback, useEffect, useState } from "react";
import { useTheme } from "@mui/material/styles";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import Button from "@mui/material/Button";
import Paper from "@mui/material/Paper";
import TextField from "@mui/material/TextField";
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
import Dialog from "@mui/material/Dialog";
import DialogTitle from "@mui/material/DialogTitle";
import DialogContent from "@mui/material/DialogContent";
import DialogActions from "@mui/material/DialogActions";
import IconButton from "@mui/material/IconButton";
import { useTenants } from "../hooks/useAdmin";
import { Tenant } from "../types";
import { useAuth } from "../../contexts/AuthContext";
import { ImpersonationModal } from "../../components/admin/ImpersonationModal";
import { ImpersonationTenantPicker } from "../../components/admin/ImpersonationTenantPicker";
import { useImpersonation } from "../../contexts/ImpersonationContext";
import type { ImpersonationScope } from "../../contexts/ImpersonationContext";
import type { ActiveImpersonationSession } from "../../contexts/ImpersonationContext";

interface TenantFormData {
  name: string;
  code: string;
  region: string;
  plan: "free" | "pro" | "enterprise";
}

export const TenantsPage: React.FC = () => {
  const theme = useTheme();
  const [limit] = useState(50);
  const [offset, setOffset] = useState(0);
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [formData, setFormData] = useState<TenantFormData>({
    name: "",
    code: "",
    region: "us-east-1",
    plan: "free",
  });
  const [impersonateTenant, setImpersonateTenant] = useState<Tenant | null>(null);
  const [pickerOpen, setPickerOpen] = useState(false);
  const [pendingScope, setPendingScope] = useState<ImpersonationScope | null>(null);
  const [activeSessions, setActiveSessions] = useState<ActiveImpersonationSession[]>([]);

  const { token: adminToken } = useAuth();
  const { recentSessions, clearRecentSessions, listActiveSessions } = useImpersonation();

  const refreshActiveSessions = useCallback(async () => {
    try {
      const sessions = await listActiveSessions();
      setActiveSessions(sessions);
    } catch {
      setActiveSessions([]);
    }
  }, [listActiveSessions]);

  const openPickerFor = (t: Tenant) => {
    setImpersonateTenant(t);
    setPendingScope(null);
    setPickerOpen(true);
    void refreshActiveSessions();
  };

  const handlePickerSelect = (
    tenant: { id: string; name: string },
    scope: ImpersonationScope,
  ) => {
    setImpersonateTenant({
      id: tenant.id,
      name: tenant.name,
      gold_copy: false,
      is_active: true,
      description: null,
    } as unknown as Tenant);
    setPendingScope(scope);
    setPickerOpen(false);
  };

  const { isGlobalAdmin } = useAuth();
  const { tenants, total, loading, error, refetch } = useTenants(limit, offset);

  const handleCreateClick = () => {
    setShowCreateForm(true);
  };

  const handleFormChange = (
    e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>
  ) => {
    const { name, value } = e.target;
    setFormData((prev) => ({
      ...prev,
      [name]: value,
    }));
  };

  const handleCreateTenant = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const response = await fetch("/api/admin/tenants", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(formData),
      });
      if (!response.ok) {
        const error = await response.json();
        alert(`Error: ${error.error || "Failed to create tenant"}`);
        return;
      }
      refetch();
      setShowCreateForm(false);
      setFormData({
        name: "",
        code: "",
        region: "us-east-1",
        plan: "free",
      });
    } catch (err) {
      alert(
        `Error: ${err instanceof Error ? err.message : "Unknown error"}`
      );
    }
  };

  const pageCount = Math.ceil(total / limit);
  const currentPage = Math.floor(offset / limit) + 1;

  return (
    <Box sx={{ p: 3 }}>
      <Box sx={{ display: "flex", justifyContent: "space-between", alignItems: "center", mb: 3 }}>
        <Typography variant="h4" sx={{ fontWeight: 600 }}>
          Tenants
        </Typography>
        <Box sx={{ display: "flex", gap: 1 }}>
          {isGlobalAdmin() && (
            <Button
              variant="contained"
              onClick={() => {
                setImpersonateTenant(null);
                setPendingScope(null);
                setPickerOpen(true);
                void refreshActiveSessions();
              }}
              sx={{ bgcolor: "warning.main", color: "white", fontWeight: 600, "&:hover": { bgcolor: "warning.dark" } }}
            >
              Assume Context (Pick Tenant)
            </Button>
          )}
          <Button variant="contained" onClick={handleCreateClick}>
            + New Tenant
          </Button>
        </Box>
      </Box>

      {error && (
        <Paper sx={{ mb: 3, p: 2, bgcolor: "error.light", color: "error.dark" }}>
          {error}
        </Paper>
      )}

      <Dialog open={showCreateForm} onClose={() => setShowCreateForm(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Create New Tenant</DialogTitle>
        <form onSubmit={handleCreateTenant}>
          <DialogContent>
            <Box sx={{ display: "flex", flexDirection: "column", gap: 2, pt: 1 }}>
              <TextField
                label="Tenant Name *"
                id="name"
                name="name"
                value={formData.name}
                onChange={handleFormChange}
                placeholder="e.g., Acme Corp"
                fullWidth
                required
              />

              <TextField
                label="Tenant Code *"
                id="code"
                name="code"
                value={formData.code}
                onChange={handleFormChange}
                placeholder="e.g., acme-corp"
                fullWidth
                required
              />

              <Box sx={{ display: "flex", gap: 2 }}>
                <FormControl fullWidth>
                  <InputLabel id="region-label">Region</InputLabel>
                  <Select
                    labelId="region-label"
                    id="region"
                    name="region"
                    value={formData.region}
                    label="Region"
                    onChange={handleFormChange as any}
                  >
                    <MenuItem value="us-east-1">US East 1</MenuItem>
                    <MenuItem value="us-west-2">US West 2</MenuItem>
                    <MenuItem value="eu-west-1">EU West 1</MenuItem>
                    <MenuItem value="eu-central-1">EU Central 1</MenuItem>
                    <MenuItem value="ap-southeast-1">AP Southeast 1</MenuItem>
                    <MenuItem value="ap-northeast-1">AP Northeast 1</MenuItem>
                  </Select>
                </FormControl>

                <FormControl fullWidth>
                  <InputLabel id="plan-label">Plan</InputLabel>
                  <Select
                    labelId="plan-label"
                    id="plan"
                    name="plan"
                    value={formData.plan}
                    label="Plan"
                    onChange={handleFormChange as any}
                  >
                    <MenuItem value="free">Free (100 req/day)</MenuItem>
                    <MenuItem value="pro">Pro (10k req/day)</MenuItem>
                    <MenuItem value="enterprise">Enterprise (Unlimited)</MenuItem>
                  </Select>
                </FormControl>
              </Box>
            </Box>
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setShowCreateForm(false)}>Cancel</Button>
            <Button type="submit" variant="contained">Create Tenant</Button>
          </DialogActions>
        </form>
      </Dialog>

      <TableContainer component={Paper}>
        {loading && (
          <Box sx={{ p: 4, textAlign: "center" }}>Loading tenants...</Box>
        )}

        {!loading && tenants.length === 0 && (
          <Box sx={{ p: 4, textAlign: "center", color: "text.secondary" }}>
            No tenants found. Create one to get started.
          </Box>
        )}

        {!loading && tenants.length > 0 && (
          <Table>
            <TableHead>
              <TableRow>
                <TableCell sx={{ fontWeight: 600 }}>Name</TableCell>
                <TableCell sx={{ fontWeight: 600 }}>Code</TableCell>
                <TableCell sx={{ fontWeight: 600 }}>Region</TableCell>
                <TableCell sx={{ fontWeight: 600 }}>Plan</TableCell>
                <TableCell sx={{ fontWeight: 600 }}>Status</TableCell>
                <TableCell sx={{ fontWeight: 600 }}>Rate Limit</TableCell>
                <TableCell sx={{ fontWeight: 600 }}>Created</TableCell>
                <TableCell sx={{ fontWeight: 600 }}>Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {tenants.map((tenant) => (
                <TableRow key={tenant.id} sx={tenant.is_suspended ? { opacity: 0.6 } : {}}>
                  <TableCell>
                    <Typography sx={{ fontWeight: 600 }}>{tenant.name}</Typography>
                  </TableCell>
                  <TableCell>{tenant.code}</TableCell>
                  <TableCell>{tenant.region}</TableCell>
                  <TableCell>
                    <Chip 
                      label={tenant.plan.charAt(0).toUpperCase() + tenant.plan.slice(1)} 
                      size="small"
                      sx={{ 
                        bgcolor: tenant.plan === 'enterprise' ? 'primary.light' : tenant.plan === 'pro' ? 'info.light' : 'default',
                        color: tenant.plan === 'enterprise' || tenant.plan === 'pro' ? 'white' : 'text.primary'
                      }} 
                    />
                  </TableCell>
                  <TableCell>
                    {tenant.is_suspended ? (
                      <Chip label="Suspended" size="small" color="error" />
                    ) : (
                      <Chip label="Active" size="small" color="success" />
                    )}
                  </TableCell>
                  <TableCell>
                    {tenant.max_requests} / {tenant.window_seconds}s
                  </TableCell>
                  <TableCell>
                    {new Date(tenant.created_at).toLocaleDateString()}
                  </TableCell>
                  <TableCell>
                    <Button size="small">Details</Button>
                    <Button size="small">Edit</Button>
                    {isGlobalAdmin() && (
                      <Button 
                        size="small"
                        onClick={() => openPickerFor(tenant)}
                        sx={{ color: "warning.main", fontWeight: 600 }}
                      >
                        Assume Context
                      </Button>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </TableContainer>

      {pageCount > 1 && (
        <Box sx={{ display: "flex", justifyContent: "center", alignItems: "center", mt: 2, gap: 2 }}>
          <Button
            disabled={offset === 0}
            onClick={() => setOffset(Math.max(0, offset - limit))}
            variant="outlined"
            size="small"
          >
            Previous
          </Button>
          <Typography variant="body2">
            Page {currentPage} of {pageCount} ({total} total)
          </Typography>
          <Button
            disabled={offset + limit >= total}
            onClick={() => setOffset(offset + limit)}
            variant="outlined"
            size="small"
          >
            Next
          </Button>
        </Box>
      )}

      {adminToken && (
        <ImpersonationTenantPicker
          open={pickerOpen}
          onClose={() => setPickerOpen(false)}
          adminToken={adminToken}
          recentSessions={recentSessions}
          onClearRecentSessions={clearRecentSessions}
          onSelect={handlePickerSelect}
          initialTenant={null}
          activeSessions={activeSessions}
        />
      )}

      {impersonateTenant && (
        <ImpersonationModal
          open={!!impersonateTenant}
          onClose={() => {
            setImpersonateTenant(null);
            setPendingScope(null);
          }}
          targetTenantId={impersonateTenant.id}
          targetTenantName={impersonateTenant.name}
          initialScope={pendingScope ?? undefined}
        />
      )}
    </Box>
  );
};

export default TenantsPage;