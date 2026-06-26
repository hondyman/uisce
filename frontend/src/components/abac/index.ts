/**
 * ABAC Component Library
 * 
 * Complete Attribute-Based Access Control component library for React.
 * All components are multi-tenant safe and integrate with the trigger system.
 * 
 * Usage:
 * ```tsx
 * import { ABACProvider, useABAC, PolicyBuilder } from './components/abac';
 * 
 * function App() {
 *   return (
 *     <ABACProvider tenantId={tenantId}>
 *       <PolicyBuilder />
 *       <DelegationManager />
 *       <AuditLogViewer />
 *     </ABACProvider>
 *   );
 * }
 * ```
 */

export { ABACProvider, useABAC, type ABACPolicy, type ABACEvaluationRequest, type ABACEvaluationResult } from './ABACProvider';
export { PolicyBuilder } from './PolicyBuilder';
export { DelegationManager } from './DelegationManager';
export { AuditLogViewer } from './AuditLogViewer';
