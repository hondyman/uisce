// frontend/src/components/editor/EditorHeader.tsx
import React, { useRef, useState, useCallback } from 'react';
import { useNotification } from '../../hooks/useNotification';
import { devError } from '../../utils/devLogger';
import { AiActions } from './AiActions';
import { validateBeforePublish, logInteraction } from '../../lib/analytics';
import { runAllA11yChecks } from '../../lib/a11yCheck';
import styles from './EditorHeader.module.css';

export interface EditorHeaderProps {
  primaryBO: string;
  tenantId: string;
  userId?: string;
  layoutName?: string;
  onApplyLayout?: (layout: unknown, draftId?: string) => void;
  onPublish: () => Promise<void>;
  onSave?: () => Promise<void>;
  isSaving?: boolean;
  isPublishing?: boolean;
}

interface PublishCheckError {
  reasons?: string[];
  warnings?: string[];
}

/**
 * EditorHeader: Complete editor header with AI actions, save/publish workflow,
 * and pre-publication governance checks.
 *
 * Features:
 * - AI prompt input for layout generation
 * - Field recommendations
 * - Save and Publish buttons
 * - Pre-publish validation (a11y + performance)
 * - Governance error messaging
 * - Analytics event logging
 */
export const EditorHeader: React.FC<EditorHeaderProps> = ({
  primaryBO,
  tenantId,
  userId,
  layoutName,
  onApplyLayout,
  onPublish,
  onSave,
  isSaving = false,
  isPublishing = false,
}) => {
  const [showPublishConfirm, setShowPublishConfirm] = useState(false);
  const [publishChecking, setPublishChecking] = useState(false);
  const [publishErrors, setPublishErrors] = useState<PublishCheckError | null>(null);
  const publishingRef = useRef(false);
  const notification = useNotification();

  /**
   * handleSave: Save layout without publishing.
   * Logs save event to analytics.
   */
  const handleSave = useCallback(async () => {
    try {
      logInteraction('layout_save', {
        primaryBO,
        layoutName,
        userId,
      });
      await onSave?.();
    } catch (err) {
      devError('Save failed:', err);
      const notification = useNotification();
      notification.error(`Save failed: ${err instanceof Error ? err.message : 'Unknown error'}`);
    }
  }, [primaryBO, layoutName, userId, onSave]);

  /**
   * handlePublish: Trigger governance validation before publish.
   * Shows confirmation dialog with errors if blocked.
   */
  const handlePublishClick = useCallback(async () => {
    if (publishingRef.current) return;

    publishingRef.current = true;
    setPublishChecking(true);
    setPublishErrors(null);

    try {
      // Run accessibility checks
      const a11yCheck = runAllA11yChecks();

      // Log the check attempt
      logInteraction('publish_validate_attempt', {
        primaryBO,
        a11yOk: a11yCheck.ok,
        a11yIssues: a11yCheck.issues,
        userId,
      });

      // Call backend governance gate
      const result = await validateBeforePublish({
        accessibilityOk: a11yCheck.ok,
        performanceOk: true, // TODO: integrate real perf budget check
      });

      if (!result.allowed) {
        setPublishErrors({
          reasons: result.reasons || ['Publish validation failed'],
          warnings: result.warnings,
        });
        publishingRef.current = false;
        setPublishChecking(false);
        return;
      }

      // Validation passed - show confirmation
      setShowPublishConfirm(true);
    } catch (err) {
      const reasons = [];
      const warnings = [];

      if (err instanceof Error) {
        reasons.push(err.message);
      } else {
        reasons.push('Validation failed');
      }

      // Parse a11y issues into reasons
      const a11yCheck = runAllA11yChecks();
      if (!a11yCheck.ok) {
        reasons.push(...a11yCheck.issues);
      }
      if (a11yCheck.warnings?.length) {
        warnings.push(...a11yCheck.warnings);
      }

      setPublishErrors({ reasons, warnings });
      publishingRef.current = false;
      setPublishChecking(false);
    } finally {
      setPublishChecking(false);
    }
  }, [primaryBO, userId]);

  /**
   * handleConfirmPublish: Actually publish after confirmation.
   */
  const handleConfirmPublish = useCallback(async () => {
    if (publishingRef.current) return;

    publishingRef.current = true;
    setShowPublishConfirm(false);

    try {
      logInteraction('layout_publish_confirmed', {
        primaryBO,
        layoutName,
        userId,
      });

      await onPublish();

      logInteraction('layout_publish_success', {
        primaryBO,
        layoutName,
        userId,
      });

      const notification = useNotification();
      notification.success('Published successfully!');
    } catch (err) {
      logInteraction('layout_publish_failed', {
        primaryBO,
        layoutName,
        userId,
        error: err instanceof Error ? err.message : String(err),
      });

      const notification = useNotification();
      notification.error(`Publish failed: ${err instanceof Error ? err.message : 'Unknown error'}`);
    } finally {
      publishingRef.current = false;
    }
  }, [primaryBO, layoutName, userId, onPublish]);

  const handleCancel = () => {
    setShowPublishConfirm(false);
    publishingRef.current = false;
  };

  return (
    <div className={styles.editorHeader}>
      <div className={styles.topBar}>
        <div className={styles.titleArea}>
          <h1 className={styles.title}>{layoutName || 'Untitled Layout'}</h1>
          {primaryBO && <span className={styles.boTag}>{primaryBO}</span>}
        </div>

        <div className={styles.actions}>
          <button
            onClick={handleSave}
            disabled={isSaving}
            className={styles.saveButton}
            title="Save draft (Ctrl+S)"
          >
            {isSaving ? 'Saving...' : 'Save'}
          </button>
          <button
            onClick={handlePublishClick}
            disabled={isPublishing || publishChecking}
            className={styles.publishButton}
            title="Publish to production"
          >
            {publishChecking ? 'Checking...' : isPublishing ? 'Publishing...' : 'Publish'}
          </button>
        </div>
      </div>

      {/* AI Actions */}
      {onApplyLayout && (
        <div className={styles.aiActionsSection}>
          <AiActions
            primaryBO={primaryBO}
            tenantId={tenantId}
            onApplyLayout={onApplyLayout}
          />
        </div>
      )}

      {/* Publish Validation Errors */}
      {publishErrors && (
        <div className={styles.governanceWarning}>
          <div className={styles.warningTitle}>⚠️ Publish Blocked</div>
          {publishErrors.reasons && publishErrors.reasons.length > 0 && (
            <div className={styles.reasons}>
              <strong>Errors:</strong>
              <ul>
                {publishErrors.reasons.map((reason, idx) => (
                  <li key={idx}>{reason}</li>
                ))}
              </ul>
            </div>
          )}
          {publishErrors.warnings && publishErrors.warnings.length > 0 && (
            <div className={styles.warnings}>
              <strong>Warnings:</strong>
              <ul>
                {publishErrors.warnings.map((warning, idx) => (
                  <li key={idx}>{warning}</li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}

      {/* Publish Confirmation Dialog */}
      {showPublishConfirm && (
        <div className={styles.dialogOverlay} onClick={handleCancel}>
          <div
            className={styles.dialogContent}
            onClick={(e) => e.stopPropagation()}
            role="dialog"
            aria-modal="true"
            aria-labelledby="publish-confirm-title"
            tabIndex={-1}
          >
            <h2 id="publish-confirm-title" className={styles.dialogTitle}>
              Confirm Publish
            </h2>
            <p className={styles.dialogText}>
              Are you ready to publish "{layoutName || 'this layout'}" for{' '}
              <strong>{primaryBO}</strong>?
            </p>
            <p className={styles.dialogSubtext}>
              This will make the layout live for all users of this business object.
            </p>
            <div className={styles.dialogActions}>
              <button
                onClick={handleCancel}
                className={styles.cancelButton}
                disabled={isPublishing}
              >
                Cancel
              </button>
              <button
                onClick={handleConfirmPublish}
                className={styles.confirmButton}
                disabled={isPublishing}
              >
                {isPublishing ? 'Publishing...' : 'Confirm & Publish'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default EditorHeader;
