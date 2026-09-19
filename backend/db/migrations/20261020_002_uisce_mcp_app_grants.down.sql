REVOKE SELECT ON public.page_definitions FROM uisce_mcp_app;
REVOKE SELECT ON public.business_objects FROM uisce_mcp_app;
REVOKE SELECT ON public.business_object_fields FROM uisce_mcp_app;
REVOKE SELECT ON public.catalog_edge FROM uisce_mcp_app;
REVOKE SELECT ON public.tenants FROM uisce_mcp_app;
-- Role itself is retained (password/DSN out-of-band); drop only via explicit ops.
