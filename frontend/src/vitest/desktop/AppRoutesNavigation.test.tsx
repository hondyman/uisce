import React from 'react';
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import App from '../../App.tsx';
import { stripLocale } from '../../i18n/locales';

// Mock child components to keep router test fast and isolated
vi.mock('../../components/MainNavigation', () => ({
  MainNavigation: () => <div data-testid="main-navigation">Main Navigation Bar</div>,
}));

vi.mock('../../routes/localeShell', () => ({
  LocaleShell: () => <div data-testid="locale-shell">Locale Shell Content</div>,
}));

vi.mock('../../components/RouteBlocker/RouteBlocker', () => ({
  RouteBlockerProvider: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}));

vi.mock('../../contexts/ScopeContext', () => ({
  ScopeProvider: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}));

vi.mock('../../components/ErrorBoundary', () => ({
  ErrorBoundary: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}));

vi.mock('../../components/ui/toaster', () => ({
  Toaster: () => null,
}));

describe('App Layout & Popout Navigation Suppression', () => {
  it('renders MainNavigation on standard app routes (e.g. /)', () => {
    render(
      <MemoryRouter initialEntries={['/']}>
        <App />
      </MemoryRouter>
    );

    expect(screen.getByTestId('main-navigation')).toBeDefined();
    expect(screen.getByTestId('locale-shell')).toBeDefined();
  });

  it('renders MainNavigation on localized standard routes (e.g. /en/orders)', () => {
    render(
      <MemoryRouter initialEntries={['/en/orders']}>
        <App />
      </MemoryRouter>
    );

    expect(screen.getByTestId('main-navigation')).toBeDefined();
  });

  it('suppresses MainNavigation on standalone popout routes (e.g. /view/page/orders)', () => {
    render(
      <MemoryRouter initialEntries={['/view/page/orders']}>
        <App />
      </MemoryRouter>
    );

    // Main navigation MUST be hidden so the popout renders full-window with DeskWindowFrame only
    expect(screen.queryByTestId('main-navigation')).toBeNull();
    expect(screen.getByTestId('locale-shell')).toBeDefined();
  });

  it('suppresses MainNavigation on localized standalone routes (e.g. /en/view/rebalancer)', () => {
    render(
      <MemoryRouter initialEntries={['/en/view/rebalancer']}>
        <App />
      </MemoryRouter>
    );

    expect(screen.queryByTestId('main-navigation')).toBeNull();
  });

  it('correctly tests stripLocale for popout paths', () => {
    expect(stripLocale('/view/page/orders')).toBe('/view/page/orders');
    expect(stripLocale('/en/view/page/orders')).toBe('/view/page/orders');
    expect(stripLocale('/es/view/rebalancer')).toBe('/view/rebalancer');
  });
});
