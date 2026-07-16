#!/usr/bin/env node
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const srcDir = path.join(__dirname, '../src');

const processed = [];
const errors = [];

function fixFile(filePath) {
  let content = fs.readFileSync(filePath, 'utf8');
  let original = content;
  let changed = false;

  content = content.replace(/<Grid([^>]*)size=\{ ([a-z]+): \{(\d+)\} \}/g, (_, attrs, bp, num) => {
    changed = true;
    return `<Grid${attrs}size={${bp}: ${num}}>`;
  });

  content = content.replace(/<Grid([^>]*)size=\{ '([a-z]+)': \{(\d+)\}, '([a-z]+)': \{(\d+)\} \}/g, (_, attrs, bp1, v1, bp2, v2) => {
    changed = true;
    return `<Grid${attrs}size={{ '${bp1}': ${v1}, '${bp2}': ${v2} }}>`;
  });

  content = content.replace(/<Grid([^>]*)size=\{ '([a-z]+)': \{(\d+)\}, '([a-z]+)': \{(\d+)\}, '([a-z]+)': \{(\d+)\} \}/g, (_, attrs, bp1, v1, bp2, v2, bp3, v3) => {
    changed = true;
    return `<Grid${attrs}size={{ '${bp1}': ${v1}, '${bp2}': ${v2}, '${bp3}': ${v3} }}>`;
  });

  content = content.replace(/<Grid([^>]*)size=\{ '([a-z]+)': \{(\d+)\} \}/g, (_, attrs, bp, num) => {
    changed = true;
    return `<Grid${attrs}size={ ${bp}: ${num} }>`;
  });

  content = content.replace(/<Grid([^>]*)size=\{ '([a-z]+)': \{(\d+)\}, '([a-z]+)': \{(\d+)\}, '([a-z]+)': \{(\d+)\}, '([a-z]+)': \{(\d+)\} \}/g, (_, attrs, bp1, v1, bp2, v2, bp3, v3, bp4, v4) => {
    changed = true;
    return `<Grid${attrs}size={{ '${bp1}': ${v1}, '${bp2}': ${v2}, '${bp3}': ${v3}, '${bp4}': ${v4} }}>`;
  });

  if (!changed) return false;
  fs.writeFileSync(filePath, content, 'utf8');
  return true;
}

function walkDir(dir) {
  const entries = fs.readdirSync(dir, { withFileTypes: true });
  for (const entry of entries) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      const skip = ['node_modules', '.git', '__tests__', '__mocks__', 'dist', 'build', '.next'];
      if (!skip.includes(entry.name) && !entry.name.startsWith('.')) {
        walkDir(full);
      }
    } else if (entry.name.endsWith('.tsx') || entry.name.endsWith('.ts')) {
      try {
        const ok = fixFile(full);
        if (ok) processed.push(full);
      } catch (e) {
        errors.push({ file: full, error: e.message });
      }
    }
  }
}

walkDir(srcDir);

console.log(`\nGrid fixup complete`);
console.log(`  Files fixed: ${processed.length}`);
console.log(`  Errors: ${errors.length}`);
if (processed.length > 0) {
  processed.forEach(f => console.log(`  ${f}`));
}
if (errors.length > 0) {
  errors.forEach(e => console.log(`  ${e.file}: ${e.error}`));
}
