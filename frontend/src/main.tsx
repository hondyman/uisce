import { initSession } from './utils/initSession';
import './i18n';

// Initialize session (dev seeding, etc.)
initSession();

import React from 'react';
import ReactDOM from 'react-dom/client';
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
import { ImpersonationProvider } from './contexts/ImpersonationContext';
import { ThemeProvider as CustomThemeProvider } from './contexts/ThemeContext';
import { useNotification } from './hooks/useNotification';
import NotificationService from './services/NotificationService';
import DevProxyWarning from './components/DevProxyWarning';
import { RootProviders } from './app/RootProviders';

/**
 * Inner component that must live inside the CustomThemeProvider.
 *
 * The provider tree itself - Router, Auth, RouterCapability and their required
 * ordering - lives in app/RootProviders so a test can mount the real composition
 * instead of a copy. See the note there for why that ordering is load-bearing.
 */
function AppWithTheme() {
  return (
    <RootProviders>
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
    </RootProviders>
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