import { Routes, Route, Navigate } from 'react-router-dom';
import ObservabilityDashboard from '../pages/ObservabilityDashboard';

export const ObservabilityRoutes = () => {
  return (
    <Routes>
       <Route path="/" element={<ObservabilityDashboard />} />
       <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
};
