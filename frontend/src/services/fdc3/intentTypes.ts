import { Fdc3Context } from './types';

/**
 * Standard FDC3-compatible Financial Trading Intents.
 * 
 * Rationale: We intentionally use the canonical FINOS FDC3 intent names
 * (ViewInstrument, ViewChart, ViewOrders, ViewAnalysis, ViewExecution) to
 * establish immediate semantic consistency across trading views and preserve
 * a frictionless path for future external interop or FDC3 2.0 app directories.
 */
export const STANDARD_INTENTS = [
  'ViewInstrument',
  'ViewChart',
  'ViewOrders',
  'ViewAnalysis',
  'ViewExecution',
] as const;

export type StandardIntent = (typeof STANDARD_INTENTS)[number];

export function isValidIntent(intent: string): intent is StandardIntent {
  return STANDARD_INTENTS.includes(intent as StandardIntent);
}

/**
 * Descriptor for an active view capable of handling an intent.
 */
export interface IntentTarget {
  intent: StandardIntent;
  viewId: string;
  windowId: string;
  title: string;
  channelId?: string;
}

/**
 * Messages transmitted over the internal system intent channel (`__fdc3_system_intents__`).
 */
export interface IntentRegistrationMessage {
  type: 'fdc3.intent.registered';
  target: IntentTarget;
  timestamp: number;
}

export interface IntentUnregistrationMessage {
  type: 'fdc3.intent.unregistered';
  intent: StandardIntent;
  viewId: string;
  windowId: string;
  timestamp: number;
}

export interface IntentDiscoverMessage {
  type: 'fdc3.intent.discover';
  sourceWindowId: string;
  timestamp: number;
}

export interface IntentInvocationMessage {
  type: 'fdc3.intent.invocation';
  invocationId: string;
  intent: StandardIntent;
  context: Fdc3Context;
  targetViewId: string;
  targetWindowId: string;
  sourceWindowId: string;
  timestamp: number;
}

export interface IntentAckMessage {
  type: 'fdc3.intent.ack';
  invocationId: string;
  targetViewId: string;
  targetWindowId: string;
  success: boolean;
  error?: string;
  timestamp: number;
}

export type IntentSystemMessage =
  | IntentRegistrationMessage
  | IntentUnregistrationMessage
  | IntentDiscoverMessage
  | IntentInvocationMessage
  | IntentAckMessage;

export interface IntentResolution {
  source: string;
  intent: StandardIntent;
  target: IntentTarget;
  success: boolean;
  error?: string;
}

export type IntentHandler = (context: Fdc3Context) => void | Promise<void>;
