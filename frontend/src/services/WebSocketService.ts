import { WebSocketMessage, RealTimeSubscription as _RealTimeSubscription } from '../types/dynamic';
import { devError } from '../utils/devLogger';

// Local type definitions for WebSocket
type WebSocketMessageType =
  | 'subscribe'
  | 'unsubscribe'
  | 'heartbeat'
  | 'metric_update'
  | 'anomaly_alert'
  | 'dashboard_refresh'
  | 'parameter_change'
  | 'notification';

type RealTimeSubscriptionType = 'metric' | 'dashboard' | 'anomaly' | 'notification';

interface LocalWebSocketMessage {
  type: WebSocketMessageType;
  payload: any;
  timestamp: string;
}

interface LocalRealTimeSubscription {
  id: string;
  type: RealTimeSubscriptionType;
  filters: Record<string, any>;
  callback: (data: any) => void;
}

export class WebSocketService {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectInterval = 1000;
  private subscriptions: Map<string, LocalRealTimeSubscription> = new Map();
  private heartbeatInterval: NodeJS.Timeout | null = null;
  private authErrorCallbacks: Array<() => void> = [];

  /**
   * Note on query string ?ticket= authentication:
   * The browser WebSocket API does not allow setting custom Authorization headers during the handshake.
   * To bound exposure risk compared to long-lived JWTs, we exchange the user's session token for an
   * ephemeral 30-second single-use ticket via POST /api/ws/ticket.
   */
  constructor(private baseUrl: string) {}

  onAuthError(callback: () => void) {
    this.authErrorCallbacks.push(callback);
  }

  private triggerAuthError() {
    this.authErrorCallbacks.forEach((cb) => {
      try {
        cb();
      } catch (err) {
        devError('Error in onAuthError callback:', err);
      }
    });
  }

  /**
   * Fetches an ephemeral single-use ticket from POST /api/ws/ticket.
   * If an auth failure occurs (401), notifies onAuthError listeners and halts.
   */
  private async fetchTicket(): Promise<string> {
    const token = localStorage.getItem('auth_token');
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }

    const response = await fetch('/api/ws/ticket', {
      method: 'POST',
      headers,
    });

    if (response.status === 401) {
      this.triggerAuthError();
      throw new Error('Unauthorized');
    }

    if (!response.ok) {
      throw new Error(`Failed to acquire ticket: ${response.status} ${response.statusText}`);
    }

    const data = await response.json();
    if (!data.ticket) {
      throw new Error('Invalid ticket response: missing ticket string');
    }

    return data.ticket;
  }

  async connect(): Promise<void> {
    const ticket = await this.fetchTicket();
    const delimiter = this.baseUrl.includes('?') ? '&' : '?';
    const connectionUrl = `${this.baseUrl}${delimiter}ticket=${encodeURIComponent(ticket)}`;

    return new Promise((resolve, reject) => {
      try {
        this.ws = new WebSocket(connectionUrl);

        this.ws.onopen = () => {
          this.reconnectAttempts = 0;
          this.startHeartbeat();
          resolve();
        };

        this.ws.onmessage = (event) => {
          try {
            const message: WebSocketMessage = JSON.parse(event.data);
            this.handleMessage(message);
          } catch (error) {
            devError('Failed to parse WebSocket message:', error);
          }
        };

        this.ws.onclose = (event) => {
          this.stopHeartbeat();
          if (event && (event.code === 4001 || event.code === 401)) {
            this.triggerAuthError();
            return;
          }
          this.handleReconnect();
        };

        this.ws.onerror = (error) => {
          devError('WebSocket error:', error);
          reject(error);
        };

      } catch (error) {
        reject(error);
      }
    });
  }

  disconnect() {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
    this.stopHeartbeat();
    this.subscriptions.clear();
  }

  subscribe(subscription: LocalRealTimeSubscription) {
    this.subscriptions.set(subscription.id, subscription);

    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.send({
        type: 'subscribe',
        payload: {
          id: subscription.id,
          type: subscription.type,
          filters: subscription.filters
        },
        timestamp: new Date().toISOString()
      });
    }
  }

  unsubscribe(subscriptionId: string) {
    this.subscriptions.delete(subscriptionId);

    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.send({
        type: 'unsubscribe',
        payload: { id: subscriptionId },
        timestamp: new Date().toISOString()
      });
    }
  }

  private send(message: LocalWebSocketMessage) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    }
  }

  private handleMessage(message: LocalWebSocketMessage) {
    // Route message to appropriate subscription
    for (const subscription of this.subscriptions.values()) {
      if (this.shouldHandleMessage(subscription, message)) {
        subscription.callback(message.payload);
      }
    }
  }

  private shouldHandleMessage(subscription: LocalRealTimeSubscription, message: LocalWebSocketMessage): boolean {
    // Check if message type matches subscription type
    if (message.type === 'metric_update' && subscription.type === 'metric') {
      return this.matchesFilters(subscription.filters, message.payload);
    }
    if (message.type === 'anomaly_alert' && subscription.type === 'anomaly') {
      return this.matchesFilters(subscription.filters, message.payload);
    }
    if (message.type === 'dashboard_refresh' && subscription.type === 'dashboard') {
      return this.matchesFilters(subscription.filters, message.payload);
    }
    if (message.type === 'notification' && subscription.type === 'notification') {
      return this.matchesFilters(subscription.filters, message.payload);
    }
    return false;
  }

  private matchesFilters(filters: Record<string, any>, payload: any): boolean {
    for (const [key, value] of Object.entries(filters)) {
      if (payload[key] !== value) {
        return false;
      }
    }
    return true;
  }

  private startHeartbeat() {
    this.heartbeatInterval = setInterval(() => {
      this.send({
        type: 'heartbeat',
        payload: { timestamp: Date.now() },
        timestamp: new Date().toISOString()
      });
    }, 30000); // Send heartbeat every 30 seconds
  }

  private stopHeartbeat() {
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval);
      this.heartbeatInterval = null;
    }
  }

  private handleReconnect() {
    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      this.reconnectAttempts++;

      setTimeout(() => {
        this.connect().catch(() => {
          // Reconnection failed, will try again
        });
      }, this.reconnectInterval * this.reconnectAttempts);
    } else {
      devError('Max reconnection attempts reached');
    }
  }

  // Utility method to create metric subscription
  createMetricSubscription(
    metricIds: string[],
    callback: (data: any) => void,
    filters: Record<string, any> = {}
  ): LocalRealTimeSubscription {
    return {
      id: `metric-${Date.now()}-${Math.random()}`,
      type: 'metric',
      filters: { ...filters, metricIds },
      callback
    };
  }

  // Utility method to create dashboard subscription
  createDashboardSubscription(
    dashboardId: string,
    callback: (data: any) => void,
    filters: Record<string, any> = {}
  ): LocalRealTimeSubscription {
    return {
      id: `dashboard-${Date.now()}-${Math.random()}`,
      type: 'dashboard',
      filters: { ...filters, dashboardId },
      callback
    };
  }

  // Utility method to create anomaly subscription
  createAnomalySubscription(
    severity: string[],
    callback: (data: any) => void,
    filters: Record<string, any> = {}
  ): LocalRealTimeSubscription {
    return {
      id: `anomaly-${Date.now()}-${Math.random()}`,
      type: 'anomaly',
      filters: { ...filters, severity },
      callback
    };
  }

  // Utility method to create notification subscription
  createNotificationSubscription(
    callback: (data: any) => void,
    userId?: string,
    filters: Record<string, any> = {}
  ): LocalRealTimeSubscription {
    return {
      id: `notification-${Date.now()}-${Math.random()}`,
      type: 'notification',
      filters: { ...filters, ...(userId && { userId }) },
      callback
    };
  }
}

// Singleton instance
let websocketService: WebSocketService | null = null;

export const getWebSocketService = (url?: string): WebSocketService => {
  if (!websocketService) {
    const wsUrl = url || `ws://${window.location.host}/api/ws/updates`;
    websocketService = new WebSocketService(wsUrl);
  }
  return websocketService;
};

export default WebSocketService;
