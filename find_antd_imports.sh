#!/bin/bash
# Script to identify all AntD imports in the project

echo "🔍 Finding all AntD imports in frontend/src..."
echo ""

# Find all tsx/ts files with antd imports
grep -r "from ['\"]antd['\"]" frontend/src --include="*.tsx" --include="*.ts" | grep -v node_modules | grep -v ".bak" | sort

echo ""
echo "📊 Summary:"
grep -r "from ['\"]antd['\"]" frontend/src --include="*.tsx" --include="*.ts" | grep -v node_modules | grep -v ".bak" | wc -l | xargs echo "Total files with AntD imports:"
