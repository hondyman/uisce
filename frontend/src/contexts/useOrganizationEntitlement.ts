import { useAuth } from './AuthContext';

export type OrganizationEntitlement = 'crud' | 'read' | 'none';

/**
 * Resolves the user's organization entitlement from the Keycloak
 * `operator_role` claim, the Keycloak realm role, the federated IdP group,
 * or the `is_global_admin` flag.
 *
 *   global_admin / global_ops / platform_operator   -> 'crud'
 *   helpdesk / professional_services                -> 'read'
 *   anything else (or missing)                      -> 'none'
 *
 * The actual signal detection lives in `AuthContext` so the rule is applied
 * consistently to every consumer (`isGlobalAdmin()`, `isReadOnlyOperator()`,
 * `isPlatformOperator`, and the page-level entitlement hook).
 */
export function useOrganizationEntitlement(): {
  level: OrganizationEntitlement;
  canRead: boolean;
  canWrite: boolean;
  isVisible: boolean;
} {
  const { isGlobalAdmin, isReadOnlyOperator } = useAuth();
  const crud = isGlobalAdmin();
  const read = !crud && isReadOnlyOperator();
  const level: OrganizationEntitlement = crud ? 'crud' : read ? 'read' : 'none';

  return {
    level,
    canRead: crud || read,
    canWrite: crud,
    isVisible: crud || read,
  };
}
