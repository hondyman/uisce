import React, { useState } from "react";
import { Modal } from "./Modal";
import { ErrorBanner, Spinner, SuccessBanner } from "./Feedback";
import { useCreateTenant } from "../hooks/useTenants";
import type { CreateTenantRequest } from "../types";
import "./CreateTenantModal.css";

export interface CreateTenantModalProps {
  open: boolean;
  onClose: () => void;
  onSuccess?: () => void;
}

export function CreateTenantModal({
  open,
  onClose,
  onSuccess,
}: CreateTenantModalProps) {
  const [formData, setFormData] = useState<CreateTenantRequest>({
    name: "",
    code: "",
    region: "us-east-1",
    plan: "free",
    maxRequests: 10000,
    windowSeconds: 3600,
  });

  const [showSuccess, setShowSuccess] = useState(false);
  const createMutation = useCreateTenant();

  const handleChange = (
    e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>
  ) => {
    const { name, value, type } = e.currentTarget;
    setFormData((prev) => ({
      ...prev,
      [name]:
        type === "number" ? parseInt(value, 10) : value,
    }));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    // Validation
    if (!formData.name.trim()) {
      return; // Form will show error state via validation feedback
    }
    if (!formData.code.trim()) {
      return;
    }

    // Submit mutation
    createMutation.mutate(formData, {
      onSuccess: () => {
        setShowSuccess(true);
        setTimeout(() => {
          setFormData({
            name: "",
            code: "",
            region: "us-east-1",
            plan: "free",
            maxRequests: 10000,
            windowSeconds: 3600,
          });
          setShowSuccess(false);
          onClose();
          onSuccess?.();
        }, 1500);
      },
    });
  };

  const isLoading = createMutation.isPending;
  const isFormValid =
    formData.name.trim().length > 0 && formData.code.trim().length > 0;

  return (
    <Modal open={open} onClose={onClose} title="Create Tenant" size="md">
      {showSuccess && (
        <SuccessBanner message={`Tenant "${formData.name}" created successfully`} />
      )}

      {createMutation.isError && (
        <ErrorBanner
          message={
            createMutation.error instanceof Error
              ? createMutation.error.message
              : "Failed to create tenant"
          }
        />
      )}

      <form onSubmit={handleSubmit} className="form">
        <div className="form-group">
          <label htmlFor="name">Name *</label>
          <input
            id="name"
            name="name"
            type="text"
            value={formData.name}
            onChange={handleChange}
            placeholder="ACME Corp"
            disabled={isLoading}
            required
          />
        </div>

        <div className="form-group">
          <label htmlFor="code">Code *</label>
          <input
            id="code"
            name="code"
            type="text"
            value={formData.code}
            onChange={handleChange}
            placeholder="acme-corp"
            disabled={isLoading}
            required
          />
        </div>

        <div className="form-group">
          <label htmlFor="region">Region</label>
          <select
            id="region"
            name="region"
            value={formData.region}
            onChange={handleChange}
            disabled={isLoading}
          >
            <option value="us-east-1">US East (N. Virginia)</option>
            <option value="us-west-2">US West (Oregon)</option>
            <option value="eu-west-1">EU (Ireland)</option>
            <option value="ap-southeast-1">Asia Pacific (Singapore)</option>
          </select>
        </div>

        <div className="form-group">
          <label htmlFor="plan">Plan</label>
          <select
            id="plan"
            name="plan"
            value={formData.plan}
            onChange={handleChange}
            disabled={isLoading}
          >
            <option value="free">Free</option>
            <option value="pro">Pro</option>
            <option value="enterprise">Enterprise</option>
          </select>
        </div>

        <div className="form-row">
          <div className="form-group">
            <label htmlFor="maxRequests">Max Requests</label>
            <input
              id="maxRequests"
              name="maxRequests"
              type="number"
              value={formData.maxRequests}
              onChange={handleChange}
              disabled={isLoading}
              min="1000"
              step="1000"
            />
          </div>

          <div className="form-group">
            <label htmlFor="windowSeconds">Window (seconds)</label>
            <input
              id="windowSeconds"
              name="windowSeconds"
              type="number"
              value={formData.windowSeconds}
              onChange={handleChange}
              disabled={isLoading}
              min="60"
              step="60"
            />
          </div>
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
            {isLoading ? "Creating..." : "Create Tenant"}
          </button>
        </div>

        {isLoading && <Spinner size="sm" />}
      </form>
    </Modal>
  );
}
