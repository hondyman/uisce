#!/bin/bash

# 1. Define and create the target directory
TARGET_DIR="docs/project_history"
mkdir -p "$TARGET_DIR"

echo "Moving documentation files to $TARGET_DIR..."

# 2. Move all .md and .txt files from root to the target directory
# We use find -maxdepth 1 to ensure we don't accidentally move files 
# that are already organized in subfolders.
find . -maxdepth 1 -name "*.md" -exec mv {} "$TARGET_DIR/" \;
find . -maxdepth 1 -name "*.txt" -exec mv {} "$TARGET_DIR/" \;

echo "Cleanup complete! Root directory is now breathable."
