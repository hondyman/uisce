import re

with open("backend/internal/api/api.go", "r") as f:
    content = f.read()

target = """	// For debugging, allow querying without headers
	if tenantID == "" || tenantDatasourceID == "" {"""

replacement = """	// For debugging, allow querying without headers
	if tenantID == "" {"""

if target in content:
    content = content.replace(target, replacement)
    with open("backend/internal/api/api.go", "w") as f:
        f.write(content)
    print("Patched debug block")
else:
    print("Could not find target block")
