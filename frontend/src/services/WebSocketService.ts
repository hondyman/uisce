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

class WebSocketService {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectInterval = 1000;
  private subscriptions: Map<string, LocalRealTimeSubscription> = new Map();
  private heartbeatInterval: NodeJS.Timeout | null = null;

  constructor(private url: string) {}

  connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      try {
        this.ws = new WebSocket(this.url);

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

        this.ws.onclose = () => {
          this.stopHeartbeat();
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
