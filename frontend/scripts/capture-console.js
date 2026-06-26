#!/usr/bin/env node
// Captures browser console logs, page errors, and API responses for a URL
// Usage: node scripts/capture-console.js --url http://localhost:5173/views/dddddddd

const puppeteer = require('puppeteer');
const argv = require('minimist')(process.argv.slice(2));

(async () => {
  const url = argv.url || 'http://localhost:5173/views/dddddddd';
  const waitMs = Number(argv.wait) || 2000;
  console.log(`Opening ${url} in headless browser...`);
  const browser = await puppeteer.launch({ args: ['--no-sandbox', '--disable-setuid-sandbox'] });
  const page = await browser.newPage();

  page.on('console', msg => {
    try {
      const args = msg.args();
      Promise.all(args.map(a => a.jsonValue().catch(() => a.toString())))
        .then(vals => console.log(`[console:${msg.type()}]`, ...vals));
    } catch (e) {
      console.log('[console] (failed to read args)', msg.text());
    }
  });

  page.on('pageerror', err => {
    console.log('[pageerror]', err && err.stack ? err.stack : String(err));
  });

  page.on('requestfailed', req => {
    console.log('[requestfailed]', req.method(), req.url(), req.failure() && req.failure().errorText);
  });

  page.on('response', async res => {
    try {
      const req = res.request();
      const url = req.url();
      const status = res.status();
      const ct = res.headers()['content-type'] || '';
      if (url.includes('/api/')) {
        let text = '';
        try {
          if (ct.includes('application/json') || ct.includes('text/')) {
            text = await res.text();
          } else {
            text = `<${ct} response omitted>`;
          }
        } catch (e) {
          text = `<failed to read body: ${e.message}>`;
        }
        console.log(`[api-response] ${req.method()} ${url} -> ${status}\n${text}`);
      }
    } catch (e) {
      console.log('[response handler error]', e && e.stack ? e.stack : String(e));
    }
  });

  try {
    await page.goto(url, { waitUntil: 'networkidle2', timeout: 30000 });
    console.log(`Page loaded, waiting ${waitMs}ms for any async activity...`);
    await page.waitForTimeout(waitMs);
  } catch (e) {
    console.log('[goto error]', e && e.message ? e.message : String(e));
  }

  // take screenshot for visual confirmation
  try {
    const out = '/tmp/capture-console-screenshot.png';
    await page.screenshot({ path: out, fullPage: false });
    console.log('Saved screenshot to', out);
  } catch (e) {
    console.log('[screenshot failed]', e && e.message ? e.message : String(e));
  }

  await browser.close();
  console.log('Browser closed.');
  process.exit(0);
})();
