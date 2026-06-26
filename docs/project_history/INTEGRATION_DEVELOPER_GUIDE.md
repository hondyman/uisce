# Integration Marketplace - Developer Guide

## Overview

This guide explains how to create custom integrations for the Fabric Builder Integration Marketplace. You'll learn how to define integration schemas, implement handlers, and publish your integrations to the marketplace.

## Architecture

### Components

The Integration Marketplace consists of:

1. **Database Schema**: PostgreSQL tables for catalog and installations
2. **Backend API**: Go handlers for CRUD operations and execution
3. **Frontend UI**: React components for browsing and management
4. **Integration Handlers**: Custom execution logic for each integration

### Data Flow

```
User Action (UI)
    ↓
API Request (REST)
    ↓
Handler (Go)
    ↓
Database (PostgreSQL)
    ↓
Integration Executor
    ↓
External Service (Slack/API/etc)
    ↓
Execution Log
```

## Database Schema

### marketplace_integrations

Catalog of available integrations:

```sql
CREATE TABLE marketplace_integrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    integration_key VARCHAR(100) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(50), -- communication, automation, storage, analytics, ai, other
    provider VARCHAR(255),
    icon_url TEXT,
    version VARCHAR(50),
    is_official BOOLEAN DEFAULT false,
    auth_type VARCHAR(50) DEFAULT 'none', -- none, api_key, oauth2, basic_auth, custom
    config_schema JSONB, -- JSON Schema for configuration
    oauth_config JSONB, -- OAuth provider details
    supports_webhooks BOOLEAN DEFAULT false,
    supports_polling BOOLEAN DEFAULT false,
    supports_actions BOOLEAN DEFAULT true,
    rating DECIMAL(3,2) DEFAULT 0.0,
    install_count INTEGER DEFAULT 0,
    documentation_url TEXT,
    example_payload JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### installed_integrations

Tenant-specific integration instances:

```sql
CREATE TABLE installed_integrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    datasource_id UUID NOT NULL,
    integration_id UUID REFERENCES marketplace_integrations(id) ON DELETE CASCADE,
    config JSONB, -- User-provided configuration
    credentials JSONB, -- Encrypted credentials (API keys, tokens)
    oauth_access_token TEXT,
    oauth_refresh_token TEXT,
    oauth_expires_at TIMESTAMP,
    is_enabled BOOLEAN DEFAULT true,
    execution_count INTEGER DEFAULT 0,
    success_count INTEGER DEFAULT 0,
    failure_count INTEGER DEFAULT 0,
    last_executed_at TIMESTAMP,
    installed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### integration_executions

Execution audit log:

```sql
CREATE TABLE integration_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    installed_integration_id UUID REFERENCES installed_integrations(id) ON DELETE CASCADE,
    workflow_id UUID,
    workflow_type VARCHAR(100),
    step_name VARCHAR(255),
    action VARCHAR(100),
    request_payload JSONB,
    response_payload JSONB,
    status VARCHAR(50), -- pending, success, failed, timeout, cancelled
    error_message TEXT,
    duration_ms INTEGER,
    retry_count INTEGER DEFAULT 0,
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP
);
```

## Creating a Custom Integration

### Step 1: Define Integration Schema

Create a SQL INSERT statement with your integration details:

```sql
INSERT INTO marketplace_integrations (
    integration_key,
    name,
    description,
    category,
    provider,
    icon_url,
    version,
    is_official,
    auth_type,
    config_schema,
    oauth_config,
    supports_webhooks,
    supports_actions,
    rating,
    documentation_url,
    example_payload
) VALUES (
    'my_integration', -- Unique key (lowercase, underscores)
    'My Integration', -- Display name
    'Description of what this integration does', -- Full description
    'automation', -- communication | automation | storage | analytics | ai | other
    'My Company', -- Provider name
    'https://example.com/icon.svg', -- Icon URL (SVG preferred)
    '1.0.0', -- Semantic version
    false, -- Is this an official integration?
    'api_key', -- none | api_key | oauth2 | basic_auth | custom
    '{
        "api_key": {
            "type": "string",
            "label": "API Key",
            "description": "Your API key from the provider",
            "required": true,
            "secret": true,
            "placeholder": "Enter your API key"
        },
        "endpoint": {
            "type": "string",
            "label": "API Endpoint",
            "description": "Base URL for API requests",
            "required": true,
            "default": "https://api.example.com"
        }
    }'::jsonb,
    NULL, -- OAuth config (if auth_type is oauth2)
    true, -- Supports webhooks?
    true, -- Supports actions?
    4.5, -- Initial rating
    'https://example.com/docs', -- Documentation URL
    '{
        "action": "send_data",
        "params": {
            "message": "Hello from workflow!"
        }
    }'::jsonb
);
```

### Step 2: Define Config Schema

The `config_schema` field uses a simplified JSON Schema format:

```json
{
  "field_name": {
    "type": "string | number | boolean | object | array",
    "label": "Human-readable label",
    "description": "Help text for the field",
    "required": true | false,
    "secret": true | false, // Hide in UI, encrypt in storage
    "default": "default_value",
    "placeholder": "Placeholder text",
    "enum": ["option1", "option2"], // For dropdowns
    "min": 1, // For numbers
    "max": 100, // For numbers
    "pattern": "regex_pattern", // For strings
    "input_type": "text | password | email | url" // HTML input type
  }
}
```

#### Example: Slack Integration Schema

```json
{
  "webhook_url": {
    "type": "string",
    "label": "Webhook URL",
    "description": "Slack incoming webhook URL",
    "required": false,
    "placeholder": "https://hooks.slack.com/services/..."
  },
  "channel": {
    "type": "string",
    "label": "Default Channel",
    "description": "Channel to post messages",
    "required": false,
    "placeholder": "#general"
  },
  "username": {
    "type": "string",
    "label": "Bot Username",
    "description": "Display name for the bot",
    "required": false,
    "default": "Workflow Bot"
  }
}
```

### Step 3: Implement OAuth Config (Optional)

If your integration uses OAuth2, define the OAuth configuration:

```json
{
  "authorization_url": "https://provider.com/oauth/authorize",
  "token_url": "https://provider.com/oauth/token",
  "scopes": ["read", "write", "admin"],
  "client_id_key": "ENV_VAR_FOR_CLIENT_ID",
  "client_secret_key": "ENV_VAR_FOR_CLIENT_SECRET"
}
```

The OAuth flow:

1. User clicks "Install" → initiates OAuth
2. System redirects to `authorization_url` with client_id and scopes
3. User authorizes → provider redirects to callback URL
4. System exchanges code for token at `token_url`
5. Tokens stored in `installed_integrations` table

### Step 4: Implement Integration Handler

Add your integration handler in `backend/internal/api/marketplace_integration_handlers.go`:

```go
func (h *MarketplaceIntegrationHandlers) executeMyIntegration(
    ctx context.Context,
    installation InstalledIntegration,
    action string,
    payload map[string]interface{},
) (map[string]interface{}, error) {
    // Extract configuration
    apiKey, _ := installation.Credentials["api_key"].(string)
    endpoint, _ := installation.Config["endpoint"].(string)
    
    // Validate configuration
    if apiKey == "" {
        return nil, errors.New("API key not configured")
    }
    
    // Execute action based on action parameter
    switch action {
    case "send_data":
        message, _ := payload["message"].(string)
        
        // Make HTTP request
        req, _ := http.NewRequestWithContext(ctx, "POST", endpoint+"/send", 
            bytes.NewBuffer([]byte(fmt.Sprintf(`{"message":"%s"}`, message))))
        req.Header.Set("Authorization", "Bearer "+apiKey)
        req.Header.Set("Content-Type", "application/json")
        
        client := &http.Client{Timeout: 30 * time.Second}
        resp, err := client.Do(req)
        if err != nil {
            return nil, fmt.Errorf("request failed: %w", err)
        }
        defer resp.Body.Close()
        
        // Parse response
        var result map[string]interface{}
        if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
            return nil, fmt.Errorf("failed to parse response: %w", err)
        }
        
        if resp.StatusCode >= 400 {
            return nil, fmt.Errorf("API error: %s", result["error"])
        }
        
        return result, nil
        
    default:
        return nil, fmt.Errorf("unknown action: %s", action)
    }
}
```

### Step 5: Register Handler in Router

Add your integration to the `executeIntegrationAction` function:

```go
func (h *MarketplaceIntegrationHandlers) executeIntegrationAction(
    ctx context.Context,
    installation InstalledIntegration,
    action string,
    payload map[string]interface{},
) (map[string]interface{}, error) {
    switch installation.IntegrationKey {
    case "slack":
        return h.executeSlackAction(ctx, installation, action, payload)
    case "email":
        return h.executeEmailAction(ctx, installation, action, payload)
    case "my_integration": // Add your integration here
        return h.executeMyIntegration(ctx, installation, action, payload)
    default:
        return nil, fmt.Errorf("unsupported integration: %s", installation.IntegrationKey)
    }
}
```

### Step 6: Test Connection Handler

Implement a test function to verify configuration:

```go
func (h *MarketplaceIntegrationHandlers) testMyIntegrationConnection(
    ctx context.Context,
    installation InstalledIntegration,
) error {
    // Extract credentials
    apiKey, _ := installation.Credentials["api_key"].(string)
    endpoint, _ := installation.Config["endpoint"].(string)
    
    if apiKey == "" || endpoint == "" {
        return errors.New("missing required configuration")
    }
    
    // Make test request
    req, _ := http.NewRequestWithContext(ctx, "GET", endpoint+"/health", nil)
    req.Header.Set("Authorization", "Bearer "+apiKey)
    
    client := &http.Client{Timeout: 10 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return fmt.Errorf("connection test failed: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode >= 400 {
        return fmt.Errorf("API returned error: status %d", resp.StatusCode)
    }
    
    return nil
}
```

Add to the test router:

```go
func (h *MarketplaceIntegrationHandlers) testIntegrationConnection(
    ctx context.Context,
    installation InstalledIntegration,
) error {
    switch installation.IntegrationKey {
    case "my_integration":
        return h.testMyIntegrationConnection(ctx, installation)
    default:
        return errors.New("test not implemented for this integration")
    }
}
```

## Advanced Features

### Webhook Support

If your integration supports webhooks (incoming events):

1. Set `supports_webhooks = true` in the catalog
2. Generate a unique webhook URL for each installation:
   ```
   https://your-domain.com/api/integrations/webhook/{installation_id}
   ```
3. Implement webhook handler:
   ```go
   func (h *MarketplaceIntegrationHandlers) HandleWebhook(w http.ResponseWriter, r *http.Request) {
       installationID := chi.URLParam(r, "installationID")
       
       // Verify webhook signature (if provider supports it)
       signature := r.Header.Get("X-Signature")
       if !h.verifyWebhookSignature(r.Body, signature) {
           http.Error(w, "invalid signature", http.StatusUnauthorized)
           return
       }
       
       // Parse payload
       var payload map[string]interface{}
       json.NewDecoder(r.Body).Decode(&payload)
       
       // Trigger workflow or process event
       h.processWebhookEvent(installationID, payload)
       
       w.WriteHeader(http.StatusOK)
   }
   ```

### Polling Support

If your integration supports polling (fetch data periodically):

1. Set `supports_polling = true` in the catalog
2. Implement polling function:
   ```go
   func (h *MarketplaceIntegrationHandlers) pollMyIntegration(
       ctx context.Context,
       installation InstalledIntegration,
   ) ([]map[string]interface{}, error) {
       // Fetch data from external service
       apiKey, _ := installation.Credentials["api_key"].(string)
       endpoint, _ := installation.Config["endpoint"].(string)
       
       req, _ := http.NewRequestWithContext(ctx, "GET", endpoint+"/events", nil)
       req.Header.Set("Authorization", "Bearer "+apiKey)
       
       resp, err := client.Do(req)
       // ... parse and return events
   }
   ```
3. Register polling job in scheduler (cron or background worker)

### Batch Operations

For efficiency, support batch operations:

```go
func (h *MarketplaceIntegrationHandlers) executeBatchAction(
    ctx context.Context,
    installation InstalledIntegration,
    action string,
    items []map[string]interface{},
) ([]map[string]interface{}, error) {
    // Batch API call
    results := make([]map[string]interface{}, len(items))
    
    for i, item := range items {
        result, err := h.executeMyIntegration(ctx, installation, action, item)
        if err != nil {
            results[i] = map[string]interface{}{"error": err.Error()}
        } else {
            results[i] = result
        }
    }
    
    return results, nil
}
```

### Token Refresh (OAuth)

Implement automatic token refresh for OAuth integrations:

```go
func (h *MarketplaceIntegrationHandlers) refreshOAuthToken(
    ctx context.Context,
    installation InstalledIntegration,
) error {
    // Check if token is expired
    if installation.OAuthExpiresAt == nil || 
       time.Now().Before(*installation.OAuthExpiresAt) {
        return nil // Token still valid
    }
    
    // Get OAuth config from marketplace
    var integration MarketplaceIntegration
    h.db.Get(&integration, "SELECT oauth_config FROM marketplace_integrations WHERE id = $1", 
        installation.IntegrationID)
    
    tokenURL := integration.OAuthConfig["token_url"].(string)
    clientID := os.Getenv(integration.OAuthConfig["client_id_key"].(string))
    clientSecret := os.Getenv(integration.OAuthConfig["client_secret_key"].(string))
    
    // Exchange refresh token
    data := url.Values{}
    data.Set("grant_type", "refresh_token")
    data.Set("refresh_token", installation.OAuthRefreshToken)
    data.Set("client_id", clientID)
    data.Set("client_secret", clientSecret)
    
    resp, err := http.PostForm(tokenURL, data)
    // ... parse response and update tokens in database
    
    return nil
}
```

## Testing Your Integration

### Unit Tests

Create unit tests for your handler:

```go
func TestExecuteMyIntegration(t *testing.T) {
    h := &MarketplaceIntegrationHandlers{}
    
    installation := InstalledIntegration{
        IntegrationKey: "my_integration",
        Config: map[string]interface{}{
            "endpoint": "https://api.example.com",
        },
        Credentials: map[string]interface{}{
            "api_key": "test_key_123",
        },
    }
    
    payload := map[string]interface{}{
        "message": "Test message",
    }
    
    result, err := h.executeMyIntegration(context.Background(), installation, "send_data", payload)
    
    assert.NoError(t, err)
    assert.NotNil(t, result)
}
```

### Integration Tests

Test with real API (use sandbox/test credentials):

```go
func TestMyIntegrationLive(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    
    apiKey := os.Getenv("MY_INTEGRATION_TEST_KEY")
    if apiKey == "" {
        t.Skip("test API key not configured")
    }
    
    // ... test with real API
}
```

### Manual Testing

1. Insert your integration into the database
2. Access the Marketplace UI
3. Install the integration
4. Fill in test credentials
5. Click "Test Connection"
6. Create a test workflow that uses the integration
7. Execute and verify in Execution Logs

## Publishing to Marketplace

### Checklist

Before publishing your integration:

- [ ] Config schema is complete and validated
- [ ] All required fields have proper labels and descriptions
- [ ] Secret fields are marked with `"secret": true`
- [ ] Integration handler is implemented and tested
- [ ] Test connection handler works correctly
- [ ] Error messages are clear and actionable
- [ ] Documentation URL points to comprehensive guide
- [ ] Example payload demonstrates common use case
- [ ] Icon URL is accessible (HTTPS, SVG preferred)
- [ ] Provider name and version are accurate
- [ ] Category is appropriate

### Deployment Steps

1. **Add to Database**:
   ```sql
   INSERT INTO marketplace_integrations (...) VALUES (...);
   ```

2. **Deploy Backend**:
   ```bash
   cd backend
   go build
   # Deploy to production
   ```

3. **Verify in UI**:
   - Access Marketplace tab
   - Confirm integration appears in correct category
   - Verify icon and description display correctly
   - Test installation flow

4. **Create Documentation**:
   - Write user guide with setup instructions
   - Include API credential acquisition steps
   - Provide example configurations
   - Document common troubleshooting steps

5. **Announce**:
   - Notify users about new integration
   - Share documentation link
   - Provide support channel

## Best Practices

### Security

1. **Never log sensitive data**:
   ```go
   // BAD
   log.Printf("API key: %s", apiKey)
   
   // GOOD
   log.Printf("Using API key ending in ...%s", apiKey[len(apiKey)-4:])
   ```

2. **Validate all inputs**:
   ```go
   if apiKey == "" || len(apiKey) < 10 {
       return nil, errors.New("invalid API key")
   }
   ```

3. **Use HTTPS only** for external requests

4. **Implement signature verification** for webhooks

### Performance

1. **Set reasonable timeouts**:
   ```go
   client := &http.Client{
       Timeout: 30 * time.Second,
   }
   ```

2. **Support batch operations** when possible

3. **Cache OAuth tokens** and refresh proactively

4. **Use connection pooling** for database queries

### Error Handling

1. **Return actionable error messages**:
   ```go
   // BAD
   return nil, errors.New("error")
   
   // GOOD
   return nil, fmt.Errorf("failed to send message: API returned 401 Unauthorized. Please verify your API key in integration settings")
   ```

2. **Distinguish between retryable and permanent errors**:
   ```go
   if resp.StatusCode == 429 {
       return nil, &RetryableError{Message: "rate limited"}
   } else if resp.StatusCode == 401 {
       return nil, &PermanentError{Message: "invalid credentials"}
   }
   ```

3. **Log execution context**:
   ```go
   log.Printf("Integration execution failed: integration=%s, action=%s, error=%v",
       installation.IntegrationKey, action, err)
   ```

## Examples

See these pre-built integrations for reference:

- **Slack**: OAuth2, webhook receiving, rich message formatting
- **Email**: Basic auth, SMTP protocol, attachment handling
- **Webhook**: API key auth, flexible HTTP methods, custom headers
- **Microsoft Teams**: OAuth2, adaptive cards, enterprise auth
- **REST API**: Custom auth, dynamic endpoints, variable substitution

Code location: `backend/internal/api/marketplace_integration_handlers.go`

## Support

- **Questions**: Open an issue on GitHub
- **Bug Reports**: Include integration key, error message, and execution log
- **Feature Requests**: Describe use case and expected behavior

---

**Ready to build?** Start with the simplest integration (API key auth, single action) and expand from there!
