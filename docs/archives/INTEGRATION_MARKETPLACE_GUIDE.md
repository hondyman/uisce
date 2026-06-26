# Integration Marketplace - User Guide

## Overview

The Integration Marketplace is your central hub for discovering, installing, and managing workflow integrations. Connect your business processes to external services like Slack, Email, Microsoft Teams, and custom REST APIs with just a few clicks.

## Getting Started

### Accessing the Marketplace

1. Open the **Business Process Builder**
2. Click the **"Integrations"** button in the left sidebar (green button with package icon)
3. The marketplace browser will open with three main sections:
   - **Marketplace**: Browse and install new integrations
   - **Installed**: Manage your installed integrations
   - **Execution Logs**: View integration execution history

## Marketplace Tab

### Browsing Integrations

The Marketplace displays all available integrations organized by category:

- **All Integrations**: View all available integrations
- **Communication**: Slack, Email, Microsoft Teams, etc.
- **Automation**: Webhooks, REST APIs, automation tools
- **Storage**: Cloud storage and database connectors
- **Analytics**: Analytics platforms and reporting tools
- **AI**: AI and machine learning services

### Searching for Integrations

Use the search bar at the top to find specific integrations by name or description:

1. Enter your search term (e.g., "Slack", "email", "webhook")
2. Click **Search** or press Enter
3. Results will filter automatically

### Integration Cards

Each integration card displays:
- **Icon and Name**: Visual identifier and integration name
- **Official Badge**: Indicates verified, officially supported integrations
- **Provider**: Company or organization that maintains the integration
- **Description**: Brief overview of the integration's capabilities
- **Rating**: User rating out of 5 stars (⭐)
- **Install Count**: Number of times this integration has been installed
- **Status**: "Installed" badge if you've already installed it
- **Documentation**: Link to external documentation

### Installing an Integration

1. Click the **"Install"** button on any integration card
2. The installation modal will open with:
   - Integration description
   - Setup instructions
   - Configuration form (fields vary by integration)
3. Fill in the required configuration fields:
   - **API Keys**: Enter your API credentials
   - **URLs**: Webhook URLs, server endpoints, etc.
   - **Settings**: Integration-specific options
4. Click **"Install"** to complete the installation
5. The integration will appear in your **Installed** tab

## Installed Tab

### Managing Your Integrations

The Installed tab shows all integrations you've installed for the current tenant and datasource.

#### Integration Information

Each installed integration displays:
- **Name and Icon**: Integration identifier
- **Status Badge**: "Enabled" (green) or "Disabled" (gray)
- **Installed Date**: When you installed this integration
- **Last Used**: Most recent execution timestamp
- **Execution Statistics**:
  - **Total Executions**: Number of times executed
  - **Success Rate**: Percentage of successful executions
  - **Last Used**: Time since last execution

#### Available Actions

**Toggle Enable/Disable** (Toggle icon):
- Click to enable or disable the integration
- Disabled integrations won't execute even if triggered by workflows
- Useful for temporary troubleshooting or maintenance

**Test Connection** (Play icon):
- Verify the integration is configured correctly
- Sends a test request without executing a real action
- Displays success/failure alert with details

**Configure** (Settings icon):
- Open configuration editor
- Update API keys, URLs, or settings
- Changes take effect immediately

**Uninstall** (Trash icon):
- Permanently remove the integration
- Requires confirmation
- Deletes all configuration and execution history

## Execution Logs Tab

### Viewing Execution History

The Execution Logs tab displays a chronological record of all integration executions.

#### Log Columns

- **Status**: Visual indicator of execution result
  - ✅ **Green Check**: Success
  - ❌ **Red X**: Failed
  - ⏱️ **Yellow Clock**: Timeout
  - ⚠️ **Gray Alert**: Cancelled
- **Action**: The action that was executed (e.g., "send_message", "trigger_webhook")
- **Workflow**: Associated workflow ID and step name
- **Duration**: Execution time in milliseconds
- **Time**: When the execution occurred
- **Actions**: "View" button to see full details

#### Viewing Execution Details

Click the **"View"** (eye icon) button to see:
- Full request payload
- Full response payload
- Error messages (if failed)
- Retry count
- Detailed timestamps

## Pre-built Integrations

### 1. Slack

**Category**: Communication  
**Auth Type**: OAuth2

**Use Cases**:
- Send workflow notifications to team channels
- Post alerts on process completion/failure
- Share reports and analytics updates

**Configuration**:
- **Webhook URL** (optional): Slack incoming webhook URL
- **Default Channel**: Channel to post messages (e.g., #alerts)
- **Bot Username**: Display name for the bot
- **Bot Icon Emoji**: Emoji icon (e.g., :robot_face:)

**Example Usage**:
```json
{
  "action": "send_message",
  "params": {
    "channel": "#alerts",
    "text": "Process completed successfully!",
    "attachments": [{
      "color": "good",
      "title": "Workflow Status",
      "text": "All steps completed."
    }]
  }
}
```

### 2. Email (SMTP)

**Category**: Communication  
**Auth Type**: Basic Auth

**Use Cases**:
- Send notification emails to users
- Deliver reports via email
- Alert stakeholders of critical events

**Configuration**:
- **SMTP Host**: Mail server hostname (e.g., smtp.gmail.com)
- **SMTP Port**: Server port (587 for TLS, 465 for SSL)
- **Use TLS**: Enable encryption (recommended)
- **Username**: SMTP authentication username
- **Password**: SMTP authentication password (encrypted)
- **From Address**: Sender email address
- **From Name**: Sender display name

**Gmail Setup**:
1. Enable 2-factor authentication in your Google account
2. Generate an App Password: https://myaccount.google.com/apppasswords
3. Use the app password in the "Password" field

**Example Usage**:
```json
{
  "action": "send_email",
  "params": {
    "to": "user@example.com",
    "subject": "Workflow Notification",
    "body": "<h1>Completed</h1><p>Your workflow has finished.</p>",
    "is_html": true
  }
}
```

### 3. Webhook

**Category**: Automation  
**Auth Type**: API Key

**Use Cases**:
- Trigger external systems on workflow events
- Send data to third-party APIs
- Integrate with custom applications

**Configuration**:
- **Webhook URL**: Target endpoint URL
- **HTTP Method**: GET, POST, PUT, DELETE, PATCH
- **Custom Headers**: Additional HTTP headers (JSON object)
- **API Key Header Name**: Header name for API key
- **Timeout**: Request timeout in seconds
- **Retry on Failure**: Automatically retry failed requests

**Example Usage**:
```json
{
  "action": "trigger_webhook",
  "params": {
    "payload": {
      "event": "workflow_completed",
      "workflow_id": "wf_123",
      "status": "success",
      "data": {"result": "Success"}
    }
  }
}
```

### 4. Microsoft Teams

**Category**: Communication  
**Auth Type**: OAuth2

**Use Cases**:
- Send notifications to Teams channels
- Post adaptive cards with rich content
- Collaborate with enterprise teams

**Configuration**:
- **Webhook URL** (optional): Teams incoming webhook URL
- **Team ID**: Microsoft Teams team identifier
- **Channel ID**: Default channel identifier
- **Use Adaptive Cards**: Enable rich card formatting

**Setup**:
1. In Teams, go to your channel → "..." → Connectors
2. Add "Incoming Webhook" connector
3. Copy the webhook URL to the configuration

**Example Usage**:
```json
{
  "action": "send_message",
  "params": {
    "title": "Workflow Notification",
    "text": "Process completed successfully.",
    "theme_color": "0078D4",
    "sections": [{
      "activityTitle": "Status Update",
      "facts": [
        {"name": "Status", "value": "Success"}
      ]
    }]
  }
}
```

### 5. Generic REST API

**Category**: Automation  
**Auth Type**: Custom

**Use Cases**:
- Connect to any REST API
- Fetch data from external services
- Post data to custom endpoints

**Configuration**:
- **Base URL**: API base URL (e.g., https://api.example.com)
- **Default HTTP Method**: Default request method
- **Default Headers**: Headers for all requests
- **Authentication Type**: none, bearer, basic, api_key, oauth2
- **Auth Token**: Bearer token or API key
- **Auth Header Name**: Header name for auth token
- **Timeout**: Request timeout in seconds
- **Verify SSL**: Enable SSL certificate verification

**Example Usage**:
```json
{
  "action": "make_request",
  "params": {
    "endpoint": "/api/v1/users",
    "method": "GET",
    "query_params": {"limit": 10},
    "headers": {"X-Custom": "value"}
  }
}
```

## Using Integrations in Workflows

### Adding Integration Actions to Steps

1. Create or edit a workflow step
2. In the step configuration, select "Integration Action" as the step type
3. Choose the installed integration from the dropdown
4. Select the action to perform (e.g., "send_message", "trigger_webhook")
5. Configure action parameters:
   - Use static values or workflow variables
   - Reference previous step outputs: `{{step_1.output}}`
   - Use tenant/datasource context: `{{tenant.name}}`

### Example: Slack Notification on Completion

```yaml
steps:
  - name: "Process Data"
    type: "data_processing"
    # ... processing logic ...
  
  - name: "Notify Team"
    type: "integration_action"
    integration: "slack"
    action: "send_message"
    params:
      channel: "#alerts"
      text: "Data processing completed!"
      attachments:
        - color: "good"
          title: "Results"
          text: "{{step_1.output.summary}}"
```

## Best Practices

### Security

1. **API Keys and Passwords**:
   - Never share API keys or credentials
   - Use environment-specific credentials (dev/staging/prod)
   - Rotate credentials regularly

2. **OAuth Integrations**:
   - Use OAuth instead of API keys when available
   - Revoke access for unused integrations
   - Review OAuth scopes before authorizing

3. **Webhook Security**:
   - Use HTTPS URLs only
   - Implement webhook signature verification if available
   - Whitelist IP addresses when possible

### Performance

1. **Timeouts**:
   - Set appropriate timeout values (default: 30s)
   - Consider external service response times
   - Use longer timeouts for complex operations

2. **Retries**:
   - Enable retry on failure for transient errors
   - Implement exponential backoff in workflows
   - Monitor retry counts in execution logs

3. **Rate Limits**:
   - Check integration provider's rate limits
   - Implement delays between bulk operations
   - Use batch operations when available

### Monitoring

1. **Execution Logs**:
   - Review logs regularly for failures
   - Monitor success rates
   - Investigate timeout patterns

2. **Testing**:
   - Use "Test Connection" before deploying workflows
   - Test with sample data first
   - Verify error handling in workflows

3. **Alerts**:
   - Set up notifications for integration failures
   - Monitor execution statistics
   - Track performance degradation

## Troubleshooting

### Common Issues

**Integration Installation Fails**:
- Verify all required fields are filled
- Check API key validity
- Ensure URLs are correct and accessible
- Review error message in modal

**Test Connection Fails**:
- Verify credentials are correct
- Check network connectivity
- Ensure external service is operational
- Review firewall/proxy settings

**Execution Fails**:
- Check execution logs for error details
- Verify integration is enabled
- Confirm API credentials haven't expired
- Test with simpler payload first

**OAuth Authorization Fails**:
- Clear browser cookies and retry
- Check OAuth client ID and secret
- Verify redirect URLs are configured correctly
- Ensure required scopes are requested

### Getting Help

1. **Documentation**: Click the documentation link on integration cards
2. **Execution Logs**: Review detailed error messages and payloads
3. **Test Connection**: Use to isolate configuration issues
4. **Provider Support**: Contact integration provider for API-specific issues

## FAQ

**Q: Can I install the same integration multiple times?**  
A: Yes, you can install multiple instances with different configurations (e.g., different Slack channels or email servers).

**Q: What happens to workflows when I disable an integration?**  
A: Workflow steps using the disabled integration will fail gracefully with an error message.

**Q: Are my API keys stored securely?**  
A: Yes, credentials are encrypted at rest. OAuth tokens are also securely stored.

**Q: Can I export/import integration configurations?**  
A: Currently not supported. You'll need to manually reconfigure integrations in different environments.

**Q: How long are execution logs retained?**  
A: Logs are retained for 90 days by default. Contact support for custom retention policies.

**Q: Can I create custom integrations?**  
A: Yes! See the [Integration Developer Guide](./INTEGRATION_DEVELOPER_GUIDE.md) for details.

---

**Need Help?** Contact support or visit our documentation portal for more resources.
