/**
 * Wealth Management Scheduler Components
 * 
 * An intelligent orchestration platform for financial services operations.
 * These components transform scheduling from simple cron jobs into a 
 * domain-aware operating system for wealth management.
 * 
 * Features:
 * - Client Lifecycle Event Intelligence with ML prediction
 * - Surge Meeting Orchestration with constraint satisfaction
 * - Straight-Through Processing with settlement automation
 * - Regulatory Deadline Intelligence with penalty exposure
 * - Multi-Custodian Calendar unification
 * - AI-powered Meeting Preparation automation
 * - Visual Workflow DAG Builder
 * - Predictive Resource Capacity Planning
 */

// Client Lifecycle Event Intelligence
// Tracks life events (retirement, inheritance, divorce) and triggers proactive workflows
export { ClientLifecycleEngine } from './ClientLifecycleEngine';

// Surge Meeting Orchestrator
// AI-powered load balancing for quarterly reviews and market volatility periods
export { SurgeMeetingOrchestrator } from './SurgeMeetingOrchestrator';

// Straight-Through Processing Integration
// Automated trade lifecycle with settlement scheduling and exception handling
export { STPIntegration } from './STPIntegration';

// Regulatory Deadline Intelligence
// SEC/FINRA/State deadline tracking with penalty exposure calculations
export { RegulatoryDeadlineIntelligence } from './RegulatoryDeadlineIntelligence';

// Multi-Custodian Calendar
// Unified calendar across Schwab, Fidelity, Pershing with holiday/settlement logic
export { MultiCustodianCalendar } from './MultiCustodianCalendar';

// Meeting Preparation Automation
// Auto-generated meeting packets with performance attribution and action items
export { MeetingPrepAutomation } from './MeetingPrepAutomation';

// Workflow DAG Builder
// Visual dependency graph builder with OR-Tools constraint satisfaction
export { WorkflowDAGBuilder } from './WorkflowDAGBuilder';

// Predictive Capacity Planning
// ML-based workload forecasting and resource optimization
export { PredictiveCapacityPlanning } from './PredictiveCapacityPlanning';
