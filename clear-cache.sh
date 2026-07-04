#!/bin/bash

# Force clear Vite cache and restart dev server

echo "🧹 Clearing Vite cache..."
rm -rf frontend/node_modules/.vite
rm -rf frontend/dist

echo "🔄 Restarting dev server..."
echo "Please run: cd frontend && npm run dev"
echo ""
echo "Then in your browser:"
echo "1. Open DevTools (Cmd+Option+I on Mac)"
echo "2. Right-click the refresh button"
echo "3. Select 'Empty Cache and Hard Reload'"
echo "   OR press Cmd+Shift+R (Mac) / Ctrl+Shift+R (Windows)"
