import { useState, useCallback } from 'react';
import { type BundleRowPolicy, type BundleColumnPolicy, type AttributeCondition } from '../../types/bundles';
import { generateId } from './bundleConstants';

export const useBundlePolicies = () => {
    const [rowPolicies, setRowPolicies] = useState<BundleRowPolicy[]>([]);
    const [columnPolicies, setColumnPolicies] = useState<BundleColumnPolicy[]>([]);

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

    const handleRowPolicyFieldChange = useCallback((policyId: string, field: keyof BundleRowPolicy, value: any) => {
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

    const handleColumnPolicyFieldChange = useCallback((policyId: string, field: keyof BundleColumnPolicy, value: any) => {
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

    const prepareRowPoliciesForSave = useCallback((policies: BundleRowPolicy[]): BundleRowPolicy[] =>
        policies.map((policy) => ({
            ...policy,
            name: policy.name.trim(),
            description: policy.description.trim(),
            member: policy.member.trim(),
            operator: policy.operator.trim() || 'equals',
            values: policy.values.map(v => v.trim()).filter(v => v.length > 0),
            conditions: policy.conditions
                .map((condition) => ({
                    attribute: condition.attribute.trim(),
                    operator: (condition.operator || 'equals').trim() || 'equals',
                    values: condition.values.map(v => v.trim()).filter(v => v.length > 0)
                }))
                .filter((condition) => condition.attribute.length > 0)
        })), []);

    const prepareColumnPoliciesForSave = useCallback((policies: BundleColumnPolicy[]): BundleColumnPolicy[] =>
        policies.map((policy) => ({
            ...policy,
            name: policy.name.trim(),
            description: policy.description.trim(),
            columns: policy.columns.map(c => c.trim()).filter(c => c.length > 0),
            maskType: policy.maskType.trim() || 'redact',
            maskValue: policy.maskValue && policy.maskValue.trim().length > 0 ? policy.maskValue.trim() : undefined,
            conditions: policy.conditions
                .map((condition) => ({
                    attribute: condition.attribute.trim(),
                    operator: (condition.operator || 'equals').trim() || 'equals',
                    values: condition.values.map(v => v.trim()).filter(v => v.length > 0)
                }))
                .filter((condition) => condition.attribute.length > 0)
        })), []);

    return {
        rowPolicies,
        columnPolicies,
        setRowPolicies,
        setColumnPolicies,
        addRowPolicy,
        removeRowPolicy,
        updateRowPolicy,
        handleRowPolicyFieldChange,
        addColumnPolicy,
        removeColumnPolicy,
        updateColumnPolicy,
        handleColumnPolicyFieldChange,
        handleColumnPolicyColumnsChange,
        prepareRowPoliciesForSave,
        prepareColumnPoliciesForSave,
    };
};