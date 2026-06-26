#!/bin/bash

echo "📊 Email Metrics Monitor"
echo "========================"
echo "Time: $(date)"
echo ""

# Get metrics from SendGrid API
METRICS=$(curl -s https://api.sendgrid.com/v3/stats \
  -H "Authorization: Bearer $SENDGRID_API_KEY" \
  -d "{
    \"aggregated_by\": \"hour\",
    \"start_date\": \"$(date +%Y-%m-%d)\",
    \"end_date\": \"$(date +%Y-%m-%d)\"
  }")

# Extract key metrics
SENT=$(echo $METRICS | jq '.stats[0].metrics.sent')
DELIVERED=$(echo $METRICS | jq '.stats[0].metrics.delivered')
OPENED=$(echo $METRICS | jq '.stats[0].metrics.unique_opens')
CLICKED=$(echo $METRICS | jq '.stats[0].metrics.unique_clicks')
BOUNCED=$(echo $METRICS | jq '.stats[0].metrics.bounce')
UNSUBSCRIBED=$(echo $METRICS | jq '.stats[0].metrics.unsubscribe')

# Calculate rates
if [ "$SENT" -gt 0 ]; then
    DELIVERY_RATE=$(echo "scale=2; $DELIVERED * 100 / $SENT" | bc)
    OPEN_RATE=$(echo "scale=2; $OPENED * 100 / $DELIVERED" | bc)
    CLICK_RATE=$(echo "scale=2; $CLICKED * 100 / $OPENED" | bc)
    UNSUB_RATE=$(echo "scale=4; $UNSUBSCRIBED * 100 / $DELIVERED" | bc)
else
    DELIVERY_RATE=0
    OPEN_RATE=0
    CLICK_RATE=0
    UNSUB_RATE=0
fi

# Display metrics
echo "=== Email Campaign Metrics ==="
echo "Sent:          $SENT"
echo "Delivered:     $DELIVERED ($DELIVERY_RATE%)"
echo "Opened:        $OPENED ($OPEN_RATE%)"
echo "Clicked:       $CLICKED ($CLICK_RATE%)"
echo "Bounced:       $BOUNCED"
echo "Unsubscribed:  $UNSUBSCRIBED ($UNSUB_RATE%)"
echo ""

# Check thresholds
echo "=== Threshold Checks ==="
if (( $(echo "$DELIVERY_RATE < 95" | bc -l) )); then
    echo "⚠️  WARNING: Delivery rate below 95%"
else
    echo "✅ Delivery rate OK ($DELIVERY_RATE%)"
fi

if (( $(echo "$OPEN_RATE < 30" | bc -l) )); then
    echo "⚠️  WARNING: Open rate below 30%"
else
    echo "✅ Open rate OK ($OPEN_RATE%)"
fi

if (( $(echo "$UNSUB_RATE > 0.5" | bc -l) )); then
    echo "⚠️  WARNING: Unsubscribe rate above 0.5%"
else
    echo "✅ Unsubscribe rate OK ($UNSUB_RATE%)"
fi

echo ""
echo "Last updated: $(date)"
