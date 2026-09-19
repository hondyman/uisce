#!/usr/bin/env node
/**
 * Bundle Size & Code-Splitting Gate.
 * 
 * Verifies that:
 * 1. Lazy-loaded trading views (FixedIncomeDashboard, AIPortfolioRebalancer, ScenarioAnalysisPro, PageBrowser)
 *    are emitted as separate async chunks, NOT bundled into the main index bundle.
 * 2. Generated assets exist and match expected distribution.
 * 
 * Usage:
 *   node scripts/check-bundle-size.mjs
 */
import { readdirSync, statSync, readFileSync } from 'node:fs';
import path from 'node:path';
import zlib from 'node:zlib';

const DIST_ASSETS = path.resolve('dist/assets');

try {
  const files = readdirSync(DIST_ASSETS);
  console.log(`[Bundle Check] Inspecting ${files.length} build artifacts in dist/assets...`);

  const jsFiles = files.filter((f) => f.endsWith('.js'));
  const mainIndex = jsFiles.find((f) => f.startsWith('index-'));

  if (!mainIndex) {
    console.error('❌ FAIL: Main application index-*.js bundle not found in dist/assets');
    process.exit(1);
  }

  const mainStats = statSync(path.join(DIST_ASSETS, mainIndex));
  const mainRaw = readFileSync(path.join(DIST_ASSETS, mainIndex));
  const mainGz = zlib.gzipSync(mainRaw);
  const mainSizeKb = +(mainStats.size / 1024).toFixed(1);
  const mainGzKb = +(mainGz.length / 1024).toFixed(1);

  console.log(`✓ Main Index Bundle: ${mainIndex} (${mainSizeKb} KB uncompressed, ${mainGzKb} KB gzip)`);

  // Print all generated JS chunks with uncompressed and gzip sizes
  console.log('\n--- GENERATED CODE-SPLIT CHUNKS & GZIP FOOTPRINT ---');
  let totalJsSizeKb = 0;
  let totalGzSizeKb = 0;
  for (const f of jsFiles) {
    const raw = readFileSync(path.join(DIST_ASSETS, f));
    const gz = zlib.gzipSync(raw);
    const sizeKb = +(raw.length / 1024).toFixed(1);
    const gzKb = +(gz.length / 1024).toFixed(1);
    totalJsSizeKb += sizeKb;
    totalGzSizeKb += gzKb;
    console.log(`  • ${f.padEnd(40)}: ${sizeKb.toString().padStart(8)} KB (gz: ${gzKb.toString().padStart(6)} KB)`);
  }

  console.log(`\nTotal JS Assets: ${totalJsSizeKb.toFixed(1)} KB across ${jsFiles.length} chunks (Total Gzip: ${totalGzSizeKb.toFixed(1)} KB)`);
  console.log(`Initial Shell Footprint (index.js): ${mainGzKb} KB gzip`);

  // Assert code-split chunks exist (> 8 chunks)
  if (jsFiles.length < 8) {
    console.error(`❌ FAIL: Expected at least 8 code-split chunks, found ${jsFiles.length}`);
    process.exit(1);
  }

  // Assertion 1: Shell Baseline Drift Alarm (< 5% growth beyond recorded calibration of 1,496.3 KB)
  const BASELINE_SHELL_GZIP_KB = 1496.3;
  const MAX_PERMISSIBLE_SHELL_GZIP_KB = +(BASELINE_SHELL_GZIP_KB * 1.05).toFixed(1); // +5% threshold = 1,571.1 KB
  if (mainGzKb > MAX_PERMISSIBLE_SHELL_GZIP_KB) {
    console.error(`❌ FAIL: Shell bundle size regression! Current: ${mainGzKb} KB gzip exceeds maximum allowed drift threshold of ${MAX_PERMISSIBLE_SHELL_GZIP_KB} KB gzip (+5% over ${BASELINE_SHELL_GZIP_KB} KB baseline)`);
    process.exit(1);
  }
  console.log(`✓ Shell baseline drift check passed: ${mainGzKb} KB <= ${MAX_PERMISSIBLE_SHELL_GZIP_KB} KB limit`);

  // Assertion 2: Workstation heavy views must exist as split chunks and be < 50 KB gzip each
  const requiredViewChunks = [
    'FixedIncomeDashboard',
    'AIPortfolioRebalancer',
    'ScenarioAnalysisPro',
    'PageBrowser',
  ];

  for (const viewName of requiredViewChunks) {
    const chunkFile = jsFiles.find((f) => f.startsWith(`${viewName}-`));
    if (!chunkFile) {
      console.error(`❌ FAIL: Expected dedicated split chunk for ${viewName}, but none was found!`);
      process.exit(1);
    }
    const raw = readFileSync(path.join(DIST_ASSETS, chunkFile));
    const gz = zlib.gzipSync(raw);
    const gzKb = +(gz.length / 1024).toFixed(1);
    if (gzKb > 50.0) {
      console.error(`❌ FAIL: View chunk ${chunkFile} size ${gzKb} KB gzip exceeds the 50 KB gzip limit!`);
      process.exit(1);
    }
    console.log(`✓ Workstation split view chunk verified: ${chunkFile} (${gzKb} KB gzip < 50 KB limit)`);
  }

  // Assertion 3: Verify no workstation view component code is statically bundled into the main shell
  const mainCode = mainRaw.toString('utf8');
  const forbiddenShellMarkers = [
    'FixedIncomeDashboard',
    'AIPortfolioRebalancer',
    'ScenarioAnalysisPro',
  ];
  // Verify that component function signatures are not embedded in main shell
  for (const marker of forbiddenShellMarkers) {
    // Check for component exports or module definitions
    if (mainCode.includes(`export{${marker}`) || mainCode.includes(`function ${marker}(`)) {
      console.error(`❌ FAIL: Workstation view marker ${marker} detected inside main shell! View is not properly lazy-loaded.`);
      process.exit(1);
    }
  }
  console.log(`✓ Main shell verified free of static workstation component definitions.`);

  console.log('\n✅ BUNDLE SIZE & CODE-SPLITTING CHECK PASSED!');
  process.exit(0);
} catch (err) {
  console.error('❌ Bundle check error:', err);
  process.exit(1);
}
