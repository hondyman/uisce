// Wealth Management Service
// Unified entry point for all wealth management functionality

// Frontend Components
export { default as UMAAlpha } from './frontend/UMAAlpha';
export { default as AttributionAlpha } from './frontend/AttributionAlpha';
export { default as TaxHarvest } from './frontend/TaxHarvest';
export { default as IndexAlpha } from './frontend/IndexAlpha';

// Workflows
export { UMAAlpha as UMAAlphaWorkflow } from './workflows/uma_alpha';
export { IndexAlpha as IndexAlphaWorkflow } from './workflows/index_alpha';
export { TaxHarvest as TaxHarvestWorkflow } from './workflows/tax_harvest';
export { AttributionAlpha as AttributionAlphaWorkflow } from './workflows/attribution_alpha';

// Activities
export { AITaxHarvest } from './activities/ai_tax';
export { AIIndexOptimize } from './activities/ai_index';
export { AIAttribution } from './activities/ai_attribution';

// API Routes
export * from './api/routes';

// Types
export * from './domain/types';