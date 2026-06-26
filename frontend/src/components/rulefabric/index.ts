/**
 * Rule Fabric Components Index
 * 
 * Unified exports for all Rule Fabric components
 */

// Core Editor Components
export { UnifiedRuleEditor } from './UnifiedRuleEditor';


// Governance Components
export {
  RuleGovernance,
  VersionHistoryPanel,
  ApprovalWorkflowPanel,
  ImpactAnalysisPanel,
  ConflictDetectionPanel
} from './RuleGovernance';

// AI & Suggestions
export { AIRuleSuggestions } from './AIRuleSuggestions';

// Testing & Simulation
export { RuleTestingPanel } from './RuleTestingPanel';

// Monitoring & Analytics
export { RuleExecutionMonitor } from './RuleExecutionMonitor';

// Template Marketplace
export { RuleTemplateMarketplace } from './RuleTemplateMarketplace';

// Scheduling & Triggers
export { AdvancedScheduling } from './AdvancedScheduling';

// Re-export types for external use
export type {
  // Rule types
  RuleCategory,
  ExecutionChannel,
  RuleStatus,
  // Common types used across components
} from './UnifiedRuleEditor';
