const { chromium } = require('playwright');

(async () => {
  const url = process.argv[2] || 'http://localhost:5173/core/domains';
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
    console.log(`[requestfailed] ${req.url()} ${req.failure().errorText}`);
  });

  console.log('Opening', url);
  try {
    await page.goto(url, { waitUntil: 'networkidle' });
  } catch (err) {
    console.error('goto failed:', err.message);
  }

  // capture some element snapshot
  try {
    const html = await page.content();
    console.log('--- page content ---');
    console.log(html.slice(0, 2000));
    console.log('--- end content ---');
  } catch (err) {
    console.error('content read failed:', err.message);
  }

  // wait to capture any async console logs/errors
  await page.waitForTimeout(1500);
  await browser.close();
  console.log('done');
})();
