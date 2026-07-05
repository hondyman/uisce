import re

with open("backend/internal/api/api.go", "r") as f:
    content = f.read()

target = """	"github.com/hondyman/uisce/backend/internal/health\""""
replacement = """	"github.com/hondyman/uisce/backend/internal/health"
	"github.com/hondyman/uisce/backend/internal/identity\""""

if target in content:
    content = content.replace(target, replacement)
    with open("backend/internal/api/api.go", "w") as f:
        f.write(content)
    print("Patched import")
else:
    print("Could not find target block")
