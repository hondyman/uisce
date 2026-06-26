import React from 'react';
import { UserPreferences } from '../../../components/Notifications/UserPreferences';
import { useTenant } from '../../../contexts/TenantContext';
import { useAuth } from '../../../contexts/AuthContext';

export const NotificationPreferencesPage: React.FC = () => {
  const { tenant, datasource } = useTenant();
  const { user } = useAuth();

  if (!tenant || !datasource) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-center">
          <h2 className="text-xl font-bold text-gray-900 mb-2">Tenant and Datasource Required</h2>
          <p className="text-gray-600">Please select a tenant and datasource to manage notification preferences.</p>
        </div>
      </div>
    );
  }

  if (!user?.id) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-center">
          <h2 className="text-xl font-bold text-gray-900 mb-2">Authentication Required</h2>
          <p className="text-gray-600">Please log in to manage your notification preferences.</p>
        </div>
      </div>
    );
  }

  return (
    <UserPreferences
      tenant={tenant}
      datasource={datasource}
      userId={user.id}
    />
  );
};
