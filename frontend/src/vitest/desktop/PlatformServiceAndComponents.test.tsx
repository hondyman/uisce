import React from 'react';
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { platformService, UniversalPlatformService } from '../../services/platform';
import { DeskWindowFrame, StandaloneWindowWrapper } from '../../components/desktop';
import { fdc3Agent } from '../../services/fdc3';

describe('PlatformService & Desktop Window Components', () => {
  const originalLocation = window.location;

  beforeEach(() => {
    delete (window as any).go;
    delete (window as any).wails;
    sessionStorage.clear();
    localStorage.clear();
  });

  afterEach(() => {
    delete (window as any).go;
    delete (window as any).wails;
    sessionStorage.clear();
    localStorage.clear();
    vi.restoreAllMocks();
  });

  describe('UniversalPlatformService', () => {
    it('lazily detects Wails runtime environment without caching false-negatives', () => {
      const service = new UniversalPlatformService();

      // Initially browser mode
      expect(service.isWails()).toBe(false);
      expect(service.getPlatformType()).toBe('web-browser');

      // Wails injects bindings asynchronously
      (window as any).go = { main: { DeskWindowManager: {} } };

      // Service immediately detects desktop without requiring restart
      expect(service.isWails()).toBe(true);
      expect(service.getPlatformType()).toBe('wails-desktop');

      delete (window as any).go;
      expect(service.isWails()).toBe(false);
    });

    it('queries screens from Wails DeskWindowManager when available', async () => {
      const mockMonitors = [
        { id: 'screen-0', name: 'Built-in Retina', isPrimary: true, index: 0, scaleFactor: 2, x: 0, y: 0, width: 1512, height: 982 },
        { id: 'screen-1', name: 'Dell 4K', isPrimary: false, index: 1, scaleFactor: 1, x: 1512, y: 0, width: 3840, height: 2160 },
      ];

      (window as any).go = {
        main: {
          DeskWindowManager: {
            GetMonitors: vi.fn(async () => mockMonitors),
          },
        },
      };

      const screens = await platformService.getScreens();
      expect(screens).toHaveLength(2);
      expect(screens[0].name).toBe('Built-in Retina');
      expect(screens[0].scale).toBe(2);
      expect(screens[1].name).toBe('Dell 4K');
      expect(screens[1].isPrimary).toBe(false);
    });

    it('spawns native window with session JWT when running in Wails desktop', async () => {
      const mockSpawn = vi.fn(async () => 'CREATED');
      (window as any).go = {
        main: {
          DeskWindowManager: {
            SpawnWindow: mockSpawn,
          },
        },
      };

      sessionStorage.setItem('AUTH_TOKEN', 'jwt_secret_token_123');

      await platformService.displayView({
        viewId: 'oms',
        route: '/view/page/orders',
        title: 'Order Blotter',
        targetScreenIndex: 1,
        width: 1400,
        height: 900,
      });

      expect(mockSpawn).toHaveBeenCalledWith({
        windowId: 'win_oms',
        route: '/view/page/orders',
        title: 'Order Blotter',
        targetScreenIndex: 1,
        x: 40,
        y: 40,
        width: 1400,
        height: 900,
        sessionJwt: 'jwt_secret_token_123',
      });
    });

    it('falls back to window.open in browser mode', async () => {
      const mockOpen = vi.spyOn(window, 'open').mockImplementation(() => null);

      await platformService.displayView({
        viewId: 'rebalance',
        route: '/view/rebalancer',
        title: 'Rebalancer Popout',
      });

      expect(mockOpen).toHaveBeenCalledWith(
        '/view/rebalancer',
        'win_rebalance',
        expect.stringContaining('width=1280')
      );
    });
  });

  describe('DeskWindowFrame', () => {
    it('renders title and FDC3 channel badge', () => {
      const onSelect = vi.fn();
      render(
        <DeskWindowFrame
          title="Execution Blotter"
          channelColor="#22c55e"
          channelName="Green"
          onChannelSelect={onSelect}
        >
          <div data-testid="content">Blotter Content</div>
        </DeskWindowFrame>
      );

      expect(screen.getByText('Execution Blotter')).toBeDefined();
      expect(screen.getByTestId('content')).toBeDefined();

      const badge = screen.getByRole('button', { name: /FDC3 Channel: Green/i });
      fireEvent.click(badge);
      expect(onSelect).toHaveBeenCalledTimes(1);
    });

    it('in browser mode, degrades cleanly without dead minimize button and wires close', () => {
      const mockClose = vi.spyOn(window, 'close').mockImplementation(() => {});

      render(
        <DeskWindowFrame title="Browser Popout">
          <div>Body</div>
        </DeskWindowFrame>
      );

      // Minimize button should NOT be rendered in browser mode
      expect(screen.queryByLabelText('Minimize')).toBeNull();

      // Close button should be wired to window.close()
      const closeBtn = screen.getByRole('button', { name: /Close Window/i });
      fireEvent.click(closeBtn);
      expect(mockClose).toHaveBeenCalledTimes(1);
    });
  });

  describe('StandaloneWindowWrapper', () => {
    it('strips init_token immediately from URL and exchanges token in desktop mode', async () => {
      const replaceStateSpy = vi.spyOn(window.history, 'replaceState');

      const mockExchange = vi.fn(async (tok: string) => 'retrieved_jwt_from_vault');
      (window as any).go = {
        main: {
          DeskWindowManager: {
            ExchangeToken: mockExchange,
          },
        },
      };

      // Set URL with init_token
      window.history.pushState({}, 'Test', '/view/page/orders?init_token=single_use_crypto_token_999');

      render(
        <StandaloneWindowWrapper title="Detached Orders">
          <div data-testid="child-view">Order Grid</div>
        </StandaloneWindowWrapper>
      );

      // Token should be stripped from history
      expect(replaceStateSpy).toHaveBeenCalledWith({}, expect.any(String), '/view/page/orders');

      // ExchangeToken should be called with init_token
      await waitFor(() => {
        expect(mockExchange).toHaveBeenCalledWith('single_use_crypto_token_999');
      });

      // Storage should have token
      expect(sessionStorage.getItem('AUTH_TOKEN')).toBe('retrieved_jwt_from_vault');

      // Content renders
      expect(screen.getByTestId('child-view')).toBeDefined();
    });

    it('in browser mode, renders immediately without token exchange', async () => {
      // Normal browser mode: no Wails bindings
      render(
        <StandaloneWindowWrapper title="Browser Popout View">
          <div data-testid="browser-child">Content</div>
        </StandaloneWindowWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('browser-child')).toBeDefined();
      });
    });
  });
});
