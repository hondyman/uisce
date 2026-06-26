import { chromium } from 'playwright';

(async () => {
  const browser = await chromium.launch();
  const ctx = await browser.newContext();
  const page = await ctx.newPage();

  const seen = [];

  page.on('request', (req) => {
    const url = req.url();
    if (url.includes('/api/')) {
      seen.push({ type: 'request', url, method: req.method(), headers: req.headers(), resourceType: req.resourceType(), timestamp: Date.now() });
      console.log('[REQ]', req.method(), url);
    }
  });

  page.on('response', async (res) => {
    const url = res.url();
    if (url.includes('/api/')) {
      let status = res.status();
      let text = '';
      try { text = await res.text(); if (text.length>400) text = text.slice(0,400) + '...'; } catch(e) {}
      seen.push({ type: 'response', url, status, text, timestamp: Date.now() });
      console.log('[RES]', status, url);
    }
  });

  const target = process.env.TARGET || 'http://localhost:5173/';
  console.log('Opening', target, 'and recording network for 20s...');

  try {
    await page.goto(target, { waitUntil: 'load', timeout: 30000 });
  } catch (e) {
    console.error('goto failed:', e && e.message ? e.message : e);
  }

  // Wait and let the app make requests (also interact a bit)
  await page.waitForTimeout(5000);

  // Try to open view editor if route exists (best-effort)
  try {
    await page.evaluate(() => {
      // Best-effort: click first element that looks like a view link/button
      const candidate = document.querySelector('a[href^="/views"], button[data-test-id="open-view-editor"], a[href*="view"]');
      if (candidate) { candidate.click(); }
    });
  } catch (e) {}

  // Wait more to capture XHRs
  await page.waitForTimeout(15000);

  // Print summary
  console.log('\n--- Network summary ---\n');
  const apiHits = seen.filter(s => s.url && s.url.includes('/api/'));
  const fabricCalls = apiHits.filter(s => s.url.includes('/api/fabric_defn') || s.url.includes('/fabric_defn'));

  console.log('Total /api events captured:', apiHits.length);
  console.log('fabric_defn related events:', fabricCalls.length);

  if (fabricCalls.length > 0) {
    console.log('\nDetails for fabric_defn calls:');
    fabricCalls.forEach((c, i) => console.log(i+1, c.type || '', c.url, c.method || '', c.status || ''));
  }

  if (apiHits.length > 0) {
    console.log('\nAll unique /api endpoints seen:');
    const uniq = Array.from(new Set(apiHits.map(s => s.url))).slice(0,200);
    uniq.forEach(u => console.log(' -', u));
  }

  await browser.close();
  process.exit(0);
})();
