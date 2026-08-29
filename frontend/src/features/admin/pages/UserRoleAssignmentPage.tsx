import React from 'react';
import { useTenant } from '../../../contexts/TenantContext';
import { UserRoleAssignmentStyled } from '../../../components/RBAC/UserRoleAssignment_Styled';

const UserRoleAssignmentPage: React.FC = () => {
  const { tenant, datasource } = useTenant();

  if (!tenant) {
    return (
      <div className="flex items-center justify-center min-h-screen bg-gradient-to-br from-slate-50 via-blue-50 to-indigo-50">
        <div className="text-center">
          <h2 className="text-2xl font-bold text-gray-900 mb-4">
            No Tenant Selected
          </h2>
          <p className="text-gray-600">
            Please select a tenant to manage user role assignments.
          </p>
        </div>
      </div>
    );
  }

  const effectiveDatasource = datasource || { id: 'none', source_name: 'Default' };

  return <UserRoleAssignmentStyled tenant={tenant} datasource={effectiveDatasource} />;
};

export default UserRoleAssignmentPage;
