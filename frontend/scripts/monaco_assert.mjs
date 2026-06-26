import { chromium } from 'playwright';

(async () => {
  const url = 'http://localhost:5173/';
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();

  const consoleMsgs = [];
  page.on('console', (m) => consoleMsgs.push({ type: m.type(), text: m.text() }));
  page.on('pageerror', (e) => consoleMsgs.push({ type: 'pageerror', text: e.message }));

  try {
    // Pre-populate localStorage with a fake authenticated session so ProtectedRoute
    // won't redirect to /login during our headless checks. Use a far-future expiry.
    const expiresAt = Date.now() + 24 * 60 * 60 * 1000; // 24h
    await page.addInitScript(({ token, user, expiresAt }) => {
      try {
        localStorage.setItem('auth_token', token);
        localStorage.setItem('auth_refresh_token', token + '_r');
        localStorage.setItem('auth_user', JSON.stringify(user));
        localStorage.setItem('auth_expires_at', String(expiresAt));
      } catch (e) {}
    }, { token: 'test-token-123', user: { id: 'tester', email: 'tester@example.com', name: 'Tester' }, expiresAt });

    await page.goto(url, { waitUntil: 'networkidle', timeout: 30000 });

    // Navigate directly to the Model Generator page and trigger generation
    const generatorUrl = url.replace(/\/$/, '') + '/model-generator';
    await page.goto(generatorUrl, { waitUntil: 'networkidle', timeout: 15000 });
    // wait for page content
    await page.waitForTimeout(800);

    // Try to click the first "Generate Model" button to open the dialog which contains the CodeEditor
    try {
      const genBtn = await page.locator('text=Generate Model').first();
      if (await genBtn.count() > 0) {
        await genBtn.click({ timeout: 5000 });
      }
    } catch (e) {
      // ignore - may not exist on page
    }

    // wait for editor to mount (monaco may be dynamically imported)
    let found = false;
    try {
      // prefer monaco-container (our custom wrapper) or .monaco-editor
      await page.waitForSelector('.monaco-container, .monaco-editor, .monaco-inner', { timeout: 8000 });
      found = true;
    } catch (_) {
      found = false;
    }

    console.log('Found monaco DOM element:', found);

    // Evaluate presence of global monaco
    const monacoInfo = await page.evaluate(() => {
      try {
        const m = window.monaco;
        if (!m) return { present: false };
        // Try to read a version string; Monaco doesn't always expose a global version.
        let version = null;
        try {
          if (m.version) version = m.version;
        } catch {}
        try {
          if (!version && m.editor && typeof m.editor.getModels === 'function') {
            const models = m.editor.getModels();
            if (models && models.length > 0) {
              const v = models[0].getVersionId ? models[0].getVersionId() : null;
              if (v !== null && v !== undefined) version = String(v);
            }
          }
        } catch {}

        return {
          present: true,
          hasEditor: !!m.editor,
          version,
        };
      } catch (e) {
        return { present: false, error: String(e) };
      }
    });

    console.log('monacoInfo:', monacoInfo);

  } catch (e) {
    console.error('Navigation/test error:', e.message);
  } finally {
    console.log('--- Console messages captured ---');
    consoleMsgs.forEach((c) => console.log(c.type, c.text));
    await browser.close();
  }
})();
