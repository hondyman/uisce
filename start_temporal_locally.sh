#!/bin/bash

# Start Temporal Server locally (not in Docker)
# This is simpler than dealing with Docker image issues

echo "🚀 Starting Temporal Server..."
temporal server start-dev \
  --headless \
  --db-filename /tmp/temporal_dev.db \
  --ip 0.0.0.0 \
  --port 7233
