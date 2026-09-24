/**
 * FINOS FDC3-compatible Context & Channel Type Definitions.
 * 
 * Implements context shapes aligned with the FDC3 standard without proprietary
 * OpenFin dependencies, usable across both browser tabs and Wails multi-window setups.
 */

export interface Fdc3Context {
  type: string;
  name?: string;
  id?: Record<string, string | undefined>;
  [key: string]: unknown;
}

export interface Fdc3InstrumentContext extends Fdc3Context {
  type: 'fdc3.instrument';
  name?: string;
  id: {
    ticker?: string;
    ISIN?: string;
    RIC?: string;
    CUSIP?: string;
    SEDOL?: string;
    [key: string]: string | undefined;
  };
  market?: {
    MIC?: string;
    name?: string;
  };
}

export interface Fdc3PortfolioContext extends Fdc3Context {
  type: 'fdc3.portfolio';
  name?: string;
  id: {
    portfolioId?: string;
    accountNumber?: string;
    [key: string]: string | undefined;
  };
  accounts?: string[];
}

export interface Fdc3OrderContext extends Fdc3Context {
  type: 'fdc3.order';
  name?: string;
  id: {
    orderId: string;
    clOrdId?: string;
    [key: string]: string | undefined;
  };
  details?: {
    symbol?: string;
    side?: 'BUY' | 'SELL';
    quantity?: number;
    limitPrice?: number;
    status?: string;
    currency?: string;
  };
}

export type UserChannelId =
  | 'red'
  | 'orange'
  | 'yellow'
  | 'green'
  | 'blue'
  | 'purple'
  | 'cyan'
  | 'pink';

export interface ChannelDescriptor {
  id: UserChannelId;
  name: string;
  color: string;
}

export const USER_CHANNELS: ChannelDescriptor[] = [
  { id: 'red', name: 'Channel 1 (Red)', color: '#ef4444' },
  { id: 'orange', name: 'Channel 2 (Orange)', color: '#f97316' },
  { id: 'yellow', name: 'Channel 3 (Yellow)', color: '#eab308' },
  { id: 'green', name: 'Channel 4 (Green)', color: '#22c55e' },
  { id: 'blue', name: 'Channel 5 (Blue)', color: '#3b82f6' },
  { id: 'purple', name: 'Channel 6 (Purple)', color: '#a855f7' },
  { id: 'cyan', name: 'Channel 7 (Cyan)', color: '#06b6d4' },
  { id: 'pink', name: 'Channel 8 (Pink)', color: '#ec4899' },
];

export interface ChannelMessageEnvelope<T extends Fdc3Context = Fdc3Context> {
  sourceWindowId: string;
  channelId: string;
  context: T;
  timestamp: number;
}

export type ContextHandler<T extends Fdc3Context = Fdc3Context> = (context: T) => void;

export interface IFdc3Transport {
  readonly name: string;
  broadcast(envelope: ChannelMessageEnvelope): void;
  subscribe(channelId: string, onMessage: (envelope: ChannelMessageEnvelope) => void): () => void;
  destroy?(): void;
}
