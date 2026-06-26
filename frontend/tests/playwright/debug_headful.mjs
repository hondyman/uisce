import playwright from 'playwright';

(async () => {
  const browser = await playwright.chromium.launch({ headless: false, slowMo: 100 });
  const context = await browser.newContext();
  const page = await context.newPage();
  try {
    console.log('navigating headful...');
    const resp = await page.goto('http://localhost:5173/', { waitUntil: 'domcontentloaded', timeout: 60000 });
    console.log('response status:', resp && resp.status());
    await page.waitForTimeout(3000);
    await page.screenshot({ path: '/tmp/schema-explorer-debug.png', fullPage: true });
    console.log('screenshot saved to /tmp/schema-explorer-debug.png');
  } catch (e) {
    console.error('error during goto', e);
  } finally {
    await browser.close();
  }
})();
