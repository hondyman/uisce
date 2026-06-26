#!/bin/bash

# ============================================================================
# Remote Server Recovery Guide
# ============================================================================
# 
# The remote server (ubuntu-2 at 100.84.126.19) has gone offline.
# This script helps diagnose and recover the remote infrastructure.
#
# Prerequisites:
# - SSH access to the remote server
# - Tailscale or direct network access
# - Admin/sudo privileges on remote
#
# Usage: bash scripts/remote_recovery.sh
# ============================================================================

set -e

REMOTE_HOST="${REMOTE_HOST:-100.84.126.19}"
REMOTE_USER="${REMOTE_USER:-ubuntu}"
BOLD='\033[1m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_remote_access() {
    log_info "Checking SSH access to $REMOTE_HOST..."
    
    if timeout 3 ssh -o ConnectTimeout=3 "$REMOTE_USER@$REMOTE_HOST" "echo OK" > /dev/null 2>&1; then
        log_info "✓ SSH access working"
        return 0
    else
        log_error "Cannot SSH to $REMOTE_HOST"
        log_warn "Possible solutions:"
        log_warn "1. Check Tailscale connection: tailscale status"
        log_warn "2. Restart Tailscale on remote: sudo systemctl restart tailscaled"
        log_warn "3. Access via physical console if available"
        return 1
    fi
}

recover_remote() {
    log_info "Attempting remote recovery..."
    
    # Check Tailscale
    log_info "Checking Tailscale daemon..."
    ssh "$REMOTE_USER@$REMOTE_HOST" sudo systemctl status tailscaled || {
        log_warn "Tailscale daemon not running, attempting restart..."
        ssh "$REMOTE_USER@$REMOTE_HOST" sudo systemctl restart tailscaled
    }
    
    # Check PostgreSQL
    log_info "Checking PostgreSQL..."
    ssh "$REMOTE_USER@$REMOTE_HOST" sudo systemctl status postgresql || {
        log_warn "PostgreSQL not running, attempting restart..."
        ssh "$REMOTE_USER@$REMOTE_HOST" sudo systemctl restart postgresql
    }
    
    # Check Docker and docker-compose
    log_info "Checking Docker services..."
    ssh "$REMOTE_USER@$REMOTE_HOST" docker compose -f /path/to/docker-compose.remote.yml ps || {
        log_warn "Docker compose not responding"
    }
    
    log_info "Recovery commands sent to remote server"
    log_info "Waiting for services to start..."
    sleep 10
}

verify_recovery() {
    log_info "Verifying recovery..."
    
    # Test ping
    if timeout 3 ping -c 1 "$REMOTE_HOST" > /dev/null 2>&1; then
        log_info "✓ Ping successful"
    else
        log_error "Ping failed"
        return 1
    fi
    
    # Test SSH
    if timeout 3 ssh -o ConnectTimeout=3 "$REMOTE_USER@$REMOTE_HOST" "echo OK" > /dev/null 2>&1; then
        log_info "✓ SSH working"
    else
        log_error "SSH failed"
        return 1
    fi
    
    # Test Postgres
    if timeout 3 ssh "$REMOTE_USER@$REMOTE_HOST" "sudo systemctl is-active postgresql" > /dev/null 2>&1; then
        log_info "✓ PostgreSQL running"
    else
        log_error "PostgreSQL not running"
        return 1
    fi
    
    log_info "Recovery verification complete"
    return 0
}

# Main
if check_remote_access; then
    recover_remote
    verify_recovery
else
    log_error "Cannot access remote server. Please fix connectivity first."
    exit 1
fi
