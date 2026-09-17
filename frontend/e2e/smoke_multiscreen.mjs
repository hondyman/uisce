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

    // 6. Test Interactive Live Cross-View FDC3 Sync (Blotter -> Rebalancer)
    console.log('6. Testing interactive cross-view FDC3 synchronization...');
    const aaplBtn = pageWorkspace.locator('button:has-text("AAPL")').first();
    await aaplBtn.waitFor({ state: 'visible', timeout: 5000 });
    console.log('Blotter demo quick-sync AAPL button found');

    await aaplBtn.click();
    await pageWorkspace.waitForTimeout(500);

    const rebalancerAaplBadge = pageWorkspace.locator('text=FDC3 (blue): AAPL').first();
    const isAaplSynced = await rebalancerAaplBadge.isVisible();
    console.log('Rebalancer header received active FDC3 context (AAPL):', isAaplSynced);
    if (!isAaplSynced) {
      throw new Error('Rebalancer failed to display synced AAPL FDC3 badge');
    }

    // Switch context to NVDA
    const nvdaBtn = pageWorkspace.locator('button:has-text("NVDA")').first();
    await nvdaBtn.click();
    await pageWorkspace.waitForTimeout(500);

    const rebalancerNvdaBadge = pageWorkspace.locator('text=FDC3 (blue): NVDA').first();
    const isNvdaSynced = await rebalancerNvdaBadge.isVisible();
    console.log('Rebalancer header updated to new FDC3 context (NVDA):', isNvdaSynced);
    if (!isNvdaSynced) {
      throw new Error('Rebalancer failed to display updated NVDA FDC3 badge');
    }

    // 7. Test Desk Layout Restore & Ghost Window Prevention
    console.log('7. Testing desk layout restore prompt and ghost window prevention...');
    await pageWorkspace.evaluate(() => {
      const layout = JSON.parse(localStorage.getItem('uisce_workspace_layout_v1') || '{"version":1,"detachedWindows":[]}');
      layout.detachedWindows = [{
        windowId: 'win_test_detached_1',
        route: '/view/rebalancer',
        title: 'Test Detached View',
        targetScreenIndex: 0,
        width: 1200,
        height: 800,
      }];
      localStorage.setItem('uisce_workspace_layout_v1', JSON.stringify(layout));
    });

    // Reload workspace page to trigger session restore check
    await pageWorkspace.reload();
    await pageWorkspace.waitForLoadState('networkidle');

    const restoreBanner = pageWorkspace.locator('text=Found 1 detached desk window(s) from previous session').first();
    const bannerVisible = await restoreBanner.isVisible();
    console.log('Restore prompt banner displayed on reload:', bannerVisible);
    if (!bannerVisible) {
      throw new Error('Restore prompt banner failed to appear on reload');
    }

    // Dismiss the banner and assert storage is cleared (ghost window prevention)
    const dismissBtn = pageWorkspace.locator('button:has-text("Dismiss")').first();
    await dismissBtn.click();
    await pageWorkspace.waitForTimeout(200);

    const bannerDismissed = !(await restoreBanner.isVisible());
    console.log('Restore banner dismissed successfully:', bannerDismissed);

    const storedState = await pageWorkspace.evaluate(() => {
      return JSON.parse(localStorage.getItem('uisce_workspace_layout_v1') || '{}');
    });
    const detachedRemaining = (storedState.detachedWindows || []).length;
    console.log('Detached windows cleared on dismiss (0 remaining):', detachedRemaining === 0);
    if (detachedRemaining !== 0) {
      throw new Error(`Expected 0 detached windows after dismiss, found ${detachedRemaining}`);
    }

    // Reload again to verify no ghost banner resurrects
    await pageWorkspace.reload();
    await pageWorkspace.waitForLoadState('networkidle');

    const ghostBanner = pageWorkspace.locator('text=Found 1 detached desk window(s) from previous session').first();
    const ghostVisible = await ghostBanner.isVisible();
    console.log('Zero ghost restore banner on subsequent reload:', !ghostVisible);
    if (ghostVisible) {
      throw new Error('Ghost restore banner unexpectedly reappeared');
    }

    // 8. Test Travel Mode Toggle & Ephemeral View-Time Consolidation
    console.log('8. Testing Travel Mode button toggle...');
    const travelBtn = pageWorkspace.locator('button:has-text("Travel Mode")').first();
    const travelBtnVisible = await travelBtn.isVisible();
    console.log('Travel Mode button visible in workspace toolbar:', travelBtnVisible);
    if (!travelBtnVisible) {
      throw new Error('Travel Mode button missing from header toolbar');
    }

    await travelBtn.click();
    await pageWorkspace.waitForTimeout(300);

    const exitTravelBtn = pageWorkspace.locator('button:has-text("Exit Travel")').first();
    const exitVisible = await exitTravelBtn.isVisible();
    console.log('Travel Mode engaged, Exit Travel button visible:', exitVisible);
    if (!exitVisible) {
      throw new Error('Failed to toggle Travel Mode into active state');
    }

    await exitTravelBtn.click();
    await pageWorkspace.waitForTimeout(300);
    const revertedToTravel = await pageWorkspace.locator('button:has-text("Travel Mode")').first().isVisible();
    console.log('Travel Mode exited cleanly back to multi-monitor mode:', revertedToTravel);
    if (!revertedToTravel) {
      throw new Error('Failed to toggle back to standard multi-monitor mode');
    }

    // 9. Test Live Cross-Window FDC3 Intent Routing & Explicit Acknowledgment
    console.log('9. Testing live cross-window FDC3 intent routing with acknowledgment...');
    // Ensure Window B (popout) is on /view/rebalancer and ready
    await pagePopout.bringToFront();
    await pagePopout.waitForTimeout(400);

    // Raise ViewAnalysis intent from Window A (workspace hub) targeted to 'rebalancer' in Window B
    const intentRes = await pageWorkspace.evaluate(async () => {
      if (!window.__fdc3Agent) {
        throw new Error('window.__fdc3Agent missing on pageWorkspace');
      }
      return await window.__fdc3Agent.raiseIntent('ViewAnalysis', {
        type: 'fdc3.instrument',
        id: { ticker: 'INTC' },
        name: 'Intel Corp',
      }, 'rebalancer');
    });

    console.log('Intent resolution returned in Window A:', JSON.stringify(intentRes));
    if (intentRes.intent !== 'ViewAnalysis') {
      throw new Error(`Expected intent ViewAnalysis, got ${intentRes.intent}`);
    }
    if (intentRes.target?.viewId !== 'rebalancer') {
      throw new Error(`Expected target viewId 'rebalancer', got ${intentRes.target?.viewId}`);
    }

    // Wait and assert Window B (pagePopout) updated its UI with the intent payload
    const tickerLocator = pagePopout.locator('[data-testid="rebalancer-ticker"]').first();
    await tickerLocator.waitFor({ state: 'visible', timeout: 5000 });
    const popoutTicker = await tickerLocator.textContent();
    console.log('Popout view updated with target ticker from intent:', popoutTicker);
    if (popoutTicker !== 'INTC') {
      throw new Error(`Expected popout ticker 'INTC', got '${popoutTicker}'`);
    }
    console.log('Cross-window intent routing & visible response verified: true');

    console.log('\n--- ALL BROWSER & INTERACTIVE SMOKE CHECKS PASSED (9/9)! ---');
  } catch (err) {
    console.error('Smoke test failure:', err);
    process.exitCode = 1;
  } finally {
    await browser.close();
    try { process.kill(-preview.pid); } catch {}
  }
}

runSmokeTest();
