import { useLocation } from 'react-router-dom';
import { Box } from '@mui/material';
import { RouteBlockerProvider } from './components/RouteBlocker/RouteBlocker';
import { ScopeProvider } from './contexts/ScopeContext';
import { MainNavigation } from './components/MainNavigation';
import { ErrorBoundary } from './components/ErrorBoundary';
import { devDebug } from './utils/devLogger';
import { Toaster } from './components/ui/toaster';
import { AppRoutes } from './AppRoutes';

function App() {
  const location = useLocation();
  // Dev-only route location debug
  devDebug('[App] location:', location.pathname + location.search);

  // Hide the main navigation for authentication routes (login, reset, signup, register)
  const authPaths = ['/login', '/reset-password', '/signup', '/register'];
  const hideNav = authPaths.some((p) => location.pathname.startsWith(p));

  return (
    <ErrorBoundary>
      <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
        <RouteBlockerProvider>
          <ScopeProvider>
            {!hideNav && <MainNavigation />}
            <Box sx={{ flexGrow: 1, overflow: 'auto' }}>
              <AppRoutes />
            </Box>
          </ScopeProvider>
        </RouteBlockerProvider>
        {/* Global toaster for feedback (e.g., logout confirmation) */}
        <Toaster />
      </Box>
    </ErrorBoundary>
  );
}

export default App;
