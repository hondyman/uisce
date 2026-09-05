import { useLocation } from 'react-router-dom';
import { Box } from '@mui/material';
import { RouteBlockerProvider } from './components/RouteBlocker/RouteBlocker';
import { ScopeProvider } from './contexts/ScopeContext';
import { MainNavigation } from './components/MainNavigation';
import { ErrorBoundary } from './components/ErrorBoundary';
import { devDebug } from './utils/devLogger';
import { Toaster } from './components/ui/toaster';
import { LocaleShell } from './routes/localeShell';
import { DirectionProvider } from './i18n/DirectionProvider';
import { stripLocale } from './i18n/locales';

function App() {
  const location = useLocation();
  // Dev-only route location debug
  devDebug('[App] location:', location.pathname + location.search);

  // Hide the main navigation for authentication routes (login, reset, signup, register).
  // Compare against the locale-stripped path so /en/login, /es/login, etc. all match.
  const authPaths = ['/login', '/reset-password', '/signup', '/register'];
  const stripped = stripLocale(location.pathname);
  const hideNav = authPaths.some((p) => stripped.startsWith(p));

  return (
    <ErrorBoundary>
      <DirectionProvider>
        <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
          <RouteBlockerProvider>
            <ScopeProvider>
              {!hideNav && <MainNavigation />}
              <Box component="main" id="main-content" tabIndex={-1} sx={{ flexGrow: 1, overflow: 'auto', outline: 'none' }}>
                <LocaleShell />
              </Box>
            </ScopeProvider>
          </RouteBlockerProvider>
          {/* Global toaster for feedback (e.g., logout confirmation) */}
          <Toaster />
        </Box>
      </DirectionProvider>
    </ErrorBoundary>
  );
}

export default App;
