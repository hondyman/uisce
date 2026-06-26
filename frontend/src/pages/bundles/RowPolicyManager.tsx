import type { FC } from 'react';
import {
    Card,
    CardHeader,
    CardContent,
    Button,
    Box,
    TextField,
    Alert,
    Divider,
} from '@mui/material';
import { AddCircleOutline as AddCircleOutlineIcon } from '@mui/icons-material';
import type { BundleRowPolicy as BundleRowPolicyType } from '../../types/bundles';
import { AttributeConditionsEditor, OperatorOption } from '../../components/AttributeConditionsEditor';

interface RowPolicyManagerProps {
    policies: BundleRowPolicyType[];
    onAddPolicy: () => void;
    onRemovePolicy: (policyId: string) => void;
    onUpdatePolicy: (policyId: string, updater: (policy: BundleRowPolicyType) => BundleRowPolicyType) => void;
    onFieldChange: (policyId: string, field: string | number | symbol, value: any) => void;
    onValuesChange: (policyId: string, values: string) => void;
    fieldErrors: Record<string, string[]>;
    fieldHelperText: (fieldPath: string) => string | undefined;
    operatorOptions: OperatorOption[];
    onFieldEdit: (fieldPath: string) => void;
    hasFieldError: (fieldPath: string) => boolean;
}

export const RowPolicyManager: FC<RowPolicyManagerProps> = ({
    policies,
    onAddPolicy,
    onRemovePolicy,
    onUpdatePolicy,
    onFieldChange,
    operatorOptions,
    fieldErrors,
    onFieldEdit,
    fieldHelperText,
    hasFieldError,
}) => {
    return (
        <Card>
            <CardHeader
                title="Row-Level Policies"
                action={
                    <Button
                        size="small"
                        startIcon={<AddCircleOutlineIcon />}
                        onClick={onAddPolicy}
                    >
                        Add Row Policy
                    </Button>
                }
            />
            <Divider />
            <CardContent>
                {policies.length === 0 ? (
                    <Alert severity="info">No row policies defined yet.</Alert>
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
                            <TextField
                                label="Policy Name"
                                value={policy.name}
                                onChange={(e) => onFieldChange(policy.id, 'name', e.target.value)}
                                fullWidth
                                sx={{ mb: 2 }}
                                                    error={hasFieldError(`rowPolicies[${index}].name`)}
                                                    helperText={fieldHelperText(`rowPolicies[${index}].name`)}
                            />

                            <TextField
                                label="Description"
                                value={policy.description}
                                onChange={(e) => onFieldChange(policy.id, 'description', e.target.value)}
                                fullWidth
                                sx={{ mb: 2 }}
                            />

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
                                basePath={`rowPolicies[${index}]`}
                                fieldErrors={fieldErrors}
                                onFieldEdit={onFieldEdit}
                            />

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