// Simple integration test script: save a test view and assert the server returns an id
// Usage:
//   API_BASE=http://localhost:8001/api node scripts/test-save-view.js <tenant_id> <datasource_id>

const API_BASE = process.env.API_BASE || 'http://localhost:8001/api';
const [,, tenantId, datasourceId] = process.argv;

if (!tenantId || !datasourceId) {
  console.log('Usage: API_BASE=http://localhost:8001/api node scripts/test-save-view.js <tenant_id> <datasource_id>');
  process.exit(0);
}

(async () => {
  try {
    // Build a minimal test view
    const testName = `test_view_${Date.now()}`;
    const viewPayload = {
      name: testName,
      title: `Test View ${Date.now()}`,
      description: 'Integration test view',
      cubes: [],
      dimensions: [],
      measures: [],
    };

    const url = `${API_BASE}/views/${encodeURIComponent(testName)}?tenant_id=${encodeURIComponent(tenantId)}&datasource_id=${encodeURIComponent(datasourceId)}`;
    console.log('Saving test view to', url);
    const resp = await fetch(url, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(viewPayload),
    });

    if (!resp.ok) {
      const txt = await resp.text();
      console.error('Save failed', resp.status, resp.statusText, txt);
      process.exit(2);
    }

    const body = await resp.json();
    // Expect the server to return a view object or wrapper containing the view
    const view = body.view || body || null;
    if (!view) {
      console.error('No view returned in response body');
      process.exit(3);
    }

    const id = view.id || view.uuid || view.core_id || view.coreId || null;
    if (!id) {
      console.error('Saved view did not include an id field in response:', JSON.stringify(view, null, 2));
      process.exit(4);
    }

    console.log('Saved view id:', id);

    // Now GET the view by id
    const getUrl = `${API_BASE}/views/${encodeURIComponent(id)}?tenant_id=${encodeURIComponent(tenantId)}&datasource_id=${encodeURIComponent(datasourceId)}`;
    const getResp = await fetch(getUrl);
    if (!getResp.ok) {
      console.error('GET by id failed', getResp.status, getResp.statusText);
      process.exit(5);
    }
    const getBody = await getResp.json();
    const fetched = getBody.view || getBody;
    if (!fetched) {
      console.error('GET response did not contain view');
      process.exit(6);
    }

    console.log('GET by id returned view with name:', fetched.name || fetched.title || '<no-name>');
    console.log('Integration test succeeded.');
    process.exit(0);
  } catch (err) {
    console.error('Test failed:', err);
    process.exit(10);
  }
})();
