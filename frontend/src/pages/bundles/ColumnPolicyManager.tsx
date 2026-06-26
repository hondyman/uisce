import type React from 'react';
import {
    Card,
    CardHeader,
    CardContent,
    Button,
    Box,
    TextField,
    Alert,
    Divider,
    Grid,
    MenuItem,
} from '@mui/material';
import { AddCircleOutline as AddCircleOutlineIcon } from '@mui/icons-material';
import { BundleColumnPolicy } from '../../types/bundles';
import { AttributeConditionsEditor, OperatorOption } from '../../components/AttributeConditionsEditor';
import { MaskTypeOption } from '../../../constants/bundleConstants';

interface MaskTypeOption {
    value: string;
    label: string;
}

interface ColumnPolicyManagerProps {
    policies: BundleColumnPolicy[];
    onAddPolicy: () => void;
    onRemovePolicy: (policyId: string) => void;
    onUpdatePolicy: (policyId: string, updater: (policy: BundleColumnPolicy) => BundleColumnPolicy) => void;
    onFieldChange: (policyId: string, field: string | number | symbol, value: any) => void;
    onColumnsChange: (policyId: string, columns: string) => void;
    fieldErrors: Record<string, string[]>;
    fieldHelperText: (fieldPath: string) => string | undefined;
    maskTypeOptions: MaskTypeOption[];
    onFieldEdit: (fieldPath: string) => void;
    hasFieldError: (fieldPath: string) => boolean;
    operatorOptions: OperatorOption[];
}

export const ColumnPolicyManager: React.FC<ColumnPolicyManagerProps> = ({
    policies,
    onAddPolicy,
    onRemovePolicy,
    onUpdatePolicy,
    onFieldChange,
    onColumnsChange,
    maskTypeOptions,
    fieldErrors,
    onFieldEdit,
    fieldHelperText,
    hasFieldError,
    operatorOptions,
}) => {
    return (
        <Card>
            <CardHeader
                title="Column-Level Policies"
                action={
                    <Button
                        size="small"
                        startIcon={<AddCircleOutlineIcon />}
                        onClick={onAddPolicy}
                    >
                        Add Column Policy
                    </Button>
                }
            />
            <Divider />
            <CardContent>
                {policies.length === 0 ? (
                    <Alert severity="info">No column policies defined yet.</Alert>
                ) : (
                    policies.map((policy, index) => (
                        <Box
                            key={policy.id}
                            sx={{
                                border: '1px solid',
                                borderColor: 'divider',
                                borderRadius: 2,
                                p: 2,
                                mb: 2
                            }}
                        >
                            <Grid container spacing={2}>
                                <Grid item xs={12} md={6}>
                                    <TextField
                                        label="Policy Name"
                                        value={policy.name}
                                        onChange={(e) => onFieldChange(policy.id, 'name', e.target.value)}
                                        fullWidth
                                        error={hasFieldError(`columnPolicies[${index}].name`)}
                                        helperText={fieldHelperText(`columnPolicies[${index}].name`)}
                                    />
                                </Grid>
                                <Grid item xs={12} md={6}>
                                    <TextField
                                        select
                                        label="Mask Type"
                                        value={policy.maskType}
                                        onChange={(e) => onFieldChange(policy.id, 'maskType', e.target.value)}
                                        fullWidth
                                        error={hasFieldError(`columnPolicies[${index}].maskType`)}
                                        helperText={fieldHelperText(`columnPolicies[${index}].maskType`)}
                                    >
                                        {maskTypeOptions.map((option) => (
                                            <MenuItem key={option.value} value={option.value}>
                                                {option.label}
                                            </MenuItem>
                                        ))}
                                    </TextField>
                                </Grid>
                                <Grid item xs={12}>
                                    <TextField
                                        label="Description"
                                        value={policy.description}
                                        onChange={(e) => onFieldChange(policy.id, 'description', e.target.value)}
                                        fullWidth
                                    />
                                </Grid>
                                <Grid item xs={12} md={6}>
                                    <TextField
                                        label="Columns (comma separated)"
                                        value={policy.columns.join(', ')}
                                        onChange={(e) => onColumnsChange(policy.id, e.target.value)}
                                        fullWidth
                                        placeholder="e.g., ssn,last_name"
                                        error={hasFieldError(`columnPolicies[${index}].columns`)}
                                        helperText={fieldHelperText(`columnPolicies[${index}].columns`)}
                                    />
                                </Grid>
                                <Grid item xs={12} md={6}>
                                    <TextField
                                        label="Mask Value"
                                        value={policy.maskValue ?? ''}
                                        onChange={(e) => onFieldChange(policy.id, 'maskValue', e.target.value)}
                                        fullWidth
                                        placeholder="Optional replacement value"
                                        error={hasFieldError(`columnPolicies[${index}].maskValue`)}
                                        helperText={fieldHelperText(`columnPolicies[${index}].maskValue`)}
                                    />
                                </Grid>
                            </Grid>

                            <Box sx={{ mt: 2 }}>
                                <AttributeConditionsEditor
                                    conditions={policy.conditions}
                                    onChange={(nextConditions: any) =>
                                        onUpdatePolicy(policy.id, (current) => ({
                                            ...current,
                                            conditions: nextConditions
                                        }))
                                    }
                                    operatorOptions={operatorOptions}
                                    emptyHelperText="No attribute conditions. Add one to scope this policy."
                                    basePath={`columnPolicies[${index}]`}
                                    fieldErrors={fieldErrors}
                                    onFieldEdit={onFieldEdit}
                                />
                            </Box>

                            <Box sx={{ mt: 2, display: 'flex', justifyContent: 'flex-end' }}>
                                <Button
                                    size="small"
                                    color="error"
                                    onClick={() => onRemovePolicy(policy.id)}
                                >
                                    Remove Policy
                                </Button>
                            </Box>
                        </Box>
                    ))
                )}
            </CardContent>
        </Card>
    );
};