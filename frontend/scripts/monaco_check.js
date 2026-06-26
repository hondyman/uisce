const { chromium } = require('playwright');

(async () => {
  const url = 'http://localhost:5173/';
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();

  const consoleMsgs = [];
  const failedRequests = [];

  page.on('console', (msg) => {
    consoleMsgs.push({ type: msg.type(), text: msg.text() });
  });

  page.on('pageerror', (err) => {
    consoleMsgs.push({ type: 'pageerror', text: err.message });
  });

  page.on('requestfailed', (req) => {
    failedRequests.push({ url: req.url(), failure: req.failure().errorText });
  });

  // Navigate to the app and wait
  try {
    await page.goto(url, { waitUntil: 'networkidle' , timeout: 30000});

    // Try to navigate to ModelBuilder page if route exists
    // Attempt a few likely paths
    const routes = ['/','/model-builder','/models','/model-builder-page','/src/pages/ModelBuilderPage.tsx'];
    for (const r of routes) {
      try {
        const full = url.replace(/\/$/, '') + r;
        await page.goto(full, { waitUntil: 'networkidle', timeout: 10000 });
      } catch (e) {}
    }

    // wait a bit for dynamic loads
    await page.waitForTimeout(2000);
  } catch (e) {
    console.error('Navigation error:', e.message);
  }

  console.error('--- CONSOLE MESSAGES ---');
  consoleMsgs.forEach((m) => console.error(m.type, ':', m.text));
  console.error('--- FAILED REQUESTS ---');
  failedRequests.forEach((f) => console.error(f.url, '->', f.failure));

  await browser.close();
})();
