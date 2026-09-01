import React, { useState } from "react";
import { useAPIKeys } from "../hooks/useAdmin";
import { APIKey } from "../types";
import { Box, Button, Chip, Table, TableBody, TableCell, TableContainer, TableHead, TableRow } from "@mui/material";
import { CircularProgress } from "@mui/material";
import { useTheme } from "@mui/material/styles";

const getAdminHeaders = () => {
  const headers: Record<string, string> = { "Content-Type": "application/json" };
  const token = localStorage.getItem("auth_token");
  if (token) headers["Authorization"] = `Bearer ${token}`;
  return headers;
};

export const APIKeysPage: React.FC = () => {
  const theme = useTheme();
  const [limit, setLimit] = useState(50);
  const [offset, setOffset] = useState(0);
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [formData, setFormData] = useState({
    name: "",
    tenant_ids: "",
    roles: ["USER"],
  });

  const { keys, total, loading, error, refetch } = useAPIKeys(limit, offset);

  const handleCreateClick = () => {
    setShowCreateForm(true);
  };

  const handleFormChange = (
    e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>
  ) => {
    const { name, value } = e.target;
    setFormData((prev) => ({
      ...prev,
      [name]: value,
    }));
  };

  const handleRoleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const role = e.target.value;
    setFormData((prev) => ({
      ...prev,
      roles: e.target.checked
        ? [...prev.roles, role]
        : prev.roles.filter((r) => r !== role),
    }));
  };

  const handleCreateKey = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const tenantIds = formData.tenant_ids
        .split(",")
        .map((id) => id.trim())
        .filter((id) => id);

      const response = await fetch("/api/admin/api-keys", {
        method: "POST",
        headers: getAdminHeaders(),
        body: JSON.stringify({
          name: formData.name,
          tenant_ids: tenantIds,
          roles: formData.roles,
        }),
      });

      if (!response.ok) {
        const error = await response.json();
        alert(`Error: ${error.error || "Failed to create API key"}`);
        return;
      }

      const data = await response.json();
      alert(
        `API Key created successfully!\n\nKey: ${data.api_key.key}\n\nMake sure to copy and save this key securely. You won't be able to see it again.`
      );

      refetch();
      setShowCreateForm(false);
      setFormData({
        name: "",
        tenant_ids: "",
        roles: ["USER"],
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
      <Box
        sx={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          mb: 3,
        }}
      >
        <Box>
          <Typography variant="h4" component="h1" sx={{ fontWeight: 600, mb: 0 }}>
            API Keys
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Manage authentication tokens
          </Typography>
        </Box>
        <Button
          variant="contained"
          onClick={() => setShowCreateForm(true)}
          sx={{ textTransform: "none" }}
        >
          + Create Key
        </Button>
      </Box>

      {error && (
        <Box sx={{ p: 3, bgcolor: "error.light", color: "error.dark" }}>
          {error}
        </Box>
      )}

      {/* Create Form Modal */}
      {showCreateForm && (
        <Box sx={{ p: 3 }}>
          <Box sx={{ bg: "background.paper", px: 4, py: 4, borderRadius: 2 }}>
            <Box sx={{ display: "flex", justifyContent: "space-between", alignItems: "center", mb: 3 }}>
              <h2>Create New API Key</h2>
              <button sx={{ color: "text.secondary" }} onClick={() => setShowCreateForm(false)}>
                ✕
              </button>
            </Box>
            <form sx={{ display: "flex", flexDirection: "column", gap: 3 }} onSubmit={handleCreateKey}>
              <div sx={{ display: "flex", flexDirection: "column", gap: 2 }}>
                <label htmlFor="name" sx={{ fontWeight: 500, mb: 1 }}>API Key Name *</label>
                <input
                  id="name"
                  type="text"
                  name="name"
                  value={formData.name}
                  onChange={handleFormChange}
                  placeholder="e.g., Production Key"
                  required
                />
              </div>

              <div sx={{ display: "flex", flexDirection: "column", gap: 2 }}>
                <label htmlFor="tenant_ids" sx={{ fontWeight: 500, mb: 1 }}>Tenant IDs *</label>
                <textarea
                  id="tenant_ids"
                  name="tenant_ids"
                  value={formData.tenant_ids}
                  onChange={handleFormChange}
                  placeholder="Enter tenant UUIDs, comma-separated"
                  rows={3}
                  required
                />
                <small sx={{ color: "text.secondary", fontSize: "0.75rem" }}>
                  Leave empty for global access. Multiple UUIDs: comma-separated.
                </small>
              </div>

              <div sx={{ display: "flex", flexDirection: "column", gap: 2 }}>
                <label>Roles</label>
                <div sx={{ display: "flex", gap: 3 }}>
                  {["USER", "TENANT_ADMIN", "GLOBAL_OPS"].map((role) => (
                    <label key={role} sx={{ display: "flex", alignItems: "center", gap: 1, userSelect: "none" }}>
                      <input
                        type="checkbox"
                        value={role}
                        checked={formData.roles.includes(role)}
                        onChange={handleRoleChange}
                      />
                      {role}
                    </label>
                  ))}
                </div>
              </div>

              <div sx={{ display: "flex", justifyContent: "flex-end", gap: 2, mt: 4 }}>
                <button
                  type="button"
                  sx={{
                    variant: "outlined",
                    color: "text.secondary",
                    "&:hover": { opacity: 0.7 },
                    "&:disabled": { opacity: 0.5 },
                  }}
                  onClick={() => setShowCreateForm(false)}
                >
                  Cancel
                </button>
                <button type="submit" sx={{ variant: "contained" }}>Create API Key</button>
              </div>
            </form>
          </Box>
        </Box>
      )}

      {/* API Keys Table */}
      <Box sx={{ mt: 3 }}>
        {loading && (
          <Box sx={{ display: "flex", justifyContent: "center", p: 4 }}>
            <CircularProgress size="medium" />
          </Box>
        )}

        {!loading && keys.length === 0 && (
          <Box sx={{ p: 4, textAlign: "center", color: "text.secondary" }}>
            <p>No API keys found. Create one to get started.</p>
          </Box>
        )}

        {!loading && keys.length > 0 && (
          <TableContainer component={Paper}>
            <Table>
              <TableHead>
                <TableRow>
                  <TableCell sx={{ fontWeight: 600 }}>Name</TableCell>
                  <TableCell sx={{ fontWeight: 600 }}>Tenant IDs</TableCell>
                  <TableCell sx={{ fontWeight: 600 }}>Roles</TableCell>
                  <TableCell sx={{ fontWeight: 600 }}>Status</TableCell>
                  <TableCell sx={{ fontWeight: 600 }}>Created</TableCell>
                  <TableCell sx={{ fontWeight: 600, textAlign: "right" }}>Actions</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {keys.map((key: APIKey) => (
                  <TableRow key={key.id} sx={{ '&:hover': { bgc: "grey.50" } }}>
                    <TableCell>
                      <strong>{key.name}</strong>
                    </TableCell>
                    <TableCell>
                      {key.tenant_ids?.length === 0 ? (
                        <Chip label="Global" size="small" sx={{ bgcolor: "grey.100", color: "grey.800" }} />
                      ) : (
                        <Chip label={`${key.tenant_ids.length} tenant(s)`} size="small" sx={{ bgcolor: "grey.100", color: "grey.800" }} />
                      )}
                    </TableCell>
                    <TableCell>
                      {key.roles?.length > 0 ? (
                        <div sx={{ display: "flex", gap: 2 }}>
                          {key.roles.map((role) => (
                            <span key={role} sx={{ display: "inline-block", px: 3, py: 1, borderRadius: 12, fontSize: "0.75rem" }}>
                              {role}
                            </span>
                          ))}
                        </div>
                      ) : (
                        <span sx={{ color: "text.secondary" }}>—</span>
                      )}
                    </TableCell>
                    <TableCell>
                      {key.is_revoked ? (
                        <Chip label="Revoked" size="small" sx={{ bgcolor: "error.light", color: "error.dark" }} />
                      ) : (
                        <Chip label="Active" size="small" sx={{ bgcolor: "success.light", color: "success.dark" }} />
                      )}
                    </TableCell>
                    <TableCell>{new Date(key.created_at).toLocaleDateString()}</TableCell>
                    <TableCell align="right" sx={{ fontSize: "0.875rem" }}>
                      <Button variant="outlined" size="small">Usage</Button>
                      <Button variant="outlined" size="small">Revoke</Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        )}
      </Box>

      {/* Pagination */}
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
          <span>Page {currentPage} of {pageCount} ({total} total)</span>
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
    </Box>
  );
};

export default APIKeysPage;