import re

with open("backend/internal/api/api.go", "r") as f:
    content = f.read()

target = """	tenantID := ""
	if id, ok := identity.TenantIDFromContext(r.Context()); ok && id != "" {
		tenantID = id
	} else if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil && claims.TenantID != "" {
		tenantID = claims.TenantID
	}
	if h := r.Header.Get("X-Tenant-ID"); h != "" {
		tenantID = h
	} else if q := r.URL.Query().Get("tenant_id"); q != "" {
		tenantID = q
	}
	
	if tenantID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}"""

replacement = """	tenantID := ""
	if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil && claims.TenantID != "" {
		tenantID = claims.TenantID
	}
	if h := r.Header.Get("X-Tenant-ID"); h != "" {
		tenantID = h
	} else if q := r.URL.Query().Get("tenant_id"); q != "" {
		tenantID = q
	}
	
	if tenantID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}"""

if target in content:
    content = content.replace(target, replacement)
    with open("backend/internal/api/api.go", "w") as f:
        f.write(content)
    print("Patched tenantID logic")
else:
    print("Could not find target block")
