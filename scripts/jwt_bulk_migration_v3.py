#!/usr/bin/env python3

import os
import re
import sys
from pathlib import Path

def patch_go_file(filepath):
    """Patch a single Go file to use JWT claims instead of X-Tenant-ID header."""
    
    with open(filepath, 'r') as f:
        content = f.read()
    
    original_content = content
    
    # Check if file already has jwt-middleware import
    if 'github.com/hondyman/semlayer/libs/jwt-middleware' not in content:
        # Add import - find the import block and add it
        import_match = re.search(r'(import \([\s\S]*?\))', content)
        if import_match:
            # Multi-line import block - add before closing paren
            content = content.replace(
                import_match.group(1),
                import_match.group(1).replace(')', '\t"github.com/hondyman/semlayer/libs/jwt-middleware"\n)')
            )
        else:
            # No import block - create one after package declaration
            if 'package ' in content:
                package_match = re.search(r'(package \w+)', content)
                if package_match:
                    insert_pos = content.find('\n', package_match.end()) + 1
                    content = (content[:insert_pos] + 
                             '\nimport (\n\t"github.com/hondyman/semlayer/libs/jwt-middleware"\n)\n' +
                             content[insert_pos:])
    
    # Pattern 1: Standard net/http handler: tenantID := r.Header.Get("X-Tenant-ID")
    content = re.sub(
        r'tenantID\s*:=\s*r\.Header\.Get\("X-Tenant-ID"\)',
        'claims := jwtmiddleware.GetClaimsFromContext(r)\n\tif claims == nil {\n\t\thttp.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)\n\t\treturn\n\t}\n\ttenantID := claims.TenantID',
        content
    )
    
    # Pattern 2: Gin context: tenantID := c.GetHeader("X-Tenant-ID")  
    content = re.sub(
        r'tenantID\s*:=\s*c\.GetHeader\("X-Tenant-ID"\)',
        'claims := jwtmiddleware.GetGinClaimsFromContext(c)\n\tif claims == nil {\n\t\tc.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"})\n\t\treturn\n\t}\n\ttenantID := claims.TenantID',
        content
    )
    
    # Pattern 3: r.Header.Get("X-Tenant-ID") inline - extract to variable first
    # This is a simple case - inline header access
    content = re.sub(
        r'r\.Header\.Get\("X-Tenant-ID"\)',
        'jwtmiddleware.GetClaimsFromContext(r).TenantID',
        content
    )
    
    # Pattern 4: c.GetHeader("X-Tenant-ID") in Gin - inline
    content = re.sub(
        r'c\.GetHeader\("X-Tenant-ID"\)',
        'jwtmiddleware.GetGinClaimsFromContext(c).TenantID',
        content
    )
    
    # Only write if changed
    if content != original_content:
        with open(filepath, 'w') as f:
            f.write(content)
        return True
    return False

def main():
    work_dir = '/Users/eganpj/GitHub/semlayer'
    os.chdir(work_dir)
    
    print("JWT Bulk Migration v3 (Python-based)")
    print("Target: All Go files with X-Tenant-ID header access")
    print()
    
    # Find all Go files
    go_files = []
    for root, dirs, files in os.walk('.'):
        # Skip vendor, test, and backup files
        dirs[:] = [d for d in dirs if d not in ['vendor', '.git', 'node_modules', '.runtime']]
        
        for file in files:
            if file.endswith('.go') and not file.endswith('_test.go') and not file.endswith('.backup'):
                filepath = os.path.join(root, file)
                # Only process backend, internal, mdm-service, calendar-service
                if any(x in filepath for x in ['backend/', 'mdm-service/', 'calendar-service/', 'internal/']):
                    go_files.append(filepath)
    
    print(f"Found {len(go_files)} Go files to check")
    
    processed = 0
    updated = 0
    
    for filepath in sorted(go_files):
        # Quick check for X-Tenant-ID
        with open(filepath, 'r', errors='ignore') as f:
            if 'X-Tenant-ID' not in f.read():
                continue
        
        processed += 1
        if patch_go_file(filepath):
            updated += 1
            print(f"✓ {filepath}")
        else:
            print(f"  {filepath} (already patched)")
    
    print()
    print(f"Files processed: {processed}")
    print(f"Files updated: {updated}")
    print()
    print("Complete! Review changes with: git diff")

if __name__ == '__main__':
    main()
