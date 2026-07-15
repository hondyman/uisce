#!/usr/bin/env node
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const srcDir = path.join(__dirname, '../src');

const processed = [];
const errors = [];

function transformGridProps(props) {
  const sizeParts = [];
  const remaining = [];

  const propPattern = /(?:^|\s)(xs|sm|md|lg|xl)(?:\s*=\s*)?(\{[^}]*\}|'[^']*'|"[^"]*"|[\w]+)/g;
  let m;
  while ((m = propPattern.exec(props)) !== null) {
    const [, bp, rawVal] = m;
    if (['xs', 'sm', 'md', 'lg', 'xl'].includes(bp)) {
      let val = rawVal.trim();
      const isBracewrapped = val.startsWith('{') && val.endsWith('}') && val.length > 2;
      if (isBracewrapped) {
        val = val.slice(1, -1);
      }
      sizeParts.push([bp, val]);
    } else {
      remaining.push(`${bp}=${rawVal}`);
    }
  }

  if (sizeParts.length === 0) return null;

  let sizeStr;
  if (sizeParts.length === 1) {
    const [bp, val] = sizeParts[0];
    if (/^\d+$/.test(val)) {
      sizeStr = `{${val}}`;
    } else {
      sizeStr = `{{ '${bp}': ${val} }}`;
    }
  } else {
    const pairs = sizeParts.map(([bp, val]) => `'${bp}': ${val}`);
    sizeStr = `{{ ${pairs.join(', ')} }}`;
  }

  return { sizeStr, remaining };
}

function processFile(filePath) {
  let content = fs.readFileSync(filePath, 'utf8');
  const original = content;

  const seen = new Set();
  let prev;
  while (prev !== content) {
    prev = content;

    content = content.replace(/<Grid(\s*)item(\s+)([^>]*)\/>/g, (match, ws1, ws2, props) => {
      const key = match + '///SC';
      if (seen.has(key)) return match;
      seen.add(key);
      const result = transformGridProps(props);
      if (!result) return match;
      const { sizeStr, remaining } = result;
      const restProps = remaining.length > 0 ? ws2 + remaining.join(ws2.trim() ? ' ' : '') : '';
      return `<Grid${ws1}size=${sizeStr}${restProps}/>`;
    });

    content = content.replace(/<Grid(\s*)item(\s+)([^>]*?)>/g, (match, ws1, ws2, props) => {
      const key = match + '/// NSC';
      if (seen.has(key)) return match;
      seen.add(key);
      const result = transformGridProps(props);
      if (!result) return match;
      const { sizeStr, remaining } = result;
      const restProps = remaining.length > 0 ? ws2 + remaining.join(ws2.trim() ? ' ' : '') : '';
      return `<Grid${ws1}size=${sizeStr}${restProps}>`;
    });
  }

  if (content !== original) {
    fs.writeFileSync(filePath, content, 'utf8');
    return true;
  }
  return false;
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
        const ok = processFile(full);
        if (ok) processed.push(full);
      } catch (e) {
        errors.push({ file: full, error: e.message });
      }
    }
  }
}

walkDir(srcDir);

console.log(`\nGrid v1→v2 migration complete`);
console.log(`  Files modified: ${processed.length}`);
console.log(`  Errors: ${errors.length}`);
if (processed.length > 0) {
  console.log(`\nModified files (first 10):`);
  processed.slice(0, 10).forEach(f => console.log(`  ${f}`));
  if (processed.length > 10) console.log(`  ... and ${processed.length - 10} more`);
}
if (errors.length > 0) {
  errors.slice(0, 5).forEach(e => console.log(`  ${e.file}: ${e.error}`));
}
