import React from 'react';
import { CalendarDashboard } from '../components/calendar/CalendarDashboard';

export const CalendarPage: React.FC = () => {
  // Hardcode tenant and user IDs for now. In a real app, these come from auth context.
  const tenantId = import.meta.env.VITE_TENANT_ID || 'tenant-1';
  const userId = 'user-1';

  return (
    <CalendarDashboard tenantId={tenantId} userId={userId} />
  );
};
