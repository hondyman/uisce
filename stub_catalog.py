#!/usr/bin/env python3
import re

with open('/Users/eganpj/GitHub/semlayer/backend/internal/api/api.go', 'r') as f:
    content = f.read()

# Remove unused tenantID declarations
content = re.sub(r'\n\s+tenantID := strings\.TrimSpace\(r\.URL\.Query\(\)\.Get\("tenant_id"\)\)\n(\s+datasourceID)', r'\n\1', content)

# Stub out all catalog.Cubes accesses by making the if false
content = re.sub(
    r'for _, c := range catalog\.Cubes \{',
    '_ = catalog // STUB\n\t\t\tfor _, c := range []cube.Cube{} {',
    content
)

# Comment out cube.Dimensions and cube.Measures accesses inside if false blocks
lines = content.split('\n')
in_false_block = False
fixed_lines = []
for i, line in enumerate(lines):
    if 'if false { // cube, exists := catalog.Cubes' in line:
        in_false_block = True
        fixed_lines.append(line)
    elif in_false_block and ('cube.Dimensions' in line or 'cube.Measures' in line):
        # Comment out the line
        fixed_lines.append(line.replace('for ', '// for '))
    elif in_false_block and line.strip() and not line.strip().startswith('//') and line.count('}') > line.count('{'):
        # Count closing braces to detect end of if false block
        in_false_block = False
        fixed_lines.append(line)
    else:
        fixed_lines.append(line)

content = '\n'.join(fixed_lines)

with open('/Users/eganpj/GitHub/semlayer/backend/internal/api/api.go', 'w') as f:
    f.write(content)

print("Catalog stubs applied")
