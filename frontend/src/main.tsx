import { initSession } from './utils/initSession';
import './i18n';

// Initialize session (dev seeding, etc.)
initSession();

import React, { useState, useMemo } from 'react';
import ReactDOM from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import { ApolloProvider } from '@apollo/client';
import { MantineProvider } from '@mantine/core';
import { ThemeProvider, createTheme } from '@mui/material/styles';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import client from './graphql/apolloClient';
import App from './App.tsx';
import { I18nextProvider } from 'react-i18next';
import i18n from './i18n';
import './index.css';
import '@mantine/core/styles.css';

import { SnackbarProvider } from 'notistack';
import { ConfirmProvider } from './components/ConfirmProvider';
import { TenantProvider } from './contexts/TenantContext';
import { AccessProvider } from './contexts/AccessContext';
import { MetadataProvider } from './contexts/MetadataContext';
import { AuthProvider } from './contexts/AuthContext';
import { ThemeProvider as CustomThemeProvider, useTheme } from './contexts/ThemeContext';
import { useNotification } from './hooks/useNotification';
import NotificationService from './services/NotificationService';
import GraphqlOutageBanner from './components/GraphqlOutageBanner';
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
    () =>
      createTheme({
        palette: {
          mode: effectiveTheme,
          primary: {
            main: effectiveTheme === 'dark' ? '#4f86f7' : '#1976d2',
          },
          background: {
            default: effectiveTheme === 'dark' ? '#0b1118' : '#fafafa',
            paper: effectiveTheme === 'dark' ? '#151b23' : '#ffffff',
          },
          text: {
            primary: effectiveTheme === 'dark' ? '#e6edf3' : '#000000',
            secondary: effectiveTheme === 'dark' ? '#8b949e' : '#666666',
          },
        },
        components: {
          MuiPaper: {
            styleOverrides: {
              root: {
                backgroundImage: 'none',
                backgroundColor: effectiveTheme === 'dark' ? '#151b23' : '#ffffff',
                borderColor: effectiveTheme === 'dark' ? '#232f3e' : '#e0e0e0',
              },
            },
          },
          MuiCard: {
            styleOverrides: {
              root: {
                backgroundImage: 'none',
                backgroundColor: effectiveTheme === 'dark' ? '#1c2636' : '#ffffff',
                borderColor: effectiveTheme === 'dark' ? '#232f3e' : '#e0e0e0',
              },
            },
          },
          MuiTextField: {
            styleOverrides: {
              root: {
                '& .MuiOutlinedInput-root': {
                  backgroundColor: effectiveTheme === 'dark' ? '#1c2636' : '#ffffff',
                  '& fieldset': {
                    borderColor: effectiveTheme === 'dark' ? '#232f3e' : '#e0e0e0',
                  },
                  '&:hover fieldset': {
                    borderColor: effectiveTheme === 'dark' ? '#4f86f7' : '#1976d2',
                  },
                  '&.Mui-focused fieldset': {
                    borderColor: effectiveTheme === 'dark' ? '#4f86f7' : '#1976d2',
                  },
                },
                '& .MuiInputBase-input': {
                  color: effectiveTheme === 'dark' ? '#e6edf3' : '#000000',
                },
                '& .MuiInputLabel-root': {
                  color: effectiveTheme === 'dark' ? '#8b949e' : '#666666',
                },
              },
            },
          },
          MuiCheckbox: {
            styleOverrides: {
              root: {
                color: effectiveTheme === 'dark' ? '#30363d' : '#666666',
                '&.Mui-checked': {
                  color: effectiveTheme === 'dark' ? '#4f86f7' : '#1976d2',
                },
              },
            },
          },
          MuiTypography: {
            styleOverrides: {
              root: {
                color: effectiveTheme === 'dark' ? '#e6edf3' : '#000000',
              },
            },
          },
        },
      }),
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
      <ApolloProvider client={client}>
        <MantineProvider>
          <ColorModeContext.Provider value={colorMode}>
            <ThemeProvider theme={theme}>
              <QueryClientProvider client={queryClient}>
                <BrowserRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
                  <AuthProvider>
                    <AccessProvider>
                      <SnackbarProvider maxSnack={3}>
                        {/* Set the global notification service using notistack */}
                        <NotificationSetter />
                        <ConfirmProvider>
                          <TenantProvider>
                            <MetadataProvider>
                              <GraphqlOutageBanner />                              {/* Dev proxy check: warns if Vite proxy is likely misconfigured for local host dev */}
                              <DevProxyWarning />                              <I18nextProvider i18n={i18n}>
                                <App />
                              </I18nextProvider>
                            </MetadataProvider>
                          </TenantProvider>
                        </ConfirmProvider>
                      </SnackbarProvider>
                    </AccessProvider>
                  </AuthProvider>
                </BrowserRouter>
              </QueryClientProvider>
            </ThemeProvider>
          </ColorModeContext.Provider>
        </MantineProvider>
      </ApolloProvider>
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