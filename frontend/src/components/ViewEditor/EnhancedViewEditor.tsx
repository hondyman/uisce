import React, { useState, useEffect } from 'react';
import { Box, CircularProgress, Alert } from '@mui/material';
import ViewEditor from './ViewEditor';
import useViewValidation from '../../hooks/useViewValidation';
import { isInvalidViewName } from '../../utils/viewNameValidation';

interface EnhancedViewEditorProps {
  viewName?: string;
  viewId?: string;
  tenantId?: string;
  datasourceId?: string;
  onViewSaved?: (viewIdentifier: string, viewData: any) => void;
}

const EnhancedViewEditor: React.FC<EnhancedViewEditorProps> = ({
  viewName,
  viewId,
  tenantId,
  datasourceId,
  onViewSaved
}) => {
  const [viewData, setViewData] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isSaving, setIsSaving] = useState(false);
  const [isValidating, setIsValidating] = useState(false);
  const [validationResult, setValidationResult] = useState<any>(null);
  const [notification, setNotification] = useState({ open: false, message: '', severity: 'info' as 'info' | 'success' | 'warning' | 'error' });

  const { validateView, saveView, loadView } = useViewValidation({
    tenantId,
    datasourceId
  });

  // Determine which identifier to use (prefer viewId over viewName)
  const viewIdentifier = viewId || viewName;

  // Load view data on mount or when viewIdentifier changes
  useEffect(() => {
    const loadViewData = async () => {
      setLoading(true);
      setError(null);
      
      try {
        const data = await loadView(viewIdentifier!);
        // Handle both direct view data and wrapped view data
        const viewData = data.view || data;
        setViewData(viewData);
      } catch (err) {
        setError(`Failed to load view: ${err}`);
      } finally {
        setLoading(false);
      }
    };

    // Validate viewIdentifier before making API calls
    if (viewIdentifier && viewIdentifier.trim()) {
      // For viewName, check if it's invalid; for viewId (UUID), skip name validation
      if (viewName && !viewId && isInvalidViewName(viewName)) {
        // Handle invalid view names (only when using viewName, not viewId)
        setError(`Invalid view name: "${viewName}". Please provide a valid view name.`);
        setLoading(false);
      } else {
        loadViewData();
      }
    } else {
      // Empty or whitespace-only identifier
      setLoading(false);
    }
  }, [viewIdentifier, loadView]);

  // Handle save with validation
  const handleSave = async () => {
    if (!tenantId || !datasourceId) {
      setNotification({ open: true, message: 'Select a tenant and datasource before saving this view.', severity: 'warning' });
      return;
    }
    setIsSaving(true);
    try {
      const normalized = { ...viewData, name: viewIdentifier, title: viewData?.title || viewIdentifier };
      setViewData(normalized);
      const resp = await saveView(viewIdentifier!, normalized);
      // If server returned an id, prefer it as canonical identifier
      const canonicalId = resp?.id || resp?.view?.id || resp?.view?.uuid || resp?.id;
      onViewSaved?.(canonicalId || viewIdentifier!, resp?.view || normalized);
      setNotification({ open: true, message: 'View saved successfully!', severity: 'success' });
    } catch (err) {
      setNotification({ open: true, message: `Failed to save view: ${err}`, severity: 'error' });
    } finally {
      setIsSaving(false);
    }
  };

  // Handle validation
  const handleValidate = async () => {
    setIsValidating(true);
    try {
      const result = await validateView(viewIdentifier!, viewData);
      setValidationResult(result);
      if (result.valid) {
        setNotification({ open: true, message: 'View is valid!', severity: 'success' });
      } else {
        setNotification({ open: true, message: `Validation issues found: ${result.issues?.length || 0}`, severity: 'warning' });
      }
    } catch (err) {
      setNotification({ open: true, message: `Validation failed: ${err}`, severity: 'error' });
    } finally {
      setIsValidating(false);
    }
  };

  if (loading) {
    return (
      <Box sx={{ 
        display: 'flex', 
        justifyContent: 'center', 
        alignItems: 'center', 
        height: '100%',
        minHeight: 400 
      }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Box sx={{ p: 2 }}>
        <Alert severity="error">
          {error}
        </Alert>
      </Box>
    );
  }

  if (!viewData) {
    return (
      <Box sx={{ p: 2 }}>
        <Alert severity="warning">
          No view data available.
        </Alert>
      </Box>
    );
  }

  return (
    <ViewEditor
      viewName={viewIdentifier!}
      viewData={viewData}
      setViewData={setViewData}
      onSave={handleSave}
      onValidate={handleValidate}
      isSaving={isSaving}
      isValidating={isValidating}
      validationResult={validationResult}
      notification={notification}
      setNotification={setNotification}
      tenantId={tenantId}
      datasourceId={datasourceId}
    />
  );
};

export default EnhancedViewEditor;
