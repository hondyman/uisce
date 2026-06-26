/**
 * Bundle Data Management Hook
 *
 * Custom hook for managing bundle data fetching and state
 */

import { useState, useEffect, useCallback } from 'react';
import { useTenant } from '../../contexts/TenantContext';
import { useAuthFetch } from '../../utils/authFetch';
import { SemanticObjectReference, BundleRowPolicy, BundleColumnPolicy } from '../../types/bundles';
import { generateId } from './bundleConstants';
import { getSelectedRegion } from '../../lib/region';

interface UseBundleDataProps {
  bundleId?: string;
}

export const useBundleData = ({ bundleId }: UseBundleDataProps = {}) => {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [includedMeasures, setIncludedMeasures] = useState<SemanticObjectReference[]>([]);
  const [includedDimensions, setIncludedDimensions] = useState<SemanticObjectReference[]>([]);
  const [rowPolicies, setRowPolicies] = useState<BundleRowPolicy[]>([]);
  const [columnPolicies, setColumnPolicies] = useState<BundleColumnPolicy[]>([]);
  const [loading, setLoading] = useState<boolean>(false);
  const [initializing, setInitializing] = useState<boolean>(!!bundleId);
  const [error, setError] = useState<string | null>(null);

  const { tenant, datasource } = useTenant();
  const { authFetch } = useAuthFetch();

  const tenantId = tenant?.id?.trim() ?? '';
  const datasourceId = (datasource?.id ?? datasource?.alpha_datasource?.datasource_name ?? '').trim();

  // Load bundle data if editing
  useEffect(() => {
    if (!bundleId) return;

    const fetchBundle = async () => {
      setInitializing(true);
      setError(null);

      try {
        const result = await authFetch<any>(`/api/bundles/${bundleId}`, {
          headers: {
            'X-Tenant-ID': tenantId,
            'X-Tenant-Datasource-ID': datasourceId,
            'X-Tenant-Region': getSelectedRegion(),
          },
        });

        if (!result.ok) {
          throw new Error('Failed to fetch bundle');
        }

        const bundle = result.data;
        setName(bundle.name || '');
        setDescription(bundle.description || '');
        setIncludedMeasures(bundle.included_measures || []);
        setIncludedDimensions(bundle.included_dimensions || []);
        setRowPolicies(bundle.row_policies || []);
        setColumnPolicies(bundle.column_policies || []);
      } catch (err: any) {
        setError(err.message || 'Failed to load bundle');
      } finally {
        setInitializing(false);
      }
    };

    fetchBundle();
  }, [bundleId, tenantId, datasourceId, authFetch]);

  const resetForm = useCallback(() => {
    setName('');
    setDescription('');
    setIncludedMeasures([]);
    setIncludedDimensions([]);
    setRowPolicies([]);
    setColumnPolicies([]);
    setError(null);
  }, []);

  // Policy management functions
  const addRowPolicy = useCallback(() => {
    const newPolicy: BundleRowPolicy = {
      id: generateId(),
      name: '',
      description: '',
      member: '',
      operator: '',
      values: [],
      conditions: [],
    };
    setRowPolicies(prev => [...prev, newPolicy]);
  }, []);

  const removeRowPolicy = useCallback((policyId: string) => {
    setRowPolicies(prev => prev.filter(policy => policy.id !== policyId));
  }, []);

  const updateRowPolicy = useCallback((policyId: string, updater: (policy: BundleRowPolicy) => BundleRowPolicy) => {
    setRowPolicies(prev => prev.map(policy =>
      policy.id === policyId ? updater(policy) : policy
    ));
  }, []);

  const handleRowPolicyFieldChange = useCallback((policyId: string, field: string | number | symbol, value: any) => {
    updateRowPolicy(policyId, (current) => ({
      ...current,
      [field]: value,
    }));
  }, [updateRowPolicy]);

  const addColumnPolicy = useCallback(() => {
    const newPolicy: BundleColumnPolicy = {
      id: generateId(),
      name: '',
      description: '',
      columns: [],
      maskType: 'redact',
      maskValue: undefined,
      conditions: [],
    };
    setColumnPolicies(prev => [...prev, newPolicy]);
  }, []);

  const removeColumnPolicy = useCallback((policyId: string) => {
    setColumnPolicies(prev => prev.filter(policy => policy.id !== policyId));
  }, []);

  const updateColumnPolicy = useCallback((policyId: string, updater: (policy: BundleColumnPolicy) => BundleColumnPolicy) => {
    setColumnPolicies(prev => prev.map(policy =>
      policy.id === policyId ? updater(policy) : policy
    ));
  }, []);

  const handleColumnPolicyFieldChange = useCallback((policyId: string, field: string | number | symbol, value: any) => {
    updateColumnPolicy(policyId, (current) => ({
      ...current,
      [field]: value,
    }));
  }, [updateColumnPolicy]);

  const handleColumnPolicyColumnsChange = useCallback((policyId: string, columnsString: string) => {
    const columns = columnsString.split(',').map(col => col.trim()).filter(col => col.length > 0);
    updateColumnPolicy(policyId, (current) => ({
      ...current,
      columns,
    }));
  }, [updateColumnPolicy]);

  const handleRowPolicyValuesChange = useCallback((policyId: string, valuesString: string) => {
    const values = valuesString.split(',').map(val => val.trim()).filter(val => val.length > 0);
    updateRowPolicy(policyId, (current) => ({
      ...current,
      values,
    }));
  }, [updateRowPolicy]);

  const prepareRowPoliciesForSave = useCallback((policies: BundleRowPolicy[]): BundleRowPolicy[] =>
    policies.map((policy) => ({
      ...policy,
      name: policy.name.trim(),
      description: policy.description.trim(),
      member: policy.member.trim(),
      operator: policy.operator.trim() || 'equals',
      values: policy.values.map((v: string) => v.trim()).filter((v: string) => v.length > 0),
      conditions: policy.conditions
        .map((condition) => ({
          attribute: condition.attribute.trim(),
          operator: (condition.operator || 'equals').trim() || 'equals',
          values: condition.values.map((v: string) => v.trim()).filter((v: string) => v.length > 0)
        }))
        .filter((condition) => condition.attribute.length > 0)
    })), []);

  const prepareColumnPoliciesForSave = useCallback((policies: BundleColumnPolicy[]): BundleColumnPolicy[] =>
    policies.map((policy) => ({
      ...policy,
      name: policy.name.trim(),
      description: policy.description.trim(),
      columns: policy.columns.map((c: string) => c.trim()).filter((c: string) => c.length > 0),
      maskType: policy.maskType.trim() || 'redact',
      maskValue: policy.maskValue && policy.maskValue.trim().length > 0 ? policy.maskValue.trim() : undefined,
      conditions: policy.conditions
        .map((condition) => ({
          attribute: condition.attribute.trim(),
          operator: (condition.operator || 'equals').trim() || 'equals',
          values: condition.values.map((v: string) => v.trim()).filter((v: string) => v.length > 0)
        }))
        .filter((condition) => condition.attribute.length > 0)
    })), []);

  return {
    // Form data
    name,
    description,
    includedMeasures,
    includedDimensions,
    rowPolicies,
    columnPolicies,

    // State
    loading,
    initializing,
    error,

    // Actions
    setName,
    setDescription,
    setIncludedMeasures,
    setIncludedDimensions,
    setRowPolicies,
    setColumnPolicies,
    setLoading,
    setError,
    resetForm,

    // Policy management
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

    // Policy sanitization for save
    prepareRowPoliciesForSave,
    prepareColumnPoliciesForSave,

    // Context
    tenantId,
    datasourceId,
  };
};