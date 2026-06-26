import React, { useState, useMemo, useCallback } from 'react';
import {
	Container,
	Typography,
	CircularProgress,
	Alert,
} from '@mui/material';
import { useTenant } from '../../contexts/TenantContext';
import { useAuthFetch } from '../../utils/authFetch';
import {
	DataBundle,
	SemanticObjectReference
} from '../../types/bundles';
import { useValidationErrors } from '../../hooks/useValidationErrors';
import { logInteraction } from '../../lib/analytics';
import { BundleMetadataForm } from './BundleMetadataForm';
import { SemanticObjectsSelector } from './SemanticObjectsSelector';
import { RowPolicyManager } from './RowPolicyManager';
import { ColumnPolicyManager } from './ColumnPolicyManager';
import { BundleActions } from './BundleActions';
import { useBundleData } from './useBundleData';
import { useSemanticObjects } from './useSemanticObjects';
import { operatorOptions, maskTypeOptions } from './bundleConstants';

interface BundleEditorProps {
    bundleId?: string;
    onSave: (bundle: DataBundle) => void;
    onCancel: () => void;
}

const BundleEditor: React.FC<BundleEditorProps> = ({ bundleId, onSave, onCancel }) => {
    const isEditMode = !!bundleId;

    // Use custom hooks for state management
    const bundleData = useBundleData({ bundleId });
    const semanticObjects = useSemanticObjects();
    const {
        allObjects,
        loadingObjects,
        objectsError,
        handleRefreshObjects,
    } = semanticObjects;

    const {
        name,
        description,
        includedMeasures,
        includedDimensions,
        rowPolicies,
        columnPolicies,
        loading,
        initializing,
        error,
        setName,
        setDescription,
        setIncludedMeasures,
        setIncludedDimensions,
        setRowPolicies,
        setColumnPolicies,
        setLoading,
        setError,
        addRowPolicy,
        removeRowPolicy,
        updateRowPolicy,
        handleRowPolicyFieldChange,
        handleRowPolicyValuesChange,
        addColumnPolicy,
        removeColumnPolicy,
        updateColumnPolicy,
        handleColumnPolicyFieldChange,
        handleColumnPolicyColumnsChange,
        prepareRowPoliciesForSave,
        prepareColumnPoliciesForSave,
        tenantId,
        datasourceId,
    } = bundleData;

    const {
        fieldErrors,
        clearFieldErrors,
        clearFieldError,
        hasFieldError,
        fieldHelperText,
        handleResponseError
    } = useValidationErrors();

    // UX Enhancements: Publish validation state
    // some publish helpers are not used in this pass; underscore them to quiet lint
    const [_showPublishConfirm, _setShowPublishConfirm] = useState(false);
    const [_publishChecking, _setPublishChecking] = useState(false);
    const [publishErrors, _setPublishErrors] = useState<string[]>([]);

    const { tenant, datasource } = useTenant();
    const { authFetch } = useAuthFetch();
    const selectionMissing = !tenantId || !datasourceId;
    const tenantLabel = tenant?.display_name?.trim() || tenant?.name?.trim() || (tenantId ? tenantId : 'No tenant selected');
    const datasourceLabel =
        datasource?.source_name?.trim() ||
        datasource?.alpha_datasource?.datasource_name?.trim() ||
        (datasourceId ? datasourceId : 'No datasource selected');

    const objectKey = useCallback((obj: SemanticObjectReference) => `${obj.modelId ?? ''}::${obj.id}::${obj.type ?? 'dimension'}`, []);

    const normalizedType = (obj: SemanticObjectReference) => (obj.type === 'measure' ? 'measure' : 'dimension');

    const handleSave = async () => {
        setLoading(true);
        setError(null);
        clearFieldErrors();

        const sanitizedRowPolicies = prepareRowPoliciesForSave(rowPolicies);
        const sanitizedColumnPolicies = prepareColumnPoliciesForSave(columnPolicies);

        try {
            // UX Enhancement: Log interaction
            logInteraction('bundle_save_started', {
                bundleId: bundleId || 'new',
                measuresCount: includedMeasures.length,
                dimensionsCount: includedDimensions.length,
                timestamp: Date.now()
            });

            let currentBundle: DataBundle | null = null;

            if (isEditMode && bundleId) {
                const updateResponse = await authFetch<DataBundle>(`/api/bundles/${bundleId}`, {
                    method: 'PUT',
                    json: { measures: includedMeasures, dimensions: includedDimensions }
                });

                if (!updateResponse.ok || !updateResponse.data) {
                    await handleResponseError(updateResponse.response, 'Failed to save bundle structure');
                }

                currentBundle = updateResponse.data as DataBundle;
            } else {
                const createResponse = await authFetch<DataBundle>('/api/bundles', {
                    method: 'POST',
                    json: { name, description }
                });

                if (!createResponse.ok || !createResponse.data) {
                    await handleResponseError(createResponse.response, 'Failed to create bundle');
                }

                const createdBundle = createResponse.data as DataBundle;
                currentBundle = createdBundle;

                const structureResponse = await authFetch<DataBundle>(`/api/bundles/${createdBundle.id}`, {
                    method: 'PUT',
                    json: { measures: includedMeasures, dimensions: includedDimensions }
                });

                if (!structureResponse.ok || !structureResponse.data) {
                    await handleResponseError(structureResponse.response, 'Failed to update bundle structure');
                }

                currentBundle = structureResponse.data as DataBundle;
            }

            if (!currentBundle) {
                throw new Error('Bundle could not be saved');
            }

            const policyResponse = await authFetch<DataBundle>(`/api/bundles/${currentBundle.id}/policies`, {
                method: 'PUT',
                json: {
                    rowPolicies: sanitizedRowPolicies,
                    columnPolicies: sanitizedColumnPolicies
                }
            });

            if (!policyResponse.ok || !policyResponse.data) {
                await handleResponseError(policyResponse.response, 'Failed to save bundle policies');
            }

            const updatedBundle = policyResponse.data as DataBundle;
            
            // UX Enhancement: Log successful save
            logInteraction('bundle_save_completed', {
                bundleId: updatedBundle.id,
                timestamp: Date.now()
            });
            
            onSave(updatedBundle);
        } catch (err: any) {
            // UX Enhancement: Log save error
            logInteraction('bundle_save_failed', {
                error: err.message,
                timestamp: Date.now()
            });
            setError(err.message || 'Failed to save bundle');
        } finally {
            setLoading(false);
        }
    };

    const scopeDescription = useMemo(() => {
        if (selectionMissing) {
            return 'Select a tenant and datasource to browse views.';
        }
        return `Scoped to ${tenantLabel} • ${datasourceLabel}`;
    }, [datasourceLabel, selectionMissing, tenantLabel]);

    if (initializing) {
        return <CircularProgress />;
    }

    return (
        <Container maxWidth="xl" sx={{ mt: 4 }}>
            <Typography variant="h4" gutterBottom>{isEditMode ? 'Edit Data Bundle' : 'Create Data Bundle'}</Typography>
            <BundleMetadataForm
                name={name}
                description={description}
                onNameChange={setName}
                onDescriptionChange={setDescription}
                fieldErrors={fieldErrors}
                fieldHelperText={fieldHelperText}
            />

            {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

            <SemanticObjectsSelector
                includedMeasures={includedMeasures}
                includedDimensions={includedDimensions}
                allObjects={allObjects}
                loadingObjects={loadingObjects}
                objectsError={objectsError}
                onMeasuresChange={setIncludedMeasures}
                onDimensionsChange={setIncludedDimensions}
                onRefreshObjects={handleRefreshObjects}
            />

            <RowPolicyManager
                policies={rowPolicies}
                onAddPolicy={addRowPolicy}
                onRemovePolicy={removeRowPolicy}
                onUpdatePolicy={updateRowPolicy}
                onFieldChange={handleRowPolicyFieldChange}
                onValuesChange={handleRowPolicyValuesChange}
                fieldErrors={fieldErrors}
                fieldHelperText={fieldHelperText}
                operatorOptions={operatorOptions}
                onFieldEdit={clearFieldError}
                hasFieldError={hasFieldError}
            />

            <ColumnPolicyManager
                policies={columnPolicies}
                onAddPolicy={addColumnPolicy}
                onRemovePolicy={removeColumnPolicy}
                onUpdatePolicy={updateColumnPolicy}
                onFieldChange={handleColumnPolicyFieldChange}
                onColumnsChange={handleColumnPolicyColumnsChange}
                fieldErrors={fieldErrors}
                fieldHelperText={fieldHelperText}
                maskTypeOptions={maskTypeOptions}
                onFieldEdit={clearFieldError}
                hasFieldError={hasFieldError}
                operatorOptions={operatorOptions}
            />

            <BundleActions
                publishErrors={publishErrors}
                loading={loading}
                onCancel={onCancel}
                onSave={handleSave}
            />
        </Container>
    );
};

export default BundleEditor;
