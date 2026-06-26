import { useState, useEffect } from 'react';
import { listAlerts, markAlertAsRead } from './api';
import type { ExplorerAlert } from './types';
import { devLog } from './utils/devLogger';

interface AlertCenterProps {
  userId: string;
}

function timeago(dateStr: string) {
  const date = new Date(dateStr);
  const seconds = Math.floor((new Date().getTime() - date.getTime()) / 1000);
  let interval = seconds / 31536000;
  if (interval > 1) return Math.floor(interval) + " years ago";
  interval = seconds / 2592000;
  if (interval > 1) return Math.floor(interval) + " months ago";
  interval = seconds / 86400;
  if (interval > 1) return Math.floor(interval) + " days ago";
  interval = seconds / 3600;
  if (interval > 1) return Math.floor(interval) + " hours ago";
  interval = seconds / 60;
  if (interval > 1) return Math.floor(interval) + " minutes ago";
  return Math.floor(seconds) + " seconds ago";
}

export default function AlertCenter({ userId }: AlertCenterProps) {
  const [alerts, setAlerts] = useState<ExplorerAlert[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    listAlerts(userId)
      .then(setAlerts)
  .catch((e) => { import('./utils/devLogger').then(({ devError }) => devError(e)).catch(() => {}); })
      .finally(() => setLoading(false));
  }, [userId]);

  const handleDismiss = (alertId: string) => {
    markAlertAsRead(alertId);
    setAlerts(alerts.filter(a => a.id !== alertId));
  };

  const handleView = (assetId: string) => {
    // In a real app, this would navigate to the asset.
  devLog(`Navigating to asset ${assetId}`);
  };

  if (loading) {
    return <div>Loading alerts...</div>;
  }

  if (alerts.length === 0) {
    return <div className="alert-center"><h4>Alerts</h4><p>No new alerts.</p></div>;
  }

  return (
    <div className="alert-center">
      <h4>Alerts</h4>
      <div className="alert-list">
        {alerts.map(a => (
          <div key={a.id} className={`alert-card severity-${a.severity}`}>
            <p>{a.message}</p>
            <div className="alert-footer">
              <small>{timeago(a.triggered_at)}</small>
              <div className="alert-actions">
                <button onClick={() => a.asset_id && handleView(a.asset_id)}>View</button>
                <button onClick={() => handleDismiss(a.id)}>Dismiss</button>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}