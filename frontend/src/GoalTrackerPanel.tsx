import { useState, useEffect } from 'react';
import { listGoals } from './api';
import { devError } from './utils/devLogger';
import type { Goal } from './types';

// A simple timeago placeholder
function timeago(dateStr?: string): string {
  if (!dateStr) return 'never';
  const date = new Date(dateStr);
  const seconds = Math.floor((new Date().getTime() - date.getTime()) / 1000);
  let interval = seconds / 31536000;
  if (interval > 1) return Math.floor(interval) + " years ago";
  interval = seconds / 2592000;
  if (interval > 1) return Math.floor(interval) + " months ago";
  interval = seconds / 86400;
  if (interval > 1) return Math.floor(interval) + " days ago";
  return "today";
}

export default function GoalTrackerPanel() {
  const [goals, setGoals] = useState<Goal[]>([]);

  useEffect(() => {
    // In a real app, user ID would come from an auth context.
  listGoals("user-123").then(setGoals).catch((e) => { devError(e); });
  }, []);

  return (
    <div className="goal-tracker-panel">
      <h4>Your Goals</h4>
      {goals.map(g => (
        <div key={g.id} className={`goal-card status-${g.status ?? 'unknown'}`}>
          <strong>{g.name}</strong>
          <small>{g.description ?? ''}</small>
          <div className="goal-status">Status: {g.status ?? 'unknown'}</div>
          <div className="goal-meta">Last checked: {timeago(g.last_checked ?? undefined)}</div>
        </div>
      ))}
    </div>
  );
}