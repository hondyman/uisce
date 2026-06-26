import React from 'react';
import { TemplateEditor } from '../../../components/Notifications/TemplateEditor';
import { useTenant } from '../../../contexts/TenantContext';
import { useNavigate } from 'react-router-dom';

export const NotificationTemplateEditorPage: React.FC = () => {
  const { tenant, datasource } = useTenant();
  const navigate = useNavigate();

  if (!tenant || !datasource) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-center">
          <h2 className="text-xl font-bold text-gray-900 mb-2">Tenant and Datasource Required</h2>
          <p className="text-gray-600">Please select a tenant and datasource to manage notification templates.</p>
        </div>
      </div>
    );
  }

  return (
    <TemplateEditor
      tenant={tenant}
      datasource={datasource}
      onSave={() => navigate('/core/notifications')}
      onCancel={() => navigate('/core/notifications')}
    />
  );
};
