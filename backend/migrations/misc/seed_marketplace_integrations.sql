-- Seed Marketplace with Pre-built Integrations
-- This script populates the marketplace_integrations table with 5 pre-built integrations
-- Run: psql "postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable" -f backend/migrations/misc/seed_marketplace_integrations.sql

-- Clear existing data (optional - comment out if you want to preserve existing integrations)
-- DELETE FROM marketplace_integrations WHERE integration_key IN ('slack', 'email', 'webhook', 'teams', 'rest_api');

-- 1. Slack Integration
INSERT INTO marketplace_integrations (
    id,
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
    supports_polling,
    supports_actions,
    rating,
    install_count,
    documentation_url,
    example_payload
) VALUES (
    gen_random_uuid(),
    'slack',
    'Slack',
    'Send notifications, messages, and interactive cards to Slack channels. Perfect for team collaboration and workflow notifications.',
    'communication',
    'Slack Technologies',
    'https://cdn.jsdelivr.net/npm/simple-icons@v9/icons/slack.svg',
    '1.0.0',
    true,
    'oauth2',
    '{
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
            "description": "Default channel to post messages (optional)",
            "required": false,
            "placeholder": "#general"
        },
        "username": {
            "type": "string",
            "label": "Bot Username",
            "description": "Display name for the bot",
            "required": false,
            "default": "Workflow Bot"
        },
        "icon_emoji": {
            "type": "string",
            "label": "Bot Icon Emoji",
            "description": "Emoji to use as bot icon",
            "required": false,
            "default": ":robot_face:"
        }
    }'::jsonb,
    '{
        "authorization_url": "https://slack.com/oauth/v2/authorize",
        "token_url": "https://slack.com/api/oauth.v2.access",
        "scopes": ["chat:write", "channels:read", "users:read"],
        "client_id_key": "SLACK_CLIENT_ID",
        "client_secret_key": "SLACK_CLIENT_SECRET"
    }'::jsonb,
    true,
    false,
    true,
    4.8,
    2547,
    'https://api.slack.com/messaging/webhooks',
    '{
        "action": "send_message",
        "params": {
            "channel": "#alerts",
            "text": "Process completed successfully!",
            "attachments": [
                {
                    "color": "good",
                    "title": "Workflow Status",
                    "text": "All steps completed without errors.",
                    "fields": [
                        {"title": "Duration", "value": "2m 34s", "short": true},
                        {"title": "Status", "value": "Success", "short": true}
                    ]
                }
            ]
        }
    }'::jsonb
) ON CONFLICT (integration_key) DO UPDATE SET
    description = EXCLUDED.description,
    config_schema = EXCLUDED.config_schema,
    rating = EXCLUDED.rating,
    install_count = EXCLUDED.install_count,
    updated_at = CURRENT_TIMESTAMP;

-- 2. Email (SMTP) Integration
INSERT INTO marketplace_integrations (
    id,
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
    supports_webhooks,
    supports_polling,
    supports_actions,
    rating,
    install_count,
    documentation_url,
    example_payload
) VALUES (
    gen_random_uuid(),
    'email',
    'Email (SMTP)',
    'Send email notifications and alerts via SMTP server. Supports HTML templates, attachments, and CC/BCC recipients.',
    'communication',
    'Generic',
    'https://cdn.jsdelivr.net/npm/simple-icons@v9/icons/gmail.svg',
    '1.0.0',
    true,
    'basic_auth',
    '{
        "smtp_host": {
            "type": "string",
            "label": "SMTP Host",
            "description": "SMTP server hostname",
            "required": true,
            "placeholder": "smtp.gmail.com"
        },
        "smtp_port": {
            "type": "number",
            "label": "SMTP Port",
            "description": "SMTP server port",
            "required": true,
            "default": 587,
            "enum": [25, 465, 587, 2525]
        },
        "use_tls": {
            "type": "boolean",
            "label": "Use TLS",
            "description": "Enable TLS encryption",
            "required": false,
            "default": true
        },
        "username": {
            "type": "string",
            "label": "Username",
            "description": "SMTP authentication username",
            "required": true,
            "placeholder": "user@example.com"
        },
        "password": {
            "type": "string",
            "label": "Password",
            "description": "SMTP authentication password",
            "required": true,
            "secret": true,
            "input_type": "password"
        },
        "from_address": {
            "type": "string",
            "label": "From Address",
            "description": "Sender email address",
            "required": true,
            "placeholder": "noreply@example.com"
        },
        "from_name": {
            "type": "string",
            "label": "From Name",
            "description": "Sender display name",
            "required": false,
            "default": "Workflow System"
        }
    }'::jsonb,
    false,
    false,
    true,
    4.6,
    1832,
    'https://nodemailer.com/smtp/',
    '{
        "action": "send_email",
        "params": {
            "to": "recipient@example.com",
            "subject": "Workflow Notification",
            "body": "<h1>Process Completed</h1><p>Your workflow has completed successfully.</p>",
            "is_html": true,
            "cc": [],
            "bcc": [],
            "attachments": []
        }
    }'::jsonb
) ON CONFLICT (integration_key) DO UPDATE SET
    description = EXCLUDED.description,
    config_schema = EXCLUDED.config_schema,
    rating = EXCLUDED.rating,
    install_count = EXCLUDED.install_count,
    updated_at = CURRENT_TIMESTAMP;

-- 3. Webhook Integration
INSERT INTO marketplace_integrations (
    id,
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
    supports_webhooks,
    supports_polling,
    supports_actions,
    rating,
    install_count,
    documentation_url,
    example_payload
) VALUES (
    gen_random_uuid(),
    'webhook',
    'Webhook',
    'Trigger HTTP webhooks with custom payloads. Supports GET, POST, PUT, DELETE methods with custom headers and authentication.',
    'automation',
    'Generic',
    'https://cdn.jsdelivr.net/npm/simple-icons@v9/icons/webhook.svg',
    '1.0.0',
    true,
    'api_key',
    '{
        "webhook_url": {
            "type": "string",
            "label": "Webhook URL",
            "description": "Target webhook endpoint URL",
            "required": true,
            "placeholder": "https://api.example.com/webhook"
        },
        "method": {
            "type": "string",
            "label": "HTTP Method",
            "description": "HTTP request method",
            "required": true,
            "default": "POST",
            "enum": ["GET", "POST", "PUT", "DELETE", "PATCH"]
        },
        "headers": {
            "type": "object",
            "label": "Custom Headers",
            "description": "Additional HTTP headers (JSON object)",
            "required": false,
            "default": {},
            "placeholder": "{\"Content-Type\": \"application/json\"}"
        },
        "api_key_header": {
            "type": "string",
            "label": "API Key Header Name",
            "description": "Header name for API key",
            "required": false,
            "default": "X-API-Key",
            "placeholder": "Authorization"
        },
        "timeout_seconds": {
            "type": "number",
            "label": "Timeout (seconds)",
            "description": "Request timeout in seconds",
            "required": false,
            "default": 30,
            "min": 1,
            "max": 300
        },
        "retry_on_failure": {
            "type": "boolean",
            "label": "Retry on Failure",
            "description": "Automatically retry failed requests",
            "required": false,
            "default": true
        }
    }'::jsonb,
    true,
    false,
    true,
    4.7,
    3104,
    'https://webhook.site/docs',
    '{
        "action": "trigger_webhook",
        "params": {
            "payload": {
                "event": "workflow_completed",
                "workflow_id": "wf_123",
                "status": "success",
                "timestamp": "2024-01-15T10:30:00Z",
                "data": {
                    "result": "Process completed successfully"
                }
            }
        }
    }'::jsonb
) ON CONFLICT (integration_key) DO UPDATE SET
    description = EXCLUDED.description,
    config_schema = EXCLUDED.config_schema,
    rating = EXCLUDED.rating,
    install_count = EXCLUDED.install_count,
    updated_at = CURRENT_TIMESTAMP;

-- 4. Microsoft Teams Integration
INSERT INTO marketplace_integrations (
    id,
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
    supports_polling,
    supports_actions,
    rating,
    install_count,
    documentation_url,
    example_payload
) VALUES (
    gen_random_uuid(),
    'teams',
    'Microsoft Teams',
    'Send notifications and adaptive cards to Microsoft Teams channels. Ideal for enterprise collaboration and workflow updates.',
    'communication',
    'Microsoft',
    'https://cdn.jsdelivr.net/npm/simple-icons@v9/icons/microsoftteams.svg',
    '1.0.0',
    true,
    'oauth2',
    '{
        "webhook_url": {
            "type": "string",
            "label": "Webhook URL",
            "description": "Teams incoming webhook URL",
            "required": false,
            "placeholder": "https://outlook.office.com/webhook/..."
        },
        "team_id": {
            "type": "string",
            "label": "Team ID",
            "description": "Microsoft Teams team ID (optional for OAuth)",
            "required": false,
            "placeholder": "19:xxx@thread.tacv2"
        },
        "channel_id": {
            "type": "string",
            "label": "Channel ID",
            "description": "Default channel ID (optional)",
            "required": false,
            "placeholder": "19:xxx@thread.tacv2"
        },
        "use_adaptive_cards": {
            "type": "boolean",
            "label": "Use Adaptive Cards",
            "description": "Send messages as adaptive cards",
            "required": false,
            "default": true
        }
    }'::jsonb,
    '{
        "authorization_url": "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
        "token_url": "https://login.microsoftonline.com/common/oauth2/v2.0/token",
        "scopes": ["ChannelMessage.Send", "Team.ReadBasic.All"],
        "client_id_key": "TEAMS_CLIENT_ID",
        "client_secret_key": "TEAMS_CLIENT_SECRET"
    }'::jsonb,
    true,
    false,
    true,
    4.5,
    1456,
    'https://docs.microsoft.com/en-us/microsoftteams/platform/webhooks-and-connectors/how-to/connectors-using',
    '{
        "action": "send_message",
        "params": {
            "title": "Workflow Notification",
            "text": "Your workflow has completed successfully.",
            "theme_color": "0078D4",
            "sections": [
                {
                    "activityTitle": "Process Status",
                    "activitySubtitle": "Completed at 10:30 AM",
                    "facts": [
                        {"name": "Workflow ID", "value": "wf_123"},
                        {"name": "Duration", "value": "2m 34s"},
                        {"name": "Status", "value": "Success"}
                    ]
                }
            ]
        }
    }'::jsonb
) ON CONFLICT (integration_key) DO UPDATE SET
    description = EXCLUDED.description,
    config_schema = EXCLUDED.config_schema,
    rating = EXCLUDED.rating,
    install_count = EXCLUDED.install_count,
    updated_at = CURRENT_TIMESTAMP;

-- 5. Generic REST API Integration
INSERT INTO marketplace_integrations (
    id,
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
    supports_webhooks,
    supports_polling,
    supports_actions,
    rating,
    install_count,
    documentation_url,
    example_payload
) VALUES (
    gen_random_uuid(),
    'rest_api',
    'Generic REST API',
    'Make HTTP requests to any REST API endpoint. Supports all HTTP methods, custom headers, authentication, and request/response transformation.',
    'automation',
    'Generic',
    'https://cdn.jsdelivr.net/npm/simple-icons@v9/icons/fastapi.svg',
    '1.0.0',
    true,
    'custom',
    '{
        "base_url": {
            "type": "string",
            "label": "Base URL",
            "description": "API base URL",
            "required": true,
            "placeholder": "https://api.example.com"
        },
        "default_method": {
            "type": "string",
            "label": "Default HTTP Method",
            "description": "Default request method",
            "required": true,
            "default": "GET",
            "enum": ["GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"]
        },
        "default_headers": {
            "type": "object",
            "label": "Default Headers",
            "description": "Headers to include in all requests",
            "required": false,
            "default": {"Content-Type": "application/json"},
            "placeholder": "{\"Accept\": \"application/json\"}"
        },
        "auth_type": {
            "type": "string",
            "label": "Authentication Type",
            "description": "API authentication method",
            "required": true,
            "default": "none",
            "enum": ["none", "bearer", "basic", "api_key", "oauth2"]
        },
        "auth_token": {
            "type": "string",
            "label": "Auth Token",
            "description": "Bearer token or API key",
            "required": false,
            "secret": true,
            "input_type": "password",
            "placeholder": "Enter your API token"
        },
        "auth_header_name": {
            "type": "string",
            "label": "Auth Header Name",
            "description": "Header name for auth token (for api_key type)",
            "required": false,
            "default": "Authorization",
            "placeholder": "X-API-Key"
        },
        "timeout_seconds": {
            "type": "number",
            "label": "Timeout (seconds)",
            "description": "Request timeout",
            "required": false,
            "default": 30,
            "min": 1,
            "max": 300
        },
        "verify_ssl": {
            "type": "boolean",
            "label": "Verify SSL",
            "description": "Verify SSL certificates",
            "required": false,
            "default": true
        }
    }'::jsonb,
    false,
    true,
    true,
    4.6,
    2198,
    'https://restfulapi.net/',
    '{
        "action": "make_request",
        "params": {
            "endpoint": "/api/v1/users",
            "method": "GET",
            "query_params": {"limit": 10, "page": 1},
            "headers": {"X-Custom-Header": "value"},
            "body": null
        }
    }'::jsonb
) ON CONFLICT (integration_key) DO UPDATE SET
    description = EXCLUDED.description,
    config_schema = EXCLUDED.config_schema,
    rating = EXCLUDED.rating,
    install_count = EXCLUDED.install_count,
    updated_at = CURRENT_TIMESTAMP;

-- Display summary
SELECT 
    integration_key,
    name,
    category,
    auth_type,
    rating,
    install_count,
    is_official
FROM marketplace_integrations
WHERE integration_key IN ('slack', 'email', 'webhook', 'teams', 'rest_api')
ORDER BY install_count DESC;

COMMENT ON TABLE marketplace_integrations IS 'Successfully seeded with 5 pre-built integrations: Slack, Email, Webhook, Teams, REST API';
