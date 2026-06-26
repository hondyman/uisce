#!/bin/bash

# Cron Job Setup for Preaggregation
# This script sets up automated cron jobs for preaggregation

echo "⏰ Setting up Preaggregation Cron Jobs"
echo "====================================="

SCRIPT_PATH="/Users/eganpj/GitHub/semlayer/run_preaggregation.sh"
LOG_FILE="/Users/eganpj/GitHub/semlayer/logs/preaggregation.log"

# Create logs directory if it doesn't exist
mkdir -p "$(dirname "$LOG_FILE")"

echo "📍 Script path: $SCRIPT_PATH"
echo "📝 Log file: $LOG_FILE"

# Check if script exists and is executable
if [ ! -f "$SCRIPT_PATH" ]; then
    echo "❌ Preaggregation script not found at $SCRIPT_PATH"
    exit 1
fi

if [ ! -x "$SCRIPT_PATH" ]; then
    echo "❌ Preaggregation script is not executable"
    exit 1
fi

echo "✅ Preaggregation script is ready"

# Display current cron jobs
echo ""
echo "📋 Current cron jobs:"
crontab -l 2>/dev/null || echo "No existing cron jobs"

# Add new cron job
echo ""
echo "🔧 Adding preaggregation cron job..."

# Create temporary crontab file
TEMP_CRON=$(mktemp)

# Export existing crontab
crontab -l 2>/dev/null > "$TEMP_CRON"

# Add new job (daily at 6 AM)
echo "# Semlayer Preaggregation Job - Daily at 6:00 AM" >> "$TEMP_CRON"
echo "0 6 * * * $SCRIPT_PATH >> $LOG_FILE 2>&1" >> "$TEMP_CRON"

# Install new crontab
crontab "$TEMP_CRON"

# Clean up
rm "$TEMP_CRON"

echo "✅ Cron job added successfully!"
echo ""
echo "📅 Schedule: Daily at 6:00 AM"
echo "🚀 Command: $SCRIPT_PATH"
echo "📝 Logs: $LOG_FILE"

# Verify the cron job was added
echo ""
echo "🔍 Verifying cron job installation:"
crontab -l | grep -A1 "Semlayer Preaggregation"

echo ""
echo "🎉 Setup complete!"
echo ""
echo "💡 Additional cron job examples:"
echo "  # Weekly preaggregation (Mondays at 6 AM)"
echo "  0 6 * * 1 $SCRIPT_PATH >> $LOG_FILE 2>&1"
echo ""
echo "  # Monthly preaggregation (1st of month at 6 AM)"
echo "  0 6 1 * * $SCRIPT_PATH >> $LOG_FILE 2>&1"
echo ""
echo "  # Custom schedule - Every 4 hours"
echo "  0 */4 * * * $SCRIPT_PATH >> $LOG_FILE 2>&1"

# Test the cron job immediately (optional)
echo ""
read -p "🧪 Would you like to test the preaggregation script now? (y/n): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "🚀 Running preaggregation test..."
    "$SCRIPT_PATH"
fi
