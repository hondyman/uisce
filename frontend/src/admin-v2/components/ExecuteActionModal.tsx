import React, { useState } from "react";
import {
  useExecuteOpsAction,
  OPS_ACTIONS,
  getActionById,
  ExecuteActionRequest,
} from "../hooks/useOpsActions";
import { Spinner } from "./Feedback";
import "./ExecuteActionModal.css";

export interface ExecuteActionModalProps {
  incidentId: string;
  actionType: string;
  isOpen: boolean;
  onClose: () => void;
  onSuccess?: () => void;
}

export function ExecuteActionModal({
  incidentId,
  actionType,
  isOpen,
  onClose,
  onSuccess,
}: ExecuteActionModalProps) {
  const [formData, setFormData] = useState<Record<string, any>>({});
  const [confirmText, setConfirmText] = useState("");
  const [showConfirm, setShowConfirm] = useState(false);

  const mutation = useExecuteOpsAction(incidentId);
  const action = getActionById(actionType);

  if (!isOpen || !action) return null;

  const handleInputChange = (
    field: string,
    value: any,
    type: string
  ) => {
    setFormData((prev) => ({
      ...prev,
      [field]: type === "checkbox" ? !prev[field] : value,
    }));
  };

  const handleExecute = () => {
    // Validate required fields
    const missingRequired = action.parameters
      .filter((p) => p.required && !formData[p.name])
      .map((p) => p.label);

    if (missingRequired.length > 0) {
      alert(`Missing required fields: ${missingRequired.join(", ")}`);
      return;
    }

    // If action requires confirmation, show confirmation dialog
    if (action.requiresConfirm && !showConfirm) {
      setShowConfirm(true);
      return;
    }

    // Execute action
    const req: ExecuteActionRequest = {
      action_type: actionType,
      parameters: formData,
    };

    mutation.mutate(req, {
      onSuccess: () => {
        setFormData({});
        setShowConfirm(false);
        onSuccess?.();
        onClose();
      },
    });
  };

  if (mutation.isPending) {
    return (
      <div className="modal-overlay">
        <div className="modal-content">
          <Spinner size="lg" />
          <p>Executing action...</p>
        </div>
      </div>
    );
  }

  if (showConfirm) {
    return (
      <div className="modal-overlay">
        <div className="modal-content modal-confirm">
          <div className="modal-header">
            <h3>Confirm Action</h3>
            <button className="modal-close" onClick={onClose}>
              ✕
            </button>
          </div>

          <div className="modal-body">
            <div className="warning-banner">
              <span className="warning-icon">⚠️</span>
              <span>You're about to execute: {action.name}</span>
            </div>

            <p className="confirmation-text">
              This action will be applied to incident {incidentId.substring(0, 8)}.
              This may impact your users.
            </p>

            <div className="confirmation-details">
              {Object.entries(formData).map(([key, value]) => (
                <div key={key} className="detail-row">
                  <span className="detail-key">{key}:</span>
                  <span className="detail-value">{String(value)}</span>
                </div>
              ))}
            </div>

            <div className="confirm-input">
              <label>
                Type <strong>{actionType}</strong> to confirm:
              </label>
              <input
                type="text"
                value={confirmText}
                onChange={(e) => setConfirmText(e.target.value)}
                placeholder={`Type '${actionType}' to confirm`}
                autoFocus
              />
            </div>
          </div>

          <div className="modal-footer">
            <button
              className="btn-secondary"
              onClick={() => setShowConfirm(false)}
            >
              Cancel
            </button>
            <button
              className="btn-danger"
              onClick={handleExecute}
              disabled={confirmText !== actionType}
            >
              Execute {action.name}
            </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="modal-overlay">
      <div className="modal-content">
        <div className="modal-header">
          <h3>{action.name}</h3>
          <button className="modal-close" onClick={onClose}>
            ✕
          </button>
        </div>

        <div className="modal-body">
          <p className="action-description">{action.description}</p>

          <div className="form-fields">
            {action.parameters.map((param) => (
              <div key={param.name} className="form-group">
                <label htmlFor={param.name}>
                  {param.label}
                  {param.required && <span className="required">*</span>}
                </label>

                {param.type === "text" && (
                  <input
                    id={param.name}
                    type="text"
                    placeholder={param.placeholder}
                    value={formData[param.name] || ""}
                    onChange={(e) =>
                      handleInputChange(param.name, e.target.value, "text")
                    }
                  />
                )}

                {param.type === "number" && (
                  <input
                    id={param.name}
                    type="number"
                    placeholder={param.placeholder}
                    value={formData[param.name] || param.defaultValue || ""}
                    onChange={(e) =>
                      handleInputChange(
                        param.name,
                        parseInt(e.target.value),
                        "number"
                      )
                    }
                  />
                )}

                {param.type === "checkbox" && (
                  <label className="checkbox-label">
                    <input
                      id={param.name}
                      type="checkbox"
                      checked={formData[param.name] || false}
                      onChange={() =>
                        handleInputChange(param.name, null, "checkbox")
                      }
                    />
                    <span>{param.label}</span>
                  </label>
                )}

                {param.type === "select" && (
                  <select
                    id={param.name}
                    value={formData[param.name] || param.defaultValue || ""}
                    onChange={(e) =>
                      handleInputChange(param.name, e.target.value, "select")
                    }
                  >
                    <option value="">Select {param.label}</option>
                    {param.options?.map((opt) => (
                      <option key={opt.value} value={opt.value}>
                        {opt.label}
                      </option>
                    ))}
                  </select>
                )}
              </div>
            ))}
          </div>

          {mutation.isError && (
            <div className="error-banner">
              <span className="error-icon">❌</span>
              <span>{mutation.error.message}</span>
            </div>
          )}
        </div>

        <div className="modal-footer">
          <button className="btn-secondary" onClick={onClose}>
            Cancel
          </button>
          <button
            className={`btn-primary btn-action btn-${action.severity}`}
            onClick={handleExecute}
          >
            {action.icon} Execute {action.name}
          </button>
        </div>
      </div>
    </div>
  );
}
