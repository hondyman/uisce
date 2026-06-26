#!/bin/bash

# Seed Mock Notification Data
# This script creates sample notifications to populate the notification center

TENANT_ID="910638ba-a459-4a3f-bb2d-78391b0595f6"
DATASOURCE_ID="982aef38-418f-46dc-acd0-35fe8f3b97b0"
USER_ID="a8d74e86-2bb4-44ab-a15d-0e68e9d22788"
API_URL="http://localhost:8080"

echo "🌱 Seeding mock notification data..."
echo "Tenant: $TENANT_ID"
echo "User: $USER_ID"
echo ""

# Function to create a notification
create_notification() {
    local subject="$1"
    local body="$2"
    local channel="$3"
    local priority="$4"
    
    psql -U postgres -d alpha << EOF
INSERT INTO notification_logs (
    id, tenant_id, datasource_id, template_key, recipient_user_id,
    subject, body, channel, status, priority, sent_at, created_at, updated_at
) VALUES (
    gen_random_uuid()::text,
    '$TENANT_ID',
    '$DATASOURCE_ID',
    'manual_notification',
    '$USER_ID',
    '$subject',
    '$body',
    '$channel',
    'sent',
    '$priority',
    NOW() - INTERVAL '$5',
    NOW() - INTERVAL '$5',
    NOW() - INTERVAL '$5'
);
EOF
}

# Create diverse notifications
echo "Creating notifications..."

# Urgent - Recent
create_notification \
    "Action Required: Approve Expense Report" \
    "John Smith has submitted an expense report for Q3 that requires your approval. Total amount: \$4,250.00. Please review and approve by end of day." \
    "email" \
    "urgent" \
    "2 hours"

create_notification \
    "Critical: Workflow 'Q4 Review' Failed" \
    "Step 'Data Validation' returned a critical error. Immediate action required to prevent delays in the quarterly review process." \
    "slack" \
    "urgent" \
    "30 minutes"

# High Priority
create_notification \
    "New feature deployed to production" \
    "The new dashboard analytics feature has been successfully deployed. Please review the changes and provide feedback on the new metrics visualization." \
    "slack" \
    "high" \
    "8 hours"

create_notification \
    "Budget Approval Needed" \
    "The Q1 2024 marketing budget proposal is ready for your review. Total requested: \$125,000. Deadline: Friday EOD." \
    "email" \
    "high" \
    "1 day"

create_notification \
    "Security Alert: Unusual Login Activity" \
    "We detected a login from an unrecognized device in San Francisco, CA. If this wasn't you, please secure your account immediately." \
    "sms" \
    "high" \
    "3 hours"

# Normal Priority
create_notification \
    "Weekly Team Sync Reminder" \
    "Reminder: The weekly sync is scheduled for tomorrow at 10:00 AM PST. Please add your updates to the agenda document." \
    "teams" \
    "normal" \
    "1 day"

create_notification \
    "New Task Assigned: Review User Feedback" \
    "You have been assigned to review user feedback for the mobile app redesign. 47 responses are waiting for your analysis." \
    "email" \
    "normal" \
    "2 days"

create_notification \
    "Document Shared: Q4 Planning" \
    "Sarah Johnson shared 'Q4 Strategic Planning.pdf' with you. The document contains the roadmap for next quarter's initiatives." \
    "email" \
    "normal" \
    "12 hours"

# Low Priority
create_notification \
    "System Maintenance Scheduled" \
    "Scheduled maintenance will occur this Saturday from 2 AM to 4 AM. Services may be temporarily unavailable during this window." \
    "push" \
    "low" \
    "3 days"

create_notification \
    "Workflow 'Daily Backup' Completed" \
    "The daily database backup workflow ran successfully. All data has been backed up to the secure storage location." \
    "email" \
    "low" \
    "1 day"

create_notification \
    "New Feature Available: Dark Mode" \
    "We've added dark mode to the notification center! Click the sun/moon icon in the top right to try it out." \
    "push" \
    "low" \
    "4 hours"

create_notification \
    "Monthly Report Ready" \
    "Your monthly activity report for December is now available. View insights on your productivity and team collaboration." \
    "email" \
    "low" \
    "2 days"

echo ""
echo "✅ Successfully created 12 mock notifications!"
echo ""
echo "To view them, open: http://localhost:5173/core/notifications"
echo ""
echo "Test features:"
echo "  • Toggle dark mode (sun/moon icon)"
echo "  • Open filters and try search"
echo "  • Select notifications with checkboxes"
echo "  • Click 'Approve' or 'Reject' buttons"
echo "  • Mark notifications as read"
