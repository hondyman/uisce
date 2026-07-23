#!/bin/sh

# Run migrations only if the migrate binary exists
if [ -f "/app/migrate" ]; then
  echo "Running migrations..."
  /app/migrate
else
  echo "WARNING: /app/migrate binary not found, skipping migrations"
fi

# Start server
/app/server