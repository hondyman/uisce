import { chromium } from 'playwright';

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
    const failure = req.failure();
    failedRequests.push({ url: req.url(), failure: failure ? failure.errorText : 'unknown' });
  });

  try {
    await page.goto(url, { waitUntil: 'networkidle' , timeout: 30000});

    // Try a few likely model builder routes
    const routes = ['/','/model-builder','/models','/model-builder-page'];
    for (const r of routes) {
      try {
        const full = url.replace(/\/$/, '') + r;
        await page.goto(full, { waitUntil: 'networkidle', timeout: 10000 });
      } catch (e) {}
    }

    await page.waitForTimeout(2000);
  } catch (e) {
    console.error('Navigation error:', e.message);
  }

  console.log('--- CONSOLE MESSAGES ---');
  consoleMsgs.forEach((m) => console.log(m.type, ':', m.text));
  console.log('--- FAILED REQUESTS ---');
  failedRequests.forEach((f) => console.log(f.url, '->', f.failure));

  await browser.close();
})();
