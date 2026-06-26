#!/bin/bash

# 1. Final email list verification
echo "=== Launch Email Campaign ==="
echo "Date: $(date)"
echo ""

# Count eligible users
TOTAL_USERS=$(psql $PROD_DATABASE_URL -t -c "
SELECT COUNT(*) FROM public.users 
WHERE email_notifications = TRUE 
AND email IS NOT NULL;")

echo "Total eligible users: $TOTAL_USERS"

# 2. Send launch email
# TODO: Call SendGrid API to dispatch the 'launch' template

# 3. Monitor SendGrid dashboard
# Go to: https://app.sendgrid.com/email_activity
# Watch: Sent, Delivered, Opened, Clicked

# 4. Track in real-time
watch -n 60 'curl -s https://api.sendgrid.com/v3/stats \
  -H "Authorization: Bearer $SENDGRID_API_KEY" \
  -d "{\"aggregated_by\": \"day\", \"start_date\": \"$(date +%Y-%m-%d)\"}' | jq .
