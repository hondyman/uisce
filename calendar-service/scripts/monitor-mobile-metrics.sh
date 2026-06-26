#!/bin/bash

echo "📱 Mobile App Metrics Monitor"
echo "============================="
echo "Time: $(date)"
echo ""

# App Store Connect (iOS)
echo "=== iOS App Store ==="
echo "Downloads: [Check App Store Connect]"
echo "Rating: [Check App Store Connect]"
echo "Crashes: [Check Crashlytics]"

# Google Play Console (Android)
echo ""
echo "=== Android Play Store ==="
echo "Downloads: [Check Play Console]"
echo "Rating: [Check Play Console]"
echo "Crashes: [Check Crashlytics]"

# Crashlytics
echo ""
echo "=== Crash Reports (Last 24h) ==="
echo "Go to: https://console.firebase.google.com"
echo "Select project -> Crashlytics"
echo "Check: Issues, Users affected"

# Target: <0.5% crash-free users

echo ""
echo "=== User Feedback ==="
echo "- Check App Store reviews"
echo "- Check Play Store reviews"
echo "- Check support tickets"

echo ""
echo "Last updated: $(date)"
