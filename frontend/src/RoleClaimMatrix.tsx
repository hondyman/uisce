import { useState, useEffect, useCallback } from 'react';
import { listAllRoles, listSemanticViews, listRoleClaims, updateRoleClaim } from './api';
import { useNotification } from './hooks/useNotification';
import { SemanticViewMeta, SemanticModelRoleClaim as _SemanticModelRoleClaim } from './types';

type PermissionMatrix = {
  [role: string]: {
    [modelId: string]: {
      read: boolean;
      write: boolean;
    };
  };
};

interface RoleClaimMatrixProps {
  domain?: string;
}

export default function RoleClaimMatrix({ domain }: RoleClaimMatrixProps) {
  const [roles, setRoles] = useState<string[]>([]);
  const [models, setModels] = useState<SemanticViewMeta[]>([]);
  const [matrix, setMatrix] = useState<PermissionMatrix>({});
  const [loading, setLoading] = useState(true);
  const notification = useNotification();
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    try {
      setLoading(true);
      // In a real app, the datasource might be selectable or global.
      const [fetchedRoles, fetchedModels, fetchedClaims] = await Promise.all([
        listAllRoles(),
        listSemanticViews("mock-datasource-id"),
        listRoleClaims(),
      ]);

      setRoles(fetchedRoles);

      // NEW: Filter models by domain if provided
      const filteredModels = domain
        ? fetchedModels.filter((m: any) => m.domain === domain)
        : fetchedModels;
      setModels(filteredModels);

      // Build the initial matrix from fetched claims
      const initialMatrix: PermissionMatrix = {};
      for (const role of fetchedRoles) {
        initialMatrix[role] = {};
        for (const model of filteredModels) {
          initialMatrix[role][model.id] = { read: false, write: false };
        }
      }

      for (const claim of fetchedClaims) {
        const roleKey = (claim.role ?? '') as string;
        const modelId = (claim.model_id ?? '') as string;
        const perms = Array.isArray(claim.permissions) ? claim.permissions : [];
        if (roleKey && modelId && initialMatrix[roleKey] && initialMatrix[roleKey][modelId]) {
          initialMatrix[roleKey][modelId] = {
            read: perms.includes('read'),
            write: perms.includes('write'),
          };
        }
      }
      setMatrix(initialMatrix);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch data');
    } finally {
      setLoading(false);
    }
  }, [domain]);

  useEffect(() => {
    fetchData();
  }, [fetchData, domain]);

  const handlePermissionChange = async (role: string, modelId: string, permission: 'read' | 'write') => {
    const currentPermissions = { ...matrix[role][modelId] };
    const newPermissionsState = { ...currentPermissions, [permission]: !currentPermissions[permission] };

    // Optimistically update UI
    setMatrix(prev => ({
      ...prev,
      [role]: { ...prev[role], [modelId]: newPermissionsState },
    }));

    const permissionsArray = Object.entries(newPermissionsState)
      .filter(([, hasPerm]) => hasPerm)
      .map(([permKey]) => permKey);

    try {
      await updateRoleClaim(role, modelId, permissionsArray);
    } catch (err) {
      notification.error('Failed to update permission.');
      setMatrix(prev => ({
        ...prev,
        [role]: { ...prev[role], [modelId]: currentPermissions },
      }));
    }
  };

  if (loading) return <div>Loading role mappings...</div>;
  if (error) return <div className="error">Error: {error}</div>;

  return (
    <div className="role-claim-matrix-container">
      <h2>Role to Model Permission Matrix</h2>
      <p>Manage which roles have read/write access to semantic models.</p>
      <div className="role-claim-matrix-wrapper">
        <table className="role-claim-matrix">
          <thead>
            <tr>
              <th className="role-header">Role</th>
              {models.map(model => (
                <th key={model.id} className="model-header" title={model.description}>
                  {model.name}
                  {model.certified && <span title="Certified"> ✅</span>}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {roles.map(role => (
              <tr key={role}>
                <td className="role-cell">{role}</td>
                {models.map(model => (
                  <td key={model.id} className="permission-cell">
                    <label title={`Read access for ${role} on ${model.name}`}>
                      R
                      <input
                        type="checkbox"
                        checked={matrix[role]?.[model.id]?.read ?? false}
                        onChange={() => handlePermissionChange(role, model.id, 'read')}
                      />
                    </label>
                    <label title={`Write access for ${role} on ${model.name}`}>
                      W
                      <input
                        type="checkbox"
                        checked={matrix[role]?.[model.id]?.write ?? false}
                        onChange={() => handlePermissionChange(role, model.id, 'write')}
                      />
                    </label>
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}