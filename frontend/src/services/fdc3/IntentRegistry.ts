import {
  IntentAckMessage,
  IntentDiscoverMessage,
  IntentHandler,
  IntentInvocationMessage,
  IntentRegistrationMessage,
  IntentResolution,
  IntentSystemMessage,
  IntentTarget,
  IntentUnregistrationMessage,
  isValidIntent,
  STANDARD_INTENTS,
  StandardIntent,
} from './intentTypes';
import { ChannelMessageEnvelope, Fdc3Context, IFdc3Transport } from './types';
import { devLog, devWarn } from '../../utils/devLogger';

export const INTENT_SYSTEM_CHANNEL = '__fdc3_system_intents__';
const ACK_TIMEOUT_MS = 1500;

export class AckTimeoutError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'AckTimeoutError';
  }
}

/**
 * Cross-window Intent Registry & Router.
 * 
 * Manages live handler discovery across all windows via the underlying FDC3 transport.
 * 
 * Architectural Invariants:
 * 1. Derived Strictly from Live Mesh: The registry is never persisted to localStorage.
 *    On startup, windows broadcast `fdc3.intent.discover` and merge live responses.
 * 2. Eventual Consistency: Target discovery is eventually consistent over ~1 second
 *    after window launch while discovery packets transit the transport.
 * 3. Dead-Handler Purging: If an intent target fails to acknowledge invocation within
 *    1.5 seconds (e.g. crashed or force-quit window), it is automatically purged
 *    from the mesh registry and resolution re-tries with remaining live targets.
 * 4. Loop-Guard Corollary: An intent handler MUST NOT raise another intent.
 * 5. Closed Vocabulary: Only the 5 standard intents are accepted.
 */
export class IntentRegistry {
  private transport: IFdc3Transport;
  private currentWindowId: string;
  private localHandlers: Map<string, { target: IntentTarget; handler: IntentHandler }> = new Map();
  private meshTargets: Map<string, IntentTarget> = new Map();
  private pendingAcks: Map<
    string,
    { resolve: (ack: IntentAckMessage) => void; reject: (err: Error) => void; timer: NodeJS.Timeout }
  > = new Map();
  private isExecutingIntent: boolean = false;
  private resolverPrompt?: (intent: StandardIntent, targets: IntentTarget[], context?: Fdc3Context) => Promise<IntentTarget>;
  private unsubscribeTransport?: () => void;

  constructor(currentWindowId: string, transport: IFdc3Transport) {
    this.currentWindowId = currentWindowId;
    this.transport = transport;
    this.bindTransport();

    // Broadcast discovery request to discover existing live handlers across the mesh
    this.broadcastDiscover();
  }

  public setTransport(transport: IFdc3Transport): void {
    this.transport = transport;
    this.bindTransport();
    this.broadcastDiscover();
  }

  public setResolverPrompt(
    prompt: (intent: StandardIntent, targets: IntentTarget[], context?: Fdc3Context) => Promise<IntentTarget>
  ): void {
    this.resolverPrompt = prompt;
  }

  private targetKey(intent: string, windowId: string, viewId: string): string {
    return `${intent}:${windowId}:${viewId}`;
  }

  private bindTransport(): void {
    if (this.unsubscribeTransport) {
      this.unsubscribeTransport();
    }

    this.unsubscribeTransport = this.transport.subscribe(
      INTENT_SYSTEM_CHANNEL,
      (envelope: ChannelMessageEnvelope<any>) => {
        const msg = envelope.context as IntentSystemMessage;
        if (!msg || !msg.type) return;

        switch (msg.type) {
          case 'fdc3.intent.registered': {
            const t = msg.target;
            const key = this.targetKey(t.intent, t.windowId, t.viewId);
            this.meshTargets.set(key, t);
            break;
          }
          case 'fdc3.intent.unregistered': {
            const key = this.targetKey(msg.intent, msg.windowId, msg.viewId);
            this.meshTargets.delete(key);
            break;
          }
          case 'fdc3.intent.discover': {
            // Re-announce all local handlers so the newcomer learns of us
            if (msg.sourceWindowId !== this.currentWindowId) {
              this.announceAllLocalHandlers();
            }
            break;
          }
          case 'fdc3.intent.invocation': {
            // Check if this window is the designated target
            if (msg.targetWindowId === this.currentWindowId) {
              this.handleIncomingInvocation(msg);
            }
            break;
          }
          case 'fdc3.intent.ack': {
            const pending = this.pendingAcks.get(msg.invocationId);
            if (pending) {
              clearTimeout(pending.timer);
              this.pendingAcks.delete(msg.invocationId);
              pending.resolve(msg);
            }
            break;
          }
        }
      }
    );
  }

  private broadcastDiscover(): void {
    const msg: IntentDiscoverMessage = {
      type: 'fdc3.intent.discover',
      sourceWindowId: this.currentWindowId,
      timestamp: Date.now(),
    };
    this.transport.broadcast({
      sourceWindowId: this.currentWindowId,
      channelId: INTENT_SYSTEM_CHANNEL,
      context: msg as unknown as Fdc3Context,
      timestamp: Date.now(),
    });
  }

  private announceAllLocalHandlers(): void {
    this.localHandlers.forEach(({ target }) => {
      const msg: IntentRegistrationMessage = {
        type: 'fdc3.intent.registered',
        target,
        timestamp: Date.now(),
      };
      this.transport.broadcast({
        sourceWindowId: this.currentWindowId,
        channelId: INTENT_SYSTEM_CHANNEL,
        context: msg as unknown as Fdc3Context,
        timestamp: Date.now(),
      });
    });
  }

  /**
   * Registers a local intent handler in this window.
   * Emits registration over the transport so other windows discover this target.
   */
  public registerIntentHandler(
    intent: string,
    viewId: string,
    title: string,
    handler: IntentHandler
  ): () => void {
    if (!isValidIntent(intent)) {
      throw new Error(
        `[IntentRegistry] Unknown intent "${intent}". Expected one of: ${STANDARD_INTENTS.join(', ')}`
      );
    }

    const target: IntentTarget = {
      intent,
      viewId,
      windowId: this.currentWindowId,
      title,
    };

    const key = this.targetKey(intent, this.currentWindowId, viewId);
    this.localHandlers.set(key, { target, handler });
    this.meshTargets.set(key, target);

    // Announce to mesh
    const msg: IntentRegistrationMessage = {
      type: 'fdc3.intent.registered',
      target,
      timestamp: Date.now(),
    };
    this.transport.broadcast({
      sourceWindowId: this.currentWindowId,
      channelId: INTENT_SYSTEM_CHANNEL,
      context: msg as unknown as Fdc3Context,
      timestamp: Date.now(),
    });

    devLog(`[IntentRegistry] Registered handler for intent: ${intent} (${viewId})`);

    // Return unregister function
    return () => {
      this.localHandlers.delete(key);
      this.meshTargets.delete(key);

      const unregMsg: IntentUnregistrationMessage = {
        type: 'fdc3.intent.unregistered',
        intent,
        viewId,
        windowId: this.currentWindowId,
        timestamp: Date.now(),
      };
      this.transport.broadcast({
        sourceWindowId: this.currentWindowId,
        channelId: INTENT_SYSTEM_CHANNEL,
        context: unregMsg as unknown as Fdc3Context,
        timestamp: Date.now(),
      });
      devLog(`[IntentRegistry] Unregistered handler for intent: ${intent} (${viewId})`);
    };
  }

  /**
   * Returns all active targets registered across the mesh for a given intent.
   */
  public findIntentTargets(intent: string): IntentTarget[] {
    if (!isValidIntent(intent)) {
      return [];
    }
    const matches: IntentTarget[] = [];
    this.meshTargets.forEach((t) => {
      if (t.intent === intent) {
        matches.push(t);
      }
    });
    return matches;
  }

  /**
   * Raises an intent across the workstation mesh.
   * 
   * Routing Logic:
   * 1. If explicit `targetViewId` provided: routes directly to that target.
   * 2. If exactly 1 matching target: routes directly without modal interruption.
   * 3. If >1 matching targets: prompts user via `resolverPrompt` modal.
   * 4. If 0 matching targets: rejects with descriptive error.
   * 5. If target times out on ack (dead handler): purges target and re-resolves.
   */
  public async raiseIntent(
    intent: string,
    context: Fdc3Context,
    targetViewId?: string
  ): Promise<IntentResolution> {
    if (this.isExecutingIntent) {
      throw new Error(
        `[IntentRegistry Loop Guard] Intent handler is currently executing. Raising another intent is forbidden.`
      );
    }

    if (!isValidIntent(intent)) {
      throw new Error(
        `[IntentRegistry] Unknown intent "${intent}". Expected one of: ${STANDARD_INTENTS.join(', ')}`
      );
    }

    return this.executeResolutionLoop(intent, context, targetViewId);
  }

  private async executeResolutionLoop(
    intent: StandardIntent,
    context: Fdc3Context,
    targetViewId?: string
  ): Promise<IntentResolution> {
    const targets = this.findIntentTargets(intent);

    if (targets.length === 0) {
      throw new Error(`No active handlers found for intent: "${intent}"`);
    }

    let selectedTarget: IntentTarget | undefined;

    if (targetViewId) {
      selectedTarget = targets.find((t) => t.viewId === targetViewId);
      if (!selectedTarget) {
        throw new Error(
          `Target view "${targetViewId}" is not currently registered for intent "${intent}"`
        );
      }
    } else if (targets.length === 1) {
      // Direct single-target route (no modal interruption)
      selectedTarget = targets[0];
    } else {
      // Ambiguous multiple targets: trigger resolver modal
      if (!this.resolverPrompt) {
        // Default to first target if no UI modal registered
        selectedTarget = targets[0];
      } else {
        selectedTarget = await this.resolverPrompt(intent, targets, context);
      }
    }

    if (!selectedTarget) {
      throw new Error(`Intent resolution cancelled by user`);
    }

    // Execute invocation with dead-handler detection & ack timeout
    try {
      return await this.dispatchInvocation(selectedTarget, intent, context);
    } catch (err) {
      if (err instanceof AckTimeoutError) {
        devWarn(
          `[IntentRegistry] Target ${selectedTarget.title} timed out. Purging dead target...`
        );
        const key = this.targetKey(selectedTarget.intent, selectedTarget.windowId, selectedTarget.viewId);
        this.meshTargets.delete(key);

        // If caller explicitly asked for targetViewId, reject since that specific target died
        if (targetViewId) {
          throw err;
        }

        // Otherwise (general resolution), retry with remaining targets
        devWarn(`[IntentRegistry] Re-resolving ${intent} with remaining targets...`);
        return this.executeResolutionLoop(intent, context, undefined);
      }
      // Re-throw any other error (loop guard, local handler exception, etc.)
      throw err;
    }
  }

  private async dispatchInvocation(
    target: IntentTarget,
    intent: StandardIntent,
    context: Fdc3Context
  ): Promise<IntentResolution> {
    // 1. In-Window Local Execution
    if (target.windowId === this.currentWindowId) {
      const key = this.targetKey(target.intent, target.windowId, target.viewId);
      const local = this.localHandlers.get(key);
      if (!local) {
        throw new Error(`Local handler for ${key} no longer exists`);
      }

      this.isExecutingIntent = true;
      try {
        await local.handler(context);
      } finally {
        this.isExecutingIntent = false;
      }

      return {
        source: this.currentWindowId,
        intent,
        target,
        success: true,
      };
    }

    // 2. Cross-Window Remote Execution
    const invocationId = `inv_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`;
    const msg: IntentInvocationMessage = {
      type: 'fdc3.intent.invocation',
      invocationId,
      intent,
      context,
      targetViewId: target.viewId,
      targetWindowId: target.windowId,
      sourceWindowId: this.currentWindowId,
      timestamp: Date.now(),
    };

    const ackPromise = new Promise<IntentAckMessage>((resolve, reject) => {
      const timer = setTimeout(() => {
        this.pendingAcks.delete(invocationId);
        reject(
          new AckTimeoutError(
            `Target ${target.title} (${target.windowId}) failed to acknowledge intent invocation within ${ACK_TIMEOUT_MS}ms`
          )
        );
      }, ACK_TIMEOUT_MS);

      this.pendingAcks.set(invocationId, { resolve, reject, timer });
    });

    this.transport.broadcast({
      sourceWindowId: this.currentWindowId,
      channelId: INTENT_SYSTEM_CHANNEL,
      context: msg as unknown as Fdc3Context,
      timestamp: Date.now(),
    });

    const ack = await ackPromise;
    if (!ack.success) {
      throw new Error(ack.error || `Target execution failed`);
    }

    return {
      source: this.currentWindowId,
      intent,
      target,
      success: true,
    };
  }

  private async handleIncomingInvocation(msg: IntentInvocationMessage): Promise<void> {
    const key = this.targetKey(msg.intent, this.currentWindowId, msg.targetViewId);
    const local = this.localHandlers.get(key);

    let success = false;
    let error: string | undefined;

    if (!local) {
      error = `No handler registered for ${key} in window ${this.currentWindowId}`;
    } else {
      this.isExecutingIntent = true;
      try {
        await local.handler(msg.context);
        success = true;
      } catch (err: any) {
        error = err?.message || String(err);
      } finally {
        this.isExecutingIntent = false;
      }
    }

    // Send Ack back to source window
    const ackMsg: IntentAckMessage = {
      type: 'fdc3.intent.ack',
      invocationId: msg.invocationId,
      targetViewId: msg.targetViewId,
      targetWindowId: this.currentWindowId,
      success,
      error,
      timestamp: Date.now(),
    };

    this.transport.broadcast({
      sourceWindowId: this.currentWindowId,
      channelId: INTENT_SYSTEM_CHANNEL,
      context: ackMsg as unknown as Fdc3Context,
      timestamp: Date.now(),
    });
  }

  public destroy(): void {
    this.localHandlers.clear();
    this.meshTargets.clear();
    this.pendingAcks.forEach(({ timer, reject }) => {
      clearTimeout(timer);
      reject(new Error('IntentRegistry destroyed'));
    });
    this.pendingAcks.clear();
    if (this.unsubscribeTransport) {
      this.unsubscribeTransport();
    }
  }
}
