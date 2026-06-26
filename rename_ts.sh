#!/bin/bash

# This script renames .jsx and .js files in the frontend/src directory
# to .tsx and .ts respectively.

echo "Renaming .jsx files to .tsx..."
# Find all .jsx files and rename them to .tsx
for file in $(find ./frontend/src -name "*.jsx"); do
    mv -- "$file" "${file%.jsx}.tsx"
done

echo "Renaming .js files to .ts..."
# Find all .js files and rename them to .ts
for file in $(find ./frontend/src -name "*.js"); do
    # We exclude config files that might be at the root
    if [[ "$file" != *"eslint.config.js"* && "$file" != *"vite.config.js"* ]]; then
        mv -- "$file" "${file%.js}.ts"
    fi
done

echo "File renaming complete! ✨"
