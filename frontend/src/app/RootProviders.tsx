import React, { useMemo, useState } from 'react';
import { BrowserRouter } from 'react-router-dom';
import { MantineProvider } from '@mantine/core';
import { ThemeProvider } from '@mui/material/styles';
import { createUisceTheme } from '../theme';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { AuthProvider } from '../contexts/AuthContext';
import { useTheme } from '../contexts/ThemeContext';import { RouterCapabilityProvider } from '../components/RouteBlocker/RouterCapabilityContext';

export const ColorModeContext = React.createContext({ toggleColorMode: () => {} });

/**
 * The root provider tree, extracted so it can be mounted in a test.
 *
 * Why this is a module and not an inline tree in main.tsx: provider ORDER here is
 * load-bearing and nothing else checks it. `RouterCapabilityProvider` calls
 * `useExtensionsService` -> `useAuthFetch` -> `useAuth`, which THROWS without an
 * AuthProvider, so it must sit below <AuthProvider>; it also needs the Router for
 * `useNavigate`/`useLocation`, so it must sit below <BrowserRouter>. An earlier
 * version of this tree mounted it directly inside <BrowserRouter> and the
 * application crashed on startup, with the unit suite green throughout because
 * nothing ever mounted the composition.
 *
 * Ordering constraints, innermost last:
 *   BrowserRouter          supplies useNavigate / useLocation
 *     AuthProvider         supplies useAuth
 *       RouterCapabilityProvider   consumes both
 *
 * `rootProviders.test.tsx` mounts THIS component, so a reordering that breaks any
 * of those constraints fails a test rather than the app. It is not a substitute for
 * booting a dev server — see the open "boot the app" item — but it does catch this
 * class, and any future one, structurally.
 *
 * Must be rendered inside <CustomThemeProvider>, which supplies the theme this
 * tree reads via useTheme().
 */
export const RootProviders: React.FC<{ children?: React.ReactNode }> = ({ children }) => {
  const { effectiveTheme } = useTheme();

  const theme = useMemo(() => createUisceTheme(effectiveTheme), [effectiveTheme]);

  const colorMode = useMemo(
    () => ({
      toggleColorMode: () => {
        // Kept for backward compatibility; the real toggle lives in ThemeContext.
      },
    }),
    [],
  );

  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: { retry: 1, refetchOnWindowFocus: false, staleTime: 30_000 },
          mutations: { retry: 1 },
        },
      }),
  );

  return (
    <React.StrictMode>
      <MantineProvider>
        <ColorModeContext.Provider value={colorMode}>
          <ThemeProvider theme={theme}>
            <QueryClientProvider client={queryClient}>
              <BrowserRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
                <AuthProvider>
                  {/* Depends on BOTH ancestors above. Keep nested, not sibling. */}
                  <RouterCapabilityProvider>{children}</RouterCapabilityProvider>
                </AuthProvider>
              </BrowserRouter>
            </QueryClientProvider>
          </ThemeProvider>
        </ColorModeContext.Provider>
      </MantineProvider>
    </React.StrictMode>
  );
};

export default RootProviders;
