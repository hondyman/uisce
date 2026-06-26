import { useState, useEffect } from 'react';
import { listTours } from './api';
import { devError } from './utils/devLogger';
import type { Tour } from './types';

interface TourLauncherProps {
  onStartTour: (tourId: string) => void;
}

export default function TourLauncher({ onStartTour }: TourLauncherProps) {
  const [tours, setTours] = useState<Tour[]>([]);

  useEffect(() => {
  // In a real app, user ID would come from an auth context.
  listTours("user-123").then(setTours).catch((e) => { devError(e); });
  }, []);

  return (
    <div className="tour-launcher-panel">
      <h4>Explorer Tours</h4>
      {tours.map(t => (
        <div key={t.id} className="tour-card">
          <strong>{t.name}</strong>
          <small>{t.description}</small>
          <button onClick={() => onStartTour(t.id)}>Start Tour</button>
        </div>
      ))}
    </div>
  );
}