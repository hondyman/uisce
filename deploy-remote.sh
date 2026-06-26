\#!/bin/bash

# Remote Infrastructure Deployment Script
# Fixed for macOS, Tailscale, and Code Syncing

set -e

REMOTE_HOST="100.84.126.19"
REMOTE_USER="${REMOTE_USER:-eganpj}"
REMOTE_DIR="${REMOTE_DIR:-semlayer}"

# Password handling - prompt once and reuse
if [ -z "$REMOTE_PASSWORD" ]; then
    echo -n "Enter remote server password: "
    read -s REMOTE_PASSWORD
    echo ""
    export REMOTE_PASSWORD
fi

echo "🚀 Deploying SemLayer Remote Infrastructure"
echo "=========================================="
echo "Remote Host: $REMOTE_HOST"
echo "Remote User: $REMOTE_USER"
echo "Remote Dir: $REMOTE_DIR"
echo ""

# Check if Tailscale is running (Compatible with macOS and Linux)
echo "📡 Checking Tailscale connection..."
if ! tailscale status >/dev/null 2>&1; then
    echo "⚠️  Tailscale not detected by CLI."
    echo "   Checking for Tailscale GUI process (macOS workaround)..."
    if ! pgrep -x "Tailscale" > /dev/null; then
        echo "❌ Tailscale app does not appear to be running."
        echo "   Please start the Tailscale app and connect."
        exit 1
    fi
fi

# Test connection to remote host
echo "🔗 Testing connection to remote host..."
if ! sshpass -p "$REMOTE_PASSWORD" ssh -o ConnectTimeout=5 "$REMOTE_USER@$REMOTE_HOST" "echo 'Connection successful'" 2>/dev/null; then
    echo "❌ Cannot connect to $REMOTE_HOST"
    echo "   - Ensure Tailscale is running on remote host"
    echo "   - Check SSH key authentication"
    echo "   - Verify Tailscale network connectivity"
    exit 1
fi

# Create remote directory
echo "📁 Creating remote directory..."
sshpass -p "$REMOTE_PASSWORD" ssh "$REMOTE_USER@$REMOTE_HOST" "mkdir -p $REMOTE_DIR"

# Copy compose file
echo "📋 Copying docker-compose.remote.yml..."
sshpass -p "$REMOTE_PASSWORD" scp docker-compose.remote.yml "$REMOTE_USER@$REMOTE_HOST:$REMOTE_DIR/"

# NEW: Sync application code folders
# This ensures the 'backend' folder exists for the Docker build
echo "📂 Syncing application code to remote..."
# List all folders that your Dockerfile needs to build
for dir in "backend" "frontend" "services" "schema" "calc-engine" "libs"; do
    if [ -d "$dir" ]; then
        echo "   - Syncing $dir..."
        # Using rsync if available for speed, falling back to scp
        if command -v rsync >/dev/null 2>&1; then
            sshpass -p "$REMOTE_PASSWORD" rsync -avz --exclude 'node_modules' --exclude '.git' -e "sshpass -p '$REMOTE_PASSWORD' ssh" "$dir" "$REMOTE_USER@$REMOTE_HOST:$REMOTE_DIR/"
        else
            sshpass -p "$REMOTE_PASSWORD" scp -r "$dir" "$REMOTE_USER@$REMOTE_HOST:$REMOTE_DIR/"
        fi
    fi
done

# Copy Trino catalog configuration if it exists
if [ -d "trino/etc/catalog" ]; then
    echo "📋 Copying Trino catalog configuration..."
    sshpass -p "$REMOTE_PASSWORD" ssh "$REMOTE_USER@$REMOTE_HOST" "mkdir -p $REMOTE_DIR/trino/etc"
    sshpass -p "$REMOTE_PASSWORD" scp -r trino/etc/catalog "$REMOTE_USER@$REMOTE_HOST:$REMOTE_DIR/trino/etc/"
fi

# Deploy services
# Added --build to ensure code changes are picked up
echo "🐳 Starting remote services..."
sshpass -p "$REMOTE_PASSWORD" ssh "$REMOTE_USER@$REMOTE_HOST" "cd $REMOTE_DIR && docker compose -f docker-compose.remote.yml up -d --build"

# Wait for services to start
echo "⏳ Waiting for services to initialize..."
sleep 15

# Check service status
echo "📊 Checking service status..."
sshpass -p "$REMOTE_PASSWORD" ssh "$REMOTE_USER@$REMOTE_HOST" "cd $REMOTE_DIR && docker compose -f docker-compose.remote.yml ps"

echo ""
echo "✅ Remote infrastructure deployment complete!"
echo ""
echo "🌐 Service Endpoints:"
echo "   - Redpanda (Kafka): $REMOTE_HOST:9092"
echo "   - Temporal UI: $REMOTE_HOST:8086"
echo "   - MinIO Console: $REMOTE_HOST:9001"
echo "   - Trino: $REMOTE_HOST:8084"
echo "   - Redis: $REMOTE_HOST:6379"
echo ""
