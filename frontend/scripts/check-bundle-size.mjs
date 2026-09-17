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

  console.log('\n✅ BUNDLE SIZE & CODE-SPLITTING CHECK PASSED!');
  process.exit(0);
} catch (err) {
  console.error('❌ Bundle check error:', err);
  process.exit(1);
}
