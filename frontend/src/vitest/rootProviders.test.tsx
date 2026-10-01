/**
 * Regression test for the boot crash shipped in 5946d868d.
 *
 * `RouterCapabilityProvider` calls `useExtensionsService` -> `useAuthFetch` ->
 * `useAuth`, and `useAuth` THROWS without an AuthProvider. Mounting that provider
 * directly inside <BrowserRouter> therefore crashed the application on startup while
 * the whole unit suite stayed green, because nothing mounted the real composition.
 *
 * This test mounts `RootProviders` — the actual tree main.tsx renders, not a copy —
 * so any provider-ordering regression fails here rather than in the browser.
 *
 * Discrimination note: the negative control is the point of this file. A test that
 * merely asserts "RouterCapabilityProvider throws outside AuthProvider" would pass
 * regardless of how main.tsx is wired, and so would not have caught the shipped bug.
 * To confirm this one can fail, reorder RootProviders so the provider sits outside
 * AuthProvider again and watch this test go red.
 *
 * Scope limit, stated so nobody over-reads it: mounting a tree in jsdom is NOT the
 * same as booting a dev server. The open "boot the app" item is unchanged by this
 * file. What this covers is provider ORDER, which was the actual defect.
 */
import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/react';
import React from 'react';
import { ThemeProvider as CustomThemeProvider } from '../contexts/ThemeContext';
import RootProviders from '../app/RootProviders';

describe('RootProviders composition', () => {
  it('mounts the real provider tree without throwing', () => {
    // RouterCapabilityProvider needs a Router (useNavigate/useLocation) AND an
    // AuthProvider (useAuth). If either is missing, this render throws.
    expect(() =>
      render(
        <CustomThemeProvider>
          <RootProviders>
            <div data-testid="child">mounted</div>
          </RootProviders>
        </CustomThemeProvider>,
      ),
    ).not.toThrow();
  });

  it('renders its children, so the assertion above is not satisfied by rendering nothing', () => {
    const { getByTestId } = render(
      <CustomThemeProvider>
        <RootProviders>
          <div data-testid="child">mounted</div>
        </RootProviders>
      </CustomThemeProvider>,
    );
    expect(getByTestId('child')).toBeTruthy();
  });
});
