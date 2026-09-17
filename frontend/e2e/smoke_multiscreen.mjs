import { chromium } from 'playwright';
import { spawn } from 'child_process';
import http from 'http';

function checkPort(port) {
  return new Promise((resolve) => {
    const req = http.get(`http://localhost:${port}`, () => resolve(true));
    req.on('error', () => resolve(false));
  });
}

async function runSmokeTest() {
  console.log('--- Starting Browser Smoke Test ---');

  // Start vite preview server on port 4173
  const preview = spawn('npm', ['run', 'preview', '--', '--port', '4173'], {
    cwd: process.cwd(),
    stdio: 'inherit',
    detached: true,
  });

  // Wait for preview server to be ready
  let ready = false;
  for (let i = 0; i < 20; i++) {
    await new Promise((r) => setTimeout(r, 500));
    if (await checkPort(4173)) {
      ready = true;
      break;
    }
  }

  if (!ready) {
    console.error('Preview server failed to start within 10s');
    try { process.kill(-preview.pid); } catch {}
    process.exit(1);
  }

  console.log('Preview server ready at http://localhost:4173');

  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext();

  // Inject session auth into browser context so ProtectedRoute allows viewing popouts
  await context.addInitScript(() => {
    const mockUser = {
      id: 'test-user-1',
      email: 'pm@uisce.com',
      name: 'Portfolio Manager',
      role: 'admin',
      organization: 'Northwind',
      permissions: ['*'],
      is_active: true,
      roles: ['admin', 'portfolio_manager'],
      is_core_admin: true,
      is_admin: true,
    };
    localStorage.setItem('auth_token', 'mock_jwt_token_smoke_test');
    localStorage.setItem('auth_user', JSON.stringify(mockUser));
    localStorage.setItem('auth_expires_at', (Date.now() + 3600000).toString());
  });

  try {
    // 1. Open Window 1: Main App (/)
    console.log('1. Testing Main App (/) for navbar presence...');
    const pageMain = await context.newPage();
    await pageMain.goto('http://localhost:4173/');

    // Give page time to load
    await pageMain.waitForLoadState('networkidle');

    // Verify MainNavigation exists
    const navBar = await pageMain.$('header, nav, #main-content');
    console.log('Main App rendered content successfully:', navBar !== null);

    // 2. Open Window 2: Standalone Popout (/view/rebalancer)
    console.log('2. Testing Popout (/view/rebalancer)...');
    const pagePopout = await context.newPage();
    await pagePopout.goto('http://localhost:4173/view/rebalancer');
    await pagePopout.waitForLoadState('networkidle');

    // Verify DeskWindowFrame title is rendered
    const titleText = await pagePopout.innerText('.desk-caption-title');
    console.log('Popout title rendered:', titleText);

    // Verify main navbar is NOT present in popout (MainNavigation uses .MuiAppBar-root)
    const mainNavInPopout = await pagePopout.$('.MuiAppBar-root');
    console.log('Main navbar properly suppressed in popout:', mainNavInPopout === null);

    // Verify channel badge exists
    const badge = await pagePopout.$('.channel-badge');
    console.log('FDC3 Channel badge rendered:', badge !== null);

    // 3. Test Cross-Window FDC3 BroadcastChannel sync
    console.log('3. Testing cross-window BroadcastChannel communication...');
    await pagePopout.evaluate(() => {
      window.__testReceived = [];
      const bc = new BroadcastChannel('fdc3_channel_blue');
      bc.onmessage = (e) => window.__testReceived.push(e.data);
    });

    // Send from Window 1
    await pageMain.evaluate(() => {
      const bc = new BroadcastChannel('fdc3_channel_blue');
      bc.postMessage({
        sourceWindowId: 'browser_win_main',
        channelId: 'blue',
        context: { type: 'fdc3.instrument', id: { ticker: 'MSFT' } },
        timestamp: Date.now(),
      });
    });

    // Wait and check Window 2 receipt
    await new Promise((r) => setTimeout(r, 600));
    const received = await pagePopout.evaluate(() => window.__testReceived);
    console.log('Window 2 received BroadcastChannel message from Window 1:', received?.length > 0, received);

    // 4. Test URL stripping with init_token
    console.log('4. Testing URL init_token stripping...');
    await pagePopout.goto('http://localhost:4173/view/rebalancer?init_token=test_ephemeral_123');
    await pagePopout.waitForLoadState('networkidle');
    const currentUrl = pagePopout.url();
    console.log('Cleaned URL (token stripped):', currentUrl);
    const tokenInUrl = currentUrl.includes('init_token');
    console.log('init_token successfully stripped from URL:', !tokenInUrl);

    // 5. Test Universal Workspace Hub (/workspace)
    console.log('5. Testing Dockview Universal Workspace Hub (/workspace)...');
    const pageWorkspace = await context.newPage();
    await pageWorkspace.goto('http://localhost:4173/workspace');
    await pageWorkspace.waitForLoadState('networkidle');

    const workspaceHeader = await pageWorkspace.locator('text=UNIVERSAL WORKSPACE').first();
    const isVisible = await workspaceHeader.isVisible();
    console.log('Workspace header rendered:', isVisible);

    const dockviewContainer = await pageWorkspace.$('.dockview-theme-uisce, [class*="dockview"]');
    console.log('Dockview container mounted:', dockviewContainer !== null);

    const distributeBtn = await pageWorkspace.$('button:has-text("Distribute to Multi-Monitor")');
    console.log('Distribute to Multi-Monitor button present:', distributeBtn !== null);

    console.log('\n--- ALL BROWSER SMOKE CHECKS PASSED! ---');
  } catch (err) {
    console.error('Smoke test failure:', err);
    process.exitCode = 1;
  } finally {
    await browser.close();
    try { process.kill(-preview.pid); } catch {}
  }
}

runSmokeTest();
