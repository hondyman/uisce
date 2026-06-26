import React, { useState } from "react";
import { Modal } from "./Modal";
import { ErrorBanner, SuccessBanner } from "./Feedback";
import { useCreateAPIKey, useAPIKeys } from "../hooks/useAPIKeys";
import { useTenants } from "../hooks/useTenants";
import type { CreateAPIKeyRequest } from "../types";
import "./CreateAPIKeyModal.css";

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
        // Show the plaintext key for user to copy
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

  // If key was generated, show the display-only view
  if (generatedKey) {
    return (
      <Modal open={open} onClose={onClose} title="API Key Created" size="md">
        <div className="key-display">
          <SuccessBanner message="API key created successfully. Copy it now as you won't see it again." />

          <div className="key-box">
            <code className="key-value">{generatedKey}</code>
            <button
              type="button"
              onClick={handleCopyKey}
              className="btn btn-small"
            >
              {copied ? "✓ Copied" : "Copy Key"}
            </button>
          </div>

          <div className="key-info">
            <p>
              <strong>Name:</strong> {formData.name}
            </p>
            {formData.tenantIds.length > 0 && (
              <p>
                <strong>Tenants:</strong> {formData.tenantIds.length} selected
              </p>
            )}
          </div>

          <div className="modal-actions">
            <button
              type="button"
              onClick={onClose}
              className="btn btn-primary"
            >
              Done
            </button>
          </div>
        </div>
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

      <form onSubmit={handleSubmit} className="form">
        <div className="form-group">
          <label htmlFor="name">Key Name *</label>
          <input
            id="name"
            name="name"
            type="text"
            value={formData.name}
            onChange={handleChange}
            placeholder="e.g., Production API Key"
            disabled={isLoading}
            required
          />
        </div>

        <div className="form-group">
          <label>Tenant Access</label>
          <div className="tenant-list">
            {tenantsQuery.isLoading ? (
              <p className="placeholder">Loading tenants...</p>
            ) : tenantsQuery.data?.data?.length === 0 ? (
              <p className="placeholder">No tenants available</p>
            ) : (
              tenantsQuery.data?.data?.map((tenant) => (
                <label key={tenant.id} className="checkbox-label">
                  <input
                    type="checkbox"
                    checked={formData.tenantIds.includes(tenant.id)}
                    onChange={() => handleTenantToggle(tenant.id)}
                    disabled={isLoading}
                  />
                  <span>{tenant.name}</span>
                </label>
              ))
            )}
          </div>
          <small className="hint">
            Leave empty for access to all tenants (admin key)
          </small>
        </div>

        <div className="form-actions">
          <button
            type="button"
            onClick={onClose}
            disabled={isLoading}
            className="btn btn-secondary"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={!isFormValid || isLoading}
            className="btn btn-primary"
          >
            {isLoading ? "Creating..." : "Create API Key"}
          </button>
        </div>
      </form>
    </Modal>
  );
}
