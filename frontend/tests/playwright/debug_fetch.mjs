import playwright from 'playwright';

(async () => {
  const browser = await playwright.chromium.launch({ headless: true });
  const context = await browser.newContext();
  const page = await context.newPage();
  try {
    console.log('navigating...');
    const resp = await page.goto('http://localhost:5173/', { waitUntil: 'load', timeout: 30000 });
    console.log('response status:', resp && resp.status());
    const html = await page.content();
    console.log('html length', html.length);
  } catch (e) {
    console.error('error during goto', e);
  } finally {
    await browser.close();
  }
})();
