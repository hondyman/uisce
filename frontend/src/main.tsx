import { initSession } from './utils/initSession';
import './i18n';

// Initialize session (dev seeding, etc.)
initSession();

import React, { useState, useMemo } from 'react';
import ReactDOM from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import { MantineProvider } from '@mantine/core';
import { ThemeProvider } from '@mui/material/styles';
import { createUisceTheme } from './theme';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import App from './App.tsx';
import { I18nextProvider } from 'react-i18next';
import i18n from './i18n';
import './index.css';
import '@mantine/core/styles.css';
import './components/brand/navStyles.css';

import { SnackbarProvider } from 'notistack';
import { ConfirmProvider } from './components/ConfirmProvider';
import { TenantProvider } from './contexts/TenantContext';
import { AccessProvider } from './contexts/AccessContext';
import { MetadataProvider } from './contexts/MetadataContext';
import { AuthProvider } from './contexts/AuthContext';
import { ImpersonationProvider } from './contexts/ImpersonationContext';
import { ThemeProvider as CustomThemeProvider, useTheme } from './contexts/ThemeContext';
import { useNotification } from './hooks/useNotification';
import NotificationService from './services/NotificationService';
import DevProxyWarning from './components/DevProxyWarning';

export const ColorModeContext = React.createContext({ toggleColorMode: () => {} });

/**
 * Inner component that uses the theme context.
 * This must be inside the CustomThemeProvider.
 */
function AppWithTheme() {
  const { effectiveTheme } = useTheme();
  const [queryClient] = useState(() => new QueryClient({
    defaultOptions: {
      queries: {
        retry: 1,
        refetchOnWindowFocus: false,
        staleTime: 30_000,
      },
      mutations: {
        retry: 1,
      },
    },
  }));

  const theme = useMemo(
    () => createUisceTheme(effectiveTheme),
    [effectiveTheme],
  );

  const colorMode = useMemo(
    () => ({
      toggleColorMode: () => {
        // This is for backward compatibility with existing code
        // The actual theme toggle is handled by the custom ThemeProvider
      },
    }),
    [],
  );

  return (
    <React.StrictMode>
      <MantineProvider>
        <ColorModeContext.Provider value={colorMode}>
          <ThemeProvider theme={theme}>
            <QueryClientProvider client={queryClient}>
              <BrowserRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
                <AuthProvider>
                  <ImpersonationProvider>
                    <AccessProvider>
                      <SnackbarProvider maxSnack={3}>
                        {/* Set the global notification service using notistack */}
                        <NotificationSetter />
                        <ConfirmProvider>
                          <TenantProvider>
                            <MetadataProvider>
                              {/* Dev proxy check: warns if Vite proxy is likely misconfigured for local host dev */}
                              <DevProxyWarning />
                              <I18nextProvider i18n={i18n}>
                                <App />
                              </I18nextProvider>
                            </MetadataProvider>
                          </TenantProvider>
                        </ConfirmProvider>
                      </SnackbarProvider>
                    </AccessProvider>
                  </ImpersonationProvider>
                </AuthProvider>
              </BrowserRouter>
            </QueryClientProvider>
          </ThemeProvider>
        </ColorModeContext.Provider>
      </MantineProvider>
    </React.StrictMode>
  );
}

function NotificationSetter() {
  const notification = useNotification();

  // Install global notifier for non-component code
  React.useEffect(() => {
    NotificationService.setNotifier((msg: string, opts?: any) => {
      if (opts?.variant === 'error') notification.error(msg);
      else if (opts?.variant === 'success') notification.success(msg);
      else if (opts?.variant === 'warning') notification.warning(msg);
      else notification.info(msg);
    });

    return () => {
      NotificationService.clear();
    };
  }, [notification]);

  return null;
}

function Main() {
  return (
    <CustomThemeProvider>
      <AppWithTheme />
    </CustomThemeProvider>
  );
}


ReactDOM.createRoot(document.getElementById('root')!).render(<Main />);