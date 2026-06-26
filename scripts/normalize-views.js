#!/usr/bin/env node
/*
  scripts/normalize-views.js
  - Fetches all views via /api/views
  - For each view: ensures cubes are [{id,name}], join_paths are [{id,path,label}], extends is UUID when found in list
  - PUTs normalized view back to /api/views/{name}?tenant_id=...&datasource_id=...

  Usage: node scripts/normalize-views.js <tenant_id> <datasource_id>
*/
// Use global fetch available in modern Node.js (18+). Avoid external dependency on node-fetch.
const fetch = global.fetch;
const [,, tenantId, datasourceId] = process.argv;
if (!tenantId || !datasourceId) {
  console.error('Usage: node scripts/normalize-views.js <tenant_id> <datasource_id>');
  process.exit(2);
}
const API = process.env.API_BASE || 'http://localhost:5175/api';
(async function main(){
  try{
    const listRes = await fetch(`${API}/views?tenant_id=${encodeURIComponent(tenantId)}&datasource_id=${encodeURIComponent(datasourceId)}&page_size=500`);
    const list = await listRes.json();
    const views = list.views || [];
    console.log(`Found ${views.length} views`);

    // Fetch cubes
    const cubesRes = await fetch(`${API}/fabric/models?tenant_id=${encodeURIComponent(tenantId)}&datasource_id=${encodeURIComponent(datasourceId)}`);
    const cubesData = await cubesRes.json();
    const cubes = cubesData.models || [];

    const nameToId = {}; // map name/title to id for extends
    views.forEach(v => { if (v.name) nameToId[(v.name||'').toLowerCase()] = v.id; if (v.title) nameToId[(v.title||'').toLowerCase()] = v.id; });

    for (const v of views) {
      const viewName = v.name;
      // fetch full view
      const getRes = await fetch(`${API}/views/${encodeURIComponent(viewName)}?tenant_id=${encodeURIComponent(tenantId)}&datasource_id=${encodeURIComponent(datasourceId)}`);
      if (!getRes.ok) { console.warn(`Failed to fetch view ${viewName}: ${getRes.status}`); continue; }
      const data = await getRes.json();
      const view = data.view || data;

      // Normalize cubes
      view.cubes = (view.cubes||[]).map(c => {
        if (!c) return c;
        if (typeof c === 'string') {
          const found = cubes.find(cm => String(cm.model_key) === String(c) || String(cm.id) === String(c));
          return { id: c, name: found?.display_name || found?.model_key || c };
        }
        if (c.id && !c.name) {
          const found = cubes.find(cm => String(cm.id) === String(c.id));
          return { ...c, name: found?.display_name || found?.model_key || c.id };
        }
        return c;
      });

      // Normalize join_paths
      view.join_paths = (view.join_paths||[]).map(jp => {
        if (!jp) return jp;
        if (typeof jp === 'string') {
          const found = cubes.find(cm => String(cm.model_key) === String(jp) || String(cm.id) === String(jp));
          return { id: found?.id || jp, path: jp, label: found?.display_name || jp };
        }
        if (jp.id && !jp.path) return { id: jp.id, path: jp.path || jp.label || jp.id, label: jp.label || jp.id };
        return jp;
      });

      // Normalize extends
      if (view.extends && typeof view.extends === 'string') {
        const key = view.extends.toLowerCase();
        if (!/^[0-9a-f-]{36}$/.test(view.extends) && nameToId[key]) {
          view.extends = nameToId[key];
        }
      }

      // PUT back
      const putRes = await fetch(`${API}/views/${encodeURIComponent(viewName)}?tenant_id=${encodeURIComponent(tenantId)}&datasource_id=${encodeURIComponent(datasourceId)}`, {
        method: 'PUT', headers: {'Content-Type':'application/json'}, body: JSON.stringify(view)
      });
      if (!putRes.ok) { console.warn(`Failed to update view ${viewName}: ${putRes.status} ${await putRes.text()}`); }
      else console.log(`Updated ${viewName}`);
    }
    console.log('Done');
  } catch(e){ console.error(e); process.exit(1); }
})();
