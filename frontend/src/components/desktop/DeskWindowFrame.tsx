import React, { useCallback, useEffect, useState } from 'react';
import { platformService } from '../../services/platform/PlatformService';
import './DeskWindowFrame.css';

export interface DeskWindowFrameProps {
  title: string;
  channelColor?: string;
  channelName?: string;
  onChannelSelect?: () => void;
  children: React.ReactNode;
}

export const DeskWindowFrame: React.FC<DeskWindowFrameProps> = ({
  title,
  channelColor = '#3b82f6',
  channelName = 'Blue',
  onChannelSelect,
  children,
}) => {
  const [isWails, setIsWails] = useState<boolean>(() => platformService.isWails());
  const [isFullscreen, setIsFullscreen] = useState<boolean>(false);

  useEffect(() => {
    setIsWails(platformService.isWails());

    const handleFullscreenChange = () => {
      setIsFullscreen(!!document.fullscreenElement);
    };

    document.addEventListener('fullscreenchange', handleFullscreenChange);
    return () => {
      document.removeEventListener('fullscreenchange', handleFullscreenChange);
    };
  }, []);

  const handleBrowserClose = useCallback(() => {
    if (typeof window !== 'undefined') {
      window.close();
    }
  }, []);

  const handleBrowserMaximize = useCallback(() => {
    if (typeof document === 'undefined') return;
    if (document.fullscreenElement) {
      document.exitFullscreen?.().catch(() => {});
    } else {
      document.documentElement.requestFullscreen?.().catch(() => {});
    }
  }, []);

  return (
    <div className="desk-window-shell">
      <header className="desk-caption-bar">
        <div className="desk-caption-left">
          {onChannelSelect && (
            <button
              type="button"
              className="channel-badge"
              style={{ backgroundColor: channelColor, color: channelColor }}
              onClick={onChannelSelect}
              title={`FDC3 Channel: ${channelName} (Click to switch)`}
              aria-label={`FDC3 Channel: ${channelName}`}
            />
          )}
          <span className="desk-caption-title">{title}</span>
        </div>

        {/* Native or DOM drag region */}
        <div
          className="desk-drag-region"
          style={isWails ? ({ '--wails-non-client-region': 'caption' } as React.CSSProperties) : undefined}
        />

        {/* Caption Controls */}
        <div className="desk-caption-controls">
          {isWails ? (
            // Native Wails v3 controls
            <>
              <button
                type="button"
                className="desk-control-btn minimize"
                style={{ '--wails-non-client-region': 'minimize' } as React.CSSProperties}
                aria-label="Minimize"
              >
                &#x2212;
              </button>
              <button
                type="button"
                className="desk-control-btn maximize"
                style={{ '--wails-non-client-region': 'maximize' } as React.CSSProperties}
                aria-label="Maximize / Snap Layouts"
              >
                &#x25A2;
              </button>
              <button
                type="button"
                className="desk-control-btn close"
                style={{ '--wails-non-client-region': 'close' } as React.CSSProperties}
                aria-label="Close"
              >
                &#x2715;
              </button>
            </>
          ) : (
            // Browser popout controls: degrade cleanly without dead minimize buttons
            <>
              <button
                type="button"
                className="desk-control-btn maximize"
                onClick={handleBrowserMaximize}
                title={isFullscreen ? 'Exit Fullscreen' : 'Fullscreen'}
                aria-label={isFullscreen ? 'Exit Fullscreen' : 'Fullscreen'}
              >
                &#x25A2;
              </button>
              <button
                type="button"
                className="desk-control-btn close"
                onClick={handleBrowserClose}
                title="Close Window"
                aria-label="Close Window"
              >
                &#x2715;
              </button>
            </>
          )}
        </div>
      </header>

      <main className="desk-window-body">{children}</main>
    </div>
  );
};
