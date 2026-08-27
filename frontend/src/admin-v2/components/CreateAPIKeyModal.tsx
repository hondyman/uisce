import React, { useState } from "react";
import { useTheme } from "@mui/material/styles";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import TextField from "@mui/material/TextField";
import Button from "@mui/material/Button";
import FormControlLabel from "@mui/material/FormControlLabel";
import Checkbox from "@mui/material/Checkbox";
import Paper from "@mui/material/Paper";
import { Modal } from "./Modal";
import { ErrorBanner, SuccessBanner } from "./Feedback";
import { useCreateAPIKey, useAPIKeys } from "../hooks/useAPIKeys";
import { useTenants } from "../hooks/useTenants";
import type { CreateAPIKeyRequest } from "@/admin-v2/types";

export interface CreateAPIKeyModalProps {
  open: boolean;
  onClose: () => void;
  onSuccess?: () => void;
}

export function CreateAPIKeyModal({
  open,
  onClose,
  onSuccess,
}: CreateAPIKeyModalProps) {
  const theme = useTheme();
  const isDark = theme.palette.mode === 'dark';
  const [formData, setFormData] = useState<CreateAPIKeyRequest>({
    name: "",
    tenantIds: [],
  });

  const [showSuccess, setShowSuccess] = useState(false);
  const [generatedKey, setGeneratedKey] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const createMutation = useCreateAPIKey();
  const tenantsQuery = useTenants();

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { value } = e.currentTarget;
    setFormData((prev) => ({ ...prev, name: value }));
  };

  const handleTenantToggle = (tenantId: string) => {
    setFormData((prev) => ({
      ...prev,
      tenantIds: prev.tenantIds.includes(tenantId)
        ? prev.tenantIds.filter((id) => id !== tenantId)
        : [...prev.tenantIds, tenantId],
    }));
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    if (!formData.name.trim()) {
      return;
    }

    createMutation.mutate(formData, {
      onSuccess: (response) => {
        if (response.data?.key) {
          setGeneratedKey(response.data.key);
        }
        setShowSuccess(true);
        setTimeout(() => {
          setFormData({ name: "", tenantIds: [] });
          setShowSuccess(false);
          setGeneratedKey(null);
          setCopied(false);
          onClose();
          onSuccess?.();
        }, 3000);
      },
    });
  };

  const handleCopyKey = () => {
    if (generatedKey) {
      navigator.clipboard.writeText(generatedKey).then(() => {
        setCopied(true);
        setTimeout(() => setCopied(false), 2000);
      });
    }
  };

  const isLoading = createMutation.isPending;
  const isFormValid = formData.name.trim().length > 0;

  if (generatedKey) {
    return (
      <Modal open={open} onClose={onClose} title="API Key Created" size="md">
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
          <SuccessBanner message="API key created successfully. Copy it now as you won't see it again." />

          <Paper
            variant="outlined"
            sx={{ p: 2, bgcolor: isDark ? '#1f2937' : '#f9fafb' }}
          >
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
              <Box
                component="code"
                sx={{
                  flex: 1,
                  fontFamily: 'monospace',
                  fontSize: '0.875rem',
                  wordBreak: 'break-all',
                }}
              >
                {generatedKey}
              </Box>
              <Button
                variant={copied ? "outlined" : "contained"}
                size="small"
                onClick={handleCopyKey}
              >
                {copied ? "✓ Copied" : "Copy Key"}
              </Button>
            </Box>
          </Paper>

          <Box>
            <Typography variant="body2">
              <strong>Name:</strong> {formData.name}
            </Typography>
            {formData.tenantIds.length > 0 && (
              <Typography variant="body2">
                <strong>Tenants:</strong> {formData.tenantIds.length} selected
              </Typography>
            )}
          </Box>

          <Box sx={{ display: 'flex', justifyContent: 'flex-end', pt: 1 }}>
            <Button variant="contained" onClick={onClose}>
              Done
            </Button>
          </Box>
        </Box>
      </Modal>
    );
  }

  return (
    <Modal open={open} onClose={onClose} title="Create API Key" size="md">
      {createMutation.isError && (
        <ErrorBanner
          message={
            createMutation.error instanceof Error
              ? createMutation.error.message
              : "Failed to create API key"
          }
        />
      )}

      <Box component="form" onSubmit={handleSubmit} sx={{ display: 'flex', flexDirection: 'column', gap: 2.5 }}>
        <Box>
          <Typography variant="body2" sx={{ fontWeight: 600, mb: 1 }}>Key Name *</Typography>
          <TextField
            fullWidth
            id="name"
            name="name"
            type="text"
            value={formData.name}
            onChange={handleChange}
            placeholder="e.g., Production API Key"
            disabled={isLoading}
            required
            size="small"
          />
        </Box>

        <Box>
          <Typography variant="body2" sx={{ fontWeight: 600, mb: 1 }}>Tenant Access</Typography>
          <Paper
            variant="outlined"
            sx={{
              p: 1.5,
              maxHeight: 150,
              overflow: 'auto',
              bgcolor: isDark ? '#1f2937' : '#f9fafb'
            }}
          >
            {tenantsQuery.isLoading ? (
              <Typography variant="body2" color="text.secondary">Loading tenants...</Typography>
            ) : tenantsQuery.data?.data?.length === 0 ? (
              <Typography variant="body2" color="text.secondary">No tenants available</Typography>
            ) : (
              tenantsQuery.data?.data?.map((tenant) => (
                <FormControlLabel
                  key={tenant.id}
                  control={
                    <Checkbox
                      checked={formData.tenantIds.includes(tenant.id)}
                      onChange={() => handleTenantToggle(tenant.id)}
                      disabled={isLoading}
                      size="small"
                    />
                  }
                  label={tenant.name}
                />
              ))
            )}
          </Paper>
          <Typography variant="caption" color="text.secondary" sx={{ mt: 0.5, display: 'block' }}>
            Leave empty for access to all tenants (admin key)
          </Typography>
        </Box>

        <Box sx={{ display: 'flex', justifyContent: 'flex-end', gap: 1, pt: 1 }}>
          <Button
            type="button"
            variant="outlined"
            onClick={onClose}
            disabled={isLoading}
          >
            Cancel
          </Button>
          <Button
            type="submit"
            variant="contained"
            disabled={!isFormValid || isLoading}
          >
            {isLoading ? "Creating..." : "Create API Key"}
          </Button>
        </Box>
      </Box>
    </Modal>
  );
}
