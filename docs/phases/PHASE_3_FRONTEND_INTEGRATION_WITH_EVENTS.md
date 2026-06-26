# Phase 3: Frontend Service Integration with Real-Time Catalog Sync

## Overview

This guide covers implementing the frontend service layer that connects to the backend API endpoints catalog and subscribes to real-time catalog node/edge updates via WebSocket and RabbitMQ events.

## Architecture

### Component Stack

```
┌─────────────────────────────────────────────────────────┐
│          EntityDetailsPage Component (React)             │
│                                                           │
│  ├─ useEffect: Initialize services                       │
│  ├─ useEffect: Subscribe to catalog updates              │
│  ├─ State: rules, nodes, edges, loading, error          │
│  └─ Render: ValidationRulesContainer                     │
└─────────────────────────────────────────────────────────┘
           ↑ ↓ (calls methods)
┌─────────────────────────────────────────────────────────┐
│        ValidationRulesService (TypeScript Class)          │
│                                                           │
│  ├─ constructor(tenantScope, httpClient)               │
│  ├─ listRules(filters, pagination)                      │
│  ├─ createRule(rule)                                    │
│  ├─ updateRule(id, updates)                             │
│  ├─ deleteRule(id)                                      │
│  ├─ executeRule(id, input)                              │
│  └─ subscribeToUpdates(callback)                        │
└─────────────────────────────────────────────────────────┘
           ↑ ↓ (HTTP + WebSocket)
┌─────────────────────────────────────────────────────────┐
│        CatalogSyncService (TypeScript Class)             │
│                                                           │
│  ├─ constructor(wsUrl)                                  │
│  ├─ connect()                                           │
│  ├─ disconnect()                                        │
│  ├─ onNodeCreated(callback)                             │
│  ├─ onNodeUpdated(callback)                             │
│  ├─ onNodeDeleted(callback)                             │
│  ├─ onEdgeCreated(callback)                             │
│  └─ onEdgeDeleted(callback)                             │
└─────────────────────────────────────────────────────────┘
           ↑ ↓ (WebSocket connection)
┌─────────────────────────────────────────────────────────┐
│              Backend WebSocket Server                     │
│                                                           │
│  ├─ RabbitMQ Consumer                                   │
│  │  ├─ Listens for CatalogNodeCreated events           │
│  │  ├─ Listens for CatalogNodeUpdated events           │
│  │  ├─ Listens for CatalogNodeDeleted events           │
│  │  ├─ Listens for CatalogEdgeCreated events           │
│  │  └─ Listens for CatalogEdgeDeleted events           │
│  │                                                      │
│  └─ WebSocket Broadcast                                │
│     └─ Forward to connected clients (tenant-scoped)    │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

## Implementation Files

### 1. CatalogSyncService (New)

File: `frontend/src/services/catalogSyncService.ts`

```typescript
import { EventEmitter } from 'eventemitter3';

export interface CatalogNode {
  id: string;
  tenant_id: string;
  node_type: 'api_endpoint' | 'entity' | 'datasource';
  metadata: Record<string, any>;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface CatalogEdge {
  id: string;
  tenant_id: string;
  source_node_id: string;
  target_node_id: string;
  relationship_type: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface CatalogEvent {
  event_id: string;
  event_type: string;
  tenant_id: string;
  timestamp: string;
  payload: CatalogNode | CatalogEdge;
}

export class CatalogSyncService extends EventEmitter {
  private ws: WebSocket | null = null;
  private url: string;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000;
  private isIntentionallyClosed = false;
  private tenantScope: {
    tenant_id: string;
    datasource_id?: string;
  };

  constructor(wsUrl: string, tenantScope: { tenant_id: string; datasource_id?: string }) {
    super();
    this.url = wsUrl;
    this.tenantScope = tenantScope;
  }

  /**
   * Connect to WebSocket server
   */
  connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      try {
        const url = new URL(this.url);
        // Add tenant scope as query parameters
        url.searchParams.append('tenant_id', this.tenantScope.tenant_id);
        if (this.tenantScope.datasource_id) {
          url.searchParams.append('datasource_id', this.tenantScope.datasource_id);
        }

        this.ws = new WebSocket(url.toString());

        this.ws.onopen = () => {
          console.log('[CatalogSync] Connected to WebSocket');
          this.reconnectAttempts = 0;
          this.emit('connected');
          resolve();
        };

        this.ws.onmessage = (event) => {
          this.handleMessage(event.data);
        };

        this.ws.onerror = (error) => {
          console.error('[CatalogSync] WebSocket error:', error);
          this.emit('error', error);
          reject(error);
        };

        this.ws.onclose = () => {
          console.log('[CatalogSync] WebSocket closed');
          this.emit('disconnected');
          
          if (!this.isIntentionallyClosed) {
            this.attemptReconnect();
          }
        };
      } catch (error) {
        reject(error);
      }
    });
  }

  /**
   * Disconnect from WebSocket
   */
  disconnect(): void {
    this.isIntentionallyClosed = true;
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  /**
   * Handle incoming WebSocket messages
   */
  private handleMessage(data: string): void {
    try {
      const event: CatalogEvent = JSON.parse(data);

      // Verify tenant scope
      if (event.tenant_id !== this.tenantScope.tenant_id) {
        console.warn('[CatalogSync] Received event for different tenant, ignoring');
        return;
      }

      // Route event to appropriate handler
      switch (event.event_type) {
        case 'catalog.node.created':
          this.emit('node:created', event.payload as CatalogNode);
          break;
        case 'catalog.node.updated':
          this.emit('node:updated', event.payload as CatalogNode);
          break;
        case 'catalog.node.deleted':
          this.emit('node:deleted', event.payload as CatalogNode);
          break;
        case 'catalog.edge.created':
          this.emit('edge:created', event.payload as CatalogEdge);
          break;
        case 'catalog.edge.deleted':
          this.emit('edge:deleted', event.payload as CatalogEdge);
          break;
        default:
          console.warn('[CatalogSync] Unknown event type:', event.event_type);
      }
    } catch (error) {
      console.error('[CatalogSync] Failed to parse message:', error);
    }
  }

  /**
   * Attempt to reconnect with exponential backoff
   */
  private attemptReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('[CatalogSync] Max reconnect attempts reached');
      this.emit('connection_failed');
      return;
    }

    this.reconnectAttempts++;
    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);
    
    console.log(`[CatalogSync] Attempting reconnect in ${delay}ms (attempt ${this.reconnectAttempts})`);
    
    setTimeout(() => {
      this.connect().catch((error) => {
        console.error('[CatalogSync] Reconnect failed:', error);
      });
    }, delay);
  }

  /**
   * Subscribe to node created events
   */
  onNodeCreated(callback: (node: CatalogNode) => void): void {
    this.on('node:created', callback);
  }

  /**
   * Subscribe to node updated events
   */
  onNodeUpdated(callback: (node: CatalogNode) => void): void {
    this.on('node:updated', callback);
  }

  /**
   * Subscribe to node deleted events
   */
  onNodeDeleted(callback: (node: CatalogNode) => void): void {
    this.on('node:deleted', callback);
  }

  /**
   * Subscribe to edge created events
   */
  onEdgeCreated(callback: (edge: CatalogEdge) => void): void {
    this.on('edge:created', callback);
  }

  /**
   * Subscribe to edge deleted events
   */
  onEdgeDeleted(callback: (edge: CatalogEdge) => void): void {
    this.on('edge:deleted', callback);
  }

  /**
   * Check if connected
   */
  isConnected(): boolean {
    return this.ws !== null && this.ws.readyState === WebSocket.OPEN;
  }

  /**
   * Get connection status
   */
  getStatus(): 'connected' | 'connecting' | 'disconnected' | 'error' {
    if (!this.ws) return 'disconnected';
    if (this.ws.readyState === WebSocket.OPEN) return 'connected';
    if (this.ws.readyState === WebSocket.CONNECTING) return 'connecting';
    return 'error';
  }
}
```

### 2. ValidationRulesService Enhancement

File: `frontend/src/services/validationRulesService.ts` (Add event support)

```typescript
// Add to existing ValidationRulesService class

export class ValidationRulesService {
  // ... existing code ...

  private catalogSyncService: CatalogSyncService;
  private catalogSyncCallbacks: Map<string, Function> = new Map();

  constructor(
    httpClient: AxiosInstance,
    tenantScope: { tenant_id: string; datasource_id?: string },
    wsUrl: string = 'ws://localhost:8080/catalog-sync'
  ) {
    this.httpClient = httpClient;
    this.tenantScope = tenantScope;
    
    // Initialize catalog sync service
    this.catalogSyncService = new CatalogSyncService(wsUrl, tenantScope);
    
    // Setup event listeners
    this.catalogSyncService.onNodeCreated((node) => {
      this.notifyObservers('nodeCreated', node);
    });

    this.catalogSyncService.onNodeUpdated((node) => {
      this.notifyObservers('nodeUpdated', node);
    });

    this.catalogSyncService.onNodeDeleted((node) => {
      this.notifyObservers('nodeDeleted', node);
    });

    this.catalogSyncService.onEdgeCreated((edge) => {
      this.notifyObservers('edgeCreated', edge);
    });

    this.catalogSyncService.onEdgeDeleted((edge) => {
      this.notifyObservers('edgeDeleted', edge);
    });
  }

  /**
   * Connect to catalog sync service
   */
  async connectCatalogSync(): Promise<void> {
    try {
      await this.catalogSyncService.connect();
      console.log('[ValidationRulesService] Connected to catalog sync');
    } catch (error) {
      console.error('[ValidationRulesService] Failed to connect to catalog sync:', error);
      throw error;
    }
  }

  /**
   * Disconnect from catalog sync service
   */
  disconnectCatalogSync(): void {
    this.catalogSyncService.disconnect();
  }

  /**
   * Subscribe to catalog updates
   */
  onCatalogUpdate(event: string, callback: Function): void {
    this.catalogSyncCallbacks.set(`${event}-${Math.random()}`, callback);
  }

  /**
   * Notify observers of catalog changes
   */
  private notifyObservers(event: string, data: any): void {
    this.catalogSyncCallbacks.forEach((callback) => {
      if (callback) {
        try {
          callback(data);
        } catch (error) {
          console.error('[ValidationRulesService] Error in observer:', error);
        }
      }
    });
  }

  /**
   * Cleanup resources
   */
  destroy(): void {
    this.catalogSyncService.disconnect();
    this.catalogSyncCallbacks.clear();
  }
}
```

### 3. Updated EntityDetailsPage Component

File: `frontend/src/pages/EntityDetailsPage.tsx`

```typescript
import React, { useEffect, useState, useRef } from 'react';
import { Alert, Spin, message } from 'antd';
import { TenantContext } from '../context/TenantContext';
import { ValidationRulesContainer } from '../components/ValidationRulesContainer';
import { ValidationRulesService } from '../services/validationRulesService';
import type { CatalogNode, CatalogEdge } from '../services/catalogSyncService';
import styles from './EntityDetailsPage.module.css';

export const EntityDetailsPage: React.FC<{ entityId: string }> = ({ entityId }) => {
  const tenantScope = TenantContext.getCurrentScope();
  const [validationService, setValidationService] = useState<ValidationRulesService | null>(null);
  const [rules, setRules] = useState([]);
  const [catalogNodes, setCatalogNodes] = useState<Map<string, CatalogNode>>(new Map());
  const [catalogEdges, setCatalogEdges] = useState<Map<string, CatalogEdge>>(new Map());
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [syncStatus, setSyncStatus] = useState<'connected' | 'disconnected' | 'error'>('disconnected');
  const serviceRef = useRef<ValidationRulesService | null>(null);

  // Initialize service and catalog sync
  useEffect(() => {
    if (!tenantScope) {
      setError('Tenant scope not selected. Please select a tenant first.');
      return;
    }

    const initializeService = async () => {
      try {
        setLoading(true);
        setError(null);

        // Create service instance
        const service = new ValidationRulesService(
          httpClient,
          tenantScope,
          `ws://${window.location.hostname}:8080/catalog-sync`
        );

        // Connect to catalog sync
        try {
          await service.connectCatalogSync();
          setSyncStatus('connected');
        } catch (syncError) {
          console.warn('Failed to connect catalog sync:', syncError);
          setSyncStatus('error');
          // Continue anyway - API endpoints still work
        }

        // Subscribe to catalog updates
        service.onCatalogUpdate('nodeCreated', (node: CatalogNode) => {
          setCatalogNodes((prev) => new Map(prev).set(node.id, node));
          message.success(`New endpoint added: ${node.metadata?.endpoint_name}`);
        });

        service.onCatalogUpdate('nodeUpdated', (node: CatalogNode) => {
          setCatalogNodes((prev) => new Map(prev).set(node.id, node));
          message.info(`Endpoint updated: ${node.metadata?.endpoint_name}`);
        });

        service.onCatalogUpdate('nodeDeleted', (node: CatalogNode) => {
          setCatalogNodes((prev) => {
            const updated = new Map(prev);
            updated.delete(node.id);
            return updated;
          });
          message.warning(`Endpoint removed: ${node.metadata?.endpoint_name}`);
        });

        service.onCatalogUpdate('edgeCreated', (edge: CatalogEdge) => {
          setCatalogEdges((prev) => new Map(prev).set(edge.id, edge));
          message.success(`New relationship created`);
        });

        service.onCatalogUpdate('edgeDeleted', (edge: CatalogEdge) => {
          setCatalogEdges((prev) => {
            const updated = new Map(prev);
            updated.delete(edge.id);
            return updated;
          });
          message.info(`Relationship removed`);
        });

        serviceRef.current = service;
        setValidationService(service);

        // Load initial rules
        const initialRules = await service.listRules({
          entityId,
          tenantId: tenantScope.tenant_id,
        });
        setRules(initialRules);
      } catch (err) {
        setError(
          err instanceof Error ? err.message : 'Failed to initialize validation rules service'
        );
        setSyncStatus('error');
      } finally {
        setLoading(false);
      }
    };

    initializeService();

    return () => {
      // Cleanup on unmount
      if (serviceRef.current) {
        serviceRef.current.disconnectCatalogSync();
        serviceRef.current.destroy();
      }
    };
  }, [tenantScope, entityId]);

  if (!tenantScope) {
    return (
      <Alert
        message="Tenant Selection Required"
        description="Please select a tenant, product, and datasource before viewing validation rules."
        type="warning"
        showIcon
        className={styles.alert}
      />
    );
  }

  if (error) {
    return (
      <Alert
        message="Error Loading Validation Rules"
        description={error}
        type="error"
        showIcon
        closable
        onClose={() => setError(null)}
        className={styles.alert}
      />
    );
  }

  return (
    <div className={styles.validationRulesContainer}>
      {syncStatus === 'error' && (
        <Alert
          message="Catalog Sync Disconnected"
          description="Real-time updates are unavailable, but API is working normally."
          type="warning"
          showIcon
          className={styles.alert}
        />
      )}

      <Spin spinning={loading} tip="Loading validation rules...">
        {validationService && (
          <ValidationRulesContainer
            service={validationService}
            rules={rules}
            catalogNodes={Array.from(catalogNodes.values())}
            catalogEdges={Array.from(catalogEdges.values())}
            onRulesChange={(updatedRules) => setRules(updatedRules)}
            syncStatus={syncStatus}
          />
        )}
      </Spin>
    </div>
  );
};
```

## Backend WebSocket Implementation

Create `backend/internal/api/catalog_websocket.go`:

```go
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/hondyman/semlayer/backend/internal/events"
)

// CatalogWebSocketHub manages WebSocket connections for catalog updates
type CatalogWebSocketHub struct {
	clients      map[string]map[*CatalogWebSocketClient]bool // tenant_id -> clients
	broadcast    chan *events.CatalogEvent
	register     chan *CatalogWebSocketClient
	unregister   chan *CatalogWebSocketClient
	mu           sync.RWMutex
	eventConsumer *events.RabbitMQConsumer
}

// CatalogWebSocketClient represents a WebSocket client connection
type CatalogWebSocketClient struct {
	hub      *CatalogWebSocketHub
	conn     *websocket.Conn
	send     chan *events.CatalogEvent
	tenantID string
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins (configure in production)
	},
}

// NewCatalogWebSocketHub creates a new WebSocket hub
func NewCatalogWebSocketHub(consumer *events.RabbitMQConsumer) *CatalogWebSocketHub {
	return &CatalogWebSocketHub{
		clients:       make(map[string]map[*CatalogWebSocketClient]bool),
		broadcast:     make(chan *events.CatalogEvent, 256),
		register:      make(chan *CatalogWebSocketClient),
		unregister:    make(chan *CatalogWebSocketClient),
		eventConsumer: consumer,
	}
}

// Run starts the WebSocket hub
func (h *CatalogWebSocketHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case event := <-h.broadcast:
			h.broadcastEvent(event)
		}
	}
}

// registerClient registers a new WebSocket client
func (h *CatalogWebSocketHub) registerClient(client *CatalogWebSocketClient) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client.tenantID]; !ok {
		h.clients[client.tenantID] = make(map[*CatalogWebSocketClient]bool)
	}

	h.clients[client.tenantID][client] = true
	log.Printf("[CatalogWebSocket] Client connected for tenant %s", client.tenantID)
}

// unregisterClient unregisters a WebSocket client
func (h *CatalogWebSocketHub) unregisterClient(client *CatalogWebSocketClient) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clientMap, ok := h.clients[client.tenantID]; ok {
		if _, exists := clientMap[client]; exists {
			delete(clientMap, client)
			close(client.send)

			if len(clientMap) == 0 {
				delete(h.clients, client.tenantID)
			}
		}
	}

	log.Printf("[CatalogWebSocket] Client disconnected for tenant %s", client.tenantID)
}

// broadcastEvent broadcasts an event to all clients for the tenant
func (h *CatalogWebSocketHub) broadcastEvent(event *events.CatalogEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, ok := h.clients[event.TenantID]; ok {
		for client := range clients {
			select {
			case client.send <- event:
			default:
				// Client's send channel is full, skip
				log.Printf("[CatalogWebSocket] Client send channel full for tenant %s", event.TenantID)
			}
		}
	}
}

// HandleWebSocketConnection handles new WebSocket connections
func (h *CatalogWebSocketHub) HandleWebSocketConnection(w http.ResponseWriter, r *http.Request) {
	// Extract tenant_id from query parameters
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		http.Error(w, "tenant_id required", http.StatusBadRequest)
		return;
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[CatalogWebSocket] WebSocket upgrade error: %v", err)
		return
	}

	// Create client
	client := &CatalogWebSocketClient{
		hub:      h,
		conn:     conn,
		send:     make(chan *events.CatalogEvent, 256),
		tenantID: tenantID,
	}

	// Register client
	h.register <- client

	// Start handling client messages and sending events
	go client.readPump()
	go client.writePump()
}

// readPump reads messages from the WebSocket connection (heartbeat/ping)
func (c *CatalogWebSocketClient) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[CatalogWebSocket] Unexpected close error: %v", err)
			}
			break
		}

		// Handle ping/keepalive messages
		if string(message) == "ping" {
			if err := c.conn.WriteMessage(websocket.TextMessage, []byte("pong")); err != nil {
				break
			}
		}
	}
}

// writePump writes messages to the WebSocket connection
func (c *CatalogWebSocketClient) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case event := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

			if err := c.conn.WriteJSON(event); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
```

## Deployment & Testing

### Testing Event Flow

```bash
# 1. Start RabbitMQ
docker run -d --name rabbitmq -p 5672:5672 -p 15672:15672 rabbitmq:3-management

# 2. Start Temporal
temporal server start-dev

# 3. Start backend with WebSocket support
go run backend/cmd/main.go

# 4. Test event publishing (from another terminal)
curl -X POST http://localhost:8080/api-endpoints \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: tenant-001" \
  -d '{
    "endpoint_name": "Test Endpoint",
    "http_method": "GET",
    "url_path": "/test",
    "category": "validation",
    "description": "Test endpoint"
  }'

# 5. Check RabbitMQ queue depth
rabbitmqctl list_queues

# 6. Monitor WebSocket connections
# Open browser console and verify 'connected' message
```

### Performance Targets

| Metric | Target | Notes |
|--------|--------|-------|
| WebSocket Connection Time | <500ms | Initial TCP + handshake |
| Event Broadcast Latency | <100ms | From RabbitMQ to client |
| Catalog Node Creation | <1s | Include database write |
| Edge Creation | <1s | Include relationship write |
| WebSocket Reconnection | 2-10s | Exponential backoff |
| Max Clients per Tenant | 100+ | Horizontal scale |

## Summary

Phase 3 implements real-time catalog synchronization by:

1. **Frontend Service Layer**: TypeScript service classes for API communication
2. **WebSocket Integration**: Real-time updates from backend via WebSocket
3. **Event-Driven Updates**: React components automatically update when catalog changes
4. **Error Recovery**: Automatic reconnection with exponential backoff
5. **Tenant Isolation**: All WebSocket connections scoped to tenant
6. **Monitoring**: Built-in logging and status tracking

**Estimated Implementation Time**: 2 hours (following code examples provided)

**Next Phase (Phase 4)**: Testing, monitoring, and production deployment
