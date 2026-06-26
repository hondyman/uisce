#!/bin/bash

# Docker Cleanup Script for MacBook
# Removes unused containers, images, volumes, and networks to free up space

set -e

echo "🧹 Docker Cleanup Script"
echo "======================="
echo ""

# Function to show disk usage
show_usage() {
    echo "💾 Current Docker disk usage:"
    docker system df 2>/dev/null || echo "Unable to check disk usage"
    echo ""
}

# Show initial usage
show_usage

echo "🛑 Stopping all running containers..."
docker stop $(docker ps -aq) 2>/dev/null || echo "No running containers to stop"

echo ""
echo "🗑️  Removing all containers..."
docker rm $(docker ps -aq) 2>/dev/null || echo "No containers to remove"

echo ""
echo "🖼️  Removing unused images..."
docker image prune -af

echo ""
echo "📦 Removing unused volumes..."
docker volume prune -f

echo ""
echo "🌐 Removing unused networks..."
docker network prune -f

echo ""
echo "🧽 Running system prune (removes everything unused)..."
docker system prune -af

echo ""
echo "🎯 Final cleanup - remove dangling build cache..."
docker builder prune -af

echo ""
# Show final usage
show_usage

echo "✅ Docker cleanup complete!"
echo ""
echo "💡 Tips:"
echo "   - Run 'docker system df' to check space usage"
echo "   - Run 'docker images' to see remaining images"
echo "   - Consider removing specific large images with 'docker rmi <image_id>'"
echo ""