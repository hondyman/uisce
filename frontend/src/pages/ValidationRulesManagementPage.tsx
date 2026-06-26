// This page intentionally re-uses the catalog validation rules implementation.
// The original file became concatenated and caused build errors; to restore a
// single canonical implementation we import and re-export the catalog page.

import React, { useState, useEffect } from 'react';
import {
	Container,
	Typography,
	Alert,
	Box,
    Stack
} from '@mui/material';
import WarningIcon from '@mui/icons-material/Warning';
import { useTenant } from '../contexts/TenantContext';
import ValidationRulesWithFacets from '../components/ValidationRules/ValidationRulesWithFacets';
import { fetchEntitySchema } from '../api/entitySchema';
import { devError } from '../utils/devLogger';

export const ValidationRulesManagementPage = () => {
	const { tenant, datasource, isSelected } = useTenant();
  
	const [entities, setEntities] = useState([] as string[]);
	const [entitySchema, setEntitySchema] = useState({} as any);

	// Fetch entities from backend
	const fetchEntities = async () => {
		if (!isSelected || !tenant?.id || !datasource?.id) {
			setEntities([]);
			setEntitySchema({});
			return;
		}

		try {
			const schema = await fetchEntitySchema(tenant.id, datasource.id);
			const entityNames = Object.keys(schema).sort();
			setEntities(entityNames);
			setEntitySchema(schema);
		} catch (error) {
			devError('Error fetching entities:', error);
			setEntities([]);
			setEntitySchema({});
		}
	};

	// Load entities when tenant/datasource selected
	useEffect(() => {
		fetchEntities();
	}, [isSelected, tenant?.id, datasource?.id]);

	return (
		<Container maxWidth="xl" sx={{ py: 3, height: 'calc(100vh - 64px)', display: 'flex', flexDirection: 'column' }}>
			{/* Tenant Scope Alert */}
			{!isSelected && (
				<Alert severity="warning" sx={{ mb: 3 }} icon={<WarningIcon />}>
					<Typography variant="body2" sx={{ fontWeight: 600 }}>
						⚠️ No Tenant Selected
					</Typography>
					<Typography variant="body2" sx={{ mt: 0.5 }}>
						Please select a tenant and datasource from the picker to create or manage validation rules.
					</Typography>
				</Alert>
			)}

            {/* Header */}
			<Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 3 }}>
				<Box>
					<Typography variant="h4" sx={{ fontWeight: 600 }}>
						✓ Validation Rules
					</Typography>
					<Typography variant="body2" sx={{ color: 'text.secondary', mt: 0.5 }}>
						{isSelected
							? `Define business logic and data quality rules for your entities (${tenant?.display_name})`
							: 'Define business logic and data quality rules for your entities'}
					</Typography>
				</Box>
			</Stack>

			{/* Check isSelected again for the main content */}
            {!isSelected ? (
                <Box sx={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'text.secondary' }}>
                   Select a tenant to view rules.
                </Box>
            ) : (
				// Use the richer rules component with facets
                // We pass key={...} to force re-mounting if tenant/datasource changes, ensuring clean state
                <Box sx={{ flex: 1, minHeight: 0 }}>
				    <ValidationRulesWithFacets 
                        key={`${tenant?.id}-${datasource?.id}`}
                        tenantId={tenant!.id} 
                        datasourceId={datasource!.id} 
                        entities={entities} 
                        entitySchema={entitySchema} 
                    />
                </Box>
			)}
		</Container>
	);
};

export default ValidationRulesManagementPage;