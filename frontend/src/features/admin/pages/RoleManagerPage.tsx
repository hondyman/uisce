import React from 'react';
import { useTenant } from '../../../contexts/TenantContext';
import { RoleManagerStyled } from '../../../components/RBAC/RoleManager_Styled';

const RoleManagerPage: React.FC = () => {
  const { tenant, datasource } = useTenant();

  if (!tenant || !datasource) {
    return (
      <div className="flex items-center justify-center min-h-screen bg-gradient-to-br from-slate-50 via-blue-50 to-indigo-50">
        <div className="text-center">
          <h2 className="text-2xl font-bold text-gray-900 mb-4">
            No Tenant/Datasource Selected
          </h2>
          <p className="text-gray-600">
            Please select a tenant and datasource to manage roles and permissions.
          </p>
        </div>
      </div>
    );
  }

  return <RoleManagerStyled tenant={tenant} datasource={datasource} />;
};

export default RoleManagerPage;
