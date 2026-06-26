import React, { useEffect, useState } from 'react';
import { devLog } from '../../../utils/devLogger';
import { useParams } from 'react-router-dom';
import useBlockableNavigate from '../../../components/RouteBlocker/useBlockableNavigate';
import { Box, Alert, Link, CircularProgress, Typography } from '@mui/material';
import resolveApiUrl from '../../../utils/resolveApiUrl';
import EnhancedViewEditor from '../../../components/ViewEditor/EnhancedViewEditor';
import { useTenant } from '../../../contexts/TenantContext';

// Helper function to detect if a string is a UUID
const isUUID = (str: string): boolean => {
  const uuidRegex = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;
  return uuidRegex.test(str);
};

const ViewDetailsPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useBlockableNavigate();
  const { tenant, datasource } = useTenant();
  
  const identifier = id ? decodeURIComponent(id) : '';
  const tenantId = tenant?.id || '';
  const datasourceId = (datasource as any)?.id || (datasource as any)?.tenant_instance_id || '';
  const [resolving, setResolving] = useState(false);
  const [resolveError, setResolveError] = useState<string | null>(null);

  if (!identifier) {
    return (
      <Box sx={{ p: 3 }}>
        <div>View identifier is required</div>
      </Box>
    );
  }

  // Determine if the identifier is a UUID or a name
  const isIdUUID = isUUID(identifier);
  const viewId = isIdUUID ? identifier : undefined;
  const viewName = !isIdUUID ? identifier : undefined;

  // If identifier is not a UUID, attempt to resolve it by fetching the view by name
  useEffect(() => {
    let mounted = true;
    const tryResolve = async () => {
      if (!identifier) return;
      if (isIdUUID) return; // nothing to resolve
      // Attempt resolution even for names that may look invalid - backend may still have them
      setResolving(true);
      setResolveError(null);
      try {
  const u = new URL(resolveApiUrl(`/api/views/${encodeURIComponent(identifier)}`));
  if (tenantId) u.searchParams.set('tenant_id', tenantId);
  if (datasourceId) u.searchParams.set('tenant_instance_id', String(datasourceId));
  const res = await fetch(u.toString());
        if (!mounted) return;
        if (res.ok) {
          // Some backends may accidentally append a validation message or stray
          // text after the JSON payload (for example: JSON then "Invalid view name: \"x\"...").
          // Attempt a tolerant parse: read as text, try JSON.parse, and if that
          // fails try to trim trailing non-JSON characters.
          const txt = await res.text();
          let data: any = null;
          try {
            data = JSON.parse(txt);
          } catch (e) {
            // Try to salvage by trimming after the last closing brace
            const last = txt.lastIndexOf('}');
            if (last !== -1) {
              const candidate = txt.slice(0, last + 1);
              try {
                data = JSON.parse(candidate);
              } catch (e2) {
                // Give up; treat as parse failure and surface a friendly message
                setResolveError(`Failed to parse view response from server`);
                if (mounted) setResolving(false);
                return;
              }
            } else {
              setResolveError(`Failed to parse view response from server`);
              if (mounted) setResolving(false);
              return;
            }
          }
          const obj = data.view ?? data;
          // Try common id fields
          const foundId = obj?.id || obj?.uuid || obj?.core_id || obj?.coreId;
          if (foundId) {
            // Navigate to canonical UUID-based URL
            void navigate(`/views/${foundId}`, { replace: true });
            return;
          }
          // If backend returned a view but no id, fall back to name-based loading
          // This can happen when the running backend hasn't been rebuilt to return canonical ids yet.
          try { devLog('View found but no canonical id returned by backend; falling back to name-based load'); } catch {}
          // Allow the page to render the editor using the original name identifier
          setResolveError(null);
          return;
        } else if (res.status >= 400 && res.status < 500) {
          // Treat client errors (including "Invalid view name") as not-found and
          // attempt the list search fallback. This avoids surfacing backend
          // validation errors like "Invalid view name: \"dddddddd\"" and
          // allows the UI to try a fuzzy/alternate resolution.
          try {
            const listUrl = new URL(resolveApiUrl('/api/views'));
            listUrl.searchParams.set('q', identifier);
            if (tenantId) listUrl.searchParams.set('tenant_id', tenantId);
            if (datasourceId) listUrl.searchParams.set('tenant_instance_id', String(datasourceId));
            const listRes = await fetch(listUrl.toString());
            if (listRes.ok) {
              const listData = await listRes.json();
              const viewsArr = Array.isArray(listData.views) ? listData.views : [];
              // try to find exact name match first
              // Match by id, name, or title. Backend may return id or may only provide name/title.
              const match = viewsArr.find((v: any) => ((v.id && String(v.id) === identifier) || (v.name === identifier) || (v.title === identifier))) || viewsArr[0];
              if (match) {
                const foundId2 = match?.id || match?.uuid || match?.core_id || match?.coreId;
                if (foundId2) {
                  void navigate(`/views/${foundId2}`, { replace: true });
                  return;
                }
                // if no id field, but we have a name, navigate with name
                if (match?.name) {
                  void navigate(`/views/${match.name}`, { replace: true });
                  return;
                }
              }
            }
            setResolveError(`View not found: "${identifier}"`);
          } catch (err) {
            setResolveError(`View not found: "${identifier}"`);
          }
        } else {
          const text = await res.text();
          setResolveError(`Failed to resolve view: ${res.status} ${res.statusText} - ${text}`);
        }
      } catch (e: any) {
        setResolveError(`Failed to resolve view: ${e?.message || String(e)}`);
      } finally {
        if (mounted) setResolving(false);
      }
    };
    void tryResolve();
    return () => { mounted = false; };
  // we intentionally include navigate, tenantId, datasourceId
  }, [identifier, isIdUUID, navigate, tenantId, datasourceId]);

  return (
    <Box sx={{ height: '100vh', display: 'flex', flexDirection: 'column' }}>
      {resolving ? (
        <Box sx={{ p: 4, display: 'flex', alignItems: 'center', gap: 2 }}>
          <CircularProgress size={20} />
          <Typography>Resolving view identifier "{identifier}"…</Typography>
        </Box>
      ) : resolveError ? (
        <Box sx={{ p: 3 }}>
          <Alert severity="error" sx={{ mb: 2 }}>{resolveError}</Alert>
          <Box sx={{ mt: 2 }}>
            <Link href="/views" underline="hover">← Back to Views List</Link>
          </Box>
        </Box>
      ) : (
        <EnhancedViewEditor
          viewId={viewId}
          viewName={viewName}
          tenantId={tenantId}
          datasourceId={datasourceId}
          onViewSaved={(savedViewId, viewData) => {
            devLog('View saved:', { savedViewId, viewData });
            // You could add additional logic here like showing notifications
          }}
        />
      )}
    </Box>
  );
};

export default ViewDetailsPage;
