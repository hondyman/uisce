import React from 'react';
import { CalendarDashboard } from '../components/calendar/CalendarDashboard';

const CalendarPage: React.FC = () => {
  const tenantId = import.meta.env.VITE_TENANT_ID || 'tenant-1';
  const userId = 'user-1';

  return (
    <CalendarDashboard tenantId={tenantId} userId={userId} />
  );
};

export default CalendarPage;
