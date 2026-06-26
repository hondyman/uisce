#!/usr/bin/env node
const fs = require('fs');
const path = require('path');

const root = path.resolve(__dirname, '..', 'frontend');
const exts = ['.tsx', '.ts', '.jsx', '.js'];

function walk(dir) {
  const entries = fs.readdirSync(dir, { withFileTypes: true });
  for (const e of entries) {
    const full = path.join(dir, e.name);
    if (e.isDirectory()) {
      walk(full);
    } else if (exts.includes(path.extname(e.name))) {
      processFile(full);
    }
  }
}

function processFile(file) {
  let src;
  try {
    src = fs.readFileSync(file, 'utf8');
  } catch (err) {
    // skip unreadable files (directories, binary, etc.)
    return;
  }
  if (!src.includes("import React from 'react'") ) return;
  // Don't touch files that import React with named hooks or use 'React.' at runtime
  const hasReactDot = src.includes('React.');
  const hasNamedImport = /import\s+\{[^}]*\}\s+from\s+['\"]react['\"]/m.test(src);
  const hasTypeImport = /import\s+type\s+React\s+from\s+['\"]react['\"]/m.test(src);
  if (hasReactDot || hasNamedImport) {
    // leave alone
    return;
  }
  // Remove the import React line(s)
  const lines = src.split('\n');
  const filtered = lines.filter(l => !/import\s+React\s+from\s+['\"]react['\"];?/.test(l.trim()));
  if (filtered.length === lines.length) return;
  fs.writeFileSync(file, filtered.join('\n'), 'utf8');
  console.log('Updated', file);
}

walk(root);
console.log('Done');
