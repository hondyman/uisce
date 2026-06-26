import { chromium } from 'playwright';

const url = process.argv[2] || 'http://localhost:5173/core/domains';

(async () => {
  const browser = await chromium.launch();
  const page = await browser.newPage();

  page.on('console', (msg) => {
    console.log(`[console:${msg.type()}] ${msg.text()}`);
  });
  page.on('pageerror', (err) => {
    console.log(`[pageerror] ${err.message}`);
    console.log(err.stack);
  });
  page.on('requestfailed', (req) => {
    const f = req.failure();
    console.log(`[requestfailed] ${req.url()} ${f ? f.errorText : 'unknown'}`);
  });

  console.log('Opening', url);
  try {
    await page.goto(url, { waitUntil: 'load', timeout: 45000 });
  } catch (err) {
    console.error('goto failed:', (err && err.message) || err);
  }

  try {
    const html = await page.content();
    console.log('--- page content ---');
    console.log(html.slice(0, 2000));
    console.log('--- end content ---');
  } catch (err) {
    console.error('content read failed:', err.message || err);
  }

  await page.waitForTimeout(1500);
  await browser.close();
  console.log('done');
})();
