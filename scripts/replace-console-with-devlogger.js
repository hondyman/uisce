#!/usr/bin/env node
const fs = require('fs');
const path = require('path');

const root = path.resolve(__dirname, '..', 'frontend', 'src');
const devLoggerPath = path.resolve(root, 'utils', 'devLogger.ts');

function relImport(fromFile) {
  const fromDir = path.dirname(fromFile);
  let rel = path.relative(fromDir, devLoggerPath).replace(/\\/g, '/');
  if (!rel.startsWith('.')) rel = './' + rel;
  // drop extension
  rel = rel.replace(/\.ts$/, '');
  return rel;
}

function processFile(file) {
  let src = fs.readFileSync(file, 'utf8');
  if (!src.includes('console.log') && !src.includes('console.warn') && !src.includes('console.info')) return false;

  // skip files that are backups, tests, or generated
  if (file.includes('.backup') || file.includes('.BACKUP.') || file.includes('.spec.') || file.includes('__tests__')) return false;

  const hasDevLogImport = /from\s+['\"](.+devLogger)['\"];?/.test(src);
  let changed = false;

  // simple replacements - avoid within comments by ignoring lines starting with //
  const lines = src.split('\n');
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const trimmed = line.trim();
    if (trimmed.startsWith('//') || trimmed.startsWith('*') || trimmed.startsWith('/*')) continue;
    if (trimmed.includes('console.log(')) {
      lines[i] = line.replace(/console\.log\(/g, 'devLog(');
      changed = true;
    }
    if (trimmed.includes('console.info(')) {
      lines[i] = line.replace(/console\.info\(/g, 'devDebug(');
      changed = true;
    }
    if (trimmed.includes('console.warn(')) {
      lines[i] = line.replace(/console\.warn\(/g, 'devWarn(');
      changed = true;
    }
  }

  if (!changed) return false;

  let out = lines.join('\n');

  if (!hasDevLogImport) {
    const rel = relImport(file);
    // prefer named import of used functions only
    let importStmt = "import { devLog, devDebug, devWarn } from '" + rel + "';\n";
    // insert after the last import
    const importRegex = /(^import[\s\S]*?;\n)(?!import)/m;
    let inserted = false;
    const importLines = out.split('\n');
    for (let i = 0; i < Math.min(importLines.length, 40); i++) {
      // find last import index in the first 40 lines
      if (!importLines[i].trim().startsWith('import')) continue;
      let j = i;
      while (j + 1 < importLines.length && importLines[j + 1].trim().startsWith('import')) j++;
      importLines.splice(j + 1, 0, importStmt);
      out = importLines.join('\n');
      inserted = true;
      break;
    }
    if (!inserted) {
      out = importStmt + '\n' + out;
    }
  }

  fs.writeFileSync(file, out, 'utf8');
  console.log('Updated', file);
  return true;
}

function walk(dir) {
  const files = fs.readdirSync(dir);
  let changed = 0;
  for (const f of files) {
    const full = path.join(dir, f);
    const stat = fs.statSync(full);
    if (stat.isDirectory()) {
      if (['node_modules', 'dist', '.git'].includes(f)) continue;
      changed += walk(full) ? 1 : 0;
    } else if (/\.tsx?$/.test(f)) {
      if (processFile(full)) changed++;
    }
  }
  return changed;
}

const totalChanged = walk(root);
console.log('Files changed:', totalChanged);
