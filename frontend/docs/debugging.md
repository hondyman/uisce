# Debugging

## Console Noise from Browser Extensions

### MaxListenersExceededWarning + ObjectMultiplex orphaned data

If you see `MaxListenersExceededWarning` (11 close/end listeners) and `ObjectMultiplex - orphaned data for stream` warnings in the browser console, these originate from browser extensions (typically MetaMask or another wallet extension) running liveness probes via `contentscript.js`. **These are not application errors** and require no action on the application side.

To suppress while debugging: disable wallet/MetaMask extensions in your browser's developer tools Extensions panel, then reload.

## Previously Fixed Console Errors

The following errors were fixed as of July 2026. If you still see them, the backend may need to be redeployed.

### GET /api/tenants/{id}/ip-whitelist → 404
**Fixed by**: adding `/api/tenants/{tenantId}/ip-whitelist` alias routes in `ip_whitelist_handlers.go`.
**Symptom**: 13 × 404 on `/fabric/tenants` page load.
**Root cause**: Frontend called `/api/tenants/{id}/ip-whitelist` but backend only had `/api/tenant/ip-whitelist` (singular, no path param).

### GET /api/lookups/regions/values → 404
**Fixed by**: adding dedicated `/api/lookups/regions/values` endpoint in `lookups_routes.go`.
**Symptom**: 2 × 404 on page load.
**Root cause**: No `regions` lookup existed in DB; handler returned "lookup not found". Added hardcoded endpoint returning the four AWS-style region codes.

### GET /api/tenants/{id} → 404
**Fixed by**: adding `GET /api/tenants/{tenantId}` handler in `tenant_access_handlers.go`.
**Symptom**: Tenant detail page failed to load with "Failed to load tenant: API Error: 404".
**Root cause**: Frontend called `/api/tenants/{id}` but backend only had `/api/tenant` (relied on X-Tenant-ID header).

### aria-hidden focus accessibility warning
**Fixed by**: adding `disableEnforceFocus` to the 3 Dialog instances in `TenantDetailPageV2.tsx`.
**Symptom**: `Blocked aria-hidden on an element because its descendant retained focus` console warning.
**Root cause**: MUI Dialog closes while focus returns to triggering button — known MUI issue.
