// BOGovernanceStudio Feature Entry Point
// Import BOGovernanceStudio to embed the 6-tab control plane anywhere in the app.

export { default as BOGovernanceStudio } from './BOGovernanceStudio';
export { default as ValidationRuleBuilder } from './ValidationRuleBuilder';
export { default as PolicyRuleBuilder } from './PolicyRuleBuilder';
export { default as AccessControlMatrix } from './AccessControlMatrix';
export { default as FieldSecurityConfigurator } from './FieldSecurityConfigurator';
export { default as BOAuditTimeline } from './BOAuditTimeline';

// Re-export shared types
export type { BusinessObjectSummary, BOField } from './BOGovernanceStudio';
