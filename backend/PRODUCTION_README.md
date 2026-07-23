# Production WebSocket Server with ML Analytics

## 🚀 Overview

This production-ready WebSocket server provides real-time financial analytics with machine learning capabilities, designed to scale for multiple concurrent users.

## ✨ Features

### 🔄 Real-Time Capabilities
- **WebSocket Connections**: Bidirectional real-time communication
- **Concurrent Scaling**: Support for up to 1000+ concurrent clients
- **Connection Management**: Automatic cleanup of inactive connections
- **Load Balancing Ready**: Designed for horizontal scaling

### 🤖 Machine Learning Analytics
- **Portfolio Return Prediction**: ML-powered return forecasting
- **Risk Assessment**: Automated risk level classification
- **Market Sentiment Analysis**: Real-time sentiment scoring
- **Volatility Forecasting**: Time-series volatility predictions
- **Real-Time Streaming**: Live analytics data broadcasting

### 📊 Financial Calculations
- **Markowitz Optimization**: Portfolio optimization
- **GBM Simulation**: Geometric Brownian Motion stock simulation
- **Efficient Frontier**: Risk-return optimization
- **Custom Calculations**: Extensible calculation framework

### 🏥 Production Features
- **Health Monitoring**: `/health` endpoint for service monitoring
- **Performance Metrics**: `/metrics` endpoint with real-time stats
- **Error Handling**: Comprehensive error handling and logging
- **Rate Limiting**: Built-in connection and request rate limiting

## 🏗️ Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Web Clients   │────│  WebSocket Hub   │────│   ML Service    │
│                 │    │                  │    │                 │
│ • Dashboards    │    │ • Connection Mgmt│    │ • Predictions   │
│ • Mobile Apps   │    │ • Message Routing│    │ • Analytics     │
│ • Trading Tools │    │ • Broadcasting   │    │ • Streaming     │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────────────┐
                    │  Calculation Engine │
                    │                     │
                    │ • Markowitz         │
                    │ • GBM Simulation    │
                    │ • Risk Analysis     │
                    └─────────────────────┘
```

## 🚀 Quick Start

### Prerequisites
- Go 1.19+
- gorilla/websocket package

### Installation
```bash
cd /Users/eganpj/GitHub/semlayer/backend
go mod tidy
```

### Running the Production Server
```bash
# Build and run
go build production_server.go websocket_hub.go ml_service.go production_websocket_server.go
./production_server
```

### Demo Script
```bash
chmod +x run_production_demo.sh
./run_production_demo.sh
```

## 📡 API Endpoints

### WebSocket Endpoints

#### Main WebSocket Connection
```
ws://localhost:8081/ws?user_id=<user>&session_id=<session>
```

**Supported Message Types:**
- `calculation`: Financial calculations (Markowitz, GBM, etc.)
- `ml_prediction`: ML-powered predictions
- `analytics_stream`: Real-time analytics streaming
- `portfolio_analysis`: Comprehensive portfolio analysis

**Example Messages:**

```json
// Calculation Request
{
  "type": "calculation",
  "data": {
    "type": "markowitz",
    "params": {}
  },
  "user_id": "user123",
  "session_id": "session456"
}

// ML Prediction Request
{
  "type": "ml_prediction",
  "data": {
    "model_type": "portfolio_return",
    "features": {
      "market_trend": 0.15,
      "volatility": 0.18,
      "diversification": 0.75
    }
  }
}
```

### HTTP Endpoints

#### Health Check
```
GET /health
```
Response:
```json
{
  "status": "healthy",
  "timestamp": "2025-09-12T21:46:23Z",
  "clients": 5,
  "max_clients": 1000,
  "uptime": "running"
}
```

#### Metrics
```
GET /metrics
```
Response:
```json
{
  "websocket": {
    "active_connections": 5,
    "max_connections": 1000,
    "connection_limit": 1000
  },
  "performance": {
    "uptime_seconds": 3600,
    "messages_processed": 1000,
    "errors_count": 5
  },
  "ml": {
    "predictions_served": 500,
    "models_loaded": 4,
    "avg_prediction_time_ms": 45.2
  }
}
```

#### Broadcast (Admin)
```
POST /broadcast
Content-Type: application/json

{
  "message": "System maintenance in 5 minutes",
  "user_id": "user123"  // Optional: broadcast to specific user
}
```

## 🤖 ML Models

### Available Models

#### 1. Portfolio Return Predictor
- **Type**: Regression
- **Features**: Market trend, volatility, diversification
- **Output**: Expected return, confidence intervals

#### 2. Risk Assessment Model
- **Type**: Classification
- **Features**: Portfolio value, volatility, liquidity
- **Output**: Risk level (low/medium/high), recommendations

#### 3. Market Sentiment Analyzer
- **Type**: NLP + Sentiment Analysis
- **Features**: News sentiment, social sentiment, economic indicators
- **Output**: Overall sentiment score, sentiment drivers

#### 4. Volatility Forecasting Model
- **Type**: Time Series
- **Features**: Historical volatility, market stress, time horizon
- **Output**: Predicted volatility, confidence intervals

### Model Management

```go
// Get model information
models := mlService.GetModelInfo()

// Update a model (retraining simulation)
err := mlService.UpdateModel("portfolio_return")

// Make predictions
prediction, err := mlService.Predict(PredictionRequest{
    ModelType: "portfolio_return",
    Features: map[string]interface{}{
        "market_trend": 0.15,
        "volatility": 0.18,
    },
})
```

## 📈 Scaling & Performance

### Connection Limits
- **Default Max Clients**: 1000
- **Configurable**: Adjust `maxClients` parameter
- **Auto Cleanup**: Inactive connections removed after 5 minutes

### Performance Optimization
- **Goroutine Pool**: Efficient concurrent processing
- **Message Buffering**: 256-message buffer per client
- **Memory Management**: Automatic cleanup and GC optimization
- **Rate Limiting**: Built-in request rate limiting

### Monitoring
- **Real-time Metrics**: Active connections, message throughput
- **Health Checks**: Automated service health monitoring
- **Performance Logging**: Detailed performance metrics
- **Error Tracking**: Comprehensive error logging

## 🔧 Configuration

### Environment Variables
```bash
export WS_PORT=8081
export MAX_CLIENTS=1000
export CLEANUP_INTERVAL=300  # seconds
export ML_UPDATE_INTERVAL=1800  # seconds
```

### Server Configuration
```go
// Create hub with custom limits
hub := NewWebSocketHub(2000)  // 2000 max clients

// Start server with custom port
StartProductionWebSocketServer(9090, 2000)
```

## 🧪 Testing

### Unit Tests
```bash
go test ./... -v
```

### Integration Tests
```bash
go run test_websocket_integration.go
```

### Load Testing
```bash
# Simulate multiple concurrent connections
for i in {1..100}; do
    go run websocket_client.go &
done
```

## 🚀 Deployment

### Docker Deployment
```dockerfile
FROM golang:1.19-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o production-server production_server.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/production-server .
CMD ["./production-server"]
```

### Kubernetes Deployment
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: websocket-server
spec:
  replicas: 3
  selector:
    matchLabels:
      app: websocket-server
  template:
    metadata:
      labels:
        app: websocket-server
    spec:
      containers:
      - name: websocket-server
        image: your-registry/websocket-server:latest
        ports:
        - containerPort: 8081
        env:
        - name: MAX_CLIENTS
          value: "1000"
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "1Gi"
            cpu: "1000m"
```

### Load Balancer Configuration
```nginx
upstream websocket_backend {
    ip_hash;  # Session stickiness for WebSocket
    server websocket-server-1:8081;
    server websocket-server-2:8081;
    server websocket-server-3:8081;
}

server {
    listen 80;
    location /ws {
        proxy_pass http://websocket_backend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## 📊 Monitoring & Observability

### Metrics Collection
- **Prometheus Integration**: Export metrics for monitoring
- **Custom Metrics**: Application-specific performance metrics
- **Health Checks**: Automated health monitoring

### Logging
- **Structured Logging**: JSON-formatted logs
- **Log Levels**: DEBUG, INFO, WARN, ERROR
- **Performance Logging**: Request/response timing

### Alerting
- **Connection Limits**: Alert when approaching max connections
- **Error Rates**: Monitor and alert on error spikes
- **Performance Degradation**: Alert on slow response times

## 🔒 Security

### Authentication
- **Session Management**: Secure session handling
- **User Identification**: User-specific message routing
- **Rate Limiting**: Prevent abuse and DoS attacks

### Data Protection
- **Input Validation**: Comprehensive message validation
- **Error Handling**: Secure error message handling
- **Connection Security**: WebSocket over WSS in production

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## 📝 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🆘 Support

For support and questions:
- Create an issue in the GitHub repository
- Check the documentation
- Review the example implementations

---

**Built with ❤️ for real-time financial analytics**
