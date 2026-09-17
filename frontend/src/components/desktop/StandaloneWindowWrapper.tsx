import React, { useCallback, useEffect, useState } from 'react';
import { DeskWindowFrame } from './DeskWindowFrame';
import { fdc3Agent, USER_CHANNELS, UserChannelId } from '../../services/fdc3';
import { platformService } from '../../services/platform/PlatformService';
import { devError } from '../../utils/devLogger';

export interface StandaloneWindowWrapperProps {
  title: string;
  children: React.ReactNode;
}

/**
 * Universal wrapper for popout / secondary monitor windows.
 * 
 * Responsibilities:
 * 1. Zero-Trust Token Exchange (Desktop Mode only):
 *    Extracts ephemeral single-use token (?init_token=...) from URL, exchanges
 *    it with Go process memory, sets session storage, and immediately strips the
 *    token from the window URL and history.
 * 2. Browser Mode Session Preservation:
 *    Skips the ephemeral token exchange in browser mode; relies on the browser's
 *    shared cookie/localStorage session.
 * 3. FDC3 Interoperability:
 *    Synchronizes the window's channel badge with the FDC3 Desktop Agent and
 *    allows cycling channels.
 */
export const StandaloneWindowWrapper: React.FC<StandaloneWindowWrapperProps> = ({
  title,
  children,
}) => {
  const [activeChannelId, setActiveChannelId] = useState<UserChannelId>(() => fdc3Agent.getCurrentChannelId());
  const [isReady, setIsReady] = useState<boolean>(false);
  const [authError, setAuthError] = useState<string | null>(null);

  // Set OS window and document title for standalone/detached view
  useEffect(() => {
    if (title) {
      document.title = `${title} — Uisce`;
    }
  }, [title]);

  // 1. First thing on mount: Clean URL and perform authentication handshake if in desktop mode
  useEffect(() => {
    let cancelled = false;

    const initSecurityAndSession = async () => {
      const urlParams = new URLSearchParams(window.location.search);
      const initToken = urlParams.get('init_token');

      // Strip the token from URL immediately to protect process logs and history
      if (initToken) {
        urlParams.delete('init_token');
        const newSearch = urlParams.toString();
        const newUrl = window.location.pathname + (newSearch ? `?${newSearch}` : '');
        window.history.replaceState({}, document.title, newUrl);
      }

      // Desktop Mode: Exchange ephemeral single-use token for JWT with Go vault
      if (platformService.isWails()) {
        if (initToken) {
          try {
            const deskManager = (window as unknown as {
              go?: {
                main?: {
                  DeskWindowManager?: {
                    ExchangeToken?: (token: string) => Promise<string>;
                  };
                };
              };
            })?.go?.main?.DeskWindowManager;

            if (typeof deskManager?.ExchangeToken === 'function') {
              const sessionJwt = await deskManager.ExchangeToken(initToken);
              if (sessionJwt) {
                sessionStorage.setItem('AUTH_TOKEN', sessionJwt);
                localStorage.setItem('AUTH_TOKEN', sessionJwt);
                localStorage.setItem('auth_token', sessionJwt);
                if (!localStorage.getItem('auth_user')) {
                  localStorage.setItem('auth_user', JSON.stringify({
                    id: 'user-pm-workstation',
                    email: 'trader@uisce.local',
                    name: 'Portfolio Manager',
                    role: 'portfolio_manager',
                    organization: 'uisce',
                    permissions: ['*'],
                    is_active: true,
                  }));
                }
              }
            }
          } catch (err) {
            devError('[StandaloneWindowWrapper] Failed to exchange initialization token:', err);
            if (!cancelled) {
              setAuthError('Authentication handshake failed for this window.');
            }
          }
        }
      }

      // Browser Mode or completed desktop exchange: session is ready
      if (!cancelled) {
        setIsReady(true);
      }
    };

    initSecurityAndSession();

    return () => {
      cancelled = true;
    };
  }, []);

  // 2. Listen for FDC3 channel changes
  useEffect(() => {
    const unsubscribe = fdc3Agent.onChannelChanged((channelId) => {
      setActiveChannelId(channelId);
    });
    return () => {
      unsubscribe();
    };
  }, []);

  // 3. Channel selection cycling
  const handleCycleChannel = useCallback(() => {
    const currentIndex = USER_CHANNELS.findIndex((c) => c.id === activeChannelId);
    const nextChannel = USER_CHANNELS[(currentIndex + 1) % USER_CHANNELS.length].id;
    fdc3Agent.joinUserChannel(nextChannel);
  }, [activeChannelId]);

  const currentChannelDesc = USER_CHANNELS.find((c) => c.id === activeChannelId) || USER_CHANNELS[4]; // default blue

  if (authError) {
    return (
      <DeskWindowFrame title={title}>
        <div style={{ padding: '32px', color: '#f87171', background: '#050d1a', height: '100%' }}>
          <h3>Security Handshake Failed</h3>
          <p>{authError}</p>
        </div>
      </DeskWindowFrame>
    );
  }

  if (!isReady) {
    return (
      <DeskWindowFrame title={title}>
        <div style={{ padding: '32px', color: '#94a3b8', background: '#050d1a', height: '100%' }}>
          Initializing detached window session...
        </div>
      </DeskWindowFrame>
    );
  }

  return (
    <DeskWindowFrame
      title={title}
      channelColor={currentChannelDesc.color}
      channelName={currentChannelDesc.name}
      onChannelSelect={handleCycleChannel}
    >
      {children}
    </DeskWindowFrame>
  );
};
