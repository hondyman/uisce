import { useEffect, useState, useCallback, useRef } from 'react';
import { fdc3Agent } from './Fdc3DesktopAgent';
import { Fdc3Context, UserChannelId } from './types';
import { IntentHandler, IntentResolution, StandardIntent } from './intentTypes';

/**
 * Hook to participate in FDC3 context synchronization.
 * 
 * CRITICAL ARCHITECTURAL RULE — LOOP GUARD:
 * Listener-driven state changes must NEVER trigger a context broadcast.
 * - Incoming context handlers should only mutate local component state.
 * - Broadcasts must strictly be initiated by explicit USER interactions
 *   (e.g., button clicks, user-driven dropdown changes, row selections).
 * Failing to adhere to this rule creates an infinite ping-pong broadcast loop
 * between multiple synchronized windows!
 */
export function useFdc3<T extends Fdc3Context = Fdc3Context>(
  contextType?: string,
  onContextReceived?: (context: T) => void
) {
  const [activeChannel, setActiveChannel] = useState<UserChannelId>(fdc3Agent.getCurrentChannelId());
  const [currentContext, setCurrentContext] = useState<T | null>(null);

  // Keep a stable ref to callback to prevent re-subscribing on every render
  const callbackRef = useRef(onContextReceived);
  callbackRef.current = onContextReceived;

  useEffect(() => {
    // 1. Subscribe to active channel changes
    const unsubChannel = fdc3Agent.onChannelChanged((chId) => {
      setActiveChannel(chId);
    });

    // 2. Subscribe to incoming context on active channel
    const unsubContext = fdc3Agent.addContextListener<T>(contextType || null, (ctx) => {
      setCurrentContext(ctx);
      if (callbackRef.current) {
        callbackRef.current(ctx);
      }
    });

    return () => {
      unsubChannel();
      unsubContext();
    };
  }, [contextType]);

  // Safe broadcast function for user actions
  const broadcast = useCallback((context: Fdc3Context) => {
    fdc3Agent.broadcast(context);
  }, []);

  const setChannel = useCallback((channelId: UserChannelId) => {
    fdc3Agent.joinUserChannel(channelId);
  }, []);

  const raiseIntent = useCallback((intent: StandardIntent, context: Fdc3Context, targetViewId?: string): Promise<IntentResolution> => {
    return fdc3Agent.raiseIntent(intent, context, targetViewId);
  }, []);

  return {
    activeChannel,
    currentContext,
    broadcast,
    setChannel,
    raiseIntent,
  };
}

/**
 * Hook to register an internal intent handler with automatic mount/unmount lifecycle.
 * Handles React StrictMode mount/unmount/mount idempotently.
 */
export function useIntentHandler(
  intent: StandardIntent,
  viewId: string,
  title: string,
  handler: IntentHandler
) {
  const handlerRef = useRef(handler);
  handlerRef.current = handler;

  useEffect(() => {
    const unsub = fdc3Agent.registerIntentHandler(intent, viewId, title, (context) => {
      if (handlerRef.current) {
        return handlerRef.current(context);
      }
    });

    return () => {
      unsub();
    };
  }, [intent, viewId, title]);
}

