/**
 * NotificationBell - Header notification indicator with unread badge
 * 
 * Features:
 * - Real-time unread count badge
 * - Auto-refresh every 30 seconds
 * - Link to notification center
 * - Pulsing animation for new notifications
 */

import React, { useState, useEffect } from 'react';
import { Bell } from 'lucide-react';
import { useTenant } from '../../contexts/TenantContext';
import { useAuth } from '../../contexts/AuthContext';
import { useNavigate } from 'react-router-dom';

export const NotificationBell: React.FC = () => {
  const { tenant, datasource } = useTenant();
  const { user } = useAuth();
  const navigate = useNavigate();
  const [unreadCount, setUnreadCount] = useState(0);
  const [loading, setLoading] = useState(false);

  const fetchUnreadCount = async () => {
    if (!tenant?.id || !datasource?.id || !user?.id) {
      setUnreadCount(0);
      return;
    }

    try {
      setLoading(true);
      const response = await fetch(
        `/api/bp-notifications/logs?tenant_id=${tenant.id}&tenant_instance_id=${datasource.id}&user_id=${user.id}`
      );
      
      if (response.ok) {
        const data = await response.json();
        const unread = (data || []).filter((n: any) => !n.opened_at).length;
        setUnreadCount(unread);
      }
    } catch (error) {
      console.error('Failed to fetch notification count:', error);
    } finally {
      setLoading(false);
    }
  };

  // Initial load
  useEffect(() => {
    fetchUnreadCount();
  }, [tenant?.id, datasource?.id, user?.id]);

  // Auto-refresh every 30 seconds
  useEffect(() => {
    if (!tenant?.id || !datasource?.id || !user?.id) return;
    
    const interval = setInterval(fetchUnreadCount, 30000);
    return () => clearInterval(interval);
  }, [tenant?.id, datasource?.id, user?.id]);

  const handleClick = () => {
    navigate('/core/notifications');
  };

  return (
    <button
      onClick={handleClick}
      className="relative p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-800 transition-all"
      title="Notifications"
    >
      <Bell 
        className={`w-6 h-6 text-gray-700 dark:text-gray-300 ${
          unreadCount > 0 ? 'animate-pulse' : ''
        }`}
      />
      {unreadCount > 0 && (
        <span className="absolute -top-1 -right-1 flex items-center justify-center w-5 h-5 text-xs font-bold text-white bg-red-500 rounded-full animate-pulse">
          {unreadCount > 9 ? '9+' : unreadCount}
        </span>
      )}
    </button>
  );
};
